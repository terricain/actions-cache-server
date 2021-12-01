package azureblob

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"

	"github.com/terrycain/actions-cache-server/pkg/s"

	"github.com/google/uuid"
	"github.com/terrycain/actions-cache-server/pkg/e"
)

type Backend struct {
	Client              azblob.ContainerClient
	container           string
	sharedKeyCredential *azblob.SharedKeyCredential
}

func ParsePartsFromConnectionString(connStr string) (string, string, string, bool) {
	container := ""
	account := ""
	key := ""

	parts := strings.Split(connStr, ";")
	for _, part := range parts {
		subParts := strings.SplitN(part, "=", 2)
		if len(subParts) < 2 {
			return "", "", "", false
		}

		if subParts[0] == "Container" {
			container = subParts[1]
		} else if subParts[0] == "AccountName" {
			account = subParts[1]
		}
		if subParts[0] == "AccountKey" {
			key = subParts[1]
		}
	}

	if container == "" || account == "" || key == "" {
		return "", "", "", false
	}

	return account, key, container, true
}

func New(connectionString string) (*Backend, error) {
	account, key, container, found := ParsePartsFromConnectionString(connectionString)
	if !found {
		return &Backend{}, errors.New("container missing from connection string")
	}

	creds, err := azblob.NewSharedKeyCredential(account, key)
	if err != nil {
		return &Backend{}, err
	}

	client, err := azblob.NewContainerClientFromConnectionString(connectionString, container, &azblob.ClientOptions{})
	if err != nil {
		return &Backend{}, err
	}

	// Enable uuid rand pool for better performance
	uuid.EnableRandPool()

	backend := Backend{
		container:           container,
		Client:              client,
		sharedKeyCredential: creds,
	}
	return &backend, nil
}

func (b *Backend) Setup() error {
	return nil
}

func (b *Backend) Type() string {
	return "azureblob"
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
func (b *Backend) Write(repoKey string, cacheID int, r io.Reader, start, end int, size int64) (string, int64, error) {
	cacheFile := fmt.Sprintf("%s_%d", repoKey, cacheID)
	blockID := fmt.Sprintf("%060d", start) // So the blockids must be of the same length, less than 64 chars before base64 encoding
	blockIDBase64 := base64.StdEncoding.EncodeToString([]byte(blockID))

	blobClient := b.Client.NewBlockBlobClient(cacheFile)

	// Hmm we need io.ReadSeekCloser but only have io.Reader, so seems making a temp file is easiest
	f, err := ioutil.TempFile(os.TempDir(), "blob-*")
	if err != nil {
		return "", 0, err
	}
	defer func() {
		name := f.Name()
		_ = f.Close()
		_ = os.Remove(name)
	}()

	a, _ := blobClient.GetBlockList(context.Background(), azblob.BlockListTypeAll, &azblob.GetBlockListOptions{})
	log.Info().Interface("a", a).Msg("a")

	if _, err = f.ReadFrom(r); err != nil {
		return "", 0, err
	}

	count, _ := f.Seek(0, io.SeekEnd)
	_, _ = f.Seek(0, io.SeekStart)

	if _, err = blobClient.StageBlock(context.Background(), blockIDBase64, f, &azblob.StageBlockOptions{}); err != nil {
		return "", 0, err
	}

	return blockID, count, nil
}

func (b *Backend) Delete(repoKey, path string) error {
	parts := strings.Split(path, "__")
	blobClient := b.Client.NewBlockBlobClient(parts[0])
	_, err := blobClient.Delete(context.Background(), &azblob.DeleteBlobOptions{})
	return err
}

func (b *Backend) Finalise(repoKey string, cacheID int, parts []s.CachePart) (string, error) {
	cacheFile := fmt.Sprintf("%s_%d", repoKey, cacheID)
	blobClient := b.Client.NewBlockBlobClient(cacheFile)

	blockIDList := make([]string, 0)
	for _, part := range parts {
		base64dID := base64.StdEncoding.EncodeToString([]byte(part.Data))
		blockIDList = append(blockIDList, base64dID)
	}

	_, err := blobClient.CommitBlockList(context.Background(), blockIDList, &azblob.CommitBlockListOptions{
		Metadata: map[string]string{
			"ownedBy":  "actions-cache-server",
			"repo":     repoKey,
			"cache_id": strconv.Itoa(cacheID),
		},
	})
	if err != nil {
		_, _ = blobClient.Delete(context.Background(), &azblob.DeleteBlobOptions{})
		return "", err
	}

	return cacheFile, nil
}

func (b *Backend) GenerateArchiveURL(scheme, host, repoKey, path string) (string, error) {
	blobClient := b.Client.NewBlockBlobClient(path)
	blobClientSharedKey, err := azblob.NewBlobClientWithSharedKey(blobClient.URL(), b.sharedKeyCredential, &azblob.ClientOptions{})
	if err != nil {
		return "", err
	}
	now := time.Now().Add(-1 * time.Minute)
	expire := time.Now().Add(5 * time.Minute)

	resp, err := blobClientSharedKey.GetSASToken(azblob.BlobSASPermissions{Read: true}, now, expire)
	if err != nil {
		return "", err
	}

	return blobClient.URL() + "?" + resp.Encode(), nil
}

func (b *Backend) GetFilePath(key string) (string, error) {
	return "", e.ErrNotImplemented
}
