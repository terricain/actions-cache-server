package awss3

import (
	"errors"
	"io"
	"net/url"
	p "path"
	"strconv"
	"strings"
	"time"

	"github.com/terrycain/actions-cache-server/pkg/s"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/google/uuid"
	"github.com/terrycain/actions-cache-server/pkg/e"
)

type Backend struct {
	BucketURL string
	Session   *session.Session
	Client    *s3.S3

	bucket string
	prefix string
}

func New(connectionString string) (*Backend, error) {
	sess, err := session.NewSession(&aws.Config{Region: aws.String("us-east-1")})
	if err != nil {
		return &Backend{}, err
	}
	// Enable uuid rand pool for better performance
	uuid.EnableRandPool()

	backend := Backend{
		BucketURL: connectionString,
		Session:   sess,
	}
	return &backend, nil
}

func (b *Backend) Setup() error {
	// Parse URL
	parsedURL, err := url.Parse(b.BucketURL)
	if err != nil {
		return err
	}

	if parsedURL.Scheme != "s3" {
		//goland:noinspection GoErrorStringFormat
		return errors.New("S3 url should be in the format of s3://bucket/prefix")
	}

	// Used if e2e testing with localstack
	if strings.Contains(parsedURL.RawQuery, "forces3path") {
		b.Session.Config.S3ForcePathStyle = aws.Bool(true)
	}

	b.bucket = parsedURL.Host
	b.prefix = strings.TrimPrefix(parsedURL.Path, "/")

	b.Client = s3.New(b.Session, b.Session.Config)

	resp, err := b.Client.GetBucketLocation(&s3.GetBucketLocationInput{Bucket: aws.String(b.bucket)})
	if err != nil {
		return err
	}

	if resp.LocationConstraint != nil {
		b.Session.Config.Region = resp.LocationConstraint
		b.Client = s3.New(b.Session, b.Session.Config)
	}

	return nil
}

func (b *Backend) Type() string {
	return "s3"
}

// S3 has UploadPartCopy to create multipart uploads from other objects. So we can upload chunks as uuid named
// files and store the filename in a db with start and end therefore when finalising we know the order in which
// to concatenate files.
//
// The reason we don't just use a regular multipart upload is chunks can be uploaded in parallel, and you could
// receive a later chunk before an earlier one, multipart upload parts need an int 1-10000 and will be assembled
// in sorted order, as we don't have data uploaded in order, we can't reliably do this.
//
// Write Uploads a part of a file to S3.
func (b *Backend) Write(repoKey string, r io.Reader, start, end int, size int64) (string, int64, error) {
	cacheFile := uuid.New().String()
	filePath := p.Join(b.prefix, repoKey, cacheFile)

	uploader := s3manager.NewUploader(b.Session)
	_, err := uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(b.bucket),
		Body:   r,
		Key:    aws.String(filePath),
		Metadata: map[string]*string{
			"chunk_start": aws.String(strconv.Itoa(start)),
			"chunk_end":   aws.String(strconv.Itoa(end)),
			"chunk_size":  aws.String(strconv.FormatInt(size, 10)),
			"ownedBy":     aws.String("actions-cache-server"),
		},
	})
	if err != nil {
		return "", 0, err
	}

	// Think I did this because we don't get returned the actual bytes uploaded count
	headResponse, err := b.Client.HeadObject(&s3.HeadObjectInput{
		Bucket: aws.String(b.bucket),
		Key:    aws.String(filePath),
	})
	if err != nil {
		return "", 0, err
	}

	return cacheFile, *headResponse.ContentLength, nil
}

func (b *Backend) Delete(repoKey, path string) error {
	filePath := p.Join(b.prefix, repoKey, path)

	_, err := b.Client.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(b.bucket),
		Key:    aws.String(filePath),
	})

	return err
}

func (b *Backend) Finalise(repoKey string, parts []s.CachePart) (string, error) {
	// If we've only got 1 part, then use that as the cache file to save more work
	if len(parts) == 1 {
		filePath := p.Join(b.prefix, repoKey, parts[0].Data) // CachePart.Data would be a UUID without the repo key
		return filePath, nil
	}

	cacheFile := uuid.New().String()
	filePath := p.Join(b.prefix, repoKey, cacheFile)

	multiPartUpload, err := b.Client.CreateMultipartUpload(&s3.CreateMultipartUploadInput{
		Bucket: aws.String(b.bucket),
		Key:    aws.String(filePath),
		Metadata: map[string]*string{
			"ownedBy": aws.String("actions-cache-server"),
		},
	})
	if err != nil {
		return "", err
	}

	uploadParts := make([]*s3.CompletedPart, 0)
	for index, part := range parts {
		// UploadPartCopy
		partNumber := aws.Int64(int64(index + 1))
		src := aws.String(b.bucket + "/" + p.Join(b.prefix, repoKey, part.Data))
		resp, err2 := b.Client.UploadPartCopy(&s3.UploadPartCopyInput{
			Bucket:     aws.String(b.bucket),
			Key:        aws.String(filePath),
			UploadId:   multiPartUpload.UploadId,
			CopySource: src,
			PartNumber: partNumber, // Part number 1->10000
		})
		if err2 != nil {
			// Try and abort the multipart upload
			_, _ = b.Client.AbortMultipartUpload(&s3.AbortMultipartUploadInput{
				Bucket:   aws.String(b.bucket),
				Key:      aws.String(filePath),
				UploadId: multiPartUpload.UploadId,
			})
			return "", err2
		}

		uploadPart := s3.CompletedPart{
			ETag:       aws.String(strings.Trim(*resp.CopyPartResult.ETag, "\"")),
			PartNumber: partNumber,
		}
		uploadParts = append(uploadParts, &uploadPart)
	}

	// By here, we've specified all the upload parts
	_, err = b.Client.CompleteMultipartUpload(&s3.CompleteMultipartUploadInput{
		Bucket:   aws.String(b.bucket),
		Key:      aws.String(filePath),
		UploadId: multiPartUpload.UploadId,
		MultipartUpload: &s3.CompletedMultipartUpload{
			Parts: uploadParts,
		},
	})
	if err != nil {
		// Try and abort the multipart upload
		_, _ = b.Client.AbortMultipartUpload(&s3.AbortMultipartUploadInput{
			Bucket:   aws.String(b.bucket),
			Key:      aws.String(filePath),
			UploadId: multiPartUpload.UploadId,
		})
		return "", err
	}

	return cacheFile, nil
}

func (b *Backend) GenerateArchiveURL(scheme, host, repoKey, path string) (string, error) {
	filePath := p.Join(b.prefix, repoKey, path)

	req, _ := b.Client.GetObjectRequest(&s3.GetObjectInput{
		Bucket: aws.String(b.bucket),
		Key:    aws.String(filePath),
	})
	presignedURL, err := req.Presign(5 * time.Minute)
	if err != nil {
		return "", err
	}

	return presignedURL, nil
}

func (b *Backend) GetFilePath(key string) (string, error) {
	return "", e.ErrNotImplemented
}
