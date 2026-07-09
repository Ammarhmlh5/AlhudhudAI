package contact

import (
	"net/http"

	"github.com/al-hudhud/backend/internal/auth"
	"github.com/al-hudhud/backend/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) ListContacts(c *gin.Context) {
	tenantID, ok := auth.GetTenantIDOrAbort(c)
	if !ok {
		return
	}

	contacts, err := h.service.ListContacts(c, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, contacts)
}

func (h *Handler) CreateContact(c *gin.Context) {
	tenantID, ok := auth.GetTenantIDOrAbort(c)
	if !ok {
		return
	}

	var contact models.Contact
	if err := c.ShouldBindJSON(&contact); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.CreateContact(c, tenantID, &contact); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, contact)
}

func (h *Handler) UpdateContact(c *gin.Context) {
	tenantID, ok := auth.GetTenantIDOrAbort(c)
	if !ok {
		return
	}

	var contact models.Contact
	if err := c.ShouldBindJSON(&contact); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.UpdateContact(c, tenantID, &contact); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, contact)
}

func (h *Handler) DeleteContact(c *gin.Context) {
	tenantID, ok := auth.GetTenantIDOrAbort(c)
	if !ok {
		return
	}

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid contact ID"})
		return
	}

	if err := h.service.DeleteContact(c, tenantID, id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Contact deleted"})
}
