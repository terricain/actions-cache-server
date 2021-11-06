package web

import "github.com/gin-gonic/gin"

func XForwardedProto(defaultScheme string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if hdr := c.GetHeader("X-Forwarded-Proto"); hdr != "" {
			c.Request.URL.Scheme = hdr
		} else {
			c.Request.URL.Scheme = defaultScheme
		}

		c.Next()
	}
}
