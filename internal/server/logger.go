package server

import (
	"time"

	"github.com/gin-gonic/gin"
	zlog "github.com/rs/zerolog/log"
)

func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		t := time.Now()
		path := c.Request.URL.Path

		c.Next()

		data := &requestData{
			status:    c.Writer.Status(),
			client_ip: c.ClientIP(),
			path:      path,
			method:    c.Request.Method,
			latency:   time.Since(t),
			err:       c.Errors.Last(),
		}

		data.log()
	}
}

type requestData struct {
	status    int
	client_ip string
	path      string
	method    string
	latency   time.Duration
	err       *gin.Error
}

func (r *requestData) log() {
	switch {
	case 500 <= r.status:
		zlog.Error().Str("ip", r.client_ip).Str("path", r.path).Int("status", r.status).Str("latency", r.latency.String()).Err(r.err).Msg("5xx")
	case 400 <= r.status:
		zlog.Warn().Str("ip", r.client_ip).Str("path", r.path).Int("status", r.status).Str("latency", r.latency.String()).Err(r.err).Msg("4xx")
	default:
		zlog.Info().Str("ip", r.client_ip).Str("path", r.path).Int("status", r.status).Str("latency", r.latency.String()).Msg("Success")
	}
}
