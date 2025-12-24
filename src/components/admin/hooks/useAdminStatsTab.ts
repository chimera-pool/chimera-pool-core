/**
 * useAdminStatsTab - Isolated hook for Admin Stats Tab
 * 
 * ISP Compliant: This hook manages ALL state for the stats tab internally,
 * preventing parent component re-renders when stats data changes.
 * 
 * The hook fetches data directly from APIs and maintains its own state,
 * completely isolated from the parent AdminPanel component.
 */

import { useState, useCallback, useEffect, useRef } from 'react';

// ============================================================================
// TYPES
// ============================================================================

export type TimeRange = '1h' | '6h' | '24h' | '7d' | '30d' | '3m' | '6m' | '1y' | 'all';

export interface HashrateDataPoint {
  time: string;
  totalGH: number;
  hashrateTH?: number;
}

export interface SharesDataPoint {
  time: string;
  validShares: number;
  invalidShares: number;
  acceptanceRate: number;
}

export interface MinersDataPoint {
  time: string;
  activeMiners: number;
  totalMiners: number;
}

export interface PayoutsDataPoint {
  time: string;
  amount: number;
  count: number;
}

export interface DistributionDataPoint {
  name: string;
  value: number;
  color: string;
}

export interface AdminStatsTabState {
  // Data
  hashrateData: HashrateDataPoint[];
  sharesData: SharesDataPoint[];
  minersData: MinersDataPoint[];
  payoutsData: PayoutsDataPoint[];
  distributionData: DistributionDataPoint[];
  
  // State
  isLoading: boolean;
  error: Error | null;
  timeRange: TimeRange;
  
  // Actions
  setTimeRange: (range: TimeRange) => void;
  refresh: () => Promise<void>;
  
  // Auto-refresh
  isAutoRefreshEnabled: boolean;
  toggleAutoRefresh: () => void;
  refreshInterval: number;
  setRefreshInterval: (ms: number) => void;
}

// ============================================================================
// HOOK IMPLEMENTATION
// ============================================================================

export function useAdminStatsTab(token: string, isActive: boolean): AdminStatsTabState {
  // All state is LOCAL to this hook - no context dependencies
  const [hashrateData, setHashrateData] = useState<HashrateDataPoint[]>([]);
  const [sharesData, setSharesData] = useState<SharesDataPoint[]>([]);
  const [minersData, setMinersData] = useState<MinersDataPoint[]>([]);
  const [payoutsData, setPayoutsData] = useState<PayoutsDataPoint[]>([]);
  const [distributionData, setDistributionData] = useState<DistributionDataPoint[]>([]);
  
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<Error | null>(null);
  const [timeRange, setTimeRangeState] = useState<TimeRange>('24h');
  
  const [isAutoRefreshEnabled, setIsAutoRefreshEnabled] = useState(true);
  const [refreshInterval, setRefreshIntervalState] = useState(30000);
  
  // Refs to prevent stale closures
  const intervalRef = useRef<NodeJS.Timeout | null>(null);
  const isMountedRef = useRef(true);

  // ============================================================================
  // DATA FETCHING - All isolated, no parent dependencies
  // ============================================================================

  const fetchHashrateHistory = useCallback(async (range: TimeRange): Promise<HashrateDataPoint[]> => {
    try {
      const response = await fetch(`/api/v1/pool/stats/hashrate?range=${range}`);
      if (!response.ok) throw new Error(`Failed to fetch hashrate: ${response.status}`);
      
      const data = await response.json();
      return (data.data || []).map((d: any) => ({
        time: new Date(d.time).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
        totalGH: (d.hashrate || 0) / 1e9,
        hashrateTH: (d.hashrate || 0) / 1e12,
      }));
    } catch (err) {
      console.error('Hashrate fetch error:', err);
      return [];
    }
  }, []);

  const fetchSharesHistory = useCallback(async (range: TimeRange): Promise<SharesDataPoint[]> => {
    try {
      const response = await fetch(`/api/v1/pool/stats/shares?range=${range}`);
      if (!response.ok) throw new Error(`Failed to fetch shares: ${response.status}`);
      
      const data = await response.json();
      return (data.data || []).map((d: any) => {
        const valid = d.validShares || d.valid || 0;
        const invalid = d.invalidShares || d.invalid || 0;
        const total = valid + invalid;
        return {
          time: new Date(d.time).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
          validShares: valid,
          invalidShares: invalid,
          acceptanceRate: total > 0 ? (valid / total) * 100 : 100,
        };
      });
    } catch (err) {
      console.error('Shares fetch error:', err);
      return [];
    }
  }, []);

  const fetchMinersHistory = useCallback(async (range: TimeRange): Promise<MinersDataPoint[]> => {
    try {
      const response = await fetch(`/api/v1/pool/stats/miners?range=${range}`);
      if (!response.ok) throw new Error(`Failed to fetch miners: ${response.status}`);
      
      const data = await response.json();
      return (data.data || []).map((d: any) => ({
        time: new Date(d.time).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
        activeMiners: d.activeMiners || d.active_miners || 0,
        totalMiners: d.totalMiners || d.total_miners || d.activeMiners || 0,
      }));
    } catch (err) {
      console.error('Miners fetch error:', err);
      return [];
    }
  }, []);

  const fetchPayoutsHistory = useCallback(async (range: TimeRange): Promise<PayoutsDataPoint[]> => {
    try {
      const response = await fetch(`/api/v1/admin/stats/payouts?range=${range}`, {
        headers: { 'Authorization': `Bearer ${token}` },
      });
      if (!response.ok) return []; // Admin endpoint might not exist
      
      const data = await response.json();
      return (data.data || []).map((d: any) => ({
        time: new Date(d.time).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
        amount: d.amount || 0,
        count: d.count || 0,
      }));
    } catch {
      return [];
    }
  }, [token]);

  const fetchDistribution = useCallback(async (): Promise<DistributionDataPoint[]> => {
    try {
      const response = await fetch('/api/v1/admin/stats/distribution', {
        headers: { 'Authorization': `Bearer ${token}` },
      });
      if (!response.ok) return getDefaultDistribution();
      
      const data = await response.json();
      return data.distribution || getDefaultDistribution();
    } catch {
      return getDefaultDistribution();
    }
  }, [token]);

  // ============================================================================
  // MAIN REFRESH FUNCTION - Updates only this hook's state
  // ============================================================================

  const refresh = useCallback(async () => {
    if (!isMountedRef.current) return;
    
    setIsLoading(true);
    setError(null);

    try {
      // Fetch all data in parallel
      const [hashrate, shares, miners, payouts, distribution] = await Promise.all([
        fetchHashrateHistory(timeRange),
        fetchSharesHistory(timeRange),
        fetchMinersHistory(timeRange),
        fetchPayoutsHistory(timeRange),
        fetchDistribution(),
      ]);

      // Only update state if still mounted
      if (isMountedRef.current) {
        if (hashrate.length > 0) setHashrateData(hashrate);
        if (shares.length > 0) setSharesData(shares);
        if (miners.length > 0) setMinersData(miners);
        if (payouts.length > 0) setPayoutsData(payouts);
        if (distribution.length > 0) setDistributionData(distribution);
      }
    } catch (err) {
      if (isMountedRef.current) {
        setError(err as Error);
      }
    } finally {
      if (isMountedRef.current) {
        setIsLoading(false);
      }
    }
  }, [timeRange, fetchHashrateHistory, fetchSharesHistory, fetchMinersHistory, fetchPayoutsHistory, fetchDistribution]);

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

  // ============================================================================
  // EFFECTS - Only run when this tab is active
  // ============================================================================

  // Initial fetch when tab becomes active
  useEffect(() => {
    if (isActive) {
      refresh();
    }
  }, [isActive, timeRange]); // Only re-fetch when active or time range changes

  // Auto-refresh management
  useEffect(() => {
    if (isActive && isAutoRefreshEnabled) {
      intervalRef.current = setInterval(() => {
        refresh();
      }, refreshInterval);

      return () => {
        if (intervalRef.current) {
          clearInterval(intervalRef.current);
          intervalRef.current = null;
        }
      };
    } else {
      if (intervalRef.current) {
        clearInterval(intervalRef.current);
        intervalRef.current = null;
      }
    }
  }, [isActive, isAutoRefreshEnabled, refreshInterval, refresh]);

  // Cleanup on unmount
  useEffect(() => {
    isMountedRef.current = true;
    return () => {
      isMountedRef.current = false;
      if (intervalRef.current) {
        clearInterval(intervalRef.current);
      }
    };
  }, []);

  return {
    hashrateData,
    sharesData,
    minersData,
    payoutsData,
    distributionData,
    isLoading,
    error,
    timeRange,
    setTimeRange,
    refresh,
    isAutoRefreshEnabled,
    toggleAutoRefresh,
    refreshInterval,
    setRefreshInterval,
  };
}

// ============================================================================
// HELPERS
// ============================================================================

function getDefaultDistribution(): DistributionDataPoint[] {
  return [
    { name: 'Pool', value: 100, color: '#00d4ff' },
  ];
}

export default useAdminStatsTab;
