package api

import (
	"net/http"

	"github.com/chimera-pool/chimera-pool-core/internal/auth"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// =============================================================================
// NETWORK CONFIGURATION API HANDLERS
// Admin endpoints for managing mining network configurations
// =============================================================================

// NetworkConfigHandlers handles network configuration API endpoints
type NetworkConfigHandlers struct {
	service *DBNetworkConfigService
}

// NewNetworkConfigHandlers creates new network config handlers
func NewNetworkConfigHandlers(service *DBNetworkConfigService) *NetworkConfigHandlers {
	return &NetworkConfigHandlers{service: service}
}

// -----------------------------------------------------------------------------
// Public Endpoints
// -----------------------------------------------------------------------------

// GetActiveNetwork returns the currently active network configuration
// GET /api/v1/network/active
func (h *NetworkConfigHandlers) GetActiveNetwork(c *gin.Context) {
	network, err := h.service.GetActiveNetwork(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No active network configured"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"network": network})
}

// ListNetworksPublic returns public network information (no sensitive data)
// GET /api/v1/networks
func (h *NetworkConfigHandlers) ListNetworksPublic(c *gin.Context) {
	networks, err := h.service.ListActiveNetworks(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch networks"})
		return
	}

	// Strip sensitive fields for public view
	publicNetworks := make([]map[string]interface{}, len(networks))
	for i, n := range networks {
		publicNetworks[i] = map[string]interface{}{
			"name":         n.Name,
			"symbol":       n.Symbol,
			"display_name": n.DisplayName,
			"algorithm":    n.Algorithm,
			"stratum_port": n.StratumPort,
			"is_default":   n.IsDefault,
			"explorer_url": n.ExplorerURL,
			"pool_fee":     n.PoolFeePercent,
		}
	}

	c.JSON(http.StatusOK, gin.H{"networks": publicNetworks})
}

// -----------------------------------------------------------------------------
// Admin Endpoints (require admin role)
// -----------------------------------------------------------------------------

// ListNetworks returns all network configurations (admin only)
// GET /api/v1/admin/networks
func (h *NetworkConfigHandlers) ListNetworks(c *gin.Context) {
	networks, err := h.service.ListNetworks(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch networks", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"networks": networks, "total": len(networks)})
}

// GetNetwork returns a specific network configuration
// GET /api/v1/admin/networks/:id
func (h *NetworkConfigHandlers) GetNetwork(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid network ID"})
		return
	}

	network, err := h.service.GetNetworkByID(c.Request.Context(), id)
	if err != nil {
		if err == ErrNetworkNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Network not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch network"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"network": network})
}

// CreateNetwork creates a new network configuration
// POST /api/v1/admin/networks
func (h *NetworkConfigHandlers) CreateNetwork(c *gin.Context) {
	var req CreateNetworkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "details": err.Error()})
		return
	}

	// Validate required fields
	if req.Name == "" || req.Symbol == "" || req.RPCURL == "" || req.PoolWalletAddress == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing required fields: name, symbol, rpc_url, pool_wallet_address"})
		return
	}

	network, err := h.service.CreateNetwork(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create network", "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"network": network, "message": "Network created successfully"})
}

// UpdateNetwork updates an existing network configuration
// PUT /api/v1/admin/networks/:id
func (h *NetworkConfigHandlers) UpdateNetwork(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid network ID"})
		return
	}

	var req UpdateNetworkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "details": err.Error()})
		return
	}

	network, err := h.service.UpdateNetwork(c.Request.Context(), id, &req)
	if err != nil {
		if err == ErrNetworkNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Network not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update network", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"network": network, "message": "Network updated successfully"})
}

// DeleteNetwork deletes a network configuration
// DELETE /api/v1/admin/networks/:id
func (h *NetworkConfigHandlers) DeleteNetwork(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid network ID"})
		return
	}

	err = h.service.DeleteNetwork(c.Request.Context(), id)
	if err != nil {
		if err == ErrNetworkNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Network not found"})
			return
		}
		if err == ErrCannotDeactivateDefault {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot delete the default network"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete network", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Network deleted successfully"})
}

// SwitchNetwork switches the active mining network
// POST /api/v1/admin/networks/switch
func (h *NetworkConfigHandlers) SwitchNetwork(c *gin.Context) {
	var req SwitchNetworkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "details": err.Error()})
		return
	}

	// Get the user who initiated the switch
	user, exists := c.Get("user")
	var userID int64 = 0
	if exists {
		if authUser, ok := user.(*auth.User); ok {
			userID = authUser.ID
		}
	}

	history, err := h.service.SwitchNetwork(c.Request.Context(), req.NetworkName, userID, req.Reason)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to switch network", "details": err.Error()})
		return
	}

	// Get the new active network
	activeNetwork, _ := h.service.GetActiveNetwork(c.Request.Context())

	c.JSON(http.StatusOK, gin.H{
		"message":        "Network switched successfully",
		"switch_history": history,
		"active_network": activeNetwork,
	})
}

// GetSwitchHistory returns network switch history
// GET /api/v1/admin/networks/history
func (h *NetworkConfigHandlers) GetSwitchHistory(c *gin.Context) {
	history, err := h.service.GetSwitchHistory(c.Request.Context(), 50)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch history"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"history": history})
}

// RollbackSwitch rolls back to the previous network
// POST /api/v1/admin/networks/rollback/:historyId
func (h *NetworkConfigHandlers) RollbackSwitch(c *gin.Context) {
	historyIDStr := c.Param("historyId")
	historyID, err := uuid.Parse(historyIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid history ID"})
		return
	}

	err = h.service.RollbackSwitch(c.Request.Context(), historyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to rollback", "details": err.Error()})
		return
	}

	activeNetwork, _ := h.service.GetActiveNetwork(c.Request.Context())

	c.JSON(http.StatusOK, gin.H{
		"message":        "Rollback successful",
		"active_network": activeNetwork,
	})
}

// TestNetworkConnection tests the RPC connection for a network
// POST /api/v1/admin/networks/:id/test
func (h *NetworkConfigHandlers) TestNetworkConnection(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid network ID"})
		return
	}

	success, err := h.service.TestConnection(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": success,
		"message": "Connection test passed",
	})
}

// -----------------------------------------------------------------------------
// Route Registration
// -----------------------------------------------------------------------------

// RegisterNetworkRoutes registers all network configuration routes
func (h *NetworkConfigHandlers) RegisterNetworkRoutes(router *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	// Public routes
	router.GET("/network/active", h.GetActiveNetwork)
	router.GET("/networks", h.ListNetworksPublic)

	// Admin routes (require authentication and admin role)
	admin := router.Group("/admin/networks")
	admin.Use(authMiddleware)
	admin.Use(RequireRole(auth.RoleAdmin))
	{
		admin.GET("", h.ListNetworks)
		admin.GET("/:id", h.GetNetwork)
		admin.POST("", h.CreateNetwork)
		admin.PUT("/:id", h.UpdateNetwork)
		admin.DELETE("/:id", h.DeleteNetwork)
		admin.POST("/switch", h.SwitchNetwork)
		admin.GET("/history", h.GetSwitchHistory)
		admin.POST("/rollback/:historyId", h.RollbackSwitch)
		admin.POST("/:id/test", h.TestNetworkConnection)
	}
}
