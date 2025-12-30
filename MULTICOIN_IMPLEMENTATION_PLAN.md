# Multi-Coin Platform Implementation Plan

**Created**: December 30, 2025  
**Status**: Phase 1 In Progress  
**Goal**: Transform Chimera Pool into a universal multi-coin mining platform

---

## Executive Summary

This plan outlines the complete implementation of multi-coin support, transforming the pool from a single-coin (Litecoin) system to a universal platform supporting multiple cryptocurrencies with hot-swap capability.

---

## Current State Audit

### ✅ What EXISTS (Complete)

| Component | Status | Details |
|-----------|--------|---------|
| `network_configs` table | ✅ | Migration 009 - stores network RPC, algorithm, settings |
| `network_switch_history` table | ✅ | Audit trail for network switches |
| `user_network_stats` table | ✅ | Migration 016 - per-user, per-network stats |
| `miner_network_assignments` table | ✅ | Tracks which network each miner is on |
| `network_pool_stats` table | ✅ | Aggregated pool stats per network |
| Network Config API | ✅ | Full CRUD, switch, rollback, test endpoints |
| Admin UI Network Tab | ✅ | Network management in admin panel |

### ❌ Critical Gaps (Must Fix)

| Component | Gap | Impact |
|-----------|-----|--------|
| `shares` table | NO `network_id` column | Cannot track shares per coin |
| `blocks` table | NO `network_id` column | Cannot track blocks per coin |
| `payouts` table | NO `network_id` column | Cannot pay out per coin |
| `miners` table | NO `network_id` column | Cannot assign miners to networks |
| Stratum Server | Hardcoded Litecoin RPC | Cannot switch networks dynamically |
| Share Recording | No network tracking | All shares go to same pool |
| API Stats | Single-network only | Cannot show per-network stats |
| Frontend | No network selector | Users can't see multi-coin data |

---

## Implementation Phases (Dependency Order)

### Phase 1: Database Schema Extension (FOUNDATION)
**Priority**: CRITICAL - Everything depends on this  
**Estimated**: 2-3 hours

#### 1.1 Create Migration 017: Add network_id to core tables

```sql
-- Add network_id to shares table
ALTER TABLE shares ADD COLUMN network_id UUID REFERENCES network_configs(id);
CREATE INDEX idx_shares_network ON shares(network_id);

-- Add network_id to blocks table  
ALTER TABLE blocks ADD COLUMN network_id UUID REFERENCES network_configs(id);
CREATE INDEX idx_blocks_network ON blocks(network_id);

-- Add network_id to payouts table
ALTER TABLE payouts ADD COLUMN network_id UUID REFERENCES network_configs(id);
CREATE INDEX idx_payouts_network ON payouts(network_id);

-- Add network_id to miners table
ALTER TABLE miners ADD COLUMN network_id UUID REFERENCES network_configs(id);
CREATE INDEX idx_miners_network ON miners(network_id);

-- Update existing records to use current active network
UPDATE shares SET network_id = (SELECT id FROM network_configs WHERE is_active = true LIMIT 1) WHERE network_id IS NULL;
UPDATE blocks SET network_id = (SELECT id FROM network_configs WHERE is_active = true LIMIT 1) WHERE network_id IS NULL;
UPDATE payouts SET network_id = (SELECT id FROM network_configs WHERE is_active = true LIMIT 1) WHERE network_id IS NULL;
UPDATE miners SET network_id = (SELECT id FROM network_configs WHERE is_active = true LIMIT 1) WHERE network_id IS NULL;
```

#### 1.2 Update Go Models
- Add `NetworkID` field to `Share`, `Block`, `Payout`, `Miner` structs
- Update all database queries to include `network_id`

---

### Phase 2: Stratum Server Dynamic Network Loading
**Priority**: HIGH - Core mining engine  
**Estimated**: 4-5 hours

#### 2.1 Create NetworkConfigLoader Service

```go
type NetworkConfigLoader interface {
    GetActiveNetwork(ctx context.Context) (*NetworkConfig, error)
    WatchNetworkChanges(ctx context.Context, callback func(*NetworkConfig))
}
```

#### 2.2 Modify Stratum Server
- Remove hardcoded Litecoin RPC config
- Load active network from `network_configs` table on startup
- Create dynamic RPC client based on network config
- Support hot-reload when network switches

#### 2.3 Update Block Template Fetching
- Use network-specific RPC URL, credentials
- Support different algorithms (scrypt, sha256, etc.)
- Adjust coinbase generation per coin

---

### Phase 3: Share Processing with Network Context
**Priority**: HIGH - Core functionality  
**Estimated**: 3-4 hours

#### 3.1 Update Share Recording
- Pass `network_id` through entire share pipeline
- Record `network_id` with every share INSERT
- Update Redis keys to be network-aware: `pool:{network}:shares:valid`

#### 3.2 Update Miner Assignment
- Track which network each miner is connected to
- Update `miner_network_assignments` on connect/disconnect
- Support miners switching networks on reconnect

#### 3.3 Update Hashrate Tracking
- Track hashrate per network
- Aggregate pool hashrate per network
- Store in `network_pool_stats` table

---

### Phase 4: API Stats Network Awareness
**Priority**: MEDIUM - User-facing data  
**Estimated**: 3-4 hours

#### 4.1 Update Pool Stats Endpoints
- `/api/v1/pool/stats` - Add `network_id` parameter
- `/api/v1/pool/stats/hashrate` - Per-network hashrate history
- Return current network info with stats

#### 4.2 Update User Stats Endpoints
- `/api/v1/user/stats` - Add network filtering
- `/api/v1/user/networks/stats` - Already exists, verify working
- Aggregate stats across networks

#### 4.3 Add Network-Specific Endpoints
- `/api/v1/networks/:id/stats` - Stats for specific network
- `/api/v1/networks/:id/blocks` - Blocks found on network
- `/api/v1/networks/:id/top-miners` - Leaderboard per network

---

### Phase 5: Frontend Multi-Coin Dashboard
**Priority**: MEDIUM - User experience  
**Estimated**: 5-6 hours

#### 5.1 Network Selector Component
- Dropdown to select active network to view
- Show network icon, name, symbol
- Remember user's last selection

#### 5.2 Stats Cards Network Awareness
- Show which network stats are for
- Update stats when network changes
- Support "All Networks" aggregate view

#### 5.3 Charts Network Filtering
- Filter charts by selected network
- Show network-specific data
- Support comparison across networks

#### 5.4 Mining Instructions Per Network
- Show correct pool address per network
- Display network-specific mining instructions
- Algorithm-specific configuration

---

### Phase 6: Payout System Network Awareness
**Priority**: MEDIUM - Financial accuracy  
**Estimated**: 4-5 hours

#### 6.1 Earnings Calculation Per Network
- Calculate earnings per network separately
- Different reward structures per coin
- Track pending balance per network

#### 6.2 Payout Processing Per Network
- Generate payouts per network
- Use correct wallet addresses per coin
- Track payout history per network

#### 6.3 User Wallet Management
- Allow users to set wallet per network
- Validate addresses per coin type
- Default to single wallet if preferred

---

### Phase 7: Integration Testing
**Priority**: HIGH - Quality assurance  
**Estimated**: 4-5 hours

#### 7.1 Database Migration Testing
- Test migration on copy of production data
- Verify no data loss
- Test rollback procedure

#### 7.2 Stratum Server Testing
- Test network loading on startup
- Test hot-switch during mining
- Test miner reconnection

#### 7.3 End-to-End Testing
- Mine on Litecoin, verify shares recorded correctly
- Switch to another network, verify transition
- Verify stats accuracy per network

---

## Implementation Order (Start Here)

```
Phase 1.1 → Phase 1.2 → Phase 2.1 → Phase 2.2 → Phase 2.3
    ↓
Phase 3.1 → Phase 3.2 → Phase 3.3
    ↓
Phase 4.1 → Phase 4.2 → Phase 4.3
    ↓
Phase 5.1 → Phase 5.2 → Phase 5.3 → Phase 5.4
    ↓
Phase 6.1 → Phase 6.2 → Phase 6.3
    ↓
Phase 7.1 → Phase 7.2 → Phase 7.3
```

---

## Files to Modify

### Database
- [ ] `migrations/017_multicoin_core_tables.up.sql` - NEW
- [ ] `migrations/017_multicoin_core_tables.down.sql` - NEW

### Backend (Go)
- [ ] `internal/database/models.go` - Add NetworkID fields
- [ ] `internal/database/operations.go` - Update queries
- [ ] `cmd/stratum/main.go` - Dynamic network loading
- [ ] `internal/api/handlers.go` - Network-aware stats
- [ ] `internal/shares/share_processor.go` - Add network context

### Frontend (React)
- [ ] `src/components/NetworkSelector/NetworkSelector.tsx` - NEW
- [ ] `src/App.tsx` - Add network context
- [ ] `src/components/stats/*.tsx` - Network awareness

---

## Success Criteria

- [ ] Shares recorded with correct `network_id`
- [ ] Stratum loads network config from database
- [ ] Network switch works without restart
- [ ] Stats show correct data per network
- [ ] Frontend displays network selector
- [ ] Payouts processed per network
- [ ] All existing tests pass
- [ ] New integration tests pass

---

## Risk Mitigation

1. **Data Migration Risk**: Test on staging first, have rollback ready
2. **Downtime Risk**: Deploy during low-activity period
3. **Miner Disconnection**: Graceful reconnection handling
4. **Stats Accuracy**: Verify with manual calculation

---

## Current Progress

- [x] Phase 1: Database Schema Extension - COMPLETE (migration 017)
- [x] Phase 2: Stratum Dynamic Network Loading - COMPLETE (NetworkConfigLoader)
- [x] Phase 3: Share Processing with Network Context - COMPLETE (network_id in Go models)
- [~] Phase 4: API Stats Network Awareness - IN PROGRESS
- [~] Phase 5: Frontend Multi-Coin Dashboard - PARTIAL (multi-network instructions done)
- [ ] Phase 6: Payout System Network Awareness
- [ ] Phase 7: Integration Testing

### Completed Items (Dec 30, 2025):
- Migration 017: Added network_id to shares, blocks, payouts, miners tables
- Go models updated with NetworkID fields
- NetworkConfigLoader service created for dynamic network loading
- Stratum server integrated with network loader
- Share/miner DB operations include network_id
- Multi-network mining instructions component
- Time range selector for Grafana charts
- Active miners query fixed (last 5 min window)
