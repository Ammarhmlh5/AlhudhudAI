package worker

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

// DLQHandler handles HTTP requests for Dead Letter Queue operations
// Implements Requirement 17.6: توفير API لإعادة معالجة رسائل DLQ
type DLQHandler struct {
	dlq    *DeadLetterQueue
	logger *zap.Logger
}

// NewDLQHandler creates a new DLQ handler
func NewDLQHandler(dlq *DeadLetterQueue, logger *zap.Logger) *DLQHandler {
	return &DLQHandler{
		dlq:    dlq,
		logger: logger,
	}
}

// RegisterRoutes registers DLQ routes
func (h *DLQHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/api/dlq/{tenant_id}", h.GetDLQMessages).Methods("GET")
	router.HandleFunc("/api/dlq/{tenant_id}/reprocess/{dlq_id}", h.ReprocessMessage).Methods("POST")
	router.HandleFunc("/api/dlq/{tenant_id}/reprocess-all", h.ReprocessAllMessages).Methods("POST")
	router.HandleFunc("/api/dlq/cleanup", h.CleanupOldMessages).Methods("POST")
}

// GetDLQMessages retrieves dead letter queue messages for a tenant
// GET /api/dlq/{tenant_id}?limit=100
func (h *DLQHandler) GetDLQMessages(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	tenantIDStr := vars["tenant_id"]

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid tenant ID")
		return
	}

	// Get limit from query params (default 100)
	limitStr := r.URL.Query().Get("limit")
	limit := 100
	if limitStr != "" {
		parsedLimit, err := strconv.Atoi(limitStr)
		if err == nil && parsedLimit > 0 && parsedLimit <= 1000 {
			limit = parsedLimit
		}
	}

	messages, err := h.dlq.GetDeadLetterMessages(r.Context(), tenantID, limit)
	if err != nil {
		h.logger.Error("Failed to get DLQ messages",
			zap.Error(err),
			zap.String("tenant_id", tenantID.String()),
		)
		h.respondError(w, http.StatusInternalServerError, "Failed to retrieve messages")
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"tenant_id": tenantID.String(),
		"count":     len(messages),
		"messages":  messages,
	})
}

// ReprocessMessage reprocesses a single message from DLQ
// POST /api/dlq/{tenant_id}/reprocess/{dlq_id}
func (h *DLQHandler) ReprocessMessage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	tenantIDStr := vars["tenant_id"]
	dlqIDStr := vars["dlq_id"]

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid tenant ID")
		return
	}

	dlqID, err := strconv.ParseInt(dlqIDStr, 10, 64)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid DLQ ID")
		return
	}

	err = h.dlq.ReprocessMessage(r.Context(), tenantID, dlqID)
	if err != nil {
		h.logger.Error("Failed to reprocess message",
			zap.Error(err),
			zap.String("tenant_id", tenantID.String()),
			zap.Int64("dlq_id", dlqID),
		)
		h.respondError(w, http.StatusInternalServerError, "Failed to reprocess message")
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"success":   true,
		"message":   "Message reprocessed successfully",
		"tenant_id": tenantID.String(),
		"dlq_id":    dlqID,
	})
}

// ReprocessAllMessages reprocesses all messages from DLQ for a tenant
// POST /api/dlq/{tenant_id}/reprocess-all
func (h *DLQHandler) ReprocessAllMessages(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	tenantIDStr := vars["tenant_id"]

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid tenant ID")
		return
	}

	count, err := h.dlq.ReprocessAllMessages(r.Context(), tenantID)
	if err != nil {
		h.logger.Error("Failed to reprocess all messages",
			zap.Error(err),
			zap.String("tenant_id", tenantID.String()),
		)
		h.respondError(w, http.StatusInternalServerError, "Failed to reprocess messages")
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"success":   true,
		"message":   "All messages reprocessed successfully",
		"tenant_id": tenantID.String(),
		"count":     count,
	})
}

// CleanupOldMessages removes messages older than retention period
// POST /api/dlq/cleanup
// Body: {"retention_days": 7}
func (h *DLQHandler) CleanupOldMessages(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RetentionDays int `json:"retention_days"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Default to 7 days if not specified
	if req.RetentionDays <= 0 {
		req.RetentionDays = 7
	}

	count, err := h.dlq.CleanupOldMessages(r.Context(), req.RetentionDays)
	if err != nil {
		h.logger.Error("Failed to cleanup old messages",
			zap.Error(err),
			zap.Int("retention_days", req.RetentionDays),
		)
		h.respondError(w, http.StatusInternalServerError, "Failed to cleanup messages")
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"success":        true,
		"message":        "Old messages cleaned up successfully",
		"retention_days": req.RetentionDays,
		"deleted_count":  count,
	})
}

// respondJSON sends a JSON response
func (h *DLQHandler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// respondError sends an error response
func (h *DLQHandler) respondError(w http.ResponseWriter, status int, message string) {
	h.respondJSON(w, status, map[string]interface{}{
		"error":   true,
		"message": message,
	})
}
