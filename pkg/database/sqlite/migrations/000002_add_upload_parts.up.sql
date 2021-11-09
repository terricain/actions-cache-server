CREATE TABLE cache_parts (
    "repository" TEXT,
    "cache_id" INTEGER,
    "part" INTEGER,

    "start_byte" INTEGER,
    "end_byte" INTEGER,
    "size" UNSIGNED BIG INT,
    "part_data" TEXT,

    PRIMARY KEY ("repository", "cache_id", "part"),
    CONSTRAINT fk_cacheid FOREIGN KEY ("repository", "cache_id") REFERENCES cache("repository", "cache_id") ON DELETE CASCADE
);
-- By not inserting "part" according to documentation it'll use rowid+1