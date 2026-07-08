package monitoring

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// SetupMetricsEndpoint sets up the /metrics endpoint for Prometheus scraping
func SetupMetricsEndpoint(router *gin.Engine) {
	// Create a new handler for Prometheus metrics
	handler := promhttp.Handler()

	// Register the /metrics endpoint
	router.GET("/metrics", func(c *gin.Context) {
		handler.ServeHTTP(c.Writer, c.Request)
	})
}

// HealthCheckHandler provides a simple health check endpoint
func HealthCheckHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "healthy",
		"service": "admin-backend",
	})
}
