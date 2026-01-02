import { test, expect } from '@playwright/test';

/**
 * Wallet Management E2E Tests
 * Tests the multi-wallet payout settings functionality
 * Following TDD and ISP principles - each test has single responsibility
 */

test.describe('Wallet Management', () => {
  test.beforeEach(async ({ page }) => {
    // Navigate to the app
    await page.goto('/');
  });

  test.describe('Unauthenticated User', () => {
    test('should not show wallet manager section when not logged in', async ({ page }) => {
      // Wallet manager should not be visible to unauthenticated users
      await expect(page.getByTestId('wallet-manager-section')).not.toBeVisible();
    });
  });

  test.describe('Authenticated User', () => {
    test.beforeEach(async ({ page }) => {
      // Login flow
      await page.getByTestId('header-login-btn').click();
      await expect(page.getByTestId('auth-modal-container')).toBeVisible();
      
      await page.getByTestId('login-email-input').fill('test@example.com');
      await page.getByTestId('login-password-input').fill('TestPassword123!');
      await page.getByTestId('login-submit-btn').click();
      
      // Wait for login to complete and modal to close
      await expect(page.getByTestId('auth-modal-container')).not.toBeVisible({ timeout: 10000 });
      
      // Navigate to settings/wallet section
      await page.getByTestId('nav-settings-link').click();
    });

    test('should display wallet manager section', async ({ page }) => {
      await expect(page.getByTestId('wallet-manager-section')).toBeVisible();
      await expect(page.getByTestId('wallet-manager-header')).toBeVisible();
    });

    test('should display add wallet button', async ({ page }) => {
      const addBtn = page.getByTestId('wallet-add-btn');
      await expect(addBtn).toBeVisible();
      await expect(addBtn).toHaveAttribute('aria-label', 'Add new wallet');
    });

    test('should display wallet summary bar', async ({ page }) => {
      await expect(page.getByTestId('wallet-summary-bar')).toBeVisible();
      await expect(page.getByTestId('wallet-summary-active')).toBeVisible();
    });

    test('should open add wallet form when clicking add button', async ({ page }) => {
      await page.getByTestId('wallet-add-btn').click();
      
      await expect(page.getByTestId('wallet-add-form-container')).toBeVisible();
      await expect(page.getByTestId('wallet-add-form')).toBeVisible();
      await expect(page.getByTestId('wallet-address-input')).toBeVisible();
      await expect(page.getByTestId('wallet-label-input')).toBeVisible();
    });

    test('should have accessible form inputs with aria-labels', async ({ page }) => {
      await page.getByTestId('wallet-add-btn').click();
      
      const addressInput = page.getByTestId('wallet-address-input');
      const labelInput = page.getByTestId('wallet-label-input');
      
      await expect(addressInput).toHaveAttribute('aria-label', 'Wallet address');
      await expect(labelInput).toHaveAttribute('aria-label', 'Wallet label');
    });

    test('should close add wallet form when clicking cancel', async ({ page }) => {
      await page.getByTestId('wallet-add-btn').click();
      await expect(page.getByTestId('wallet-add-form-container')).toBeVisible();
      
      await page.getByTestId('wallet-cancel-btn').click();
      await expect(page.getByTestId('wallet-add-form-container')).not.toBeVisible();
    });

    test('should show empty state when no wallets configured', async ({ page }) => {
      // This test assumes a fresh user with no wallets
      const emptyState = page.getByTestId('wallet-empty-state');
      const walletList = page.getByTestId('wallet-list');
      
      // Either empty state or wallet list should be visible, not both
      const emptyVisible = await emptyState.isVisible().catch(() => false);
      const listVisible = await walletList.isVisible().catch(() => false);
      
      expect(emptyVisible !== listVisible).toBeTruthy();
    });

    test('should validate wallet address format', async ({ page }) => {
      await page.getByTestId('wallet-add-btn').click();
      
      // Fill in invalid address
      await page.getByTestId('wallet-address-input').fill('invalid-address');
      await page.getByTestId('wallet-label-input').fill('Test Wallet');
      await page.getByTestId('wallet-submit-btn').click();
      
      // Should show error (form should still be visible)
      await expect(page.getByTestId('wallet-add-form-container')).toBeVisible();
    });

    test('should require wallet address field', async ({ page }) => {
      await page.getByTestId('wallet-add-btn').click();
      
      const addressInput = page.getByTestId('wallet-address-input');
      await expect(addressInput).toHaveAttribute('required', '');
    });
  });
});

test.describe('Wallet Form Validation', () => {
  test('address input should have correct placeholder', async ({ page }) => {
    await page.goto('/');
    
    // Login first
    await page.getByTestId('header-login-btn').click();
    await page.getByTestId('login-email-input').fill('test@example.com');
    await page.getByTestId('login-password-input').fill('TestPassword123!');
    await page.getByTestId('login-submit-btn').click();
    await expect(page.getByTestId('auth-modal-container')).not.toBeVisible({ timeout: 10000 });
    
    await page.getByTestId('nav-settings-link').click();
    await page.getByTestId('wallet-add-btn').click();
    
    const addressInput = page.getByTestId('wallet-address-input');
    await expect(addressInput).toHaveAttribute('placeholder', '0x...');
  });
});

test.describe('Wallet Security', () => {
  test('should not expose wallet data in page source for unauthenticated users', async ({ page }) => {
    await page.goto('/');
    
    const content = await page.content();
    // Should not contain wallet addresses in page source
    expect(content).not.toContain('wallet-list');
  });
});
