package gateway

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/al-hudhud/backend/internal/metrics"
	"github.com/google/uuid"
)

// MessageBuffer interface for buffering messages
type MessageBuffer interface {
	Push(ctx context.Context, tenantID uuid.UUID, msg BufferedMessage) error
	GetSize(ctx context.Context, tenantID uuid.UUID) (int64, error)
}

// RateLimiter interface for rate limiting messages
type RateLimiter interface {
	Allow(ctx context.Context, tenantID uuid.UUID) (bool, *RateLimitInfo, error)
}

// Handler handles HTTP requests for message gateway
type Handler struct {
	validator   *Validator
	rateLimiter RateLimiter
	buffer      MessageBuffer
}

// NewHandler creates a new gateway handler
func NewHandler(rateLimiter RateLimiter, buffer MessageBuffer) *Handler {
	return &Handler{
		validator:   NewValidator(),
		rateLimiter: rateLimiter,
		buffer:      buffer,
	}
}

// ReceiveMessage handles POST /api/messages endpoint
// Validates request and returns 400 for invalid requests
// Optimized for fast response time (< 10ms target)
func (h *Handler) ReceiveMessage(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	
	// Only accept POST method
	if r.Method != http.MethodPost {
		h.sendError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Only POST method is allowed", nil)
		return
	}

	// Parse request body
	var req MessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON format", nil)
		return
	}

	// Validate request
	validationErrors := h.validator.ValidateMessageRequest(req)
	if len(validationErrors) > 0 {
		h.sendError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Request validation failed", validationErrors)
		return
	}

	// Check rate limit
	allowed, rateLimitInfo, err := h.rateLimiter.Allow(r.Context(), req.TenantID)
	if err != nil {
		h.sendError(w, http.StatusInternalServerError, "RATE_LIMIT_ERROR", "Failed to check rate limit", nil)
		return
	}

	// Add rate limit headers (pre-set before buffer push for faster response)
	for key, value := range rateLimitInfo.ToHeaders() {
		w.Header().Set(key, value)
	}

	// Reject if rate limit exceeded
	if !allowed {
		h.sendError(w, http.StatusTooManyRequests, "RATE_LIMIT_EXCEEDED", "Too many requests. Please try again later.", nil)
		return
	}

	// Create buffered message
	msg := NewBufferedMessage(req)

	// Push to Redis buffer with context
	// This is the critical path - buffer.Push is optimized for < 5ms
	if err := h.buffer.Push(r.Context(), req.TenantID, msg); err != nil {
		h.sendError(w, http.StatusInternalServerError, "BUFFER_ERROR", "Failed to queue message", nil)
		return
	}

	// Record metrics
	tenantIDStr := req.TenantID.String()
	priority := req.Priority
	if priority == "" {
		priority = "NORMAL"
	}
	metrics.RecordMessageReceived(tenantIDStr, priority)
	
	// Record API latency
	duration := time.Since(start).Seconds()
	metrics.ObserveMessageLatency("api_gateway", duration)

	// Send success response immediately after buffer push
	// Pre-construct response to minimize processing time
	response := MessageResponse{
		MessageID: msg.MessageID,
		Status:    "QUEUED",
		QueuedAt:  msg.QueuedAt,
	}

	h.sendJSON(w, http.StatusOK, response)
}

// ReceiveBulk handles POST /api/messages/bulk endpoint
func (h *Handler) ReceiveBulk(w http.ResponseWriter, r *http.Request) {
	// Only accept POST method
	if r.Method != http.MethodPost {
		h.sendError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Only POST method is allowed", nil)
		return
	}

	// Parse request body
	var req BulkMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON format", nil)
		return
	}

	// Validate bulk request
	validationErrors := h.validator.ValidateBulkRequest(req)
	if len(validationErrors) > 0 {
		h.sendError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Request validation failed", validationErrors)
		return
	}

	// Check rate limit (use first message's tenant_id)
	if len(req.Messages) > 0 {
		allowed, rateLimitInfo, err := h.rateLimiter.Allow(r.Context(), req.Messages[0].TenantID)
		if err != nil {
			h.sendError(w, http.StatusInternalServerError, "RATE_LIMIT_ERROR", "Failed to check rate limit", nil)
			return
		}

		// Add rate limit headers
		for key, value := range rateLimitInfo.ToHeaders() {
			w.Header().Set(key, value)
		}

		// Reject if rate limit exceeded
		if !allowed {
			h.sendError(w, http.StatusTooManyRequests, "RATE_LIMIT_EXCEEDED", "Too many requests. Please try again later.", nil)
			return
		}
	}

	// Process and buffer all messages
	acceptedCount := 0
	rejectedCount := 0
	
	for _, msgReq := range req.Messages {
		// Create buffered message
		msg := NewBufferedMessage(msgReq)
		
		// Push to Redis buffer
		if err := h.buffer.Push(r.Context(), msgReq.TenantID, msg); err != nil {
			rejectedCount++
			continue
		}
		
		acceptedCount++
	}

	response := BulkResponse{
		BatchID:       "batch_" + time.Now().Format("20060102150405"),
		Status:        "PROCESSING",
		TotalCount:    len(req.Messages),
		AcceptedCount: acceptedCount,
		RejectedCount: rejectedCount,
		QueuedAt:      time.Now(),
	}

	h.sendJSON(w, http.StatusOK, response)
}

// sendJSON sends JSON response
func (h *Handler) sendJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

// sendError sends error response
func (h *Handler) sendError(w http.ResponseWriter, statusCode int, errorCode string, message string, validationErrors []ValidationError) {
	response := ErrorResponse{
		Error:   errorCode,
		Message: message,
		Errors:  validationErrors,
	}

	h.sendJSON(w, statusCode, response)
}

// RegisterRoutes registers HTTP routes
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/messages", h.ReceiveMessage)
	mux.HandleFunc("/api/messages/bulk", h.ReceiveBulk)
}
