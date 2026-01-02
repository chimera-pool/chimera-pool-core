# Chimeria Pool Security Remediation Roadmap

**Created:** January 2, 2026  
**Based on:** Security Assessment Round 2 Retest Report  
**Total Vulnerabilities:** 34 (18 Critical, 9 High, 6 Medium, 1 Low)  
**Fixed in Previous Round:** 3 (Authorization headers, Token expiration, Admin endpoint protection)

---

## Priority 1: IMMEDIATE ACTION (24-48 hours)

### 1.1 SQL Injection in Wallet Address Field
- **Severity:** CRITICAL (CVSS 9.8)
- **Status:** [ ] PENDING
- **Location:** `/api/v1/user/profile` endpoint
- **Issue:** Wallet address field accepts SQL injection payloads without sanitization
- **Fix:** Implement parameterized queries, input validation
- **Files to modify:**
  - `cmd/api/main.go` - Profile update handler
  - Add wallet validation utility

### 1.2 XSS in Wallet Address Field
- **Severity:** CRITICAL (CVSS 8.8)
- **Status:** [ ] PENDING
- **Location:** `/api/v1/user/profile` endpoint
- **Issue:** Script tags and XSS payloads accepted in wallet address
- **Fix:** Input sanitization, output encoding, CSP headers
- **Files to modify:**
  - `cmd/api/main.go` - Profile update handler
  - Frontend wallet display components

### 1.3 Zero Difficulty Share Acceptance
- **Severity:** CRITICAL (CVSS 9.8)
- **Status:** [ ] PENDING
- **Location:** Stratum server share validation
- **Issue:** Pool accepts shares with diff 0.000, 100% acceptance rate
- **Fix:** Implement minimum difficulty validation, reject zero-diff shares
- **Files to modify:**
  - `cmd/stratum/main.go` - Share submission handler
  - `internal/stratum/` - Difficulty validation logic

### 1.4 No Wallet Address Validation
- **Severity:** CRITICAL (CVSS 9.1)
- **Status:** [ ] PENDING
- **Location:** Profile update, wallet management
- **Issue:** Any string accepted as wallet address (e.g., "INVALID123FAKE")
- **Fix:** Validate LTC address format (starts with L/M/ltc1), checksum, length
- **Files to modify:**
  - `cmd/api/main.go` - All wallet-related handlers
  - Create `internal/validation/wallet.go`

---

## Priority 2: URGENT (Week 1)

### 2.1 Hashrate Inflation (25,675x)
- **Severity:** CRITICAL (CVSS 9.1)
- **Status:** [ ] PENDING
- **Issue:** Dashboard shows 693.25 MH/s vs actual 27 kH/s
- **Fix:** Fix hashrate calculation algorithm, add sanity checks
- **Files to modify:**
  - `cmd/api/main.go` - Stats calculation
  - `internal/api/pool_service.go`

### 2.2 Share Counting Discrepancy
- **Severity:** HIGH (CVSS 7.4)
- **Status:** [ ] PENDING
- **Issue:** Dashboard shows 0 valid/invalid shares despite 63,700+ submitted
- **Fix:** Fix frontend display logic to show backend data correctly
- **Files to modify:**
  - Frontend stats components
  - API response mapping

### 2.3 JWT in localStorage
- **Severity:** HIGH (CVSS 7.5)
- **Status:** [ ] PENDING
- **Issue:** Tokens stored in localStorage, accessible to XSS
- **Fix:** Migrate to HTTP-only cookies
- **Files to modify:**
  - `cmd/api/main.go` - Auth handlers
  - Frontend auth logic

### 2.4 User Enumeration via API
- **Severity:** MEDIUM (CVSS 5.3)
- **Status:** [ ] PENDING
- **Issue:** `/api/v1/user/profile?user_id=1` allows enumeration
- **Fix:** Enforce user can only access own data, return 403 for others
- **Files to modify:**
  - `cmd/api/main.go` - Profile handler

---

## Priority 3: IMPORTANT (Week 2-4)

### 3.1 Password Complexity
- **Severity:** MEDIUM (CVSS 5.3)
- **Status:** [ ] PENDING
- **Issue:** Only 8-char minimum, no complexity requirements
- **Fix:** Require uppercase, lowercase, number, special char
- **Files to modify:**
  - `cmd/api/main.go` - Registration/password change handlers

### 3.2 Frontend Performance
- **Severity:** LOW (CVSS 3.1)
- **Status:** [ ] PENDING
- **Issue:** INP 312ms (should be <200ms), browser slowdown
- **Fix:** Optimize React rendering, reduce polling, code splitting
- **Files to modify:**
  - Frontend components with heavy rendering

### 3.3 Session Persistence
- **Severity:** MEDIUM
- **Status:** [ ] PENDING
- **Issue:** Tokens persist after browser close
- **Fix:** Implement session vs persistent token option
- **Files to modify:**
  - Frontend auth logic

---

## User Bug Reports (BUG-000007 to BUG-000010)

### BUG-000007: Wallet Allocation Switching
- **Status:** [ ] PENDING
- **Issue:** Must reduce one wallet before increasing another
- **Fix:** Auto-adjust other wallets when one is changed

### BUG-000008: Wallet Management
- **Status:** [ ] PENDING
- **Issue:** Can't change to 1 wallet without deleting second, 99%/1% limit
- **Fix:** Add active/inactive toggle for wallets, allow 100% single wallet

### BUG-000009: Assign Wallet to Miner (Feature Request)
- **Status:** [ ] PENDING
- **Issue:** Want to assign specific wallets to specific miners
- **Fix:** Add miner-wallet association feature

### BUG-000010: Equipment Display Issue
- **Status:** [ ] PENDING
- **Issue:** Showing X30 in London incorrectly, log issues
- **Fix:** Investigate equipment assignment and geolocation

---

## Already Fixed (From Previous Assessment)

- [x] API Authorization Headers - 401 for unauthenticated requests
- [x] Token Expiration - 24-hour expiry implemented
- [x] Admin Endpoint Protection - 403 for non-admin users
- [x] Payout Manipulation Endpoints - Removed/secured (404)
- [x] Share-Related API Endpoints - Protected (404)

---

## Estimated Timeline

| Phase | Duration | Focus |
|-------|----------|-------|
| Immediate | 24-48 hrs | SQL Injection, XSS, Wallet Validation, Zero Difficulty |
| Week 1 | 5 days | Hashrate, Share Display, JWT Cookies, User Enumeration |
| Week 2-3 | 10 days | Password Policy, Performance, User Bug Reports |
| Week 4 | 5 days | Final Audit, Testing, Documentation |

**Target Completion:** End of January 2026
