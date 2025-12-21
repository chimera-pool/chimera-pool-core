import React from 'react';

// ============================================================================
// CHIMERA POOL - SHARED STYLE CONSTANTS
// Consolidated from 13 duplicate style objects in App.tsx
// ============================================================================

// Color Palette
export const colors = {
  primary: '#00d4ff',
  primaryDark: '#0a0a0f',
  secondary: '#f59e0b',
  success: '#4ade80',
  error: '#ef4444',
  warning: '#f59e0b',
  
  bgDark: '#0a0a0f',
  bgCard: '#1a1a2e',
  bgInput: '#0a0a15',
  
  border: '#2a2a4a',
  borderLight: '#3a3a5a',
  
  textPrimary: '#e0e0e0',
  textSecondary: '#888',
  textMuted: '#666',
};

// Common gradients
export const gradients = {
  card: 'linear-gradient(135deg, #1a1a2e 0%, #0f0f1a 100%)',
  header: 'linear-gradient(135deg, #1a1a2e 0%, #16213e 100%)',
};

// Shared base styles
export const baseStyles: { [key: string]: React.CSSProperties } = {
  // Modal overlay - used in 5+ places
  modalOverlay: {
    position: 'fixed',
    top: 0,
    left: 0,
    right: 0,
    bottom: 0,
    backgroundColor: 'rgba(0,0,0,0.85)',
    display: 'flex',
    justifyContent: 'center',
    alignItems: 'center',
    zIndex: 1000,
    padding: '15px',
    boxSizing: 'border-box',
  },
  
  // Modal container - used in 5+ places
  modal: {
    backgroundColor: colors.bgCard,
    padding: '20px',
    borderRadius: '12px',
    border: `2px solid ${colors.primary}`,
    maxWidth: '500px',
    width: '100%',
    maxHeight: 'calc(100vh - 30px)',
    overflowY: 'auto',
    boxSizing: 'border-box',
  },
  
  // Section container - used in graphs, maps, community
  section: {
    background: gradients.card,
    borderRadius: '12px',
    padding: '24px',
    border: `1px solid ${colors.border}`,
    marginBottom: '20px',
  },
  
  // Section header - used in graphs, maps, community
  sectionHeader: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: '20px',
    flexWrap: 'wrap',
    gap: '15px',
  },
  
  // Section title
  sectionTitle: {
    fontSize: '1.3rem',
    color: colors.primary,
    margin: 0,
  },
  
  // Card - used in stats, equipment, wallets
  card: {
    backgroundColor: colors.bgCard,
    padding: '15px',
    borderRadius: '10px',
    border: `1px solid ${colors.border}`,
  },
  
  // Stat card
  statCard: {
    backgroundColor: colors.bgCard,
    padding: '15px',
    borderRadius: '10px',
    border: `2px solid ${colors.border}`,
    textAlign: 'center',
  },
  
  // Form label
  label: {
    display: 'block',
    color: colors.textSecondary,
    marginBottom: '4px',
    fontSize: '0.85rem',
  },
  
  // Form input
  input: {
    width: '100%',
    padding: '10px',
    backgroundColor: colors.bgInput,
    border: `1px solid ${colors.border}`,
    borderRadius: '6px',
    color: colors.textPrimary,
    fontSize: '0.95rem',
    marginBottom: '12px',
    boxSizing: 'border-box',
  },
  
  // Primary button
  btnPrimary: {
    padding: '10px 20px',
    backgroundColor: colors.primary,
    border: 'none',
    borderRadius: '6px',
    color: colors.primaryDark,
    fontWeight: 'bold',
    cursor: 'pointer',
    fontSize: '0.9rem',
  },
  
  // Secondary/Cancel button
  btnSecondary: {
    padding: '10px 20px',
    backgroundColor: 'transparent',
    border: `1px solid ${colors.textSecondary}`,
    borderRadius: '6px',
    color: colors.textSecondary,
    cursor: 'pointer',
    fontSize: '0.9rem',
  },
  
  // Outline button
  btnOutline: {
    padding: '8px 16px',
    backgroundColor: 'transparent',
    border: `1px solid ${colors.primary}`,
    borderRadius: '6px',
    color: colors.primary,
    cursor: 'pointer',
    fontSize: '0.85rem',
  },
  
  // Small button
  btnSmall: {
    padding: '5px 10px',
    backgroundColor: 'transparent',
    border: `1px solid ${colors.primary}`,
    borderRadius: '4px',
    color: colors.primary,
    cursor: 'pointer',
    fontSize: '0.8rem',
  },
  
  // Badge
  badge: {
    fontSize: '0.65rem',
    padding: '2px 6px',
    borderRadius: '4px',
    fontWeight: 'bold',
  },
  
  // Badge variants
  badgePrimary: {
    backgroundColor: colors.success,
    color: colors.primaryDark,
  },
  
  // Time selector buttons
  timeBtn: {
    padding: '6px 12px',
    backgroundColor: colors.bgInput,
    border: `1px solid ${colors.border}`,
    borderRadius: '6px',
    color: colors.textSecondary,
    cursor: 'pointer',
    fontSize: '0.85rem',
  },
  
  timeBtnActive: {
    backgroundColor: colors.primary,
    color: colors.primaryDark,
    borderColor: colors.primary,
  },
  
  // Loading state
  loading: {
    textAlign: 'center',
    padding: '60px',
    color: colors.primary,
  },
  
  // Error state
  error: {
    textAlign: 'center',
    padding: '60px',
    color: colors.error,
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
