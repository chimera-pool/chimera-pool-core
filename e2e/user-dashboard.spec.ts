import { test, expect } from '@playwright/test';

/**
 * Chimeria Pool User Dashboard E2E Tests
 * Tests authenticated user functionality
 */

// Test user credentials - should be configured in environment
const TEST_USER = {
  email: process.env.TEST_USER_EMAIL || 'test@example.com',
  password: process.env.TEST_USER_PASSWORD || 'testpassword123',
};

test.describe('User Dashboard (Authenticated)', () => {
  test.skip(({ browserName }) => !process.env.TEST_USER_EMAIL, 'Requires test user credentials');

  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    
    // Login
    const loginBtn = page.locator('button:has-text("Login")').first();
    await loginBtn.click();
    
    await page.fill('input[type="email"], input[placeholder*="email" i]', TEST_USER.email);
    await page.fill('input[type="password"]', TEST_USER.password);
    await page.click('button[type="submit"]:has-text("Login")');
    
    // Wait for login to complete
    await page.waitForSelector('text=Dashboard', { timeout: 10000 });
  });

  test('should display user stats', async ({ page }) => {
    await expect(page.locator('text=Your Hashrate')).toBeVisible();
    await expect(page.locator('text=Total Earnings')).toBeVisible();
    await expect(page.locator('text=Pending Payout')).toBeVisible();
  });

  test('should display miners table', async ({ page }) => {
    await expect(page.locator('text=Your Miners')).toBeVisible();
  });

  test('should display payout settings', async ({ page }) => {
    await expect(page.locator('text=Payout Settings')).toBeVisible();
  });

  test('should display notification settings', async ({ page }) => {
    await expect(page.locator('text=Notification Settings')).toBeVisible();
  });

  test('should display monitoring dashboard', async ({ page }) => {
    await expect(page.locator('text=Pool Monitoring')).toBeVisible();
  });
});

test.describe('User Profile', () => {
  test.skip(({ browserName }) => !process.env.TEST_USER_EMAIL, 'Requires test user credentials');

  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    
    // Login
    const loginBtn = page.locator('button:has-text("Login")').first();
    await loginBtn.click();
    
    await page.fill('input[type="email"], input[placeholder*="email" i]', TEST_USER.email);
    await page.fill('input[type="password"]', TEST_USER.password);
    await page.click('button[type="submit"]:has-text("Login")');
    
    await page.waitForSelector('text=Dashboard', { timeout: 10000 });
  });

  test('should open account settings modal', async ({ page }) => {
    // Click on username in header to open settings
    const usernameBtn = page.locator('header').locator('button').filter({ hasText: /@/ });
    await usernameBtn.click();
    
    await expect(page.locator('text=Account Settings')).toBeVisible({ timeout: 5000 });
  });
});
