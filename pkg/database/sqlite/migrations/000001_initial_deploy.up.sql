CREATE TABLE cache (
    "repository" TEXT,
    "scope" TEXT,
    "key" TEXT,
    "version" TEXT,
    "cache_id" INTEGER,
    "created_date" DATETIME,
    "finished" BOOLEAN DEFAULT 0,
    "size" UNSIGNED BIG INT,
    "storage_backend" TEXT,
    "storage_path" TEXT,

    PRIMARY KEY ("repository", "scope", "key", "version")
);

CREATE INDEX idx_cache_cache_id ON cache (repository, cache_id);