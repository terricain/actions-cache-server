package web

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"github.com/terrycain/actions-cache-server/pkg/database"
	"github.com/terrycain/actions-cache-server/pkg/e"
	"github.com/terrycain/actions-cache-server/pkg/s"
	"github.com/terrycain/actions-cache-server/pkg/storage"
	"github.com/terrycain/actions-cache-server/pkg/utils"
)

type Handlers struct {
	Storage  storage.Backend
	Database database.Backend
	Debug    bool
}

func (h *Handlers) SearchCache(c *gin.Context) {
	repo := c.Param("repo")
	scopes := c.MustGet("scopes").([]s.Scope)
	key := c.Query("keys")
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing keys query parameter"})
		return
	}
	version := c.Query("version")
	if version == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing version query parameter"})
		return
	}
	keys := utils.CleanStringSlice(strings.Split(key, ","))
	if len(keys) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid keys"})
		return
	}
	primaryKey := keys[0]

	// Search path on failure goes through all keys through each scope
	cache, err := h.Database.SearchCache(repo, primaryKey, version, scopes, keys)
	if err != nil {
		if errors.Is(err, e.ErrNoCacheFound) {
			c.Data(http.StatusNoContent, gin.MIMEJSON, nil)
		} else {
			log.Error().Err(err).Msg("Failed to search for cache")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to search for cache entry"})
		}
		return
	}

	// If storage backend != current, then return 204 as the backend path isnt usable to generate an archive url
	if h.Storage.Type() != cache.StorageBackendType {
		c.Data(http.StatusNoContent, gin.MIMEJSON, nil)
	}

	archiveURL, err := h.Storage.GenerateArchiveURL(c, repo, cache.StorageBackendPath)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get archive url")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get archive url"})
	}

	cache.ArchiveLocation = archiveURL
	log.Debug().Str("url", cache.ArchiveLocation).Msg("Archive location")

	c.JSON(http.StatusCreated, cache)
}

type StartCacheRequest struct {
	Key     string `json:"key"`
	Version string `json:"version"`
}

type StartCacheResponse struct {
	CacheID int `json:"cacheId"`
}

func (h *Handlers) StartCache(c *gin.Context) {
	scopes := c.MustGet("scopes").([]s.Scope)

	var json StartCacheRequest
	if err := c.ShouldBindJSON(&json); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	repo := c.Param("repo")

	id, err := h.Database.CreateCache(repo, json.Key, json.Version, scopes, h.Storage.Type())
	if err != nil {
		if errors.Is(err, e.ErrCacheAlreadyExists) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		} else {
			log.Error().Err(err).Msg("Failed to create cache")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create cache"})
		}
		return
	}

	c.JSON(http.StatusCreated, StartCacheResponse{CacheID: id})
}

func (h *Handlers) UploadCache(c *gin.Context) {
	repo := c.Param("repo")
	cacheID, err := strconv.Atoi(c.Param("cacheid"))
	if err != nil || cacheID < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cacheId must be a positive integer"})
		return
	}

	parsedContentRange, err := ParseContentRange(c.GetHeader("Content-Range"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "missing or invalid content-range header"})
		return
	}
	calculatedSize := int64(parsedContentRange.End - (parsedContentRange.Start - 1)) // Seems to be inclusive of the range 0-10/* == 11 bytes

	partData, bytesWritten, err := h.Storage.Write(repo, c.Request.Body, parsedContentRange.Start, parsedContentRange.End, calculatedSize)
	if err != nil {
		log.Error().Err(err).Msg("Failed to store file")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to store file"})
		return
	}

	if bytesWritten != calculatedSize {
		_ = h.Storage.Delete(repo, partData)
		log.Error().Int64("calculated_size", calculatedSize).Int64("bytes_written", bytesWritten).Msg("Calculated size vs bytes written mismatch")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "calculated size vs bytes written mismatch"})
		return
	}

	part := s.CachePart{
		Start: parsedContentRange.Start,
		End:   parsedContentRange.End,
		Size:  bytesWritten,
		Data:  partData,
	}
	if err = h.Database.AddUploadPart(repo, cacheID, part); err != nil {
		_ = h.Storage.Delete(repo, partData) // Attempt to clean up file as we've failed to save it to db
		log.Error().Err(err).Msg("Failed to store file")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to store file"})
		return
	}

	c.Data(http.StatusNoContent, gin.MIMEJSON, nil)
}

type FinishCacheRequest struct {
	Size int64 `json:"size"`
}

func (h *Handlers) FinishCache(c *gin.Context) {
	repo := c.Param("repo")
	cacheID, err := strconv.Atoi(c.Param("cacheid"))
	if err != nil || cacheID < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cacheId must be a positive integer"})
		return
	}

	var json FinishCacheRequest
	if err = c.ShouldBindJSON(&json); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check all the parts are good
	parts, err := h.Database.ValidateUpload(repo, cacheID, json.Size)
	if err != nil {
		if errors.Is(err, e.ErrCacheSizeMismatch) || errors.Is(err, e.ErrCacheInvalidParts) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		} else {
			log.Error().Err(err).Msg("Failed to finalise cache")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to finalise cache"})
		}
		return
	}

	path, err := h.Storage.Finalise(repo, parts)
	if err != nil {
		log.Error().Err(err).Msg("Failed to finalise cache file")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to finalise cache file"})
		return
	}

	err = h.Database.FinishCache(repo, cacheID, path)
	if err != nil {
		log.Error().Err(err).Msg("Failed to finalise cache entry")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to finalise cache entry"})
		return
	}

	c.Data(http.StatusNoContent, gin.MIMEJSON, nil)
}

func (h *Handlers) ArchivePath(c *gin.Context) {
	key := c.Param("key")

	if h.Storage.Type() != "disk" {
		c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
	}

	path, err := h.Storage.GetFilePath(key)
	if err != nil {
		if errors.Is(err, e.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
		} else {
			log.Error().Err(err).Msg("Failed to get file")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get file"})
		}
		return
	}

	c.File(path)
}
