import { renderHook, act } from '@testing-library/react';
import { useRateLimit } from '../useRateLimit';

describe('useRateLimit', () => {
  describe('initial state', () => {
    it('should have null info initially', () => {
      const { result } = renderHook(() => useRateLimit());
      expect(result.current.info).toBeNull();
    });

    it('should have ok status initially', () => {
      const { result } = renderHook(() => useRateLimit());
      expect(result.current.status).toBe('ok');
    });

    it('should not be limited initially', () => {
      const { result } = renderHook(() => useRateLimit());
      expect(result.current.isLimited).toBe(false);
    });

    it('should have 0 usage percent initially', () => {
      const { result } = renderHook(() => useRateLimit());
      expect(result.current.usagePercent).toBe(0);
    });
  });

  describe('updateFromHeaders', () => {
    it('should update info from headers', () => {
      const { result } = renderHook(() => useRateLimit());
      
      const headers = new Headers({
        'X-RateLimit-Limit': '100',
        'X-RateLimit-Remaining': '50',
        'X-RateLimit-Reset': String(Math.floor(Date.now() / 1000) + 60),
      });

      act(() => {
        result.current.updateFromHeaders(headers);
      });

      expect(result.current.info).not.toBeNull();
      expect(result.current.info?.limit).toBe(100);
      expect(result.current.info?.remaining).toBe(50);
    });

    it('should not update if headers are missing', () => {
      const { result } = renderHook(() => useRateLimit());
      
      const headers = new Headers({});

      act(() => {
        result.current.updateFromHeaders(headers);
      });

      expect(result.current.info).toBeNull();
    });
  });

  describe('status calculation', () => {
    it('should return ok when plenty remaining', () => {
      const { result } = renderHook(() => useRateLimit());
      
      const headers = new Headers({
        'X-RateLimit-Limit': '100',
        'X-RateLimit-Remaining': '80',
        'X-RateLimit-Reset': String(Math.floor(Date.now() / 1000) + 60),
      });

      act(() => {
        result.current.updateFromHeaders(headers);
      });

      expect(result.current.status).toBe('ok');
    });

    it('should return warning when low remaining', () => {
      const { result } = renderHook(() => useRateLimit());
      
      const headers = new Headers({
        'X-RateLimit-Limit': '100',
        'X-RateLimit-Remaining': '10',
        'X-RateLimit-Reset': String(Math.floor(Date.now() / 1000) + 60),
      });

      act(() => {
        result.current.updateFromHeaders(headers);
      });

      expect(result.current.status).toBe('warning');
    });

    it('should return exceeded when none remaining', () => {
      const { result } = renderHook(() => useRateLimit());
      
      const headers = new Headers({
        'X-RateLimit-Limit': '100',
        'X-RateLimit-Remaining': '0',
        'X-RateLimit-Reset': String(Math.floor(Date.now() / 1000) + 60),
      });

      act(() => {
        result.current.updateFromHeaders(headers);
      });

      expect(result.current.status).toBe('exceeded');
      expect(result.current.isLimited).toBe(true);
    });
  });

  describe('usage percent', () => {
    it('should calculate usage percent correctly', () => {
      const { result } = renderHook(() => useRateLimit());
      
      const headers = new Headers({
        'X-RateLimit-Limit': '100',
        'X-RateLimit-Remaining': '25',
        'X-RateLimit-Reset': String(Math.floor(Date.now() / 1000) + 60),
      });

      act(() => {
        result.current.updateFromHeaders(headers);
      });

      expect(result.current.usagePercent).toBe(75);
    });
  });

  describe('reset', () => {
    it('should reset all state', () => {
      const { result } = renderHook(() => useRateLimit());
      
      const headers = new Headers({
        'X-RateLimit-Limit': '100',
        'X-RateLimit-Remaining': '0',
        'X-RateLimit-Reset': String(Math.floor(Date.now() / 1000) + 60),
      });

      act(() => {
        result.current.updateFromHeaders(headers);
      });

      expect(result.current.isLimited).toBe(true);

      act(() => {
        result.current.reset();
      });

      expect(result.current.info).toBeNull();
      expect(result.current.isLimited).toBe(false);
      expect(result.current.status).toBe('ok');
    });
  });

  describe('updateFromResponse', () => {
    it('should extract headers from response', () => {
      const { result } = renderHook(() => useRateLimit());
      
      const response = new Response(null, {
        headers: {
          'X-RateLimit-Limit': '100',
          'X-RateLimit-Remaining': '50',
          'X-RateLimit-Reset': String(Math.floor(Date.now() / 1000) + 60),
        },
      });

      act(() => {
        result.current.updateFromResponse(response);
      });

      expect(result.current.info?.limit).toBe(100);
      expect(result.current.info?.remaining).toBe(50);
    });
  });
});
