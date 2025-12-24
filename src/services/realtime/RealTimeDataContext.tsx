/**
 * Real-Time Data Context
 * 
 * Provides unified real-time mining data to all components in the app
 * Prevents duplicate API calls by sharing a single data service
 */

import React, { createContext, useContext, useEffect, useState, useCallback, useRef } from 'react';
import { RealTimeDataService, getRealTimeDataService } from './RealTimeDataService';
import {
  TimeRange,
  HashrateDataPoint,
  ShareDataPoint,
  MinerDataPoint,
  MinerInfo,
  PoolStats,
  UserStats,
  DEFAULT_REFRESH_CONFIG,
} from './types';

// ============================================================================
// CONTEXT TYPES
// ============================================================================

export interface RealTimeDataContextValue {
  // Connection state
  isConnected: boolean;
  isLoading: boolean;
  error: Error | null;

  // Current data
  poolStats: PoolStats;
  userStats: UserStats | null;
  activeMiners: MinerInfo[];

  // Historical data
  hashrateHistory: HashrateDataPoint[];
  sharesHistory: ShareDataPoint[];
  minersHistory: MinerDataPoint[];

  // Time range
  timeRange: TimeRange;
  setTimeRange: (range: TimeRange) => void;

  // Auto-refresh
  isAutoRefreshEnabled: boolean;
  refreshInterval: number;
  toggleAutoRefresh: () => void;
  setRefreshInterval: (ms: number) => void;

  // Manual controls
  refresh: () => Promise<void>;
  
  // Auth
  setAuthToken: (token: string | null) => void;
}

const defaultPoolStats: PoolStats = {
  totalHashrate: 0,
  activeMiners: 0,
  totalShares: 0,
  validShares: 0,
  invalidShares: 0,
  acceptanceRate: 100,
  blocksFound: 0,
  networkDifficulty: 0,
  poolDifficulty: 0,
  currency: 'LTC',
  algorithm: 'Scrypt',
};

const defaultContextValue: RealTimeDataContextValue = {
  isConnected: false,
  isLoading: true,
  error: null,
  poolStats: defaultPoolStats,
  userStats: null,
  activeMiners: [],
  hashrateHistory: [],
  sharesHistory: [],
  minersHistory: [],
  timeRange: '24h',
  setTimeRange: () => {},
  isAutoRefreshEnabled: true,
  refreshInterval: DEFAULT_REFRESH_CONFIG.interval,
  toggleAutoRefresh: () => {},
  setRefreshInterval: () => {},
  refresh: async () => {},
  setAuthToken: () => {},
};

// ============================================================================
// CONTEXT
// ============================================================================

export const RealTimeDataContext = createContext<RealTimeDataContextValue>(defaultContextValue);

// ============================================================================
// PROVIDER
// ============================================================================

export interface RealTimeDataProviderProps {
  children: React.ReactNode;
  initialTimeRange?: TimeRange;
  initialRefreshInterval?: number;
  autoRefreshEnabled?: boolean;
}

export function RealTimeDataProvider({
  children,
  initialTimeRange = '24h',
  initialRefreshInterval = DEFAULT_REFRESH_CONFIG.interval,
  autoRefreshEnabled = true,
}: RealTimeDataProviderProps) {
  // Service instance
  const serviceRef = useRef<RealTimeDataService>(getRealTimeDataService());

  // State
  const [isConnected, setIsConnected] = useState(false);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);
  const [poolStats, setPoolStats] = useState<PoolStats>(defaultPoolStats);
  const [userStats, setUserStats] = useState<UserStats | null>(null);
  const [activeMiners, setActiveMiners] = useState<MinerInfo[]>([]);
  const [hashrateHistory, setHashrateHistory] = useState<HashrateDataPoint[]>([]);
  const [sharesHistory, setSharesHistory] = useState<ShareDataPoint[]>([]);
  const [minersHistory, setMinersHistory] = useState<MinerDataPoint[]>([]);
  const [timeRange, setTimeRangeState] = useState<TimeRange>(initialTimeRange);
  const [isAutoRefreshEnabled, setIsAutoRefreshEnabled] = useState(autoRefreshEnabled);
  const [refreshInterval, setRefreshIntervalState] = useState(initialRefreshInterval);

  // Auth token ref (to avoid re-renders)
  const authTokenRef = useRef<string | null>(null);

  // ============================================================================
  // DATA FETCHING
  // ============================================================================

  const fetchAllData = useCallback(async () => {
    const service = serviceRef.current;
    setIsLoading(true);

    try {
      // Fetch main stats first (non-blocking)
      service.refresh().catch(err => console.warn('Stats refresh failed:', err));

      // Fetch historical data in parallel - each fetch is independent
      const [hashResult, shareResult, minerResult] = await Promise.allSettled([
        service.fetchHashrateHistory(timeRange),
        service.fetchSharesHistory(timeRange),
        service.fetchMinersHistory(timeRange),
      ]);

      // Update state with whatever data we got
      setPoolStats(service.getPoolStats());
      setUserStats(service.getUserStats());
      setActiveMiners(service.getActiveMiners());
      
      // Only update history if we got valid arrays
      if (hashResult.status === 'fulfilled' && hashResult.value.length > 0) {
        setHashrateHistory(hashResult.value);
      }
      if (shareResult.status === 'fulfilled' && shareResult.value.length > 0) {
        setSharesHistory(shareResult.value);
      }
      if (minerResult.status === 'fulfilled' && minerResult.value.length > 0) {
        setMinersHistory(minerResult.value);
      }
      
      setIsConnected(service.isConnected);
      setError(service.error);
    } catch (err) {
      console.error('fetchAllData error:', err);
      setError(err as Error);
      setIsConnected(false);
    } finally {
      setIsLoading(false);
    }
  }, [timeRange]);

  // ============================================================================
  // CONTROL METHODS
  // ============================================================================

  const setTimeRange = useCallback((range: TimeRange) => {
    setTimeRangeState(range);
  }, []);

  const toggleAutoRefresh = useCallback(() => {
    setIsAutoRefreshEnabled(prev => !prev);
  }, []);

  const setRefreshInterval = useCallback((ms: number) => {
    setRefreshIntervalState(ms);
  }, []);

  const setAuthToken = useCallback((token: string | null) => {
    authTokenRef.current = token;
    serviceRef.current.setAuthToken(token);
  }, []);

  const refresh = useCallback(async () => {
    await fetchAllData();
  }, [fetchAllData]);

  // ============================================================================
  // EFFECTS
  // ============================================================================

  // Initial fetch and time range changes
  useEffect(() => {
    fetchAllData();
  }, [fetchAllData]);

  // Auto-refresh management - fetch ALL data including history on each refresh
  useEffect(() => {
    if (isAutoRefreshEnabled) {
      // Use custom interval that fetches all data including history
      const intervalId = setInterval(() => {
        fetchAllData();
      }, refreshInterval);

      return () => {
        clearInterval(intervalId);
      };
    }
  }, [isAutoRefreshEnabled, refreshInterval, fetchAllData]);

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      serviceRef.current.stopAutoRefresh();
    };
  }, []);

  // ============================================================================
  // CONTEXT VALUE
  // ============================================================================

  const contextValue: RealTimeDataContextValue = {
    isConnected,
    isLoading,
    error,
    poolStats,
    userStats,
    activeMiners,
    hashrateHistory,
    sharesHistory,
    minersHistory,
    timeRange,
    setTimeRange,
    isAutoRefreshEnabled,
    refreshInterval,
    toggleAutoRefresh,
    setRefreshInterval,
    refresh,
    setAuthToken,
  };

  return (
    <RealTimeDataContext.Provider value={contextValue}>
      {children}
    </RealTimeDataContext.Provider>
  );
}

// ============================================================================
// HOOK
// ============================================================================

export function useRealTimeDataContext(): RealTimeDataContextValue {
  const context = useContext(RealTimeDataContext);
  if (context === undefined) {
    throw new Error('useRealTimeDataContext must be used within a RealTimeDataProvider');
  }
  return context;
}

export default RealTimeDataProvider;
