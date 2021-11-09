CREATE TABLE cache_parts (
    "cache_id" INTEGER,
    "part" INTEGER,

    "start_byte" INTEGER,
    "end_byte" INTEGER,
    "size" INTEGER,

   PRIMARY KEY ("cache_id", "part"),
   FOREIGN KEY ("cache_id") REFERENCES cache("cache_id")
);
-- By not inserting "part" according to documentation it'll use rowid+1