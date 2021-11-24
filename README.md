![Tests CI](https://github.com/terrycain/actions-cache-server/actions/workflows/test.yml/badge.svg)
![Release CI](https://github.com/terrycain/actions-cache-server/actions/workflows/release.yml/badge.svg)
![Scan CI](https://github.com/terrycain/actions-cache-server/actions/workflows/cve-scan.yml/badge.svg)
# actions-cache-server

A GitHub Actions Caching server for self-hosted runners. By using [terrycain/cache@custom-url](https://github.com/terrycain/cache/tree/custom-url) you can now
upload job caches to a local server.

This repo is still in an alpha like state, thing will most likely change, especially the Helm charts and Kustomize manifests.

- [Why does this exist](#why-does-this-exist)
- [How to run](#how-to-run)
  - [Cache action setup](#cache-action-setup)
  - [Server setup](#server-setup)
- [Supported backends](#supported-backends)
- [What doesn't work](#what-doesnt-work)
- [Roadmap](#roadmap)

## Why does this exist
I have a variety of self-hosted runners to run actions for some private repos as I'd rather not pay for build minutes. I decided to look into
building some Go projects with [Bazel](https://bazel.build/), all was going well until I tried to speed up build times with [actions/cache](https://github.com/actions/cache).
It turns out Bazel's build cache is rather large, and where I'm building these actions does not have the best upload speed so I decended the rabbit hole
getting self-hosted runners to cache using an alternate caching sever. See [here](ARCHITECTURE.md) for a more in-depth view on how GitHub actions caching works.

## How to run

### Cache action setup

To start with you will need to update your `actions/cache` step and use my fork until I can PR the changes upstream. Below is an example 
of a cache action using an external server ([github repo here](https://github.com/terrycain/cache/tree/custom-url)):

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

For now, there should not be any additional path on the server's URL. The forked action currently takes the path of the original actions server
and appends it to the url as this path from the original url contains some repository identifier. There is no fundamental reason why the caching
server could not run on a subpath but that's just not done yet (feel free to PR).

### Server setup

The server consists of 2 major parts, a part that deals with cache metadata, and a part that deals with cache storage. I have tried to design the
server with some degree of modularity in mind so that it can fit in whatever environment so eventually there will be various database and storage
backends to choose from. There will be some tradeoffs depending on what combination you choose, e.g. if you choose SQLite for the database, you'll
probably not want to read/write to it from multiple processes (albeit the docs seems to claim this works now :/).

Docker image: `ghcr.io/terrycain/actions-cache-server:0.1.3` ([image repo](https://github.com/terrycain/actions-cache-server/pkgs/container/actions-cache-server) if I forget to update the image tag)

Running --help on the container will list all arguments it takes, which all can be defined as environment variables. You will need to specify a `--db-something` and `--storage-somthing` argument. 
See [BACKENDS.md](BACKENDS.md) for a more detailed description of each backend and the format of the args.

Example deployment using Docker which will use SQLite and basic disk storage.
```shell
mkdir db cache

docker run --rm -it -p8080:8080 -v $(pwd)/db:/tmp/db -v $(pwd)/cache:/tmp/cache docker pull ghcr.io/terrycain/actions-cache-server:0.1.3 \
  --db-sqlite /tmp/db/db.sqlite \
  --storage-disk /tmp/cache \
  --listen-address 0.0.0.0:8080
```

## Supported backends

| Type     | Name                 | Supported                                             |
|----------|----------------------|-------------------------------------------------------|
| Database | SQLite               | :heavy_check_mark:                                    |
| Database | Postgres             | Planned                                               |
| Database | MySQL                | :x: Will do if there is demand                        |
| Database | DynamoDB             | :x: Will do if there is demand                        |
| Database | MongoDB              | :x: Will do if there is demand                        |
| Database | CosmosDB             | :x: Will do if there is demand                        |
| Storage  | Disk                 | :heavy_check_mark:                                    |
| Storage  | AWS S3               | :heavy_check_mark:                                    |
| Storage  | Azure Blob Storage   | :x: Planned                                           |
| Storage  | Google Cloud Storage | :x: Will do if there is demand                        |

## What doesn't work

GitHub Actions seem to indicate that caching is partially shared to forks, this is not supported and currently have no
idea how to make it work nor the intention to. If someone really needs this, then raise an issue and we can look into it.

## Roadmap 

Roadmap sounds better than a glorified todo list :smile:

* Plan out and implement end-to-end tests 
* Postgres backend 
* Azure Blob Storage backend 
* Cache space usage / management 
* Benchmark cpu and memory usage especially on large PATCH's

## Contributing

Feel free to raise Pull Requests. For any major changes please raise an issue so that they can be discussed and avoid any
duplication of work etc...

Obviously update/add tests where appropriate. Some info around testing is [here](TESTING.md)

## License
[Apache 2.0](https://choosealicense.com/licenses/apache-2.0/)