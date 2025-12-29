import React from 'react';
import { render, screen, fireEvent, act } from '@testing-library/react';
import { ThemeProvider, useTheme, useThemeColors } from '../ThemeContext';

// Mock localStorage
const localStorageMock = (() => {
  let store: Record<string, string> = {};
  return {
    getItem: (key: string) => store[key] || null,
    setItem: (key: string, value: string) => { store[key] = value; },
    removeItem: (key: string) => { delete store[key]; },
    clear: () => { store = {}; },
  };
})();

Object.defineProperty(window, 'localStorage', { value: localStorageMock });

// Mock matchMedia
const mockMatchMedia = (matches: boolean) => ({
  matches,
  media: '(prefers-color-scheme: dark)',
  onchange: null,
  addListener: jest.fn(),
  removeListener: jest.fn(),
  addEventListener: jest.fn(),
  removeEventListener: jest.fn(),
  dispatchEvent: jest.fn(),
});

beforeEach(() => {
  localStorageMock.clear();
  window.matchMedia = jest.fn().mockImplementation(() => mockMatchMedia(true));
});

// Test component that uses the theme
function TestComponent() {
  const { mode, isDark, toggleMode, setMode, colors } = useTheme();
  return (
    <div>
      <span data-testid="mode">{mode}</span>
      <span data-testid="isDark">{isDark ? 'dark' : 'light'}</span>
      <span data-testid="primary">{colors.primary}</span>
      <button onClick={toggleMode}>Toggle</button>
      <button onClick={() => setMode('light')}>Set Light</button>
      <button onClick={() => setMode('system')}>Set System</button>
    </div>
  );
}

// Test component for useThemeColors
function ColorsTestComponent() {
  const colors = useThemeColors();
  return <span data-testid="colors-primary">{colors.primary}</span>;
}

describe('ThemeContext', () => {
  describe('ThemeProvider', () => {
    it('should provide default dark theme', () => {
      render(
        <ThemeProvider>
          <TestComponent />
        </ThemeProvider>
      );
      
      expect(screen.getByTestId('mode')).toHaveTextContent('dark');
      expect(screen.getByTestId('isDark')).toHaveTextContent('dark');
    });

    it('should provide dark theme colors by default', () => {
      render(
        <ThemeProvider>
          <TestComponent />
        </ThemeProvider>
      );
      
      expect(screen.getByTestId('primary')).toHaveTextContent('#D4A84B');
    });

    it('should toggle between dark and light', () => {
      render(
        <ThemeProvider>
          <TestComponent />
        </ThemeProvider>
      );
      
      expect(screen.getByTestId('isDark')).toHaveTextContent('dark');
      
      fireEvent.click(screen.getByText('Toggle'));
      
      expect(screen.getByTestId('isDark')).toHaveTextContent('light');
    });

    it('should set specific mode', () => {
      render(
        <ThemeProvider>
          <TestComponent />
        </ThemeProvider>
      );
      
      fireEvent.click(screen.getByText('Set Light'));
      
      expect(screen.getByTestId('mode')).toHaveTextContent('light');
    });

    it('should persist theme to localStorage', () => {
      render(
        <ThemeProvider>
          <TestComponent />
        </ThemeProvider>
      );
      
      fireEvent.click(screen.getByText('Set Light'));
      
      expect(localStorageMock.getItem('chimera-pool-theme')).toBe('light');
    });

    it('should load theme from localStorage', () => {
      localStorageMock.setItem('chimera-pool-theme', 'light');
      
      render(
        <ThemeProvider>
          <TestComponent />
        </ThemeProvider>
      );
      
      expect(screen.getByTestId('mode')).toHaveTextContent('light');
    });

    it('should support system mode', () => {
      render(
        <ThemeProvider>
          <TestComponent />
        </ThemeProvider>
      );
      
      fireEvent.click(screen.getByText('Set System'));
      
      expect(screen.getByTestId('mode')).toHaveTextContent('system');
      // With dark system preference mocked, should resolve to dark
      expect(screen.getByTestId('isDark')).toHaveTextContent('dark');
    });

    it('should use custom default mode', () => {
      render(
        <ThemeProvider defaultMode="light">
          <TestComponent />
        </ThemeProvider>
      );
      
      expect(screen.getByTestId('mode')).toHaveTextContent('light');
    });
  });

  describe('useTheme', () => {
    it('should throw error when used outside provider', () => {
      const consoleError = jest.spyOn(console, 'error').mockImplementation(() => {});
      
      expect(() => {
        render(<TestComponent />);
      }).toThrow('useTheme must be used within a ThemeProvider');
      
      consoleError.mockRestore();
    });
  });

  describe('useThemeColors', () => {
    it('should return theme colors', () => {
      render(
        <ThemeProvider>
          <ColorsTestComponent />
        </ThemeProvider>
      );
      
      expect(screen.getByTestId('colors-primary')).toHaveTextContent('#D4A84B');
    });
  });

  describe('Light theme colors', () => {
    it('should provide light theme colors when in light mode', () => {
      render(
        <ThemeProvider defaultMode="light">
          <TestComponent />
        </ThemeProvider>
      );
      
      // Light theme has different primary color
      expect(screen.getByTestId('primary')).toHaveTextContent('#B8923A');
    });
  });
});
