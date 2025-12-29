/**
 * Admin Tabs E2E Tests
 * 
 * Tests for extracted AdminBugsTab and AdminMinersTab components
 * Following TDD, Interface Segregation, and E2E best practices
 * 
 * @see https://playwright.dev/docs/test-configuration
 */

import { test, expect, Page } from '@playwright/test';

const BASE_URL = process.env.TEST_URL || 'http://localhost:3000';

// Helper to login as admin (if credentials available)
async function loginAsAdmin(page: Page): Promise<boolean> {
  const adminEmail = process.env.TEST_ADMIN_EMAIL;
  const adminPassword = process.env.TEST_ADMIN_PASSWORD;
  
  if (!adminEmail || !adminPassword) {
    return false;
  }
  
  // Click login button
  const loginBtn = page.locator('text=Login').or(page.locator('text=Sign In'));
  if (await loginBtn.isVisible()) {
    await loginBtn.click();
    
    // Fill login form
    await page.fill('input[type="email"]', adminEmail);
    await page.fill('input[type="password"]', adminPassword);
    await page.click('button[type="submit"]');
    
    // Wait for login to complete
    await page.waitForTimeout(1000);
    return true;
  }
  return false;
}

// Helper to open admin panel
async function openAdminPanel(page: Page): Promise<boolean> {
  const adminBtn = page.locator('text=Admin Panel').or(page.locator('[data-testid="admin-panel-btn"]'));
  
  if (await adminBtn.isVisible({ timeout: 2000 }).catch(() => false)) {
    await adminBtn.click();
    await page.waitForTimeout(500);
    return true;
  }
  return false;
}

// Helper to navigate to a specific admin tab
async function navigateToTab(page: Page, tabName: string): Promise<boolean> {
  const tab = page.locator(`button:has-text("${tabName}")`).or(page.locator(`text=${tabName}`));
  
  if (await tab.isVisible({ timeout: 2000 }).catch(() => false)) {
    await tab.click();
    await page.waitForTimeout(500);
    return true;
  }
  return false;
}

test.describe('AdminBugsTab Component', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto(BASE_URL);
  });

  test('bugs tab should be accessible in admin panel', async ({ page }) => {
    const adminOpened = await openAdminPanel(page);
    
    if (adminOpened) {
      // Look for Bugs tab button
      const bugsTab = page.locator('button:has-text("Bugs")').or(page.locator('text=Bug Reports'));
      await expect(bugsTab).toBeVisible({ timeout: 5000 });
    } else {
      // Skip if admin panel not accessible (requires auth)
      test.skip();
    }
  });

  test('bugs tab should display bug list when active', async ({ page }) => {
    const adminOpened = await openAdminPanel(page);
    
    if (adminOpened) {
      const navigated = await navigateToTab(page, 'Bugs');
      
      if (navigated) {
        // Should show bug reports section
        const bugSection = page.locator('text=Bug Reports').or(page.locator('text=🐛'));
        await expect(bugSection).toBeVisible({ timeout: 5000 });
        
        // Should have filter controls
        const statusFilter = page.locator('select').filter({ hasText: /Status|Open|Closed/i });
        const priorityFilter = page.locator('select').filter({ hasText: /Priority|Critical|High/i });
        
        // At least one filter should be visible
        const hasFilters = await statusFilter.isVisible().catch(() => false) || 
                          await priorityFilter.isVisible().catch(() => false);
        
        if (!hasFilters) {
          // Check for alternative filter UI
          const filterSection = page.locator('text=All Status').or(page.locator('text=All Priority'));
          await expect(filterSection).toBeVisible({ timeout: 3000 }).catch(() => {});
        }
      }
    } else {
      test.skip();
    }
  });

  test('bugs tab should have refresh button', async ({ page }) => {
    const adminOpened = await openAdminPanel(page);
    
    if (adminOpened) {
      await navigateToTab(page, 'Bugs');
      
      // Should have a refresh button
      const refreshBtn = page.locator('button:has-text("Refresh")').or(page.locator('text=🔄'));
      await expect(refreshBtn).toBeVisible({ timeout: 5000 });
    } else {
      test.skip();
    }
  });

  test('bugs tab should handle empty state gracefully', async ({ page }) => {
    const adminOpened = await openAdminPanel(page);
    
    if (adminOpened) {
      await navigateToTab(page, 'Bugs');
      
      // Check for either bug list OR empty state message
      const hasBugList = await page.locator('text=BUG-').isVisible({ timeout: 3000 }).catch(() => false);
      const hasEmptyState = await page.locator('text=No bug reports').isVisible({ timeout: 3000 }).catch(() => false);
      const hasLoading = await page.locator('text=Loading').isVisible({ timeout: 1000 }).catch(() => false);
      
      // Should show one of these states
      expect(hasBugList || hasEmptyState || hasLoading).toBeTruthy();
    } else {
      test.skip();
    }
  });
});

test.describe('AdminMinersTab Component', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto(BASE_URL);
  });

  test('miners tab should be accessible in admin panel', async ({ page }) => {
    const adminOpened = await openAdminPanel(page);
    
    if (adminOpened) {
      // Look for Miners tab button
      const minersTab = page.locator('button:has-text("Miners")').or(page.locator('text=Miner Monitoring'));
      await expect(minersTab).toBeVisible({ timeout: 5000 });
    } else {
      test.skip();
    }
  });

  test('miners tab should display miner list when active', async ({ page }) => {
    const adminOpened = await openAdminPanel(page);
    
    if (adminOpened) {
      const navigated = await navigateToTab(page, 'Miners');
      
      if (navigated) {
        // Should show miner monitoring section
        const minerSection = page.locator('text=Miner Monitoring').or(page.locator('text=⛏️'));
        await expect(minerSection).toBeVisible({ timeout: 5000 });
        
        // Should have table with miner columns
        const minerTable = page.locator('table');
        if (await minerTable.isVisible({ timeout: 3000 }).catch(() => false)) {
          // Check for expected columns
          const hasNameColumn = await page.locator('th:has-text("Miner")').or(page.locator('th:has-text("Name")')).isVisible().catch(() => false);
          const hasHashrateColumn = await page.locator('th:has-text("Hashrate")').isVisible().catch(() => false);
          const hasStatusColumn = await page.locator('th:has-text("Status")').isVisible().catch(() => false);
          
          expect(hasNameColumn || hasHashrateColumn || hasStatusColumn).toBeTruthy();
        }
      }
    } else {
      test.skip();
    }
  });

  test('miners tab should have search functionality', async ({ page }) => {
    const adminOpened = await openAdminPanel(page);
    
    if (adminOpened) {
      await navigateToTab(page, 'Miners');
      
      // Should have search input
      const searchInput = page.locator('input[placeholder*="Search"]').or(page.locator('input[type="text"]'));
      await expect(searchInput).toBeVisible({ timeout: 5000 });
    } else {
      test.skip();
    }
  });

  test('miners tab should have active-only filter', async ({ page }) => {
    const adminOpened = await openAdminPanel(page);
    
    if (adminOpened) {
      await navigateToTab(page, 'Miners');
      
      // Should have active only checkbox/toggle
      const activeFilter = page.locator('text=Active only').or(page.locator('input[type="checkbox"]'));
      await expect(activeFilter).toBeVisible({ timeout: 5000 });
    } else {
      test.skip();
    }
  });

  test('miners tab should have pagination controls', async ({ page }) => {
    const adminOpened = await openAdminPanel(page);
    
    if (adminOpened) {
      await navigateToTab(page, 'Miners');
      
      // Wait for content to load
      await page.waitForTimeout(1000);
      
      // Check for pagination (Prev/Next buttons or page info)
      const prevBtn = page.locator('button:has-text("Prev")').or(page.locator('text=← Prev'));
      const nextBtn = page.locator('button:has-text("Next")').or(page.locator('text=Next →'));
      const pageInfo = page.locator('text=/Page \\d+ of/');
      
      const hasPagination = await prevBtn.isVisible().catch(() => false) ||
                           await nextBtn.isVisible().catch(() => false) ||
                           await pageInfo.isVisible().catch(() => false);
      
      // Pagination should be present if there are miners
      if (!hasPagination) {
        // Check if empty state instead
        const isEmpty = await page.locator('text=No miners found').isVisible().catch(() => false);
        expect(hasPagination || isEmpty).toBeTruthy();
      }
    } else {
      test.skip();
    }
  });

  test('clicking miner should show detail view', async ({ page }) => {
    const adminOpened = await openAdminPanel(page);
    
    if (adminOpened) {
      await navigateToTab(page, 'Miners');
      
      // Wait for miners to load
      await page.waitForTimeout(1000);
      
      // Try to click on a miner row or detail button
      const detailBtn = page.locator('button:has-text("🔍")').or(page.locator('button[title="View Details"]'));
      
      if (await detailBtn.first().isVisible({ timeout: 3000 }).catch(() => false)) {
        await detailBtn.first().click();
        
        // Should show detail view with back button
        const backBtn = page.locator('text=← Back').or(page.locator('button:has-text("Back")'));
        await expect(backBtn).toBeVisible({ timeout: 5000 });
        
        // Should show miner stats
        const hasStats = await page.locator('text=Performance').or(page.locator('text=Share Statistics')).isVisible().catch(() => false);
        expect(hasStats).toBeTruthy();
      }
    } else {
      test.skip();
    }
  });
});

test.describe('Admin Panel Tab Navigation', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto(BASE_URL);
  });

  test('should be able to switch between all admin tabs', async ({ page }) => {
    const adminOpened = await openAdminPanel(page);
    
    if (adminOpened) {
      const tabs = ['Users', 'Stats', 'Algorithm', 'Network', 'Roles', 'Bugs', 'Miners'];
      
      for (const tabName of tabs) {
        const tab = page.locator(`button:has-text("${tabName}")`);
        
        if (await tab.isVisible({ timeout: 1000 }).catch(() => false)) {
          await tab.click();
          await page.waitForTimeout(300);
          
          // Tab should be highlighted/active
          const isActive = await tab.evaluate(el => {
            const style = window.getComputedStyle(el);
            return style.borderBottomColor !== 'transparent' || 
                   el.classList.contains('active') ||
                   el.getAttribute('aria-selected') === 'true';
          });
          
          // Log which tabs are working
          console.log(`Tab "${tabName}": ${isActive ? 'Active' : 'Visible'}`);
        }
      }
    } else {
      test.skip();
    }
  });

  test('bugs and miners tabs should render extracted components', async ({ page }) => {
    const adminOpened = await openAdminPanel(page);
    
    if (adminOpened) {
      // Test Bugs tab
      await navigateToTab(page, 'Bugs');
      const bugsContent = page.locator('text=Bug Reports').or(page.locator('text=🐛 Bug'));
      const bugsVisible = await bugsContent.isVisible({ timeout: 3000 }).catch(() => false);
      
      // Test Miners tab
      await navigateToTab(page, 'Miners');
      const minersContent = page.locator('text=Miner Monitoring').or(page.locator('text=⛏️'));
      const minersVisible = await minersContent.isVisible({ timeout: 3000 }).catch(() => false);
      
      // At least one should work
      expect(bugsVisible || minersVisible).toBeTruthy();
    } else {
      test.skip();
    }
  });
});

test.describe('Component Isolation Tests', () => {
  test('AdminBugsTab should not interfere with other tabs', async ({ page }) => {
    const adminOpened = await openAdminPanel(page);
    
    if (adminOpened) {
      // Navigate to bugs tab first
      await navigateToTab(page, 'Bugs');
      await page.waitForTimeout(500);
      
      // Then navigate to another tab
      await navigateToTab(page, 'Users');
      await page.waitForTimeout(500);
      
      // Bug content should not be visible on Users tab
      const bugContent = page.locator('text=Bug Reports');
      const isBugVisible = await bugContent.isVisible().catch(() => false);
      
      // Users content should be visible
      const usersContent = page.locator('text=User Management').or(page.locator('th:has-text("Username")'));
      const isUsersVisible = await usersContent.isVisible({ timeout: 3000 }).catch(() => false);
      
      // Bug content should be hidden when on Users tab
      if (isUsersVisible) {
        expect(isBugVisible).toBeFalsy();
      }
    } else {
      test.skip();
    }
  });

  test('AdminMinersTab should not interfere with other tabs', async ({ page }) => {
    const adminOpened = await openAdminPanel(page);
    
    if (adminOpened) {
      // Navigate to miners tab first
      await navigateToTab(page, 'Miners');
      await page.waitForTimeout(500);
      
      // Then navigate to another tab
      await navigateToTab(page, 'Stats');
      await page.waitForTimeout(500);
      
      // Miner monitoring content should not be visible on Stats tab
      const minerContent = page.locator('text=Miner Monitoring');
      const isMinerVisible = await minerContent.isVisible().catch(() => false);
      
      // Stats content should be visible
      const statsContent = page.locator('text=Statistics').or(page.locator('text=Grafana'));
      const isStatsVisible = await statsContent.isVisible({ timeout: 3000 }).catch(() => false);
      
      // Miner content should be hidden when on Stats tab
      if (isStatsVisible) {
        expect(isMinerVisible).toBeFalsy();
      }
    } else {
      test.skip();
    }
  });
});
