package agent

import (
	"golang.org/x/crypto/bcrypt"

	"github.com/al-hudhud/backend/internal/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ModuleInfo معلومات عن الوحدة
type ModuleInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Path    string `json:"path"`
}

var Module = ModuleInfo{
	Name:    "Agents System",
	Version: "1.0.0",
	Path:    "/api/v1/agents",
}

// RegisterRoutes - 注册路由（推荐使用）
func RegisterRoutes(router *gin.Engine, db *gorm.DB) {
	Initialize(router, db)
}

// AutoInitialize - 自动初始化（简单）
func AutoInitialize(router *gin.Engine, db *gorm.DB) error {
	Initialize(router, db)
	return nil
}

// InitializeWithMiddleware - 使用自定义中间件初始化
func InitializeWithMiddleware(router *gin.Engine, db *gorm.DB, middlewares ...gin.HandlerFunc) {
	service := NewService(db)
	handler := NewHandler(service)

	agentGroup := router.Group("/api/v1/agents")
	agentGroup.Use(middlewares...)

	setupRoutes(agentGroup, handler)
}

// RunMigrations - 运行数据库迁移
func RunMigrations(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.Agent{},
		&models.AgentEmployee{},
		&models.AgentTenant{},
		&models.AgentPayment{},
		&models.AgentSession{},
		&models.AgentActivityLog{},
	)
}

// CreateDefaultAgent - 创建默认代理（用于测试）
func CreateDefaultAgent(db *gorm.DB, companyName, email, password string) (*models.Agent, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	agent := &models.Agent{
		CompanyName:  companyName,
		Email:        email,
		PasswordHash: string(hashedPassword),
		IsActive:     true,
	}

	if err := db.Create(agent).Error; err != nil {
		return nil, err
	}

	return agent, nil
}

// Error codes for agent system
const (
	ErrCodeInvalidCredentials = "INVALID_CREDENTIALS"
	ErrCodeAgentNotFound      = "AGENT_NOT_FOUND"
	ErrCodeEmployeeNotFound   = "EMPLOYEE_NOT_FOUND"
	ErrCodeTenantNotFound     = "TENANT_NOT_FOUND"
	ErrCodeUnauthorized       = "UNAUTHORIZED"
	ErrCodeForbidden          = "FORBIDDEN"
	ErrCodeAlreadyExists      = "ALREADY_EXISTS"
	ErrCodeValidationError    = "VALIDATION_ERROR"
)

// IntegrationErrorResponse - 创建错误响应
type IntegrationErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

func NewIntegrationErrorResponse(code, message, details string) *IntegrationErrorResponse {
	return &IntegrationErrorResponse{
		Code:    code,
		Message: message,
		Details: details,
	}
}

// PaginationParams - 分页参数
type PaginationParams struct {
	Page   int `json:"page"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

// DefaultPagination - 默认分页
func DefaultPagination() PaginationParams {
	return PaginationParams{
		Page:   1,
		Limit:  20,
		Offset: 0,
	}
}

// IntegrationPaginatedResponse - 分页响应
type IntegrationPaginatedResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
	Meta    Meta        `json:"meta"`
}

// Meta - 元信息
type Meta struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

// NewIntegrationPaginatedResponse - 创建分页响应
func NewIntegrationPaginatedResponse(data interface{}, total int64, params PaginationParams) *IntegrationPaginatedResponse {
	totalPages := int(total) / params.Limit
	if int(total)%params.Limit > 0 {
		totalPages++
	}

	return &IntegrationPaginatedResponse{
		Success: true,
		Data:    data,
		Meta: Meta{
			Page:       params.Page,
			Limit:      params.Limit,
			Total:      total,
			TotalPages: totalPages,
		},
	}
}
