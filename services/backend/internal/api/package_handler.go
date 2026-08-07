package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

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
	// Packages are tenant-scoped, so TENANT and SUPER_ADMIN can create them
	userType := c.GetString("user_type")
	if userType != "TENANT" && userType != "SUPER_ADMIN" {
		h.logger.Warn("Non-admin user attempted to create package",
			zap.String("tenant_id", tenantID.String()),
			zap.String("user_type", userType),
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
	isSuperAdmin := userType == "SUPER_ADMIN"
	packageID, err := h.createPackage(ctx, tenantID, isSuperAdmin, &req)
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
func (h *PackageHandler) createPackage(ctx context.Context, tenantID uuid.UUID, isSuperAdmin bool, req *CreatePackageRequest) (int64, error) {
	// Start transaction
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Set tenant and admin context for RLS
	_, err = tx.Exec(ctx, "SELECT set_config('app.current_tenant', $1, true)", tenantID.String())
	if err != nil {
		return 0, fmt.Errorf("failed to set tenant context: %w", err)
	}

	adminVal := "false"
	if isSuperAdmin {
		adminVal = "true"
	}
	_, err = tx.Exec(ctx, "SELECT set_config('app.is_super_admin', $1, true)", adminVal)
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
	Page      int    `form:"page"`
	PageSize  int    `form:"page_size"`
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
	IsActive             bool      `json:"is_active"`
	DisplayOrder         int       `json:"display_order"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
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
	scope := resolveTenantScope(c)
	if !scope.Exists {
		h.logger.Error("tenant_id not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "Unauthorized: tenant_id not found",
		})
		return
	}

	if !scope.Valid {
		// Super admins have no tenant scope; return an empty catalog.
		if scope.IsSuperAdmin {
			c.JSON(http.StatusOK, ListPackagesResponse{
				Success:    true,
				Packages:   []PackageResponse{},
				TotalCount: 0,
				Page:       1,
				PageSize:   20,
			})
			return
		}
		h.logger.Error("Invalid tenant_id format")
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid tenant_id format",
		})
		return
	}
	tenantID := scope.TenantID

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
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 || req.PageSize > 100 {
		req.PageSize = 20
	}

	// List packages
	isSuperAdmin := scope.IsSuperAdmin
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
	scope := resolveTenantScope(c)
	if !scope.Exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "Unauthorized: tenant_id not found",
		})
		return
	}

	if !scope.Valid {
		if scope.IsSuperAdmin {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   "Package not found",
			})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid tenant_id format",
		})
		return
	}
	tenantID := scope.TenantID

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
	isSuperAdmin := scope.IsSuperAdmin
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

// UpdatePackageRequest represents a request to update a package
type UpdatePackageRequest struct {
	ServiceID            *int64           `json:"service_id"`
	Name                 *string          `json:"name"`
	Code                 *string          `json:"code"`
	PriceUSD             *decimal.Decimal `json:"price_usd"`
	IncludedMessages     *int64           `json:"included_messages"`
	OverageRate          *decimal.Decimal `json:"overage_rate"`
	IsUnlimited          *bool            `json:"is_unlimited"`
	BillingCycle         *string          `json:"billing_cycle"`
	RolloverEnabled      *bool            `json:"rollover_enabled"`
	RolloverLimit        *int64           `json:"rollover_limit"`
	RolloverExpiryMonths *int             `json:"rollover_expiry_months"`
	IsActive             *bool            `json:"is_active"`
	DisplayOrder         *int             `json:"display_order"`
}

// UpdatePackage updates an existing package
// Implements Requirements 3.6, 18.6
// - Admin only access
// - Tenant isolation
// - Partial updates (only provided fields)
//
// PUT /api/v1/packages/:id
func (h *PackageHandler) UpdatePackage(c *gin.Context) {
	ctx := c.Request.Context()

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

	packageID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid package ID",
		})
		return
	}

	userType := c.GetString("user_type")
	if userType != "TENANT" && userType != "SUPER_ADMIN" {
		h.logger.Warn("Non-admin user attempted to update package",
			zap.String("tenant_id", tenantID.String()),
			zap.String("user_type", userType),
		)
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   "Forbidden: Admin access required",
		})
		return
	}

	var req UpdatePackageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Invalid request body", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   fmt.Sprintf("Invalid request: %v", err),
		})
		return
	}

	isSuperAdmin := userType == "SUPER_ADMIN"
	if err := h.updatePackage(ctx, tenantID, isSuperAdmin, packageID, &req); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   "Package not found",
			})
			return
		}
		h.logger.Error("Failed to update package",
			zap.String("tenant_id", tenantID.String()),
			zap.Int64("package_id", packageID),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to update package",
		})
		return
	}

	h.logger.Info("Package updated successfully",
		zap.String("tenant_id", tenantID.String()),
		zap.Int64("package_id", packageID),
	)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Package updated successfully",
	})
}

// updatePackage updates package fields in the database (partial update)
func (h *PackageHandler) updatePackage(ctx context.Context, tenantID uuid.UUID, isSuperAdmin bool, packageID int64, req *UpdatePackageRequest) error {
	return db.RunInTenantPgxTx(ctx, tenantID, isSuperAdmin, h.pool, func(tx pgx.Tx) error {
		// Verify the package belongs to the tenant
		var existingID int64
		err := tx.QueryRow(ctx, "SELECT id FROM packages WHERE tenant_id = $1 AND id = $2", tenantID, packageID).Scan(&existingID)
		if err != nil {
			return err
		}

		args := []interface{}{}
		argIndex := 1
		sets := []string{}

		if req.ServiceID != nil {
			sets = append(sets, fmt.Sprintf("service_id = $%d", argIndex))
			args = append(args, *req.ServiceID)
			argIndex++
		}
		if req.Name != nil {
			sets = append(sets, fmt.Sprintf("name = $%d", argIndex))
			args = append(args, *req.Name)
			argIndex++
		}
		if req.Code != nil {
			sets = append(sets, fmt.Sprintf("code = $%d", argIndex))
			args = append(args, *req.Code)
			argIndex++
		}
		if req.PriceUSD != nil {
			sets = append(sets, fmt.Sprintf("price_usd = $%d", argIndex))
			args = append(args, *req.PriceUSD)
			argIndex++
		}
		if req.IncludedMessages != nil {
			sets = append(sets, fmt.Sprintf("included_messages = $%d", argIndex))
			args = append(args, *req.IncludedMessages)
			argIndex++
		}
		if req.OverageRate != nil {
			sets = append(sets, fmt.Sprintf("overage_rate = $%d", argIndex))
			args = append(args, *req.OverageRate)
			argIndex++
		}
		if req.IsUnlimited != nil {
			sets = append(sets, fmt.Sprintf("is_unlimited = $%d", argIndex))
			args = append(args, *req.IsUnlimited)
			argIndex++
		}
		if req.BillingCycle != nil {
			sets = append(sets, fmt.Sprintf("billing_cycle = $%d", argIndex))
			args = append(args, *req.BillingCycle)
			argIndex++
		}
		if req.RolloverEnabled != nil {
			sets = append(sets, fmt.Sprintf("rollover_enabled = $%d", argIndex))
			args = append(args, *req.RolloverEnabled)
			argIndex++
		}
		if req.RolloverLimit != nil {
			sets = append(sets, fmt.Sprintf("rollover_limit = $%d", argIndex))
			args = append(args, *req.RolloverLimit)
			argIndex++
		}
		if req.RolloverExpiryMonths != nil {
			sets = append(sets, fmt.Sprintf("rollover_expiry_months = $%d", argIndex))
			args = append(args, *req.RolloverExpiryMonths)
			argIndex++
		}
		if req.IsActive != nil {
			sets = append(sets, fmt.Sprintf("is_active = $%d", argIndex))
			args = append(args, *req.IsActive)
			argIndex++
		}
		if req.DisplayOrder != nil {
			sets = append(sets, fmt.Sprintf("display_order = $%d", argIndex))
			args = append(args, *req.DisplayOrder)
			argIndex++
		}

		if len(sets) == 0 {
			return fmt.Errorf("no fields to update")
		}

		sets = append(sets, fmt.Sprintf("updated_at = $%d", argIndex))
		args = append(args, time.Now().UTC())
		argIndex++

		query := fmt.Sprintf("UPDATE packages SET %s WHERE tenant_id = $%d AND id = $%d",
			strings.Join(sets, ", "), argIndex, argIndex+1)
		args = append(args, tenantID, packageID)

		tag, err := tx.Exec(ctx, query, args...)
		if err != nil {
			return fmt.Errorf("failed to update package: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return pgx.ErrNoRows
		}
		return nil
	})
}

// deletePackage deletes a package within a tenant-scoped transaction
func (h *PackageHandler) deletePackage(ctx context.Context, tenantID uuid.UUID, isSuperAdmin bool, packageID int64) error {
	return db.RunInTenantPgxTx(ctx, tenantID, isSuperAdmin, h.pool, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, "DELETE FROM packages WHERE tenant_id = $1 AND id = $2", tenantID, packageID)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return pgx.ErrNoRows
		}
		return nil
	})
}

// DeletePackage deletes a package
// Implements Requirements 3.6, 18.6
// - Admin only access
// - Tenant isolation
//
// DELETE /api/v1/packages/:id
func (h *PackageHandler) DeletePackage(c *gin.Context) {
	ctx := c.Request.Context()

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

	packageID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid package ID",
		})
		return
	}

	userType := c.GetString("user_type")
	if userType != "TENANT" && userType != "SUPER_ADMIN" {
		h.logger.Warn("Non-admin user attempted to delete package",
			zap.String("tenant_id", tenantID.String()),
			zap.String("user_type", userType),
		)
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   "Forbidden: Admin access required",
		})
		return
	}

	isSuperAdmin := userType == "SUPER_ADMIN"
	err = db.RunInTenantPgxTx(ctx, tenantID, isSuperAdmin, h.pool, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, "DELETE FROM packages WHERE tenant_id = $1 AND id = $2", tenantID, packageID)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return pgx.ErrNoRows
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   "Package not found",
			})
			return
		}
		h.logger.Error("Failed to delete package",
			zap.String("tenant_id", tenantID.String()),
			zap.Int64("package_id", packageID),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to delete package",
		})
		return
	}

	h.logger.Info("Package deleted successfully",
		zap.String("tenant_id", tenantID.String()),
		zap.Int64("package_id", packageID),
	)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Package deleted successfully",
	})
}
