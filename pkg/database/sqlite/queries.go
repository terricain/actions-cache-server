package sqlite

const InsertNewCache = `INSERT INTO cache ("repository","scope","key","version","cache_id","created_date","finished","size", "storage_backend") VALUES (
  $1,
  $2,
  $3,
  $4,
  (SELECT IIF(s.m IS NULL, 0, s.m) + 1 FROM (SELECT max(cache_id) AS m FROM cache WHERE repository = $1) s),
  $5,
  0,
  0,
  $6
) RETURNING "cache_id";`

const (
	SetCacheFinished   = `UPDATE cache SET finished = 1, storage_path = ? WHERE repository = ? AND cache_id = ?;`
	SearchCacheExact   = `SELECT created_date, storage_backend, storage_path, scope, key, version FROM cache WHERE repository = ? AND scope = ? AND key = ? AND version = ? AND finished = 1;`
	SearchCachePartial = `SELECT created_date, storage_backend, storage_path, scope, key, version
FROM cache 
WHERE repository = ? AND scope = ? AND finished = 1
ORDER BY scope, created_date DESC;`

	InsertPart = `INSERT INTO cache_parts ("repository", "cache_id", "start_byte", "end_byte", "size", "part_data")
VALUES (?, ?, ?, ?, ?, ?);`

	GetAllParts = `SELECT "start_byte", "end_byte", "size", "part_data" FROM cache_parts WHERE "repository" = ? AND "cache_id" = ? ORDER BY start_byte ASC`
)
