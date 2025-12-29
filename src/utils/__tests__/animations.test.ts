import {
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
} from '../animations';

// Mock prefersReducedMotion
jest.mock('../accessibility', () => ({
  prefersReducedMotion: jest.fn(() => false),
}));

import { prefersReducedMotion } from '../accessibility';

const mockPrefersReducedMotion = prefersReducedMotion as jest.Mock;

describe('Animation Utilities', () => {
  beforeEach(() => {
    mockPrefersReducedMotion.mockReturnValue(false);
  });

  describe('timings', () => {
    it('should have correct timing values', () => {
      expect(timings.instant).toBe('0ms');
      expect(timings.fast).toBe('150ms');
      expect(timings.normal).toBe('250ms');
      expect(timings.slow).toBe('400ms');
    });
  });

  describe('easings', () => {
    it('should have correct easing values', () => {
      expect(easings.linear).toBe('linear');
      expect(easings.ease).toBe('ease');
      expect(easings.spring).toContain('cubic-bezier');
    });
  });

  describe('transition', () => {
    it('should create transition string for single property', () => {
      const result = transition('opacity', 'normal', 'smooth');
      expect(result).toContain('opacity');
      expect(result).toContain('250ms');
    });

    it('should create transition string for multiple properties', () => {
      const result = transition(['opacity', 'transform'], 'fast');
      expect(result).toContain('opacity');
      expect(result).toContain('transform');
    });

    it('should return none when reduced motion is preferred', () => {
      mockPrefersReducedMotion.mockReturnValue(true);
      const result = transition('opacity');
      expect(result).toBe('none');
    });

    it('should accept custom duration string', () => {
      const result = transition('opacity', '500ms');
      expect(result).toContain('500ms');
    });

    it('should accept custom easing string', () => {
      const result = transition('opacity', 'normal', 'cubic-bezier(0.5, 0, 0.5, 1)');
      expect(result).toContain('cubic-bezier');
    });
  });

  describe('transitions presets', () => {
    it('should have fade transition', () => {
      expect(transitions.fade).toContain('opacity');
    });

    it('should have slide transition', () => {
      expect(transitions.slide).toContain('transform');
    });

    it('should have interactive transition', () => {
      expect(transitions.interactive).toContain('background-color');
      expect(transitions.interactive).toContain('color');
    });
  });

  describe('keyframes', () => {
    it('should have fadeIn keyframe', () => {
      expect(keyframes.fadeIn).toContain('@keyframes fadeIn');
      expect(keyframes.fadeIn).toContain('opacity');
    });

    it('should have slideInUp keyframe', () => {
      expect(keyframes.slideInUp).toContain('@keyframes slideInUp');
      expect(keyframes.slideInUp).toContain('translateY');
    });

    it('should have pulse keyframe', () => {
      expect(keyframes.pulse).toContain('@keyframes pulse');
    });

    it('should have spin keyframe', () => {
      expect(keyframes.spin).toContain('@keyframes spin');
      expect(keyframes.spin).toContain('rotate');
    });
  });

  describe('animation', () => {
    it('should create animation string', () => {
      const result = animation('fadeIn', 'normal', 'smooth');
      expect(result).toContain('fadeIn');
      expect(result).toContain('250ms');
    });

    it('should return none when reduced motion is preferred', () => {
      mockPrefersReducedMotion.mockReturnValue(true);
      const result = animation('fadeIn');
      expect(result).toBe('none');
    });

    it('should accept options', () => {
      const result = animation('fadeIn', 'normal', 'smooth', {
        delay: '100ms',
        iterations: 'infinite',
        direction: 'alternate',
      });
      expect(result).toContain('100ms');
      expect(result).toContain('infinite');
      expect(result).toContain('alternate');
    });
  });

  describe('injectKeyframes', () => {
    it('should inject keyframes into document', () => {
      // Clean up any existing style
      const existing = document.getElementById('chimera-animations');
      if (existing) existing.remove();

      injectKeyframes();

      const style = document.getElementById('chimera-animations');
      expect(style).toBeTruthy();
      expect(style?.textContent).toContain('@keyframes');
    });

    it('should not inject twice', () => {
      injectKeyframes();
      injectKeyframes();

      const styles = document.querySelectorAll('#chimera-animations');
      expect(styles.length).toBe(1);
    });
  });

  describe('staggerDelay', () => {
    it('should calculate stagger delay', () => {
      expect(staggerDelay(0)).toBe('0ms');
      expect(staggerDelay(1)).toBe('50ms');
      expect(staggerDelay(3)).toBe('150ms');
    });

    it('should use custom base delay', () => {
      expect(staggerDelay(2, 100)).toBe('200ms');
    });

    it('should return 0ms when reduced motion is preferred', () => {
      mockPrefersReducedMotion.mockReturnValue(true);
      expect(staggerDelay(5)).toBe('0ms');
    });
  });

  describe('animationStyles', () => {
    it('should return fadeIn style', () => {
      const style = animationStyles.fadeIn();
      expect(style.animation).toContain('fadeIn');
    });

    it('should return slideUp style with delay', () => {
      const style = animationStyles.slideUp(100);
      expect(style.animation).toContain('slideInUp');
      expect(style.animation).toContain('100ms');
    });

    it('should return none when reduced motion is preferred', () => {
      mockPrefersReducedMotion.mockReturnValue(true);
      const style = animationStyles.fadeIn();
      expect(style.animation).toBe('none');
    });

    it('should return pulse style', () => {
      const style = animationStyles.pulse();
      expect(style.animation).toContain('pulse');
      expect(style.animation).toContain('infinite');
    });

    it('should return spin style', () => {
      const style = animationStyles.spin();
      expect(style.animation).toContain('spin');
    });
  });

  describe('getAnimationStyles', () => {
    it('should return idle styles', () => {
      const styles = getAnimationStyles('idle');
      expect(styles.opacity).toBe(0);
    });

    it('should return entering styles', () => {
      const styles = getAnimationStyles('entering');
      expect(styles.opacity).toBe(1);
      expect(styles.transition).toBeDefined();
    });

    it('should return entered styles', () => {
      const styles = getAnimationStyles('entered');
      expect(styles.opacity).toBe(1);
    });

    it('should return exiting styles', () => {
      const styles = getAnimationStyles('exiting');
      expect(styles.opacity).toBe(0);
    });

    it('should handle slide type', () => {
      const idle = getAnimationStyles('idle', 'slide');
      expect(idle.transform).toContain('translateY');
    });

    it('should handle scale type', () => {
      const idle = getAnimationStyles('idle', 'scale');
      expect(idle.transform).toContain('scale');
    });

    it('should respect reduced motion preference', () => {
      mockPrefersReducedMotion.mockReturnValue(true);
      const entering = getAnimationStyles('entering', 'slide');
      expect(entering.transform).toBeUndefined();
    });
  });
});
