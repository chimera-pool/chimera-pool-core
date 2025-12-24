package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// MinerMonitoringHandlers handles admin miner monitoring endpoints
type MinerMonitoringHandlers struct {
	service MinerMonitoringService
}

// NewMinerMonitoringHandlers creates new miner monitoring handlers
func NewMinerMonitoringHandlers(service MinerMonitoringService) *MinerMonitoringHandlers {
	return &MinerMonitoringHandlers{service: service}
}

// GetUserMiners handles GET /api/v1/admin/users/:id/miners
// Returns all miners for a specific user with summary stats
func (h *MinerMonitoringHandlers) GetUserMiners(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	summary, err := h.service.GetUserMinerSummary(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user miners", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, summary)
}

// GetMinerDetail handles GET /api/v1/admin/miners/:id
// Returns comprehensive details for a specific miner
func (h *MinerMonitoringHandlers) GetMinerDetail(c *gin.Context) {
	minerIDStr := c.Param("id")
	minerID, err := strconv.ParseInt(minerIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid miner ID"})
		return
	}

	detail, err := h.service.GetMinerDetail(minerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch miner details", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, detail)
}

// GetAllMiners handles GET /api/v1/admin/miners
// Returns paginated list of all miners for admin monitoring
func (h *MinerMonitoringHandlers) GetAllMiners(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	search := c.Query("search")
	activeOnly := c.Query("active") == "true"

	miners, total, err := h.service.GetAllMinersForAdmin(page, limit, search, activeOnly)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch miners", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"miners": miners,
		"total":  total,
		"page":   page,
		"limit":  limit,
	})
}

// GetMinerShares handles GET /api/v1/admin/miners/:id/shares
// Returns share history for a specific miner
func (h *MinerMonitoringHandlers) GetMinerShares(c *gin.Context) {
	minerIDStr := c.Param("id")
	minerID, err := strconv.ParseInt(minerIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid miner ID"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))

	shares, err := h.service.GetMinerShareHistory(minerID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch shares", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"shares": shares})
}

// RegisterMinerMonitoringRoutes registers miner monitoring routes
func (h *MinerMonitoringHandlers) RegisterRoutes(router *gin.RouterGroup) {
	// Admin routes for miner monitoring
	admin := router.Group("/admin")
	{
		admin.GET("/miners", h.GetAllMiners)
		admin.GET("/miners/:id", h.GetMinerDetail)
		admin.GET("/miners/:id/shares", h.GetMinerShares)
		admin.GET("/users/:id/miners", h.GetUserMiners)
	}
}
