package postgres

const InsertNewCache = `INSERT INTO cache ("repository","scope","key","version","created_date", "storage_backend") VALUES (
  $1, $2, $3, $4, $5, $6
) RETURNING "cache_id";`

const (
	SetCacheFinished   = `UPDATE cache SET finished = TRUE, storage_path = $1 WHERE repository = $2 AND cache_id = $3;`
	SearchCacheExact   = `SELECT created_date, storage_backend, storage_path, scope, key, version FROM cache WHERE repository = $1 AND scope = $2 AND key = $3 AND version = $4 AND finished = TRUE;`
	SearchCachePartial = `SELECT created_date, storage_backend, storage_path, scope, key, version
FROM cache 
WHERE repository = $1 AND scope = $2 AND finished = TRUE
ORDER BY scope, created_date DESC;`
	InsertPart  = `INSERT INTO cache_part ("repository", "cache_id", "start_byte", "end_byte", "size", "part_data") VALUES ($1, $2, $3, $4, $5, $6);`
	GetAllParts = `SELECT "start_byte", "end_byte", "size", "part_data" FROM cache_part WHERE "repository" = $1 AND "cache_id" = $2 ORDER BY start_byte ASC`
)
