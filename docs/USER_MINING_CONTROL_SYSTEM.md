# Comprehensive User Mining Control System
## Elite-Level Mining Pool User Management

### Overview
This document outlines a comprehensive, industry-standard user control system for ChimeriaPool that gives users complete control over their mining operations, wallets, equipment, and payout allocations.

---

## 1. User Profile & Account Management

### 1.1 Profile Settings
- **Personal Information**: Username, email, timezone, language preference
- **Security Settings**: 2FA, API keys, session management, login history
- **Notification Preferences**: Email, push, in-app alerts for various events
- **Privacy Settings**: Public profile visibility, leaderboard participation

### 1.2 Account Tiers
| Tier | Features |
|------|----------|
| Basic | 3 wallets, 5 miners, 2 networks |
| Pro | 10 wallets, 25 miners, all networks |
| Enterprise | Unlimited wallets, miners, networks, API access |

---

## 2. Wallet Management System

### 2.1 Multi-Wallet Support
- **Add/Remove Wallets**: Support for multiple wallet addresses per network
- **Wallet Types**: 
  - Hot wallets (active payouts)
  - Cold wallets (long-term storage)
  - Exchange wallets (direct to exchange)
  - Staking wallets (for PoS networks)

### 2.2 Wallet Allocation System (Bug Reports #7, #8, #9)

#### 2.2.1 Percentage-Based Allocation
```
Wallet A: 50% ─────────────────────────▶ Cold Storage
Wallet B: 30% ─────────────────────────▶ Staking
Wallet C: 20% ─────────────────────────▶ Trading
         ────
         100% (Auto-balanced)
```

#### 2.2.2 Smart Allocation Features
- **Auto-Balance**: When increasing one wallet %, others auto-decrease proportionally
- **Active/Inactive Toggle**: Set wallet as inactive (0%) without deleting
- **Lock Wallet**: Prevent accidental changes to critical wallets
- **Minimum Threshold**: Set minimum payout threshold per wallet

#### 2.2.3 Per-Miner Wallet Assignment (Bug Report #9)
```
┌─────────────────────────────────────────────────────────────┐
│ MINER WALLET ASSIGNMENT                                     │
├─────────────────────────────────────────────────────────────┤
│ X100 ASIC      ──▶ Wallet A (Cold Storage)     100%        │
│ X30 ASIC #1    ──▶ Wallet B (Staking)          100%        │
│ X30 ASIC #2    ──▶ Wallet C (Trading)          100%        │
│ GPU Rig #1     ──▶ Split: Wallet A 50% / B 50%             │
└─────────────────────────────────────────────────────────────┘
```

#### 2.2.4 Wallet Database Schema Enhancement
```sql
-- Enhanced wallet allocation table
CREATE TABLE wallet_allocations (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id),
    wallet_id INTEGER REFERENCES wallets(id),
    miner_id INTEGER REFERENCES miners(id) NULL, -- NULL = applies to all miners
    network_id INTEGER REFERENCES networks(id),
    allocation_percent DECIMAL(5,2) NOT NULL CHECK (allocation_percent >= 0 AND allocation_percent <= 100),
    is_active BOOLEAN DEFAULT true,
    is_locked BOOLEAN DEFAULT false,
    min_payout_threshold DECIMAL(18,8) DEFAULT 0,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(user_id, wallet_id, miner_id, network_id)
);

-- Wallet status tracking
ALTER TABLE wallets ADD COLUMN status VARCHAR(20) DEFAULT 'active'; -- active, inactive, locked
ALTER TABLE wallets ADD COLUMN wallet_type VARCHAR(20) DEFAULT 'hot'; -- hot, cold, exchange, staking
ALTER TABLE wallets ADD COLUMN label VARCHAR(100); -- User-friendly name
```

---

## 3. Miner Management System

### 3.1 Miner Registration & Configuration
- **Auto-Discovery**: Miners auto-register on first connection
- **Manual Registration**: Pre-register miners with expected specs
- **Miner Naming**: Custom names for easy identification
- **Miner Groups**: Organize miners into logical groups

### 3.2 Per-Miner Settings
| Setting | Description |
|---------|-------------|
| Name | Custom identifier (e.g., "Basement Rig #1") |
| Wallet Assignment | Which wallet(s) receive this miner's rewards |
| Network Priority | Which networks this miner should mine |
| Difficulty | Custom difficulty settings |
| Power Mode | Performance/Efficiency/Eco modes |
| Schedule | Mining schedule (time-based on/off) |
| Alerts | Custom alert thresholds |

### 3.3 Miner Monitoring Dashboard
```
┌─────────────────────────────────────────────────────────────────────┐
│ MINER: X100 Picaxe ASIC                              [⚡ ONLINE]    │
├─────────────────────────────────────────────────────────────────────┤
│ Hashrate: 100 TH/s    │ Shares: 1,234/hr  │ Efficiency: 98.5%      │
│ Temperature: 65°C     │ Power: 3,250W     │ Uptime: 99.2%          │
├─────────────────────────────────────────────────────────────────────┤
│ Network: Litecoin     │ Wallet: Cold Storage (Wallet A)            │
│ Difficulty: Auto      │ Last Share: 2 seconds ago                  │
├─────────────────────────────────────────────────────────────────────┤
│ [📊 Stats] [⚙️ Settings] [💰 Earnings] [📜 Logs] [🔔 Alerts]        │
└─────────────────────────────────────────────────────────────────────┘
```

### 3.4 Miner Database Schema Enhancement
```sql
-- Enhanced miner configuration
ALTER TABLE miners ADD COLUMN custom_name VARCHAR(100);
ALTER TABLE miners ADD COLUMN miner_group_id INTEGER REFERENCES miner_groups(id);
ALTER TABLE miners ADD COLUMN power_mode VARCHAR(20) DEFAULT 'performance';
ALTER TABLE miners ADD COLUMN custom_difficulty DECIMAL(20,8);
ALTER TABLE miners ADD COLUMN schedule_enabled BOOLEAN DEFAULT false;
ALTER TABLE miners ADD COLUMN schedule_json JSONB; -- {"mon": {"start": "08:00", "end": "22:00"}, ...}

-- Miner groups for organization
CREATE TABLE miner_groups (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id),
    name VARCHAR(100) NOT NULL,
    description TEXT,
    color VARCHAR(7), -- Hex color for UI
    created_at TIMESTAMP DEFAULT NOW()
);

-- Per-miner wallet assignments
CREATE TABLE miner_wallet_assignments (
    id SERIAL PRIMARY KEY,
    miner_id INTEGER REFERENCES miners(id),
    wallet_id INTEGER REFERENCES wallets(id),
    allocation_percent DECIMAL(5,2) NOT NULL,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(miner_id, wallet_id)
);
```

---

## 4. Multi-Network Mining Support

### 4.1 Supported Networks
- **Litecoin (LTC)** - Scrypt
- **BlockDAG (BDAG)** - Custom algorithm
- **Future**: Bitcoin, Dogecoin, Kaspa, etc.

### 4.2 Network Configuration Per Miner
```
┌─────────────────────────────────────────────────────────────┐
│ NETWORK ASSIGNMENT: X100 ASIC                               │
├─────────────────────────────────────────────────────────────┤
│ ☑ Litecoin    Priority: 1    Allocation: 70%               │
│ ☑ BlockDAG    Priority: 2    Allocation: 30%               │
│ ☐ Dogecoin    (Not configured)                             │
└─────────────────────────────────────────────────────────────┘
```

### 4.3 Smart Network Switching
- **Profitability-Based**: Auto-switch to most profitable network
- **Manual Override**: User can force specific network
- **Failover**: Automatic failover if primary network unavailable

---

## 5. Payout Management

### 5.1 Payout Settings
- **Minimum Threshold**: Per-wallet minimum before payout
- **Payout Schedule**: Daily, weekly, on-demand, or threshold-based
- **Fee Preferences**: Priority fee levels for faster confirmations

### 5.2 Payout History & Tracking
- Complete transaction history with blockchain links
- Export to CSV/PDF for accounting
- Tax reporting helpers

---

## 6. User Dashboard Architecture

### 6.1 Dashboard Sections
```
┌─────────────────────────────────────────────────────────────────────┐
│ USER DASHBOARD                                        [Reid Davis]  │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐              │
│  │ 💰 EARNINGS  │  │ ⛏️ MINERS    │  │ 📊 HASHRATE  │              │
│  │   0.0234 LTC │  │   6 Active   │  │   125 TH/s   │              │
│  │   +12% today │  │   0 Offline  │  │   +5% avg    │              │
│  └──────────────┘  └──────────────┘  └──────────────┘              │
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │ QUICK ACTIONS                                                │   │
│  │ [➕ Add Miner] [💳 Add Wallet] [⚙️ Settings] [📈 Reports]    │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                     │
│  NAVIGATION TABS:                                                   │
│  [Overview] [Miners] [Wallets] [Payouts] [Networks] [Settings]     │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### 6.2 Tab Structure

#### Overview Tab
- Real-time earnings summary
- Active miners status
- Recent payouts
- Network status
- Alerts & notifications

#### Miners Tab
- List all miners with status
- Add/edit/remove miners
- Group management
- Per-miner settings
- Bulk actions

#### Wallets Tab
- All wallets with balances
- Add/edit/remove wallets
- Allocation management
- Active/inactive toggle
- Per-miner assignments

#### Payouts Tab
- Pending payouts
- Payout history
- Manual payout requests
- Export options

#### Networks Tab
- Available networks
- Per-network settings
- Profitability comparison
- Network health status

#### Settings Tab
- Profile settings
- Security settings
- Notification preferences
- API key management
- Account tier info

---

## 7. API Endpoints Required

### 7.1 Wallet Management
```
GET    /api/v1/user/wallets                    # List all wallets
POST   /api/v1/user/wallets                    # Add new wallet
PUT    /api/v1/user/wallets/:id                # Update wallet
DELETE /api/v1/user/wallets/:id                # Remove wallet
PUT    /api/v1/user/wallets/:id/status         # Toggle active/inactive
PUT    /api/v1/user/wallets/:id/allocation     # Update allocation %
GET    /api/v1/user/wallets/allocations        # Get all allocations
PUT    /api/v1/user/wallets/allocations/bulk   # Bulk update allocations
```

### 7.2 Miner Management
```
GET    /api/v1/user/miners                     # List all miners
POST   /api/v1/user/miners                     # Register miner
PUT    /api/v1/user/miners/:id                 # Update miner settings
DELETE /api/v1/user/miners/:id                 # Remove miner
PUT    /api/v1/user/miners/:id/wallet          # Assign wallet to miner
GET    /api/v1/user/miners/:id/stats           # Get miner statistics
GET    /api/v1/user/miners/:id/shares          # Get share history
PUT    /api/v1/user/miners/:id/network         # Set network preference
```

### 7.3 Miner Groups
```
GET    /api/v1/user/miner-groups               # List groups
POST   /api/v1/user/miner-groups               # Create group
PUT    /api/v1/user/miner-groups/:id           # Update group
DELETE /api/v1/user/miner-groups/:id           # Delete group
PUT    /api/v1/user/miners/:id/group           # Assign miner to group
```

### 7.4 Payouts
```
GET    /api/v1/user/payouts                    # Payout history
GET    /api/v1/user/payouts/pending            # Pending payouts
POST   /api/v1/user/payouts/request            # Request manual payout
PUT    /api/v1/user/payouts/settings           # Update payout settings
```

---

## 8. Implementation Phases

### Phase 1: Wallet Allocation Enhancement (Priority - Bug Reports #7, #8, #9)
1. Add wallet active/inactive toggle
2. Implement auto-balance allocation
3. Add per-miner wallet assignment
4. Update UI for new allocation controls

### Phase 2: Enhanced Miner Management
1. Add miner naming and grouping
2. Implement per-miner settings
3. Add miner scheduling
4. Enhanced monitoring dashboard

### Phase 3: Multi-Network Support
1. Network selection per miner
2. Profitability-based switching
3. Network-specific wallet assignments

### Phase 4: Advanced Features
1. API key management for external tools
2. Advanced reporting and exports
3. Mobile app integration
4. Webhook notifications

---

## 9. Database Migration Plan

### Migration 022: Wallet Enhancements
```sql
-- 022_wallet_enhancements.up.sql
ALTER TABLE wallets ADD COLUMN IF NOT EXISTS status VARCHAR(20) DEFAULT 'active';
ALTER TABLE wallets ADD COLUMN IF NOT EXISTS wallet_type VARCHAR(20) DEFAULT 'hot';
ALTER TABLE wallets ADD COLUMN IF NOT EXISTS label VARCHAR(100);
ALTER TABLE wallets ADD COLUMN IF NOT EXISTS is_locked BOOLEAN DEFAULT false;
ALTER TABLE wallets ADD COLUMN IF NOT EXISTS min_payout_threshold DECIMAL(18,8) DEFAULT 0;

-- Create miner-wallet assignments table
CREATE TABLE IF NOT EXISTS miner_wallet_assignments (
    id SERIAL PRIMARY KEY,
    miner_id INTEGER REFERENCES miners(id) ON DELETE CASCADE,
    wallet_id INTEGER REFERENCES wallets(id) ON DELETE CASCADE,
    allocation_percent DECIMAL(5,2) NOT NULL DEFAULT 100,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(miner_id, wallet_id)
);

-- Create miner groups table
CREATE TABLE IF NOT EXISTS miner_groups (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    color VARCHAR(7) DEFAULT '#00d4ff',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Add group reference to miners
ALTER TABLE miners ADD COLUMN IF NOT EXISTS miner_group_id INTEGER REFERENCES miner_groups(id);
ALTER TABLE miners ADD COLUMN IF NOT EXISTS custom_name VARCHAR(100);
```

---

## 10. UI Component Structure

```
src/components/
├── dashboard/
│   ├── UserDashboard.tsx          # Main dashboard container
│   ├── DashboardOverview.tsx      # Overview tab
│   ├── EarningsSummary.tsx        # Earnings widget
│   └── QuickActions.tsx           # Quick action buttons
├── miners/
│   ├── MinerList.tsx              # List all miners
│   ├── MinerCard.tsx              # Individual miner card
│   ├── MinerSettings.tsx          # Per-miner settings
│   ├── MinerGroupManager.tsx      # Group management
│   └── MinerWalletAssignment.tsx  # Wallet assignment UI
├── wallets/
│   ├── WalletManager.tsx          # Main wallet management
│   ├── WalletCard.tsx             # Individual wallet card
│   ├── AllocationSlider.tsx       # Allocation percentage control
│   ├── WalletStatusToggle.tsx     # Active/inactive toggle
│   └── BulkAllocationEditor.tsx   # Edit all allocations at once
├── payouts/
│   ├── PayoutHistory.tsx          # Payout history list
│   ├── PendingPayouts.tsx         # Pending payouts
│   └── PayoutSettings.tsx         # Payout preferences
└── networks/
    ├── NetworkSelector.tsx        # Network selection
    ├── NetworkStats.tsx           # Per-network statistics
    └── ProfitabilityChart.tsx     # Network profitability comparison
```

---

## Summary

This comprehensive system provides users with:
- ✅ Full wallet management with active/inactive toggles
- ✅ Auto-balancing allocation percentages
- ✅ Per-miner wallet assignments
- ✅ Miner grouping and organization
- ✅ Multi-network support
- ✅ Complete payout control
- ✅ Industry-standard dashboard experience

The implementation follows best practices for mining pool management and provides an elite-level user experience comparable to top-tier mining pools like F2Pool, Antpool, and NiceHash.
