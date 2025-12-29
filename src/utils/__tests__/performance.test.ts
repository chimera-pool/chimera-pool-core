import {
  debounce,
  throttle,
  getVisibleItems,
  RequestPool,
  LRUCache,
  getPerformanceRating,
  formatBytes,
  formatDuration,
  batchUpdates,
} from '../performance';

describe('Performance Utilities', () => {
  describe('debounce', () => {
    jest.useFakeTimers();

    it('should debounce function calls', () => {
      const fn = jest.fn();
      const debounced = debounce(fn, 100);

      debounced();
      debounced();
      debounced();

      expect(fn).not.toHaveBeenCalled();

      jest.advanceTimersByTime(100);

      expect(fn).toHaveBeenCalledTimes(1);
    });

    it('should pass arguments to debounced function', () => {
      const fn = jest.fn();
      const debounced = debounce(fn, 100);

      debounced('arg1', 'arg2');
      jest.advanceTimersByTime(100);

      expect(fn).toHaveBeenCalledWith('arg1', 'arg2');
    });
  });

  describe('throttle', () => {
    jest.useFakeTimers();

    it('should throttle function calls', () => {
      const fn = jest.fn();
      const throttled = throttle(fn, 100);

      throttled();
      throttled();
      throttled();

      expect(fn).toHaveBeenCalledTimes(1);

      jest.advanceTimersByTime(100);
      throttled();

      expect(fn).toHaveBeenCalledTimes(2);
    });
  });

  describe('getVisibleItems', () => {
    it('should return visible items for virtual scrolling', () => {
      const items = Array.from({ length: 100 }, (_, i) => i);
      const result = getVisibleItems(items, 0, 500, 50, 2);

      expect(result.startIndex).toBe(0);
      expect(result.visibleItems.length).toBeLessThanOrEqual(15); // 10 visible + overscan
      expect(result.offsetY).toBe(0);
    });

    it('should handle scroll position', () => {
      const items = Array.from({ length: 100 }, (_, i) => i);
      const result = getVisibleItems(items, 500, 500, 50, 2);

      expect(result.startIndex).toBeGreaterThan(0);
      expect(result.offsetY).toBeGreaterThan(0);
    });

    it('should handle empty items', () => {
      const result = getVisibleItems([], 0, 500, 50);

      expect(result.visibleItems).toHaveLength(0);
      expect(result.startIndex).toBe(0);
      expect(result.endIndex).toBe(0);
    });
  });

  describe('RequestPool', () => {
    it('should limit concurrent requests', async () => {
      const pool = new RequestPool(2);
      let concurrent = 0;
      let maxConcurrent = 0;

      const makeRequest = () => {
        concurrent++;
        maxConcurrent = Math.max(maxConcurrent, concurrent);
        return Promise.resolve().then(() => {
          concurrent--;
        });
      };

      await Promise.all([
        pool.add(makeRequest),
        pool.add(makeRequest),
        pool.add(makeRequest),
        pool.add(makeRequest),
      ]);

      expect(maxConcurrent).toBeLessThanOrEqual(2);
    });

    it('should return correct initial state', () => {
      const pool = new RequestPool(2);
      expect(pool.pending).toBe(0);
      expect(pool.running).toBe(0);
    });
  });

  describe('LRUCache', () => {
    it('should store and retrieve values', () => {
      const cache = new LRUCache<string, number>(3);
      
      cache.set('a', 1);
      cache.set('b', 2);
      
      expect(cache.get('a')).toBe(1);
      expect(cache.get('b')).toBe(2);
    });

    it('should evict least recently used item', () => {
      const cache = new LRUCache<string, number>(2);
      
      cache.set('a', 1);
      cache.set('b', 2);
      cache.get('a'); // Access 'a' to make it recently used
      cache.set('c', 3); // Should evict 'b'
      
      expect(cache.has('a')).toBe(true);
      expect(cache.has('b')).toBe(false);
      expect(cache.has('c')).toBe(true);
    });

    it('should track size', () => {
      const cache = new LRUCache<string, number>(5);
      
      cache.set('a', 1);
      cache.set('b', 2);
      
      expect(cache.size).toBe(2);
      
      cache.delete('a');
      
      expect(cache.size).toBe(1);
    });

    it('should clear all items', () => {
      const cache = new LRUCache<string, number>(5);
      
      cache.set('a', 1);
      cache.set('b', 2);
      cache.clear();
      
      expect(cache.size).toBe(0);
    });
  });

  describe('getPerformanceRating', () => {
    it('should return good for values below threshold', () => {
      expect(getPerformanceRating('fcp', 1000)).toBe('good');
      expect(getPerformanceRating('lcp', 2000)).toBe('good');
    });

    it('should return needs-improvement for middle values', () => {
      expect(getPerformanceRating('fcp', 2500)).toBe('needs-improvement');
    });

    it('should return poor for high values', () => {
      expect(getPerformanceRating('fcp', 5000)).toBe('poor');
      expect(getPerformanceRating('lcp', 5000)).toBe('poor');
    });
  });

  describe('formatBytes', () => {
    it('should format bytes correctly', () => {
      expect(formatBytes(0)).toBe('0 B');
      expect(formatBytes(500)).toBe('500 B');
      expect(formatBytes(1024)).toBe('1 KB');
      expect(formatBytes(1024 * 1024)).toBe('1 MB');
      expect(formatBytes(1024 * 1024 * 1024)).toBe('1 GB');
    });
  });

  describe('formatDuration', () => {
    it('should format milliseconds', () => {
      expect(formatDuration(500)).toBe('500ms');
    });

    it('should format seconds', () => {
      expect(formatDuration(2500)).toBe('2.5s');
    });

    it('should format minutes', () => {
      expect(formatDuration(90000)).toBe('1.5m');
    });
  });

  describe('batchUpdates', () => {
    jest.useFakeTimers();

    it('should process items in batches', () => {
      const items = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10];
      const batches: number[][] = [];
      
      batchUpdates(items, 3, (batch) => {
        batches.push([...batch]);
      }, 0);

      // Process all RAF callbacks
      jest.runAllTimers();

      expect(batches.length).toBe(4); // 3 + 3 + 3 + 1
      expect(batches[0]).toEqual([1, 2, 3]);
      expect(batches[3]).toEqual([10]);
    });
  });
});
