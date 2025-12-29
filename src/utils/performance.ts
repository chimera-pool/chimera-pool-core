// ============================================================================
// PERFORMANCE UTILITIES
// Monitoring and optimization helpers for high-scale mining pool
// Designed for 10,000+ concurrent connections
// ============================================================================

/** Performance metrics interface */
export interface PerformanceMetrics {
  /** Time to first contentful paint (ms) */
  fcp: number | null;
  /** Largest contentful paint (ms) */
  lcp: number | null;
  /** First input delay (ms) */
  fid: number | null;
  /** Cumulative layout shift */
  cls: number | null;
  /** Time to interactive (ms) */
  tti: number | null;
  /** Total blocking time (ms) */
  tbt: number | null;
  /** Memory usage (MB) */
  memory: number | null;
  /** DOM nodes count */
  domNodes: number;
  /** Active WebSocket connections */
  wsConnections: number;
}

/** Request timing info */
export interface RequestTiming {
  url: string;
  method: string;
  duration: number;
  size: number;
  status: number;
  timestamp: number;
}

/** Performance thresholds for alerts */
export const performanceThresholds = {
  fcp: { good: 1800, poor: 3000 },
  lcp: { good: 2500, poor: 4000 },
  fid: { good: 100, poor: 300 },
  cls: { good: 0.1, poor: 0.25 },
  tti: { good: 3800, poor: 7300 },
  memory: { good: 100, poor: 500 }, // MB
  domNodes: { good: 1500, poor: 3000 },
} as const;

/** Get Web Vitals metrics */
export function getWebVitals(): Promise<Partial<PerformanceMetrics>> {
  return new Promise((resolve) => {
    const metrics: Partial<PerformanceMetrics> = {};

    // Get paint timing
    if ('PerformanceObserver' in window) {
      try {
        const paintObserver = new PerformanceObserver((list) => {
          for (const entry of list.getEntries()) {
            if (entry.name === 'first-contentful-paint') {
              metrics.fcp = entry.startTime;
            }
          }
        });
        paintObserver.observe({ entryTypes: ['paint'] });
      } catch (e) {
        // Observer not supported
      }
    }

    // Get memory usage (Chrome only)
    const memory = (performance as any).memory;
    if (memory) {
      metrics.memory = Math.round(memory.usedJSHeapSize / 1024 / 1024);
    }

    // Get DOM node count
    metrics.domNodes = document.querySelectorAll('*').length;

    // Resolve after a short delay to collect metrics
    setTimeout(() => resolve(metrics), 100);
  });
}

/** Debounce function for performance optimization */
export function debounce<T extends (...args: any[]) => any>(
  func: T,
  wait: number
): (...args: Parameters<T>) => void {
  let timeout: NodeJS.Timeout | null = null;

  return (...args: Parameters<T>) => {
    if (timeout) clearTimeout(timeout);
    timeout = setTimeout(() => func(...args), wait);
  };
}

/** Throttle function for performance optimization */
export function throttle<T extends (...args: any[]) => any>(
  func: T,
  limit: number
): (...args: Parameters<T>) => void {
  let inThrottle = false;

  return (...args: Parameters<T>) => {
    if (!inThrottle) {
      func(...args);
      inThrottle = true;
      setTimeout(() => (inThrottle = false), limit);
    }
  };
}

/** Request Animation Frame throttle for smooth animations */
export function rafThrottle<T extends (...args: any[]) => any>(
  func: T
): (...args: Parameters<T>) => void {
  let rafId: number | null = null;

  return (...args: Parameters<T>) => {
    if (rafId) return;
    rafId = requestAnimationFrame(() => {
      func(...args);
      rafId = null;
    });
  };
}

/** Batch updates for React state */
export function batchUpdates<T>(
  items: T[],
  batchSize: number,
  onBatch: (batch: T[], index: number) => void,
  delay: number = 0
): void {
  let index = 0;

  const processBatch = () => {
    const batch = items.slice(index, index + batchSize);
    if (batch.length === 0) return;

    onBatch(batch, index);
    index += batchSize;

    if (index < items.length) {
      if (delay > 0) {
        setTimeout(processBatch, delay);
      } else {
        requestAnimationFrame(processBatch);
      }
    }
  };

  processBatch();
}

/** Virtual scrolling helper - calculate visible items */
export function getVisibleItems<T>(
  items: T[],
  scrollTop: number,
  containerHeight: number,
  itemHeight: number,
  overscan: number = 3
): { visibleItems: T[]; startIndex: number; endIndex: number; offsetY: number } {
  const totalHeight = items.length * itemHeight;
  const startIndex = Math.max(0, Math.floor(scrollTop / itemHeight) - overscan);
  const endIndex = Math.min(
    items.length,
    Math.ceil((scrollTop + containerHeight) / itemHeight) + overscan
  );

  return {
    visibleItems: items.slice(startIndex, endIndex),
    startIndex,
    endIndex,
    offsetY: startIndex * itemHeight,
  };
}

/** Connection pool for managing multiple API requests */
export class RequestPool {
  private queue: Array<() => Promise<any>> = [];
  private active = 0;
  private maxConcurrent: number;

  constructor(maxConcurrent: number = 6) {
    this.maxConcurrent = maxConcurrent;
  }

  async add<T>(request: () => Promise<T>): Promise<T> {
    return new Promise((resolve, reject) => {
      const execute = async () => {
        this.active++;
        try {
          const result = await request();
          resolve(result);
        } catch (error) {
          reject(error);
        } finally {
          this.active--;
          this.processQueue();
        }
      };

      if (this.active < this.maxConcurrent) {
        execute();
      } else {
        this.queue.push(execute);
      }
    });
  }

  private processQueue(): void {
    if (this.queue.length > 0 && this.active < this.maxConcurrent) {
      const next = this.queue.shift();
      if (next) next();
    }
  }

  get pending(): number {
    return this.queue.length;
  }

  get running(): number {
    return this.active;
  }
}

/** Memory-efficient data cache with LRU eviction */
export class LRUCache<K, V> {
  private cache = new Map<K, V>();
  private maxSize: number;

  constructor(maxSize: number = 100) {
    this.maxSize = maxSize;
  }

  get(key: K): V | undefined {
    const value = this.cache.get(key);
    if (value !== undefined) {
      // Move to end (most recently used)
      this.cache.delete(key);
      this.cache.set(key, value);
    }
    return value;
  }

  set(key: K, value: V): void {
    if (this.cache.has(key)) {
      this.cache.delete(key);
    } else if (this.cache.size >= this.maxSize) {
      // Remove oldest (first) item
      const firstKey = this.cache.keys().next().value;
      this.cache.delete(firstKey);
    }
    this.cache.set(key, value);
  }

  has(key: K): boolean {
    return this.cache.has(key);
  }

  delete(key: K): boolean {
    return this.cache.delete(key);
  }

  clear(): void {
    this.cache.clear();
  }

  get size(): number {
    return this.cache.size;
  }
}

/** Performance rating based on thresholds */
export type PerformanceRating = 'good' | 'needs-improvement' | 'poor';

export function getPerformanceRating(
  metric: keyof typeof performanceThresholds,
  value: number
): PerformanceRating {
  const threshold = performanceThresholds[metric];
  if (value <= threshold.good) return 'good';
  if (value <= threshold.poor) return 'needs-improvement';
  return 'poor';
}

/** Format bytes to human readable */
export function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(1))} ${sizes[i]}`;
}

/** Format duration to human readable */
export function formatDuration(ms: number): string {
  if (ms < 1000) return `${Math.round(ms)}ms`;
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
  return `${(ms / 60000).toFixed(1)}m`;
}

export default {
  performanceThresholds,
  getWebVitals,
  debounce,
  throttle,
  rafThrottle,
  batchUpdates,
  getVisibleItems,
  RequestPool,
  LRUCache,
  getPerformanceRating,
  formatBytes,
  formatDuration,
};
