package community

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// PostgreSQLRepository implements the Repository interface using PostgreSQL
type PostgreSQLRepository struct {
	db *sqlx.DB
}

// NewPostgreSQLRepository creates a new PostgreSQL repository
func NewPostgreSQLRepository(db *sqlx.DB) *PostgreSQLRepository {
	return &PostgreSQLRepository{
		db: db,
	}
}

// CreateTeam creates a new team
func (r *PostgreSQLRepository) CreateTeam(ctx context.Context, team *Team) error {
	query := `
		INSERT INTO teams (id, name, description, owner_id, created_at, updated_at, is_active, total_hashrate, member_count, total_shares, blocks_found)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	
	_, err := r.db.ExecContext(ctx, query, team.ID, team.Name, team.Description, team.OwnerID, team.CreatedAt, team.UpdatedAt, team.IsActive, team.TotalHashrate, team.MemberCount, team.TotalShares, team.BlocksFound)
	if err != nil {
		return fmt.Errorf("failed to create team: %w", err)
	}
	
	return nil
}

// GetTeam retrieves a team by ID
func (r *PostgreSQLRepository) GetTeam(ctx context.Context, id uuid.UUID) (*Team, error) {
	query := `
		SELECT id, name, description, owner_id, created_at, updated_at, is_active, total_hashrate, member_count, total_shares, blocks_found
		FROM teams
		WHERE id = $1
	`
	
	var team Team
	err := r.db.GetContext(ctx, &team, query, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get team: %w", err)
	}
	
	return &team, nil
}

// UpdateTeam updates an existing team
func (r *PostgreSQLRepository) UpdateTeam(ctx context.Context, team *Team) error {
	query := `
		UPDATE teams
		SET name = $2, description = $3, updated_at = $4, is_active = $5, total_hashrate = $6, member_count = $7, total_shares = $8, blocks_found = $9
		WHERE id = $1
	`
	
	_, err := r.db.ExecContext(ctx, query, team.ID, team.Name, team.Description, team.UpdatedAt, team.IsActive, team.TotalHashrate, team.MemberCount, team.TotalShares, team.BlocksFound)
	if err != nil {
		return fmt.Errorf("failed to update team: %w", err)
	}
	
	return nil
}

// DeleteTeam deletes a team
func (r *PostgreSQLRepository) DeleteTeam(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE teams SET is_active = false WHERE id = $1`
	
	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete team: %w", err)
	}
	
	return nil
}

// AddTeamMember adds a member to a team
func (r *PostgreSQLRepository) AddTeamMember(ctx context.Context, member *TeamMember) error {
	query := `
		INSERT INTO team_members (id, team_id, user_id, role, joined_at, is_active)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	
	_, err := r.db.ExecContext(ctx, query, member.ID, member.TeamID, member.UserID, member.Role, member.JoinedAt, member.IsActive)
	if err != nil {
		return fmt.Errorf("failed to add team member: %w", err)
	}
	
	return nil
}

// RemoveTeamMember removes a member from a team
func (r *PostgreSQLRepository) RemoveTeamMember(ctx context.Context, teamID, userID uuid.UUID) error {
	query := `UPDATE team_members SET is_active = false WHERE team_id = $1 AND user_id = $2`
	
	_, err := r.db.ExecContext(ctx, query, teamID, userID)
	if err != nil {
		return fmt.Errorf("failed to remove team member: %w", err)
	}
	
	return nil
}

// GetTeamMembers retrieves all members of a team
func (r *PostgreSQLRepository) GetTeamMembers(ctx context.Context, teamID uuid.UUID) ([]*TeamMember, error) {
	query := `
		SELECT id, team_id, user_id, role, joined_at, is_active
		FROM team_members
		WHERE team_id = $1 AND is_active = true
		ORDER BY joined_at ASC
	`
	
	var members []*TeamMember
	err := r.db.SelectContext(ctx, &members, query, teamID)
	if err != nil {
		return nil, fmt.Errorf("failed to get team members: %w", err)
	}
	
	return members, nil
}

// CreateReferral creates a new referral
func (r *PostgreSQLRepository) CreateReferral(ctx context.Context, referral *Referral) error {
	query := `
		INSERT INTO referrals (id, referrer_id, referred_id, code, created_at, completed_at, bonus_amount, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	
	_, err := r.db.ExecContext(ctx, query, referral.ID, referral.ReferrerID, referral.ReferredID, referral.Code, referral.CreatedAt, referral.CompletedAt, referral.BonusAmount, referral.Status)
	if err != nil {
		return fmt.Errorf("failed to create referral: %w", err)
	}
	
	return nil
}

// GetReferral retrieves a referral by code
func (r *PostgreSQLRepository) GetReferral(ctx context.Context, code string) (*Referral, error) {
	query := `
		SELECT id, referrer_id, referred_id, code, created_at, completed_at, bonus_amount, status
		FROM referrals
		WHERE code = $1
	`
	
	var referral Referral
	err := r.db.GetContext(ctx, &referral, query, code)
	if err != nil {
		return nil, fmt.Errorf("failed to get referral: %w", err)
	}
	
	return &referral, nil
}

// UpdateReferral updates an existing referral
func (r *PostgreSQLRepository) UpdateReferral(ctx context.Context, referral *Referral) error {
	query := `
		UPDATE referrals
		SET referred_id = $2, completed_at = $3, bonus_amount = $4, status = $5
		WHERE id = $1
	`
	
	_, err := r.db.ExecContext(ctx, query, referral.ID, referral.ReferredID, referral.CompletedAt, referral.BonusAmount, referral.Status)
	if err != nil {
		return fmt.Errorf("failed to update referral: %w", err)
	}
	
	return nil
}

// CreateCompetition creates a new competition
func (r *PostgreSQLRepository) CreateCompetition(ctx context.Context, competition *Competition) error {
	query := `
		INSERT INTO competitions (id, name, description, start_time, end_time, prize_pool, rules, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	
	_, err := r.db.ExecContext(ctx, query, competition.ID, competition.Name, competition.Description, competition.StartTime, competition.EndTime, competition.PrizePool, competition.Rules, competition.Status, competition.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to create competition: %w", err)
	}
	
	return nil
}

// GetActiveCompetitions retrieves all active competitions
func (r *PostgreSQLRepository) GetActiveCompetitions(ctx context.Context) ([]*Competition, error) {
	query := `
		SELECT id, name, description, start_time, end_time, prize_pool, rules, status, created_at
		FROM competitions
		WHERE status IN ('upcoming', 'active')
		ORDER BY start_time ASC
	`
	
	var competitions []*Competition
	err := r.db.SelectContext(ctx, &competitions, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get active competitions: %w", err)
	}
	
	return competitions, nil
}

// JoinCompetition adds a participant to a competition
func (r *PostgreSQLRepository) JoinCompetition(ctx context.Context, participant *CompetitionParticipant) error {
	query := `
		INSERT INTO competition_participants (id, competition_id, user_id, team_id, joined_at, total_shares, total_hashrate, rank, prize)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	
	_, err := r.db.ExecContext(ctx, query, participant.ID, participant.CompetitionID, participant.UserID, participant.TeamID, participant.JoinedAt, participant.TotalShares, participant.TotalHashrate, participant.Rank, participant.Prize)
	if err != nil {
		return fmt.Errorf("failed to join competition: %w", err)
	}
	
	return nil
}

// RecordSocialShare records a social media share
func (r *PostgreSQLRepository) RecordSocialShare(ctx context.Context, share *SocialShare) error {
	query := `
		INSERT INTO social_shares (id, user_id, platform, content, share_url, milestone, shared_at, bonus_amount)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	
	_, err := r.db.ExecContext(ctx, query, share.ID, share.UserID, share.Platform, share.Content, share.ShareURL, share.Milestone, share.SharedAt, share.BonusAmount)
	if err != nil {
		return fmt.Errorf("failed to record social share: %w", err)
	}
	
	return nil
}

// GetTeamStatistics retrieves team statistics for a given period
func (r *PostgreSQLRepository) GetTeamStatistics(ctx context.Context, teamID uuid.UUID, period string, days int) ([]*TeamStatistics, error) {
	query := `
		SELECT team_id, period, date, total_hashrate, total_shares, blocks_found, earnings, member_count
		FROM team_statistics
		WHERE team_id = $1 AND period = $2 AND date >= $3
		ORDER BY date ASC
	`
	
	startDate := time.Now().AddDate(0, 0, -days)
	
	var stats []*TeamStatistics
	err := r.db.SelectContext(ctx, &stats, query, teamID, period, startDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get team statistics: %w", err)
	}
	
	return stats, nil
}

// Custom types for JSON handling
type JSONStringMap map[string]string

func (j JSONStringMap) Value() (driver.Value, error) {
	return json.Marshal(j)
}

func (j *JSONStringMap) Scan(value interface{}) error {
	if value == nil {
		*j = make(map[string]string)
		return nil
	}
	
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("cannot scan %T into JSONStringMap", value)
	}
	
	return json.Unmarshal(bytes, j)
}