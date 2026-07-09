package notifications

import (
	"encoding/json"
	"net/http"

	"github.com/al-hudhud/backend/internal/models"
	"github.com/al-hudhud/backend/pkg/logger"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Get("/notifications", h.GetNotifications)
	r.Get("/notifications/unread-count", h.GetUnreadCount)
	r.Post("/notifications/mark-read", h.MarkAsRead)
	r.Post("/notifications/mark-all-read", h.MarkAllAsRead)
	r.Get("/notifications/stats", h.GetStats)
}

func getTenantID(r *http.Request) (uuid.UUID, bool) {
	idStr, _ := r.Context().Value("tenant_id").(string)
	if idStr == "" {
		return uuid.Nil, false
	}
	id, err := uuid.Parse(idStr)
	return id, err == nil
}

func getUserID(r *http.Request) (uuid.UUID, bool) {
	idStr, _ := r.Context().Value("user_id").(string)
	if idStr == "" {
		return uuid.Nil, false
	}
	id, err := uuid.Parse(idStr)
	return id, err == nil
}

func (h *Handler) GetNotifications(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantID, ok := getTenantID(r)
	if !ok {
		http.Error(w, "tenant not found", http.StatusUnauthorized)
		return
	}

	userID, ok := getUserID(r)
	if !ok {
		http.Error(w, "user not found", http.StatusUnauthorized)
		return
	}

	limit := 50
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsedLimit, err := parseInt(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	notifications, err := h.service.GetByTenantAndUser(ctx, tenantID, userID, limit)
	if err != nil {
		logger.GetLogger().Error("Failed to get notifications", zap.Error(err))
		http.Error(w, "failed to get notifications", http.StatusInternalServerError)
		return
	}

	response := make([]map[string]interface{}, len(notifications))
	for i, n := range notifications {
		response[i] = map[string]interface{}{
			"id":         n.ID.String(),
			"tenant_id":  n.TenantID.String(),
			"type":       n.Type,
			"title":      n.Title,
			"body":       n.Body,
			"data":       n.Data,
			"read":       n.ReadAt != nil,
			"read_at":    n.ReadAt,
			"created_at": n.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *Handler) GetUnreadCount(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantID, ok := getTenantID(r)
	if !ok {
		http.Error(w, "tenant not found", http.StatusUnauthorized)
		return
	}

	userID, ok := getUserID(r)
	if !ok {
		http.Error(w, "user not found", http.StatusUnauthorized)
		return
	}

	count, err := h.service.GetUnreadCount(ctx, tenantID, userID)
	if err != nil {
		logger.GetLogger().Error("Failed to get unread count", zap.Error(err))
		http.Error(w, "failed to get unread count", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"unread_count": count,
	})
}

func (h *Handler) MarkAsRead(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantID, ok := getTenantID(r)
	if !ok {
		http.Error(w, "tenant not found", http.StatusUnauthorized)
		return
	}

	var req struct {
		NotificationID string `json:"notification_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	notificationID, err := uuid.Parse(req.NotificationID)
	if err != nil {
		http.Error(w, "invalid notification_id format", http.StatusBadRequest)
		return
	}

	success, err := h.service.MarkAsRead(ctx, notificationID, tenantID)
	if err != nil {
		logger.GetLogger().Error("Failed to mark notification as read", zap.Error(err))
		http.Error(w, "failed to mark as read", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": success,
	})
}

func (h *Handler) MarkAllAsRead(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantID, ok := getTenantID(r)
	if !ok {
		http.Error(w, "tenant not found", http.StatusUnauthorized)
		return
	}

	userID, ok := getUserID(r)
	if !ok {
		http.Error(w, "user not found", http.StatusUnauthorized)
		return
	}

	count, err := h.service.MarkAllAsRead(ctx, tenantID, userID)
	if err != nil {
		logger.GetLogger().Error("Failed to mark all as read", zap.Error(err))
		http.Error(w, "failed to mark all as read", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"marked_count": count,
	})
}

func (h *Handler) GetStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantID, ok := getTenantID(r)
	if !ok {
		http.Error(w, "tenant not found", http.StatusUnauthorized)
		return
	}

	stats, err := h.service.GetStats(ctx, tenantID)
	if err != nil {
		logger.GetLogger().Error("Failed to get notification stats", zap.Error(err))
		http.Error(w, "failed to get stats", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func parseInt(s string) (int, error) {
	var result int
	err := json.Unmarshal([]byte(s), &result)
	return result, err
}

func init() {
	_ = models.NotificationStats{}
}
