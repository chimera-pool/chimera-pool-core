/**
 * Hook for fetching and managing user equipment status
 * Used to determine visibility of "Your Mining Dashboard" section
 */

import { useState, useEffect, useCallback } from 'react';
import {
  IUserEquipmentStatus,
  IEquipmentDevice,
  calculateEquipmentStatus,
} from '../components/dashboard/interfaces/IUserMiningDashboard';

export interface UseUserEquipmentStatusOptions {
  /** Auth token for API calls */
  token: string | null;
  /** Polling interval in ms (default: 30000) */
  pollInterval?: number;
  /** Whether to enable polling (default: true) */
  enablePolling?: boolean;
}

export interface UseUserEquipmentStatusResult {
  /** Current equipment status */
  status: IUserEquipmentStatus;
  /** Manually refresh equipment status */
  refresh: () => Promise<void>;
  /** Whether initial load is complete */
  isInitialized: boolean;
}

const DEFAULT_STATUS: IUserEquipmentStatus = {
  hasEquipment: false,
  hasActiveEquipment: false,
  totalEquipmentCount: 0,
  activeEquipmentCount: 0,
  hasPendingSupport: false,
  isLoading: true,
  error: null,
};

/**
 * Hook to fetch and monitor user's mining equipment status
 */
export function useUserEquipmentStatus({
  token,
  pollInterval = 30000,
  enablePolling = true,
}: UseUserEquipmentStatusOptions): UseUserEquipmentStatusResult {
  const [status, setStatus] = useState<IUserEquipmentStatus>(DEFAULT_STATUS);
  const [isInitialized, setIsInitialized] = useState(false);

  const fetchEquipmentStatus = useCallback(async () => {
    if (!token) {
      setStatus({
        ...DEFAULT_STATUS,
        isLoading: false,
      });
      setIsInitialized(true);
      return;
    }

    try {
      const response = await fetch('/api/v1/user/equipment', {
        headers: { 'Authorization': `Bearer ${token}` },
      });

      if (response.ok) {
        const data = await response.json();
        const equipment: IEquipmentDevice[] = (data.equipment || []).map((e: any) => ({
          id: e.id,
          name: e.name || `Device ${e.id}`,
          type: e.type || 'unknown',
          status: e.status || 'offline',
          hashrate: e.hashrate || 0,
          lastSeen: e.last_seen || new Date(),
          isActive: ['mining', 'online', 'idle'].includes(e.status),
        }));

        const newStatus = calculateEquipmentStatus(
          equipment,
          false,
          null,
          data.pending_support || false
        );
        setStatus(newStatus);
      } else if (response.status === 401) {
        // Unauthorized - token expired
        setStatus({
          ...DEFAULT_STATUS,
          isLoading: false,
          error: 'Session expired',
        });
      } else {
        // API error but not auth issue - use empty status
        setStatus({
          ...DEFAULT_STATUS,
          isLoading: false,
        });
      }
    } catch (error) {
      console.error('Failed to fetch equipment status:', error);
      setStatus({
        ...DEFAULT_STATUS,
        isLoading: false,
        error: error instanceof Error ? error.message : 'Network error',
      });
    } finally {
      setIsInitialized(true);
    }
  }, [token]);

  // Initial fetch
  useEffect(() => {
    fetchEquipmentStatus();
  }, [fetchEquipmentStatus]);

  // Polling
  useEffect(() => {
    if (!enablePolling || !token) return;

    const interval = setInterval(fetchEquipmentStatus, pollInterval);
    return () => clearInterval(interval);
  }, [fetchEquipmentStatus, pollInterval, enablePolling, token]);

  return {
    status,
    refresh: fetchEquipmentStatus,
    isInitialized,
  };
}

export default useUserEquipmentStatus;
