package web

import (
	"github.com/gin-gonic/gin"
	"github.com/terrycain/actions-cache-server/pkg/metrics"
)

func GetRouter(metricsListenAddress string, webHandler Handlers, withMetrics bool) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery(), GinLogger())
	if withMetrics {
		router.Use(metrics.PromReqMiddleware())
		go metrics.Server(metricsListenAddress)
	}
	router.Use(XForwardedProto("http"))

	router.GET("/healthz", HealthCheckEndpoint)
	router.GET("/ping", PingEndpoint)

	authedGroup := router.Group("/:repo")
	authedGroup.Use(webHandler.AuthRequired())
	authedGroup.GET("/_apis/artifactcache/cache", webHandler.SearchCache)
	authedGroup.POST("/_apis/artifactcache/caches", webHandler.StartCache)
	authedGroup.PATCH("/_apis/artifactcache/caches/:cacheid", webHandler.UploadCache)
	authedGroup.POST("/_apis/artifactcache/caches/:cacheid", webHandler.FinishCache)

	// Not authed
	router.GET("/archive/:key", webHandler.ArchivePath)

	return router
}
