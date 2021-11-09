package web

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

func GinLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		t := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		c.Next()

		if path == "/healthz" || path == "/metrics" {
			return
		}

		latency := time.Since(t)
		clientIP := c.ClientIP()
		if raw != "" {
			path = path + "?" + raw
		}
		msg := c.Errors.String()
		if msg == "" {
			msg = "Request"
		}

		statusCode := c.Writer.Status()
		ua := c.Request.Header.Get("User-Agent")
		switch {
		case statusCode >= 400 && statusCode < 500:
			{
				log.Warn().Str("logger", "access").Str("method", c.Request.Method).
					Str("path", path).Dur("resp_time", latency).Int("status", statusCode).
					Str("client_ip", clientIP).Str("user_agent", ua).Msg(msg)
			}
		case statusCode >= 500:
			{
				log.Error().Str("logger", "access").Str("method", c.Request.Method).
					Str("path", path).Dur("resp_time", latency).Int("status", statusCode).
					Str("client_ip", clientIP).Str("user_agent", ua).Msg(msg)
			}
		default:
			log.Info().Str("logger", "access").Str("method", c.Request.Method).
				Str("path", path).Dur("resp_time", latency).Int("status", statusCode).
				Str("client_ip", clientIP).Str("user_agent", ua).Msg(msg)
		}
	}
}
