package web

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

func HealthCheckEndpoint(c *gin.Context) {
	c.Data(http.StatusNoContent, gin.MIMEJSON, nil)
}
