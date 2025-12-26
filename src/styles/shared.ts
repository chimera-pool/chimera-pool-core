import React from 'react';

// ============================================================================
// CHIMERA POOL - ELITE DESIGN SYSTEM
// Inspired by the mythological Chimera: Lion, Goat, and Serpent
// ============================================================================

// Color Palette - Extracted from Chimera Logo
export const colors = {
  // Primary brand colors
  primary: '#D4A84B',        // Lion Gold - main accent
  primaryDark: '#1A0F1E',    // Deep background
  primaryLight: '#E8C171',   // Light gold for hover states
  
  // Secondary brand colors
  secondary: '#7B5EA7',      // Mystic Violet - links, interactive
  secondaryDark: '#5A4580',  // Darker violet
  secondaryLight: '#9B7EC7', // Light violet
  
  // Accent colors from logo
  coral: '#C45C5C',          // Serpent Coral - alerts, live indicators
  coralLight: '#E07777',     // Light coral
  silver: '#B8B4C8',         // Goat Silver - secondary text
  
  // Semantic colors
  success: '#4ADE80',        // Green for positive states
  error: '#EF4444',          // Red for errors
  warning: '#FBBF24',        // Amber for warnings
  info: '#60A5FA',           // Blue for info
  
  // Background hierarchy (darkest to lightest)
  bgDeep: '#0D0811',         // Deepest background
  bgDark: '#1A0F1E',         // Primary dark background
  bgCard: '#2D1F3D',         // Card background (Chimera Purple)
  bgElevated: '#3A2850',     // Elevated surfaces
  bgInput: '#1F1428',        // Input fields
  bgHover: '#3A1F2E',        // Hover state (Deep Maroon)
  
  // Border colors
  border: '#4A2C5A',         // Standard border (Royal Purple)
  borderLight: '#5A3C6A',    // Light border
  borderGold: 'rgba(212, 168, 75, 0.3)', // Gold accent border
  
  // Text colors
  textPrimary: '#F0EDF4',    // Primary text (light)
  textSecondary: '#B8B4C8',  // Secondary text (Goat Silver)
  textMuted: '#7A7490',      // Muted text
  textGold: '#D4A84B',       // Gold accent text
};

// Common gradients - mythological theme
export const gradients = {
  // Main background gradient
  hero: 'linear-gradient(135deg, #2D1F3D 0%, #3A1F2E 50%, #1A0F1E 100%)',
  
  // Card gradient with glass effect
  card: 'linear-gradient(180deg, rgba(45, 31, 61, 0.8) 0%, rgba(26, 15, 30, 0.9) 100%)',
  cardHover: 'linear-gradient(180deg, rgba(58, 40, 80, 0.9) 0%, rgba(45, 31, 61, 0.95) 100%)',
  
  // Header gradient
  header: 'linear-gradient(135deg, #2D1F3D 0%, #3A1F2E 100%)',
  
  // Gold accent gradient (for buttons, highlights)
  gold: 'linear-gradient(135deg, #D4A84B 0%, #B8923A 100%)',
  goldHover: 'linear-gradient(135deg, #E8C171 0%, #D4A84B 100%)',
  
  // Purple accent gradient
  purple: 'linear-gradient(135deg, #7B5EA7 0%, #5A4580 100%)',
  
  // Coral accent gradient
  coral: 'linear-gradient(135deg, #C45C5C 0%, #A04545 100%)',
};

// Shadows and glows
export const shadows = {
  card: '0 4px 24px rgba(0, 0, 0, 0.4)',
  cardHover: '0 8px 32px rgba(0, 0, 0, 0.5), 0 0 0 1px rgba(212, 168, 75, 0.2)',
  elevated: '0 8px 32px rgba(0, 0, 0, 0.5)',
  modal: '0 24px 48px rgba(0, 0, 0, 0.6)',
  
  // Glow effects
  glowGold: '0 0 20px rgba(212, 168, 75, 0.3)',
  glowPurple: '0 0 20px rgba(123, 94, 167, 0.3)',
  glowCoral: '0 0 20px rgba(196, 92, 92, 0.3)',
  
  // Focus ring
  focusRing: '0 0 0 3px rgba(123, 94, 167, 0.4)',
};

// Animation timings
export const transitions = {
  fast: '150ms ease',
  normal: '200ms ease',
  slow: '300ms ease',
  spring: '300ms cubic-bezier(0.34, 1.56, 0.64, 1)',
};

// Shared base styles
export const baseStyles: { [key: string]: React.CSSProperties } = {
  // Modal overlay - glassmorphism effect
  modalOverlay: {
    position: 'fixed',
    top: 0,
    left: 0,
    right: 0,
    bottom: 0,
    backgroundColor: 'rgba(13, 8, 17, 0.9)',
    backdropFilter: 'blur(8px)',
    display: 'flex',
    justifyContent: 'center',
    alignItems: 'center',
    zIndex: 1000,
    padding: '15px',
    boxSizing: 'border-box',
  },
  
  // Modal container - elevated glass effect
  modal: {
    background: gradients.card,
    padding: '28px',
    borderRadius: '16px',
    border: `1px solid ${colors.border}`,
    boxShadow: shadows.modal,
    maxWidth: '500px',
    width: '100%',
    maxHeight: 'calc(100vh - 30px)',
    overflowY: 'auto',
    boxSizing: 'border-box',
  },
  
  // Section container - glass card effect
  section: {
    background: gradients.card,
    borderRadius: '16px',
    padding: '24px',
    border: `1px solid ${colors.border}`,
    boxShadow: shadows.card,
    marginBottom: '24px',
    backdropFilter: 'blur(10px)',
  },
  
  // Section header
  sectionHeader: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: '20px',
    flexWrap: 'wrap',
    gap: '15px',
  },
  
  // Section title - gold accent
  sectionTitle: {
    fontSize: '1.25rem',
    fontWeight: 600,
    color: colors.textPrimary,
    margin: 0,
    letterSpacing: '0.02em',
  },
  
  // Card - elevated surface
  card: {
    background: gradients.card,
    padding: '20px',
    borderRadius: '12px',
    border: `1px solid ${colors.border}`,
    boxShadow: shadows.card,
    transition: `all ${transitions.normal}`,
  },
  
  // Stat card - with gold accent on hover
  statCard: {
    background: gradients.card,
    padding: '20px',
    borderRadius: '12px',
    border: `1px solid ${colors.border}`,
    textAlign: 'center',
    boxShadow: shadows.card,
    transition: `all ${transitions.normal}`,
  },
  
  // Form label
  label: {
    display: 'block',
    color: colors.textSecondary,
    marginBottom: '6px',
    fontSize: '0.875rem',
    fontWeight: 500,
    letterSpacing: '0.02em',
  },
  
  // Form input - modern with focus glow
  input: {
    width: '100%',
    padding: '12px 14px',
    backgroundColor: colors.bgInput,
    border: `1px solid ${colors.border}`,
    borderRadius: '10px',
    color: colors.textPrimary,
    fontSize: '0.95rem',
    marginBottom: '16px',
    boxSizing: 'border-box',
    transition: `all ${transitions.normal}`,
    outline: 'none',
  },
  
  // Primary button - Lion Gold gradient
  btnPrimary: {
    padding: '12px 24px',
    background: gradients.gold,
    border: 'none',
    borderRadius: '10px',
    color: colors.primaryDark,
    fontWeight: 600,
    cursor: 'pointer',
    fontSize: '0.95rem',
    letterSpacing: '0.02em',
    transition: `all ${transitions.normal}`,
    boxShadow: '0 2px 8px rgba(212, 168, 75, 0.3)',
  },
  
  // Secondary button - Purple outline
  btnSecondary: {
    padding: '12px 24px',
    backgroundColor: 'transparent',
    border: `2px solid ${colors.secondary}`,
    borderRadius: '10px',
    color: colors.secondary,
    fontWeight: 600,
    cursor: 'pointer',
    fontSize: '0.95rem',
    letterSpacing: '0.02em',
    transition: `all ${transitions.normal}`,
  },
  
  // Outline button - subtle
  btnOutline: {
    padding: '10px 18px',
    backgroundColor: 'transparent',
    border: `1px solid ${colors.border}`,
    borderRadius: '8px',
    color: colors.textSecondary,
    cursor: 'pointer',
    fontSize: '0.875rem',
    transition: `all ${transitions.normal}`,
  },
  
  // Small button
  btnSmall: {
    padding: '6px 12px',
    backgroundColor: 'rgba(123, 94, 167, 0.15)',
    border: `1px solid ${colors.secondary}`,
    borderRadius: '6px',
    color: colors.secondary,
    cursor: 'pointer',
    fontSize: '0.8rem',
    fontWeight: 500,
    transition: `all ${transitions.fast}`,
  },
  
  // Badge - modern pill shape
  badge: {
    fontSize: '0.7rem',
    padding: '4px 10px',
    borderRadius: '20px',
    fontWeight: 600,
    letterSpacing: '0.03em',
    textTransform: 'uppercase',
  },
  
  // Badge variants
  badgePrimary: {
    backgroundColor: 'rgba(74, 222, 128, 0.15)',
    color: colors.success,
    border: `1px solid rgba(74, 222, 128, 0.3)`,
  },
  
  badgeGold: {
    backgroundColor: 'rgba(212, 168, 75, 0.15)',
    color: colors.primary,
    border: `1px solid ${colors.borderGold}`,
  },
  
  badgeCoral: {
    backgroundColor: 'rgba(196, 92, 92, 0.15)',
    color: colors.coral,
    border: `1px solid rgba(196, 92, 92, 0.3)`,
  },
  
  // Time selector buttons - pill style
  timeBtn: {
    padding: '8px 14px',
    backgroundColor: colors.bgInput,
    border: `1px solid ${colors.border}`,
    borderRadius: '8px',
    color: colors.textMuted,
    cursor: 'pointer',
    fontSize: '0.85rem',
    fontWeight: 500,
    transition: `all ${transitions.fast}`,
  },
  
  timeBtnActive: {
    background: gradients.gold,
    color: colors.primaryDark,
    borderColor: colors.primary,
    boxShadow: shadows.glowGold,
  },
  
  // Loading state - with gold accent
  loading: {
    textAlign: 'center',
    padding: '60px',
    color: colors.primary,
  },
  
  // Error state
  error: {
    textAlign: 'center',
    padding: '60px',
    color: colors.coral,
  },
  
  // Live indicator - pulsing coral
  liveIndicator: {
    display: 'inline-flex',
    alignItems: 'center',
    gap: '6px',
    padding: '4px 12px',
    backgroundColor: 'rgba(196, 92, 92, 0.15)',
    border: `1px solid rgba(196, 92, 92, 0.3)`,
    borderRadius: '20px',
    color: colors.coral,
    fontSize: '0.8rem',
    fontWeight: 600,
  },
  
  // Status indicator dot
  statusDot: {
    width: '8px',
    height: '8px',
    borderRadius: '50%',
    backgroundColor: colors.success,
  },
  
  // Grid layouts
  gridAuto: {
    display: 'grid',
    gridTemplateColumns: 'repeat(auto-fit, minmax(150px, 1fr))',
    gap: '15px',
  },
  
  gridTwo: {
    display: 'grid',
    gridTemplateColumns: '1fr 1fr',
    gap: '15px',
  },
  
  gridThree: {
    display: 'grid',
    gridTemplateColumns: 'repeat(3, 1fr)',
    gap: '15px',
  },
  
  gridFour: {
    display: 'grid',
    gridTemplateColumns: 'repeat(4, 1fr)',
    gap: '15px',
  },
  
  // Flex layouts
  flexBetween: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
  },
  
  flexCenter: {
    display: 'flex',
    justifyContent: 'center',
    alignItems: 'center',
  },
  
  flexGap: {
    display: 'flex',
    gap: '10px',
  },
  
  // Tab navigation
  tabContainer: {
    display: 'flex',
    gap: '5px',
    backgroundColor: colors.bgInput,
    borderRadius: '8px',
    padding: '4px',
  },
  
  tab: {
    padding: '10px 16px',
    backgroundColor: 'transparent',
    border: 'none',
    borderBottom: '3px solid transparent',
    color: colors.textSecondary,
    cursor: 'pointer',
    fontSize: '0.9rem',
    transition: 'all 0.2s',
  },
  
  tabActive: {
    color: colors.primary,
    borderBottomColor: colors.primary,
  },
};

// Helper to merge styles
export const mergeStyles = (...styles: React.CSSProperties[]): React.CSSProperties => {
  return Object.assign({}, ...styles);
};

// Helper to conditionally apply styles
export const conditionalStyle = (
  condition: boolean,
  trueStyle: React.CSSProperties,
  falseStyle?: React.CSSProperties
): React.CSSProperties => {
  return condition ? trueStyle : (falseStyle || {});
};
