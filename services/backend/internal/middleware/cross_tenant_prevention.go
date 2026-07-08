package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CrossTenantAccessLog represents a log entry for cross-tenant access attempts
type CrossTenantAccessLog struct {
	ID               uint      `gorm:"primaryKey"`
	TenantID         uuid.UUID `gorm:"type:uuid;not null;index"`
	RequestedTenantID uuid.UUID `gorm:"type:uuid;not null"`
	UserID           uuid.UUID `gorm:"type:uuid"`
	UserType         string    `gorm:"type:varchar(50)"`
	IPAddress        string    `gorm:"type:varchar(45)"`
	RequestPath      string    `gorm:"type:text"`
	RequestMethod    string    `gorm:"type:varchar(10)"`
	Blocked          bool      `gorm:"default:true"`
	CreatedAt        time.Time `gorm:"autoCreateTime"`
}

// PreventCrossTenantAccess middleware prevents tenants from accessing other tenants' resources
func PreventCrossTenantAccess(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userType := c.GetString("user_type")
		
		// Super admins and staff can access all tenants
		if userType == "SUPER_ADMIN" || userType == "STAFF" {
			c.Next()
			return
		}

		// Get the authenticated tenant ID
		tenantIDStr := c.GetString("tenant_id")
		if tenantIDStr == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "TENANT_ID_MISSING",
					"message": "Tenant ID not found in authentication context",
				},
			})
			c.Abort()
			return
		}

		tenantID, err := uuid.Parse(tenantIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "INVALID_TENANT_ID",
					"message": "Invalid tenant ID format",
				},
			})
			c.Abort()
			return
		}

		// Check URL parameters for tenant_id
		if paramTenantID := c.Param("tenant_id"); paramTenantID != "" {
			if err := validateTenantAccess(c, db, tenantID, paramTenantID); err != nil {
				return
			}
		}

		// Check query parameters for tenant_id
		if queryTenantID := c.Query("tenant_id"); queryTenantID != "" {
			if err := validateTenantAccess(c, db, tenantID, queryTenantID); err != nil {
				return
			}
		}

		// Check request body for tenant_id (if JSON)
		if c.Request.Method == "POST" || c.Request.Method == "PUT" || c.Request.Method == "PATCH" {
			if c.Request.Body != nil {
				importBytes, _ := io.ReadAll(c.Request.Body)
				fmt.Printf("[DEBUG] PreventCrossTenantAccess read body len: %d\n", len(importBytes))
				if len(importBytes) > 0 {
					// Restore the body so subsequent handlers can read it
					c.Request.Body = io.NopCloser(bytes.NewBuffer(importBytes))
					
					var body map[string]interface{}
					if err := json.Unmarshal(importBytes, &body); err == nil {
						if bodyTenantID, exists := body["tenant_id"]; exists {
							if tenantIDStr, ok := bodyTenantID.(string); ok {
								if err := validateTenantAccess(c, db, tenantID, tenantIDStr); err != nil {
									return
								}
							}
						}
					}
				}
			}
		}

		c.Next()
	}
}

// validateTenantAccess validates that the requested tenant ID matches the authenticated tenant ID
func validateTenantAccess(c *gin.Context, db *gorm.DB, authenticatedTenantID uuid.UUID, requestedTenantIDStr string) error {
	requestedTenantID, err := uuid.Parse(requestedTenantIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INVALID_REQUESTED_TENANT_ID",
				"message": "Invalid tenant ID in request",
			},
		})
		c.Abort()
		return err
	}

	if authenticatedTenantID != requestedTenantID {
		// Log the cross-tenant access attempt
		logCrossTenantAttempt(c, db, authenticatedTenantID, requestedTenantID)

		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "CROSS_TENANT_ACCESS_DENIED",
				"message": "You do not have permission to access resources from another tenant",
				"details": gin.H{
					"your_tenant_id":      authenticatedTenantID.String(),
					"requested_tenant_id": requestedTenantID.String(),
				},
			},
		})
		c.Abort()
		return fmt.Errorf("cross-tenant access denied")
	}

	return nil
}

// logCrossTenantAttempt logs a cross-tenant access attempt to the database
func logCrossTenantAttempt(c *gin.Context, db *gorm.DB, tenantID, requestedTenantID uuid.UUID) {
	userIDStr := c.GetString("user_id")
	userID, _ := uuid.Parse(userIDStr)

	log := CrossTenantAccessLog{
		TenantID:          tenantID,
		RequestedTenantID: requestedTenantID,
		UserID:            userID,
		UserType:          c.GetString("user_type"),
		IPAddress:         c.ClientIP(),
		RequestPath:       c.Request.URL.Path,
		RequestMethod:     c.Request.Method,
		Blocked:           true,
	}

	// Log asynchronously to avoid blocking the request
	go func() {
		if err := db.Create(&log).Error; err != nil {
			// Log error but don't fail the request
			fmt.Printf("Failed to log cross-tenant access attempt: %v\n", err)
		}
	}()
}
