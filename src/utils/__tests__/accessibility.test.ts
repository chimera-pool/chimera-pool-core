import {
  getFocusableElements,
  createFocusTrap,
  announceToScreenReader,
  generateAriaId,
  KeyboardKeys,
  isActivationKey,
  handleKeyboardActivation,
  createAriaInputProps,
  prefersReducedMotion,
  getContrastRatio,
  meetsContrastAA,
  meetsContrastAAA,
} from '../accessibility';

describe('Accessibility Utilities', () => {
  describe('getFocusableElements', () => {
    it('should return all focusable elements', () => {
      document.body.innerHTML = `
        <div id="container">
          <button>Button 1</button>
          <a href="#">Link</a>
          <input type="text" />
          <button disabled>Disabled</button>
          <div tabindex="0">Focusable div</div>
          <div tabindex="-1">Not focusable</div>
        </div>
      `;
      
      const container = document.getElementById('container')!;
      const focusable = getFocusableElements(container);
      
      expect(focusable.length).toBe(4); // button, link, input, tabindex=0 div
    });

    it('should exclude disabled elements', () => {
      document.body.innerHTML = `
        <div id="container">
          <button>Enabled</button>
          <button disabled>Disabled</button>
          <input disabled />
        </div>
      `;
      
      const container = document.getElementById('container')!;
      const focusable = getFocusableElements(container);
      
      expect(focusable.length).toBe(1);
    });
  });

  describe('createFocusTrap', () => {
    it('should focus initial element', () => {
      document.body.innerHTML = `
        <div id="modal">
          <button id="first">First</button>
          <button id="second">Second</button>
          <button id="last">Last</button>
        </div>
      `;
      
      const container = document.getElementById('modal')!;
      const initialFocus = document.getElementById('second')!;
      
      const cleanup = createFocusTrap({ container, initialFocus });
      
      expect(document.activeElement).toBe(initialFocus);
      cleanup();
    });

    it('should call onEscape when Escape is pressed', () => {
      document.body.innerHTML = `
        <div id="modal">
          <button>Button</button>
        </div>
      `;
      
      const container = document.getElementById('modal')!;
      const onEscape = jest.fn();
      
      const cleanup = createFocusTrap({ container, onEscape });
      
      const event = new KeyboardEvent('keydown', { key: 'Escape' });
      container.dispatchEvent(event);
      
      expect(onEscape).toHaveBeenCalled();
      cleanup();
    });
  });

  describe('announceToScreenReader', () => {
    it('should create live region if not exists', () => {
      // Clear any existing live region
      const existing = document.getElementById('sr-live-region');
      if (existing) existing.remove();
      
      announceToScreenReader('Test message');
      
      const liveRegion = document.getElementById('sr-live-region');
      expect(liveRegion).toBeTruthy();
      expect(liveRegion?.getAttribute('aria-live')).toBe('polite');
    });

    it('should support assertive priority', () => {
      announceToScreenReader('Urgent message', 'assertive');
      
      const liveRegion = document.getElementById('sr-live-region');
      expect(liveRegion?.getAttribute('aria-live')).toBe('assertive');
    });
  });

  describe('generateAriaId', () => {
    it('should generate unique IDs', () => {
      const id1 = generateAriaId('test');
      const id2 = generateAriaId('test');
      
      expect(id1).not.toBe(id2);
      expect(id1).toMatch(/^test-\d+-\d+$/);
    });

    it('should use default prefix', () => {
      const id = generateAriaId();
      expect(id).toMatch(/^aria-\d+-\d+$/);
    });
  });

  describe('KeyboardKeys', () => {
    it('should have correct key values', () => {
      expect(KeyboardKeys.ENTER).toBe('Enter');
      expect(KeyboardKeys.SPACE).toBe(' ');
      expect(KeyboardKeys.ESCAPE).toBe('Escape');
      expect(KeyboardKeys.TAB).toBe('Tab');
    });
  });

  describe('isActivationKey', () => {
    it('should return true for Enter', () => {
      const event = new KeyboardEvent('keydown', { key: 'Enter' });
      expect(isActivationKey(event)).toBe(true);
    });

    it('should return true for Space', () => {
      const event = new KeyboardEvent('keydown', { key: ' ' });
      expect(isActivationKey(event)).toBe(true);
    });

    it('should return false for other keys', () => {
      const event = new KeyboardEvent('keydown', { key: 'a' });
      expect(isActivationKey(event)).toBe(false);
    });
  });

  describe('handleKeyboardActivation', () => {
    it('should call callback on Enter', () => {
      const callback = jest.fn();
      const event = new KeyboardEvent('keydown', { key: 'Enter' });
      
      handleKeyboardActivation(event, callback);
      
      expect(callback).toHaveBeenCalled();
    });

    it('should not call callback on other keys', () => {
      const callback = jest.fn();
      const event = new KeyboardEvent('keydown', { key: 'Tab' });
      
      handleKeyboardActivation(event, callback);
      
      expect(callback).not.toHaveBeenCalled();
    });
  });

  describe('createAriaInputProps', () => {
    it('should return empty props when no options', () => {
      const props = createAriaInputProps('field', {});
      
      expect(props['aria-invalid']).toBeUndefined();
      expect(props['aria-required']).toBeUndefined();
    });

    it('should set aria-invalid when hasError', () => {
      const props = createAriaInputProps('field', { hasError: true });
      
      expect(props['aria-invalid']).toBe(true);
      expect(props['aria-errormessage']).toBe('field-error');
    });

    it('should set aria-required when isRequired', () => {
      const props = createAriaInputProps('field', { isRequired: true });
      
      expect(props['aria-required']).toBe(true);
    });

    it('should set aria-describedby for description and error', () => {
      const props = createAriaInputProps('field', { 
        description: 'Help text', 
        hasError: true 
      });
      
      expect(props['aria-describedby']).toBe('field-description field-error');
    });
  });

  describe('prefersReducedMotion', () => {
    it('should return boolean', () => {
      // Mock matchMedia
      Object.defineProperty(window, 'matchMedia', {
        writable: true,
        value: jest.fn().mockImplementation(query => ({
          matches: false,
          media: query,
          onchange: null,
          addListener: jest.fn(),
          removeListener: jest.fn(),
          addEventListener: jest.fn(),
          removeEventListener: jest.fn(),
          dispatchEvent: jest.fn(),
        })),
      });
      
      const result = prefersReducedMotion();
      expect(typeof result).toBe('boolean');
    });
  });

  describe('Color Contrast', () => {
    describe('getContrastRatio', () => {
      it('should return 21:1 for black and white', () => {
        const ratio = getContrastRatio('#000000', '#FFFFFF');
        expect(ratio).toBeCloseTo(21, 0);
      });

      it('should return 1:1 for same colors', () => {
        const ratio = getContrastRatio('#FF0000', '#FF0000');
        expect(ratio).toBe(1);
      });
    });

    describe('meetsContrastAA', () => {
      it('should return true for black on white', () => {
        expect(meetsContrastAA('#000000', '#FFFFFF')).toBe(true);
      });

      it('should return false for low contrast colors', () => {
        expect(meetsContrastAA('#CCCCCC', '#FFFFFF')).toBe(false);
      });
    });

    describe('meetsContrastAAA', () => {
      it('should return true for black on white', () => {
        expect(meetsContrastAAA('#000000', '#FFFFFF')).toBe(true);
      });

      it('should return false for medium contrast', () => {
        // This gray on white has ~4.5:1 ratio (AA but not AAA)
        expect(meetsContrastAAA('#767676', '#FFFFFF')).toBe(false);
      });
    });
  });
});
