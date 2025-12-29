import { useEffect, useRef, useCallback, useState } from 'react';
import {
  createFocusTrap,
  releaseFocusTrap,
  announceToScreenReader,
  generateAriaId,
  handleKeyboardInteraction,
  getInputAriaProps,
  getButtonAriaProps,
  prefersReducedMotion,
} from '../utils/accessibility';

// ============================================================================
// ACCESSIBILITY HOOKS
// React hooks for accessibility features
// Following Interface Segregation Principle
// ============================================================================

/** Focus trap hook for modals and dialogs */
export function useFocusTrap(isActive: boolean = true) {
  const containerRef = useRef<HTMLElement | null>(null);

  useEffect(() => {
    if (!isActive || !containerRef.current) return;

    createFocusTrap(containerRef.current);

    return () => {
      if (containerRef.current) {
        releaseFocusTrap(containerRef.current);
      }
    };
  }, [isActive]);

  return containerRef;
}

/** Screen reader announcement hook */
export function useAnnounce() {
  const announce = useCallback((message: string, priority: 'polite' | 'assertive' = 'polite') => {
    announceToScreenReader(message, priority);
  }, []);

  return announce;
}

/** Unique ARIA ID generator hook */
export function useAriaId(prefix: string = 'aria') {
  const [id] = useState(() => generateAriaId(prefix));
  return id;
}

/** Keyboard interaction handler hook */
export function useKeyboardHandler(
  onEnter?: () => void,
  onSpace?: () => void,
  onEscape?: () => void
) {
  const handleKeyDown = useCallback(
    (event: React.KeyboardEvent) => {
      handleKeyboardInteraction(event.nativeEvent, onEnter, onSpace, onEscape);
    },
    [onEnter, onSpace, onEscape]
  );

  return handleKeyDown;
}

/** Reduced motion preference hook */
export function useReducedMotion() {
  const [reducedMotion, setReducedMotion] = useState(prefersReducedMotion);

  useEffect(() => {
    if (typeof window === 'undefined') return;

    const mediaQuery = window.matchMedia('(prefers-reduced-motion: reduce)');
    
    const handleChange = (e: MediaQueryListEvent) => {
      setReducedMotion(e.matches);
    };

    mediaQuery.addEventListener('change', handleChange);
    return () => mediaQuery.removeEventListener('change', handleChange);
  }, []);

  return reducedMotion;
}

/** Input accessibility props hook */
export function useInputA11y(options: {
  id: string;
  label?: string;
  description?: string;
  required?: boolean;
  hasError?: boolean;
  errorMessage?: string;
}) {
  const props = getInputAriaProps(options);
  
  return {
    inputProps: props,
    labelId: `${options.id}-label`,
    descriptionId: options.description ? `${options.id}-description` : undefined,
    errorId: options.hasError ? `${options.id}-error` : undefined,
  };
}

/** Button accessibility props hook */
export function useButtonA11y(options: {
  label?: string;
  isPressed?: boolean;
  isExpanded?: boolean;
  controls?: string;
  isDisabled?: boolean;
}) {
  return getButtonAriaProps(options);
}

/** Focus management hook */
export function useFocusOnMount(shouldFocus: boolean = true) {
  const elementRef = useRef<HTMLElement | null>(null);

  useEffect(() => {
    if (shouldFocus && elementRef.current) {
      elementRef.current.focus();
    }
  }, [shouldFocus]);

  return elementRef;
}

/** Return focus on unmount hook */
export function useReturnFocus() {
  const previousActiveElement = useRef<Element | null>(null);

  useEffect(() => {
    previousActiveElement.current = document.activeElement;

    return () => {
      if (previousActiveElement.current instanceof HTMLElement) {
        previousActiveElement.current.focus();
      }
    };
  }, []);
}

export default {
  useFocusTrap,
  useAnnounce,
  useAriaId,
  useKeyboardHandler,
  useReducedMotion,
  useInputA11y,
  useButtonA11y,
  useFocusOnMount,
  useReturnFocus,
};
