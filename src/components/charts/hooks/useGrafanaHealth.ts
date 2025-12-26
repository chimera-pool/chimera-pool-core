import { useState, useEffect, useCallback } from 'react';
import { IGrafanaStatus } from '../interfaces/IGrafanaPanel';

const HEALTH_CHECK_INTERVAL = 60000; // 1 minute
const HEALTH_CHECK_TIMEOUT = 5000; // 5 seconds

/**
 * Hook to check Grafana availability
 * Falls back to native charts if Grafana is unavailable
 */
export function useGrafanaHealth(baseUrl: string) {
  const [status, setStatus] = useState<IGrafanaStatus>({
    available: true, // Assume available initially
    lastCheck: new Date(),
  });
  const [isChecking, setIsChecking] = useState(false);

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

  // Initial health check
  useEffect(() => {
    checkHealth();
  }, [checkHealth]);

  // Periodic health checks
  useEffect(() => {
    const interval = setInterval(checkHealth, HEALTH_CHECK_INTERVAL);
    return () => clearInterval(interval);
  }, [checkHealth]);

  /**
   * Force a health check
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
