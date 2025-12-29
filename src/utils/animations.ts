import { prefersReducedMotion } from './accessibility';

// ============================================================================
// ANIMATION UTILITIES
// Smooth, performant animations that respect user preferences
// Following Interface Segregation Principle with focused, composable utilities
// ============================================================================

/** Animation timing presets */
export const timings = {
  instant: '0ms',
  fast: '150ms',
  normal: '250ms',
  slow: '400ms',
  slower: '600ms',
} as const;

/** Easing function presets */
export const easings = {
  linear: 'linear',
  ease: 'ease',
  easeIn: 'ease-in',
  easeOut: 'ease-out',
  easeInOut: 'ease-in-out',
  // Custom curves for more natural motion
  spring: 'cubic-bezier(0.34, 1.56, 0.64, 1)',
  bounce: 'cubic-bezier(0.68, -0.55, 0.265, 1.55)',
  smooth: 'cubic-bezier(0.4, 0, 0.2, 1)',
  snappy: 'cubic-bezier(0.2, 0, 0, 1)',
} as const;

/** CSS transition helper */
export function transition(
  properties: string | string[],
  duration: keyof typeof timings | string = 'normal',
  easing: keyof typeof easings | string = 'smooth'
): string {
  if (prefersReducedMotion()) {
    return 'none';
  }

  const props = Array.isArray(properties) ? properties : [properties];
  const time = timings[duration as keyof typeof timings] || duration;
  const ease = easings[easing as keyof typeof easings] || easing;

  return props.map(prop => `${prop} ${time} ${ease}`).join(', ');
}

/** Common transition presets */
export const transitions = {
  fade: transition('opacity', 'normal'),
  fadeQuick: transition('opacity', 'fast'),
  slide: transition('transform', 'normal'),
  slideSpring: transition('transform', 'normal', 'spring'),
  scale: transition('transform', 'fast', 'spring'),
  color: transition('color', 'fast'),
  background: transition('background-color', 'fast'),
  border: transition('border-color', 'fast'),
  shadow: transition('box-shadow', 'normal'),
  all: transition('all', 'normal'),
  allFast: transition('all', 'fast'),
  interactive: transition(['background-color', 'border-color', 'color', 'box-shadow'], 'fast'),
} as const;

/** Keyframe animation definitions */
export const keyframes = {
  fadeIn: `
    @keyframes fadeIn {
      from { opacity: 0; }
      to { opacity: 1; }
    }
  `,
  fadeOut: `
    @keyframes fadeOut {
      from { opacity: 1; }
      to { opacity: 0; }
    }
  `,
  slideInUp: `
    @keyframes slideInUp {
      from { transform: translateY(20px); opacity: 0; }
      to { transform: translateY(0); opacity: 1; }
    }
  `,
  slideInDown: `
    @keyframes slideInDown {
      from { transform: translateY(-20px); opacity: 0; }
      to { transform: translateY(0); opacity: 1; }
    }
  `,
  slideInLeft: `
    @keyframes slideInLeft {
      from { transform: translateX(-20px); opacity: 0; }
      to { transform: translateX(0); opacity: 1; }
    }
  `,
  slideInRight: `
    @keyframes slideInRight {
      from { transform: translateX(20px); opacity: 0; }
      to { transform: translateX(0); opacity: 1; }
    }
  `,
  scaleIn: `
    @keyframes scaleIn {
      from { transform: scale(0.95); opacity: 0; }
      to { transform: scale(1); opacity: 1; }
    }
  `,
  scaleOut: `
    @keyframes scaleOut {
      from { transform: scale(1); opacity: 1; }
      to { transform: scale(0.95); opacity: 0; }
    }
  `,
  pulse: `
    @keyframes pulse {
      0%, 100% { opacity: 1; }
      50% { opacity: 0.5; }
    }
  `,
  spin: `
    @keyframes spin {
      from { transform: rotate(0deg); }
      to { transform: rotate(360deg); }
    }
  `,
  bounce: `
    @keyframes bounce {
      0%, 100% { transform: translateY(0); }
      50% { transform: translateY(-10px); }
    }
  `,
  shake: `
    @keyframes shake {
      0%, 100% { transform: translateX(0); }
      25% { transform: translateX(-5px); }
      75% { transform: translateX(5px); }
    }
  `,
  glow: `
    @keyframes glow {
      0%, 100% { box-shadow: 0 0 5px rgba(212, 168, 75, 0.3); }
      50% { box-shadow: 0 0 20px rgba(212, 168, 75, 0.6); }
    }
  `,
} as const;

/** Get animation CSS with reduced motion support */
export function animation(
  name: keyof typeof keyframes,
  duration: keyof typeof timings | string = 'normal',
  easing: keyof typeof easings | string = 'smooth',
  options: {
    delay?: string;
    iterations?: number | 'infinite';
    direction?: 'normal' | 'reverse' | 'alternate' | 'alternate-reverse';
    fillMode?: 'none' | 'forwards' | 'backwards' | 'both';
  } = {}
): string {
  if (prefersReducedMotion()) {
    return 'none';
  }

  const time = timings[duration as keyof typeof timings] || duration;
  const ease = easings[easing as keyof typeof easings] || easing;
  const { delay = '0ms', iterations = 1, direction = 'normal', fillMode = 'both' } = options;

  return `${name} ${time} ${ease} ${delay} ${iterations} ${direction} ${fillMode}`;
}

/** Inject keyframes into document (call once on app init) */
export function injectKeyframes(): void {
  if (typeof document === 'undefined') return;
  
  const styleId = 'chimera-animations';
  if (document.getElementById(styleId)) return;

  const style = document.createElement('style');
  style.id = styleId;
  style.textContent = Object.values(keyframes).join('\n');
  document.head.appendChild(style);
}

/** Staggered animation delay calculator */
export function staggerDelay(index: number, baseDelay: number = 50): string {
  if (prefersReducedMotion()) return '0ms';
  return `${index * baseDelay}ms`;
}

/** React style object for common animations */
export const animationStyles = {
  fadeIn: (delay = 0): React.CSSProperties => ({
    animation: prefersReducedMotion() ? 'none' : `fadeIn 250ms ease-out ${delay}ms both`,
  }),
  slideUp: (delay = 0): React.CSSProperties => ({
    animation: prefersReducedMotion() ? 'none' : `slideInUp 300ms cubic-bezier(0.34, 1.56, 0.64, 1) ${delay}ms both`,
  }),
  scaleIn: (delay = 0): React.CSSProperties => ({
    animation: prefersReducedMotion() ? 'none' : `scaleIn 200ms cubic-bezier(0.34, 1.56, 0.64, 1) ${delay}ms both`,
  }),
  pulse: (): React.CSSProperties => ({
    animation: prefersReducedMotion() ? 'none' : 'pulse 2s ease-in-out infinite',
  }),
  spin: (): React.CSSProperties => ({
    animation: prefersReducedMotion() ? 'none' : 'spin 1s linear infinite',
  }),
  glow: (): React.CSSProperties => ({
    animation: prefersReducedMotion() ? 'none' : 'glow 2s ease-in-out infinite',
  }),
};

/** Hook-friendly animation state types */
export type AnimationState = 'idle' | 'entering' | 'entered' | 'exiting' | 'exited';

/** Get styles for enter/exit animations */
export function getAnimationStyles(
  state: AnimationState,
  type: 'fade' | 'slide' | 'scale' = 'fade'
): React.CSSProperties {
  if (prefersReducedMotion()) {
    return { opacity: state === 'entered' || state === 'entering' ? 1 : 0 };
  }

  const baseStyles: Record<AnimationState, React.CSSProperties> = {
    idle: { opacity: 0 },
    entering: {
      opacity: 1,
      transform: type === 'slide' ? 'translateY(0)' : type === 'scale' ? 'scale(1)' : undefined,
      transition: transitions.all,
    },
    entered: { opacity: 1 },
    exiting: {
      opacity: 0,
      transform: type === 'slide' ? 'translateY(-10px)' : type === 'scale' ? 'scale(0.95)' : undefined,
      transition: transitions.all,
    },
    exited: { opacity: 0 },
  };

  const typeStyles: Record<typeof type, Partial<Record<AnimationState, React.CSSProperties>>> = {
    fade: {},
    slide: {
      idle: { opacity: 0, transform: 'translateY(10px)' },
      exited: { opacity: 0, transform: 'translateY(-10px)' },
    },
    scale: {
      idle: { opacity: 0, transform: 'scale(0.95)' },
      exited: { opacity: 0, transform: 'scale(0.95)' },
    },
  };

  return { ...baseStyles[state], ...typeStyles[type][state] };
}

export default {
  timings,
  easings,
  transition,
  transitions,
  keyframes,
  animation,
  injectKeyframes,
  staggerDelay,
  animationStyles,
  getAnimationStyles,
};
