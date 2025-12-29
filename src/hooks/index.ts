// ============================================================================
// HOOKS INDEX
// Central export for all React hooks
// Following Interface Segregation Principle - import only what you need
// ============================================================================

// Core data hooks
export { useAutoRefresh } from './useAutoRefresh';
export { useFetch, useAuthFetch } from './useFetch';
export { usePoolStats } from './usePoolStats';
export { useUserEquipmentStatus } from './useUserEquipmentStatus';
export { useWebSocket } from './useWebSocket';
export { useRateLimit, createRateLimitedFetch } from './useRateLimit';

// Accessibility hooks
export {
  useFocusTrap,
  useAnnounce,
  useAriaId,
  useKeyboardHandler,
  useReducedMotion,
  useInputA11y,
  useButtonA11y,
  useFocusOnMount,
  useReturnFocus,
} from './useAccessibility';

// Animation hooks
export {
  useAnimationSetup,
  useFadeAnimation,
  useSlideAnimation,
  useScaleAnimation,
  useStaggeredAnimation,
  useTransitionStyles,
  usePulseAnimation,
  useSpinAnimation,
  useDelayedVisibility,
} from './useAnimations';

// Performance hooks
export {
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
} from './usePerformance';

// Types
export type { PoolStats } from './usePoolStats';
export type { RateLimitInfo, RateLimitStatus, UseRateLimitReturn } from './useRateLimit';
