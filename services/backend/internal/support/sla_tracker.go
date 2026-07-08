package support

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/al-hudhud/backend/pkg/logger"
	"go.uber.org/zap"
)

// SLAConfig defines SLA configuration for each priority
type SLAConfig struct {
	Priority         Priority
	FirstResponseSLA time.Duration // Time to first response
	ResolutionSLA    time.Duration // Time to resolution
}

// DefaultSLAConfig defines default SLA for standard customers
var DefaultSLAConfig = map[Priority]SLAConfig{
	PriorityUrgent: {
		Priority:         PriorityUrgent,
		FirstResponseSLA: 1 * time.Hour,
		ResolutionSLA:    4 * time.Hour,
	},
	PriorityHigh: {
		Priority:         PriorityHigh,
		FirstResponseSLA: 4 * time.Hour,
		ResolutionSLA:    24 * time.Hour,
	},
	PriorityMedium: {
		Priority:         PriorityMedium,
		FirstResponseSLA: 8 * time.Hour,
		ResolutionSLA:    48 * time.Hour,
	},
	PriorityLow: {
		Priority:         PriorityLow,
		FirstResponseSLA: 24 * time.Hour,
		ResolutionSLA:    72 * time.Hour,
	},
}

// EnterpriseSLAConfig defines enhanced SLA for enterprise customers
var EnterpriseSLAConfig = map[Priority]SLAConfig{
	PriorityUrgent: {
		Priority:         PriorityUrgent,
		FirstResponseSLA: 15 * time.Minute,
		ResolutionSLA:    2 * time.Hour,
	},
	PriorityHigh: {
		Priority:         PriorityHigh,
		FirstResponseSLA: 1 * time.Hour,
		ResolutionSLA:    8 * time.Hour,
	},
	PriorityMedium: {
		Priority:         PriorityMedium,
		FirstResponseSLA: 4 * time.Hour,
		ResolutionSLA:    24 * time.Hour,
	},
	PriorityLow: {
		Priority:         PriorityLow,
		FirstResponseSLA: 8 * time.Hour,
		ResolutionSLA:    48 * time.Hour,
	},
}

// SLATracker tracks and monitors SLA compliance
type SLATracker struct {
	db                *sql.DB
	notificationSvc   *NotificationService
}

// NewSLATracker creates a new SLA tracker
func NewSLATracker(db *sql.DB, notificationSvc *NotificationService) *SLATracker {
	return &SLATracker{
		db:              db,
		notificationSvc: notificationSvc,
	}
}

// CalculateSLADueTime calculates when SLA is due for a ticket
func (t *SLATracker) CalculateSLADueTime(ticket *Ticket, isEnterprise bool) time.Time {
	config := DefaultSLAConfig
	if isEnterprise {
		config = EnterpriseSLAConfig
	}
	
	slaConfig := config[ticket.Priority]
	return ticket.CreatedAt.Add(slaConfig.ResolutionSLA)
}

// CheckSLABreach checks if ticket has breached SLA
func (t *SLATracker) CheckSLABreach(ctx context.Context, ticketID int64) (bool, error) {
	query := `
		SELECT sla_due_at, status
		FROM support_tickets
		WHERE id = $1
	`
	
	var slaDueAt *time.Time
	var status Status
	
	err := t.db.QueryRowContext(ctx, query, ticketID).Scan(&slaDueAt, &status)
	if err != nil {
		return false, fmt.Errorf("failed to check SLA: %w", err)
	}
	
	// If ticket is resolved or closed, no breach
	if status == StatusResolved || status == StatusClosed {
		return false, nil
	}
	
	// Check if SLA is breached
	if slaDueAt != nil && time.Now().After(*slaDueAt) {
		return true, nil
	}
	
	return false, nil
}

// CheckFirstResponseSLA checks if first response SLA is breached
func (t *SLATracker) CheckFirstResponseSLA(ctx context.Context, ticket *Ticket, isEnterprise bool) (bool, error) {
	// If already responded, no breach
	if ticket.FirstResponseAt != nil {
		return false, nil
	}
	
	config := DefaultSLAConfig
	if isEnterprise {
		config = EnterpriseSLAConfig
	}
	
	slaConfig := config[ticket.Priority]
	firstResponseDue := ticket.CreatedAt.Add(slaConfig.FirstResponseSLA)
	
	return time.Now().After(firstResponseDue), nil
}

// SendSLAAlerts sends alerts for tickets approaching SLA breach
func (t *SLATracker) SendSLAAlerts(ctx context.Context) error {
	// Find tickets approaching SLA (80% of time elapsed)
	query := `
		SELECT 
			id, tenant_id, user_id, ticket_number, subject, description,
			priority, status, category, assigned_to, created_at, updated_at,
			resolved_at, closed_at, first_response_at, sla_due_at
		FROM support_tickets
		WHERE status NOT IN ('resolved', 'closed')
		AND sla_due_at IS NOT NULL
		AND sla_due_at > NOW()
		AND sla_due_at < NOW() + INTERVAL '20%' * (sla_due_at - created_at)
	`
	
	rows, err := t.db.QueryContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to query tickets for SLA alerts: %w", err)
	}
	defer rows.Close()
	
	for rows.Next() {
		var ticket Ticket
		err := rows.Scan(
			&ticket.ID, &ticket.TenantID, &ticket.UserID, &ticket.TicketNumber,
			&ticket.Subject, &ticket.Description, &ticket.Priority, &ticket.Status,
			&ticket.Category, &ticket.AssignedTo, &ticket.CreatedAt, &ticket.UpdatedAt,
			&ticket.ResolvedAt, &ticket.ClosedAt, &ticket.FirstResponseAt, &ticket.SLADueAt,
		)
		if err != nil {
			return fmt.Errorf("failed to scan ticket: %w", err)
		}
		
		// Send alert to assigned staff
		if ticket.AssignedTo != nil {
			// TODO: Get staff email from database
			staffEmail := "support@example.com"
			err = t.notificationSvc.SendSLAAlert(ctx, &ticket, staffEmail)
			if err != nil {
				logger.GetLogger().Error("Failed to send SLA alert",
					zap.String("ticket_number", ticket.TicketNumber),
					zap.Error(err),
				)
			}
		}
	}
	
	return nil
}

// CheckSLABreaches checks for SLA breaches and sends notifications
func (t *SLATracker) CheckSLABreaches(ctx context.Context) error {
	// Find tickets that have breached SLA
	query := `
		SELECT 
			id, tenant_id, user_id, ticket_number, subject, description,
			priority, status, category, assigned_to, created_at, updated_at,
			resolved_at, closed_at, first_response_at, sla_due_at
		FROM support_tickets
		WHERE status NOT IN ('resolved', 'closed')
		AND sla_due_at IS NOT NULL
		AND sla_due_at < NOW()
	`
	
	rows, err := t.db.QueryContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to query tickets for SLA breaches: %w", err)
	}
	defer rows.Close()
	
	for rows.Next() {
		var ticket Ticket
		err := rows.Scan(
			&ticket.ID, &ticket.TenantID, &ticket.UserID, &ticket.TicketNumber,
			&ticket.Subject, &ticket.Description, &ticket.Priority, &ticket.Status,
			&ticket.Category, &ticket.AssignedTo, &ticket.CreatedAt, &ticket.UpdatedAt,
			&ticket.ResolvedAt, &ticket.ClosedAt, &ticket.FirstResponseAt, &ticket.SLADueAt,
		)
		if err != nil {
			return fmt.Errorf("failed to scan ticket: %w", err)
		}
		
		// Send breach notification to assigned staff
		if ticket.AssignedTo != nil {
			// TODO: Get staff email from database
			staffEmail := "support@example.com"
			err = t.notificationSvc.SendSLABreach(ctx, &ticket, staffEmail)
			if err != nil {
				logger.GetLogger().Error("Failed to send SLA breach notification",
					zap.String("ticket_number", ticket.TicketNumber),
					zap.Error(err),
				)
			}
		}
	}
	
	return nil
}

// GetSLAMetrics gets SLA performance metrics for a tenant
func (t *SLATracker) GetSLAMetrics(ctx context.Context, tenantID int64, from, to time.Time) (*SLAMetrics, error) {
	query := `
		SELECT 
			COUNT(*) as total_tickets,
			COUNT(CASE WHEN resolved_at IS NOT NULL AND resolved_at <= sla_due_at THEN 1 END) as within_sla,
			COUNT(CASE WHEN resolved_at IS NOT NULL AND resolved_at > sla_due_at THEN 1 END) as breached_sla,
			AVG(EXTRACT(EPOCH FROM (first_response_at - created_at))) as avg_first_response_seconds,
			AVG(EXTRACT(EPOCH FROM (resolved_at - created_at))) as avg_resolution_seconds
		FROM support_tickets
		WHERE tenant_id = $1
		AND created_at >= $2
		AND created_at <= $3
		AND status IN ('resolved', 'closed')
	`
	
	var totalTickets, withinSLA, breachedSLA int
	var avgFirstResponse, avgResolution sql.NullFloat64
	
	err := t.db.QueryRowContext(ctx, query, tenantID, from, to).Scan(
		&totalTickets, &withinSLA, &breachedSLA,
		&avgFirstResponse, &avgResolution,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get SLA metrics: %w", err)
	}
	
	complianceRate := 0.0
	if totalTickets > 0 {
		complianceRate = float64(withinSLA) / float64(totalTickets) * 100
	}
	
	metrics := &SLAMetrics{
		TotalTickets:         totalTickets,
		TicketsWithinSLA:     withinSLA,
		TicketsBreachedSLA:   breachedSLA,
		SLAComplianceRate:    complianceRate,
		AvgFirstResponseTime: avgFirstResponse.Float64,
		AvgResolutionTime:    avgResolution.Float64,
	}
	
	return metrics, nil
}

// IsEnterpriseCustomer checks if customer has enterprise support
func (t *SLATracker) IsEnterpriseCustomer(ctx context.Context, tenantID int64) (bool, error) {
	// Check if tenant has dedicated gateway (enterprise feature)
	query := `
		SELECT EXISTS(
			SELECT 1 FROM dedicated_gateways
			WHERE tenant_id = $1 AND status = 'active'
		)
	`
	
	var isEnterprise bool
	err := t.db.QueryRowContext(ctx, query, tenantID).Scan(&isEnterprise)
	if err != nil {
		return false, fmt.Errorf("failed to check enterprise status: %w", err)
	}
	
	return isEnterprise, nil
}

// GetSLAConfig returns appropriate SLA config based on customer type
func (t *SLATracker) GetSLAConfig(ctx context.Context, tenantID int64) (map[Priority]SLAConfig, error) {
	isEnterprise, err := t.IsEnterpriseCustomer(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	
	if isEnterprise {
		return EnterpriseSLAConfig, nil
	}
	
	return DefaultSLAConfig, nil
}

// SLAMetrics represents SLA performance metrics
type SLAMetrics struct {
	TotalTickets         int     `json:"total_tickets"`
	TicketsWithinSLA     int     `json:"tickets_within_sla"`
	TicketsBreachedSLA   int     `json:"tickets_breached_sla"`
	SLAComplianceRate    float64 `json:"sla_compliance_rate"`
	AvgFirstResponseTime float64 `json:"avg_first_response_time"` // in seconds
	AvgResolutionTime    float64 `json:"avg_resolution_time"`     // in seconds
}
