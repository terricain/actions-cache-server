
# Databases

| Database | CLI Argument    | Environment Variable | Description             | Example Value                                     |
|----------|-----------------|----------------------|-------------------------|---------------------------------------------------|
| SQLite   | `--db-sqlite`   | `DB_SQLITE`          | SQLite database         | `/tmp/db.sqlite`                                  |
| Postgres | `--db-postgres` | `DB_POSTGRES`        | **not implemented yet** | `postgresql://user:pass@host:port/dbname?options` |

Examples deployments coming soon

# Storage Backends
| Store  | CLI Argument     | Environment Variable | Description                                                                | Example Value            |
|--------|------------------|----------------------|----------------------------------------------------------------------------|--------------------------|
| Disk   | `--storage-disk` | `STORAGE_DISK`       | Filesystem based cache storage. The directory is expected to already exist | `/tmp/cache`             |
| AWS S3 | `--storage-s3`   | `STORAGE_S3`         | S3 storage. Not been tested with anything fancy like KMS yet.              | `s3://bucketname/prefix` |

The cache plugin supports an `upload-chunk-size` parameter, if using S3 either don't specify this as it'll default to 32MiB or make sure its greater than 5MiB 
that is the minimum size to start a multipart upload and this server makes the assumption that a chunked upload greater than 5MiB can be uploaded using multipart.
Otherwise it becomes a pain as chunked uploads can upload multiple sections of a file in parallel and one needs to recombine them which is simple with either only
1 chunk, or S3's built in multipart uploads.

Examples deployments coming soon