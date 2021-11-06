package sqlite

const InsertNewCache = `INSERT INTO cache ("repository","scope","key","version","cache_id","created_date","finished","size") VALUES (
  $1,
  $2,
  $3,
  $4,
  (SELECT IIF(s.m IS NULL, 0, s.m) + 1 FROM (SELECT max(cache_id) AS m FROM cache WHERE repository = $1) s),
  $5,
  0,
  0
) RETURNING "cache_id";`

const (
	GetCacheSize        = `SELECT size FROM cache WHERE repository = ? AND cache_id = ?;`
	SetCacheFinished    = `UPDATE cache SET finished = 1 WHERE repository = ? AND cache_id = ?;`
	SetCacheSizeBackend = `UPDATE cache SET size = ?, storage_backend = ?, storage_path = ? WHERE repository = ? AND cache_id = ?;`
	SearchCacheExact    = `SELECT created_date, storage_backend, storage_path, scope, key, version FROM cache WHERE repository = ? AND scope = ? AND key = ? AND version = ? AND finished = 1;`
	SearchCachePartial  = `SELECT created_date, storage_backend, storage_path, scope, key, version
FROM cache 
WHERE repository = ? AND scope = ? AND finished = 1
ORDER BY scope, created_date DESC;`
)
