package notifications

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/al-hudhud/backend/pkg/logger"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type SettingsHandler struct {
	service *SettingsService
}

func NewSettingsHandler(service *SettingsService) *SettingsHandler {
	return &SettingsHandler{service: service}
}

func (h *SettingsHandler) RegisterRoutes(r chi.Router) {
	r.Get("/admin/notifications/settings", h.GetSettings)
	r.Put("/admin/notifications/settings/{type}", h.UpdateSetting)
	r.Get("/admin/notifications/templates", h.GetTemplates)
	r.Put("/admin/notifications/templates/{type}", h.UpdateTemplate)
	r.Get("/admin/notifications/logs", h.GetLogs)
	r.Get("/admin/notifications/stats", h.GetStats)

	r.Get("/notifications/preferences", h.GetUserPreferences)
	r.Put("/notifications/preferences", h.UpdateUserPreferences)
}

func getTenantIDFromCtx(r *http.Request) (uuid.UUID, bool) {
	idStr, _ := r.Context().Value("tenant_id").(string)
	if idStr == "" {
		return uuid.Nil, false
	}
	id, err := uuid.Parse(idStr)
	return id, err == nil
}

func (h *SettingsHandler) GetSettings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantID, ok := getTenantIDFromCtx(r)
	if !ok {
		http.Error(w, "tenant not found", http.StatusUnauthorized)
		return
	}

	settings, err := h.service.GetSettings(ctx, tenantID)
	if err != nil {
		logger.GetLogger().Error("Failed to get notification settings", zap.Error(err))
		http.Error(w, "failed to get settings", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(settings)
}

func (h *SettingsHandler) UpdateSetting(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantID, ok := getTenantIDFromCtx(r)
	if !ok {
		http.Error(w, "tenant not found", http.StatusUnauthorized)
		return
	}

	notificationType := chi.URLParam(r, "type")
	if notificationType == "" {
		http.Error(w, "notification type required", http.StatusBadRequest)
		return
	}

	var req struct {
		IsEnabled bool            `json:"is_enabled"`
		Config    json.RawMessage `json:"config"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	setting, err := h.service.UpdateSetting(ctx, tenantID, notificationType, req.IsEnabled, req.Config)
	if err != nil {
		logger.GetLogger().Error("Failed to update notification setting", zap.Error(err))
		http.Error(w, "failed to update setting", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(setting)
}

func (h *SettingsHandler) GetTemplates(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantID, ok := getTenantIDFromCtx(r)
	if !ok {
		http.Error(w, "tenant not found", http.StatusUnauthorized)
		return
	}

	templates, err := h.service.GetTemplates(ctx, &tenantID)
	if err != nil {
		logger.GetLogger().Error("Failed to get notification templates", zap.Error(err))
		http.Error(w, "failed to get templates", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(templates)
}

func (h *SettingsHandler) UpdateTemplate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantID, ok := getTenantIDFromCtx(r)
	if !ok {
		http.Error(w, "tenant not found", http.StatusUnauthorized)
		return
	}

	notificationType := chi.URLParam(r, "type")
	if notificationType == "" {
		http.Error(w, "notification type required", http.StatusBadRequest)
		return
	}

	var req struct {
		TitleTemplate string `json:"title_template"`
		BodyTemplate  string `json:"body_template"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	template, err := h.service.UpdateTemplate(ctx, &tenantID, notificationType, req.TitleTemplate, req.BodyTemplate)
	if err != nil {
		logger.GetLogger().Error("Failed to update notification template", zap.Error(err))
		http.Error(w, "failed to update template", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(template)
}

func (h *SettingsHandler) GetLogs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantID, ok := getTenantIDFromCtx(r)
	if !ok {
		http.Error(w, "tenant not found", http.StatusUnauthorized)
		return
	}

	limit := 50
	offset := 0

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if parsedOffset, err := strconv.Atoi(offsetStr); err == nil && parsedOffset > 0 {
			offset = parsedOffset
		}
	}

	logs, err := h.service.GetNotificationLogs(ctx, tenantID, limit, offset)
	if err != nil {
		logger.GetLogger().Error("Failed to get notification logs", zap.Error(err))
		http.Error(w, "failed to get logs", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logs)
}

func (h *SettingsHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantID, ok := getTenantIDFromCtx(r)
	if !ok {
		http.Error(w, "tenant not found", http.StatusUnauthorized)
		return
	}

	stats, err := h.service.GetNotificationStats(ctx, tenantID)
	if err != nil {
		logger.GetLogger().Error("Failed to get notification stats", zap.Error(err))
		http.Error(w, "failed to get stats", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (h *SettingsHandler) GetUserPreferences(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantID, ok := getTenantIDFromCtx(r)
	if !ok {
		http.Error(w, "tenant not found", http.StatusUnauthorized)
		return
	}

	settings, err := h.service.GetSettings(ctx, tenantID)
	if err != nil {
		logger.GetLogger().Error("Failed to get user preferences", zap.Error(err))
		http.Error(w, "failed to get preferences", http.StatusInternalServerError)
		return
	}

	preferences := make(map[string]interface{})
	for _, setting := range settings {
		preferences[setting.NotificationType] = map[string]interface{}{
			"enabled": setting.IsEnabled,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(preferences)
}

func (h *SettingsHandler) UpdateUserPreferences(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantID, ok := getTenantIDFromCtx(r)
	if !ok {
		http.Error(w, "tenant not found", http.StatusUnauthorized)
		return
	}

	var preferences map[string]struct {
		Enabled bool `json:"enabled"`
	}

	if err := json.NewDecoder(r.Body).Decode(&preferences); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	for notificationType, pref := range preferences {
		_, err := h.service.UpdateSetting(ctx, tenantID, notificationType, pref.Enabled, nil)
		if err != nil {
			logger.GetLogger().Error("Failed to update preference",
				zap.String("notification_type", notificationType),
				zap.Error(err),
			)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Preferences updated successfully",
	})
}
