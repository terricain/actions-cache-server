package main

import (
	"github.com/alecthomas/kong"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"github.com/terrycain/actions-cache-server/pkg/database"
	"github.com/terrycain/actions-cache-server/pkg/metrics"
	"github.com/terrycain/actions-cache-server/pkg/storage"
	"github.com/terrycain/actions-cache-server/pkg/utils/logging"
	"github.com/terrycain/actions-cache-server/pkg/web"
)

var cli struct {
	// Database backends
	DBSqlite   string `env:"DB_SQLITE" required:"" xor:"db" help:"SQLite filepath e.g. /tmp/db.sqlite"`
	DBPostgres string `env:"DB_POSTGRES" required:"" xor:"db" help:"Postgres URI e.g. postgresql://blah"`

	// Storage backends
	StorageDisk string `env:"STORAGE_DISK" required:"" xor:"storage" help:"Use disk storage for cache data e.g. /tmp/cache"`
	StorageS3   string `env:"STORAGE_S3" required:"" xor:"storage" name:"storage-s3" help:"Use S3 storage for cache data e.g. s3://bucket"`

	// Misc
	LogLevel             string `env:"LOG_LEVEL" default:"info" enum:"debug,info,warn,error"`
	ListenAddress        string `env:"LISTEN_ADDR" default:"0.0.0.0:8080" help:"Listen address e.g. 0.0.0.0:8080"`
	MetricsListenAddress string `env:"METRICS_LISTEN_ADDR" default:"0.0.0.0:9102" help:"Listen address for prometheus metrics e.g. 0.0.0.0:9102"`
	Debug                bool   `env:"DEBUG" help:"Enable debug mode"`
}

func main() {
	kong.Parse(&cli)

	logging.SetupLogging(cli.LogLevel)

	var databaseBackendName, dbConnectionString string
	if cli.DBSqlite != "" {
		databaseBackendName = "sqlite"
		dbConnectionString = cli.DBSqlite
	}

	var storageBackendName, storageConnectionString string
	if cli.StorageDisk != "" {
		storageBackendName = "disk"
		storageConnectionString = cli.StorageDisk
	}
	if cli.StorageS3 != "" {
		storageBackendName = "s3"
		storageConnectionString = cli.StorageS3
	}

	dbBackend, err := database.GetBackend(databaseBackendName, dbConnectionString)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initiate database backend")
	}

	storageBackend, err := storage.GetStorageBackend(storageBackendName, storageConnectionString)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initiate storage backend")
	}

	handlers := web.Handlers{
		Database: dbBackend,
		Storage:  storageBackend,
		Debug:    cli.Debug,
	}

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery(), web.GinLogger(), metrics.PromReqMiddleware())
	router.Use(web.XForwardedProto("http"))

	go metrics.Server(cli.MetricsListenAddress)

	router.GET("/healthz", web.HealthCheckEndpoint)
	router.GET("/ping", web.PingEndpoint)

	authedGroup := router.Group("/:repo")
	authedGroup.Use(handlers.AuthRequired())
	authedGroup.GET("/_apis/artifactcache/cache", handlers.SearchCache)
	authedGroup.POST("/_apis/artifactcache/caches", handlers.StartCache)
	authedGroup.PATCH("/_apis/artifactcache/caches/:cacheid", handlers.UploadCache)
	authedGroup.POST("/_apis/artifactcache/caches/:cacheid", handlers.FinishCache)

	// Not authed
	router.GET("/archive/:key", handlers.ArchivePath)

	log.Info().Msgf("Listening on %s", cli.ListenAddress)
	if err = router.Run(cli.ListenAddress); err != nil {
		log.Fatal().Err(err).Msg("Failed HTTP server loop")
	}
}
