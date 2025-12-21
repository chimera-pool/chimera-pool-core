package equipment

import (
	"context"
	"errors"
	"testing"
	"time"
)

// === Mock Implementations ===

type MockEquipmentReader struct {
	equipment      map[string]*Equipment
	userEquipment  map[string][]Equipment
	equipmentStats *EquipmentStats
	poolStats      *EquipmentStats
	err            error
}

func NewMockEquipmentReader() *MockEquipmentReader {
	return &MockEquipmentReader{
		equipment:     make(map[string]*Equipment),
		userEquipment: make(map[string][]Equipment),
	}
}

func (m *MockEquipmentReader) GetEquipment(ctx context.Context, equipmentID string) (*Equipment, error) {
	if m.err != nil {
		return nil, m.err
	}
	eq, ok := m.equipment[equipmentID]
	if !ok {
		return nil, errors.New("equipment not found")
	}
	return eq, nil
}

func (m *MockEquipmentReader) GetEquipmentByWorker(ctx context.Context, userID, workerName string) (*Equipment, error) {
	if m.err != nil {
		return nil, m.err
	}
	for _, eq := range m.equipment {
		if eq.UserID == userID && eq.WorkerName == workerName {
			return eq, nil
		}
	}
	return nil, errors.New("equipment not found")
}

func (m *MockEquipmentReader) ListUserEquipment(ctx context.Context, userID string) ([]Equipment, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.userEquipment[userID], nil
}

func (m *MockEquipmentReader) ListEquipment(ctx context.Context, filter EquipmentFilter) ([]Equipment, error) {
	if m.err != nil {
		return nil, m.err
	}
	var result []Equipment
	for _, eq := range m.equipment {
		if filter.UserID != "" && eq.UserID != filter.UserID {
			continue
		}
		if filter.Status != "" && eq.Status != filter.Status {
			continue
		}
		if filter.Type != "" && eq.Type != filter.Type {
			continue
		}
		result = append(result, *eq)
	}
	return result, nil
}

func (m *MockEquipmentReader) GetUserEquipmentStats(ctx context.Context, userID string) (*EquipmentStats, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.equipmentStats, nil
}

func (m *MockEquipmentReader) GetPoolEquipmentStats(ctx context.Context) (*EquipmentStats, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.poolStats, nil
}

type MockEquipmentWriter struct {
	equipment map[string]*Equipment
	err       error
}

func NewMockEquipmentWriter() *MockEquipmentWriter {
	return &MockEquipmentWriter{
		equipment: make(map[string]*Equipment),
	}
}

func (m *MockEquipmentWriter) CreateEquipment(ctx context.Context, equipment *Equipment) error {
	if m.err != nil {
		return m.err
	}
	m.equipment[equipment.ID] = equipment
	return nil
}

func (m *MockEquipmentWriter) UpdateEquipment(ctx context.Context, equipment *Equipment) error {
	if m.err != nil {
		return m.err
	}
	if _, ok := m.equipment[equipment.ID]; !ok {
		return errors.New("equipment not found")
	}
	m.equipment[equipment.ID] = equipment
	return nil
}

func (m *MockEquipmentWriter) DeleteEquipment(ctx context.Context, equipmentID string) error {
	if m.err != nil {
		return m.err
	}
	if _, ok := m.equipment[equipmentID]; !ok {
		return errors.New("equipment not found")
	}
	delete(m.equipment, equipmentID)
	return nil
}

func (m *MockEquipmentWriter) SetEquipmentName(ctx context.Context, equipmentID, name string) error {
	if m.err != nil {
		return m.err
	}
	eq, ok := m.equipment[equipmentID]
	if !ok {
		return errors.New("equipment not found")
	}
	eq.Name = name
	return nil
}

func (m *MockEquipmentWriter) SetEquipmentStatus(ctx context.Context, equipmentID string, status EquipmentStatus) error {
	if m.err != nil {
		return m.err
	}
	eq, ok := m.equipment[equipmentID]
	if !ok {
		return errors.New("equipment not found")
	}
	eq.Status = status
	return nil
}

type MockPayoutSplitManager struct {
	splits map[string][]PayoutSplit
	err    error
}

func NewMockPayoutSplitManager() *MockPayoutSplitManager {
	return &MockPayoutSplitManager{
		splits: make(map[string][]PayoutSplit),
	}
}

func (m *MockPayoutSplitManager) GetPayoutSplits(ctx context.Context, equipmentID string) ([]PayoutSplit, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.splits[equipmentID], nil
}

func (m *MockPayoutSplitManager) SetPayoutSplits(ctx context.Context, equipmentID string, splits []PayoutSplit) error {
	if m.err != nil {
		return m.err
	}
	if err := m.ValidatePayoutSplits(splits); err != nil {
		return err
	}
	m.splits[equipmentID] = splits
	return nil
}

func (m *MockPayoutSplitManager) AddPayoutSplit(ctx context.Context, split *PayoutSplit) error {
	if m.err != nil {
		return m.err
	}
	m.splits[split.EquipmentID] = append(m.splits[split.EquipmentID], *split)
	return nil
}

func (m *MockPayoutSplitManager) UpdatePayoutSplit(ctx context.Context, split *PayoutSplit) error {
	if m.err != nil {
		return m.err
	}
	splits := m.splits[split.EquipmentID]
	for i, s := range splits {
		if s.ID == split.ID {
			splits[i] = *split
			m.splits[split.EquipmentID] = splits
			return nil
		}
	}
	return errors.New("split not found")
}

func (m *MockPayoutSplitManager) RemovePayoutSplit(ctx context.Context, splitID string) error {
	if m.err != nil {
		return m.err
	}
	for eqID, splits := range m.splits {
		for i, s := range splits {
			if s.ID == splitID {
				m.splits[eqID] = append(splits[:i], splits[i+1:]...)
				return nil
			}
		}
	}
	return errors.New("split not found")
}

func (m *MockPayoutSplitManager) ValidatePayoutSplits(splits []PayoutSplit) error {
	var totalPercentage float64
	for _, split := range splits {
		if split.Percentage <= 0 || split.Percentage > 100 {
			return errors.New("percentage must be between 0 and 100")
		}
		if split.WalletAddress == "" {
			return errors.New("wallet address required")
		}
		totalPercentage += split.Percentage
	}
	if len(splits) > 0 && totalPercentage != 100 {
		return errors.New("total percentage must equal 100")
	}
	return nil
}

// === Tests ===

func TestEquipmentStatus(t *testing.T) {
	tests := []struct {
		status   EquipmentStatus
		expected string
	}{
		{StatusOnline, "online"},
		{StatusOffline, "offline"},
		{StatusMining, "mining"},
		{StatusIdle, "idle"},
		{StatusError, "error"},
		{StatusMaintenance, "maintenance"},
	}

	for _, tt := range tests {
		if string(tt.status) != tt.expected {
			t.Errorf("expected %s, got %s", tt.expected, string(tt.status))
		}
	}
}

func TestEquipmentType(t *testing.T) {
	tests := []struct {
		eqType   EquipmentType
		expected string
	}{
		{TypeASIC, "asic"},
		{TypeGPU, "gpu"},
		{TypeCPU, "cpu"},
		{TypeFPGA, "fpga"},
		{TypeOfficialX30, "blockdag_x30"},
		{TypeOfficialX100, "blockdag_x100"},
	}

	for _, tt := range tests {
		if string(tt.eqType) != tt.expected {
			t.Errorf("expected %s, got %s", tt.expected, string(tt.eqType))
		}
	}
}

func TestEquipmentReader_GetEquipment(t *testing.T) {
	ctx := context.Background()
	reader := NewMockEquipmentReader()

	// Add test equipment
	eq := &Equipment{
		ID:              "eq-001",
		UserID:          "user-001",
		Name:            "My X100",
		Type:            TypeOfficialX100,
		Status:          StatusMining,
		CurrentHashrate: 150000000,
	}
	reader.equipment[eq.ID] = eq

	// Test successful retrieval
	result, err := reader.GetEquipment(ctx, "eq-001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != eq.ID {
		t.Errorf("expected ID %s, got %s", eq.ID, result.ID)
	}
	if result.Name != eq.Name {
		t.Errorf("expected Name %s, got %s", eq.Name, result.Name)
	}

	// Test not found
	_, err = reader.GetEquipment(ctx, "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent equipment")
	}
}

func TestEquipmentReader_ListUserEquipment(t *testing.T) {
	ctx := context.Background()
	reader := NewMockEquipmentReader()

	userID := "user-001"
	reader.userEquipment[userID] = []Equipment{
		{ID: "eq-001", UserID: userID, Name: "GPU Rig 1", Type: TypeGPU, Status: StatusMining},
		{ID: "eq-002", UserID: userID, Name: "X100 ASIC", Type: TypeOfficialX100, Status: StatusMining},
		{ID: "eq-003", UserID: userID, Name: "CPU Miner", Type: TypeCPU, Status: StatusOffline},
	}

	result, err := reader.ListUserEquipment(ctx, userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 3 {
		t.Errorf("expected 3 equipment, got %d", len(result))
	}
}

func TestEquipmentReader_ListEquipmentWithFilter(t *testing.T) {
	ctx := context.Background()
	reader := NewMockEquipmentReader()

	reader.equipment["eq-001"] = &Equipment{ID: "eq-001", UserID: "user-001", Type: TypeGPU, Status: StatusMining}
	reader.equipment["eq-002"] = &Equipment{ID: "eq-002", UserID: "user-001", Type: TypeOfficialX100, Status: StatusOffline}
	reader.equipment["eq-003"] = &Equipment{ID: "eq-003", UserID: "user-002", Type: TypeGPU, Status: StatusMining}

	// Filter by user
	result, err := reader.ListEquipment(ctx, EquipmentFilter{UserID: "user-001"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 equipment for user-001, got %d", len(result))
	}

	// Filter by status
	result, err = reader.ListEquipment(ctx, EquipmentFilter{Status: StatusMining})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 mining equipment, got %d", len(result))
	}

	// Filter by type
	result, err = reader.ListEquipment(ctx, EquipmentFilter{Type: TypeGPU})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 GPU equipment, got %d", len(result))
	}
}

func TestEquipmentWriter_CreateEquipment(t *testing.T) {
	ctx := context.Background()
	writer := NewMockEquipmentWriter()

	eq := &Equipment{
		ID:           "eq-new",
		UserID:       "user-001",
		Name:         "New Miner",
		Type:         TypeGPU,
		Status:       StatusOffline,
		RegisteredAt: time.Now(),
	}

	err := writer.CreateEquipment(ctx, eq)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify it was stored
	if _, ok := writer.equipment[eq.ID]; !ok {
		t.Error("equipment was not stored")
	}
}

func TestEquipmentWriter_UpdateEquipment(t *testing.T) {
	ctx := context.Background()
	writer := NewMockEquipmentWriter()

	// Create initial equipment
	eq := &Equipment{ID: "eq-001", Name: "Original Name"}
	writer.equipment[eq.ID] = eq

	// Update it
	eq.Name = "Updated Name"
	err := writer.UpdateEquipment(ctx, eq)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if writer.equipment[eq.ID].Name != "Updated Name" {
		t.Error("equipment was not updated")
	}

	// Try to update nonexistent
	err = writer.UpdateEquipment(ctx, &Equipment{ID: "nonexistent"})
	if err == nil {
		t.Error("expected error for nonexistent equipment")
	}
}

func TestEquipmentWriter_DeleteEquipment(t *testing.T) {
	ctx := context.Background()
	writer := NewMockEquipmentWriter()

	writer.equipment["eq-001"] = &Equipment{ID: "eq-001"}

	err := writer.DeleteEquipment(ctx, "eq-001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, ok := writer.equipment["eq-001"]; ok {
		t.Error("equipment was not deleted")
	}

	// Try to delete nonexistent
	err = writer.DeleteEquipment(ctx, "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent equipment")
	}
}

func TestEquipmentWriter_SetEquipmentStatus(t *testing.T) {
	ctx := context.Background()
	writer := NewMockEquipmentWriter()

	writer.equipment["eq-001"] = &Equipment{ID: "eq-001", Status: StatusOffline}

	err := writer.SetEquipmentStatus(ctx, "eq-001", StatusMining)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if writer.equipment["eq-001"].Status != StatusMining {
		t.Error("status was not updated")
	}
}

func TestPayoutSplitManager_ValidatePayoutSplits(t *testing.T) {
	manager := NewMockPayoutSplitManager()

	// Valid splits
	validSplits := []PayoutSplit{
		{ID: "s1", WalletAddress: "wallet1", Percentage: 70},
		{ID: "s2", WalletAddress: "wallet2", Percentage: 30},
	}
	err := manager.ValidatePayoutSplits(validSplits)
	if err != nil {
		t.Errorf("unexpected error for valid splits: %v", err)
	}

	// Invalid: doesn't add up to 100
	invalidSplits := []PayoutSplit{
		{ID: "s1", WalletAddress: "wallet1", Percentage: 50},
		{ID: "s2", WalletAddress: "wallet2", Percentage: 30},
	}
	err = manager.ValidatePayoutSplits(invalidSplits)
	if err == nil {
		t.Error("expected error for splits not adding to 100")
	}

	// Invalid: percentage out of range
	invalidSplits2 := []PayoutSplit{
		{ID: "s1", WalletAddress: "wallet1", Percentage: 150},
	}
	err = manager.ValidatePayoutSplits(invalidSplits2)
	if err == nil {
		t.Error("expected error for percentage > 100")
	}

	// Invalid: missing wallet
	invalidSplits3 := []PayoutSplit{
		{ID: "s1", WalletAddress: "", Percentage: 100},
	}
	err = manager.ValidatePayoutSplits(invalidSplits3)
	if err == nil {
		t.Error("expected error for missing wallet address")
	}
}

func TestPayoutSplitManager_SetPayoutSplits(t *testing.T) {
	ctx := context.Background()
	manager := NewMockPayoutSplitManager()

	equipmentID := "eq-001"
	splits := []PayoutSplit{
		{ID: "s1", EquipmentID: equipmentID, WalletAddress: "wallet1", Percentage: 60},
		{ID: "s2", EquipmentID: equipmentID, WalletAddress: "wallet2", Percentage: 40},
	}

	err := manager.SetPayoutSplits(ctx, equipmentID, splits)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result, _ := manager.GetPayoutSplits(ctx, equipmentID)
	if len(result) != 2 {
		t.Errorf("expected 2 splits, got %d", len(result))
	}
}

func TestPayoutSplitManager_AddAndRemoveSplit(t *testing.T) {
	ctx := context.Background()
	manager := NewMockPayoutSplitManager()

	equipmentID := "eq-001"
	split := &PayoutSplit{
		ID:            "s1",
		EquipmentID:   equipmentID,
		WalletAddress: "wallet1",
		Percentage:    100,
	}

	// Add split
	err := manager.AddPayoutSplit(ctx, split)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result, _ := manager.GetPayoutSplits(ctx, equipmentID)
	if len(result) != 1 {
		t.Errorf("expected 1 split, got %d", len(result))
	}

	// Remove split
	err = manager.RemovePayoutSplit(ctx, "s1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result, _ = manager.GetPayoutSplits(ctx, equipmentID)
	if len(result) != 0 {
		t.Errorf("expected 0 splits, got %d", len(result))
	}
}

func TestEquipment_StructFields(t *testing.T) {
	now := time.Now()
	eq := Equipment{
		ID:              "eq-001",
		UserID:          "user-001",
		Name:            "My Mining Rig",
		Type:            TypeOfficialX100,
		Status:          StatusMining,
		WorkerName:      "x100-main",
		IPAddress:       "192.168.1.100",
		Model:           "BlockDAG X100",
		Manufacturer:    "BlockDAG",
		FirmwareVersion: "1.2.3",
		CurrentHashrate: 150000000,
		AverageHashrate: 145000000,
		MaxHashrate:     160000000,
		Efficiency:      1.5,
		PowerUsage:      1200,
		Temperature:     65.5,
		FanSpeed:        80,
		Latency:         25.5,
		ConnectionType:  "stratum_v2",
		LastSeen:        now,
		Uptime:          86400,
		SharesAccepted:  10000,
		SharesRejected:  50,
		SharesStale:     10,
		BlocksFound:     2,
		TotalEarnings:   125.5,
		RegisteredAt:    now,
		LastUpdated:     now,
	}

	if eq.ID != "eq-001" {
		t.Error("ID field mismatch")
	}
	if eq.CurrentHashrate != 150000000 {
		t.Error("CurrentHashrate field mismatch")
	}
	if eq.Temperature != 65.5 {
		t.Error("Temperature field mismatch")
	}
	if eq.SharesAccepted != 10000 {
		t.Error("SharesAccepted field mismatch")
	}
}

func TestEquipmentStats_Struct(t *testing.T) {
	stats := EquipmentStats{
		TotalEquipment:      10,
		OnlineCount:         8,
		OfflineCount:        2,
		MiningCount:         7,
		ErrorCount:          1,
		TotalHashrate:       1500000000,
		AverageLatency:      30.5,
		TotalPowerUsage:     12000,
		TotalEarnings:       5000.50,
		TotalSharesAccepted: 100000,
		TotalSharesRejected: 500,
	}

	if stats.TotalEquipment != 10 {
		t.Error("TotalEquipment mismatch")
	}
	if stats.OnlineCount != 8 {
		t.Error("OnlineCount mismatch")
	}
	if stats.TotalHashrate != 1500000000 {
		t.Error("TotalHashrate mismatch")
	}
}

// Test interface compliance
func TestInterfaceCompliance(t *testing.T) {
	// These assignments verify that mocks implement interfaces correctly
	var _ EquipmentReader = (*MockEquipmentReader)(nil)
	var _ EquipmentWriter = (*MockEquipmentWriter)(nil)
	var _ PayoutSplitManager = (*MockPayoutSplitManager)(nil)
}
