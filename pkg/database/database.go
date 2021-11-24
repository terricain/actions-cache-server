package database

import (
	"errors"

	"github.com/terrycain/actions-cache-server/pkg/database/postgres"
	"github.com/terrycain/actions-cache-server/pkg/database/sqlite"
	"github.com/terrycain/actions-cache-server/pkg/s"
)

type Backend interface {
	Type() string
	SearchCache(repoKey, key, version string, scopes []s.Scope, restoreKeys []string) (s.Cache, error)
	CreateCache(repoKey, key, version string, scopes []s.Scope, backend string) (int, error)
	AddUploadPart(repoKey string, id int, part s.CachePart) error
	ValidateUpload(repoKey string, id int, size int64) ([]s.CachePart, error)
	FinishCache(repoKey string, id int, path string) error
}

func GetBackend(backend, connectionString string) (Backend, error) {
	switch backend {
	case "sqlite":
		return sqlite.NewSQLiteBackend(connectionString)
	case "postgres":
		return postgres.NewPostgresBackend(connectionString)
	default:
		return nil, errors.New("invalid storage backend")
	}
}
