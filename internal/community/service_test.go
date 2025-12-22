package community

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockRepository is a mock implementation of Repository
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) CreateTeam(ctx context.Context, team *Team) error {
	args := m.Called(ctx, team)
	return args.Error(0)
}

func (m *MockRepository) GetTeam(ctx context.Context, id uuid.UUID) (*Team, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*Team), args.Error(1)
}

func (m *MockRepository) UpdateTeam(ctx context.Context, team *Team) error {
	args := m.Called(ctx, team)
	return args.Error(0)
}

func (m *MockRepository) DeleteTeam(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRepository) AddTeamMember(ctx context.Context, member *TeamMember) error {
	args := m.Called(ctx, member)
	return args.Error(0)
}

func (m *MockRepository) RemoveTeamMember(ctx context.Context, teamID, userID uuid.UUID) error {
	args := m.Called(ctx, teamID, userID)
	return args.Error(0)
}

func (m *MockRepository) GetTeamMembers(ctx context.Context, teamID uuid.UUID) ([]*TeamMember, error) {
	args := m.Called(ctx, teamID)
	return args.Get(0).([]*TeamMember), args.Error(1)
}

func (m *MockRepository) CreateReferral(ctx context.Context, referral *Referral) error {
	args := m.Called(ctx, referral)
	return args.Error(0)
}

func (m *MockRepository) GetReferral(ctx context.Context, code string) (*Referral, error) {
	args := m.Called(ctx, code)
	return args.Get(0).(*Referral), args.Error(1)
}

func (m *MockRepository) UpdateReferral(ctx context.Context, referral *Referral) error {
	args := m.Called(ctx, referral)
	return args.Error(0)
}

func (m *MockRepository) CreateCompetition(ctx context.Context, competition *Competition) error {
	args := m.Called(ctx, competition)
	return args.Error(0)
}

func (m *MockRepository) GetActiveCompetitions(ctx context.Context) ([]*Competition, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*Competition), args.Error(1)
}

func (m *MockRepository) JoinCompetition(ctx context.Context, participant *CompetitionParticipant) error {
	args := m.Called(ctx, participant)
	return args.Error(0)
}

func (m *MockRepository) RecordSocialShare(ctx context.Context, share *SocialShare) error {
	args := m.Called(ctx, share)
	return args.Error(0)
}

func (m *MockRepository) GetTeamStatistics(ctx context.Context, teamID uuid.UUID, period string, days int) ([]*TeamStatistics, error) {
	args := m.Called(ctx, teamID, period, days)
	return args.Get(0).([]*TeamStatistics), args.Error(1)
}

func TestCommunityService_CreateTeam(t *testing.T) {
	tests := []struct {
		name        string
		teamName    string
		description string
		ownerID     uuid.UUID
		setupMock   func(*MockRepository)
		wantErr     bool
		errMsg      string
	}{
		{
			name:        "should create team successfully",
			teamName:    "Test Team",
			description: "A test mining team",
			ownerID:     uuid.New(),
			setupMock: func(m *MockRepository) {
				m.On("CreateTeam", mock.Anything, mock.AnythingOfType("*community.Team")).Return(nil)
				m.On("AddTeamMember", mock.Anything, mock.AnythingOfType("*community.TeamMember")).Return(nil)
			},
			wantErr: false,
		},
		{
			name:        "should fail with empty team name",
			teamName:    "",
			description: "A test mining team",
			ownerID:     uuid.New(),
			setupMock:   func(m *MockRepository) {},
			wantErr:     true,
			errMsg:      "team name cannot be empty",
		},
		{
			name:        "should fail with invalid owner ID",
			teamName:    "Test Team",
			description: "A test mining team",
			ownerID:     uuid.Nil,
			setupMock:   func(m *MockRepository) {},
			wantErr:     true,
			errMsg:      "owner ID cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &MockRepository{}
			tt.setupMock(mockRepo)

			service := NewService(mockRepo)

			team, err := service.CreateTeam(context.Background(), tt.teamName, tt.description, tt.ownerID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Nil(t, team)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, team)
				assert.Equal(t, tt.teamName, team.Name)
				assert.Equal(t, tt.description, team.Description)
				assert.Equal(t, tt.ownerID, team.OwnerID)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestCommunityService_JoinTeam(t *testing.T) {
	teamID := uuid.New()
	userID := uuid.New()

	tests := []struct {
		name      string
		teamID    uuid.UUID
		userID    uuid.UUID
		setupMock func(*MockRepository)
		wantErr   bool
		errMsg    string
	}{
		{
			name:   "should join team successfully",
			teamID: teamID,
			userID: userID,
			setupMock: func(m *MockRepository) {
				team := &Team{
					ID:       teamID,
					Name:     "Test Team",
					IsActive: true,
				}
				m.On("GetTeam", mock.Anything, teamID).Return(team, nil)
				m.On("AddTeamMember", mock.Anything, mock.AnythingOfType("*community.TeamMember")).Return(nil)
			},
			wantErr: false,
		},
		{
			name:   "should fail when team is inactive",
			teamID: teamID,
			userID: userID,
			setupMock: func(m *MockRepository) {
				team := &Team{
					ID:       teamID,
					Name:     "Test Team",
					IsActive: false,
				}
				m.On("GetTeam", mock.Anything, teamID).Return(team, nil)
			},
			wantErr: true,
			errMsg:  "team is not active",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &MockRepository{}
			tt.setupMock(mockRepo)

			service := NewService(mockRepo)

			err := service.JoinTeam(context.Background(), tt.teamID, tt.userID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestCommunityService_CreateReferral(t *testing.T) {
	referrerID := uuid.New()

	tests := []struct {
		name       string
		referrerID uuid.UUID
		setupMock  func(*MockRepository)
		wantErr    bool
		errMsg     string
	}{
		{
			name:       "should create referral successfully",
			referrerID: referrerID,
			setupMock: func(m *MockRepository) {
				m.On("CreateReferral", mock.Anything, mock.AnythingOfType("*community.Referral")).Return(nil)
			},
			wantErr: false,
		},
		{
			name:       "should fail with invalid referrer ID",
			referrerID: uuid.Nil,
			setupMock:  func(m *MockRepository) {},
			wantErr:    true,
			errMsg:     "referrer ID cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &MockRepository{}
			tt.setupMock(mockRepo)

			service := NewService(mockRepo)

			referral, err := service.CreateReferral(context.Background(), tt.referrerID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Nil(t, referral)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, referral)
				assert.Equal(t, tt.referrerID, referral.ReferrerID)
				assert.NotEmpty(t, referral.Code)
				assert.Equal(t, "pending", referral.Status)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestCommunityService_ProcessReferral(t *testing.T) {
	referralCode := "REF123456"
	referredID := uuid.New()

	tests := []struct {
		name         string
		referralCode string
		referredID   uuid.UUID
		setupMock    func(*MockRepository)
		wantErr      bool
		errMsg       string
	}{
		{
			name:         "should process referral successfully",
			referralCode: referralCode,
			referredID:   referredID,
			setupMock: func(m *MockRepository) {
				referral := &Referral{
					ID:         uuid.New(),
					ReferrerID: uuid.New(),
					Code:       referralCode,
					Status:     "pending",
					CreatedAt:  time.Now().Add(-24 * time.Hour),
				}
				m.On("GetReferral", mock.Anything, referralCode).Return(referral, nil)
				m.On("UpdateReferral", mock.Anything, mock.AnythingOfType("*community.Referral")).Return(nil)
			},
			wantErr: false,
		},
		{
			name:         "should fail with invalid referral code",
			referralCode: "INVALID",
			referredID:   referredID,
			setupMock: func(m *MockRepository) {
				m.On("GetReferral", mock.Anything, "INVALID").Return((*Referral)(nil), assert.AnError)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &MockRepository{}
			tt.setupMock(mockRepo)

			service := NewService(mockRepo)

			err := service.ProcessReferral(context.Background(), tt.referralCode, tt.referredID)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestCommunityService_CreateCompetition(t *testing.T) {
	startTime := time.Now().Add(24 * time.Hour)
	endTime := startTime.Add(7 * 24 * time.Hour)

	tests := []struct {
		name        string
		compName    string
		description string
		startTime   time.Time
		endTime     time.Time
		prizePool   float64
		setupMock   func(*MockRepository)
		wantErr     bool
		errMsg      string
	}{
		{
			name:        "should create competition successfully",
			compName:    "Weekly Mining Challenge",
			description: "Compete for the highest hashrate",
			startTime:   startTime,
			endTime:     endTime,
			prizePool:   1000.0,
			setupMock: func(m *MockRepository) {
				m.On("CreateCompetition", mock.Anything, mock.AnythingOfType("*community.Competition")).Return(nil)
			},
			wantErr: false,
		},
		{
			name:        "should fail with empty name",
			compName:    "",
			description: "Compete for the highest hashrate",
			startTime:   startTime,
			endTime:     endTime,
			prizePool:   1000.0,
			setupMock:   func(m *MockRepository) {},
			wantErr:     true,
			errMsg:      "competition name cannot be empty",
		},
		{
			name:        "should fail with end time before start time",
			compName:    "Weekly Mining Challenge",
			description: "Compete for the highest hashrate",
			startTime:   endTime,
			endTime:     startTime,
			prizePool:   1000.0,
			setupMock:   func(m *MockRepository) {},
			wantErr:     true,
			errMsg:      "end time must be after start time",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &MockRepository{}
			tt.setupMock(mockRepo)

			service := NewService(mockRepo)

			competition, err := service.CreateCompetition(context.Background(), tt.compName, tt.description, tt.startTime, tt.endTime, tt.prizePool)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Nil(t, competition)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, competition)
				assert.Equal(t, tt.compName, competition.Name)
				assert.Equal(t, tt.description, competition.Description)
				assert.Equal(t, tt.prizePool, competition.PrizePool)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestCommunityService_RecordSocialShare(t *testing.T) {
	userID := uuid.New()

	tests := []struct {
		name      string
		userID    uuid.UUID
		platform  string
		content   string
		milestone string
		setupMock func(*MockRepository)
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "should record social share successfully",
			userID:    userID,
			platform:  "twitter",
			content:   "Just reached 1000 shares on ChimeraPool!",
			milestone: "1000_shares",
			setupMock: func(m *MockRepository) {
				m.On("RecordSocialShare", mock.Anything, mock.AnythingOfType("*community.SocialShare")).Return(nil)
			},
			wantErr: false,
		},
		{
			name:      "should fail with invalid platform",
			userID:    userID,
			platform:  "invalid",
			content:   "Just reached 1000 shares on ChimeraPool!",
			milestone: "1000_shares",
			setupMock: func(m *MockRepository) {},
			wantErr:   true,
			errMsg:    "unsupported platform",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &MockRepository{}
			tt.setupMock(mockRepo)

			service := NewService(mockRepo)

			share, err := service.RecordSocialShare(context.Background(), tt.userID, tt.platform, tt.content, tt.milestone)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Nil(t, share)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, share)
				assert.Equal(t, tt.userID, share.UserID)
				assert.Equal(t, tt.platform, share.Platform)
				assert.Equal(t, tt.content, share.Content)
				assert.Equal(t, tt.milestone, share.Milestone)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

// =============================================================================
// ADDITIONAL COMPREHENSIVE TESTS FOR 55%+ COVERAGE
// =============================================================================

func TestCommunityService_JoinTeam_EmptyTeamID(t *testing.T) {
	mockRepo := &MockRepository{}
	service := NewService(mockRepo)

	err := service.JoinTeam(context.Background(), uuid.Nil, uuid.New())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "team ID cannot be empty")
}

func TestCommunityService_JoinTeam_EmptyUserID(t *testing.T) {
	mockRepo := &MockRepository{}
	service := NewService(mockRepo)

	err := service.JoinTeam(context.Background(), uuid.New(), uuid.Nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user ID cannot be empty")
}

func TestCommunityService_LeaveTeam_Success(t *testing.T) {
	mockRepo := &MockRepository{}
	teamID := uuid.New()
	userID := uuid.New()

	mockRepo.On("RemoveTeamMember", mock.Anything, teamID, userID).Return(nil)

	service := NewService(mockRepo)
	err := service.LeaveTeam(context.Background(), teamID, userID)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestCommunityService_LeaveTeam_EmptyTeamID(t *testing.T) {
	mockRepo := &MockRepository{}
	service := NewService(mockRepo)

	err := service.LeaveTeam(context.Background(), uuid.Nil, uuid.New())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "team ID cannot be empty")
}

func TestCommunityService_LeaveTeam_EmptyUserID(t *testing.T) {
	mockRepo := &MockRepository{}
	service := NewService(mockRepo)

	err := service.LeaveTeam(context.Background(), uuid.New(), uuid.Nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user ID cannot be empty")
}

func TestCommunityService_ProcessReferral_EmptyCode(t *testing.T) {
	mockRepo := &MockRepository{}
	service := NewService(mockRepo)

	err := service.ProcessReferral(context.Background(), "", uuid.New())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "referral code cannot be empty")
}

func TestCommunityService_ProcessReferral_EmptyReferredID(t *testing.T) {
	mockRepo := &MockRepository{}
	service := NewService(mockRepo)

	err := service.ProcessReferral(context.Background(), "REF123", uuid.Nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "referred user ID cannot be empty")
}

func TestCommunityService_ProcessReferral_NotPending(t *testing.T) {
	mockRepo := &MockRepository{}
	referralCode := "REF123"

	referral := &Referral{
		ID:         uuid.New(),
		ReferrerID: uuid.New(),
		Code:       referralCode,
		Status:     "completed",
		CreatedAt:  time.Now(),
	}
	mockRepo.On("GetReferral", mock.Anything, referralCode).Return(referral, nil)

	service := NewService(mockRepo)
	err := service.ProcessReferral(context.Background(), referralCode, uuid.New())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "referral is not pending")
}

func TestCommunityService_ProcessReferral_Expired(t *testing.T) {
	mockRepo := &MockRepository{}
	referralCode := "REFEXP"

	referral := &Referral{
		ID:         uuid.New(),
		ReferrerID: uuid.New(),
		Code:       referralCode,
		Status:     "pending",
		CreatedAt:  time.Now().Add(-31 * 24 * time.Hour),
	}
	mockRepo.On("GetReferral", mock.Anything, referralCode).Return(referral, nil)

	service := NewService(mockRepo)
	err := service.ProcessReferral(context.Background(), referralCode, uuid.New())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "referral has expired")
}

func TestCommunityService_CreateCompetition_NegativePrizePool(t *testing.T) {
	mockRepo := &MockRepository{}
	service := NewService(mockRepo)

	startTime := time.Now().Add(time.Hour)
	endTime := startTime.Add(24 * time.Hour)

	_, err := service.CreateCompetition(context.Background(), "Test", "Desc", startTime, endTime, -100)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "prize pool cannot be negative")
}

func TestCommunityService_JoinCompetition_Success(t *testing.T) {
	mockRepo := &MockRepository{}
	competitionID := uuid.New()
	userID := uuid.New()

	mockRepo.On("JoinCompetition", mock.Anything, mock.AnythingOfType("*community.CompetitionParticipant")).Return(nil)

	service := NewService(mockRepo)
	err := service.JoinCompetition(context.Background(), competitionID, userID, nil)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestCommunityService_JoinCompetition_EmptyCompetitionID(t *testing.T) {
	mockRepo := &MockRepository{}
	service := NewService(mockRepo)

	err := service.JoinCompetition(context.Background(), uuid.Nil, uuid.New(), nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "competition ID cannot be empty")
}

func TestCommunityService_JoinCompetition_EmptyUserID(t *testing.T) {
	mockRepo := &MockRepository{}
	service := NewService(mockRepo)

	err := service.JoinCompetition(context.Background(), uuid.New(), uuid.Nil, nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user ID cannot be empty")
}

func TestCommunityService_RecordSocialShare_EmptyUserID(t *testing.T) {
	mockRepo := &MockRepository{}
	service := NewService(mockRepo)

	_, err := service.RecordSocialShare(context.Background(), uuid.Nil, "twitter", "content", "milestone")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user ID cannot be empty")
}

func TestCommunityService_RecordSocialShare_EmptyContent(t *testing.T) {
	mockRepo := &MockRepository{}
	service := NewService(mockRepo)

	_, err := service.RecordSocialShare(context.Background(), uuid.New(), "twitter", "", "milestone")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "content cannot be empty")
}

func TestCommunityService_GetTeamStatistics_Success(t *testing.T) {
	mockRepo := &MockRepository{}
	teamID := uuid.New()

	stats := []*TeamStatistics{
		{TeamID: teamID, Period: "daily", TotalHashrate: 1000},
	}
	mockRepo.On("GetTeamStatistics", mock.Anything, teamID, "daily", 7).Return(stats, nil)

	service := NewService(mockRepo)
	result, err := service.GetTeamStatistics(context.Background(), teamID, "daily", 7)

	assert.NoError(t, err)
	assert.Len(t, result, 1)
}

func TestCommunityService_GetTeamStatistics_EmptyTeamID(t *testing.T) {
	mockRepo := &MockRepository{}
	service := NewService(mockRepo)

	_, err := service.GetTeamStatistics(context.Background(), uuid.Nil, "daily", 7)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "team ID cannot be empty")
}

func TestCommunityService_GetTeamStatistics_InvalidPeriod(t *testing.T) {
	mockRepo := &MockRepository{}
	service := NewService(mockRepo)

	_, err := service.GetTeamStatistics(context.Background(), uuid.New(), "invalid", 7)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid period")
}

func TestCommunityService_GetTeamStatistics_InvalidDays(t *testing.T) {
	mockRepo := &MockRepository{}
	service := NewService(mockRepo)

	_, err := service.GetTeamStatistics(context.Background(), uuid.New(), "daily", 0)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "days must be positive")
}

func TestNewCommunityService(t *testing.T) {
	mockRepo := &MockRepository{}
	service := NewService(mockRepo)

	assert.NotNil(t, service)
}

func TestChannelType_Constants(t *testing.T) {
	assert.Equal(t, ChannelType("text"), ChannelTypeText)
	assert.Equal(t, ChannelType("announcement"), ChannelTypeAnnouncement)
	assert.Equal(t, ChannelType("regional"), ChannelTypeRegional)
}
