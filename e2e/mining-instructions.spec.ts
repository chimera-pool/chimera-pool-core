import { test, expect } from '@playwright/test';

/**
 * Mining Instructions E2E Tests
 * Tests the Connect Your Miner section with multi-network support
 * Following TDD and ISP principles
 */

test.describe('Mining Instructions Multi-Network', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
  });

  test('should display mining instructions section', async ({ page }) => {
    const instructions = page.getByTestId('mining-instructions-multi-network');
    await expect(instructions).toBeVisible({ timeout: 15000 });
  });

  test('should display instructions title', async ({ page }) => {
    const title = page.getByTestId('instructions-title');
    await expect(title).toBeVisible({ timeout: 15000 });
    await expect(title).toContainText('Connect Your Miner');
  });

  test('should display network selector', async ({ page }) => {
    const selector = page.getByTestId('network-selector');
    await expect(selector).toBeVisible({ timeout: 15000 });
  });

  test('should have Litecoin network button', async ({ page }) => {
    const ltcBtn = page.getByTestId('network-btn-litecoin');
    await expect(ltcBtn).toBeVisible({ timeout: 15000 });
  });

  test('should display connection details section for active network', async ({ page }) => {
    await expect(page.getByTestId('connection-details-section')).toBeVisible({ timeout: 15000 });
  });

  test('should display pool address', async ({ page }) => {
    const poolAddress = page.getByTestId('pool-address');
    await expect(poolAddress).toBeVisible({ timeout: 15000 });
    await expect(poolAddress).toContainText('stratum+tcp://');
  });

  test('should have copy pool address button', async ({ page }) => {
    const copyBtn = page.getByTestId('copy-pool-address-btn');
    await expect(copyBtn).toBeVisible({ timeout: 15000 });
    await expect(copyBtn).toHaveAttribute('aria-label', 'Copy pool address');
  });

  test('should display step-by-step guide', async ({ page }) => {
    await expect(page.getByTestId('step-by-step-guide')).toBeVisible({ timeout: 15000 });
  });

  test('should display miner configs section', async ({ page }) => {
    await expect(page.getByTestId('miner-configs-section')).toBeVisible({ timeout: 15000 });
  });

  test('should display CGMiner card', async ({ page }) => {
    const cgminerCard = page.getByTestId('miner-card-cgminer');
    await expect(cgminerCard).toBeVisible({ timeout: 15000 });
  });

  test('should display troubleshooting section', async ({ page }) => {
    await expect(page.getByTestId('troubleshooting-section')).toBeVisible({ timeout: 15000 });
  });

  test('should display wallet reminder', async ({ page }) => {
    await expect(page.getByTestId('wallet-reminder')).toBeVisible({ timeout: 15000 });
  });
});

test.describe('Mining Instructions Network Switching', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
  });

  test('network buttons should be accessible with aria attributes', async ({ page }) => {
    const ltcBtn = page.getByTestId('network-btn-litecoin');
    await expect(ltcBtn).toBeVisible({ timeout: 15000 });
    await expect(ltcBtn).toHaveAttribute('role', 'tab');
    await expect(ltcBtn).toHaveAttribute('aria-selected', 'true');
  });

  test('clicking inactive network should show coming soon message', async ({ page }) => {
    // Click on Bitcoin (inactive network)
    const btcBtn = page.getByTestId('network-btn-bitcoin');
    await expect(btcBtn).toBeVisible({ timeout: 15000 });
    await btcBtn.click();
    
    // Should show coming soon message
    await expect(page.getByTestId('coming-soon-message')).toBeVisible();
  });

  test('network selector should have tablist role', async ({ page }) => {
    const selector = page.getByTestId('network-selector');
    await expect(selector).toHaveAttribute('role', 'tablist');
  });
});

test.describe('Mining Instructions Copy Functionality', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
  });

  test('copy button should be clickable', async ({ page }) => {
    const copyBtn = page.getByTestId('copy-pool-address-btn');
    await expect(copyBtn).toBeVisible({ timeout: 15000 });
    await expect(copyBtn).toBeEnabled();
  });

  test('miner card should be expandable', async ({ page }) => {
    const cgminerCard = page.getByTestId('miner-card-cgminer');
    await expect(cgminerCard).toBeVisible({ timeout: 15000 });
    
    // CGMiner should be expanded by default
    const copyBtn = page.getByTestId('copy-cgminer-btn');
    await expect(copyBtn).toBeVisible();
  });
});
