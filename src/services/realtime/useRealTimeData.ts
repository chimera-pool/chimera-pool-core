/**
 * Real-Time Data Hooks
 * 
 * ISP-compliant hooks for consuming specific slices of real-time data
 * Each hook provides only the data and methods needed for its specific use case
 */

import { useContext, useMemo } from 'react';
import { RealTimeDataContext } from './RealTimeDataContext';
import { TimeRange, HashrateDataPoint, ShareDataPoint, MinerInfo, PoolStats, UserStats } from './types';

// ============================================================================
// MAIN HOOK - Full access to all real-time data
// ============================================================================

export function useRealTimeData() {
  const context = useContext(RealTimeDataContext);
  if (context === undefined) {
    throw new Error('useRealTimeData must be used within a RealTimeDataProvider');
  }
  return context;
}

// ============================================================================
// ISP-COMPLIANT HOOKS - Specific data slices
// ============================================================================

/**
 * Hook for hashrate data only
 * Use when component only needs hashrate information
 */
export function useHashrateData() {
  const context = useRealTimeData();
  
  return useMemo(() => ({
    currentHashrate: context.poolStats.totalHashrate,
    currentHashrateTH: context.poolStats.totalHashrate / 1e12,
    history: context.hashrateHistory,
    isLoading: context.isLoading,
    error: context.error,
    timeRange: context.timeRange,
    setTimeRange: context.setTimeRange,
    refresh: context.refresh,
  }), [
    context.poolStats.totalHashrate,
    context.hashrateHistory,
    context.isLoading,
    context.error,
    context.timeRange,
    context.setTimeRange,
    context.refresh,
  ]);
}

/**
 * Hook for share statistics only
 * Use when component only needs share acceptance data
 */
export function useSharesData() {
  const context = useRealTimeData();
  
  return useMemo(() => ({
    validShares: context.poolStats.validShares,
    invalidShares: context.poolStats.invalidShares,
    totalShares: context.poolStats.totalShares,
    acceptanceRate: context.poolStats.acceptanceRate,
    history: context.sharesHistory,
    isLoading: context.isLoading,
    error: context.error,
    timeRange: context.timeRange,
    setTimeRange: context.setTimeRange,
    refresh: context.refresh,
  }), [
    context.poolStats.validShares,
    context.poolStats.invalidShares,
    context.poolStats.totalShares,
    context.poolStats.acceptanceRate,
    context.sharesHistory,
    context.isLoading,
    context.error,
    context.timeRange,
    context.setTimeRange,
    context.refresh,
  ]);
}

/**
 * Hook for miner data only
 * Use when component only needs miner information
 */
export function useMinersData() {
  const context = useRealTimeData();
  
  return useMemo(() => ({
    activeMiners: context.activeMiners,
    minerCount: context.poolStats.activeMiners,
    history: context.minersHistory,
    isLoading: context.isLoading,
    error: context.error,
    timeRange: context.timeRange,
    setTimeRange: context.setTimeRange,
    refresh: context.refresh,
  }), [
    context.activeMiners,
    context.poolStats.activeMiners,
    context.minersHistory,
    context.isLoading,
    context.error,
    context.timeRange,
    context.setTimeRange,
    context.refresh,
  ]);
}

/**
 * Hook for pool overview data
 * Use for dashboard summary components
 */
export function usePoolStats() {
  const context = useRealTimeData();
  
  return useMemo(() => ({
    stats: context.poolStats,
    isConnected: context.isConnected,
    isLoading: context.isLoading,
    error: context.error,
    refresh: context.refresh,
  }), [
    context.poolStats,
    context.isConnected,
    context.isLoading,
    context.error,
    context.refresh,
  ]);
}

/**
 * Hook for user-specific mining data
 * Use for personal dashboard components
 */
export function useUserStats() {
  const context = useRealTimeData();
  
  return useMemo(() => ({
    stats: context.userStats,
    isAuthenticated: context.userStats !== null,
    isLoading: context.isLoading,
    error: context.error,
    setAuthToken: context.setAuthToken,
    refresh: context.refresh,
  }), [
    context.userStats,
    context.isLoading,
    context.error,
    context.setAuthToken,
    context.refresh,
  ]);
}

/**
 * Hook for auto-refresh controls
 * Use when component needs to control refresh behavior
 */
export function useAutoRefreshControls() {
  const context = useRealTimeData();
  
  return useMemo(() => ({
    isEnabled: context.isAutoRefreshEnabled,
    interval: context.refreshInterval,
    toggle: context.toggleAutoRefresh,
    setInterval: context.setRefreshInterval,
    manualRefresh: context.refresh,
    isLoading: context.isLoading,
  }), [
    context.isAutoRefreshEnabled,
    context.refreshInterval,
    context.toggleAutoRefresh,
    context.setRefreshInterval,
    context.refresh,
    context.isLoading,
  ]);
}

/**
 * Hook for time range controls
 * Use when component needs to change the data time range
 */
export function useTimeRangeControls() {
  const context = useRealTimeData();
  
  return useMemo(() => ({
    timeRange: context.timeRange,
    setTimeRange: context.setTimeRange,
    isLoading: context.isLoading,
  }), [
    context.timeRange,
    context.setTimeRange,
    context.isLoading,
  ]);
}

// ============================================================================
// COMBINED HOOKS - For specific UI sections
// ============================================================================

/**
 * Hook for Dashboard graphs section
 * Provides all data needed for the MiningGraphs component
 */
export function useDashboardGraphs() {
  const context = useRealTimeData();
  
  return useMemo(() => ({
    // Data
    hashrateHistory: context.hashrateHistory,
    sharesHistory: context.sharesHistory,
    minersHistory: context.minersHistory,
    poolStats: context.poolStats,
    
    // State
    isLoading: context.isLoading,
    error: context.error,
    isConnected: context.isConnected,
    
    // Controls
    timeRange: context.timeRange,
    setTimeRange: context.setTimeRange,
    isAutoRefreshEnabled: context.isAutoRefreshEnabled,
    toggleAutoRefresh: context.toggleAutoRefresh,
    refresh: context.refresh,
  }), [
    context.hashrateHistory,
    context.sharesHistory,
    context.minersHistory,
    context.poolStats,
    context.isLoading,
    context.error,
    context.isConnected,
    context.timeRange,
    context.setTimeRange,
    context.isAutoRefreshEnabled,
    context.toggleAutoRefresh,
    context.refresh,
  ]);
}

/**
 * Hook for Admin Panel stats section
 * Provides all data needed for admin statistics
 */
export function useAdminStats() {
  const context = useRealTimeData();
  
  return useMemo(() => ({
    // Data
    hashrateHistory: context.hashrateHistory,
    sharesHistory: context.sharesHistory,
    minersHistory: context.minersHistory,
    poolStats: context.poolStats,
    activeMiners: context.activeMiners,
    
    // State
    isLoading: context.isLoading,
    error: context.error,
    
    // Controls
    timeRange: context.timeRange,
    setTimeRange: context.setTimeRange,
    isAutoRefreshEnabled: context.isAutoRefreshEnabled,
    refreshInterval: context.refreshInterval,
    toggleAutoRefresh: context.toggleAutoRefresh,
    setRefreshInterval: context.setRefreshInterval,
    refresh: context.refresh,
  }), [
    context.hashrateHistory,
    context.sharesHistory,
    context.minersHistory,
    context.poolStats,
    context.activeMiners,
    context.isLoading,
    context.error,
    context.timeRange,
    context.setTimeRange,
    context.isAutoRefreshEnabled,
    context.refreshInterval,
    context.toggleAutoRefresh,
    context.setRefreshInterval,
    context.refresh,
  ]);
}

/**
 * Hook for User Dashboard/Miner section
 * Provides user-specific mining data
 */
export function useUserDashboard() {
  const context = useRealTimeData();
  
  return useMemo(() => ({
    // User data
    userStats: context.userStats,
    isAuthenticated: context.userStats !== null,
    
    // Pool context
    poolStats: context.poolStats,
    
    // State
    isLoading: context.isLoading,
    error: context.error,
    isConnected: context.isConnected,
    
    // Controls
    setAuthToken: context.setAuthToken,
    refresh: context.refresh,
  }), [
    context.userStats,
    context.poolStats,
    context.isLoading,
    context.error,
    context.isConnected,
    context.setAuthToken,
    context.refresh,
  ]);
}

export default useRealTimeData;
