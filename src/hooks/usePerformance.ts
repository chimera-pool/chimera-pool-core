import { useState, useEffect, useCallback, useRef, useMemo } from 'react';
import {
  debounce,
  throttle,
  rafThrottle,
  getVisibleItems,
  RequestPool,
  LRUCache,
  getWebVitals,
  formatBytes,
  formatDuration,
  type PerformanceMetrics,
} from '../utils/performance';

// ============================================================================
// PERFORMANCE HOOKS
// React hooks for performance optimization
// Designed for 10K+ miner scaling
// ============================================================================

/** Debounced value hook */
export function useDebounce<T>(value: T, delay: number = 300): T {
  const [debouncedValue, setDebouncedValue] = useState(value);

  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedValue(value);
    }, delay);

    return () => clearTimeout(timer);
  }, [value, delay]);

  return debouncedValue;
}

/** Debounced callback hook */
export function useDebouncedCallback<T extends (...args: any[]) => any>(
  callback: T,
  delay: number = 300
): T {
  const callbackRef = useRef(callback);
  callbackRef.current = callback;

  const debouncedFn = useMemo(
    () => debounce((...args: Parameters<T>) => callbackRef.current(...args), delay),
    [delay]
  );

  return debouncedFn as T;
}

/** Throttled callback hook */
export function useThrottledCallback<T extends (...args: any[]) => any>(
  callback: T,
  limit: number = 100
): T {
  const callbackRef = useRef(callback);
  callbackRef.current = callback;

  const throttledFn = useMemo(
    () => throttle((...args: Parameters<T>) => callbackRef.current(...args), limit),
    [limit]
  );

  return throttledFn as T;
}

/** RAF throttled callback hook */
export function useRAFCallback<T extends (...args: any[]) => any>(callback: T): T {
  const callbackRef = useRef(callback);
  callbackRef.current = callback;

  const rafFn = useMemo(
    () => rafThrottle((...args: Parameters<T>) => callbackRef.current(...args)),
    []
  );

  return rafFn as T;
}

/** Virtual scrolling hook */
export function useVirtualScroll<T>(
  items: T[],
  containerHeight: number,
  itemHeight: number,
  overscan: number = 3
) {
  const [scrollTop, setScrollTop] = useState(0);

  const handleScroll = useCallback((e: React.UIEvent<HTMLElement>) => {
    setScrollTop(e.currentTarget.scrollTop);
  }, []);

  const { visibleItems, startIndex, endIndex, offsetY } = useMemo(
    () => getVisibleItems(items, scrollTop, containerHeight, itemHeight, overscan),
    [items, scrollTop, containerHeight, itemHeight, overscan]
  );

  const totalHeight = items.length * itemHeight;

  return {
    visibleItems,
    startIndex,
    endIndex,
    offsetY,
    totalHeight,
    handleScroll,
  };
}

/** Request pool hook for concurrent API calls */
export function useRequestPool(maxConcurrent: number = 6) {
  const poolRef = useRef(new RequestPool(maxConcurrent));

  const add = useCallback(<T>(request: () => Promise<T>): Promise<T> => {
    return poolRef.current.add(request);
  }, []);

  return {
    add,
    get pending() {
      return poolRef.current.pending;
    },
    get running() {
      return poolRef.current.running;
    },
  };
}

/** LRU cache hook */
export function useCache<K, V>(maxSize: number = 100) {
  const cacheRef = useRef(new LRUCache<K, V>(maxSize));

  const get = useCallback((key: K): V | undefined => {
    return cacheRef.current.get(key);
  }, []);

  const set = useCallback((key: K, value: V): void => {
    cacheRef.current.set(key, value);
  }, []);

  const has = useCallback((key: K): boolean => {
    return cacheRef.current.has(key);
  }, []);

  const remove = useCallback((key: K): boolean => {
    return cacheRef.current.delete(key);
  }, []);

  const clear = useCallback((): void => {
    cacheRef.current.clear();
  }, []);

  return {
    get,
    set,
    has,
    remove,
    clear,
    get size() {
      return cacheRef.current.size;
    },
  };
}

/** Web Vitals monitoring hook */
export function useWebVitals() {
  const [metrics, setMetrics] = useState<Partial<PerformanceMetrics>>({});

  useEffect(() => {
    getWebVitals().then(setMetrics);
  }, []);

  return metrics;
}

/** Intersection observer hook for lazy loading */
export function useIntersectionObserver(
  options: IntersectionObserverInit = {}
) {
  const [isIntersecting, setIsIntersecting] = useState(false);
  const [hasIntersected, setHasIntersected] = useState(false);
  const elementRef = useRef<HTMLElement | null>(null);

  useEffect(() => {
    const element = elementRef.current;
    if (!element) return;

    const observer = new IntersectionObserver(([entry]) => {
      setIsIntersecting(entry.isIntersecting);
      if (entry.isIntersecting) {
        setHasIntersected(true);
      }
    }, options);

    observer.observe(element);

    return () => observer.disconnect();
  }, [options.threshold, options.root, options.rootMargin]);

  return {
    ref: elementRef,
    isIntersecting,
    hasIntersected,
  };
}

/** Memory monitoring hook */
export function useMemoryMonitor(intervalMs: number = 5000) {
  const [memory, setMemory] = useState<number | null>(null);

  useEffect(() => {
    const updateMemory = () => {
      const mem = (performance as any).memory;
      if (mem) {
        setMemory(Math.round(mem.usedJSHeapSize / 1024 / 1024));
      }
    };

    updateMemory();
    const interval = setInterval(updateMemory, intervalMs);

    return () => clearInterval(interval);
  }, [intervalMs]);

  return {
    memoryMB: memory,
    memoryFormatted: memory ? `${memory} MB` : null,
  };
}

/** Previous value hook for comparison */
export function usePrevious<T>(value: T): T | undefined {
  const ref = useRef<T>();
  
  useEffect(() => {
    ref.current = value;
  }, [value]);
  
  return ref.current;
}

/** Stable callback hook (prevents unnecessary re-renders) */
export function useStableCallback<T extends (...args: any[]) => any>(callback: T): T {
  const callbackRef = useRef(callback);
  
  useEffect(() => {
    callbackRef.current = callback;
  }, [callback]);
  
  return useCallback((...args: Parameters<T>) => {
    return callbackRef.current(...args);
  }, []) as T;
}

/** Format helpers */
export { formatBytes, formatDuration };

export default {
  useDebounce,
  useDebouncedCallback,
  useThrottledCallback,
  useRAFCallback,
  useVirtualScroll,
  useRequestPool,
  useCache,
  useWebVitals,
  useIntersectionObserver,
  useMemoryMonitor,
  usePrevious,
  useStableCallback,
  formatBytes,
  formatDuration,
};
