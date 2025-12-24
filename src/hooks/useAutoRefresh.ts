import { useState, useEffect, useCallback, useRef } from 'react';

export interface AutoRefreshConfig {
  /** Refresh interval in milliseconds */
  interval: number;
  /** Whether auto-refresh is enabled */
  enabled: boolean;
  /** Callback when refresh occurs */
  onRefresh: () => Promise<void> | void;
  /** Optional callback for errors */
  onError?: (error: Error) => void;
  /** Whether to refresh immediately on mount */
  refreshOnMount?: boolean;
}

export interface AutoRefreshState {
  /** Whether a refresh is currently in progress */
  isRefreshing: boolean;
  /** Last refresh timestamp */
  lastRefresh: Date | null;
  /** Time until next refresh in seconds */
  nextRefreshIn: number;
  /** Error from last refresh attempt */
  error: Error | null;
  /** Whether auto-refresh is currently active */
  isActive: boolean;
}

export interface AutoRefreshControls {
  /** Manually trigger a refresh */
  refresh: () => Promise<void>;
  /** Pause auto-refresh */
  pause: () => void;
  /** Resume auto-refresh */
  resume: () => void;
  /** Toggle auto-refresh on/off */
  toggle: () => void;
  /** Set a new refresh interval */
  setInterval: (ms: number) => void;
}

export type UseAutoRefreshReturn = AutoRefreshState & AutoRefreshControls;

/**
 * Custom hook for auto-refreshing data at configurable intervals
 * Implements clean lifecycle management and error handling
 */
export const useAutoRefresh = (config: AutoRefreshConfig): UseAutoRefreshReturn => {
  const {
    interval,
    enabled,
    onRefresh,
    onError,
    refreshOnMount = true,
  } = config;

  const [isRefreshing, setIsRefreshing] = useState(false);
  const [lastRefresh, setLastRefresh] = useState<Date | null>(null);
  const [nextRefreshIn, setNextRefreshIn] = useState(interval / 1000);
  const [error, setError] = useState<Error | null>(null);
  const [isActive, setIsActive] = useState(enabled);
  const [currentInterval, setCurrentInterval] = useState(interval);

  const intervalRef = useRef<NodeJS.Timeout | null>(null);
  const countdownRef = useRef<NodeJS.Timeout | null>(null);
  const isMountedRef = useRef(true);

  const refresh = useCallback(async () => {
    if (!isMountedRef.current) return;
    
    setIsRefreshing(true);
    setError(null);

    try {
      await onRefresh();
      if (isMountedRef.current) {
        setLastRefresh(new Date());
        setNextRefreshIn(currentInterval / 1000);
      }
    } catch (err) {
      const error = err instanceof Error ? err : new Error(String(err));
      if (isMountedRef.current) {
        setError(error);
        onError?.(error);
      }
    } finally {
      if (isMountedRef.current) {
        setIsRefreshing(false);
      }
    }
  }, [onRefresh, onError, currentInterval]);

  const pause = useCallback(() => {
    setIsActive(false);
  }, []);

  const resume = useCallback(() => {
    setIsActive(true);
  }, []);

  const toggle = useCallback(() => {
    setIsActive(prev => !prev);
  }, []);

  const updateInterval = useCallback((ms: number) => {
    setCurrentInterval(ms);
    setNextRefreshIn(ms / 1000);
  }, []);

  // Main refresh interval
  useEffect(() => {
    if (!isActive) {
      if (intervalRef.current) {
        clearInterval(intervalRef.current);
        intervalRef.current = null;
      }
      return;
    }

    intervalRef.current = setInterval(() => {
      refresh();
    }, currentInterval);

    return () => {
      if (intervalRef.current) {
        clearInterval(intervalRef.current);
      }
    };
  }, [isActive, currentInterval, refresh]);

  // Countdown timer
  useEffect(() => {
    if (!isActive) {
      if (countdownRef.current) {
        clearInterval(countdownRef.current);
        countdownRef.current = null;
      }
      return;
    }

    countdownRef.current = setInterval(() => {
      setNextRefreshIn(prev => {
        if (prev <= 1) {
          return currentInterval / 1000;
        }
        return prev - 1;
      });
    }, 1000);

    return () => {
      if (countdownRef.current) {
        clearInterval(countdownRef.current);
      }
    };
  }, [isActive, currentInterval]);

  // Initial refresh on mount
  useEffect(() => {
    isMountedRef.current = true;
    
    if (refreshOnMount) {
      refresh();
    }

    return () => {
      isMountedRef.current = false;
    };
  }, []); // Only run on mount

  // Sync with enabled prop
  useEffect(() => {
    setIsActive(enabled);
  }, [enabled]);

  return {
    isRefreshing,
    lastRefresh,
    nextRefreshIn,
    error,
    isActive,
    refresh,
    pause,
    resume,
    toggle,
    setInterval: updateInterval,
  };
};

// Preset intervals for common use cases
export const REFRESH_INTERVALS = {
  REALTIME: 5000,      // 5 seconds - for live mining data
  FAST: 10000,         // 10 seconds - for active monitoring
  NORMAL: 30000,       // 30 seconds - for general dashboards
  SLOW: 60000,         // 1 minute - for less critical data
  VERY_SLOW: 300000,   // 5 minutes - for historical data
} as const;

export default useAutoRefresh;
