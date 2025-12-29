# Chimera Pool Codebase Audit & Segmentation Plan

**Date:** December 28, 2025  
**Status:** Ready for Execution

---

## Executive Summary

The codebase is **85% elite quality** with excellent patterns in newer components. However, **4 critical monolith files** require segmentation to meet elite coding standards.

---

## Critical Files Requiring Segmentation

### 🔴 CRITICAL PRIORITY

| File | Size | Lines | Issues | Proposed Solution |
|------|------|-------|--------|-------------------|
| **AdminPanel.tsx** | 138.4 KB | 2,542 | 7 tabs in one file, 30+ useState hooks | Split into 7 tab components |
| **App.tsx** | 91.2 KB | 1,636 | Auth, modals, routing, 25+ unused imports | Extract providers, modals, layouts |
| **EquipmentPage.tsx** | 55 KB | 1,062 | 3 tabs, multiple modals | Split into tab components |
| **CommunityPage.tsx** | 51.9 KB | 1,039 | Chat, forums, leaderboard combined | Split into feature components |

### 🟠 HIGH PRIORITY

| File | Size | Lines | Issues |
|------|------|-------|--------|
| **MiningGraphs.tsx** | 24.5 KB | ~500 | Multiple chart types in one file |
| **WalletManager.tsx** | 20.5 KB | ~400 | Could be split into wallet list/detail |

### 🟢 GOOD PATTERNS (No Changes Needed)

| File | Size | Lines | Notes |
|------|------|-------|-------|
| **MiningInstructionsLitecoin.tsx** | 8.5 KB | 219 | ✅ Perfect size, ISP compliant |
| **MinerStatusMonitor.tsx** | 5.5 KB | 139 | ✅ Well-segmented |
| **ChartSlot.tsx** | 6.2 KB | ~150 | ✅ Good single responsibility |
| **GrafanaEmbed.tsx** | 4.8 KB | ~100 | ✅ Focused component |

---

## Detailed Segmentation Plan

### 1. AdminPanel.tsx (138.4 KB → ~8 files)

**Current Structure:**
- Users tab (user management)
- Stats tab (pool statistics)
- Algorithm tab (mining algorithm config)
- Network tab (blockchain network config)
- Roles tab (user role management)
- Bugs tab (bug report management)
- Miners tab (miner management)

**Proposed Structure:**
```
src/components/admin/
├── AdminPanel.tsx (~200 lines - shell/router only)
├── AdminLayout.tsx (~100 lines - shared layout)
├── tabs/
│   ├── AdminUsersTab.tsx (existing pattern)
│   ├── AdminStatsTab.tsx (already exists ✅)
│   ├── AdminAlgorithmTab.tsx (new)
│   ├── AdminNetworkTab.tsx (new)
│   ├── AdminRolesTab.tsx (new)
│   ├── AdminBugsTab.tsx (new)
│   └── AdminMinersTab.tsx (new)
├── hooks/
│   ├── useAdminUsers.ts (user CRUD logic)
│   ├── useAdminBugs.ts (bug management logic)
│   └── useAdminNetwork.ts (network config logic)
└── interfaces/
    └── IAdminPanel.ts (shared interfaces)
```

### 2. App.tsx (91.2 KB → ~6 files)

**Current Issues:**
- 25+ unused imports (visible in build warnings)
- Auth logic mixed with routing
- Modal state scattered across component
- Profile modal inline
- Bug report modal inline

**Proposed Structure:**
```
src/
├── App.tsx (~300 lines - clean shell)
├── AppLayout.tsx (~150 lines - header, nav, footer)
├── AppRoutes.tsx (~100 lines - view switching)
├── providers/
│   └── AppProviders.tsx (wraps all context providers)
├── modals/
│   ├── ProfileModal.tsx (extract from App)
│   ├── BugReportModal.tsx (extract from App)
│   └── MyBugsModal.tsx (extract from App)
└── hooks/
    └── useAppAuth.ts (auth logic extraction)
```

### 3. EquipmentPage.tsx (55 KB → ~5 files)

**Current Structure:**
- Equipment list/detail
- Wallets management
- Alerts configuration
- Settings modal
- Split management modal

**Proposed Structure:**
```
src/components/equipment/
├── EquipmentPage.tsx (~150 lines - container)
├── tabs/
│   ├── EquipmentListTab.tsx
│   ├── WalletsTab.tsx
│   └── AlertsTab.tsx
├── modals/
│   ├── EquipmentSettingsModal.tsx
│   └── SplitManagementModal.tsx
└── interfaces/
    └── IEquipment.ts
```

### 4. CommunityPage.tsx (51.9 KB → ~5 files)

**Current Structure:**
- Channel sidebar
- Chat interface
- Forums section
- Leaderboard

**Proposed Structure:**
```
src/components/community/
├── CommunityPage.tsx (~150 lines - container)
├── ChannelSidebar.tsx
├── ChatInterface.tsx
├── ForumsSection.tsx
├── LeaderboardSection.tsx
└── interfaces/
    └── ICommunity.ts
```

---

## Execution Order

### Phase 1: Foundation Cleanup (Immediate)
1. ✅ Fix corrupted code in App.tsx
2. ✅ Add Litecoin mining instructions to home page
3. Remove unused imports from App.tsx
4. Extract interfaces to separate files

### Phase 2: AdminPanel Segmentation
1. Create AdminLayout component
2. Extract each tab to separate file
3. Extract hooks for data fetching
4. Update imports and test

### Phase 3: App.tsx Refactoring
1. Extract modals to separate files
2. Create AppLayout component
3. Create AppProviders wrapper
4. Clean up unused imports

### Phase 4: Secondary Components
1. EquipmentPage segmentation
2. CommunityPage segmentation
3. MiningGraphs cleanup (if time permits)

---

## Testing Strategy

- Maintain 30/30 test suites passing
- Add tests for new components as created
- Run full test suite after each phase
- Deploy to staging after each phase completion

---

## Time Estimates

| Phase | Effort | Priority |
|-------|--------|----------|
| Phase 1 | 1 hour | IMMEDIATE |
| Phase 2 | 3-4 hours | HIGH |
| Phase 3 | 2-3 hours | HIGH |
| Phase 4 | 3-4 hours | MEDIUM |

**Total Estimated:** 9-12 hours

---

## Success Criteria

- [ ] No file exceeds 500 lines
- [ ] Each component has single responsibility
- [ ] All tests passing (30/30 suites)
- [ ] No unused imports
- [ ] Interfaces extracted to separate files
- [ ] Build completes with zero errors
