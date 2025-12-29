import { useState, useEffect, useCallback, useRef } from 'react';
import {
  transitions,
  animationStyles,
  getAnimationStyles,
  injectKeyframes,
  staggerDelay,
  type AnimationState,
} from '../utils/animations';
import { useReducedMotion } from './useAccessibility';

// ============================================================================
// ANIMATION HOOKS
// React hooks for smooth, accessible animations
// Following Interface Segregation Principle
// ============================================================================

/** Initialize animation keyframes (call once at app root) */
export function useAnimationSetup() {
  useEffect(() => {
    injectKeyframes();
  }, []);
}

/** Fade animation hook */
export function useFadeAnimation(isVisible: boolean, duration: number = 250) {
  const [state, setState] = useState<AnimationState>(isVisible ? 'entered' : 'exited');
  const reducedMotion = useReducedMotion();

  useEffect(() => {
    if (isVisible) {
      setState('entering');
      const timer = setTimeout(() => setState('entered'), reducedMotion ? 0 : duration);
      return () => clearTimeout(timer);
    } else {
      setState('exiting');
      const timer = setTimeout(() => setState('exited'), reducedMotion ? 0 : duration);
      return () => clearTimeout(timer);
    }
  }, [isVisible, duration, reducedMotion]);

  return {
    state,
    styles: getAnimationStyles(state, 'fade'),
    isVisible: state !== 'exited',
    isAnimating: state === 'entering' || state === 'exiting',
  };
}

/** Slide animation hook */
export function useSlideAnimation(isVisible: boolean, duration: number = 300) {
  const [state, setState] = useState<AnimationState>(isVisible ? 'entered' : 'exited');
  const reducedMotion = useReducedMotion();

  useEffect(() => {
    if (isVisible) {
      setState('entering');
      const timer = setTimeout(() => setState('entered'), reducedMotion ? 0 : duration);
      return () => clearTimeout(timer);
    } else {
      setState('exiting');
      const timer = setTimeout(() => setState('exited'), reducedMotion ? 0 : duration);
      return () => clearTimeout(timer);
    }
  }, [isVisible, duration, reducedMotion]);

  return {
    state,
    styles: getAnimationStyles(state, 'slide'),
    isVisible: state !== 'exited',
    isAnimating: state === 'entering' || state === 'exiting',
  };
}

/** Scale animation hook */
export function useScaleAnimation(isVisible: boolean, duration: number = 200) {
  const [state, setState] = useState<AnimationState>(isVisible ? 'entered' : 'exited');
  const reducedMotion = useReducedMotion();

  useEffect(() => {
    if (isVisible) {
      setState('entering');
      const timer = setTimeout(() => setState('entered'), reducedMotion ? 0 : duration);
      return () => clearTimeout(timer);
    } else {
      setState('exiting');
      const timer = setTimeout(() => setState('exited'), reducedMotion ? 0 : duration);
      return () => clearTimeout(timer);
    }
  }, [isVisible, duration, reducedMotion]);

  return {
    state,
    styles: getAnimationStyles(state, 'scale'),
    isVisible: state !== 'exited',
    isAnimating: state === 'entering' || state === 'exiting',
  };
}

/** Staggered list animation hook */
export function useStaggeredAnimation(
  items: any[],
  baseDelay: number = 50
) {
  const reducedMotion = useReducedMotion();

  const getItemDelay = useCallback(
    (index: number) => {
      return reducedMotion ? '0ms' : staggerDelay(index, baseDelay);
    },
    [baseDelay, reducedMotion]
  );

  const getItemStyle = useCallback(
    (index: number, delay?: number): React.CSSProperties => {
      if (reducedMotion) {
        return { opacity: 1 };
      }
      return animationStyles.fadeIn(delay ?? index * baseDelay);
    },
    [baseDelay, reducedMotion]
  );

  return {
    getItemDelay,
    getItemStyle,
    itemCount: items.length,
  };
}

/** Transition styles hook */
export function useTransitionStyles() {
  const reducedMotion = useReducedMotion();

  if (reducedMotion) {
    return {
      fade: { transition: 'none' },
      slide: { transition: 'none' },
      scale: { transition: 'none' },
      interactive: { transition: 'none' },
      all: { transition: 'none' },
    };
  }

  return {
    fade: { transition: transitions.fade },
    slide: { transition: transitions.slide },
    scale: { transition: transitions.scale },
    interactive: { transition: transitions.interactive },
    all: { transition: transitions.all },
  };
}

/** Pulse animation hook for loading states */
export function usePulseAnimation(isActive: boolean = true) {
  const reducedMotion = useReducedMotion();

  if (!isActive || reducedMotion) {
    return { style: {} };
  }

  return {
    style: animationStyles.pulse(),
  };
}

/** Spin animation hook for loading spinners */
export function useSpinAnimation(isActive: boolean = true) {
  const reducedMotion = useReducedMotion();

  if (!isActive || reducedMotion) {
    return { style: {} };
  }

  return {
    style: animationStyles.spin(),
  };
}

/** Delayed visibility hook for smoother loading */
export function useDelayedVisibility(isLoading: boolean, minDelay: number = 300) {
  const [showLoader, setShowLoader] = useState(false);
  const timerRef = useRef<NodeJS.Timeout | null>(null);

  useEffect(() => {
    if (isLoading) {
      timerRef.current = setTimeout(() => {
        setShowLoader(true);
      }, minDelay);
    } else {
      if (timerRef.current) {
        clearTimeout(timerRef.current);
      }
      setShowLoader(false);
    }

    return () => {
      if (timerRef.current) {
        clearTimeout(timerRef.current);
      }
    };
  }, [isLoading, minDelay]);

  return showLoader;
}

export default {
  useAnimationSetup,
  useFadeAnimation,
  useSlideAnimation,
  useScaleAnimation,
  useStaggeredAnimation,
  useTransitionStyles,
  usePulseAnimation,
  useSpinAnimation,
  useDelayedVisibility,
};
