package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/al-hudhud/backend/internal/db"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// AdminPackageHandler handles global package management (SUPER_ADMIN).
// All operations run through tenant-scoped transactions (db.RunInTenantPgxTx)
// with the super-admin RLS bypass enabled, instead of using GORM directly
// against the RLS-protected packages table.
type AdminPackageHandler struct {
	pool           *pgxpool.Pool
	logger         *zap.Logger
	packageHandler *PackageHandler
}

// NewAdminPackageHandler creates a new admin package handler
func NewAdminPackageHandler(pool *pgxpool.Pool, logger *zap.Logger, packageHandler *PackageHandler) *AdminPackageHandler {
	return &AdminPackageHandler{
		pool:           pool,
		logger:         logger,
		packageHandler: packageHandler,
	}
}

// AdminPackageResponse extends PackageResponse with the owning tenant.
type AdminPackageResponse struct {
	PackageResponse
	TenantID uuid.UUID `json:"tenant_id"`
}

// AdminCreatePackageRequest adds the target tenant to the standard create request.
// If TenantID is omitted, the authenticated SUPER_ADMIN's own tenant is used.
type AdminCreatePackageRequest struct {
	TenantID uuid.UUID `json:"tenant_id"`
	CreatePackageRequest
}

// ListPackages handles GET /api/v1/admin/packages
// Optional filters: tenant_id, service_id, is_active
func (h *AdminPackageHandler) ListPackages(c *gin.Context) {
	ctx := c.Request.Context()

	var tenantID uuid.UUID
	hasTenantFilter := false
	if tenantIDStr := c.Query("tenant_id"); tenantIDStr != "" {
		parsed, err := uuid.Parse(tenantIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tenant_id format"})
			return
		}
		tenantID = parsed
		hasTenantFilter = true
	}

	var packages []AdminPackageResponse
	// RLS context: the super-admin bypass (is_super_admin=true) makes the
	// current_tenant value irrelevant for SELECT visibility.
	err := db.RunInTenantPgxTx(ctx, tenantID, true, h.pool, func(tx pgx.Tx) error {
		query := `
			SELECT tenant_id, id, service_id, name, code, price_usd,
			       included_messages, overage_rate, is_unlimited, billing_cycle,
			       rollover_enabled, rollover_limit, rollover_expiry_months,
			       is_active, display_order, created_at, updated_at
			FROM packages
		`
		var args []interface{}
		argIndex := 1
		var conditions []string

		if hasTenantFilter {
			conditions = append(conditions, fmt.Sprintf("tenant_id = $%d", argIndex))
			args = append(args, tenantID)
			argIndex++
		}
		if serviceIDStr := c.Query("service_id"); serviceIDStr != "" {
			serviceID, err := strconv.ParseInt(serviceIDStr, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid service_id: %w", err)
			}
			conditions = append(conditions, fmt.Sprintf("service_id = $%d", argIndex))
			args = append(args, serviceID)
			argIndex++
		}
		if isActiveStr := c.Query("is_active"); isActiveStr != "" {
			isActive, err := strconv.ParseBool(isActiveStr)
			if err != nil {
				return fmt.Errorf("invalid is_active: %w", err)
			}
			conditions = append(conditions, fmt.Sprintf("is_active = $%d", argIndex))
			args = append(args, isActive)
			argIndex++
		}

		if len(conditions) > 0 {
			query += " WHERE " + strings.Join(conditions, " AND ")
		}
		query += " ORDER BY display_order ASC, created_at DESC"

		rows, err := tx.Query(ctx, query, args...)
		if err != nil {
			return fmt.Errorf("failed to query packages: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var pkg AdminPackageResponse
			if err := rows.Scan(
				&pkg.TenantID,
				&pkg.ID,
				&pkg.ServiceID,
				&pkg.Name,
				&pkg.Code,
				&pkg.PriceUSD,
				&pkg.IncludedMessages,
				&pkg.OverageRate,
				&pkg.IsUnlimited,
				&pkg.BillingCycle,
				&pkg.RolloverEnabled,
				&pkg.RolloverLimit,
				&pkg.RolloverExpiryMonths,
				&pkg.IsActive,
				&pkg.DisplayOrder,
				&pkg.CreatedAt,
				&pkg.UpdatedAt,
			); err != nil {
				return fmt.Errorf("failed to scan package: %w", err)
			}
			packages = append(packages, pkg)
		}

		return rows.Err()
	})
	if err != nil {
		h.logger.Error("Failed to list packages", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list packages"})
		return
	}

	if packages == nil {
		packages = []AdminPackageResponse{}
	}
	c.JSON(http.StatusOK, packages)
}

// GetPackage handles GET /api/v1/admin/packages/:id
func (h *AdminPackageHandler) GetPackage(c *gin.Context) {
	ctx := c.Request.Context()

	packageID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid package ID"})
		return
	}

	tenantID, err := h.findPackageTenant(ctx, packageID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Package not found"})
		return
	}

	pkg, err := h.packageHandler.getPackage(ctx, tenantID, true, packageID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Package not found"})
		return
	}

	c.JSON(http.StatusOK, pkg)
}

// CreatePackage handles POST /api/v1/admin/packages
func (h *AdminPackageHandler) CreatePackage(c *gin.Context) {
	ctx := c.Request.Context()

	var req AdminCreatePackageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Resolve the target tenant: explicit tenant_id wins, otherwise fall back
	// to the authenticated admin's own tenant.
	targetTenant := req.TenantID
	if targetTenant == uuid.Nil {
		if tenantIDStr := c.GetString("tenant_id"); tenantIDStr != "" {
			parsed, err := uuid.Parse(tenantIDStr)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tenant_id format"})
				return
			}
			targetTenant = parsed
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id is required"})
			return
		}
	}

	if err := h.packageHandler.validatePackageRequest(&req.CreatePackageRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	packageID, err := h.packageHandler.createPackage(ctx, targetTenant, true, &req.CreatePackageRequest)
	if err != nil {
		h.logger.Error("Failed to create package",
			zap.String("tenant_id", targetTenant.String()),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create package"})
		return
	}

	pkg, err := h.packageHandler.getPackage(ctx, targetTenant, true, packageID)
	if err != nil {
		c.JSON(http.StatusCreated, gin.H{
			"success":    true,
			"package_id": packageID,
			"message":    "Package created successfully",
		})
		return
	}

	c.JSON(http.StatusCreated, pkg)
}

// UpdatePackage handles PUT /api/v1/admin/packages/:id
func (h *AdminPackageHandler) UpdatePackage(c *gin.Context) {
	ctx := c.Request.Context()

	packageID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid package ID"})
		return
	}

	var req UpdatePackageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenantID, err := h.findPackageTenant(ctx, packageID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Package not found"})
		return
	}

	if err := h.packageHandler.updatePackage(ctx, tenantID, true, packageID, &req); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Package not found"})
			return
		}
		h.logger.Error("Failed to update package", zap.Int64("package_id", packageID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update package"})
		return
	}

	pkg, err := h.packageHandler.getPackage(ctx, tenantID, true, packageID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": true, "message": "Package updated successfully"})
		return
	}

	c.JSON(http.StatusOK, pkg)
}

// DeletePackage handles DELETE /api/v1/admin/packages/:id
func (h *AdminPackageHandler) DeletePackage(c *gin.Context) {
	ctx := c.Request.Context()

	packageID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid package ID"})
		return
	}

	tenantID, err := h.findPackageTenant(ctx, packageID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Package not found"})
		return
	}

	if err := h.packageHandler.deletePackage(ctx, tenantID, true, packageID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Package not found"})
			return
		}
		h.logger.Error("Failed to delete package", zap.Int64("package_id", packageID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete package"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Package deleted successfully"})
}

// findPackageTenant resolves the owning tenant of a package across all tenants.
func (h *AdminPackageHandler) findPackageTenant(ctx context.Context, packageID int64) (uuid.UUID, error) {
	var tenantID uuid.UUID
	err := db.RunInTenantPgxTx(ctx, uuid.Nil, true, h.pool, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, "SELECT tenant_id FROM packages WHERE id = $1", packageID).Scan(&tenantID)
	})
	return tenantID, err
}
