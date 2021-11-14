// Taken from https://github.com/zsais/go-gin-prometheus/blob/master/middleware.go, didn't need all the bells
// and whistles, all props goes to @zsais

package metrics

import (
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var reqCount = prometheus.NewCounterVec(prometheus.CounterOpts{
	Name:        "http_requests_total",
	Help:        "How many HTTP requests processed, partitioned by status code and HTTP method",
}, []string{"code", "method", "handler", "host", "url"})

var reqDur = prometheus.NewHistogramVec(prometheus.HistogramOpts{
	Name:        "http_request_duration_seconds",
	Help:        "The HTTP request latencies in seconds",
	//	Labels:        ,
	// Buckets:     nil,
}, []string{"code", "method", "url"})

var respSize = prometheus.NewSummary(prometheus.SummaryOpts{
	Name:        "http_response_size_bytes",
	Help:        "The HTTP response sizes in bytes",
})

var reqSize = prometheus.NewSummary(prometheus.SummaryOpts{
	Name:        "http_request_size_bytes",
	Help:        "The HTTP request sizes in bytes",
})

func requestPathMapper(c *gin.Context) string {
	url := c.Request.URL.Path
	for _, p := range c.Params {
		url = strings.Replace(url, p.Value, p.Key, 1)
	}
	return url
}

func PromReqMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		reqSz := float64(computeApproximateRequestSize(c.Request))

		c.Next()

		status := strconv.Itoa(c.Writer.Status())
		elapsed := float64(time.Since(start)) / float64(time.Second)
		resSz := float64(c.Writer.Size())

		url := requestPathMapper(c)
		reqDur.WithLabelValues(status, c.Request.Method, url).Observe(elapsed)
		reqCount.WithLabelValues(status, c.Request.Method, c.HandlerName(), c.Request.Host, url).Inc()
		reqSize.Observe(reqSz)
		respSize.Observe(resSz)
	}
}

func computeApproximateRequestSize(r *http.Request) int {
	s := 0
	if r.URL != nil {
		s = len(r.URL.Path)
	}

	s += len(r.Method)
	s += len(r.Proto)
	for name, values := range r.Header {
		s += len(name)
		for _, value := range values {
			s += len(value)
		}
	}
	s += len(r.Host)

	// N.B. r.Form and r.MultipartForm are assumed to be included in r.URL.

	if r.ContentLength != -1 {
		s += int(r.ContentLength)
	}
	return s
}
