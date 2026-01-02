import { test, expect } from '@playwright/test';

/**
 * Authentication Security E2E Tests
 * Tests login, registration, and security features
 * Following TDD and ISP principles
 */

test.describe('Authentication Security', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
  });

  test.describe('Login Form', () => {
    test('should display login modal when clicking login button', async ({ page }) => {
      await page.getByTestId('header-login-btn').click();
      
      await expect(page.getByTestId('auth-modal-container')).toBeVisible();
      await expect(page.getByTestId('login-form')).toBeVisible();
    });

    test('should have accessible login inputs', async ({ page }) => {
      await page.getByTestId('header-login-btn').click();
      
      const emailInput = page.getByTestId('login-email-input');
      const passwordInput = page.getByTestId('login-password-input');
      
      await expect(emailInput).toHaveAttribute('aria-label', 'Email Address');
      await expect(passwordInput).toHaveAttribute('aria-label', 'Password');
      await expect(emailInput).toHaveAttribute('type', 'email');
      await expect(passwordInput).toHaveAttribute('type', 'password');
    });

    test('should have autocomplete attributes for password managers', async ({ page }) => {
      await page.getByTestId('header-login-btn').click();
      
      await expect(page.getByTestId('login-email-input')).toHaveAttribute('autocomplete', 'email');
      await expect(page.getByTestId('login-password-input')).toHaveAttribute('autocomplete', 'current-password');
    });

    test('should close modal when clicking close button', async ({ page }) => {
      await page.getByTestId('header-login-btn').click();
      await expect(page.getByTestId('auth-modal-container')).toBeVisible();
      
      await page.getByTestId('auth-modal-close-btn').click();
      await expect(page.getByTestId('auth-modal-container')).not.toBeVisible();
    });

    test('should close modal when clicking overlay', async ({ page }) => {
      await page.getByTestId('header-login-btn').click();
      await expect(page.getByTestId('auth-modal-container')).toBeVisible();
      
      // Click on overlay (outside modal)
      await page.getByTestId('auth-modal-overlay').click({ position: { x: 10, y: 10 } });
      await expect(page.getByTestId('auth-modal-container')).not.toBeVisible();
    });

    test('should show error message for invalid credentials', async ({ page }) => {
      await page.getByTestId('header-login-btn').click();
      
      await page.getByTestId('login-email-input').fill('invalid@example.com');
      await page.getByTestId('login-password-input').fill('wrongpassword');
      await page.getByTestId('login-submit-btn').click();
      
      // Should show error message
      await expect(page.getByTestId('login-error-message')).toBeVisible({ timeout: 5000 });
    });

    test('should navigate to forgot password form', async ({ page }) => {
      await page.getByTestId('header-login-btn').click();
      await page.getByTestId('login-forgot-password-link').click();
      
      await expect(page.getByTestId('forgot-password-form')).toBeVisible();
      await expect(page.getByTestId('forgot-password-email-input')).toBeVisible();
    });

    test('should navigate to registration form', async ({ page }) => {
      await page.getByTestId('header-login-btn').click();
      await page.getByTestId('login-create-account-link').click();
      
      await expect(page.getByTestId('register-form')).toBeVisible();
    });
  });

  test.describe('Registration Form', () => {
    test.beforeEach(async ({ page }) => {
      await page.getByTestId('header-login-btn').click();
      await page.getByTestId('login-create-account-link').click();
    });

    test('should display all registration fields', async ({ page }) => {
      await expect(page.getByTestId('register-username-input')).toBeVisible();
      await expect(page.getByTestId('register-email-input')).toBeVisible();
      await expect(page.getByTestId('register-password-input')).toBeVisible();
      await expect(page.getByTestId('register-confirm-password-input')).toBeVisible();
    });

    test('should have accessible registration inputs', async ({ page }) => {
      await expect(page.getByTestId('register-username-input')).toHaveAttribute('aria-label', 'Username');
      await expect(page.getByTestId('register-email-input')).toHaveAttribute('aria-label', 'Email');
      await expect(page.getByTestId('register-password-input')).toHaveAttribute('aria-label', 'Password');
      await expect(page.getByTestId('register-confirm-password-input')).toHaveAttribute('aria-label', 'Confirm Password');
    });

    test('should have secure password autocomplete attributes', async ({ page }) => {
      await expect(page.getByTestId('register-password-input')).toHaveAttribute('autocomplete', 'new-password');
      await expect(page.getByTestId('register-confirm-password-input')).toHaveAttribute('autocomplete', 'new-password');
    });

    test('should enforce minimum password length', async ({ page }) => {
      await expect(page.getByTestId('register-password-input')).toHaveAttribute('minLength', '8');
    });

    test('should show error for password mismatch', async ({ page }) => {
      await page.getByTestId('register-username-input').fill('testuser');
      await page.getByTestId('register-email-input').fill('test@example.com');
      await page.getByTestId('register-password-input').fill('Password123!');
      await page.getByTestId('register-confirm-password-input').fill('DifferentPassword123!');
      await page.getByTestId('register-submit-btn').click();
      
      await expect(page.getByTestId('register-error-message')).toBeVisible();
      await expect(page.getByTestId('register-error-message')).toContainText('match');
    });

    test('should navigate back to login form', async ({ page }) => {
      await page.getByTestId('register-login-link').click();
      
      await expect(page.getByTestId('login-form')).toBeVisible();
    });
  });

  test.describe('Forgot Password Form', () => {
    test.beforeEach(async ({ page }) => {
      await page.getByTestId('header-login-btn').click();
      await page.getByTestId('login-forgot-password-link').click();
    });

    test('should display forgot password form', async ({ page }) => {
      await expect(page.getByTestId('forgot-password-form')).toBeVisible();
      await expect(page.getByTestId('forgot-password-email-input')).toBeVisible();
      await expect(page.getByTestId('forgot-password-submit-btn')).toBeVisible();
    });

    test('should have accessible email input', async ({ page }) => {
      await expect(page.getByTestId('forgot-password-email-input')).toHaveAttribute('aria-label', 'Email Address');
      await expect(page.getByTestId('forgot-password-email-input')).toHaveAttribute('type', 'email');
    });

    test('should navigate back to login', async ({ page }) => {
      await page.getByTestId('forgot-password-back-link').click();
      
      await expect(page.getByTestId('login-form')).toBeVisible();
    });
  });

  test.describe('Modal Accessibility', () => {
    test('close button should have aria-label', async ({ page }) => {
      await page.getByTestId('header-login-btn').click();
      
      await expect(page.getByTestId('auth-modal-close-btn')).toHaveAttribute('aria-label', 'Close modal');
    });
  });
});

test.describe('Session Security', () => {
  test('should not persist sensitive data in localStorage for failed login', async ({ page }) => {
    await page.goto('/');
    await page.getByTestId('header-login-btn').click();
    
    await page.getByTestId('login-email-input').fill('invalid@example.com');
    await page.getByTestId('login-password-input').fill('wrongpassword');
    await page.getByTestId('login-submit-btn').click();
    
    // Wait for error
    await expect(page.getByTestId('login-error-message')).toBeVisible({ timeout: 5000 });
    
    // Check localStorage doesn't contain token
    const token = await page.evaluate(() => localStorage.getItem('token'));
    expect(token).toBeNull();
  });
});
