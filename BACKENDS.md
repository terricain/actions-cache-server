
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

Examples deployments coming soon