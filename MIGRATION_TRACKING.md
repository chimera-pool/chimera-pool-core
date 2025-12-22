# Chimeria Pool Migration & Feature Tracking

## Current Database Schema Version: 007

## Applied Migrations

| Migration | File | Description | Status |
|-----------|------|-------------|--------|
| 001 | `001_initial_schema.up.sql` | Initial schema: users, miners, shares, blocks, payouts | ✅ Applied |
| 002 | `002_community_monitoring.up.sql` | Community features, monitoring, user_profiles, badges | ✅ Applied |
| 003 | `003_roles_channels.up.sql` | Role system (user/moderator/admin/super_admin), channels, categories | ✅ Applied |
| 004 | `004_user_wallets.up.sql` | Multi-wallet support, wallet_address_history, payout splitting | ✅ Applied |
| 005 | `005_seed_community_categories.up.sql` | Seed community categories and channels | ✅ Applied |
| 006 | `006_bug_reports.up.sql` | Bug reporting system with attachments, comments, email notifications | ✅ Applied |
| 007 | `007_equipment_management.up.sql` | Equipment management, metrics history, payout splits, pool stats cache | ✅ Applied |

## Recent Feature Changes

### December 22, 2025 - Frontend Dead Code Cleanup & CI Fix

**Codebase Audit Session - App.tsx Cleanup**

**Phase 1 - Dead Code Removed from `src/App.tsx`**:
- ✅ `UserDashboard` duplicate (-215 lines) → extracted to `components/dashboard/UserDashboard.tsx`
- ✅ `MiningGraphs` duplicate (-256 lines) → extracted to `components/charts/MiningGraphs.tsx`
- ✅ `GlobalMinerMap` duplicate (-232 lines) → extracted to `components/maps/GlobalMinerMap.tsx`
- ✅ `WalletManager` duplicate (-365 lines) → extracted to `components/wallet/WalletManager.tsx`
- ✅ Associated style objects (`dashStyles`, `mapStyles`, `walletStyles`) removed

**Phase 2 - Additional Dead Code Removed**:
- ✅ `CommunitySection` function (-312 lines) - unused, replaced by `CommunityPage`
- ✅ `commStyles` object (-72 lines) - only used by `CommunitySection`
- ✅ `AuthModal` function (-163 lines) - unused, replaced by `AuthModalLazy`
- ✅ `AuthModalProps` interface removed

**Final Results**:
- `App.tsx`: 6,855 → 4,914 lines (-1,941 lines, -28%)
- Bundle size: 234.67 kB → 195.82 kB (-38.85 kB, -17%)
- Frontend build: ✅ Passes
- Go tests: ✅ All 24 packages pass

**Backend Fixes**:
- Added `UpdateCategoryRequest` type to `internal/community/models.go`
- Fixed unreachable code in `internal/stratum/server.go` (cleanupConnection defer)

**Git Commits**:
- `341dec4` - Phase 1 dead code cleanup
- `ee153d7` - Phase 2 dead code cleanup (CommunitySection, AuthModal)

**Remaining Future Work** (Active Components - Need Extraction):
1. `EquipmentPage` (~960 lines) → `components/equipment/EquipmentPage.tsx`
2. `CommunityPage` (~600 lines) → `components/community/CommunityPage.tsx`
3. `AdminPanel` (~900 lines) → `components/admin/AdminPanel.tsx`

**Audit Documentation**: Updated `CODEBASE_AUDIT.md` with current status

---

### December 21, 2025 - Equipment Management System

**Migration**: `007_equipment_management.up.sql`

**Database Tables Created**:
- `equipment` - Mining hardware tracking (ASIC, GPU, CPU, FPGA, BlockDAG X30/X100)
- `equipment_metrics_history` - Time-series performance data
- `payout_splits` - Per-equipment multi-wallet distribution
- `user_wallets` - User wallet address management
- `pool_stats_cache` - Cached pool-wide statistics
- `equipment_alerts` - Notification system for equipment issues

**Key Features**:
- Hardware type classification (asic, gpu, cpu, fpga, blockdag_x30, blockdag_x100)
- Equipment status tracking (online, offline, mining, idle, error, maintenance)
- Real-time metrics: hashrate, temperature, power usage, latency
- Stratum V1/V2 connection type tracking
- Per-equipment payout splitting to multiple wallets
- Automatic pool stats cache with trigger-based updates
- Alert system for offline, error, high temp, performance drops

**Indexes Created**:
- `idx_equipment_user_id` - Fast user lookups
- `idx_equipment_status` - Status filtering
- `idx_equipment_type` - Type filtering
- `idx_equipment_worker_name` - Worker name lookups
- `idx_metrics_equipment_time` - Time-series queries
- `idx_payout_splits_equipment` - Payout lookups

---

### December 20, 2025 - Bug Reporting System

**Commit**: `f8ae27f` - "Add bug reporting system with email notifications"

**Database Tables Created** (`migrations/006_bug_reports.up.sql`):
- `bug_reports` - Main bug report storage with status, priority, category
- `bug_attachments` - Screenshots and file attachments (stored as BYTEA)
- `bug_comments` - Threaded conversation with internal notes support
- `bug_email_notifications` - Email tracking for notifications
- `bug_subscribers` - Users subscribed to bug updates

**Backend API Endpoints** (`cmd/api/main.go`):
```
# User endpoints (authenticated)
POST /api/v1/bugs                    - Submit bug report
GET  /api/v1/bugs                    - List user's bug reports
GET  /api/v1/bugs/:id                - Get bug details with comments
POST /api/v1/bugs/:id/comments       - Add comment to bug
POST /api/v1/bugs/:id/attachments    - Upload attachment
POST /api/v1/bugs/:id/subscribe      - Subscribe to updates
DELETE /api/v1/bugs/:id/subscribe    - Unsubscribe

# Admin endpoints
GET  /api/v1/admin/bugs              - List all bugs (filterable)
GET  /api/v1/admin/bugs/:id          - Get bug with internal comments
PUT  /api/v1/admin/bugs/:id/status   - Update status (sends email)
PUT  /api/v1/admin/bugs/:id/priority - Update priority
PUT  /api/v1/admin/bugs/:id/assign   - Assign to admin
POST /api/v1/admin/bugs/:id/comments - Add comment (can be internal)
DELETE /api/v1/admin/bugs/:id        - Delete bug report
```

**Frontend Changes** (`src/App.tsx`):
- 🐛 Bug button in header - Opens bug report modal
- 📋 My Bugs button - View and track submitted bugs
- Bug report modal with category, description, steps to reproduce
- Bug details view with comment thread
- Status and priority badges with color coding

**Email Notifications**:
- New bug → Admins notified
- Status change → Subscribers notified
- New comment → Subscribers notified (except commenter)
- Bug assigned → Assignee notified

---

### December 20, 2025 - Account Settings with Password Change & Forgot Password

**Commits**:
- `a349b50` - "Add PUT /user/profile endpoint for username and payout_address updates"
- `001c731` - "Add password change feature with tabbed profile modal"
- `24d4cea` - "Add forgot password feature to Security tab"

**Backend Changes** (`cmd/api/main.go`):
- Added `PUT /api/v1/user/profile` route - Update username/payout_address
- Added `PUT /api/v1/user/password` route - Change password (authenticated)
- Added `handleChangePassword` function with bcrypt verification

**Frontend Changes** (`src/App.tsx`):
- Tabbed Account Settings modal (Profile / Security tabs)
- Profile tab: Edit username, payout address
- Security tab: Change password, Forgot password email trigger

**API Endpoints**:
```
PUT /api/v1/user/password (authenticated)
{ "current_password": "...", "new_password": "..." }

POST /api/v1/auth/forgot-password (public)
{ "email": "user@email.com" }
```

**Next Session TODO**: Configure GoDaddy SMTP for email delivery

---

### December 20, 2025 - User Profile Editing (Initial)

**Commit**: `a349b50` - "Add PUT /user/profile endpoint for username and payout_address updates"

**Files Modified**:
- `cmd/api/main.go`
  - Added `PUT /api/v1/user/profile` route (line 90)
  - Added `handleUpdateUserProfile` function (lines 486-567)
  - Updated `handleUserProfile` to return `payout_address` (lines 457-484)

**Frontend** (already existed in `src/App.tsx`):
- Click handler on username (line 162)
- Profile modal with username/payout_address fields (lines 191-239)
- `handleUpdateProfile` function (lines 110-130)

**API Endpoint**:
```
PUT /api/v1/user/profile
Authorization: Bearer <token>
Content-Type: application/json

{
  "username": "new_username",      // optional, 3-50 chars
  "payout_address": "0x..."        // optional
}

Response:
{
  "message": "Profile updated successfully",
  "user": {
    "user_id": 1,
    "username": "new_username",
    "email": "user@email.com",
    "payout_address": "0x...",
    "is_admin": false
  }
}
```

**Database Columns Used**:
- `users.username` (VARCHAR 50, UNIQUE)
- `users.payout_address` (VARCHAR 255, nullable)

---

## Rollback Procedures

### Rollback Migration 005
```sql
-- From migrations/005_seed_community_categories.down.sql
DELETE FROM channel_messages;
DELETE FROM channels;
DELETE FROM channel_categories;
```

### Rollback Migration 004
```sql
-- From migrations/004_user_wallets.down.sql
DROP TABLE IF EXISTS user_wallets CASCADE;
DROP TABLE IF EXISTS wallet_address_history CASCADE;
ALTER TABLE payouts DROP COLUMN IF EXISTS wallet_id;
```

### Rollback Migration 003
```sql
-- From migrations/003_roles_channels.down.sql
DROP TABLE IF EXISTS moderation_log CASCADE;
DROP TABLE IF EXISTS channel_messages CASCADE;
DROP TABLE IF EXISTS channels CASCADE;
DROP TABLE IF EXISTS channel_categories CASCADE;
ALTER TABLE users DROP COLUMN IF EXISTS role;
```

---

## Feature Tracking

### Completed Features
- [x] User registration/login with JWT
- [x] Pool statistics dashboard
- [x] Miner management
- [x] Multi-wallet support with payout splitting
- [x] Community channels and messaging
- [x] Admin/Moderator role system
- [x] User badges and achievements
- [x] Stratum V2 hybrid protocol
- [x] User profile editing (username, payout_address)

### In Progress
- [ ] None currently

### Planned Features
- [ ] Email verification for profile changes
- [ ] Profile avatar upload
- [ ] Two-factor authentication

---

## Key Database Tables

### users
```sql
id, username, email, password_hash, payout_address, role, is_admin, 
is_active, created_at, updated_at
```

### user_profiles
```sql
user_id, avatar_url, bio, country, country_code, show_earnings, 
show_country, reputation, forum_post_count, lifetime_hashrate, online_status
```

### user_wallets
```sql
id, user_id, address, label, percentage, is_primary, is_active, 
created_at, updated_at
```

### channels
```sql
id, category_id, name, description, slug, channel_type, is_active,
created_by, created_at, updated_at
```

---

## Environment Variables

Key variables in `.env`:
- `DATABASE_URL` - PostgreSQL connection
- `JWT_SECRET` - JWT signing key
- `BLOCKDAG_RPC_URL` - BlockDAG node RPC
- `SMTP_*` - Email configuration
- `FRONTEND_URL` - https://206.162.80.230

---

## Testing Commands

```powershell
# Go tests
go test ./... -v

# Build API
go build -o api.exe ./cmd/api

# React tests
npm test -- --watchAll=false
```
