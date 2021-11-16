package tests

import (
	"bytes"
	"math/rand"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/terrycain/actions-cache-server/pkg/s"
	"github.com/terrycain/actions-cache-server/pkg/storage"
	s3backend "github.com/terrycain/actions-cache-server/pkg/storage/aws-s3"
	"github.com/terrycain/actions-cache-server/pkg/storage/disk"
)

func GetDiskBackend(filepath string, t *testing.T) storage.Backend {
	t.Helper()
	backend, err := disk.New(filepath)
	if err != nil {
		t.Fatal(err)
	}
	if err = backend.Setup(); err != nil {
		t.Fatal(err)
	}
	return backend
}

func GetS3Backend(t *testing.T, localstack string) storage.Backend {
	t.Helper()
	bucket := uuid.NewString()

	// Create s3 bucket
	sess, err := session.NewSession(&aws.Config{
		Region:           aws.String("us-east-1"),
		Endpoint:         aws.String(localstack),
		DisableSSL:       aws.Bool(strings.HasPrefix(localstack, "http://")),
		Credentials:      credentials.NewStaticCredentials("test", "test", ""),
		S3ForcePathStyle: aws.Bool(true),
	})
	if err != nil {
		t.Fatal(err)
	}

	s3Client := s3.New(sess, sess.Config)
	_, err = s3Client.CreateBucket(&s3.CreateBucketInput{
		Bucket:                    aws.String(bucket),
		CreateBucketConfiguration: &s3.CreateBucketConfiguration{LocationConstraint: aws.String("eu-west-1")},
	})
	if err != nil {
		t.Fatal(err)
	}

	query := url.Values{}
	query.Add("localstack", localstack)
	URL := url.URL{
		Scheme:      "s3",
		Host:        bucket,
		Path:        "someprefix",
		RawQuery: query.Encode(),
	}

	backend, err := s3backend.New(URL.String())
	if err != nil {
		t.Fatal(err)
	}

	if err = backend.Setup(); err != nil {
		t.Fatal(err)
	}
	return backend
}

// TestStorageBackends performs basic tests over all the storage backends.
// As we're not testing from the backend package itself, these tests are mote
// generic and don't necessarily test that the actual files have been stored
// correctly, more that the storage backends return and are not horrifically
// broken. More tests will be done in e2e tests.
func TestStorageBackends(t *testing.T) {
	// List of basic tests to cover database backend logic
	runTests := func(backend storage.Backend, t *testing.T) {
		t.Run("type-string", testStorageBackendTypeString(backend))
		t.Run("part-upload-delete", testPartUploadDelete(backend))
		t.Run("part-upload-finalise", testPartUploadFinalise(backend))
		t.Run("multi-part-upload-finalise", testMultiPartUploadFinalise(backend))
	}

	// Disk backend
	t.Run("disk", func(t *testing.T) {
		testDir, err := os.MkdirTemp(os.TempDir(), "disk-cache-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir for disk cache tests: %s", err.Error())
		}
		backend := GetDiskBackend(testDir, t)

		runTests(backend, t)

		os.RemoveAll(testDir)
	})

	t.Run("s3", func(t *testing.T) {
		s3Endpoint := os.Getenv("STORAGE_S3")
		if s3Endpoint == "" {
			t.Skip("Skipped postgres as no env var")
		}
		backend := GetS3Backend(t, s3Endpoint)

		runTests(backend, t)
	})
}

func testStorageBackendTypeString(backend storage.Backend) func(t *testing.T) {
	return func(t *testing.T) {
		if len(backend.Type()) == 0 {
			t.Fatal("Backend needs a type string set")
		}
	}
}

// testPartUploadDelete Delete would be called if uploading a part completed and then needs to be removed / cleaned up.
func testPartUploadDelete(backend storage.Backend) func(t *testing.T) {
	return func(t *testing.T) {
		repo := uuid.NewString()
		randBuf := make([]byte, 6*1024*1024)

		if _, err := rand.Read(randBuf); err != nil {
			t.Fatalf("Failed to generate random file data: %s", err.Error())
		}
		r := bytes.NewReader(randBuf)

		partData, bytesWritten, err := backend.Write(repo, r, 0, len(randBuf)-1, int64(len(randBuf)))
		if err != nil {
			t.Fatalf("Failed to write part: %s", err.Error())
		}
		if len(partData) == 0 {
			t.Fatal("b.Write() part data should be not null")
		}
		if diff := cmp.Diff(int64(len(randBuf)), bytesWritten); diff != "" {
			t.Fatal(diff)
		}
	}
}

// testPartUploadFinalise Tests uploading a part and generating a downloadable url.
func testPartUploadFinalise(backend storage.Backend) func(t *testing.T) {
	return func(t *testing.T) {
		repo := uuid.NewString()
		randBuf := make([]byte, 6*1024*1024)

		if _, err := rand.Read(randBuf); err != nil {
			t.Fatalf("Failed to generate random file data: %s", err.Error())
		}
		r := bytes.NewReader(randBuf)

		partData, _, err := backend.Write(repo, r, 0, len(randBuf)-1, int64(len(randBuf)))
		if err != nil {
			t.Fatalf("Failed to write part: %s", err.Error())
		}

		parts := []s.CachePart{{Start: 0, End: len(randBuf) - 1, Size: int64(len(randBuf)), Data: partData}}
		path, err := backend.Finalise(repo, parts)
		if err != nil {
			t.Fatalf("Failed to finialise cache archive: %s", err.Error())
		}

		archiveURL, err := backend.GenerateArchiveURL("https", "somehostname.com", repo, path)
		if err != nil {
			t.Fatalf("Failed to convert path to archive URL: %s", err.Error())
		}

		// Not checking scheme or host as anything downloadable should work i.e S3 pre-signed urls
		if _, err = url.Parse(archiveURL); err != nil {
			t.Fatalf("Archive URL is not valid: %s", err.Error())
		}
	}
}

// testMultiPartUploadFinalise Tests uploading and concatenating multiple parts and generating a downloadable url.
func testMultiPartUploadFinalise(backend storage.Backend) func(t *testing.T) {
	return func(t *testing.T) {
		repo := uuid.NewString()
		randBuf := make([]byte, 6*1024*1024)
		randBuf2 := make([]byte, 3*1024*1024)
		r1Start := 0
		r1End := len(randBuf) - 1
		r1Size := int64(len(randBuf))
		r2Start := r1End + 1
		r2End := r2Start + len(randBuf2) - 1
		r2Size := int64(len(randBuf2))

		if _, err := rand.Read(randBuf); err != nil {
			t.Fatalf("Failed to generate random file data: %s", err.Error())
		}
		if _, err := rand.Read(randBuf2); err != nil {
			t.Fatalf("Failed to generate random file data: %s", err.Error())
		}
		r1 := bytes.NewReader(randBuf)
		r2 := bytes.NewReader(randBuf)

		partData1, _, err := backend.Write(repo, r1, r1Start, r1End, r1Size)
		if err != nil {
			t.Fatalf("Failed to write part: %s", err.Error())
		}
		partData2, _, err := backend.Write(repo, r2, r2Start, r2End, r2Size)
		if err != nil {
			t.Fatalf("Failed to write part: %s", err.Error())
		}

		parts := []s.CachePart{
			{Start: r1Start, End: r1End, Size: r1Size, Data: partData1},
			{Start: r2Start, End: r2End, Size: r2Size, Data: partData2},
		}
		path, err := backend.Finalise(repo, parts)
		if err != nil {
			t.Fatalf("Failed to finialise cache archive: %s", err.Error())
		}

		archiveURL, err := backend.GenerateArchiveURL("https", "somehostname.com", repo, path)
		if err != nil {
			t.Fatalf("Failed to convert path to archive URL: %s", err.Error())
		}

		// Not checking scheme or host as anything downloadable should work i.e S3 pre-signed urls
		if _, err = url.Parse(archiveURL); err != nil {
			t.Fatalf("Archive URL is not valid: %s", err.Error())
		}
	}
}
