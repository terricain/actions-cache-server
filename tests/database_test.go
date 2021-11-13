package tests

import (
	"errors"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	db "github.com/terrycain/actions-cache-server/pkg/database"
	dbsqlite "github.com/terrycain/actions-cache-server/pkg/database/sqlite"
	"github.com/terrycain/actions-cache-server/pkg/e"
	"github.com/terrycain/actions-cache-server/pkg/s"
)

func TestMain(m *testing.M) {
	zerolog.SetGlobalLevel(zerolog.Disabled)
	if _, exists := os.LookupEnv("DEBUG"); exists {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	uuid.EnableRandPool()
	os.Exit(m.Run())
}

func GetSQLiteBackend(filepath string, t *testing.T) db.Backend {
	backend, err := dbsqlite.NewSQLiteBackend(filepath)
	if err != nil {
		t.Fatal(err)
	}
	return backend
}

func GetPostgresBackend(connectionURL string, t *testing.T) db.Backend {
	t.Fatal("Not implemented yet")
	return nil
}

func TestDatabaseBackends(t *testing.T) {
	// List of basic tests to cover database backend logic
	runTests := func(backend db.Backend, t *testing.T) {
		t.Run("type-string", testDBBackendTypeString(backend))
		t.Run("search-cache-exact", testSearchCacheExact(backend))
		t.Run("search-cache-missing", testSearchMissing(backend))
		t.Run("search-cache-restorekeys-samebranch", testSearchCacheRestoreKeySameBranch(backend))
		t.Run("search-cache-restorekeys-differentbranch", testSearchCacheRestoreKeyDifferentBranch(backend))
		t.Run("create-cache-duplicate-error", testDuplicateCreate(backend))
		t.Run("part-validation-success", testPartValidationSuccess(backend))
		t.Run("part-validation-missing-start", testPartValidationMissingStart(backend))
		t.Run("part-validation-missing-part", testPartValidationMissingPart(backend))
		t.Run("part-validation-incorrect-size", testPartValidationIncorrectSize(backend))
	}

	// SQLite backend
	t.Run("sqlite", func(t *testing.T) {
		backend := GetSQLiteBackend("file::memory:", t)

		runTests(backend, t)
	})

	// Postgres backend
	t.Run("postgres", func(t *testing.T) {
		pgURL := os.Getenv("DB_POSTGRES")
		if pgURL == "" {
			t.Skip("Skipped postgres as no env var")
		}
		backend := GetPostgresBackend(pgURL, t)

		runTests(backend, t)
	})
}

func addCompleteCacheEntry(repo, key, version, path string, scopes []s.Scope, backend db.Backend) error {
	log.Debug().Str("repo", repo).Str("key", key).Str("version", version).Interface("scopes", scopes).Msg("Adding cache")
	// Create cache entry
	cacheID, err := backend.CreateCache(repo, key, version, scopes, "somedb")
	if err != nil {
		return err
	}

	// Finalise cache entry
	if err = backend.FinishCache(repo, cacheID, path); err != nil {
		return err
	}
	return nil
}

func testDBBackendTypeString(backend db.Backend) func(t *testing.T) {
	return func(t *testing.T) {
		if len(backend.Type()) == 0 {
			t.Fatal("Backend needs a type string set")
		}
	}
}

func testSearchCacheExact(backend db.Backend) func(t *testing.T) {
	return func(t *testing.T) {
		repo := uuid.NewString()
		key := uuid.NewString()
		version := uuid.NewString()
		scopes := []s.Scope{{Scope: "refs/heads/master", Permission: 3}}

		// Add entry
		if err := addCompleteCacheEntry(repo, key, version, "somepath1", scopes, backend); err != nil {
			t.Fatalf("Failed to add cache entry: %s", err.Error())
		}

		cache, err := backend.SearchCache(repo, key, version, scopes, []string{})
		if err != nil {
			t.Fatalf("Failed to search cache entries: %s", err.Error())
		}

		if diff := cmp.Diff(key, cache.CacheKey); diff != "" {
			t.Fatal(diff)
		}
		if diff := cmp.Diff(scopes[0].Scope, cache.Scope); diff != "" {
			t.Fatal(diff)
		}
	}
}

func testSearchMissing(backend db.Backend) func(t *testing.T) {
	return func(t *testing.T) {
		repo := uuid.NewString()
		key := uuid.NewString()
		key2 := uuid.NewString()
		version := uuid.NewString()
		version2 := uuid.NewString()
		scopes := []s.Scope{{Scope: "refs/heads/master", Permission: 3}}

		// Add entry
		if err := addCompleteCacheEntry(repo, key, version, "somepatha", scopes, backend); err != nil {
			t.Fatalf("Failed to add cache entry: %s", err.Error())
		}

		_, err := backend.SearchCache(repo, key2, version2, scopes, []string{})
		if err == nil {
			t.Fatal("Cache search should return an error")
		}

		if !errors.Is(err, e.ErrNoCacheFound) {
			t.Fatalf("Error is not ErrNoCacheFound: %s", err.Error())
		}
	}
}

func testSearchCacheRestoreKeySameBranch(backend db.Backend) func(t *testing.T) {
	return func(t *testing.T) {
		repo := uuid.NewString()
		key := "some-prefix-" + uuid.NewString()
		key2 := "some-prefix-" + uuid.NewString()
		version := uuid.NewString()
		version2 := uuid.NewString()
		scopes := []s.Scope{{Scope: "refs/heads/master", Permission: 3}}

		// Add entry
		if err := addCompleteCacheEntry(repo, key, version, "somepathb", scopes, backend); err != nil {
			t.Fatalf("Failed to add cache entry: %s", err.Error())
		}

		cache, err := backend.SearchCache(repo, key2, version2, scopes, []string{"some-prefix-"})
		if err != nil {
			t.Fatalf("Failed to search cache entries: %s", err.Error())
		}

		if diff := cmp.Diff(key, cache.CacheKey); diff != "" {
			t.Fatal(diff)
		}
		if diff := cmp.Diff(scopes[0].Scope, cache.Scope); diff != "" {
			t.Fatal(diff)
		}
	}
}

func testSearchCacheRestoreKeyDifferentBranch(backend db.Backend) func(t *testing.T) {
	return func(t *testing.T) {
		repo := uuid.NewString()
		key := "some-prefix-" + uuid.NewString()
		key2 := "some-prefix-" + uuid.NewString()
		version := uuid.NewString()
		version2 := uuid.NewString()
		scopes := []s.Scope{{Scope: "refs/heads/master", Permission: 3}}
		scopes2 := []s.Scope{{Scope: "refs/heads/test", Permission: 3}, {Scope: "refs/heads/master", Permission: 1}}

		// Add entry
		if err := addCompleteCacheEntry(repo, key, version, "somepathn", scopes, backend); err != nil {
			t.Fatalf("Failed to add cache entry: %s", err.Error())
		}

		cache, err := backend.SearchCache(repo, key2, version2, scopes2, []string{"some-prefix-"})
		if err != nil {
			t.Fatalf("Failed to search cache entries: %s", err.Error())
		}

		if diff := cmp.Diff(key, cache.CacheKey); diff != "" {
			t.Fatal(diff)
		}
		if diff := cmp.Diff(scopes[0].Scope, cache.Scope); diff != "" {
			t.Fatal(diff)
		}
	}
}

func testDuplicateCreate(backend db.Backend) func(t *testing.T) {
	return func(t *testing.T) {
		repo := uuid.NewString()
		key := uuid.NewString()
		version := uuid.NewString()
		scopes := []s.Scope{{Scope: "refs/heads/master", Permission: 3}}

		// Add entry
		if err := addCompleteCacheEntry(repo, key, version, "somepath", scopes, backend); err != nil {
			t.Fatalf("Failed to add cache entry: %s", err.Error())
		}

		_, err := backend.CreateCache(repo, key, version, scopes, "somedb")
		if err == nil {
			t.Fatalf("Cache create should return an error")
		}

		if !errors.Is(err, e.ErrCacheAlreadyExists) {
			t.Fatalf("Error is not ErrCacheAlreadyExists: %s", err.Error())
		}
	}
}

func testPartValidationSuccess(backend db.Backend) func(t *testing.T) {
	return func(t *testing.T) {
		repo := uuid.NewString()
		key := uuid.NewString()
		version := uuid.NewString()
		scopes := []s.Scope{{Scope: "refs/heads/master", Permission: 3}}

		cacheID, err := backend.CreateCache(repo, key, version, scopes, "somedb")
		if err != nil {
			t.Fatalf("Failed to start cache entry: %s", err.Error())
		}

		expectedParts := []s.CachePart{
			{Start: 0, End: 100, Size: 101, Data: "somedata"},
			{Start: 101, End: 201, Size: 101, Data: "somedata"},
		}

		// Add parts out of order
		if partErr := backend.AddUploadPart(repo, cacheID, expectedParts[1]); partErr != nil {
			t.Fatal(partErr)
		}
		if partErr := backend.AddUploadPart(repo, cacheID, expectedParts[0]); partErr != nil {
			t.Fatal(partErr)
		}

		parts, err := backend.ValidateUpload(repo, cacheID, 202)
		if err != nil {
			t.Fatalf("Failed to validate cache parts: %s", err.Error())
		}

		// Ensure correct number of parts left and that they are in order
		if diff := cmp.Diff(expectedParts, parts); diff != "" {
			t.Fatal(diff)
		}
	}
}

func testPartValidationMissingStart(backend db.Backend) func(t *testing.T) {
	return func(t *testing.T) {
		repo := uuid.NewString()
		key := uuid.NewString()
		version := uuid.NewString()
		scopes := []s.Scope{{Scope: "refs/heads/master", Permission: 3}}

		cacheID, err := backend.CreateCache(repo, key, version, scopes, "somedb")
		if err != nil {
			t.Fatalf("Failed to start cache entry: %s", err.Error())
		}

		expectedParts := []s.CachePart{
			{Start: 1, End: 100, Size: 100, Data: "somedata"},
		}

		// Add parts out of order
		if partErr := backend.AddUploadPart(repo, cacheID, expectedParts[0]); partErr != nil {
			t.Fatal(partErr)
		}

		_, err = backend.ValidateUpload(repo, cacheID, 100)
		if err == nil {
			t.Fatal("Cache parts validation should return an error")
		}

		if !errors.Is(err, e.ErrCacheInvalidParts) {
			t.Fatalf("Error is not ErrCacheInvalidParts: %s", err.Error())
		}
	}
}

func testPartValidationIncorrectSize(backend db.Backend) func(t *testing.T) {
	return func(t *testing.T) {
		repo := uuid.NewString()
		key := uuid.NewString()
		version := uuid.NewString()
		scopes := []s.Scope{{Scope: "refs/heads/master", Permission: 3}}

		cacheID, err := backend.CreateCache(repo, key, version, scopes, "somedb")
		if err != nil {
			t.Fatalf("Failed to start cache entry: %s", err.Error())
		}

		expectedParts := []s.CachePart{
			{Start: 0, End: 100, Size: 101, Data: "somedata"},
		}

		// Add parts out of order
		if partErr := backend.AddUploadPart(repo, cacheID, expectedParts[0]); partErr != nil {
			t.Fatal(partErr)
		}

		_, err = backend.ValidateUpload(repo, cacheID, 100)
		if err == nil {
			t.Fatal("Cache parts validation should return an error")
		}

		if !errors.Is(err, e.ErrCacheSizeMismatch) {
			t.Fatalf("Error is not ErrCacheSizeMismatch: %s", err.Error())
		}
	}
}

func testPartValidationMissingPart(backend db.Backend) func(t *testing.T) {
	return func(t *testing.T) {
		repo := uuid.NewString()
		key := uuid.NewString()
		version := uuid.NewString()
		scopes := []s.Scope{{Scope: "refs/heads/master", Permission: 3}}

		cacheID, err := backend.CreateCache(repo, key, version, scopes, "somedb")
		if err != nil {
			t.Fatalf("Failed to start cache entry: %s", err.Error())
		}

		expectedParts := []s.CachePart{
			{Start: 0, End: 100, Size: 101, Data: "somedata"},
			{Start: 201, End: 301, Size: 101, Data: "somedata"},
		}

		if partErr := backend.AddUploadPart(repo, cacheID, expectedParts[0]); partErr != nil {
			t.Fatal(partErr)
		}
		if partErr := backend.AddUploadPart(repo, cacheID, expectedParts[1]); partErr != nil {
			t.Fatal(partErr)
		}

		_, err = backend.ValidateUpload(repo, cacheID, 202)
		if err == nil {
			t.Fatal("Cache parts validation should return an error")
		}

		if !errors.Is(err, e.ErrCacheInvalidParts) {
			t.Fatalf("Error is not ErrCacheInvalidParts: %s", err.Error())
		}
	}
}
