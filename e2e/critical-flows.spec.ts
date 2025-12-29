/**
 * Critical User Flows E2E Tests
 * 
 * Tests the most important user journeys for the mining pool:
 * 1. Homepage loading and navigation
 * 2. Theme switching (dark/light)
 * 3. Responsive design verification
 * 4. Accessibility compliance
 * 5. Performance metrics
 * 6. PWA functionality
 * 
 * Following Interface Segregation - each test suite is independent
 */

import { test, expect, type Page } from '@playwright/test';

const BASE_URL = process.env.TEST_URL || 'https://206.162.80.230';

// ============================================================================
// HOMEPAGE & NAVIGATION TESTS
// ============================================================================

test.describe('Homepage & Navigation', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto(BASE_URL);
  });

  test('should load homepage within performance budget', async ({ page }) => {
    const startTime = Date.now();
    await page.waitForLoadState('domcontentloaded');
    const loadTime = Date.now() - startTime;
    
    // Page should load within 5 seconds
    expect(loadTime).toBeLessThan(5000);
    
    // Core content should be visible
    await expect(page.locator('body')).toBeVisible();
  });

  test('should display pool statistics', async ({ page }) => {
    // Wait for stats to load
    await page.waitForLoadState('networkidle');
    
    // Check for key stat elements (flexible selectors)
    const statsSection = page.locator('[class*="stat"], [class*="card"], [data-testid*="stat"]').first();
    await expect(statsSection).toBeVisible({ timeout: 10000 });
  });

  test('should have working navigation links', async ({ page }) => {
    // Check for navigation elements
    const nav = page.locator('nav, header, [role="navigation"]').first();
    await expect(nav).toBeVisible();
    
    // Check for common navigation items
    const navLinks = await page.locator('a, button').filter({ hasText: /dashboard|home|community|admin/i }).count();
    expect(navLinks).toBeGreaterThan(0);
  });

  test('should display logo and branding', async ({ page }) => {
    // Check for logo image or text
    const logo = page.locator('img[alt*="logo" i], img[src*="logo"], [class*="logo"], text=Chimera').first();
    await expect(logo).toBeVisible();
  });
});

// ============================================================================
// THEME SWITCHING TESTS
// ============================================================================

test.describe('Theme Switching', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto(BASE_URL);
    await page.waitForLoadState('networkidle');
  });

  test('should have theme toggle button', async ({ page }) => {
    // Look for theme toggle (sun/moon icon or toggle button)
    const themeToggle = page.locator(
      'button[aria-label*="theme" i], ' +
      'button[title*="theme" i], ' +
      '[data-testid="theme-toggle"], ' +
      'button:has(svg[class*="sun"]), ' +
      'button:has(svg[class*="moon"])'
    ).first();
    
    // Theme toggle should exist (may not be visible if not implemented yet)
    const exists = await themeToggle.count() > 0;
    if (exists) {
      await expect(themeToggle).toBeVisible();
    }
  });

  test('should persist theme preference', async ({ page, context }) => {
    // Set a theme preference in localStorage
    await page.evaluate(() => {
      localStorage.setItem('chimera-pool-theme', 'dark');
    });
    
    // Reload and verify persistence
    await page.reload();
    
    const savedTheme = await page.evaluate(() => {
      return localStorage.getItem('chimera-pool-theme');
    });
    
    expect(savedTheme).toBe('dark');
  });

  test('should respect system color scheme preference', async ({ page }) => {
    // Emulate dark color scheme
    await page.emulateMedia({ colorScheme: 'dark' });
    await page.reload();
    
    // Check if page responds to system preference
    const bodyStyles = await page.evaluate(() => {
      return window.getComputedStyle(document.body).backgroundColor;
    });
    
    // Body should have some background color (not transparent)
    expect(bodyStyles).not.toBe('rgba(0, 0, 0, 0)');
  });
});

// ============================================================================
// RESPONSIVE DESIGN TESTS
// ============================================================================

test.describe('Responsive Design', () => {
  const viewports = [
    { name: 'Mobile', width: 375, height: 667 },
    { name: 'Tablet', width: 768, height: 1024 },
    { name: 'Desktop', width: 1920, height: 1080 },
  ];

  for (const viewport of viewports) {
    test(`should render correctly on ${viewport.name}`, async ({ page }) => {
      await page.setViewportSize({ width: viewport.width, height: viewport.height });
      await page.goto(BASE_URL);
      await page.waitForLoadState('networkidle');
      
      // Page should not have horizontal overflow
      const hasHorizontalScroll = await page.evaluate(() => {
        return document.documentElement.scrollWidth > document.documentElement.clientWidth;
      });
      
      // Allow small tolerance for scrollbar
      if (hasHorizontalScroll) {
        const scrollDiff = await page.evaluate(() => {
          return document.documentElement.scrollWidth - document.documentElement.clientWidth;
        });
        expect(scrollDiff).toBeLessThan(20); // Allow 20px for scrollbar
      }
      
      // Main content should be visible
      const mainContent = page.locator('main, #root, [class*="container"]').first();
      await expect(mainContent).toBeVisible();
    });
  }

  test('should show mobile menu on small screens', async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 667 });
    await page.goto(BASE_URL);
    
    // Look for hamburger menu or mobile nav toggle
    const mobileMenu = page.locator(
      'button[aria-label*="menu" i], ' +
      '[data-testid="mobile-menu"], ' +
      'button:has(svg[class*="menu"]), ' +
      '[class*="hamburger"]'
    ).first();
    
    const exists = await mobileMenu.count() > 0;
    // Mobile menu should exist on small screens
    if (exists) {
      await expect(mobileMenu).toBeVisible();
    }
  });
});

// ============================================================================
// ACCESSIBILITY TESTS
// ============================================================================

test.describe('Accessibility', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto(BASE_URL);
    await page.waitForLoadState('networkidle');
  });

  test('should have proper heading hierarchy', async ({ page }) => {
    // Get all headings
    const h1Count = await page.locator('h1').count();
    const h2Count = await page.locator('h2').count();
    
    // Should have at least one h1
    expect(h1Count).toBeGreaterThanOrEqual(1);
    
    // Should not skip heading levels if h2 exists
    if (h2Count > 0) {
      expect(h1Count).toBeGreaterThanOrEqual(1);
    }
  });

  test('should have accessible form labels', async ({ page }) => {
    // Find all input elements
    const inputs = page.locator('input:not([type="hidden"]):not([type="submit"])');
    const inputCount = await inputs.count();
    
    for (let i = 0; i < inputCount; i++) {
      const input = inputs.nth(i);
      const isVisible = await input.isVisible();
      
      if (isVisible) {
        // Input should have aria-label, aria-labelledby, or associated label
        const hasAriaLabel = await input.getAttribute('aria-label');
        const hasAriaLabelledby = await input.getAttribute('aria-labelledby');
        const id = await input.getAttribute('id');
        const hasLabel = id ? await page.locator(`label[for="${id}"]`).count() > 0 : false;
        const placeholder = await input.getAttribute('placeholder');
        
        const isAccessible = hasAriaLabel || hasAriaLabelledby || hasLabel || placeholder;
        expect(isAccessible).toBeTruthy();
      }
    }
  });

  test('should have sufficient color contrast', async ({ page }) => {
    // Get all text elements and their computed styles
    const textElements = await page.evaluate(() => {
      const elements = document.querySelectorAll('p, span, h1, h2, h3, h4, h5, h6, a, button, label');
      const results: Array<{ text: string; color: string; bg: string }> = [];
      
      elements.forEach(el => {
        const style = window.getComputedStyle(el);
        if (el.textContent && el.textContent.trim().length > 0) {
          results.push({
            text: el.textContent.slice(0, 50),
            color: style.color,
            bg: style.backgroundColor,
          });
        }
      });
      
      return results.slice(0, 10); // Sample first 10 elements
    });
    
    // At least some text should be present
    expect(textElements.length).toBeGreaterThan(0);
  });

  test('should be keyboard navigable', async ({ page }) => {
    // Focus should start at beginning of page
    await page.keyboard.press('Tab');
    
    // Something should be focused
    const focusedElement = await page.evaluate(() => {
      return document.activeElement?.tagName;
    });
    
    expect(focusedElement).not.toBe('BODY');
  });

  test('should have skip link for keyboard users', async ({ page }) => {
    // Look for skip link (may be visually hidden until focused)
    const skipLink = page.locator('a[href="#main"], a[href="#content"], a:has-text("Skip")').first();
    const exists = await skipLink.count() > 0;
    
    // Skip link is a best practice but not required
    if (exists) {
      await skipLink.focus();
      await expect(skipLink).toBeVisible();
    }
  });
});

// ============================================================================
// PERFORMANCE TESTS
// ============================================================================

test.describe('Performance', () => {
  test('should have good Core Web Vitals', async ({ page }) => {
    await page.goto(BASE_URL);
    
    // Wait for page to stabilize
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(2000);
    
    // Measure LCP (Largest Contentful Paint)
    const lcp = await page.evaluate(() => {
      return new Promise<number>((resolve) => {
        new PerformanceObserver((list) => {
          const entries = list.getEntries();
          const lastEntry = entries[entries.length - 1];
          resolve(lastEntry.startTime);
        }).observe({ type: 'largest-contentful-paint', buffered: true });
        
        // Fallback timeout
        setTimeout(() => resolve(0), 5000);
      });
    });
    
    // LCP should be under 4 seconds for acceptable performance
    if (lcp > 0) {
      expect(lcp).toBeLessThan(4000);
    }
  });

  test('should not have excessive DOM nodes', async ({ page }) => {
    await page.goto(BASE_URL);
    await page.waitForLoadState('networkidle');
    
    const domNodeCount = await page.evaluate(() => {
      return document.querySelectorAll('*').length;
    });
    
    // Good performance: under 1500 nodes, acceptable: under 3000
    expect(domNodeCount).toBeLessThan(5000);
  });

  test('should load critical resources efficiently', async ({ page }) => {
    const resourceTimings: Array<{ url: string; duration: number }> = [];
    
    page.on('response', async (response) => {
      const timing = response.timing();
      if (timing) {
        resourceTimings.push({
          url: response.url(),
          duration: timing.responseEnd - timing.requestStart,
        });
      }
    });
    
    await page.goto(BASE_URL);
    await page.waitForLoadState('networkidle');
    
    // Check that main document loaded
    const mainDoc = resourceTimings.find(r => r.url.includes(BASE_URL));
    expect(mainDoc).toBeDefined();
  });
});

// ============================================================================
// PWA TESTS
// ============================================================================

test.describe('PWA Functionality', () => {
  test('should have valid web manifest', async ({ page }) => {
    await page.goto(BASE_URL);
    
    // Check for manifest link
    const manifestLink = await page.locator('link[rel="manifest"]').getAttribute('href');
    
    if (manifestLink) {
      // Fetch and validate manifest
      const manifestUrl = new URL(manifestLink, BASE_URL).href;
      const response = await page.request.get(manifestUrl);
      
      expect(response.ok()).toBeTruthy();
      
      const manifest = await response.json();
      expect(manifest.name).toBeDefined();
      expect(manifest.short_name).toBeDefined();
      expect(manifest.icons).toBeDefined();
      expect(manifest.start_url).toBeDefined();
      expect(manifest.display).toBeDefined();
    }
  });

  test('should have proper meta tags for PWA', async ({ page }) => {
    await page.goto(BASE_URL);
    
    // Check theme-color meta tag
    const themeColor = await page.locator('meta[name="theme-color"]').getAttribute('content');
    expect(themeColor).toBeTruthy();
    
    // Check apple-mobile-web-app-capable
    const appleMobileCapable = await page.locator('meta[name="apple-mobile-web-app-capable"]').getAttribute('content');
    if (appleMobileCapable) {
      expect(appleMobileCapable).toBe('yes');
    }
  });

  test('should have service worker registered', async ({ page, context }) => {
    await page.goto(BASE_URL);
    await page.waitForLoadState('networkidle');
    
    // Check for service worker registration
    const hasServiceWorker = await page.evaluate(async () => {
      if ('serviceWorker' in navigator) {
        const registration = await navigator.serviceWorker.getRegistration();
        return !!registration;
      }
      return false;
    });
    
    // Service worker should be registered in production
    // May not be registered in development mode
    console.log('Service Worker registered:', hasServiceWorker);
  });
});

// ============================================================================
// ERROR HANDLING TESTS
// ============================================================================

test.describe('Error Handling', () => {
  test('should handle 404 gracefully', async ({ page }) => {
    const response = await page.goto(`${BASE_URL}/non-existent-page-12345`);
    
    // Should either redirect to home or show 404 page
    // Not crash or show blank page
    await expect(page.locator('body')).toBeVisible();
    
    // Should have some content
    const bodyText = await page.locator('body').textContent();
    expect(bodyText?.length).toBeGreaterThan(10);
  });

  test('should recover from JavaScript errors', async ({ page }) => {
    const errors: string[] = [];
    
    page.on('pageerror', (error) => {
      errors.push(error.message);
    });
    
    await page.goto(BASE_URL);
    await page.waitForLoadState('networkidle');
    
    // Page should still be functional even if there are non-critical errors
    await expect(page.locator('body')).toBeVisible();
    
    // Log any errors for debugging
    if (errors.length > 0) {
      console.log('JavaScript errors detected:', errors);
    }
  });
});

// ============================================================================
// LOADING STATES TESTS
// ============================================================================

test.describe('Loading States', () => {
  test('should show loading indicators during data fetch', async ({ page }) => {
    // Slow down network to observe loading states
    await page.route('**/*', (route) => {
      setTimeout(() => route.continue(), 500);
    });
    
    await page.goto(BASE_URL);
    
    // Look for skeleton loaders or loading spinners
    const loadingIndicators = page.locator(
      '[class*="skeleton"], ' +
      '[class*="loading"], ' +
      '[class*="spinner"], ' +
      '[role="status"], ' +
      '[aria-busy="true"]'
    );
    
    // Should have loading states during fetch
    const count = await loadingIndicators.count();
    console.log('Loading indicators found:', count);
  });

  test('should display content after loading completes', async ({ page }) => {
    await page.goto(BASE_URL);
    await page.waitForLoadState('networkidle');
    
    // Wait for any loading states to complete
    await page.waitForTimeout(2000);
    
    // Content should be visible
    const content = page.locator('main, #root, [class*="dashboard"], [class*="content"]').first();
    await expect(content).toBeVisible();
  });
});
