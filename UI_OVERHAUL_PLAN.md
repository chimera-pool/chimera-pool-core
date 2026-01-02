# Chimera Pool UI Overhaul Master Plan

**Created**: December 26, 2025  
**Status**: Phase 1.7 Complete (Homepage Redesign)  
**Brand Colors**: Gold (#D4A84B), Deep Purple (#2D1F3D), Dark (#1A0F1E), Accent Purple (#7B5EA7)

---

## Executive Summary

This plan outlines the comprehensive UI overhaul for Chimera Pool mining software. The goal is to create a world-class user experience that:
- Attracts new miners through intuitive design and trust-building visuals
- Retains miners with clear, actionable data and professional aesthetics
- Follows the cyber-minimal theme established by the Chimera logo
- Uses psychology-driven chart prioritization for maximum engagement

---

## Design Principles

1. **Trust Through Transparency**: Show real-time data prominently (hashrate, earnings, uptime)
2. **Action-Oriented**: Every screen should have a clear primary action
3. **Progressive Disclosure**: Essential info first, details on demand
4. **Consistent Branding**: Gold accents, purple gradients, dark backgrounds throughout
5. **Mobile-First**: All views must work on mobile devices

---

## Phase 1: Foundation (COMPLETED - 1.0-1.7)

### 1.1 ✅ Core Infrastructure
- Grafana integration with chart selectors
- Responsive grid system (mobile single-column, desktop 2x2)
- Chart crash fix (removed polling, added kiosk mode)

### 1.2 ✅ Homepage Stat Cards
- Live indicators on Active Miners and Pool Hashrate
- Hover animations with gold glow
- Text shadow effects on values

### 1.3 ✅ Navigation Enhancement
- Gold gradient on active tab with shadow
- Hover underline animation
- Consistent color transitions

---

## Phase 2: Homepage Polish (IN PROGRESS)

### 2.1 ✅ Pool Mining Statistics Section (Completed Jan 1, 2026)
**Priority**: HIGH (First thing users see after stats)
- [x] Redesign chart card headers with better visual hierarchy
- [x] Add chart category icons (hashrate = ⚡, workers = 👷, shares = 📊, earnings = 💰)
- [x] Improve dropdown selector styling to match brand
- [x] Add subtle loading shimmer animation
- [ ] Implement chart tooltips explaining each metric

### 2.2 ✅ Pool Monitoring Section (Completed Jan 1, 2026)
- [x] Redesign node health indicators with animated status dots
- [x] Style Grafana dashboard links as premium cards
- [x] Add hover effects to monitoring links

### 2.3 ✅ Your Mining Dashboard Section (Completed Jan 1, 2026)
- [x] Redesign empty state with compelling call-to-action
- [x] Add animated "connect miner" illustration
- [x] Style links as branded buttons

### 2.4 ✅ Multi-Wallet Payout Settings (Completed Jan 1, 2026)
- [x] Enhance wallet card design
- [x] Add progress visualization for allocation
- [x] Improve "+ Add Wallet" button prominence

### 2.5 ✅ Global Miner Network Map (Completed Jan 2, 2026)
- [x] Enhance map styling with brand colors
- [x] Add animated connection lines
- [x] Style country/continent stats cards with hover effects

### 2.6 ✅ Connect Your Miner Section (Completed Jan 1, 2026)
- [x] Redesign step indicators with brand styling
- [x] Enhance hardware type cards (ASIC, GPU, CPU)
- [x] Improve code block styling with copy buttons
- [x] Add visual protocol badges

### 2.7 ✅ Pool Information Footer Cards (Completed Jan 2, 2026)
- [x] Apply consistent card styling
- [x] Add hover effects
- [x] Enhance icon presentation

---

## Phase 3: Equipment Control Center (COMPLETED - Jan 2, 2026)

### 3.1 ✅ Stats Overview Cards
- [x] Apply enhanced stat card styling from homepage
- [x] Add live indicators for real-time metrics
- [x] Implement hover effects with transitions
- [x] Enhanced shadows and visual hierarchy

### 3.2 ✅ Equipment List Cards
- [x] Redesign equipment cards with brand styling
- [x] Add status indicator colors (mining=green, offline=gray)
- [x] Enhance metric display grid with improved spacing
- [x] Add equipment-specific icons (⚡ ASIC, 🎮 GPU, 💻 CPU)

### 3.3 ✅ Tab Navigation
- [x] Style tabs to match main navigation
- [x] Add tab transition animations
- [x] Gold accent on active tab with text shadow

### 3.4 ✅ Equipment Detail Views
- [x] Create expanded equipment view with sections
- [x] Add performance charts per equipment (hashrate, temp, power)
- [x] Design alert configuration UI with settings modal
- [x] Uptime/downtime tracking display

---

## Phase 4: Community Hub

### 4.1 Channel Sidebar
- [ ] Redesign channel list with brand colors
- [ ] Add animated online indicator
- [ ] Style category collapsible headers
- [ ] Enhance channel icons

### 4.2 Chat Interface
- [ ] Style message bubbles with user colors
- [ ] Add message timestamps styling
- [ ] Design mention highlighting
- [ ] Style input field and send button

### 4.3 Forums Tab
- [ ] Design thread list cards
- [ ] Style thread detail view
- [ ] Add reaction/voting UI

### 4.4 Leaderboard Tab
- [ ] Design leaderboard table with rankings
- [ ] Add trophy icons for top positions
- [ ] Style metric columns

---

## Phase 5: Admin Panel

### 5.1 Panel Header & Navigation
- [ ] Redesign tab navigation with brand styling
- [ ] Add tab icons consistency
- [ ] Style panel close button

### 5.2 Users Tab
- [ ] Enhance table styling with brand colors
- [ ] Redesign action buttons
- [ ] Add user avatar placeholders
- [ ] Style pagination

### 5.3 Stats Tab
- [ ] Add charts for admin metrics
- [ ] Design stats overview cards

### 5.4 Network Tab
- [ ] Style network configuration cards
- [ ] Add network switch UI

### 5.5 Bugs Tab
- [ ] Redesign bug report cards
- [ ] Add priority/status badges
- [ ] Style comment threads

### 5.6 Miners Tab
- [ ] Design connected miners list
- [ ] Add real-time hashrate display

---

## Phase 6: Modals & Forms

### 6.1 Authentication Modals
- [ ] Redesign login/register forms
- [ ] Style input fields with brand colors
- [ ] Add form validation styling
- [ ] Design password strength indicator

### 6.2 Profile Modal
- [ ] Enhance profile edit form
- [ ] Style security tab
- [ ] Add avatar upload UI

### 6.3 Bug Report Modal
- [ ] Redesign form layout
- [ ] Style category/priority dropdowns
- [ ] Add screenshot preview

### 6.4 Wallet Modal
- [ ] Design wallet add/edit form
- [ ] Style allocation slider
- [ ] Add wallet type icons

---

## Phase 7: Final Polish & Testing

### 7.1 Animations & Transitions
- [ ] Audit all transitions for consistency
- [ ] Add page transition animations
- [ ] Implement loading state animations

### 7.2 Responsive Audit
- [ ] Test all views on mobile (375px)
- [ ] Test tablet breakpoint (768px)
- [ ] Test desktop (1200px+)

### 7.3 Accessibility
- [ ] Verify color contrast ratios
- [ ] Test keyboard navigation
- [ ] Add ARIA labels

### 7.4 Performance
- [ ] Optimize CSS bundle
- [ ] Lazy load non-critical components
- [ ] Audit render performance

---

## Chart Psychology Prioritization

### Tier 1 - Trust Builders (Default Charts)
These build immediate trust with new visitors:
1. **Pool Hashrate** - Shows pool strength and reliability
2. **Active Workers** - Social proof of active mining community
3. **Blocks Found** - Proves the pool actually finds blocks
4. **Wallet Balance/Payouts** - Shows money flows to miners

### Tier 2 - Retention Charts (Dropdown Options)
Keep miners engaged and informed:
- Hashrate History - Track trends
- Worker Status Timeline - Monitor equipment
- Share Acceptance Rate - Quality indicator
- Payout History - Financial transparency

### Tier 3 - Power User Charts (Advanced)
For experienced miners:
- Alerts by Type
- System metrics
- Network difficulty
- Detailed share statistics

---

## Testing Strategy

Each phase includes:
1. **Unit Tests**: Component behavior verification
2. **Visual Tests**: Playwright screenshot comparisons
3. **Responsive Tests**: Mobile/tablet/desktop verification
4. **Interface Segregation**: Clean component boundaries

---

## Success Metrics

- Page load time < 2 seconds
- All interactive elements have hover states
- Consistent brand colors throughout
- Mobile usability score > 90
- Zero console errors
- All charts load within 3 seconds
