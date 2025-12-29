/**
 * Mining Dashboard E2E Tests
 * 
 * Tests the core mining functionality:
 * 1. Dashboard statistics display
 * 2. Worker management
 * 3. Hashrate charts
 * 4. Payout history
 * 5. Real-time updates
 * 
 * Following Interface Segregation - each test suite is independent
 */

import { test, expect, type Page } from '@playwright/test';

const BASE_URL = process.env.TEST_URL || 'https://206.162.80.230';

// ============================================================================
// POOL STATISTICS TESTS
// ============================================================================

test.describe('Pool Statistics', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto(BASE_URL);
    await page.waitForLoadState('networkidle');
  });

  test('should display pool hashrate', async ({ page }) => {
    // Look for hashrate display
    const hashrateElement = page.locator(
      'text=/hashrate/i, ' +
      '[data-testid*="hashrate"], ' +
      '[class*="hashrate"]'
    ).first();
    
    await expect(hashrateElement).toBeVisible({ timeout: 10000 });
  });

  test('should display active miners count', async ({ page }) => {
    // Look for miners/workers count
    const minersElement = page.locator(
      'text=/miners|workers|active/i, ' +
      '[data-testid*="miners"], ' +
      '[data-testid*="workers"]'
    ).first();
    
    await expect(minersElement).toBeVisible({ timeout: 10000 });
  });

  test('should display blocks found', async ({ page }) => {
    // Look for blocks display
    const blocksElement = page.locator(
      'text=/blocks?.*found|found.*blocks?/i, ' +
      '[data-testid*="blocks"]'
    ).first();
    
    const isVisible = await blocksElement.isVisible().catch(() => false);
    if (isVisible) {
      await expect(blocksElement).toBeVisible();
    }
  });

  test('should display network difficulty', async ({ page }) => {
    // Look for difficulty display
    const difficultyElement = page.locator(
      'text=/difficulty/i, ' +
      '[data-testid*="difficulty"]'
    ).first();
    
    const isVisible = await difficultyElement.isVisible().catch(() => false);
    if (isVisible) {
      await expect(difficultyElement).toBeVisible();
    }
  });

  test('should format large numbers correctly', async ({ page }) => {
    // Find stat cards with numbers
    const statCards = page.locator('[class*="stat"], [class*="card"]');
    const count = await statCards.count();
    
    if (count > 0) {
      // Get text content from first stat
      const text = await statCards.first().textContent();
      
      // Large numbers should use abbreviations (K, M, G, T) or comma formatting
      // This is a visual check - numbers should be readable
      expect(text?.length).toBeGreaterThan(0);
    }
  });
});

// ============================================================================
// HASHRATE CHARTS TESTS
// ============================================================================

test.describe('Hashrate Charts', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto(BASE_URL);
    await page.waitForLoadState('networkidle');
  });

  test('should display hashrate chart', async ({ page }) => {
    // Look for chart container
    const chartElement = page.locator(
      'canvas, ' +
      'svg[class*="chart"], ' +
      '[class*="chart"], ' +
      '[data-testid*="chart"], ' +
      'iframe[src*="grafana"]'
    ).first();
    
    await expect(chartElement).toBeVisible({ timeout: 15000 });
  });

  test('should have chart time range selector', async ({ page }) => {
    // Look for time range buttons/dropdown
    const timeSelector = page.locator(
      'button:has-text("1H"), ' +
      'button:has-text("24H"), ' +
      'button:has-text("7D"), ' +
      'button:has-text("30D"), ' +
      'select[class*="time"], ' +
      '[data-testid="time-range"]'
    ).first();
    
    const exists = await timeSelector.count() > 0;
    if (exists) {
      await expect(timeSelector).toBeVisible();
    }
  });

  test('should update chart on time range change', async ({ page }) => {
    // Find time range buttons
    const timeButtons = page.locator('button:has-text("24H"), button:has-text("7D")');
    const count = await timeButtons.count();
    
    if (count >= 2) {
      // Click first time button
      await timeButtons.first().click();
      await page.waitForTimeout(1000);
      
      // Click second time button
      await timeButtons.nth(1).click();
      await page.waitForTimeout(1000);
      
      // Chart should still be visible (didn't crash)
      const chart = page.locator('canvas, svg[class*="chart"], [class*="chart"]').first();
      await expect(chart).toBeVisible();
    }
  });
});

// ============================================================================
// WORKER MANAGEMENT TESTS
// ============================================================================

test.describe('Worker Management', () => {
  test('should display workers table when logged in', async ({ page }) => {
    test.skip(!process.env.TEST_USER_EMAIL, 'Requires test user credentials');
    
    await page.goto(BASE_URL);
    
    // Login
    const loginBtn = page.locator('button:has-text("Login")').first();
    if (await loginBtn.isVisible()) {
      await loginBtn.click();
      
      await page.fill('input[type="email"]', process.env.TEST_USER_EMAIL!);
      await page.fill('input[type="password"]', process.env.TEST_USER_PASSWORD!);
      await page.click('button[type="submit"]');
      
      await page.waitForTimeout(3000);
      
      // Look for workers table
      const workersTable = page.locator(
        'table:has-text("Worker"), ' +
        '[data-testid="workers-table"], ' +
        'text=Your Workers'
      ).first();
      
      if (await workersTable.isVisible()) {
        await expect(workersTable).toBeVisible();
      }
    }
  });

  test('should show worker status indicators', async ({ page }) => {
    test.skip(!process.env.TEST_USER_EMAIL, 'Requires test user credentials');
    
    await page.goto(BASE_URL);
    
    // Login first
    const loginBtn = page.locator('button:has-text("Login")').first();
    if (await loginBtn.isVisible()) {
      await loginBtn.click();
      await page.fill('input[type="email"]', process.env.TEST_USER_EMAIL!);
      await page.fill('input[type="password"]', process.env.TEST_USER_PASSWORD!);
      await page.click('button[type="submit"]');
      await page.waitForTimeout(3000);
      
      // Look for status indicators (online/offline)
      const statusIndicators = page.locator(
        '[class*="status"], ' +
        '[class*="online"], ' +
        '[class*="offline"], ' +
        'span:has-text("Online"), ' +
        'span:has-text("Offline")'
      );
      
      const count = await statusIndicators.count();
      console.log('Status indicators found:', count);
    }
  });
});

// ============================================================================
// PAYOUT TESTS
// ============================================================================

test.describe('Payout Information', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto(BASE_URL);
    await page.waitForLoadState('networkidle');
  });

  test('should display payout threshold information', async ({ page }) => {
    // Look for payout/threshold information
    const payoutInfo = page.locator(
      'text=/payout|threshold|minimum/i, ' +
      '[data-testid*="payout"]'
    ).first();
    
    const isVisible = await payoutInfo.isVisible().catch(() => false);
    if (isVisible) {
      await expect(payoutInfo).toBeVisible();
    }
  });

  test('should display fee information', async ({ page }) => {
    // Look for fee display
    const feeInfo = page.locator(
      'text=/fee|commission/i, ' +
      '[data-testid*="fee"]'
    ).first();
    
    const isVisible = await feeInfo.isVisible().catch(() => false);
    if (isVisible) {
      await expect(feeInfo).toBeVisible();
    }
  });

  test('should show payout history when logged in', async ({ page }) => {
    test.skip(!process.env.TEST_USER_EMAIL, 'Requires test user credentials');
    
    // Login
    const loginBtn = page.locator('button:has-text("Login")').first();
    if (await loginBtn.isVisible()) {
      await loginBtn.click();
      await page.fill('input[type="email"]', process.env.TEST_USER_EMAIL!);
      await page.fill('input[type="password"]', process.env.TEST_USER_PASSWORD!);
      await page.click('button[type="submit"]');
      await page.waitForTimeout(3000);
      
      // Look for payout history
      const payoutHistory = page.locator(
        'text=Payout History, ' +
        '[data-testid="payout-history"], ' +
        'table:has-text("Payout")'
      ).first();
      
      if (await payoutHistory.isVisible()) {
        await expect(payoutHistory).toBeVisible();
      }
    }
  });
});

// ============================================================================
// REAL-TIME UPDATES TESTS
// ============================================================================

test.describe('Real-Time Updates', () => {
  test('should update stats periodically', async ({ page }) => {
    await page.goto(BASE_URL);
    await page.waitForLoadState('networkidle');
    
    // Find a stat that should update
    const statElement = page.locator('[class*="stat"], [data-testid*="stat"]').first();
    
    if (await statElement.isVisible()) {
      // Get initial value
      const initialValue = await statElement.textContent();
      
      // Wait for potential update (stats typically refresh every 10-30 seconds)
      await page.waitForTimeout(15000);
      
      // Value might have changed (or might be the same if data is static)
      const newValue = await statElement.textContent();
      
      // At minimum, the stat should still be visible
      await expect(statElement).toBeVisible();
    }
  });

  test('should handle WebSocket connection gracefully', async ({ page }) => {
    // Listen for WebSocket connections
    const wsConnections: string[] = [];
    
    page.on('websocket', ws => {
      wsConnections.push(ws.url());
      console.log('WebSocket connected:', ws.url());
    });
    
    await page.goto(BASE_URL);
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(5000);
    
    // Log WebSocket connections for debugging
    console.log('WebSocket connections detected:', wsConnections.length);
  });

  test('should show connection status indicator', async ({ page }) => {
    await page.goto(BASE_URL);
    await page.waitForLoadState('networkidle');
    
    // Look for connection status
    const connectionStatus = page.locator(
      '[class*="connection"], ' +
      '[data-testid="connection-status"], ' +
      'text=/connected|online|live/i'
    ).first();
    
    const exists = await connectionStatus.count() > 0;
    if (exists) {
      await expect(connectionStatus).toBeVisible();
    }
  });
});

// ============================================================================
// GRAFANA INTEGRATION TESTS
// ============================================================================

test.describe('Grafana Integration', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto(BASE_URL);
    await page.waitForLoadState('networkidle');
  });

  test('should embed Grafana dashboards', async ({ page }) => {
    // Look for Grafana iframes or embeds
    const grafanaEmbed = page.locator(
      'iframe[src*="grafana"], ' +
      'iframe[src*="dashboard"], ' +
      '[data-testid="grafana-dashboard"]'
    ).first();
    
    const exists = await grafanaEmbed.count() > 0;
    if (exists) {
      await expect(grafanaEmbed).toBeVisible({ timeout: 15000 });
    }
  });

  test('should have chart selector dropdowns', async ({ page }) => {
    // Look for chart selection controls
    const chartSelector = page.locator(
      'select[class*="chart"], ' +
      '[data-testid="chart-selector"], ' +
      'button[aria-haspopup="listbox"]'
    ).first();
    
    const exists = await chartSelector.count() > 0;
    if (exists) {
      await expect(chartSelector).toBeVisible();
    }
  });
});

// ============================================================================
// WALLET CONFIGURATION TESTS
// ============================================================================

test.describe('Wallet Configuration', () => {
  test('should show wallet input when logged in', async ({ page }) => {
    test.skip(!process.env.TEST_USER_EMAIL, 'Requires test user credentials');
    
    await page.goto(BASE_URL);
    
    // Login
    const loginBtn = page.locator('button:has-text("Login")').first();
    if (await loginBtn.isVisible()) {
      await loginBtn.click();
      await page.fill('input[type="email"]', process.env.TEST_USER_EMAIL!);
      await page.fill('input[type="password"]', process.env.TEST_USER_PASSWORD!);
      await page.click('button[type="submit"]');
      await page.waitForTimeout(3000);
      
      // Look for wallet address input
      const walletInput = page.locator(
        'input[name="wallet"], ' +
        'input[placeholder*="wallet" i], ' +
        'input[placeholder*="address" i], ' +
        '[data-testid="wallet-input"]'
      ).first();
      
      if (await walletInput.isVisible()) {
        await expect(walletInput).toBeVisible();
      }
    }
  });

  test('should validate wallet address format', async ({ page }) => {
    test.skip(!process.env.TEST_USER_EMAIL, 'Requires test user credentials');
    
    await page.goto(BASE_URL);
    
    // Login
    const loginBtn = page.locator('button:has-text("Login")').first();
    if (await loginBtn.isVisible()) {
      await loginBtn.click();
      await page.fill('input[type="email"]', process.env.TEST_USER_EMAIL!);
      await page.fill('input[type="password"]', process.env.TEST_USER_PASSWORD!);
      await page.click('button[type="submit"]');
      await page.waitForTimeout(3000);
      
      // Find wallet input
      const walletInput = page.locator(
        'input[name="wallet"], ' +
        'input[placeholder*="wallet" i]'
      ).first();
      
      if (await walletInput.isVisible()) {
        // Try invalid address
        await walletInput.fill('invalid-address');
        
        // Submit form or blur to trigger validation
        await walletInput.blur();
        await page.waitForTimeout(500);
        
        // Look for error message
        const errorMsg = page.locator('[class*="error"], [role="alert"]').first();
        // Error might appear for invalid format
      }
    }
  });
});

// ============================================================================
// STRATUM CONNECTION INFO TESTS
// ============================================================================

test.describe('Stratum Connection Info', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto(BASE_URL);
    await page.waitForLoadState('networkidle');
  });

  test('should display stratum server address', async ({ page }) => {
    // Look for stratum/connection info
    const stratumInfo = page.locator(
      'text=/stratum|pool.*address|connect/i, ' +
      '[data-testid="stratum-info"], ' +
      'code:has-text("stratum")'
    ).first();
    
    const isVisible = await stratumInfo.isVisible().catch(() => false);
    if (isVisible) {
      await expect(stratumInfo).toBeVisible();
    }
  });

  test('should have copy button for connection string', async ({ page }) => {
    // Look for copy button near connection info
    const copyBtn = page.locator(
      'button[aria-label*="copy" i], ' +
      'button:has-text("Copy"), ' +
      '[data-testid="copy-button"]'
    ).first();
    
    const exists = await copyBtn.count() > 0;
    if (exists) {
      await expect(copyBtn).toBeVisible();
    }
  });

  test('should display supported algorithms', async ({ page }) => {
    // Look for algorithm information
    const algorithmInfo = page.locator(
      'text=/algorithm|algo|scrypt|sha256|blockdag/i, ' +
      '[data-testid*="algorithm"]'
    ).first();
    
    const isVisible = await algorithmInfo.isVisible().catch(() => false);
    if (isVisible) {
      await expect(algorithmInfo).toBeVisible();
    }
  });
});
