import { useState, useEffect, useCallback, useRef } from 'react';
import { IGrafanaStatus } from '../interfaces/IGrafanaPanel';

const HEALTH_CHECK_TIMEOUT = 5000; // 5 seconds

/**
 * Hook to check Grafana availability
 * Only checks once on mount to avoid excessive polling that causes browser crashes
 */
export function useGrafanaHealth(baseUrl: string) {
  const [status, setStatus] = useState<IGrafanaStatus>({
    available: true, // Assume available initially
    lastCheck: new Date(),
  });
  const [isChecking, setIsChecking] = useState(false);
  const hasChecked = useRef(false);

  /**
   * Check if Grafana is reachable
   */
  const checkHealth = useCallback(async () => {
    if (isChecking) return;
    
    setIsChecking(true);
    
    try {
      const controller = new AbortController();
      const timeout = setTimeout(() => controller.abort(), HEALTH_CHECK_TIMEOUT);
      
      // Try to reach Grafana's health endpoint
      const response = await fetch(`${baseUrl}/api/health`, {
        method: 'GET',
        signal: controller.signal,
        mode: 'no-cors', // Grafana may not have CORS configured
      });
      
      clearTimeout(timeout);
      
      // In no-cors mode, we can't read the response, but if we got here, it's reachable
      setStatus({
        available: true,
        lastCheck: new Date(),
      });
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : 'Unknown error';
      setStatus({
        available: false,
        lastCheck: new Date(),
        error: errorMessage,
      });
    } finally {
      setIsChecking(false);
    }
  }, [baseUrl, isChecking]);

  // Single health check on mount only - no continuous polling
  // This prevents the excessive network requests that crash the browser
  useEffect(() => {
    if (!hasChecked.current) {
      hasChecked.current = true;
      checkHealth();
    }
  }, [checkHealth]);

  /**
   * Force a health check (manual refresh only)
   */
  const refresh = useCallback(() => {
    checkHealth();
  }, [checkHealth]);

  return {
    ...status,
    isChecking,
    refresh,
  };
}

export default useGrafanaHealth;
