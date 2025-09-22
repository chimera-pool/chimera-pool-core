// Cyber-Minimal Theme Configuration
export const cyberTheme = {
  colors: {
    // Primary Colors
    cyberBlack: '#0a0a0a',
    cyberDark: '#1a1a1a',
    cyberGray: '#2a2a2a',
    cyberLightGray: '#3a3a3a',
    
    // Accent Colors
    cyberNeonGreen: '#00ff41',
    cyberNeonBlue: '#00d4ff',
    cyberNeonPurple: '#b300ff',
    cyberNeonOrange: '#ff6b00',
    
    // Status Colors
    cyberSuccess: '#00ff41',
    cyberWarning: '#ffaa00',
    cyberError: '#ff0040',
    cyberInfo: '#00d4ff',
    
    // Text Colors
    cyberTextPrimary: '#ffffff',
    cyberTextSecondary: '#b0b0b0',
    cyberTextMuted: '#707070',
  },
  fonts: {
    mono: "'JetBrains Mono', 'Fira Code', monospace",
    sans: "'Inter', -apple-system, BlinkMacSystemFont, sans-serif",
  },
  animations: {
    pulse: 'cyber-pulse 2s infinite',
    glow: 'cyber-glow 1.5s ease-in-out infinite alternate',
    scanLine: 'cyber-scan-line 3s linear infinite',
  },
  breakpoints: {
    mobile: '768px',
    tablet: '1024px',
    desktop: '1200px',
  },
} as const;

export type CyberTheme = typeof cyberTheme;