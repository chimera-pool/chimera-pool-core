package payouts

import (
	"context"
	"time"
)

// =============================================================================
// ISP-COMPLIANT PAYOUT INTERFACES
// Each interface is small and focused on a single responsibility
// =============================================================================

// -----------------------------------------------------------------------------
// Payout Mode Enum
// -----------------------------------------------------------------------------

// PayoutMode represents the payout calculation method
type PayoutMode string

const (
	// PayoutModePPLNS - Pay Per Last N Shares
	// Miners share block variance risk. Rewards based on shares in sliding window.
	// Best for: Loyal miners who mine consistently
	PayoutModePPLNS PayoutMode = "pplns"

	// PayoutModePPS - Pay Per Share
	// Pool absorbs all variance. Fixed payment per share regardless of blocks found.
	// Best for: Risk-averse miners wanting predictable income
	PayoutModePPS PayoutMode = "pps"

	// PayoutModePPSPlus - Pay Per Share Plus
	// PPS for block reward + PPLNS for transaction fees
	// Best for: Balance of stability and upside from tx fees
	PayoutModePPSPlus PayoutMode = "pps_plus"

	// PayoutModeFPPS - Full Pay Per Share
	// Pool pays expected block reward + expected transaction fees per share
	// Best for: Maximum predictability (highest pool risk)
	PayoutModeFPPS PayoutMode = "fpps"

	// PayoutModeSCORE - Score-based (time-weighted PPLNS)
	// Shares weighted by time - older shares worth less
	// Best for: Discouraging pool hopping
	PayoutModeSCORE PayoutMode = "score"

	// PayoutModeSOLO - Solo mining through pool
	// Miner keeps entire block reward minus pool fee
	// Best for: Large miners wanting full block rewards
	PayoutModeSOLO PayoutMode = "solo"

	// PayoutModeSLICE - Stratum V2 Job Declaration Enhanced PPLNS
	// Custom V2-enhanced payout built on Job Declaration for sliced PPLNS
	// Features: auditable shares, disconnect-tolerant windows, demand response hooks
	// Best for: V2 miners with job negotiation, advanced miners wanting transparency
	PayoutModeSLICE PayoutMode = "slice"
)

// AllPayoutModes returns all supported payout modes
func AllPayoutModes() []PayoutMode {
	return []PayoutMode{
		PayoutModePPLNS,
		PayoutModePPS,
		PayoutModePPSPlus,
		PayoutModeFPPS,
		PayoutModeSCORE,
		PayoutModeSOLO,
		PayoutModeSLICE,
	}
}

// IsValid checks if the payout mode is valid
func (m PayoutMode) IsValid() bool {
	switch m {
	case PayoutModePPLNS, PayoutModePPS, PayoutModePPSPlus,
		PayoutModeFPPS, PayoutModeSCORE, PayoutModeSOLO, PayoutModeSLICE:
		return true
	default:
		return false
	}
}

// String returns the string representation
func (m PayoutMode) String() string {
	return string(m)
}

// Description returns a human-readable description
func (m PayoutMode) Description() string {
	switch m {
	case PayoutModePPLNS:
		return "Pay Per Last N Shares - Rewards based on recent share contribution"
	case PayoutModePPS:
		return "Pay Per Share - Fixed payment per share, pool absorbs variance"
	case PayoutModePPSPlus:
		return "PPS+ - PPS for block reward, PPLNS for transaction fees"
	case PayoutModeFPPS:
		return "Full PPS - Fixed payment including expected transaction fees"
	case PayoutModeSCORE:
		return "Score-based - Time-weighted shares discourage pool hopping"
	case PayoutModeSOLO:
		return "Solo Mining - Keep entire block reward minus pool fee"
	case PayoutModeSLICE:
		return "SLICE - V2 Job Declaration enhanced PPLNS with auditable, disconnect-tolerant shares"
	default:
		return "Unknown payout mode"
	}
}

// DefaultFeePercent returns the default pool fee for this mode
func (m PayoutMode) DefaultFeePercent() float64 {
	switch m {
	case PayoutModePPLNS:
		return 1.0 // Low fee - miners share variance
	case PayoutModePPS:
		return 2.0 // Higher fee - pool absorbs variance
	case PayoutModePPSPlus:
		return 1.5 // Medium fee - split variance
	case PayoutModeFPPS:
		return 2.0 // Highest fee - pool absorbs all variance
	case PayoutModeSCORE:
		return 1.0 // Same as PPLNS
	case PayoutModeSOLO:
		return 0.5 // Lowest fee - solo mining
	case PayoutModeSLICE:
		return 0.8 // Low fee - V2 efficiency savings passed to miners
	default:
		return 1.0
	}
}

// -----------------------------------------------------------------------------
// Core Payout Calculator Interface
// -----------------------------------------------------------------------------

// PayoutCalculator defines the interface for any payout calculation strategy
type PayoutCalculator interface {
	// Mode returns the payout mode this calculator implements
	Mode() PayoutMode

	// CalculatePayouts calculates payouts for a found block
	CalculatePayouts(shares []Share, blockReward int64, txFees int64, blockTime time.Time) ([]Payout, error)

	// GetPoolFeePercent returns the pool fee percentage
	GetPoolFeePercent() float64

	// SetPoolFeePercent sets the pool fee percentage
	SetPoolFeePercent(fee float64) error

	// ValidateConfiguration validates the calculator configuration
	ValidateConfiguration() error
}

// -----------------------------------------------------------------------------
// Segregated Calculator Interfaces
// -----------------------------------------------------------------------------

// ShareWindowCalculator is for calculators that use a share window (PPLNS, SCORE)
type ShareWindowCalculator interface {
	PayoutCalculator
	GetWindowSize() int64
	SetWindowSize(size int64) error
}

// ExpectedValueCalculator is for calculators that need network difficulty (PPS, FPPS)
type ExpectedValueCalculator interface {
	PayoutCalculator
	SetNetworkDifficulty(difficulty float64)
	SetExpectedTxFees(fees int64)
}

// TimeWeightedCalculator is for calculators that weight shares by time (SCORE)
type TimeWeightedCalculator interface {
	ShareWindowCalculator
	SetDecayFactor(factor float64) error
	GetDecayFactor() float64
}

// -----------------------------------------------------------------------------
// Payout Configuration
// -----------------------------------------------------------------------------

// PayoutConfig holds configuration for payout processing
type PayoutConfig struct {
	// Default payout mode for new users
	DefaultMode PayoutMode `json:"default_mode" yaml:"default_mode"`

	// Pool fee percentages per mode (0-100)
	FeePPLNS   float64 `json:"fee_pplns" yaml:"fee_pplns"`
	FeePPS     float64 `json:"fee_pps" yaml:"fee_pps"`
	FeePPSPlus float64 `json:"fee_pps_plus" yaml:"fee_pps_plus"`
	FeeFPPS    float64 `json:"fee_fpps" yaml:"fee_fpps"`
	FeeSCORE   float64 `json:"fee_score" yaml:"fee_score"`
	FeeSOLO    float64 `json:"fee_solo" yaml:"fee_solo"`
	FeeSLICE   float64 `json:"fee_slice" yaml:"fee_slice"`

	// PPLNS window size (total difficulty)
	PPLNSWindowSize int64 `json:"pplns_window_size" yaml:"pplns_window_size"`

	// SCORE decay factor (0-1, how fast old shares lose value)
	SCOREDecayFactor float64 `json:"score_decay_factor" yaml:"score_decay_factor"`

	// SLICE configuration
	SLICEWindowSize    int64   `json:"slice_window_size" yaml:"slice_window_size"`       // Number of slices in window
	SLICESliceDuration int64   `json:"slice_slice_duration" yaml:"slice_slice_duration"` // Duration of each slice in seconds
	SLICEDecayFactor   float64 `json:"slice_decay_factor" yaml:"slice_decay_factor"`     // Time decay within slices

	// Minimum payout thresholds per coin (in smallest unit)
	MinPayoutLTC  int64 `json:"min_payout_ltc" yaml:"min_payout_ltc"`   // Litoshis
	MinPayoutBDAG int64 `json:"min_payout_bdag" yaml:"min_payout_bdag"` // BDAG smallest unit

	// Enable/disable specific modes
	EnablePPLNS   bool `json:"enable_pplns" yaml:"enable_pplns"`
	EnablePPS     bool `json:"enable_pps" yaml:"enable_pps"`
	EnablePPSPlus bool `json:"enable_pps_plus" yaml:"enable_pps_plus"`
	EnableFPPS    bool `json:"enable_fpps" yaml:"enable_fpps"`
	EnableSCORE   bool `json:"enable_score" yaml:"enable_score"`
	EnableSOLO    bool `json:"enable_solo" yaml:"enable_solo"`
	EnableSLICE   bool `json:"enable_slice" yaml:"enable_slice"`
}

// DefaultPayoutConfig returns sensible defaults
func DefaultPayoutConfig() *PayoutConfig {
	return &PayoutConfig{
		DefaultMode: PayoutModePPLNS,

		// Conservative fees that protect pool
		FeePPLNS:   1.0,
		FeePPS:     2.0,
		FeePPSPlus: 1.5,
		FeeFPPS:    2.0,
		FeeSCORE:   1.0,
		FeeSOLO:    0.5,
		FeeSLICE:   0.8, // Lower fee - V2 efficiency passed to miners

		// PPLNS: 2x network difficulty as window
		PPLNSWindowSize: 200000,

		// SCORE: 50% decay per hour
		SCOREDecayFactor: 0.5,

		// SLICE: 10 slices of 10 minutes each, 70% decay
		SLICEWindowSize:    10,
		SLICESliceDuration: 600, // 10 minutes per slice
		SLICEDecayFactor:   0.7,

		// Minimum payouts
		MinPayoutLTC:  1000000,    // 0.01 LTC (1M litoshis)
		MinPayoutBDAG: 1000000000, // 10 BDAG (configurable down to 10)

		// Enable common modes by default
		EnablePPLNS:   true,
		EnablePPS:     false, // Disabled by default (high pool risk)
		EnablePPSPlus: true,
		EnableFPPS:    false, // Disabled by default (highest pool risk)
		EnableSCORE:   true,
		EnableSOLO:    true,
		EnableSLICE:   true, // V2 enhanced - enabled for V2 miners
	}
}

// GetFeeForMode returns the fee for a specific payout mode
func (c *PayoutConfig) GetFeeForMode(mode PayoutMode) float64 {
	switch mode {
	case PayoutModePPLNS:
		return c.FeePPLNS
	case PayoutModePPS:
		return c.FeePPS
	case PayoutModePPSPlus:
		return c.FeePPSPlus
	case PayoutModeFPPS:
		return c.FeeFPPS
	case PayoutModeSCORE:
		return c.FeeSCORE
	case PayoutModeSOLO:
		return c.FeeSOLO
	case PayoutModeSLICE:
		return c.FeeSLICE
	default:
		return c.FeePPLNS
	}
}

// IsModeEnabled checks if a payout mode is enabled
func (c *PayoutConfig) IsModeEnabled(mode PayoutMode) bool {
	switch mode {
	case PayoutModePPLNS:
		return c.EnablePPLNS
	case PayoutModePPS:
		return c.EnablePPS
	case PayoutModePPSPlus:
		return c.EnablePPSPlus
	case PayoutModeFPPS:
		return c.EnableFPPS
	case PayoutModeSCORE:
		return c.EnableSCORE
	case PayoutModeSOLO:
		return c.EnableSOLO
	case PayoutModeSLICE:
		return c.EnableSLICE
	default:
		return false
	}
}

// GetEnabledModes returns all enabled payout modes
func (c *PayoutConfig) GetEnabledModes() []PayoutMode {
	modes := make([]PayoutMode, 0)
	for _, mode := range AllPayoutModes() {
		if c.IsModeEnabled(mode) {
			modes = append(modes, mode)
		}
	}
	return modes
}

// -----------------------------------------------------------------------------
// User Payout Settings
// -----------------------------------------------------------------------------

// UserPayoutSettings represents a user's payout preferences
type UserPayoutSettings struct {
	UserID           int64      `json:"user_id" db:"user_id"`
	PayoutMode       PayoutMode `json:"payout_mode" db:"payout_mode"`
	MinPayoutAmount  int64      `json:"min_payout_amount" db:"min_payout_amount"`
	PayoutAddress    string     `json:"payout_address" db:"payout_address"`
	AutoPayoutEnable bool       `json:"auto_payout_enable" db:"auto_payout_enable"`
	CreatedAt        time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at" db:"updated_at"`
}

// -----------------------------------------------------------------------------
// Merged Mining Interfaces (Future-proof placeholders)
// -----------------------------------------------------------------------------

// AuxChainConfig represents configuration for an auxiliary chain in merged mining
type AuxChainConfig struct {
	ChainID       string  `json:"chain_id" yaml:"chain_id"`
	ChainName     string  `json:"chain_name" yaml:"chain_name"`
	Symbol        string  `json:"symbol" yaml:"symbol"`
	Enabled       bool    `json:"enabled" yaml:"enabled"`
	RPCURL        string  `json:"rpc_url" yaml:"rpc_url"`
	WalletAddress string  `json:"wallet_address" yaml:"wallet_address"`
	FeePercent    float64 `json:"fee_percent" yaml:"fee_percent"`
}

// MergedMiningProvider defines the interface for merged mining support
type MergedMiningProvider interface {
	// GetAuxChains returns all configured auxiliary chains
	GetAuxChains() []AuxChainConfig

	// GetAuxBlockTemplate gets a block template from an aux chain
	GetAuxBlockTemplate(ctx context.Context, chainID string) ([]byte, error)

	// SubmitAuxBlock submits a solved block to an aux chain
	SubmitAuxBlock(ctx context.Context, chainID string, blockData []byte) error

	// CalculateAuxReward calculates expected aux chain reward
	CalculateAuxReward(ctx context.Context, chainID string, shares []Share) (int64, error)
}

// MergedMiningHook is called when processing blocks that may have aux chain rewards
type MergedMiningHook interface {
	// OnBlockFound is called when a primary chain block is found
	OnBlockFound(ctx context.Context, block *Block) error

	// OnAuxBlockFound is called when an auxiliary chain block is found
	OnAuxBlockFound(ctx context.Context, chainID string, block *Block) error

	// GetAuxPayouts calculates payouts for aux chain blocks
	GetAuxPayouts(ctx context.Context, chainID string, shares []Share, reward int64) ([]Payout, error)
}

// NullMergedMiningProvider is a no-op implementation for when merged mining is disabled
type NullMergedMiningProvider struct{}

func (n *NullMergedMiningProvider) GetAuxChains() []AuxChainConfig {
	return nil
}

func (n *NullMergedMiningProvider) GetAuxBlockTemplate(ctx context.Context, chainID string) ([]byte, error) {
	return nil, nil
}

func (n *NullMergedMiningProvider) SubmitAuxBlock(ctx context.Context, chainID string, blockData []byte) error {
	return nil
}

func (n *NullMergedMiningProvider) CalculateAuxReward(ctx context.Context, chainID string, shares []Share) (int64, error) {
	return 0, nil
}

// -----------------------------------------------------------------------------
// Extended Database Interface
// -----------------------------------------------------------------------------

// PayoutSettingsRepository handles user payout settings persistence
type PayoutSettingsRepository interface {
	// GetUserSettings retrieves payout settings for a user
	GetUserSettings(ctx context.Context, userID int64) (*UserPayoutSettings, error)

	// SaveUserSettings saves or updates user payout settings
	SaveUserSettings(ctx context.Context, settings *UserPayoutSettings) error

	// GetUsersWithMode retrieves all users using a specific payout mode
	GetUsersWithMode(ctx context.Context, mode PayoutMode) ([]int64, error)

	// GetPendingPayouts retrieves users with pending payouts above threshold
	GetPendingPayouts(ctx context.Context, minAmount int64) ([]UserPayoutSettings, error)
}

// ShareRepository handles share data access for payout calculations
type ShareRepository interface {
	// GetSharesInWindow retrieves shares within a difficulty window before blockTime
	GetSharesInWindow(ctx context.Context, blockTime time.Time, windowSize int64) ([]Share, error)

	// GetSharesSince retrieves all shares since a given time
	GetSharesSince(ctx context.Context, since time.Time) ([]Share, error)

	// GetUserSharesSince retrieves shares for a specific user since a given time
	GetUserSharesSince(ctx context.Context, userID int64, since time.Time) ([]Share, error)

	// GetTotalDifficultySince calculates total share difficulty since a time
	GetTotalDifficultySince(ctx context.Context, since time.Time) (float64, error)
}

// NetworkStatsProvider provides network statistics for PPS calculations
type NetworkStatsProvider interface {
	// GetNetworkDifficulty returns current network difficulty
	GetNetworkDifficulty(ctx context.Context) (float64, error)

	// GetExpectedBlockReward returns expected block reward at current height
	GetExpectedBlockReward(ctx context.Context) (int64, error)

	// GetAverageTxFees returns average transaction fees per block
	GetAverageTxFees(ctx context.Context, lastNBlocks int) (int64, error)

	// GetBlockTime returns average block time
	GetBlockTime(ctx context.Context) (time.Duration, error)
}
