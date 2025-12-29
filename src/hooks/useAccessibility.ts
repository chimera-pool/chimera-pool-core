import { useEffect, useRef, useCallback, useState } from 'react';
import {
  createFocusTrap,
  announceToScreenReader,
  generateAriaId,
  handleKeyboardActivation,
  createAriaInputProps,
  prefersReducedMotion,
  type FocusTrapOptions,
  type AriaInputProps,
  type AriaButtonProps,
} from '../utils/accessibility';

// ============================================================================
// ACCESSIBILITY HOOKS
// React hooks for accessibility features
// Following Interface Segregation Principle
// ============================================================================

/** Focus trap hook for modals and dialogs */
export function useFocusTrap(isActive: boolean = true, onEscape?: () => void) {
  const containerRef = useRef<HTMLElement | null>(null);
  const cleanupRef = useRef<(() => void) | null>(null);

  useEffect(() => {
    if (!isActive || !containerRef.current) return;

    cleanupRef.current = createFocusTrap({
      container: containerRef.current,
      onEscape,
    });

    return () => {
      cleanupRef.current?.();
    };
  }, [isActive, onEscape]);

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
export function useKeyboardHandler(onActivate?: () => void) {
  const handleKeyDown = useCallback(
    (event: React.KeyboardEvent) => {
      if (onActivate) {
        handleKeyboardActivation(event.nativeEvent, onActivate);
      }
    },
    [onActivate]
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
  const props = createAriaInputProps(options.id, {
    hasError: options.hasError,
    errorMessage: options.errorMessage,
    isRequired: options.required,
    description: options.description,
  });
  
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
}): AriaButtonProps {
  return {
    'aria-label': options.label,
    'aria-pressed': options.isPressed,
    'aria-expanded': options.isExpanded,
    'aria-controls': options.controls,
  };
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
