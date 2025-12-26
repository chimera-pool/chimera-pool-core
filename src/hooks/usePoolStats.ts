import { useState, useEffect, useCallback } from 'react';

export interface PoolStats {
  total_miners: number;
  total_hashrate: number;
  blocks_found: number;
  pool_fee: number;
  minimum_payout: number;
  payment_interval: string;
  network: string;
  currency: string;
}

interface UsePoolStatsOptions {
  refreshInterval?: number;
  enabled?: boolean;
}

interface UsePoolStatsReturn {
  stats: PoolStats | null;
  loading: boolean;
  error: Error | null;
  refresh: () => Promise<void>;
}

export const usePoolStats = (options: UsePoolStatsOptions = {}): UsePoolStatsReturn => {
  const { refreshInterval = 30000, enabled = true } = options;
  
  const [stats, setStats] = useState<PoolStats | null>(null);
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<Error | null>(null);

  const fetchStats = useCallback(async () => {
    if (!enabled) return;
    
    try {
      const response = await fetch('/api/v1/pool/stats');
      if (!response.ok) {
        throw new Error(`Failed to fetch pool stats: ${response.status}`);
      }
      const data = await response.json();
      setStats(data);
      setError(null);
    } catch (err) {
      console.error('Failed to fetch pool stats:', err);
      setError(err instanceof Error ? err : new Error('Unknown error'));
    } finally {
      setLoading(false);
    }
  }, [enabled]);

  useEffect(() => {
    fetchStats();
    
    if (enabled && refreshInterval > 0) {
      const interval = setInterval(fetchStats, refreshInterval);
      return () => clearInterval(interval);
    }
  }, [fetchStats, refreshInterval, enabled]);

  return {
    stats,
    loading,
    error,
    refresh: fetchStats,
  };
};

export default usePoolStats;
