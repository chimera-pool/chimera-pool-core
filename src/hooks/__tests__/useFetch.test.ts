import { renderHook } from '@testing-library/react';
import { useFetch, useAuthFetch } from '../useFetch';

/**
 * useFetch Hook Tests
 * Simplified tests to avoid memory issues in CI
 */
describe('useFetch', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    global.fetch = jest.fn();
  });

  afterEach(() => {
    jest.restoreAllMocks();
  });

  it('should initialize with loading false when immediate is false', () => {
    const { result } = renderHook(() => 
      useFetch('/api/test', { immediate: false })
    );

    expect(result.current.loading).toBe(false);
    expect(result.current.data).toBeNull();
    expect(result.current.error).toBeNull();
  });

  it('should not fetch when url is null', () => {
    const { result } = renderHook(() => useFetch(null));

    expect(result.current.loading).toBe(false);
    expect(global.fetch).not.toHaveBeenCalled();
  });

  it('should provide refetch and reset functions', () => {
    const { result } = renderHook(() => 
      useFetch('/api/test', { immediate: false })
    );

    expect(typeof result.current.refetch).toBe('function');
    expect(typeof result.current.reset).toBe('function');
  });
});

describe('useAuthFetch', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    global.fetch = jest.fn();
  });

  it('should not fetch when token is null', () => {
    const { result } = renderHook(() => useAuthFetch('/api/user', null));

    expect(result.current.loading).toBe(false);
    expect(global.fetch).not.toHaveBeenCalled();
  });

  it('should not fetch when token is empty string', () => {
    const { result } = renderHook(() => useAuthFetch('/api/user', ''));

    expect(result.current.loading).toBe(false);
    expect(global.fetch).not.toHaveBeenCalled();
  });
});
