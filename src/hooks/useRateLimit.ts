import { useState, useCallback, useEffect, useRef } from 'react';

// ============================================================================
// RATE LIMIT HOOK
// Track and display rate limiting status for API calls
// Following Interface Segregation Principle
// ============================================================================

/** Rate limit info from API response headers */
export interface RateLimitInfo {
  /** Maximum requests allowed in the window */
  limit: number;
  /** Remaining requests in current window */
  remaining: number;
  /** Unix timestamp when the window resets */
  reset: number;
  /** Time until reset in seconds */
  retryAfter?: number;
}

/** Rate limit status */
export type RateLimitStatus = 'ok' | 'warning' | 'exceeded';

/** Hook return type */
export interface UseRateLimitReturn {
  /** Current rate limit info */
  info: RateLimitInfo | null;
  /** Current status (ok, warning, exceeded) */
  status: RateLimitStatus;
  /** Whether rate limit is exceeded */
  isLimited: boolean;
  /** Percentage of limit used (0-100) */
  usagePercent: number;
  /** Time until reset in human-readable format */
  timeUntilReset: string;
  /** Update rate limit info from response headers */
  updateFromHeaders: (headers: Headers) => void;
  /** Update rate limit info from API response */
  updateFromResponse: (response: Response) => void;
  /** Reset rate limit tracking */
  reset: () => void;
}

/** Parse rate limit headers from response */
function parseRateLimitHeaders(headers: Headers): RateLimitInfo | null {
  const limit = headers.get('X-RateLimit-Limit');
  const remaining = headers.get('X-RateLimit-Remaining');
  const reset = headers.get('X-RateLimit-Reset');
  const retryAfter = headers.get('Retry-After');

  if (!limit || !remaining) {
    return null;
  }

  return {
    limit: parseInt(limit, 10),
    remaining: parseInt(remaining, 10),
    reset: reset ? parseInt(reset, 10) : Date.now() / 1000 + 60,
    retryAfter: retryAfter ? parseInt(retryAfter, 10) : undefined,
  };
}

/** Format seconds into human-readable time */
function formatTimeUntilReset(seconds: number): string {
  if (seconds <= 0) return 'now';
  if (seconds < 60) return `${Math.ceil(seconds)}s`;
  if (seconds < 3600) return `${Math.ceil(seconds / 60)}m`;
  return `${Math.ceil(seconds / 3600)}h`;
}

/** Get status based on remaining requests */
function getStatus(info: RateLimitInfo | null): RateLimitStatus {
  if (!info) return 'ok';
  if (info.remaining <= 0) return 'exceeded';
  if (info.remaining / info.limit < 0.2) return 'warning';
  return 'ok';
}

/** Rate limit hook options */
export interface UseRateLimitOptions {
  /** Warning threshold (0-1, default 0.2 = 20% remaining) */
  warningThreshold?: number;
  /** Auto-reset when window expires */
  autoReset?: boolean;
}

/** Hook for tracking rate limit status */
export function useRateLimit(options: UseRateLimitOptions = {}): UseRateLimitReturn {
  const { warningThreshold = 0.2, autoReset = true } = options;
  
  const [info, setInfo] = useState<RateLimitInfo | null>(null);
  const [timeUntilReset, setTimeUntilReset] = useState('');
  const resetTimerRef = useRef<NodeJS.Timeout | null>(null);

  // Update time until reset every second
  useEffect(() => {
    if (!info) {
      setTimeUntilReset('');
      return;
    }

    const updateTime = () => {
      const now = Date.now() / 1000;
      const seconds = Math.max(0, info.reset - now);
      setTimeUntilReset(formatTimeUntilReset(seconds));

      // Auto-reset when window expires
      if (autoReset && seconds <= 0) {
        setInfo(null);
      }
    };

    updateTime();
    const interval = setInterval(updateTime, 1000);

    return () => clearInterval(interval);
  }, [info, autoReset]);

  // Clean up timer on unmount
  useEffect(() => {
    return () => {
      if (resetTimerRef.current) {
        clearTimeout(resetTimerRef.current);
      }
    };
  }, []);

  const updateFromHeaders = useCallback((headers: Headers) => {
    const newInfo = parseRateLimitHeaders(headers);
    if (newInfo) {
      setInfo(newInfo);
    }
  }, []);

  const updateFromResponse = useCallback((response: Response) => {
    updateFromHeaders(response.headers);
  }, [updateFromHeaders]);

  const reset = useCallback(() => {
    setInfo(null);
    setTimeUntilReset('');
  }, []);

  // Calculate derived values
  const status = getStatus(info);
  const isLimited = info ? info.remaining <= 0 : false;
  const usagePercent = info ? Math.round(((info.limit - info.remaining) / info.limit) * 100) : 0;

  return {
    info,
    status,
    isLimited,
    usagePercent,
    timeUntilReset,
    updateFromHeaders,
    updateFromResponse,
    reset,
  };
}

/** Helper to wrap fetch with rate limit tracking */
export function createRateLimitedFetch(
  updateFromResponse: (response: Response) => void
): typeof fetch {
  return async (input: RequestInfo | URL, init?: RequestInit): Promise<Response> => {
    const response = await fetch(input, init);
    updateFromResponse(response);
    return response;
  };
}

/** Rate limit status indicator component props */
export interface RateLimitIndicatorProps {
  status: RateLimitStatus;
  remaining?: number;
  limit?: number;
  timeUntilReset?: string;
}

export default useRateLimit;
