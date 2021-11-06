package disk

import (
	"errors"
	"io"
	"net/url"
	"os"
	p "path"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/terrycain/actions-cache-server/pkg/e"
)

type Backend struct {
	BaseDir string
}

func New(connectionString string) (*Backend, error) {
	if _, err := os.Stat(connectionString); os.IsNotExist(err) {
		return nil, errors.New("path does not exist")
	}

	// Enable uuid rand pool for better performance
	uuid.EnableRandPool()

	backend := Backend{BaseDir: connectionString}
	return &backend, nil
}

func (b *Backend) Setup() error {
	return nil
}

func (b *Backend) Type() string {
	return "disk"
}

func (b *Backend) Write(repoKey string, r io.Reader) (string, int64, error) {
	cacheFile := uuid.New().String()
	filePath := p.Join(b.BaseDir, cacheFile)

	fp, err := os.OpenFile(filePath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return "", 0, err
	}

	writtenBytes, err := io.Copy(fp, r)
	_ = fp.Close()

	if err != nil {
		_ = os.Remove(filePath)
		return "", 0, err
	}

	return cacheFile, writtenBytes, nil
}

func (b *Backend) Delete(repoKey, path string) error {
	filePath := p.Join(b.BaseDir, path)
	return os.Remove(filePath)
}

func (b *Backend) GenerateArchiveURL(c *gin.Context, repoKey, path string) (string, error) {
	urlPath := "/archive/" + path

	archiveURL := url.URL{
		Scheme: c.Request.URL.Scheme,
		Host:   c.Request.Host,
		Path:   urlPath,
	}
	return archiveURL.String(), nil
}

func (b *Backend) GetFilePath(key string) (string, error) {
	filePath := p.Clean(p.Join(b.BaseDir, key))
	if !strings.HasPrefix(filePath, b.BaseDir) {
		return "", e.ErrNotFound
	}

	return filePath, nil
}
