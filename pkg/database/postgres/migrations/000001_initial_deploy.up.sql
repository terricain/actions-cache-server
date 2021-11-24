BEGIN TRANSACTION;

CREATE SEQUENCE seq_cache
    INCREMENT BY 1
    MINVALUE 0
    MAXVALUE 2147483647
    START WITH 1
    CACHE 1
    NO CYCLE;

CREATE SEQUENCE seq_cache_part
    INCREMENT BY 1
    MINVALUE 0
    MAXVALUE 2147483647
    START WITH 1
    CACHE 1
    NO CYCLE;

CREATE TABLE cache (
    "repository" TEXT,
    "scope" TEXT,
    "key" TEXT,
    "version" TEXT,
    "cache_id" INTEGER NOT NULL DEFAULT nextval('seq_cache'::regclass),
    "created_date" TIMESTAMP,
    "finished" BOOLEAN DEFAULT FALSE,
    "size" BIGINT DEFAULT 0,
    "storage_backend" TEXT,
    "storage_path" TEXT,

    PRIMARY KEY ("repository", "scope", "key", "version"),
    UNIQUE ("repository", "cache_id")
);

CREATE INDEX idx_cache_cache_id ON cache (repository, cache_id);

CREATE TABLE cache_part (
    "repository" TEXT,
    "cache_id" INTEGER,
    "part" INTEGER NOT NULL DEFAULT nextval('seq_cache_part'::regclass),

    "start_byte" INTEGER,
    "end_byte" INTEGER,
    "size" BIGINT,
    "part_data" TEXT,

    PRIMARY KEY ("repository", "cache_id", "part"),
    CONSTRAINT fk_cacheid FOREIGN KEY ("repository", "cache_id") REFERENCES cache("repository", "cache_id") ON DELETE CASCADE
);

COMMIT;