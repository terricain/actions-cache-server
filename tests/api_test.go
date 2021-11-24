package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/terrycain/actions-cache-server/pkg/e"
	"github.com/terrycain/actions-cache-server/pkg/s"
	"github.com/terrycain/actions-cache-server/pkg/web"
	"github.com/terrycain/actions-cache-server/tests/mock_backend"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"testing"
)

const JWT = "eyJ0eXAiOiJKV1QiLCJhbGciOiJSUzI1NiIsIng1dCI6IjJtM1VTZURvQ1ZtYzdOLXp2YmFpMTlEQ1VEbyJ9.eyJuYW1laWQiOiJkZGRkZGRkZC1kZGRkLWRkZGQtZGRkZC1kZGRkZGRkZGRkZGQiLCJzY3AiOiJBY3Rpb25zLkdlbmVyaWNSZWFkOjAwMDAwMDAwLTAwMDAtMDAwMC0wMDAwLTAwMDAwMDAwMDAwMCBBY3Rpb25zLlVwbG9hZEFydGlmYWN0czowMDAwMDAwMC0wMDAwLTAwMDAtMDAwMC0wMDAwMDAwMDAwMDAvMTpCdWlsZC9CdWlsZC8zMyBMb2NhdGlvblNlcnZpY2UuQ29ubmVjdCBSZWFkQW5kVXBkYXRlQnVpbGRCeVVyaTowMDAwMDAwMC0wMDAwLTAwMDAtMDAwMC0wMDAwMDAwMDAwMDAvMTpCdWlsZC9CdWlsZC8zMyIsIklkZW50aXR5VHlwZUNsYWltIjoiU3lzdGVtOlNlcnZpY2VJZGVudGl0eSIsImh0dHA6Ly9zY2hlbWFzLnhtbHNvYXAub3JnL3dzLzIwMDUvMDUvaWRlbnRpdHkvY2xhaW1zL3NpZCI6IkRERERERERELUREREQtRERERC1ERERELURERERERERERERERCIsImh0dHA6Ly9zY2hlbWFzLm1pY3Jvc29mdC5jb20vd3MvMjAwOC8wNi9pZGVudGl0eS9jbGFpbXMvcHJpbWFyeXNpZCI6ImRkZGRkZGRkLWRkZGQtZGRkZC1kZGRkLWRkZGRkZGRkZGRkZCIsImF1aSI6IjQ0OGVmMWY2LTMxNzAtNDEwNC04OTBiLTIwMmM1YmIzZmU3NSIsInNpZCI6IjUxNmQ5ODAzLTM4MzctNGEyZC05ZGUwLWI5NTdkMDhmYWU5OSIsImFjIjoiW3tcIlNjb3BlXCI6XCJyZWZzL2hlYWRzL21hc3RlclwiLFwiUGVybWlzc2lvblwiOjN9XSIsIm9yY2hpZCI6ImRiMTZlMzZmLTc1NmEtNGVmYi1hMTZjLTUwYjE4ZTRhMjdiNi50ZXN0Ll9fZGVmYXVsdCIsImlzcyI6InZzdG9rZW4uYWN0aW9ucy5naXRodWJ1c2VyY29udGVudC5jb20iLCJhdWQiOiJ2c3Rva2VuLmFjdGlvbnMuZ2l0aHVidXNlcmNvbnRlbnQuY29tfHZzbzpiNTIyYjg4Yi02MzFlLTQ1MTEtOGZiNi02YzI1OWM1YjI3NzIiLCJuYmYiOjE2MzU4OTM1MzQsImV4cCI6MTYzNTkxNjMzNH0.aJlSr8IW25Xihe3YTL5bAXHSVq1ZcbYgtx22YbSbywnntKaPP0FdzX0c4Be6XR83Or7PGFDj8tusnD4yE2D_BNHkOotLgXkkce569QBv2gjkgACD6vdALjP7eufC1AUiZip-p4NYp_j4W-giCuJtg2x_eJSVmsknwVhTffQeJN58T-sS1eIuZNLhx-gMfMmcJSU3N69BVGtKv6bcrgiCBwfLqPyroHZK_dyfOZQEPxH8Qqob3ImjHmJKyJfIhz8SAf4bjSNSPTSMBAp4Fe7_ca79ikPWVTEyTBcQOvG_zrgR26X9m-lQT_dibNV62Ir4-aY2xk52wKU93pUjBZSSaQ"
var Scopes = []s.Scope{{Scope: "refs/heads/master", Permission: 3}}


func getWebStuff(t *testing.T) (*gomock.Controller, *mock_backend.MockStorageBackend, *mock_backend.MockDatabaseBackend, *gin.Engine) {
	t.Helper()
	ctrl := gomock.NewController(t)

	storage := mock_backend.NewMockStorageBackend(ctrl)
	database := mock_backend.NewMockDatabaseBackend(ctrl)

	handler := web.Handlers{
		Storage:  storage,
		Database: database,
		Debug:    true,
	}

	router := web.GetRouter("", handler, false)

	return ctrl, storage, database, router
}

func TestSearchCacheExist(t *testing.T) {
	ctrl, storage, database, router := getWebStuff(t)
	defer ctrl.Finish()

	repoKey := uuid.NewString()
	cacheKey := "some-cache-key"
	version := uuid.NewString()

	returnResult := s.Cache{
		Scope:              Scopes[0].Scope,
		CacheKey:           cacheKey,
		CacheVersion:       version,
		CreationTime:       "2020-12-15T00:00:00Z",
		ArchiveLocation:    "",
		StorageBackendType: "disk",
		StorageBackendPath: "someopaquestring",
	}

	database.EXPECT().
		SearchCache(gomock.Eq(repoKey), gomock.Eq(cacheKey), gomock.Eq(version), gomock.Eq(Scopes), gomock.Eq([]string{cacheKey})).
		Times(1).
		Return(returnResult, nil)

	storage.EXPECT().Type().Return("disk").Times(1)

	storage.EXPECT().GenerateArchiveURL(gomock.Any(), gomock.Any(), repoKey, returnResult.StorageBackendPath).
		Times(1).
		Return("http://example.xxx", nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/" + repoKey + "/_apis/artifactcache/cache?keys=" + cacheKey + "&version=" + version, nil)
	req.Header.Add("Authorization", "Bearer " + JWT)
	router.ServeHTTP(w, req)

	if diff := cmp.Diff(200, w.Code); diff != "" {
		t.Fatal(diff)
	}

	var jsonData = make(map[string]string)
	if err := json.Unmarshal(w.Body.Bytes(), &jsonData); err != nil {
		t.Fatalf("Failed to unmarshal json from cache search: %#v", err.Error())
	}

	expectedResult := map[string]string {
		"scope": Scopes[0].Scope,
		"cacheKey": cacheKey,
		"cacheVersion": version,
		"creationTime": "2020-12-15T00:00:00Z",
		"archiveLocation": "http://example.xxx",
	}

	if diff := cmp.Diff(expectedResult, jsonData); diff != "" {
		t.Fatal(diff)
	}
}

func TestSearchCacheMissing(t *testing.T) {
	ctrl, _, database, router := getWebStuff(t)
	defer ctrl.Finish()

	repoKey := uuid.NewString()
	cacheKey := "some-cache-key"
	version := uuid.NewString()

	database.EXPECT().
		SearchCache(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Times(1).
		Return(s.Cache{}, e.ErrNoCacheFound)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/" + repoKey + "/_apis/artifactcache/cache?keys=" + cacheKey + "&version=" + version, nil)
	req.Header.Add("Authorization", "Bearer " + JWT)
	router.ServeHTTP(w, req)

	if diff := cmp.Diff(204, w.Code); diff != "" {
		t.Fatal(diff)
	}
}

func TestStartCache(t *testing.T) {
	ctrl, storage, database, router := getWebStuff(t)
	defer ctrl.Finish()

	repoKey := uuid.NewString()
	cacheKey := "some-cache-key"
	version := uuid.NewString()

	startCacheJSON := map[string]string{
		"key": cacheKey,
		"version": version,
	}
	startCacheData, _ := json.Marshal(&startCacheJSON)
	startCacheDataBuf := bytes.NewReader(startCacheData)

	storage.EXPECT().Type().Times(1).Return("disk")

	database.EXPECT().
		CreateCache(gomock.Eq(repoKey), gomock.Eq(cacheKey), gomock.Eq(version), gomock.Eq(Scopes), gomock.Eq("disk")).
		Times(1).
		Return(5, nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/" + repoKey + "/_apis/artifactcache/caches", startCacheDataBuf)
	req.Header.Add("Authorization", "Bearer " + JWT)
	router.ServeHTTP(w, req)

	if diff := cmp.Diff(201, w.Code); diff != "" {
		t.Fatal(diff)
	}

	var jsonData = make(map[string]int)
	if err := json.Unmarshal(w.Body.Bytes(), &jsonData); err != nil {
		t.Fatalf("Failed to unmarshal json from start cache: %#v", err.Error())
	}

	expectedResult := map[string]int {
		"cacheId": 5,
	}

	if diff := cmp.Diff(expectedResult, jsonData); diff != "" {
		t.Fatal(diff)
	}
}

func TestUploadCache(t *testing.T) {
	ctrl, storage, database, router := getWebStuff(t)
	defer ctrl.Finish()

	repoKey := uuid.NewString()
	size := 5*1024*1024
	uploadData := make([]byte, size)
	rand.Read(uploadData)
	uploadDataReader := bytes.NewReader(uploadData)

	part := s.CachePart{
		Start: 0,
		End:   size - 1,
		Size:  int64(size),
		Data:  "partData",
	}

	storage.EXPECT().Write(repoKey, gomock.Any(), 0, size -1, int64(size)).
		Times(1).
		Return("partData", int64(size), nil)

	database.EXPECT().
		AddUploadPart(repoKey, 5, part).
		Times(1).
		Return(nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PATCH", "/" + repoKey + "/_apis/artifactcache/caches/5", uploadDataReader)
	req.Header.Add("Authorization", "Bearer " + JWT)
	req.Header.Add("Content-Range", fmt.Sprintf("bytes %d-%d/*", 0, size-1))
	router.ServeHTTP(w, req)

	if diff := cmp.Diff(204, w.Code); diff != "" {
		t.Fatal(diff)
	}
}

func TestFinishCache(t *testing.T) {
	ctrl, storage, database, router := getWebStuff(t)
	defer ctrl.Finish()

	repoKey := uuid.NewString()
	size := 5*1024*1024

	finishCacheJSON := map[string]int{
		"size": size,
	}
	finishCacheData, _ := json.Marshal(&finishCacheJSON)
	finishCacheDataBuf := bytes.NewReader(finishCacheData)

	cacheParts := []s.CachePart{
		{0, size - 1, int64(size), "part"},
	}

	database.EXPECT().
		ValidateUpload(repoKey, 5, int64(size)).
		Times(1).
		Return(cacheParts, nil)

	storage.EXPECT().Finalise(repoKey, cacheParts).
		Times(1).
		Return("path", nil)

	database.EXPECT().FinishCache(repoKey, 5, "path").
		Times(1).
		Return(nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/" + repoKey + "/_apis/artifactcache/caches/5", finishCacheDataBuf)
	req.Header.Add("Authorization", "Bearer " + JWT)
	router.ServeHTTP(w, req)

	if diff := cmp.Diff(204, w.Code); diff != "" {
		t.Fatal(diff)
	}
}
