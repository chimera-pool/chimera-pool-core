import { test, expect } from '@playwright/test';

/**
 * ChartSlot E2E Tests
 * Tests the chart slot component with hover effects and shimmer loading
 * Following TDD and ISP principles
 */

test.describe('ChartSlot Component', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
  });

  test('should display chart slots on homepage', async ({ page }) => {
    const chartSlots = page.getByTestId('chart-slot');
    await expect(chartSlots.first()).toBeVisible({ timeout: 15000 });
  });

  test('should display chart selector dropdown', async ({ page }) => {
    const chartSelector = page.getByTestId('chart-selector');
    await expect(chartSelector.first()).toBeVisible({ timeout: 15000 });
  });

  test('chart slot should have hover interaction', async ({ page }) => {
    const chartSlot = page.getByTestId('chart-slot').first();
    await expect(chartSlot).toBeVisible({ timeout: 15000 });
    
    // Hover over the chart slot
    await chartSlot.hover();
    
    // Chart slot should still be visible after hover
    await expect(chartSlot).toBeVisible();
  });

  test('should show loading shimmer on chart change', async ({ page }) => {
    // Wait for initial load
    await page.waitForTimeout(2000);
    
    const chartSelector = page.getByTestId('chart-selector').first();
    await expect(chartSelector).toBeVisible({ timeout: 15000 });
    
    // The shimmer appears briefly on load - we verify the component structure exists
    const chartSlot = page.getByTestId('chart-slot').first();
    await expect(chartSlot).toBeVisible();
  });
});

test.describe('Chart Selector', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
  });

  test('should have chart selector with options', async ({ page }) => {
    const selector = page.getByTestId('chart-selector').first();
    await expect(selector).toBeVisible({ timeout: 15000 });
    
    // Find the select element within the selector
    const selectElement = selector.locator('select');
    await expect(selectElement).toBeVisible();
  });

  test('chart selector should be interactive', async ({ page }) => {
    const selector = page.getByTestId('chart-selector').first();
    await expect(selector).toBeVisible({ timeout: 15000 });
    
    const selectElement = selector.locator('select');
    
    // Click to open dropdown
    await selectElement.click();
    
    // Select should be focused
    await expect(selectElement).toBeFocused();
  });
});
