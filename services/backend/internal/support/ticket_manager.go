package support

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

var (
	ErrTicketNotFound      = errors.New("ticket not found")
	ErrUnauthorizedAccess  = errors.New("unauthorized access to ticket")
	ErrInvalidStatus       = errors.New("invalid status transition")
	ErrInvalidPriority     = errors.New("invalid priority")
)

// TicketManager manages support tickets
type TicketManager struct {
	db *sql.DB
}

// NewTicketManager creates a new ticket manager
func NewTicketManager(db *sql.DB) *TicketManager {
	return &TicketManager{db: db}
}

// CreateTicket creates a new support ticket
func (m *TicketManager) CreateTicket(ctx context.Context, tenantID, userID int64, req *CreateTicketRequest) (*Ticket, error) {
	// Calculate SLA due time based on priority
	slaDueAt := calculateSLADueTime(req.Priority)
	
	query := `
		INSERT INTO support_tickets (
			tenant_id, user_id, ticket_number, subject, description,
			priority, status, category, sla_due_at
		) VALUES (
			$1, $2, '', $3, $4, $5, $6, $7, $8
		) RETURNING id, ticket_number, created_at, updated_at
	`
	
	ticket := &Ticket{
		TenantID:    tenantID,
		UserID:      userID,
		Subject:     req.Subject,
		Description: req.Description,
		Priority:    req.Priority,
		Status:      StatusOpen,
		Category:    req.Category,
		SLADueAt:    &slaDueAt,
	}
	
	err := m.db.QueryRowContext(
		ctx, query,
		tenantID, userID, req.Subject, req.Description,
		req.Priority, StatusOpen, req.Category, slaDueAt,
	).Scan(&ticket.ID, &ticket.TicketNumber, &ticket.CreatedAt, &ticket.UpdatedAt)
	
	if err != nil {
		return nil, fmt.Errorf("failed to create ticket: %w", err)
	}
	
	return ticket, nil
}

// GetTicket retrieves a ticket by ID
func (m *TicketManager) GetTicket(ctx context.Context, tenantID, ticketID int64) (*Ticket, error) {
	query := `
		SELECT 
			id, tenant_id, user_id, ticket_number, subject, description,
			priority, status, category, assigned_to, created_at, updated_at,
			resolved_at, closed_at, first_response_at, sla_due_at
		FROM support_tickets
		WHERE id = $1 AND tenant_id = $2
	`
	
	ticket := &Ticket{}
	err := m.db.QueryRowContext(ctx, query, ticketID, tenantID).Scan(
		&ticket.ID, &ticket.TenantID, &ticket.UserID, &ticket.TicketNumber,
		&ticket.Subject, &ticket.Description, &ticket.Priority, &ticket.Status,
		&ticket.Category, &ticket.AssignedTo, &ticket.CreatedAt, &ticket.UpdatedAt,
		&ticket.ResolvedAt, &ticket.ClosedAt, &ticket.FirstResponseAt, &ticket.SLADueAt,
	)
	
	if err == sql.ErrNoRows {
		return nil, ErrTicketNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get ticket: %w", err)
	}
	
	return ticket, nil
}

// ListTickets lists tickets with filters
func (m *TicketManager) ListTickets(ctx context.Context, tenantID int64, filters *TicketFilters) ([]Ticket, error) {
	query := `
		SELECT 
			id, tenant_id, user_id, ticket_number, subject, description,
			priority, status, category, assigned_to, created_at, updated_at,
			resolved_at, closed_at, first_response_at, sla_due_at
		FROM support_tickets
		WHERE tenant_id = $1
	`
	
	args := []interface{}{tenantID}
	argCount := 1
	
	// Apply filters
	if filters.Status != nil {
		argCount++
		query += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, *filters.Status)
	}
	
	if filters.Priority != nil {
		argCount++
		query += fmt.Sprintf(" AND priority = $%d", argCount)
		args = append(args, *filters.Priority)
	}
	
	if filters.Category != nil {
		argCount++
		query += fmt.Sprintf(" AND category = $%d", argCount)
		args = append(args, *filters.Category)
	}
	
	if filters.AssignedTo != nil {
		argCount++
		query += fmt.Sprintf(" AND assigned_to = $%d", argCount)
		args = append(args, *filters.AssignedTo)
	}
	
	if filters.UserID != nil {
		argCount++
		query += fmt.Sprintf(" AND user_id = $%d", argCount)
		args = append(args, *filters.UserID)
	}
	
	query += " ORDER BY created_at DESC"
	
	// Apply pagination
	if filters.Limit > 0 {
		argCount++
		query += fmt.Sprintf(" LIMIT $%d", argCount)
		args = append(args, filters.Limit)
	}
	
	if filters.Offset > 0 {
		argCount++
		query += fmt.Sprintf(" OFFSET $%d", argCount)
		args = append(args, filters.Offset)
	}
	
	rows, err := m.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list tickets: %w", err)
	}
	defer rows.Close()
	
	var tickets []Ticket
	for rows.Next() {
		var ticket Ticket
		err := rows.Scan(
			&ticket.ID, &ticket.TenantID, &ticket.UserID, &ticket.TicketNumber,
			&ticket.Subject, &ticket.Description, &ticket.Priority, &ticket.Status,
			&ticket.Category, &ticket.AssignedTo, &ticket.CreatedAt, &ticket.UpdatedAt,
			&ticket.ResolvedAt, &ticket.ClosedAt, &ticket.FirstResponseAt, &ticket.SLADueAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan ticket: %w", err)
		}
		tickets = append(tickets, ticket)
	}
	
	return tickets, nil
}

// ListAllTickets lists all tickets across all tenants for admin
func (m *TicketManager) ListAllTickets(ctx context.Context, limit, offset int) ([]Ticket, error) {
	query := `
		SELECT 
			id, tenant_id, user_id, ticket_number, subject, description,
			priority, status, category, assigned_to, created_at, updated_at,
			resolved_at, closed_at, first_response_at, sla_due_at
		FROM support_tickets
		ORDER BY created_at DESC
	`
	
	args := []interface{}{}
	argCount := 0
	
	if limit > 0 {
		argCount++
		query += fmt.Sprintf(" LIMIT $%d", argCount)
		args = append(args, limit)
	}
	
	if offset > 0 {
		argCount++
		query += fmt.Sprintf(" OFFSET $%d", argCount)
		args = append(args, offset)
	}
	
	rows, err := m.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list all tickets: %w", err)
	}
	defer rows.Close()
	
	var tickets []Ticket
	for rows.Next() {
		var ticket Ticket
		err := rows.Scan(
			&ticket.ID, &ticket.TenantID, &ticket.UserID, &ticket.TicketNumber,
			&ticket.Subject, &ticket.Description, &ticket.Priority, &ticket.Status,
			&ticket.Category, &ticket.AssignedTo, &ticket.CreatedAt, &ticket.UpdatedAt,
			&ticket.ResolvedAt, &ticket.ClosedAt, &ticket.FirstResponseAt, &ticket.SLADueAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan ticket: %w", err)
		}
		tickets = append(tickets, ticket)
	}
	
	return tickets, nil
}

// UpdateTicket updates ticket status/priority/assignment
func (m *TicketManager) UpdateTicket(ctx context.Context, tenantID, ticketID int64, req *UpdateTicketRequest) error {
	// Get current ticket
	ticket, err := m.GetTicket(ctx, tenantID, ticketID)
	if err != nil {
		return err
	}
	
	// Build update query dynamically
	updates := []string{}
	args := []interface{}{}
	argCount := 0
	
	if req.Status != nil {
		// Validate status transition
		if !isValidStatusTransition(ticket.Status, *req.Status) {
			return ErrInvalidStatus
		}
		argCount++
		updates = append(updates, fmt.Sprintf("status = $%d", argCount))
		args = append(args, *req.Status)
		
		// Set resolved_at if status is resolved
		if *req.Status == StatusResolved {
			now := time.Now()
			argCount++
			updates = append(updates, fmt.Sprintf("resolved_at = $%d", argCount))
			args = append(args, now)
		}
		
		// Set closed_at if status is closed
		if *req.Status == StatusClosed {
			now := time.Now()
			argCount++
			updates = append(updates, fmt.Sprintf("closed_at = $%d", argCount))
			args = append(args, now)
		}
	}
	
	if req.Priority != nil {
		argCount++
		updates = append(updates, fmt.Sprintf("priority = $%d", argCount))
		args = append(args, *req.Priority)
		
		// Recalculate SLA if priority changed
		newSLA := calculateSLADueTime(*req.Priority)
		argCount++
		updates = append(updates, fmt.Sprintf("sla_due_at = $%d", argCount))
		args = append(args, newSLA)
	}
	
	if req.AssignedTo != nil {
		argCount++
		updates = append(updates, fmt.Sprintf("assigned_to = $%d", argCount))
		args = append(args, *req.AssignedTo)
	}
	
	if len(updates) == 0 {
		return nil // Nothing to update
	}
	
	// Add WHERE clause parameters
	argCount++
	args = append(args, ticketID)
	argCount++
	args = append(args, tenantID)
	
	query := fmt.Sprintf(
		"UPDATE support_tickets SET %s WHERE id = $%d AND tenant_id = $%d",
		joinStrings(updates, ", "),
		argCount-1,
		argCount,
	)
	
	_, err = m.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update ticket: %w", err)
	}
	
	return nil
}

// CloseTicket closes a ticket
func (m *TicketManager) CloseTicket(ctx context.Context, tenantID, ticketID int64) error {
	req := &UpdateTicketRequest{
		Status: ptrStatus(StatusClosed),
	}
	return m.UpdateTicket(ctx, tenantID, ticketID, req)
}

// AddComment adds a comment to a ticket
func (m *TicketManager) AddComment(ctx context.Context, tenantID, ticketID, userID int64, isStaff bool, req *AddCommentRequest) (*Comment, error) {
	// Verify ticket exists and belongs to tenant
	_, err := m.GetTicket(ctx, tenantID, ticketID)
	if err != nil {
		return nil, err
	}
	
	query := `
		INSERT INTO ticket_comments (ticket_id, user_id, is_staff, comment)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at
	`
	
	comment := &Comment{
		TicketID: ticketID,
		UserID:   userID,
		IsStaff:  isStaff,
		Comment:  req.Comment,
	}
	
	err = m.db.QueryRowContext(ctx, query, ticketID, userID, isStaff, req.Comment).
		Scan(&comment.ID, &comment.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to add comment: %w", err)
	}
	
	// Set first_response_at if this is the first staff comment
	if isStaff {
		_, err = m.db.ExecContext(ctx, `
			UPDATE support_tickets
			SET first_response_at = $1
			WHERE id = $2 AND first_response_at IS NULL
		`, comment.CreatedAt, ticketID)
		if err != nil {
			return nil, fmt.Errorf("failed to update first_response_at: %w", err)
		}
	}
	
	return comment, nil
}

// GetComments retrieves all comments for a ticket
func (m *TicketManager) GetComments(ctx context.Context, tenantID, ticketID int64) ([]Comment, error) {
	// Verify ticket exists and belongs to tenant
	_, err := m.GetTicket(ctx, tenantID, ticketID)
	if err != nil {
		return nil, err
	}
	
	query := `
		SELECT id, ticket_id, user_id, is_staff, comment, created_at
		FROM ticket_comments
		WHERE ticket_id = $1
		ORDER BY created_at ASC
	`
	
	rows, err := m.db.QueryContext(ctx, query, ticketID)
	if err != nil {
		return nil, fmt.Errorf("failed to get comments: %w", err)
	}
	defer rows.Close()
	
	var comments []Comment
	for rows.Next() {
		var comment Comment
		err := rows.Scan(
			&comment.ID, &comment.TicketID, &comment.UserID,
			&comment.IsStaff, &comment.Comment, &comment.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan comment: %w", err)
		}
		comments = append(comments, comment)
	}
	
	return comments, nil
}

// AddAttachment adds an attachment to a ticket
func (m *TicketManager) AddAttachment(ctx context.Context, tenantID, ticketID, userID int64, attachment *Attachment) error {
	// Verify ticket exists and belongs to tenant
	_, err := m.GetTicket(ctx, tenantID, ticketID)
	if err != nil {
		return err
	}
	
	query := `
		INSERT INTO ticket_attachments (
			ticket_id, filename, file_path, file_size, mime_type, uploaded_by
		) VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at
	`
	
	err = m.db.QueryRowContext(
		ctx, query,
		ticketID, attachment.Filename, attachment.FilePath,
		attachment.FileSize, attachment.MimeType, userID,
	).Scan(&attachment.ID, &attachment.CreatedAt)
	
	if err != nil {
		return fmt.Errorf("failed to add attachment: %w", err)
	}
	
	return nil
}

// Helper functions

func calculateSLADueTime(priority Priority) time.Time {
	now := time.Now()
	
	switch priority {
	case PriorityUrgent:
		return now.Add(4 * time.Hour)
	case PriorityHigh:
		return now.Add(24 * time.Hour)
	case PriorityMedium:
		return now.Add(48 * time.Hour)
	case PriorityLow:
		return now.Add(72 * time.Hour)
	default:
		return now.Add(48 * time.Hour)
	}
}

func isValidStatusTransition(from, to Status) bool {
	// Define valid status transitions
	validTransitions := map[Status][]Status{
		StatusOpen: {
			StatusInProgress,
			StatusWaitingCustomer,
			StatusResolved,
			StatusClosed,
		},
		StatusInProgress: {
			StatusWaitingCustomer,
			StatusResolved,
			StatusClosed,
			StatusOpen,
		},
		StatusWaitingCustomer: {
			StatusInProgress,
			StatusResolved,
			StatusClosed,
		},
		StatusResolved: {
			StatusClosed,
			StatusOpen, // Reopen
		},
		StatusClosed: {
			StatusOpen, // Reopen
		},
	}
	
	allowed, exists := validTransitions[from]
	if !exists {
		return false
	}
	
	for _, status := range allowed {
		if status == to {
			return true
		}
	}
	
	return false
}

func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}

func ptrStatus(s Status) *Status {
	return &s
}
