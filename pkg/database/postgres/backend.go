package postgres

import (
	"database/sql"
	"embed"
	"errors"
	"strings"
	"time"

	"github.com/golang-migrate/migrate/v4"
	gomigratepostgres "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/lib/pq"
	_ "github.com/lib/pq" // initialises postgres
	"github.com/rs/zerolog/log"
	"github.com/terrycain/actions-cache-server/pkg/e"
	"github.com/terrycain/actions-cache-server/pkg/s"
)

//go:embed migrations/*.sql
var fs embed.FS

type Backend struct {
	db *sql.DB
}

func NewPostgresBackend(connectionString string) (*Backend, error) {
	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		return &Backend{}, err
	}

	backend := Backend{
		db: db,
	}

	if err = backend.Migrate(); err != nil {
		return &Backend{}, err
	}

	return &backend, nil
}

func (b *Backend) Type() string { return "postgres" }

func (b *Backend) Migrate() error {
	driver, err := gomigratepostgres.WithInstance(b.db, &gomigratepostgres.Config{})
	if err != nil {
		return err
	}

	d, err := iofs.New(fs, "migrations")
	if err != nil {
		return err
	}
	m, err := migrate.NewWithInstance("iofs", d, "sqlite", driver)
	if err != nil {
		return err
	}

	log.Info().Msg("Starting database migrations")
	err = m.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	log.Info().Msg("Finished database migrations")

	return nil
}

func (b *Backend) SearchCache(repoKey, key, version string, scopes []s.Scope, restoreKeys []string) (s.Cache, error) {
	r := s.Cache{}

	// created_date, storage_backend, storage_path, scope, key, version
	err := b.db.QueryRow(SearchCacheExact, repoKey, scopes[0].Scope, key, version).Scan(&r.CreationTime, &r.StorageBackendType, &r.StorageBackendPath, &r.Scope, &r.CacheKey, &r.CacheVersion)
	if err != nil && err != sql.ErrNoRows {
		return s.Cache{}, err
	} else if err == sql.ErrNoRows {
		// Need to recursively look for caches
		// Yeah.... this isn't optimal, but it'll do till the logic of finding alternate caches is ironed out
		for _, scopeItem := range scopes {
			foundCache, found, err2 := b.lookupCacheForScope(repoKey, scopeItem.Scope, restoreKeys)

			if err2 != nil {
				return s.Cache{}, err2
			}
			if found {
				r = foundCache
				break
			}
		}

		// Not found anything
		if r.CacheKey == "" {
			return s.Cache{}, e.ErrNoCacheFound
		}
	}

	return r, nil
}

func (b *Backend) lookupCacheForScope(repoKey, scope string, restoreKeys []string) (s.Cache, bool, error) {
	potentialCache := make([]s.Cache, 0)

	rows, err := b.db.Query(SearchCachePartial, repoKey, scope)
	if err != nil {
		return s.Cache{}, false, err
	}
	defer rows.Close()

	for rows.Next() {
		newCache := s.Cache{}
		if err2 := rows.Scan(&newCache.CreationTime, &newCache.StorageBackendType, &newCache.StorageBackendPath, &newCache.Scope, &newCache.CacheKey, &newCache.CacheVersion); err2 != nil {
			return s.Cache{}, false, err2
		}
		potentialCache = append(potentialCache, newCache)
	}

	log.Debug().Interface("caches", potentialCache).Strs("restoreKeys", restoreKeys).Str("scope", scope).Msg("Potential caches")
	if err = rows.Err(); err != nil {
		return s.Cache{}, false, err
	}

	for _, restoreKey := range restoreKeys {
		for _, cacheItem := range potentialCache {
			if strings.HasPrefix(cacheItem.CacheKey, restoreKey) {
				return cacheItem, true, nil
			}
		}
	}

	return s.Cache{}, false, nil
}

func (b *Backend) CreateCache(repoKey, key, version string, scopes []s.Scope, backend string) (int, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	// err := b.db.QueryRow(SearchCacheExact, repoKey, scopes[0].Scope, key, version).Scan(&r.CreationTime, &r.StorageBackendType, &r.StorageBackendPath, &r.Scope, &r.CacheKey, &r.CacheVersion)
	var cacheID int64
	err := b.db.QueryRow(InsertNewCache, repoKey, scopes[0].Scope, key, version, now, backend).Scan(&cacheID)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) {
			// Trying to start cache twice
			if pqErr.Code == "23505" {
				return -1, e.ErrCacheAlreadyExists
			}
		}

		return -1, err
	}

	log.Debug().Int64("cache_id", cacheID).Msg("Created new cache")
	if err != nil {
		return -1, err
	}

	return int(cacheID), nil
}

func (b *Backend) ValidateUpload(repoKey string, id int, size int64) ([]s.CachePart, error) {
	rows, err := b.db.Query(GetAllParts, repoKey, id)
	if err != nil {
		return []s.CachePart{}, err
	}
	defer rows.Close()

	result := make([]s.CachePart, 0)
	totalSize := int64(0)
	nextStartByte := 0

	for rows.Next() {
		newPart := s.CachePart{}
		if err2 := rows.Scan(&newPart.Start, &newPart.End, &newPart.Size, &newPart.Data); err2 != nil {
			return []s.CachePart{}, err
		}

		// Check the parts are sequential, as they've been ordered by start
		if newPart.Start != nextStartByte {
			log.Warn().Str("repo", repoKey).Int("cache_id", id).Msg("Invalid upload ")
			return []s.CachePart{}, e.ErrCacheInvalidParts
		}
		nextStartByte = newPart.End + 1

		totalSize += newPart.Size
		result = append(result, newPart)
	}
	if err = rows.Err(); err != nil {
		return []s.CachePart{}, err
	}

	if totalSize != size {
		return []s.CachePart{}, e.ErrCacheSizeMismatch
	}

	return result, nil
}

func (b *Backend) FinishCache(repoKey string, id int, path string) error {
	result, err := b.db.Exec(SetCacheFinished, path, repoKey, id)
	if err != nil {
		return err
	}
	rowsAffected, _ := result.RowsAffected()
	log.Debug().Int64("rows", rowsAffected).Msg("Rows affected")

	return nil
}

func (b *Backend) AddUploadPart(repoKey string, id int, part s.CachePart) error {
	if _, err := b.db.Exec(InsertPart, repoKey, id, part.Start, part.End, part.Size, part.Data); err != nil {
		return err
	}

	return nil
}
