# GitHub Actions Cache Server for self-hosted runners

# How to run:

The cache server is composed of 2 main areas, a storage backend and a database backend. 
The database is purely responsible for maintaining some metadata around caches stored.

Once you have the server running, you'll need to setup the cache action to use that as an external cache server. The cache
plugin needs to be modified to use an external server, I have said modified cache action [here](https://github.com/terrycain/cache). I intend to upstream
these changes at some point, for those that are interested the changes are on the `custom-url` branch.

```shell
mkdir db cache

docker run --rm -it -p8080:8080 -v $(pwd)/db:/tmp/db -v $(pwd)/cache:/tmp/cache INSERT_GHCR_IMAGE_HERE \
  --db-sqlite /tmp/db/db.sqlite \
  --storage-disk /tmp/cache \
  --listen-address 0.0.0.0:8080
```

## Database Backends
| Database | CLI Argument    | Environment Variable | Description             | Example Value                                     |
|----------|-----------------|----------------------|-------------------------|---------------------------------------------------|
| SQLite   | `--db-sqlite`   | `DB_SQLITE`          | SQLite database         | `/tmp/db.sqlite`                                  |
| Postgres | `--db-postgres` | `DB_POSTGRES`        | **not implemented yet** | `postgresql://user:pass@host:port/dbname?options` |

## Storage Backends
| Store  | CLI Argument     | Environment Variable | Description                                                                | Example Value            |
|--------|------------------|----------------------|----------------------------------------------------------------------------|--------------------------|
| Disk   | `--storage-disk` | `STORAGE_DISK`       | Filesystem based cache storage. The directory is expected to already exist | `/tmp/cache`             |
| AWS S3 | `--storage-s3`   | `STORAGE_S3`         | S3 storage. Not been tested with anything fancy like KMS yet.              | `s3://bucketname/prefix` |

## Example cache YAML

This is a fork of the `actions/cache` action with an external-url parameter added. The external URL should end in a / and not have any additional path as the
action preserves some path that would usually be sent to the GitHub cache API as the path identifies a repository.

```yaml
- name: Cache test
  uses: terrycain/cache@custom-url
  with:
    external-url: "http://172.20.0.20:8080/"
    path: /tmp/test1234/
    key: ${{ runner.os }}-docker-${{ github.sha }}
    restore-keys: |
      ${{ runner.os }}-docker-
```


### Why:
I ran into an issue where I am running self-hosted runners on-premise to access some local resources.
I wanted some caching, which was fine, but then I ended up with a rather large cache and uploading that from where the runners are took ages hence this project :)

### What doesn't work
* Sharing caches on forks
* Probably something else I've not come across

### Roadmap: 

^^ sounds better than a todo list

* ~~Add S3 Backend~~
* ~~Dockerfile~~
* Make test harness to test end to end
* ~~More documentation~~
* PostgreSQL and possibly Azure Blob storage/MySQL backends
* Add Helm Chart / Kustomize manifests
* Cache expiry/cleanup
* Benchmark cpu and memory usage especially on PATCH

# How it works:

The architecture and documentation about the GitHub Cache API is [here](ARCHITECTURE.md)