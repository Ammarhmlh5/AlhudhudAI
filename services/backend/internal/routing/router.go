package routing

import (
	"context"
	"errors"
	"sort"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Route struct {
	ID           uuid.UUID       `json:"id"`
	Name         string          `json:"name"`
	Cost         decimal.Decimal `json:"cost"`
	SuccessRate  float64         `json:"success_rate"`
	Latency      int             `json:"latency"`
	Status       string          `json:"status"`
	CurrentLoad  int             `json:"current_load"`
	TotalRequests int            `json:"total_requests"`
}

const (
	OptimizeCost        = "cost"
	OptimizeSpeed       = "speed"
	OptimizeReliability = "reliability"
	OptimizeLoad        = "load"
)

var ErrNoAvailableRoutes = errors.New("no available routes")

type RouteRepository interface {
	GetRoutes(ctx context.Context, tenantID uuid.UUID) ([]Route, error)
	GetRouteByID(ctx context.Context, routeID uuid.UUID) (*Route, error)
	CreateRoute(ctx context.Context, route *Route) error
	UpdateRouteStatus(ctx context.Context, routeID uuid.UUID, status string) error
	DeleteRoute(ctx context.Context, routeID uuid.UUID) error
	UpdateRouteLoad(ctx context.Context, routeID uuid.UUID, load int) error
}

type Router struct {
	repo    RouteRepository
	failed  map[uuid.UUID]bool
}

func NewRouter(repo RouteRepository) *Router {
	return &Router{
		repo:   repo,
		failed: make(map[uuid.UUID]bool),
	}
}

func (r *Router) SelectOptimalRoute(ctx context.Context, tenantID uuid.UUID, optimize string) (*Route, error) {
	routes, err := r.repo.GetRoutes(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	var available []Route
	for _, route := range routes {
		if r.failed[route.ID] {
			continue
		}
		if route.Status != "" && route.Status != "ONLINE" {
			continue
		}
		available = append(available, route)
	}

	if len(available) == 0 {
		return nil, ErrNoAvailableRoutes
	}

	switch optimize {
	case OptimizeCost:
		sort.Slice(available, func(i, j int) bool {
			return available[i].Cost.LessThan(available[j].Cost)
		})
	case OptimizeSpeed:
		sort.Slice(available, func(i, j int) bool {
			return available[i].Latency < available[j].Latency
		})
	case OptimizeReliability:
		sort.Slice(available, func(i, j int) bool {
			return available[i].SuccessRate > available[j].SuccessRate
		})
	case OptimizeLoad:
		sort.Slice(available, func(i, j int) bool {
			return available[i].CurrentLoad < available[j].CurrentLoad
		})
	}

	return &available[0], nil
}

func (r *Router) MarkRouteFailed(routeID uuid.UUID) {
	r.failed[routeID] = true
}

func (r *Router) UpdateRouteLoad(ctx context.Context, routeID uuid.UUID, load int) error {
	return r.repo.UpdateRouteLoad(ctx, routeID, load)
}

func (r *Router) GetRouteStats(ctx context.Context, tenantID uuid.UUID) (map[string]interface{}, error) {
	routes, err := r.repo.GetRoutes(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	var totalRequests int
	var totalSuccessRate float64
	for _, route := range routes {
		totalRequests += route.TotalRequests
		totalSuccessRate += route.SuccessRate
	}

	avgSuccessRate := float64(0)
	if len(routes) > 0 {
		avgSuccessRate = totalSuccessRate / float64(len(routes))
	}

	return map[string]interface{}{
		"total_requests":       totalRequests,
		"average_success_rate": avgSuccessRate,
	}, nil
}
