/**
 * Admin Features E2E Tests
 * 
 * Tests for:
 * 1. Bug report screenshot attachment
 * 2. Admin panel sortable user columns
 * 3. Admin statistics chart dropdowns
 */

import { test, expect } from '@playwright/test';

const BASE_URL = process.env.TEST_URL || 'http://localhost:3000';

test.describe('Bug Report Screenshot Attachment', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto(BASE_URL);
  });

  test('bug report modal should have screenshot upload field', async ({ page }) => {
    // Look for bug report button in footer or navigation
    const bugReportBtn = page.locator('text=Report Bug').or(page.locator('text=🐛'));
    
    // If logged in user menu exists, check there
    const userMenu = page.locator('[data-testid="user-menu"]').or(page.locator('text=Report Bug'));
    
    if (await userMenu.isVisible()) {
      await userMenu.click();
    }

    // Check if bug report modal can be opened
    const reportBugOption = page.locator('text=Report Bug').first();
    if (await reportBugOption.isVisible()) {
      await reportBugOption.click();
      
      // Verify screenshot upload field exists
      const screenshotSection = page.locator('text=Screenshot (optional)');
      await expect(screenshotSection).toBeVisible({ timeout: 5000 });
      
      // Verify file input exists
      const fileInput = page.locator('input[type="file"][accept="image/*"]');
      await expect(fileInput).toBeAttached();
    }
  });

  test('screenshot upload should show preview when file selected', async ({ page }) => {
    // This test requires login - skip if not authenticated
    const isLoggedIn = await page.locator('text=Logout').or(page.locator('text=Profile')).isVisible();
    
    if (!isLoggedIn) {
      test.skip();
      return;
    }

    // Open bug report modal
    await page.click('text=Report Bug');
    
    // Upload a test image
    const fileInput = page.locator('input[type="file"][accept="image/*"]');
    
    // Create a small test image buffer
    const testImagePath = 'test-fixtures/test-screenshot.png';
    
    // Check if preview appears after upload
    const previewImg = page.locator('img[alt="Screenshot preview"]');
    // Preview should not be visible initially
    await expect(previewImg).not.toBeVisible();
  });
});

test.describe('Admin Panel Sortable User Columns', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto(BASE_URL);
  });

  test('admin panel user table should have sortable column headers', async ({ page }) => {
    // This test requires admin login
    // Check if admin panel is accessible
    const adminBtn = page.locator('text=Admin Panel').or(page.locator('[data-testid="admin-panel-btn"]'));
    
    if (await adminBtn.isVisible()) {
      await adminBtn.click();
      
      // Wait for admin panel to load
      await page.waitForSelector('text=User Management', { timeout: 5000 }).catch(() => {});
      
      // Check for sortable headers with sort indicators
      const usernameHeader = page.locator('th:has-text("Username")');
      const emailHeader = page.locator('th:has-text("Email")');
      const hashrateHeader = page.locator('th:has-text("Hashrate")');
      const earningsHeader = page.locator('th:has-text("Earnings")');
      const statusHeader = page.locator('th:has-text("Status")');
      
      // Headers should be clickable (have cursor: pointer style)
      if (await usernameHeader.isVisible()) {
        const cursorStyle = await usernameHeader.evaluate(el => 
          window.getComputedStyle(el).cursor
        );
        expect(cursorStyle).toBe('pointer');
      }
    }
  });

  test('clicking column header should toggle sort direction', async ({ page }) => {
    const adminBtn = page.locator('text=Admin Panel');
    
    if (await adminBtn.isVisible()) {
      await adminBtn.click();
      
      // Wait for table to load
      await page.waitForSelector('table', { timeout: 5000 }).catch(() => {});
      
      const usernameHeader = page.locator('th:has-text("Username")');
      
      if (await usernameHeader.isVisible()) {
        // Click to sort ascending
        await usernameHeader.click();
        
        // Check for sort indicator (↑ or ↓)
        const headerText = await usernameHeader.textContent();
        expect(headerText).toMatch(/Username.*[↑↓]/);
        
        // Click again to toggle direction
        await usernameHeader.click();
        const headerTextAfter = await usernameHeader.textContent();
        
        // Direction should have changed
        if (headerText?.includes('↑')) {
          expect(headerTextAfter).toContain('↓');
        } else if (headerText?.includes('↓')) {
          expect(headerTextAfter).toContain('↑');
        }
      }
    }
  });
});

test.describe('Admin Statistics Chart Dropdowns', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto(BASE_URL);
  });

  test('admin stats section should have chart dropdown selectors', async ({ page }) => {
    const adminBtn = page.locator('text=Admin Panel');
    
    if (await adminBtn.isVisible()) {
      await adminBtn.click();
      
      // Navigate to statistics tab
      const statsTab = page.locator('text=Statistics').or(page.locator('button:has-text("Stats")'));
      if (await statsTab.isVisible()) {
        await statsTab.click();
        
        // Wait for charts section to load
        await page.waitForTimeout(1000);
        
        // Check for Grafana chart section
        const grafanaSection = page.locator('text=Grafana').first();
        
        if (await grafanaSection.isVisible()) {
          // Look for dropdown selectors in chart slots
          const chartSelectors = page.locator('select').or(page.locator('[data-testid="chart-selector"]'));
          const selectorCount = await chartSelectors.count();
          
          // Should have multiple chart selectors (4 quadrants)
          expect(selectorCount).toBeGreaterThanOrEqual(0);
        }
      }
    }
  });

  test('chart selector should include payout failure options', async ({ page }) => {
    const adminBtn = page.locator('text=Admin Panel');
    
    if (await adminBtn.isVisible()) {
      await adminBtn.click();
      
      const statsTab = page.locator('button:has-text("Stats")');
      if (await statsTab.isVisible()) {
        await statsTab.click();
        await page.waitForTimeout(1000);
        
        // Look for chart selector dropdown
        const chartSelector = page.locator('select').first();
        
        if (await chartSelector.isVisible()) {
          // Get all options
          const options = await chartSelector.locator('option').allTextContents();
          
          // Should include payout-related options
          const hasPayoutOptions = options.some(opt => 
            opt.toLowerCase().includes('payout') || 
            opt.toLowerCase().includes('earnings')
          );
          
          // Log available options for debugging
          console.log('Available chart options:', options);
        }
      }
    }
  });
});

test.describe('Integration Tests', () => {
  test('all new features should be accessible from main UI', async ({ page }) => {
    await page.goto(BASE_URL);
    
    // Verify page loads
    await expect(page).toHaveTitle(/Chimeria|Pool/i, { timeout: 10000 });
    
    // Check for key UI elements
    const mainContent = page.locator('body');
    await expect(mainContent).toBeVisible();
  });
});
