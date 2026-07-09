package routing_test

import (
	"context"
	"testing"

	"github.com/al-hudhud/backend/internal/routing"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockRouteRepository - Mock for route operations
type MockRouteRepository struct {
	mock.Mock
}

func (m *MockRouteRepository) GetRoutes(ctx context.Context, tenantID uuid.UUID) ([]routing.Route, error) {
	args := m.Called(ctx, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]routing.Route), args.Error(1)
}

func (m *MockRouteRepository) GetRouteByID(ctx context.Context, routeID uuid.UUID) (*routing.Route, error) {
	args := m.Called(ctx, routeID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*routing.Route), args.Error(1)
}

func (m *MockRouteRepository) CreateRoute(ctx context.Context, route *routing.Route) error {
	args := m.Called(ctx, route)
	return args.Error(0)
}

func (m *MockRouteRepository) UpdateRouteStatus(ctx context.Context, routeID uuid.UUID, status string) error {
	args := m.Called(ctx, routeID, status)
	return args.Error(0)
}

func (m *MockRouteRepository) UpdateRouteLoad(ctx context.Context, routeID uuid.UUID, load int) error {
	args := m.Called(ctx, routeID, load)
	return args.Error(0)
}

func (m *MockRouteRepository) DeleteRoute(ctx context.Context, routeID uuid.UUID) error {
	args := m.Called(ctx, routeID)
	return args.Error(0)
}

func TestRoutingEngine_SelectOptimalRoute_ByCost(t *testing.T) {
	// Arrange
	tenantID := uuid.New()
	routes := []routing.Route{
		{ID: uuid.New(), Name: "Route A", Cost: decimal.NewFromFloat(0.05), SuccessRate: 0.99, Latency: 100},
		{ID: uuid.New(), Name: "Route B", Cost: decimal.NewFromFloat(0.03), SuccessRate: 0.95, Latency: 200},
		{ID: uuid.New(), Name: "Route C", Cost: decimal.NewFromFloat(0.04), SuccessRate: 0.97, Latency: 150},
	}

	mockRepo := new(MockRouteRepository)
	mockRepo.On("GetRoutes", mock.Anything, tenantID).Return(routes, nil)

	engine := routing.NewRouter(mockRepo)

	// Act
	selected, err := engine.SelectOptimalRoute(context.Background(), tenantID, routing.OptimizeCost)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, selected)
	assert.Equal(t, "Route B", selected.Name) // Cheapest route
	mockRepo.AssertExpectations(t)
}

func TestRoutingEngine_SelectOptimalRoute_BySpeed(t *testing.T) {
	// Arrange
	tenantID := uuid.New()
	routes := []routing.Route{
		{ID: uuid.New(), Name: "Fast Route", Cost: decimal.NewFromFloat(0.08), SuccessRate: 0.99, Latency: 50},
		{ID: uuid.New(), Name: "Slow Route", Cost: decimal.NewFromFloat(0.03), SuccessRate: 0.95, Latency: 500},
		{ID: uuid.New(), Name: "Medium Route", Cost: decimal.NewFromFloat(0.05), SuccessRate: 0.97, Latency: 200},
	}

	mockRepo := new(MockRouteRepository)
	mockRepo.On("GetRoutes", mock.Anything, tenantID).Return(routes, nil)

	engine := routing.NewRouter(mockRepo)

	// Act
	selected, err := engine.SelectOptimalRoute(context.Background(), tenantID, routing.OptimizeSpeed)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, selected)
	assert.Equal(t, "Fast Route", selected.Name) // Fastest route
	mockRepo.AssertExpectations(t)
}

func TestRoutingEngine_SelectOptimalRoute_ByReliability(t *testing.T) {
	// Arrange
	tenantID := uuid.New()
	routes := []routing.Route{
		{ID: uuid.New(), Name: "Reliable Route", Cost: decimal.NewFromFloat(0.06), SuccessRate: 0.999, Latency: 100},
		{ID: uuid.New(), Name: "Unreliable Route", Cost: decimal.NewFromFloat(0.02), SuccessRate: 0.85, Latency: 50},
		{ID: uuid.New(), Name: "Medium Route", Cost: decimal.NewFromFloat(0.04), SuccessRate: 0.95, Latency: 150},
	}

	mockRepo := new(MockRouteRepository)
	mockRepo.On("GetRoutes", mock.Anything, tenantID).Return(routes, nil)

	engine := routing.NewRouter(mockRepo)

	// Act
	selected, err := engine.SelectOptimalRoute(context.Background(), tenantID, routing.OptimizeReliability)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, selected)
	assert.Equal(t, "Reliable Route", selected.Name) // Most reliable
	mockRepo.AssertExpectations(t)
}

func TestRoutingEngine_SelectOptimalRoute_ExcludesOffline(t *testing.T) {
	// Arrange
	tenantID := uuid.New()
	routes := []routing.Route{
		{ID: uuid.New(), Name: "Online Route", Cost: decimal.NewFromFloat(0.05), SuccessRate: 0.99, Latency: 100, Status: "ONLINE"},
		{ID: uuid.New(), Name: "Offline Route", Cost: decimal.NewFromFloat(0.02), SuccessRate: 0.99, Latency: 50, Status: "OFFLINE"},
		{ID: uuid.New(), Name: "Maintenance Route", Cost: decimal.NewFromFloat(0.03), SuccessRate: 0.99, Latency: 75, Status: "MAINTENANCE"},
	}

	mockRepo := new(MockRouteRepository)
	mockRepo.On("GetRoutes", mock.Anything, tenantID).Return(routes, nil)

	engine := routing.NewRouter(mockRepo)

	// Act
	selected, err := engine.SelectOptimalRoute(context.Background(), tenantID, routing.OptimizeCost)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, selected)
	assert.Equal(t, "Online Route", selected.Name)
	mockRepo.AssertExpectations(t)
}

func TestRoutingEngine_SelectOptimalRoute_NoRoutesAvailable(t *testing.T) {
	// Arrange
	tenantID := uuid.New()

	mockRepo := new(MockRouteRepository)
	mockRepo.On("GetRoutes", mock.Anything, tenantID).Return([]routing.Route{}, nil)

	engine := routing.NewRouter(mockRepo)

	// Act
	selected, err := engine.SelectOptimalRoute(context.Background(), tenantID, routing.OptimizeCost)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, routing.ErrNoAvailableRoutes, err)
	assert.Nil(t, selected)
	mockRepo.AssertExpectations(t)
}

func TestRoutingEngine_FallbackRoute(t *testing.T) {
	// Arrange
	tenantID := uuid.New()
	routes := []routing.Route{
		{ID: uuid.New(), Name: "Primary", Cost: decimal.NewFromFloat(0.05), SuccessRate: 0.99, Latency: 100, Status: "ONLINE"},
		{ID: uuid.New(), Name: "Backup", Cost: decimal.NewFromFloat(0.08), SuccessRate: 0.95, Latency: 200, Status: "ONLINE"},
	}

	mockRepo := new(MockRouteRepository)
	mockRepo.On("GetRoutes", mock.Anything, tenantID).Return(routes, nil)

	engine := routing.NewRouter(mockRepo)

	// Act - Select primary
	primary, _ := engine.SelectOptimalRoute(context.Background(), tenantID, routing.OptimizeCost)
	
	// Set primary as failed
	engine.MarkRouteFailed(primary.ID)
	
	// Act - Should select backup
	backup, err := engine.SelectOptimalRoute(context.Background(), tenantID, routing.OptimizeCost)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, backup)
	assert.Equal(t, "Backup", backup.Name)
}

func TestRoutingEngine_LoadBalancing(t *testing.T) {
	// Arrange
	tenantID := uuid.New()
	routes := []routing.Route{
		{ID: uuid.New(), Name: "Route 1", Cost: decimal.NewFromFloat(0.05), SuccessRate: 0.99, Latency: 100, CurrentLoad: 90},
		{ID: uuid.New(), Name: "Route 2", Cost: decimal.NewFromFloat(0.05), SuccessRate: 0.99, Latency: 100, CurrentLoad: 10},
	}

	mockRepo := new(MockRouteRepository)
	mockRepo.On("GetRoutes", mock.Anything, tenantID).Return(routes, nil)

	engine := routing.NewRouter(mockRepo)

	// Act - Should select less loaded route
	selected, err := engine.SelectOptimalRoute(context.Background(), tenantID, routing.OptimizeLoad)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, selected)
	assert.Equal(t, "Route 2", selected.Name)
}

func TestRoutingEngine_UpdateRouteLoad(t *testing.T) {
	// Arrange
	routeID := uuid.New()
	newLoad := 75

	mockRepo := new(MockRouteRepository)
	mockRepo.On("UpdateRouteLoad", mock.Anything, routeID, newLoad).Return(nil)

	engine := routing.NewRouter(mockRepo)

	// Act
	err := engine.UpdateRouteLoad(context.Background(), routeID, newLoad)

	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestRoutingEngine_GetRouteStats(t *testing.T) {
	// Arrange
	tenantID := uuid.New()
	routes := []routing.Route{
		{ID: uuid.New(), Name: "Route 1", SuccessRate: 0.99, Latency: 100, TotalRequests: 1000},
		{ID: uuid.New(), Name: "Route 2", SuccessRate: 0.95, Latency: 150, TotalRequests: 500},
	}

	mockRepo := new(MockRouteRepository)
	mockRepo.On("GetRoutes", mock.Anything, tenantID).Return(routes, nil)

	engine := routing.NewRouter(mockRepo)

	// Act
	stats, err := engine.GetRouteStats(context.Background(), tenantID)

	// Assert
	assert.NoError(t, err)
	assert.Len(t, stats, 2)
	assert.Equal(t, 1500, stats["total_requests"])
	assert.Equal(t, 0.98, stats["average_success_rate"])
}