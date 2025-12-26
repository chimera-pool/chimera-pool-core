/**
 * Charts Integration E2E Tests
 * 
 * Tests for verifying Grafana chart functionality:
 * 1. Grafana health check works
 * 2. Charts load and display data
 * 3. Prometheus datasource connection works
 */

import { test, expect } from '@playwright/test';

const BASE_URL = process.env.TEST_URL || 'https://206.162.80.230';

test.describe('Charts Integration', () => {
  // Skip beforeEach page load - handle in individual tests as needed
  
  test('Grafana health endpoint responds', async ({ request }) => {
    const response = await request.get(`${BASE_URL}/grafana/api/health`);
    expect(response.status()).toBe(200);
    const health = await response.json();
    expect(health.database).toBe('ok');
  });

  test('Pool Mining Statistics section is visible', async ({ page }) => {
    await page.goto(BASE_URL);
    const statsSection = page.locator('text=Pool Mining Statistics');
    await expect(statsSection).toBeVisible({ timeout: 15000 });
  });

  test('Grafana Connected indicator is shown', async ({ page }) => {
    await page.goto(BASE_URL);
    const indicator = page.locator('text=Grafana Connected');
    await expect(indicator).toBeVisible({ timeout: 15000 });
  });

  test('Chart dropdowns are present', async ({ page }) => {
    await page.goto(BASE_URL);
    // Wait for charts section to load
    await page.waitForSelector('text=Pool Mining Statistics', { timeout: 15000 });
    
    // Check for chart dropdown selectors
    const dropdowns = page.locator('select');
    const count = await dropdowns.count();
    expect(count).toBeGreaterThanOrEqual(4); // 4 chart quadrants
  });

  test('Charts load without 400 errors', async ({ page }) => {
    const errors: string[] = [];
    
    // Listen for failed Prometheus query requests
    page.on('response', response => {
      if (response.url().includes('/grafana/api/ds/query') && response.status() === 400) {
        errors.push(`Prometheus query failed: ${response.url()}`);
      }
    });

    await page.goto(BASE_URL);
    // Wait for charts to attempt loading
    await page.waitForSelector('text=Pool Mining Statistics', { timeout: 15000 });
    await page.waitForTimeout(3000);

    // Should have no 400 errors from Prometheus queries
    expect(errors.length).toBe(0);
  });

  test('Prometheus proxy endpoint works', async ({ request }) => {
    // This tests the nginx proxy to prometheus - use metrics endpoint
    const response = await request.get(`${BASE_URL}/prometheus/api/v1/status/runtimeinfo`);
    // Should return 200 if prometheus is accessible
    expect(response.status()).toBe(200);
  });

  test('AlertManager proxy endpoint works', async ({ request }) => {
    // This tests the nginx proxy to alertmanager - use status endpoint
    const response = await request.get(`${BASE_URL}/alertmanager/api/v2/status`);
    // Should return 200 if alertmanager is accessible
    expect(response.status()).toBe(200);
  });
});
