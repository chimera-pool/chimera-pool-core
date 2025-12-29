import React, { createContext, useContext, useState, useEffect, useCallback, ReactNode } from 'react';

// ============================================================================
// THEME CONTEXT - Dark/Light Mode Support
// Following Interface Segregation Principle with clear, focused interfaces
// ============================================================================

export type ThemeMode = 'dark' | 'light' | 'system';

/** Theme colors interface - ISP compliant */
export interface ThemeColors {
  // Primary brand colors
  primary: string;
  primaryDark: string;
  primaryLight: string;
  
  // Secondary colors
  secondary: string;
  secondaryDark: string;
  secondaryLight: string;
  
  // Accent colors
  coral: string;
  coralLight: string;
  silver: string;
  
  // Semantic colors
  success: string;
  error: string;
  warning: string;
  info: string;
  
  // Background hierarchy
  bgDeep: string;
  bgDark: string;
  bgCard: string;
  bgElevated: string;
  bgInput: string;
  bgHover: string;
  
  // Border colors
  border: string;
  borderLight: string;
  borderGold: string;
  
  // Text colors
  textPrimary: string;
  textSecondary: string;
  textMuted: string;
  textGold: string;
}

/** Theme context value interface */
export interface ThemeContextValue {
  mode: ThemeMode;
  resolvedMode: 'dark' | 'light';
  colors: ThemeColors;
  setMode: (mode: ThemeMode) => void;
  toggleMode: () => void;
  isDark: boolean;
}

// Dark theme colors (current design)
const darkColors: ThemeColors = {
  primary: '#D4A84B',
  primaryDark: '#1A0F1E',
  primaryLight: '#E8C171',
  secondary: '#7B5EA7',
  secondaryDark: '#5A4580',
  secondaryLight: '#9B7EC7',
  coral: '#C45C5C',
  coralLight: '#E07777',
  silver: '#B8B4C8',
  success: '#4ADE80',
  error: '#EF4444',
  warning: '#FBBF24',
  info: '#60A5FA',
  bgDeep: '#0D0811',
  bgDark: '#1A0F1E',
  bgCard: '#2D1F3D',
  bgElevated: '#3A2850',
  bgInput: '#1F1428',
  bgHover: '#3A1F2E',
  border: '#4A2C5A',
  borderLight: '#5A3C6A',
  borderGold: 'rgba(212, 168, 75, 0.3)',
  textPrimary: '#F0EDF4',
  textSecondary: '#B8B4C8',
  textMuted: '#7A7490',
  textGold: '#D4A84B',
};

// Light theme colors (new)
const lightColors: ThemeColors = {
  primary: '#B8923A',
  primaryDark: '#F5F3F0',
  primaryLight: '#D4A84B',
  secondary: '#6B4E97',
  secondaryDark: '#5A4580',
  secondaryLight: '#8B6EB7',
  coral: '#C45C5C',
  coralLight: '#E07777',
  silver: '#6B6680',
  success: '#22C55E',
  error: '#DC2626',
  warning: '#F59E0B',
  info: '#3B82F6',
  bgDeep: '#FFFFFF',
  bgDark: '#F8F6F4',
  bgCard: '#FFFFFF',
  bgElevated: '#F0EDE8',
  bgInput: '#FFFFFF',
  bgHover: '#F5F0EA',
  border: '#E0D8D0',
  borderLight: '#D0C8C0',
  borderGold: 'rgba(184, 146, 58, 0.3)',
  textPrimary: '#1A0F1E',
  textSecondary: '#5A5068',
  textMuted: '#8A8498',
  textGold: '#B8923A',
};

const ThemeContext = createContext<ThemeContextValue | undefined>(undefined);

/** Storage key for persisting theme preference */
const THEME_STORAGE_KEY = 'chimera-pool-theme';

/** Get system preference */
const getSystemPreference = (): 'dark' | 'light' => {
  if (typeof window === 'undefined') return 'dark';
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
};

/** Theme provider props */
interface ThemeProviderProps {
  children: ReactNode;
  defaultMode?: ThemeMode;
}

/** Theme provider component */
export function ThemeProvider({ children, defaultMode = 'dark' }: ThemeProviderProps) {
  const [mode, setModeState] = useState<ThemeMode>(() => {
    if (typeof window === 'undefined') return defaultMode;
    const stored = localStorage.getItem(THEME_STORAGE_KEY) as ThemeMode | null;
    return stored || defaultMode;
  });

  const [systemPreference, setSystemPreference] = useState<'dark' | 'light'>(getSystemPreference);

  // Listen for system preference changes
  useEffect(() => {
    const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
    const handler = (e: MediaQueryListEvent) => {
      setSystemPreference(e.matches ? 'dark' : 'light');
    };
    
    mediaQuery.addEventListener('change', handler);
    return () => mediaQuery.removeEventListener('change', handler);
  }, []);

  // Persist theme preference
  const setMode = useCallback((newMode: ThemeMode) => {
    setModeState(newMode);
    localStorage.setItem(THEME_STORAGE_KEY, newMode);
  }, []);

  // Toggle between dark and light
  const toggleMode = useCallback(() => {
    setMode(mode === 'dark' ? 'light' : mode === 'light' ? 'dark' : 
      systemPreference === 'dark' ? 'light' : 'dark');
  }, [mode, systemPreference, setMode]);

  // Resolve actual theme
  const resolvedMode: 'dark' | 'light' = mode === 'system' ? systemPreference : mode;
  const isDark = resolvedMode === 'dark';
  const colors = isDark ? darkColors : lightColors;

  // Apply theme class to document
  useEffect(() => {
    document.documentElement.setAttribute('data-theme', resolvedMode);
    document.body.style.backgroundColor = colors.bgDark;
    document.body.style.color = colors.textPrimary;
  }, [resolvedMode, colors]);

  const value: ThemeContextValue = {
    mode,
    resolvedMode,
    colors,
    setMode,
    toggleMode,
    isDark,
  };

  return (
    <ThemeContext.Provider value={value}>
      {children}
    </ThemeContext.Provider>
  );
}

/** Hook to use theme context */
export function useTheme(): ThemeContextValue {
  const context = useContext(ThemeContext);
  if (!context) {
    throw new Error('useTheme must be used within a ThemeProvider');
  }
  return context;
}

/** Hook to use theme colors only (lighter weight) */
export function useThemeColors(): ThemeColors {
  const { colors } = useTheme();
  return colors;
}

export default ThemeContext;
