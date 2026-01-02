import { renderHook, act, waitFor } from '@testing-library/react';
import { useAutoRefresh, AutoRefreshConfig } from '../useAutoRefresh';

describe('useAutoRefresh', () => {
  beforeEach(() => {
    jest.useFakeTimers();
  });

  afterEach(() => {
    jest.useRealTimers();
    jest.clearAllMocks();
  });

  const createConfig = (overrides: Partial<AutoRefreshConfig> = {}): AutoRefreshConfig => ({
    interval: 5000,
    enabled: true,
    onRefresh: jest.fn().mockResolvedValue(undefined),
    ...overrides,
  });

  describe('Initial State', () => {
    it('should initialize with correct default state', () => {
      const config = createConfig({ refreshOnMount: false });
      const { result } = renderHook(() => useAutoRefresh(config));

      expect(result.current.error).toBeNull();
      expect(result.current.isActive).toBe(true);
    });

    it('should initialize as inactive when enabled is false', () => {
      const config = createConfig({ enabled: false });
      const { result } = renderHook(() => useAutoRefresh(config));

      expect(result.current.isActive).toBe(false);
    });

    it('should set nextRefreshIn based on interval', () => {
      const config = createConfig({ interval: 10000 });
      const { result } = renderHook(() => useAutoRefresh(config));

      expect(result.current.nextRefreshIn).toBe(10);
    });
  });

  describe('Refresh on Mount', () => {
    it('should refresh on mount when refreshOnMount is true', async () => {
      const onRefresh = jest.fn().mockResolvedValue(undefined);
      const config = createConfig({ onRefresh, refreshOnMount: true });
      
      renderHook(() => useAutoRefresh(config));

      await act(async () => {
        await Promise.resolve();
      });

      expect(onRefresh).toHaveBeenCalled();
    });

    it('should not refresh on mount when refreshOnMount is false', async () => {
      const onRefresh = jest.fn().mockResolvedValue(undefined);
      const config = createConfig({ onRefresh, refreshOnMount: false });
      
      renderHook(() => useAutoRefresh(config));

      await act(async () => {
        await Promise.resolve();
      });

      // Should not be called immediately
      expect(onRefresh).not.toHaveBeenCalled();
    });
  });

  describe('Manual Refresh', () => {
    it('should allow manual refresh', async () => {
      const onRefresh = jest.fn().mockResolvedValue(undefined);
      const config = createConfig({ onRefresh, refreshOnMount: false });
      const { result } = renderHook(() => useAutoRefresh(config));

      await act(async () => {
        await result.current.refresh();
      });

      expect(onRefresh).toHaveBeenCalled();
    });

    it('should set lastRefresh after successful refresh', async () => {
      const onRefresh = jest.fn().mockResolvedValue(undefined);
      const config = createConfig({ onRefresh, refreshOnMount: false });
      const { result } = renderHook(() => useAutoRefresh(config));

      expect(result.current.lastRefresh).toBeNull();

      await act(async () => {
        await result.current.refresh();
      });

      expect(result.current.lastRefresh).toBeInstanceOf(Date);
    });

    it('should handle refresh errors', async () => {
      const error = new Error('Refresh failed');
      const onRefresh = jest.fn().mockRejectedValue(error);
      const onError = jest.fn();
      const config = createConfig({ onRefresh, onError, refreshOnMount: false });
      const { result } = renderHook(() => useAutoRefresh(config));

      await act(async () => {
        await result.current.refresh();
      });

      expect(result.current.error).toEqual(error);
      expect(onError).toHaveBeenCalledWith(error);
    });
  });

  describe('Pause and Resume', () => {
    it('should pause auto-refresh', () => {
      const config = createConfig();
      const { result } = renderHook(() => useAutoRefresh(config));

      expect(result.current.isActive).toBe(true);

      act(() => {
        result.current.pause();
      });

      expect(result.current.isActive).toBe(false);
    });

    it('should resume auto-refresh', () => {
      const config = createConfig({ enabled: false });
      const { result } = renderHook(() => useAutoRefresh(config));

      expect(result.current.isActive).toBe(false);

      act(() => {
        result.current.resume();
      });

      expect(result.current.isActive).toBe(true);
    });

    it('should toggle auto-refresh', () => {
      const config = createConfig();
      const { result } = renderHook(() => useAutoRefresh(config));

      expect(result.current.isActive).toBe(true);

      act(() => {
        result.current.toggle();
      });

      expect(result.current.isActive).toBe(false);

      act(() => {
        result.current.toggle();
      });

      expect(result.current.isActive).toBe(true);
    });
  });

  describe('Interval Management', () => {
    it('should allow setting a new interval', () => {
      const config = createConfig({ interval: 5000 });
      const { result } = renderHook(() => useAutoRefresh(config));

      expect(result.current.nextRefreshIn).toBe(5);

      act(() => {
        result.current.setInterval(10000);
      });

      expect(result.current.nextRefreshIn).toBe(10);
    });
  });

  describe('Auto Refresh Cycle', () => {
    it('should auto-refresh at specified interval', async () => {
      const onRefresh = jest.fn().mockResolvedValue(undefined);
      const config = createConfig({ 
        onRefresh, 
        interval: 5000, 
        refreshOnMount: false 
      });
      
      renderHook(() => useAutoRefresh(config));

      // Advance timer by interval
      await act(async () => {
        jest.advanceTimersByTime(5000);
        await Promise.resolve();
      });

      expect(onRefresh).toHaveBeenCalled();
    });

    it('should not auto-refresh when paused', async () => {
      const onRefresh = jest.fn().mockResolvedValue(undefined);
      const config = createConfig({ 
        onRefresh, 
        interval: 5000, 
        enabled: false,
        refreshOnMount: false 
      });
      
      renderHook(() => useAutoRefresh(config));

      // Advance timer by interval
      await act(async () => {
        jest.advanceTimersByTime(5000);
        await Promise.resolve();
      });

      expect(onRefresh).not.toHaveBeenCalled();
    });
  });

  describe('Cleanup', () => {
    it('should clean up intervals on unmount', () => {
      const config = createConfig();
      const { unmount } = renderHook(() => useAutoRefresh(config));

      // Should not throw when unmounting
      expect(() => unmount()).not.toThrow();
    });
  });
});
