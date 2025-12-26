import { useState, useEffect, useCallback, useRef } from 'react';

interface FetchState<T> {
  data: T | null;
  loading: boolean;
  error: Error | null;
}

interface UseFetchOptions {
  immediate?: boolean;
  refreshInterval?: number;
  headers?: Record<string, string>;
  onSuccess?: (data: any) => void;
  onError?: (error: Error) => void;
}

interface UseFetchReturn<T> extends FetchState<T> {
  refetch: () => Promise<void>;
  reset: () => void;
}

/**
 * Custom hook for data fetching with automatic state management
 * Follows ISP - single responsibility for HTTP GET requests
 * 
 * @param url - The URL to fetch from
 * @param options - Configuration options
 * @returns Fetch state and control functions
 */
export function useFetch<T = any>(
  url: string | null,
  options: UseFetchOptions = {}
): UseFetchReturn<T> {
  const {
    immediate = true,
    refreshInterval,
    headers = {},
    onSuccess,
    onError,
  } = options;

  const [state, setState] = useState<FetchState<T>>({
    data: null,
    loading: immediate && !!url,
    error: null,
  });

  const isMountedRef = useRef(true);
  const abortControllerRef = useRef<AbortController | null>(null);

  const fetchData = useCallback(async () => {
    if (!url) return;

    // Cancel any pending request
    if (abortControllerRef.current) {
      abortControllerRef.current.abort();
    }

    abortControllerRef.current = new AbortController();

    setState(prev => ({ ...prev, loading: true, error: null }));

    try {
      const response = await fetch(url, {
        headers: {
          'Content-Type': 'application/json',
          ...headers,
        },
        signal: abortControllerRef.current.signal,
      });

      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`);
      }

      const data = await response.json();

      if (isMountedRef.current) {
        setState({ data, loading: false, error: null });
        onSuccess?.(data);
      }
    } catch (error) {
      if (error instanceof Error && error.name === 'AbortError') {
        return; // Request was cancelled, don't update state
      }

      if (isMountedRef.current) {
        const err = error instanceof Error ? error : new Error('Unknown error');
        setState(prev => ({ ...prev, loading: false, error: err }));
        onError?.(err);
      }
    }
  }, [url, headers, onSuccess, onError]);

  const reset = useCallback(() => {
    setState({ data: null, loading: false, error: null });
  }, []);

  // Initial fetch
  useEffect(() => {
    isMountedRef.current = true;

    if (immediate && url) {
      fetchData();
    }

    return () => {
      isMountedRef.current = false;
      if (abortControllerRef.current) {
        abortControllerRef.current.abort();
      }
    };
  }, [url, immediate, fetchData]);

  // Refresh interval
  useEffect(() => {
    if (!refreshInterval || !url) return;

    const intervalId = setInterval(fetchData, refreshInterval);
    return () => clearInterval(intervalId);
  }, [refreshInterval, url, fetchData]);

  return {
    ...state,
    refetch: fetchData,
    reset,
  };
}

/**
 * Custom hook for authenticated API requests
 * Automatically includes the auth token in headers
 */
export function useAuthFetch<T = any>(
  url: string | null,
  token: string | null,
  options: UseFetchOptions = {}
): UseFetchReturn<T> {
  const authHeaders = token
    ? { Authorization: `Bearer ${token}`, ...options.headers }
    : options.headers;

  return useFetch<T>(url, {
    ...options,
    headers: authHeaders,
    immediate: options.immediate !== false && !!token,
  });
}

export default useFetch;
