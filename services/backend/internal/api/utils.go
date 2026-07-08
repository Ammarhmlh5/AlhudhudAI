package api

import (
	"time"

	"github.com/gin-gonic/gin"
)

// RespondWithJSON sends a JSON response using Gin
func RespondWithJSON(c *gin.Context, code int, payload interface{}) {
	c.JSON(code, payload)
}

// RespondWithError sends a structured error response using Gin
func RespondWithError(c *gin.Context, code int, errorCode, message string, details map[string]interface{}) {
	payload := gin.H{
		"success": false,
		"error": gin.H{
			"code":      errorCode,
			"message":   message,
			"timestamp": time.Now().Format(time.RFC3339),
		},
	}

	if details != nil {
		payload["error"].(gin.H)["details"] = details
	}

	c.JSON(code, payload)
}

// RespondWithSuccess sends a structured success response using Gin
func RespondWithSuccess(c *gin.Context, code int, data interface{}, message string) {
	payload := gin.H{
		"success": true,
		"data":    data,
	}
	if message != "" {
		payload["message"] = message
	}
	c.JSON(code, payload)
}
