// ============================================================================
// ACCESSIBILITY UTILITIES
// WCAG 2.1 AA compliant helpers for keyboard navigation, focus management,
// and ARIA attributes following Interface Segregation Principle
// ============================================================================

/** Focus trap for modals and dialogs */
export interface FocusTrapOptions {
  /** Container element to trap focus within */
  container: HTMLElement;
  /** Initial element to focus */
  initialFocus?: HTMLElement | null;
  /** Callback when escape is pressed */
  onEscape?: () => void;
}

/** Get all focusable elements within a container */
export function getFocusableElements(container: HTMLElement): HTMLElement[] {
  const focusableSelectors = [
    'button:not([disabled])',
    'a[href]',
    'input:not([disabled])',
    'select:not([disabled])',
    'textarea:not([disabled])',
    '[tabindex]:not([tabindex="-1"])',
    '[contenteditable="true"]',
  ].join(', ');

  return Array.from(container.querySelectorAll<HTMLElement>(focusableSelectors));
}

/** Create a focus trap for modal dialogs */
export function createFocusTrap({ container, initialFocus, onEscape }: FocusTrapOptions) {
  const focusableElements = getFocusableElements(container);
  const firstElement = focusableElements[0];
  const lastElement = focusableElements[focusableElements.length - 1];

  // Focus initial element or first focusable
  const elementToFocus = initialFocus || firstElement;
  elementToFocus?.focus();

  const handleKeyDown = (e: KeyboardEvent) => {
    // Handle Escape
    if (e.key === 'Escape' && onEscape) {
      e.preventDefault();
      onEscape();
      return;
    }

    // Handle Tab
    if (e.key === 'Tab') {
      if (focusableElements.length === 0) {
        e.preventDefault();
        return;
      }

      if (e.shiftKey) {
        // Shift + Tab: go backwards
        if (document.activeElement === firstElement) {
          e.preventDefault();
          lastElement?.focus();
        }
      } else {
        // Tab: go forwards
        if (document.activeElement === lastElement) {
          e.preventDefault();
          firstElement?.focus();
        }
      }
    }
  };

  container.addEventListener('keydown', handleKeyDown);

  // Return cleanup function
  return () => {
    container.removeEventListener('keydown', handleKeyDown);
  };
}

/** Announce message to screen readers via live region */
export function announceToScreenReader(
  message: string,
  priority: 'polite' | 'assertive' = 'polite'
): void {
  const liveRegion = document.getElementById('sr-live-region') || createLiveRegion();
  liveRegion.setAttribute('aria-live', priority);
  
  // Clear and set message (forces announcement)
  liveRegion.textContent = '';
  setTimeout(() => {
    liveRegion.textContent = message;
  }, 50);
}

/** Create screen reader live region if it doesn't exist */
function createLiveRegion(): HTMLElement {
  const region = document.createElement('div');
  region.id = 'sr-live-region';
  region.setAttribute('aria-live', 'polite');
  region.setAttribute('aria-atomic', 'true');
  region.style.cssText = `
    position: absolute;
    width: 1px;
    height: 1px;
    padding: 0;
    margin: -1px;
    overflow: hidden;
    clip: rect(0, 0, 0, 0);
    white-space: nowrap;
    border: 0;
  `;
  document.body.appendChild(region);
  return region;
}

/** Skip link helper - creates skip to main content link */
export function createSkipLink(targetId: string = 'main-content'): HTMLElement {
  const skipLink = document.createElement('a');
  skipLink.href = `#${targetId}`;
  skipLink.className = 'skip-link';
  skipLink.textContent = 'Skip to main content';
  skipLink.style.cssText = `
    position: absolute;
    top: -40px;
    left: 0;
    background: #D4A84B;
    color: #1A0F1E;
    padding: 8px 16px;
    z-index: 10000;
    text-decoration: none;
    font-weight: bold;
    border-radius: 0 0 4px 0;
    transition: top 0.2s;
  `;
  
  skipLink.addEventListener('focus', () => {
    skipLink.style.top = '0';
  });
  
  skipLink.addEventListener('blur', () => {
    skipLink.style.top = '-40px';
  });

  return skipLink;
}

/** Generate unique ID for ARIA relationships */
let idCounter = 0;
export function generateAriaId(prefix: string = 'aria'): string {
  return `${prefix}-${++idCounter}-${Date.now()}`;
}

/** Keyboard navigation helpers */
export const KeyboardKeys = {
  ENTER: 'Enter',
  SPACE: ' ',
  ESCAPE: 'Escape',
  TAB: 'Tab',
  ARROW_UP: 'ArrowUp',
  ARROW_DOWN: 'ArrowDown',
  ARROW_LEFT: 'ArrowLeft',
  ARROW_RIGHT: 'ArrowRight',
  HOME: 'Home',
  END: 'End',
} as const;

/** Check if keyboard event is activation key (Enter or Space) */
export function isActivationKey(event: KeyboardEvent): boolean {
  return event.key === KeyboardKeys.ENTER || event.key === KeyboardKeys.SPACE;
}

/** Handle keyboard activation for custom interactive elements */
export function handleKeyboardActivation(
  event: KeyboardEvent,
  callback: () => void
): void {
  if (isActivationKey(event)) {
    event.preventDefault();
    callback();
  }
}

/** ARIA props helper for buttons */
export interface AriaButtonProps {
  'aria-pressed'?: boolean;
  'aria-expanded'?: boolean;
  'aria-controls'?: string;
  'aria-describedby'?: string;
  'aria-label'?: string;
}

/** ARIA props helper for inputs */
export interface AriaInputProps {
  'aria-invalid'?: boolean;
  'aria-describedby'?: string;
  'aria-errormessage'?: string;
  'aria-required'?: boolean;
  'aria-label'?: string;
}

/** Create ARIA props for form field with error */
export function createAriaInputProps(
  fieldId: string,
  options: {
    hasError?: boolean;
    errorMessage?: string;
    isRequired?: boolean;
    description?: string;
  }
): AriaInputProps {
  const { hasError, isRequired, description } = options;
  const describedBy: string[] = [];
  
  if (description) {
    describedBy.push(`${fieldId}-description`);
  }
  if (hasError) {
    describedBy.push(`${fieldId}-error`);
  }

  return {
    'aria-invalid': hasError || undefined,
    'aria-required': isRequired || undefined,
    'aria-describedby': describedBy.length > 0 ? describedBy.join(' ') : undefined,
    'aria-errormessage': hasError ? `${fieldId}-error` : undefined,
  };
}

/** Reduced motion preference check */
export function prefersReducedMotion(): boolean {
  if (typeof window === 'undefined') return false;
  return window.matchMedia('(prefers-reduced-motion: reduce)').matches;
}

/** High contrast mode check */
export function prefersHighContrast(): boolean {
  if (typeof window === 'undefined') return false;
  return window.matchMedia('(prefers-contrast: more)').matches;
}

/** Color contrast ratio calculator (WCAG 2.1) */
export function getContrastRatio(color1: string, color2: string): number {
  const lum1 = getRelativeLuminance(color1);
  const lum2 = getRelativeLuminance(color2);
  const lighter = Math.max(lum1, lum2);
  const darker = Math.min(lum1, lum2);
  return (lighter + 0.05) / (darker + 0.05);
}

/** Calculate relative luminance of a color */
function getRelativeLuminance(hex: string): number {
  const rgb = hexToRgb(hex);
  if (!rgb) return 0;
  
  const [r, g, b] = [rgb.r, rgb.g, rgb.b].map((c) => {
    c = c / 255;
    return c <= 0.03928 ? c / 12.92 : Math.pow((c + 0.055) / 1.055, 2.4);
  });
  
  return 0.2126 * r + 0.7152 * g + 0.0722 * b;
}

/** Convert hex color to RGB */
function hexToRgb(hex: string): { r: number; g: number; b: number } | null {
  const result = /^#?([a-f\d]{2})([a-f\d]{2})([a-f\d]{2})$/i.exec(hex);
  return result
    ? {
        r: parseInt(result[1], 16),
        g: parseInt(result[2], 16),
        b: parseInt(result[3], 16),
      }
    : null;
}

/** Check if contrast ratio meets WCAG AA (4.5:1 for normal text) */
export function meetsContrastAA(color1: string, color2: string): boolean {
  return getContrastRatio(color1, color2) >= 4.5;
}

/** Check if contrast ratio meets WCAG AAA (7:1 for normal text) */
export function meetsContrastAAA(color1: string, color2: string): boolean {
  return getContrastRatio(color1, color2) >= 7;
}

export default {
  getFocusableElements,
  createFocusTrap,
  announceToScreenReader,
  createSkipLink,
  generateAriaId,
  KeyboardKeys,
  isActivationKey,
  handleKeyboardActivation,
  createAriaInputProps,
  prefersReducedMotion,
  prefersHighContrast,
  getContrastRatio,
  meetsContrastAA,
  meetsContrastAAA,
};
