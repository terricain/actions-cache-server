package disk

import (
	"errors"
	"io"
	"net/url"
	"os"
	p "path"
	"strings"

	"github.com/terrycain/actions-cache-server/pkg/s"

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

func (b *Backend) Write(repoKey string, cacheID int, r io.Reader, start, end int, size int64) (string, int64, error) {
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

func (b *Backend) Delete(repoKey, partData string) error {
	filePath := p.Join(b.BaseDir, partData)
	return os.Remove(filePath)
}

func (b *Backend) GenerateArchiveURL(scheme, host, repoKey, path string) (string, error) {
	urlPath := "/archive/" + path

	archiveURL := url.URL{
		Scheme: scheme,
		Host:   host,
		Path:   urlPath,
	}
	return archiveURL.String(), nil
}

func (b *Backend) Finalise(repoKey string, cacheID int, parts []s.CachePart) (string, error) {
	cacheFile := uuid.New().String()
	filePath := p.Join(b.BaseDir, cacheFile)

	fp, err := os.OpenFile(filePath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return "", err
	}
	defer fp.Close()

	var loopErr error

	for _, part := range parts {
		partPath := p.Join(b.BaseDir, part.Data)
		partFp, err2 := os.OpenFile(partPath, os.O_RDONLY, 0o644)
		if err2 != nil {
			loopErr = err2
			break
		}

		_, err2 = io.Copy(fp, partFp)
		_ = partFp.Close()
		if err2 != nil {
			loopErr = err2
			break
		}

		// We've written part of the file, delete current part now as best cast we'll never need it again
		_ = os.Remove(partPath)
	}

	if loopErr != nil { // Got an error, try and clean up FS
		_ = fp.Close()
		_ = os.Remove(filePath)
		for _, part := range parts {
			_ = b.Delete(repoKey, part.Data)
		}
		return "", loopErr
	}

	return cacheFile, nil
}

func (b *Backend) GetFilePath(key string) (string, error) {
	filePath := p.Clean(p.Join(b.BaseDir, key))
	if !strings.HasPrefix(filePath, b.BaseDir) {
		return "", e.ErrNotFound
	}

	return filePath, nil
}
