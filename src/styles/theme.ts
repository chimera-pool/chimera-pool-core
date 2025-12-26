import React from 'react';

// ============================================================================
// CHIMERA POOL - ELITE DESIGN SYSTEM
// Color palette inspired by the mythological Chimera: Lion, Goat, and Serpent
// ============================================================================

// Brand Colors
export const colors = {
  // Primary - Lion Gold
  gold: '#D4A84B',
  goldDark: '#B8923A',
  goldLight: '#E5C77A',
  
  // Secondary - Deep Purple (Serpent)
  purple: '#7B5EA7',
  purpleDark: '#5A3D7A',
  purpleLight: '#9B7EC7',
  
  // Background - Dark Mystical
  bgDark: '#0D0811',
  bgMedium: '#1A0F1E',
  bgLight: '#2D1F3D',
  bgCard: '#1F1428',
  
  // Accent - Goat Fire
  accent: '#4A2C5A',
  accentLight: '#6A4C7A',
  
  // Text
  textPrimary: '#F0EDF4',
  textSecondary: '#B8B4C8',
  textMuted: '#7A7490',
  
  // Status Colors
  success: '#4ADE80',
  successDark: '#22C55E',
  error: '#C45C5C',
  errorLight: '#EF4444',
  warning: '#F59E0B',
  info: '#3B82F6',
  
  // Borders
  border: '#4A2C5A',
  borderLight: 'rgba(74, 44, 90, 0.4)',
} as const;

// Gradients
export const gradients = {
  goldButton: 'linear-gradient(135deg, #D4A84B 0%, #B8923A 100%)',
  purpleAccent: 'linear-gradient(135deg, #7B5EA7 0%, #5A3D7A 100%)',
  cardBg: 'linear-gradient(180deg, rgba(45, 31, 61, 0.8) 0%, rgba(26, 15, 30, 0.9) 100%)',
  sectionBg: 'linear-gradient(180deg, rgba(45, 31, 61, 0.6) 0%, rgba(26, 15, 30, 0.8) 100%)',
  headerBg: 'linear-gradient(135deg, #2D1F3D 0%, #3A1F2E 100%)',
  pageBg: 'linear-gradient(180deg, #1A0F1E 0%, #0D0811 100%)',
} as const;

// Shadows
export const shadows = {
  card: '0 4px 24px rgba(0, 0, 0, 0.3)',
  button: '0 2px 12px rgba(212, 168, 75, 0.3)',
  modal: '0 24px 48px rgba(0, 0, 0, 0.5)',
  glow: '0 0 20px rgba(212, 168, 75, 0.2)',
} as const;

// Typography
export const typography = {
  fontFamily: "'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif",
  monoFamily: "'JetBrains Mono', 'Fira Code', monospace",
  sizes: {
    xs: '0.75rem',
    sm: '0.85rem',
    base: '0.95rem',
    lg: '1.15rem',
    xl: '1.5rem',
    '2xl': '1.75rem',
  },
} as const;

// Spacing
export const spacing = {
  xs: '4px',
  sm: '8px',
  md: '16px',
  lg: '24px',
  xl: '32px',
  '2xl': '48px',
} as const;

// Border Radius
export const borderRadius = {
  sm: '8px',
  md: '12px',
  lg: '14px',
  xl: '16px',
  full: '9999px',
} as const;

// Main Application Styles
export const appStyles: { [key: string]: React.CSSProperties } = {
  // Layout
  container: {
    minHeight: '100vh',
    background: gradients.pageBg,
    color: colors.textPrimary,
    fontFamily: typography.fontFamily,
  },
  
  // Header
  header: {
    background: gradients.headerBg,
    padding: '8px 24px',
    borderBottom: `1px solid rgba(74, 44, 90, 0.5)`,
    backdropFilter: 'blur(10px)',
    position: 'sticky' as const,
    top: 0,
    zIndex: 100,
  },
  headerContent: {
    maxWidth: '1400px',
    margin: '0 auto',
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    flexWrap: 'wrap' as const,
    gap: '20px',
  },
  
  // Main content
  main: {
    maxWidth: '1400px',
    margin: '0 auto',
    padding: '32px 24px',
  },
  
  // Stats Grid
  statsGrid: {
    display: 'grid',
    gridTemplateColumns: 'repeat(auto-fit, minmax(180px, 1fr))',
    gap: '16px',
    marginBottom: '32px',
  },
  statCard: {
    background: gradients.cardBg,
    borderRadius: borderRadius.lg,
    padding: '20px',
    border: `1px solid ${colors.accent}`,
    textAlign: 'center' as const,
    boxShadow: shadows.card,
    transition: 'all 0.2s ease',
  },
  statLabel: {
    fontSize: typography.sizes.xs,
    color: colors.textSecondary,
    margin: '0 0 8px',
    textTransform: 'uppercase' as const,
    letterSpacing: '0.08em',
    fontWeight: 500,
  },
  statValue: {
    fontSize: typography.sizes.xl,
    color: colors.gold,
    margin: 0,
    fontWeight: 700,
    letterSpacing: '-0.02em',
  },
  
  // Sections
  section: {
    background: gradients.sectionBg,
    borderRadius: borderRadius.xl,
    padding: '24px',
    border: `1px solid ${colors.accent}`,
    marginBottom: '24px',
    boxShadow: shadows.card,
    backdropFilter: 'blur(10px)',
  },
  sectionTitle: {
    fontSize: typography.sizes.lg,
    color: colors.textPrimary,
    margin: '0 0 20px',
    fontWeight: 600,
    letterSpacing: '0.01em',
  },
  
  // Buttons
  primaryButton: {
    padding: '14px 24px',
    background: gradients.goldButton,
    border: 'none',
    borderRadius: borderRadius.md,
    color: colors.bgMedium,
    fontSize: '1rem',
    fontWeight: 600,
    cursor: 'pointer',
    boxShadow: shadows.button,
    transition: 'all 0.2s ease',
  },
  secondaryButton: {
    padding: '10px 20px',
    backgroundColor: 'transparent',
    border: `1px solid ${colors.purple}`,
    color: colors.textSecondary,
    borderRadius: borderRadius.sm,
    cursor: 'pointer',
    fontSize: '0.9rem',
    fontWeight: 500,
    transition: 'all 0.2s ease',
  },
  
  // Form Elements
  input: {
    width: '100%',
    padding: '14px 16px',
    marginBottom: '16px',
    backgroundColor: colors.bgCard,
    border: `1px solid ${colors.accent}`,
    borderRadius: borderRadius.md,
    color: colors.textPrimary,
    fontSize: typography.sizes.base,
    boxSizing: 'border-box' as const,
    outline: 'none',
    transition: 'all 0.2s ease',
  },
  
  // Modal
  modalOverlay: {
    position: 'fixed' as const,
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
  },
  modal: {
    background: `linear-gradient(180deg, ${colors.bgLight} 0%, ${colors.bgMedium} 100%)`,
    padding: '32px',
    borderRadius: '20px',
    border: `1px solid ${colors.accent}`,
    width: '100%',
    maxWidth: '420px',
    position: 'relative' as const,
    boxShadow: shadows.modal,
  },
  
  // Code blocks
  code: {
    display: 'block',
    backgroundColor: colors.bgDark,
    color: colors.success,
    padding: '14px 18px',
    borderRadius: borderRadius.sm,
    fontFamily: typography.monoFamily,
    fontSize: typography.sizes.base,
    margin: '10px 0',
    border: `1px solid ${colors.accent}`,
  },
  
  // Messages
  successMessage: {
    backgroundColor: 'rgba(74, 222, 128, 0.15)',
    color: colors.success,
    padding: '12px 16px',
    borderRadius: borderRadius.sm,
    border: '1px solid rgba(74, 222, 128, 0.3)',
  },
  errorMessage: {
    backgroundColor: 'rgba(196, 92, 92, 0.15)',
    color: colors.error,
    padding: '12px 16px',
    borderRadius: borderRadius.sm,
    border: '1px solid rgba(196, 92, 92, 0.3)',
  },
  
  // Loading
  loading: {
    textAlign: 'center' as const,
    padding: '60px',
    color: colors.gold,
  },
  
  // Footer
  footer: {
    textAlign: 'center' as const,
    padding: '32px 24px',
    borderTop: `1px solid ${colors.accent}`,
    color: colors.textMuted,
    background: 'linear-gradient(180deg, transparent 0%, rgba(13, 8, 17, 0.5) 100%)',
  },
};

// Navigation Styles
export const navStyles: { [key: string]: React.CSSProperties } = {
  mainNav: {
    display: 'flex',
    gap: '4px',
    backgroundColor: 'rgba(31, 20, 40, 0.8)',
    borderRadius: borderRadius.md,
    padding: '4px',
    border: `1px solid ${colors.accent}`,
  },
  navTab: {
    padding: '10px 20px',
    background: 'transparent',
    border: 'none',
    color: colors.textSecondary,
    fontSize: '0.9rem',
    cursor: 'pointer',
    borderRadius: borderRadius.sm,
    fontWeight: 500,
    transition: 'all 0.2s ease',
  },
  navTabActive: {
    background: gradients.goldButton,
    color: colors.bgMedium,
    fontWeight: 600,
    boxShadow: shadows.button,
  },
};

// Export theme object for convenience
export const theme = {
  colors,
  gradients,
  shadows,
  typography,
  spacing,
  borderRadius,
  styles: appStyles,
  navStyles,
};

export default theme;
