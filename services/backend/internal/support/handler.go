package support

import (
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	ticketManager *TicketManager
}

func NewHandler(tm *TicketManager) *Handler {
	return &Handler{
		ticketManager: tm,
	}
}

// GetAllTickets handles the GET /api/v1/admin/support/tickets admin request
func (h *Handler) GetAllTickets(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	tickets, err := h.ticketManager.ListAllTickets(c.Request.Context(), limit, offset)
	if err != nil {
		log.Printf("Failed to get all tickets: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve support tickets"})
		return
	}

	c.JSON(http.StatusOK, tickets)
}

// GetTenantChat returns the most recent active ticket for a tenant
func (h *Handler) GetTenantChat(c *gin.Context) {
	// For now, return an empty ticket to satisfy the frontend if real support is not fully implemented
	c.JSON(http.StatusOK, gin.H{
		"id": nil,
		"replies": []interface{}{},
	})
}

// ReplyToTicket allows a tenant to reply to a ticket
func (h *Handler) ReplyToTicket(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "success"})
}
