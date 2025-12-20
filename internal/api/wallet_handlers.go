package api

import (
	"net/http"
	"strconv"

	"chimera-pool/internal/payouts"

	"github.com/gin-gonic/gin"
)

// WalletHandlers handles wallet-related API endpoints
type WalletHandlers struct {
	walletService *payouts.WalletService
}

// NewWalletHandlers creates a new wallet handlers instance
func NewWalletHandlers(walletService *payouts.WalletService) *WalletHandlers {
	return &WalletHandlers{walletService: walletService}
}

// GetWallets returns all wallets for the authenticated user
func (h *WalletHandlers) GetWallets(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	wallets, err := h.walletService.GetUserWallets(userID.(int64))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get wallets"})
		return
	}

	summary, err := h.walletService.GetWalletSummary(userID.(int64))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get wallet summary"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"wallets": wallets,
		"summary": summary,
	})
}

// CreateWallet creates a new wallet for the authenticated user
func (h *WalletHandlers) CreateWallet(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req payouts.CreateWalletRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	wallet, err := h.walletService.CreateWallet(userID.(int64), req)
	if err != nil {
		status := http.StatusInternalServerError
		if err == payouts.ErrWalletPercentageOver || err == payouts.ErrInvalidPercentage {
			status = http.StatusBadRequest
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"wallet":  wallet,
		"message": "Wallet created successfully",
	})
}

// UpdateWallet updates an existing wallet
func (h *WalletHandlers) UpdateWallet(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	walletID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid wallet ID"})
		return
	}

	var req payouts.UpdateWalletRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	wallet, err := h.walletService.UpdateWallet(userID.(int64), walletID, req)
	if err != nil {
		status := http.StatusInternalServerError
		if err == payouts.ErrWalletNotFound {
			status = http.StatusNotFound
		} else if err == payouts.ErrWalletPercentageOver || err == payouts.ErrInvalidPercentage {
			status = http.StatusBadRequest
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"wallet":  wallet,
		"message": "Wallet updated successfully",
	})
}

// DeleteWallet removes a wallet
func (h *WalletHandlers) DeleteWallet(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	walletID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid wallet ID"})
		return
	}

	if err := h.walletService.DeleteWallet(userID.(int64), walletID); err != nil {
		status := http.StatusInternalServerError
		if err == payouts.ErrWalletNotFound {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Wallet deleted successfully"})
}

// GetPayoutPreview calculates how a payout would be split across wallets
func (h *WalletHandlers) GetPayoutPreview(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	amountStr := c.Query("amount")
	if amountStr == "" {
		amountStr = "1000000000" // Default 10 BDAG (assuming 8 decimals)
	}

	amount, err := strconv.ParseInt(amountStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid amount"})
		return
	}

	splits, err := h.walletService.CalculateSplitPayouts(userID.(int64), amount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to calculate splits"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"total_amount": amount,
		"splits":       splits,
	})
}
