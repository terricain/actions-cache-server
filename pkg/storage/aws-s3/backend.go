package awss3

import (
	"errors"
	"io"
	"net/url"
	p "path"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/terrycain/actions-cache-server/pkg/e"
)

type Backend struct {
	BucketURL string
	Session   *session.Session
	Client    *s3.S3

	bucket string
	prefix string
	region string
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
		region:    "us-east-1", // Region is calculated in Setup()
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

	b.bucket = parsedURL.Host
	b.prefix = strings.TrimPrefix(parsedURL.Path, "/")

	b.Client = s3.New(b.Session, &aws.Config{Region: aws.String(b.region)})
	resp, err := b.Client.GetBucketLocation(&s3.GetBucketLocationInput{Bucket: aws.String(b.bucket)})
	if err != nil {
		return err
	}

	if resp.LocationConstraint != nil {
		b.region = *resp.LocationConstraint
		b.Session.Config.Region = resp.LocationConstraint
		b.Client = s3.New(b.Session, &aws.Config{Region: resp.LocationConstraint})
	}

	return nil
}

func (b *Backend) Type() string {
	return "s3"
}

func (b *Backend) Write(repoKey string, r io.Reader) (string, int64, error) {
	cacheFile := uuid.New().String()
	filePath := p.Join(b.prefix, repoKey, cacheFile)

	uploader := s3manager.NewUploader(b.Session)
	_, err := uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(b.bucket),
		Body:   r,
		Key:    aws.String(filePath),
	})
	if err != nil {
		return "", 0, err
	}

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

func (b *Backend) GenerateArchiveURL(c *gin.Context, repoKey, path string) (string, error) {
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
