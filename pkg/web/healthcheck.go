package web

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func HealthCheckEndpoint(c *gin.Context) {
	c.Data(http.StatusNoContent, gin.MIMEJSON, nil)
}

func PingEndpoint(c *gin.Context) {
	c.String(http.StatusOK, "pong")
}
