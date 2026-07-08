package api

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/al-hudhud/backend/internal/db"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

// PackageHandler handles package management APIs
type PackageHandler struct {
	pool   *pgxpool.Pool
	logger *zap.Logger
}

// NewPackageHandler creates a new package handler
func NewPackageHandler(pool *pgxpool.Pool, logger *zap.Logger) *PackageHandler {
	return &PackageHandler{
		pool:   pool,
		logger: logger,
	}
}

// CreatePackageRequest represents a request to create a new package
type CreatePackageRequest struct {
	ServiceID            int64           `json:"service_id" binding:"required"`
	Name                 string          `json:"name" binding:"required"`
	Code                 string          `json:"code" binding:"required"`
	PriceUSD             decimal.Decimal `json:"price_usd" binding:"required"`
	IncludedMessages     *int64          `json:"included_messages"` // NULL for unlimited
	OverageRate          decimal.Decimal `json:"overage_rate"`
	IsUnlimited          bool            `json:"is_unlimited"`
	BillingCycle         string          `json:"billing_cycle" binding:"required,oneof=monthly yearly"`
	RolloverEnabled      bool            `json:"rollover_enabled"`
	RolloverLimit        *int64          `json:"rollover_limit"`
	RolloverExpiryMonths *int            `json:"rollover_expiry_months"`
	DisplayOrder         int             `json:"display_order"`
}

// CreatePackageResponse represents the response for package creation
type CreatePackageResponse struct {
	Success   bool   `json:"success"`
	PackageID int64  `json:"package_id,omitempty"`
	Message   string `json:"message"`
}

// CreatePackage creates a new package
// Implements Requirements 3.6, 18.6
// - Admin only access
// - Tenant validation
// - Creates package with all settings
// - Returns package ID on success
//
// POST /api/v1/packages
func (h *PackageHandler) CreatePackage(c *gin.Context) {
	ctx := c.Request.Context()

	// Extract tenant_id from context (set by middleware)
	tenantIDStr, exists := c.Get("tenant_id")
	if !exists {
		h.logger.Error("tenant_id not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "Unauthorized: tenant_id not found",
		})
		return
	}

	tenantID, err := uuid.Parse(tenantIDStr.(string))
	if err != nil {
		h.logger.Error("Invalid tenant_id format", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid tenant_id format",
		})
		return
	}

	// Check if user is admin (Requirement 3.6, 18.6)
	userRole, exists := c.Get("user_role")
	if !exists || userRole != "admin" {
		h.logger.Warn("Non-admin user attempted to create package",
			zap.String("tenant_id", tenantID.String()),
			zap.String("user_role", fmt.Sprintf("%v", userRole)),
		)
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   "Forbidden: Admin access required",
		})
		return
	}

	// Parse request body
	var req CreatePackageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Invalid request body", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   fmt.Sprintf("Invalid request: %v", err),
		})
		return
	}

	// Validate business rules
	if err := h.validatePackageRequest(&req); err != nil {
		h.logger.Error("Package validation failed", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// Create package
	packageID, err := h.createPackage(ctx, tenantID, &req)
	if err != nil {
		h.logger.Error("Failed to create package",
			zap.String("tenant_id", tenantID.String()),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to create package",
		})
		return
	}

	h.logger.Info("Package created successfully",
		zap.String("tenant_id", tenantID.String()),
		zap.Int64("package_id", packageID),
		zap.String("package_name", req.Name),
	)

	c.JSON(http.StatusCreated, CreatePackageResponse{
		Success:   true,
		PackageID: packageID,
		Message:   fmt.Sprintf("Package '%s' created successfully", req.Name),
	})
}

// validatePackageRequest validates the package creation request
func (h *PackageHandler) validatePackageRequest(req *CreatePackageRequest) error {
	// Validate price
	if req.PriceUSD.LessThanOrEqual(decimal.Zero) {
		return fmt.Errorf("price_usd must be greater than 0")
	}

	// Validate unlimited vs limited package
	if req.IsUnlimited {
		// Unlimited packages should have NULL included_messages
		if req.IncludedMessages != nil {
			return fmt.Errorf("unlimited packages should not have included_messages")
		}
	} else {
		// Limited packages must have included_messages
		if req.IncludedMessages == nil || *req.IncludedMessages <= 0 {
			return fmt.Errorf("limited packages must have included_messages > 0")
		}
	}

	// Validate overage rate for limited packages
	if !req.IsUnlimited && req.OverageRate.LessThanOrEqual(decimal.Zero) {
		return fmt.Errorf("overage_rate must be greater than 0 for limited packages")
	}

	// Validate rollover settings
	if req.RolloverEnabled {
		if req.RolloverLimit != nil && *req.RolloverLimit <= 0 {
			return fmt.Errorf("rollover_limit must be greater than 0 if specified")
		}
		if req.RolloverExpiryMonths != nil && *req.RolloverExpiryMonths <= 0 {
			return fmt.Errorf("rollover_expiry_months must be greater than 0 if specified")
		}
	}

	// Validate billing cycle
	if req.BillingCycle != "monthly" && req.BillingCycle != "yearly" {
		return fmt.Errorf("billing_cycle must be 'monthly' or 'yearly'")
	}

	return nil
}

// createPackage creates a package in the database
func (h *PackageHandler) createPackage(ctx context.Context, tenantID uuid.UUID, req *CreatePackageRequest) (int64, error) {
	// Start transaction
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Set tenant and admin context for RLS
	_, err = tx.Exec(ctx, "SET LOCAL app.current_tenant = $1", tenantID.String())
	if err != nil {
		return 0, fmt.Errorf("failed to set tenant context: %w", err)
	}

	isSuperAdmin := false
	if val, ok := ctx.Value("user_type").(string); ok {
		isSuperAdmin = val == "SUPER_ADMIN"
	}
	adminVal := "false"
	if isSuperAdmin {
		adminVal = "true"
	}
	_, err = tx.Exec(ctx, "SET LOCAL app.is_super_admin = $1", adminVal)
	if err != nil {
		return 0, fmt.Errorf("failed to set admin context: %w", err)
	}

	// Check if package code already exists for this tenant
	var existingID int64
	checkQuery := `
		SELECT id FROM packages 
		WHERE tenant_id = $1 AND code = $2
	`
	err = tx.QueryRow(ctx, checkQuery, tenantID, req.Code).Scan(&existingID)
	if err == nil {
		return 0, fmt.Errorf("package with code '%s' already exists", req.Code)
	}

	// Insert package
	insertQuery := `
		INSERT INTO packages (
			tenant_id,
			service_id,
			name,
			code,
			price_usd,
			included_messages,
			overage_rate,
			is_unlimited,
			billing_cycle,
			rollover_enabled,
			rollover_limit,
			rollover_expiry_months,
			is_active,
			display_order,
			created_at,
			updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, true, $13, NOW(), NOW()
		)
		RETURNING id
	`

	var packageID int64
	err = tx.QueryRow(ctx, insertQuery,
		tenantID,
		req.ServiceID,
		req.Name,
		req.Code,
		req.PriceUSD,
		req.IncludedMessages,
		req.OverageRate,
		req.IsUnlimited,
		req.BillingCycle,
		req.RolloverEnabled,
		req.RolloverLimit,
		req.RolloverExpiryMonths,
		req.DisplayOrder,
	).Scan(&packageID)
	if err != nil {
		return 0, fmt.Errorf("failed to insert package: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return packageID, nil
}

// ListPackagesRequest represents query parameters for listing packages
type ListPackagesRequest struct {
	ServiceID *int64 `form:"service_id"`
	IsActive  *bool  `form:"is_active"`
	Page      int    `form:"page" binding:"min=1"`
	PageSize  int    `form:"page_size" binding:"min=1,max=100"`
}

// PackageResponse represents a package in API responses
type PackageResponse struct {
	ID                   int64           `json:"id"`
	ServiceID            int64           `json:"service_id"`
	Name                 string          `json:"name"`
	Code                 string          `json:"code"`
	PriceUSD             decimal.Decimal `json:"price_usd"`
	IncludedMessages     *int64          `json:"included_messages"`
	OverageRate          decimal.Decimal `json:"overage_rate"`
	IsUnlimited          bool            `json:"is_unlimited"`
	BillingCycle         string          `json:"billing_cycle"`
	RolloverEnabled      bool            `json:"rollover_enabled"`
	RolloverLimit        *int64          `json:"rollover_limit"`
	RolloverExpiryMonths *int            `json:"rollover_expiry_months"`
	IsActive             bool            `json:"is_active"`
	DisplayOrder         int             `json:"display_order"`
	CreatedAt            string          `json:"created_at"`
	UpdatedAt            string          `json:"updated_at"`
}

// ListPackagesResponse represents the response for listing packages
type ListPackagesResponse struct {
	Success    bool              `json:"success"`
	Packages   []PackageResponse `json:"packages"`
	TotalCount int               `json:"total_count"`
	Page       int               `json:"page"`
	PageSize   int               `json:"page_size"`
}

// ListPackages lists all packages with filtering and pagination
// Implements Requirements 18.5, 21.1
// - Tenant isolation
// - Filtering by service_id and is_active
// - Pagination support
//
// GET /api/v1/packages
func (h *PackageHandler) ListPackages(c *gin.Context) {
	ctx := c.Request.Context()

	// Extract tenant_id from context
	tenantIDStr, exists := c.Get("tenant_id")
	if !exists {
		h.logger.Error("tenant_id not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "Unauthorized: tenant_id not found",
		})
		return
	}

	tenantID, err := uuid.Parse(tenantIDStr.(string))
	if err != nil {
		h.logger.Error("Invalid tenant_id format", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid tenant_id format",
		})
		return
	}

	// Parse query parameters
	var req ListPackagesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		h.logger.Error("Invalid query parameters", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   fmt.Sprintf("Invalid query parameters: %v", err),
		})
		return
	}

	// Set defaults
	if req.Page == 0 {
		req.Page = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 20
	}

	// List packages
	isSuperAdmin := c.GetString("user_type") == "SUPER_ADMIN"
	packages, totalCount, err := h.listPackages(ctx, tenantID, isSuperAdmin, &req)
	if err != nil {
		h.logger.Error("Failed to list packages",
			zap.String("tenant_id", tenantID.String()),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to list packages",
		})
		return
	}

	c.JSON(http.StatusOK, ListPackagesResponse{
		Success:    true,
		Packages:   packages,
		TotalCount: totalCount,
		Page:       req.Page,
		PageSize:   req.PageSize,
	})
}

func (h *PackageHandler) listPackages(ctx context.Context, tenantID uuid.UUID, isSuperAdmin bool, req *ListPackagesRequest) ([]PackageResponse, int, error) {
	var totalCount int
	var packages []PackageResponse

	err := db.RunInTenantPgxTx(ctx, tenantID, isSuperAdmin, h.pool, func(tx pgx.Tx) error {
		// Build query with filters
		query := `
			SELECT 
				id, service_id, name, code, price_usd,
				included_messages, overage_rate, is_unlimited, billing_cycle,
				rollover_enabled, rollover_limit, rollover_expiry_months,
				is_active, display_order, created_at, updated_at
			FROM packages
			WHERE tenant_id = $1
		`
		countQuery := `SELECT COUNT(*) FROM packages WHERE tenant_id = $1`

		args := []interface{}{tenantID}
		argIndex := 2

		// Add filters
		if req.ServiceID != nil {
			query += fmt.Sprintf(" AND service_id = $%d", argIndex)
			countQuery += fmt.Sprintf(" AND service_id = $%d", argIndex)
			args = append(args, *req.ServiceID)
			argIndex++
		}

		if req.IsActive != nil {
			query += fmt.Sprintf(" AND is_active = $%d", argIndex)
			countQuery += fmt.Sprintf(" AND is_active = $%d", argIndex)
			args = append(args, *req.IsActive)
			argIndex++
		}

		// Get total count
		err := tx.QueryRow(ctx, countQuery, args...).Scan(&totalCount)
		if err != nil {
			return fmt.Errorf("failed to get total count: %w", err)
		}

		// Add ordering and pagination
		query += " ORDER BY display_order ASC, created_at DESC"
		offset := (req.Page - 1) * req.PageSize
		query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
		args = append(args, req.PageSize, offset)

		// Execute query
		rows, err := tx.Query(ctx, query, args...)
		if err != nil {
			return fmt.Errorf("failed to query packages: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var pkg PackageResponse
			err := rows.Scan(
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
			)
			if err != nil {
				return fmt.Errorf("failed to scan package: %w", err)
			}
			packages = append(packages, pkg)
		}

		if err := rows.Err(); err != nil {
			return fmt.Errorf("error iterating packages: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, 0, err
	}

	return packages, totalCount, nil
}

// GetPackage retrieves a single package by ID
// GET /api/v1/packages/:id
func (h *PackageHandler) GetPackage(c *gin.Context) {
	ctx := c.Request.Context()

	// Extract tenant_id from context
	tenantIDStr, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "Unauthorized: tenant_id not found",
		})
		return
	}

	tenantID, err := uuid.Parse(tenantIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid tenant_id format",
		})
		return
	}

	// Parse package ID
	packageIDStr := c.Param("id")
	packageID, err := strconv.ParseInt(packageIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid package ID",
		})
		return
	}

	// Get package
	isSuperAdmin := c.GetString("user_type") == "SUPER_ADMIN"
	pkg, err := h.getPackage(ctx, tenantID, isSuperAdmin, packageID)
	if err != nil {
		h.logger.Error("Failed to get package",
			zap.String("tenant_id", tenantID.String()),
			zap.Int64("package_id", packageID),
			zap.Error(err),
		)
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Package not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"package": pkg,
	})
}

// getPackage retrieves a package from the database
func (h *PackageHandler) getPackage(ctx context.Context, tenantID uuid.UUID, isSuperAdmin bool, packageID int64) (*PackageResponse, error) {
	var pkg PackageResponse

	err := db.RunInTenantPgxTx(ctx, tenantID, isSuperAdmin, h.pool, func(tx pgx.Tx) error {
		query := `
			SELECT 
				id, service_id, name, code, price_usd,
				included_messages, overage_rate, is_unlimited, billing_cycle,
				rollover_enabled, rollover_limit, rollover_expiry_months,
				is_active, display_order, created_at, updated_at
			FROM packages
			WHERE tenant_id = $1 AND id = $2
		`

		return tx.QueryRow(ctx, query, tenantID, packageID).Scan(
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
		)
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get package: %w", err)
	}

	return &pkg, nil
}
