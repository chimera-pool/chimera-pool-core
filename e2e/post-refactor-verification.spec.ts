import { test, expect } from '@playwright/test';

/**
 * Post-Refactor Verification E2E Tests
 * Comprehensive testing after AdminPanel segmentation and modal extraction
 * 
 * Tests verify:
 * - Dashboard loads correctly
 * - Navigation works
 * - Auth modals (Login, Register) open/close
 * - Chart dropdowns function
 * - Community page loads
 * - Pool statistics display
 */

test.describe('Post-Refactor Verification Suite', () => {
  
  test.describe('Dashboard Loading', () => {
    test('should load the main dashboard', async ({ page }) => {
      await page.goto('/');
      
      // Verify page title
      await expect(page).toHaveTitle(/Chimera Pool/);
      
      // Verify header elements
      await expect(page.getByRole('banner')).toBeVisible();
      await expect(page.getByRole('img', { name: 'Chimera Pool' })).toBeVisible();
      
      // Verify navigation buttons
      await expect(page.getByRole('button', { name: 'Dashboard' })).toBeVisible();
      await expect(page.getByRole('button', { name: 'Community' })).toBeVisible();
    });

    test('should display pool statistics', async ({ page }) => {
      await page.goto('/');
      
      // Verify stats cards are present
      await expect(page.getByRole('heading', { name: 'Network' })).toBeVisible();
      await expect(page.getByRole('heading', { name: 'Currency' })).toBeVisible();
      await expect(page.getByRole('heading', { name: /Active Miners/i })).toBeVisible();
      await expect(page.getByRole('heading', { name: /Pool Hashrate/i })).toBeVisible();
      await expect(page.getByRole('heading', { name: 'Blocks Found' })).toBeVisible();
      await expect(page.getByRole('heading', { name: 'Min Payout' })).toBeVisible();
    });

    test('should display Grafana charts section', async ({ page }) => {
      await page.goto('/');
      
      // Verify chart section heading
      await expect(page.getByRole('heading', { name: /Pool Mining Statistics/i })).toBeVisible();
      
      // Verify chart dropdowns are present
      const dropdowns = page.getByRole('combobox');
      await expect(dropdowns).toHaveCount(4); // 4 chart quadrants
    });
  });

  test.describe('Navigation', () => {
    test('should navigate to Community page', async ({ page }) => {
      await page.goto('/');
      
      await page.getByRole('button', { name: 'Community' }).click();
      
      // Verify Community page content
      await expect(page.getByRole('heading', { name: /Community Hub/i })).toBeVisible();
      await expect(page.getByText('Announcements')).toBeVisible();
      await expect(page.getByText('Mining Tips')).toBeVisible();
    });

    test('should navigate back to Dashboard', async ({ page }) => {
      await page.goto('/');
      
      // Go to Community
      await page.getByRole('button', { name: 'Community' }).click();
      await expect(page.getByRole('heading', { name: /Community Hub/i })).toBeVisible();
      
      // Go back to Dashboard
      await page.getByRole('button', { name: 'Dashboard' }).click();
      await expect(page.getByRole('heading', { name: /Pool Mining Statistics/i })).toBeVisible();
    });
  });

  test.describe('Authentication Modals', () => {
    test('should open and close Login modal', async ({ page }) => {
      await page.goto('/');
      
      // Open Login modal
      await page.getByRole('banner').getByRole('button', { name: 'Login' }).click();
      
      // Verify modal content
      await expect(page.getByRole('heading', { name: 'Login' })).toBeVisible();
      await expect(page.getByRole('textbox', { name: 'Email Address' })).toBeVisible();
      await expect(page.getByRole('textbox', { name: 'Password' })).toBeVisible();
      
      // Close modal
      await page.getByRole('button', { name: '×' }).click();
      
      // Verify modal closed
      await expect(page.getByRole('heading', { name: 'Login' })).not.toBeVisible();
    });

    test('should open and close Register modal', async ({ page }) => {
      await page.goto('/');
      
      // Open Register modal
      await page.getByRole('button', { name: 'Register' }).click();
      
      // Verify modal content
      await expect(page.getByRole('heading', { name: 'Create Account' })).toBeVisible();
      await expect(page.getByRole('textbox', { name: 'Username' })).toBeVisible();
      await expect(page.getByRole('textbox', { name: 'Email' })).toBeVisible();
      await expect(page.getByRole('textbox', { name: /Password.*min 8/i })).toBeVisible();
      
      // Close modal
      await page.getByRole('button', { name: '×' }).click();
      
      // Verify modal closed
      await expect(page.getByRole('heading', { name: 'Create Account' })).not.toBeVisible();
    });

    test('should show error for invalid login', async ({ page }) => {
      await page.goto('/');
      
      // Open Login modal
      await page.getByRole('banner').getByRole('button', { name: 'Login' }).click();
      
      // Enter invalid credentials
      await page.getByRole('textbox', { name: 'Email Address' }).fill('invalid@test.com');
      await page.getByRole('textbox', { name: 'Password' }).fill('wrongpassword');
      
      // Submit
      await page.locator('form').getByRole('button', { name: 'Login' }).click();
      
      // Verify error message
      await expect(page.getByText(/Invalid credentials/i)).toBeVisible();
    });

    test('should switch between Login and Register', async ({ page }) => {
      await page.goto('/');
      
      // Open Login modal
      await page.getByRole('banner').getByRole('button', { name: 'Login' }).click();
      await expect(page.getByRole('heading', { name: 'Login' })).toBeVisible();
      
      // Switch to Register
      await page.getByText('Create Account').click();
      await expect(page.getByRole('heading', { name: 'Create Account' })).toBeVisible();
      
      // Switch back to Login
      await page.getByText(/Already have an account/i).click();
      await expect(page.getByRole('heading', { name: 'Login' })).toBeVisible();
    });
  });

  test.describe('Chart Dropdowns', () => {
    test('should change chart selection in first quadrant', async ({ page }) => {
      await page.goto('/');
      
      // Find first dropdown and change selection
      const dropdown = page.getByRole('combobox').first();
      await dropdown.selectOption('Pool Hashrate History');
      
      // Verify selection changed
      await expect(page.getByRole('heading', { name: /Pool Hashrate History/i })).toBeVisible();
    });

    test('should change chart selection in workers quadrant', async ({ page }) => {
      await page.goto('/');
      
      // Find workers dropdown (second one)
      const dropdown = page.getByRole('combobox').nth(1);
      await dropdown.selectOption('Workers History');
      
      // Verify selection changed
      await expect(page.getByRole('heading', { name: /Workers History/i })).toBeVisible();
    });
  });

  test.describe('Pool Information Section', () => {
    test('should display pool information', async ({ page }) => {
      await page.goto('/');
      
      // Scroll to pool info section
      await page.getByRole('heading', { name: /Pool Information/i }).scrollIntoViewIfNeeded();
      
      // Verify information items
      await expect(page.getByText('Algorithm')).toBeVisible();
      await expect(page.getByText('Payout System')).toBeVisible();
      await expect(page.getByText('Minimum Payout')).toBeVisible();
      await expect(page.getByText('Payout Frequency')).toBeVisible();
    });
  });

  test.describe('Connect Your Miner Section', () => {
    test('should display miner connection guide', async ({ page }) => {
      await page.goto('/');
      
      // Scroll to connection section
      await page.getByRole('heading', { name: /Connect Your Miner/i }).scrollIntoViewIfNeeded();
      
      // Verify quick start guide
      await expect(page.getByRole('heading', { name: /Quick Start Guide/i })).toBeVisible();
      await expect(page.getByText(/Create an Account/i)).toBeVisible();
      await expect(page.getByText(/Set Your Wallet Address/i)).toBeVisible();
    });

    test('should display hardware type setup options', async ({ page }) => {
      await page.goto('/');
      
      // Scroll to hardware section
      await page.getByRole('heading', { name: /Setup by Hardware Type/i }).scrollIntoViewIfNeeded();
      
      // Verify hardware options
      await expect(page.getByRole('heading', { name: /BlockDAG X30/i })).toBeVisible();
      await expect(page.getByRole('heading', { name: /GPU Mining/i })).toBeVisible();
      await expect(page.getByRole('heading', { name: /CPU Mining/i })).toBeVisible();
    });
  });

  test.describe('Global Miner Map', () => {
    test('should display miner map section', async ({ page }) => {
      await page.goto('/');
      
      // Verify map section
      await expect(page.getByRole('heading', { name: /Global Miner Network/i })).toBeVisible();
      
      // Verify stats
      await expect(page.getByText('Total Miners')).toBeVisible();
      await expect(page.getByText('Active')).toBeVisible();
      await expect(page.getByText('Countries')).toBeVisible();
    });
  });

  test.describe('Pool Monitoring Section', () => {
    test('should display monitoring dashboard links', async ({ page }) => {
      await page.goto('/');
      
      // Verify monitoring section
      await expect(page.getByRole('heading', { name: /Pool Monitoring/i })).toBeVisible();
      
      // Verify Grafana dashboard links
      await expect(page.getByRole('link', { name: /Pool Overview/i })).toBeVisible();
      await expect(page.getByRole('link', { name: /Workers/i })).toBeVisible();
      await expect(page.getByRole('link', { name: /Payouts/i })).toBeVisible();
      await expect(page.getByRole('link', { name: /Alerts/i })).toBeVisible();
    });

    test('should display node health status', async ({ page }) => {
      await page.goto('/');
      
      // Verify health status indicators
      await expect(page.getByText('Node Health')).toBeVisible();
      await expect(page.getByText('Litecoin Node')).toBeVisible();
      await expect(page.getByText('Stratum Server')).toBeVisible();
    });
  });

  test.describe('Footer', () => {
    test('should display footer with links', async ({ page }) => {
      await page.goto('/');
      
      // Verify footer
      await expect(page.getByRole('contentinfo')).toBeVisible();
      await expect(page.getByRole('link', { name: 'Block Explorer' })).toBeVisible();
      await expect(page.getByRole('link', { name: 'Faucet' })).toBeVisible();
    });
  });
});
