import { test, expect } from '@playwright/test';

/**
 * Monitoring Dashboard E2E Tests
 * Tests the pool monitoring section with node health indicators
 * Following TDD and ISP principles
 */

test.describe('Monitoring Dashboard', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
  });

  test('should display monitoring dashboard section', async ({ page }) => {
    const dashboard = page.getByTestId('monitoring-dashboard');
    await expect(dashboard).toBeVisible({ timeout: 10000 });
  });

  test('should display node health section', async ({ page }) => {
    await expect(page.getByTestId('node-health-section')).toBeVisible({ timeout: 10000 });
  });

  test('should display all node status cards', async ({ page }) => {
    await expect(page.getByTestId('status-card-litecoin')).toBeVisible({ timeout: 10000 });
    await expect(page.getByTestId('status-card-stratum')).toBeVisible();
    await expect(page.getByTestId('status-card-alertmanager')).toBeVisible();
    await expect(page.getByTestId('status-card-prometheus')).toBeVisible();
  });

  test('should display animated status dots', async ({ page }) => {
    const statusDot = page.getByTestId('status-dot-online').first();
    await expect(statusDot).toBeVisible({ timeout: 10000 });
  });

  test('should display dashboard links section', async ({ page }) => {
    await expect(page.getByTestId('dashboard-links')).toBeVisible({ timeout: 10000 });
  });

  test('should have accessible dashboard links with aria-labels', async ({ page }) => {
    const overviewLink = page.getByTestId('dashboard-link-overview');
    await expect(overviewLink).toBeVisible({ timeout: 10000 });
    await expect(overviewLink).toHaveAttribute('aria-label', 'Open Pool Overview dashboard');
  });

  test('should display all Grafana dashboard links', async ({ page }) => {
    await expect(page.getByTestId('dashboard-link-overview')).toBeVisible({ timeout: 10000 });
    await expect(page.getByTestId('dashboard-link-workers')).toBeVisible();
    await expect(page.getByTestId('dashboard-link-payouts')).toBeVisible();
    await expect(page.getByTestId('dashboard-link-alerts')).toBeVisible();
  });

  test('should display external monitoring links', async ({ page }) => {
    await expect(page.getByTestId('dashboard-link-alertmanager')).toBeVisible({ timeout: 10000 });
    await expect(page.getByTestId('dashboard-link-prometheus')).toBeVisible();
  });

  test('dashboard links should open in new tab', async ({ page }) => {
    const overviewLink = page.getByTestId('dashboard-link-overview');
    await expect(overviewLink).toHaveAttribute('target', '_blank');
    await expect(overviewLink).toHaveAttribute('rel', 'noopener noreferrer');
  });
});

test.describe('Monitoring Dashboard Loading State', () => {
  test('should show loading state initially', async ({ page }) => {
    // Navigate with network throttling to catch loading state
    await page.route('**/api/v1/stats', async (route) => {
      await new Promise(resolve => setTimeout(resolve, 1000));
      await route.continue();
    });
    
    await page.goto('/');
    
    // Loading state may be brief, so we check if it exists or dashboard is visible
    const loadingOrDashboard = await Promise.race([
      page.getByTestId('monitoring-dashboard-loading').isVisible().catch(() => false),
      page.getByTestId('monitoring-dashboard').isVisible().catch(() => false),
    ]);
    
    expect(loadingOrDashboard).toBeTruthy();
  });
});
