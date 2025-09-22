package community

import (
	"time"
	"github.com/google/uuid"
)

// Team represents a mining team
type Team struct {
	ID          uuid.UUID `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description" db:"description"`
	OwnerID     uuid.UUID `json:"owner_id" db:"owner_id"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
	IsActive    bool      `json:"is_active" db:"is_active"`
	
	// Statistics
	TotalHashrate   float64 `json:"total_hashrate" db:"total_hashrate"`
	MemberCount     int     `json:"member_count" db:"member_count"`
	TotalShares     int64   `json:"total_shares" db:"total_shares"`
	BlocksFound     int     `json:"blocks_found" db:"blocks_found"`
}

// TeamMember represents a team membership
type TeamMember struct {
	ID       uuid.UUID `json:"id" db:"id"`
	TeamID   uuid.UUID `json:"team_id" db:"team_id"`
	UserID   uuid.UUID `json:"user_id" db:"user_id"`
	Role     string    `json:"role" db:"role"` // owner, admin, member
	JoinedAt time.Time `json:"joined_at" db:"joined_at"`
	IsActive bool      `json:"is_active" db:"is_active"`
}

// Referral represents a referral relationship
type Referral struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	ReferrerID  uuid.UUID  `json:"referrer_id" db:"referrer_id"`
	ReferredID  uuid.UUID  `json:"referred_id" db:"referred_id"`
	Code        string     `json:"code" db:"code"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	CompletedAt *time.Time `json:"completed_at" db:"completed_at"`
	BonusAmount float64    `json:"bonus_amount" db:"bonus_amount"`
	Status      string     `json:"status" db:"status"` // pending, completed, expired
}

// Competition represents a mining competition
type Competition struct {
	ID          uuid.UUID `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description" db:"description"`
	StartTime   time.Time `json:"start_time" db:"start_time"`
	EndTime     time.Time `json:"end_time" db:"end_time"`
	PrizePool   float64   `json:"prize_pool" db:"prize_pool"`
	Rules       string    `json:"rules" db:"rules"`
	Status      string    `json:"status" db:"status"` // upcoming, active, completed, cancelled
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// CompetitionParticipant represents a competition participant
type CompetitionParticipant struct {
	ID            uuid.UUID `json:"id" db:"id"`
	CompetitionID uuid.UUID `json:"competition_id" db:"competition_id"`
	UserID        uuid.UUID `json:"user_id" db:"user_id"`
	TeamID        *uuid.UUID `json:"team_id" db:"team_id"`
	JoinedAt      time.Time `json:"joined_at" db:"joined_at"`
	
	// Performance metrics
	TotalShares   int64   `json:"total_shares" db:"total_shares"`
	TotalHashrate float64 `json:"total_hashrate" db:"total_hashrate"`
	Rank          int     `json:"rank" db:"rank"`
	Prize         float64 `json:"prize" db:"prize"`
}

// SocialShare represents a social media share
type SocialShare struct {
	ID          uuid.UUID `json:"id" db:"id"`
	UserID      uuid.UUID `json:"user_id" db:"user_id"`
	Platform    string    `json:"platform" db:"platform"` // twitter, discord, telegram
	Content     string    `json:"content" db:"content"`
	ShareURL    string    `json:"share_url" db:"share_url"`
	Milestone   string    `json:"milestone" db:"milestone"`
	SharedAt    time.Time `json:"shared_at" db:"shared_at"`
	BonusAmount float64   `json:"bonus_amount" db:"bonus_amount"`
}

// TeamStatistics represents team performance statistics
type TeamStatistics struct {
	TeamID        uuid.UUID `json:"team_id"`
	Period        string    `json:"period"` // daily, weekly, monthly
	Date          time.Time `json:"date"`
	TotalHashrate float64   `json:"total_hashrate"`
	TotalShares   int64     `json:"total_shares"`
	BlocksFound   int       `json:"blocks_found"`
	Earnings      float64   `json:"earnings"`
	MemberCount   int       `json:"member_count"`
}