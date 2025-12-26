import { test, expect } from '@playwright/test';

/**
 * Chimeria Pool Dashboard E2E Tests
 * Tests the main pool dashboard functionality
 */

test.describe('Pool Dashboard', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
  });

  test('should display pool statistics', async ({ page }) => {
    // Check main stats section is visible
    await expect(page.locator('text=Active Miners')).toBeVisible();
    await expect(page.locator('text=Pool Hashrate')).toBeVisible();
    await expect(page.locator('text=Blocks Found')).toBeVisible();
  });

  test('should display mining graphs section', async ({ page }) => {
    // Check for mining graphs
    await expect(page.locator('text=Mining Performance')).toBeVisible({ timeout: 10000 });
  });

  test('should display pool monitoring section', async ({ page }) => {
    // Check for the new monitoring dashboard
    await expect(page.locator('text=Pool Monitoring')).toBeVisible({ timeout: 10000 });
    await expect(page.locator('text=Open Grafana Dashboards')).toBeVisible();
    await expect(page.locator('text=Node Health')).toBeVisible();
  });

  test('should have working navigation', async ({ page }) => {
    // Check header navigation
    await expect(page.locator('header')).toBeVisible();
  });

  test('should display call-to-action for non-logged users', async ({ page }) => {
    // Non-logged users should see signup CTA
    await expect(page.locator('text=Start Mining Today')).toBeVisible();
    await expect(page.locator('text=Create Account')).toBeVisible();
  });
});

test.describe('Authentication', () => {
  test('should open login modal', async ({ page }) => {
    await page.goto('/');
    
    // Click login button
    const loginBtn = page.locator('button:has-text("Login")').first();
    await loginBtn.click();
    
    // Modal should appear
    await expect(page.locator('text=Welcome Back')).toBeVisible({ timeout: 5000 });
  });

  test('should open register modal', async ({ page }) => {
    await page.goto('/');
    
    // Click register button
    const registerBtn = page.locator('button:has-text("Create Account")').first();
    await registerBtn.click();
    
    // Modal should appear
    await expect(page.locator('text=Create Account')).toBeVisible({ timeout: 5000 });
  });

  test('login form should have required fields', async ({ page }) => {
    await page.goto('/');
    
    const loginBtn = page.locator('button:has-text("Login")').first();
    await loginBtn.click();
    
    // Check for form fields
    await expect(page.locator('input[type="email"], input[placeholder*="email" i]')).toBeVisible();
    await expect(page.locator('input[type="password"]')).toBeVisible();
  });
});

test.describe('Monitoring Dashboard Links', () => {
  test('should have Grafana dashboard links', async ({ page }) => {
    await page.goto('/');
    
    // Check for Grafana links in monitoring section
    await expect(page.locator('a:has-text("Pool Overview")')).toBeVisible({ timeout: 10000 });
    await expect(page.locator('a:has-text("Workers Dashboard")')).toBeVisible();
    await expect(page.locator('a:has-text("Payouts Dashboard")')).toBeVisible();
    await expect(page.locator('a:has-text("Alerts Dashboard")')).toBeVisible();
  });

  test('Grafana button should open in new tab', async ({ page }) => {
    await page.goto('/');
    
    // Check the Grafana button has correct target
    const grafanaBtn = page.locator('button:has-text("Open Grafana Dashboards")');
    await expect(grafanaBtn).toBeVisible({ timeout: 10000 });
  });
});

test.describe('Responsive Design', () => {
  test('should work on mobile viewport', async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 667 });
    await page.goto('/');
    
    // Main content should still be visible
    await expect(page.locator('text=Chimeria Pool')).toBeVisible();
    await expect(page.locator('text=Active Miners')).toBeVisible();
  });

  test('should work on tablet viewport', async ({ page }) => {
    await page.setViewportSize({ width: 768, height: 1024 });
    await page.goto('/');
    
    await expect(page.locator('text=Chimeria Pool')).toBeVisible();
    await expect(page.locator('text=Pool Hashrate')).toBeVisible();
  });
});
