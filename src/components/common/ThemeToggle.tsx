import React from 'react';
import { useTheme, ThemeMode } from '../../contexts/ThemeContext';

// ============================================================================
// THEME TOGGLE COMPONENT
// Elegant toggle button for switching between dark/light/system themes
// ============================================================================

interface ThemeToggleProps {
  /** Show mode selector dropdown instead of simple toggle */
  showModeSelector?: boolean;
  /** Custom className */
  className?: string;
  /** Size variant */
  size?: 'small' | 'medium' | 'large';
}

const sizeStyles = {
  small: { padding: '6px 10px', fontSize: '0.8rem', iconSize: '14px' },
  medium: { padding: '8px 14px', fontSize: '0.9rem', iconSize: '18px' },
  large: { padding: '10px 18px', fontSize: '1rem', iconSize: '22px' },
};

export function ThemeToggle({ 
  showModeSelector = false, 
  className,
  size = 'medium' 
}: ThemeToggleProps) {
  const { mode, resolvedMode, isDark, toggleMode, setMode } = useTheme();
  const styles = sizeStyles[size];

  if (showModeSelector) {
    return (
      <div 
        className={className}
        style={{
          display: 'flex',
          gap: '4px',
          backgroundColor: 'rgba(255, 255, 255, 0.05)',
          borderRadius: '8px',
          padding: '4px',
        }}
      >
        {(['dark', 'light', 'system'] as ThemeMode[]).map((m) => (
          <button
            key={m}
            onClick={() => setMode(m)}
            aria-pressed={mode === m}
            aria-label={`Switch to ${m} mode`}
            data-testid={`theme-mode-${m}-btn`}
            style={{
              padding: styles.padding,
              fontSize: styles.fontSize,
              backgroundColor: mode === m ? 'rgba(212, 168, 75, 0.2)' : 'transparent',
              border: mode === m ? '1px solid rgba(212, 168, 75, 0.4)' : '1px solid transparent',
              borderRadius: '6px',
              color: mode === m ? '#D4A84B' : 'rgba(255, 255, 255, 0.6)',
              cursor: 'pointer',
              transition: 'all 150ms ease',
              display: 'flex',
              alignItems: 'center',
              gap: '6px',
            }}
          >
            {m === 'dark' && '🌙'}
            {m === 'light' && '☀️'}
            {m === 'system' && '💻'}
            <span style={{ textTransform: 'capitalize' }}>{m}</span>
          </button>
        ))}
      </div>
    );
  }

  return (
    <button
      onClick={toggleMode}
      className={className}
      aria-label={`Switch to ${isDark ? 'light' : 'dark'} mode`}
      aria-pressed={isDark}
      data-testid="theme-toggle-btn"
      style={{
        padding: styles.padding,
        fontSize: styles.iconSize,
        backgroundColor: 'rgba(255, 255, 255, 0.05)',
        border: '1px solid rgba(255, 255, 255, 0.1)',
        borderRadius: '8px',
        color: isDark ? '#F0EDF4' : '#1A0F1E',
        cursor: 'pointer',
        transition: 'all 200ms ease',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        width: size === 'small' ? '32px' : size === 'medium' ? '40px' : '48px',
        height: size === 'small' ? '32px' : size === 'medium' ? '40px' : '48px',
      }}
    >
      {isDark ? '☀️' : '🌙'}
    </button>
  );
}

export default ThemeToggle;
