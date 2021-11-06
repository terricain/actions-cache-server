package storage

import (
	"errors"
	"io"

	"github.com/gin-gonic/gin"
	s3 "github.com/terrycain/actions-cache-server/pkg/storage/aws-s3"
	"github.com/terrycain/actions-cache-server/pkg/storage/disk"
)

type Backend interface {
	Setup() error
	Type() string
	Write(repoKey string, r io.Reader) (string, int64, error)
	Delete(repoKey string, path string) error
	GenerateArchiveURL(c *gin.Context, repoKey, path string) (string, error)
	GetFilePath(key string) (string, error)
}

func GetStorageBackend(backend, connectionString string) (Backend, error) {
	var b Backend
	var err error

	switch backend {
	case "disk":
		b, err = disk.New(connectionString)
	case "s3":
		b, err = s3.New(connectionString)
	default:
		return nil, errors.New("invalid storage backend")
	}

	if err != nil {
		return nil, err
	}

	if err := b.Setup(); err != nil {
		return nil, err
	}

	return b, nil
}
