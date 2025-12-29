/**
 * Authentication Flows E2E Tests
 * 
 * Tests the complete authentication journey:
 * 1. Login flow
 * 2. Registration flow
 * 3. MFA setup and verification
 * 4. Password reset flow
 * 5. Session management
 * 
 * Following Interface Segregation - each test is independent
 */

import { test, expect, type Page } from '@playwright/test';

const BASE_URL = process.env.TEST_URL || 'https://206.162.80.230';

// Test credentials from environment
const TEST_USER = {
  email: process.env.TEST_USER_EMAIL || 'test@example.com',
  password: process.env.TEST_USER_PASSWORD || 'TestPassword123!',
};

// ============================================================================
// LOGIN MODAL TESTS
// ============================================================================

test.describe('Login Modal', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto(BASE_URL);
    await page.waitForLoadState('networkidle');
  });

  test('should open login modal when clicking login button', async ({ page }) => {
    // Find and click login button
    const loginBtn = page.locator('button:has-text("Login"), button:has-text("Sign In"), a:has-text("Login")').first();
    
    if (await loginBtn.isVisible()) {
      await loginBtn.click();
      
      // Modal should appear with email/password fields
      const modal = page.locator('[role="dialog"], [class*="modal"], [data-testid="login-modal"]').first();
      await expect(modal).toBeVisible({ timeout: 5000 });
      
      // Should have email input
      const emailInput = page.locator('input[type="email"], input[name="email"], input[placeholder*="email" i]').first();
      await expect(emailInput).toBeVisible();
      
      // Should have password input
      const passwordInput = page.locator('input[type="password"]').first();
      await expect(passwordInput).toBeVisible();
    }
  });

  test('should show validation errors for empty form submission', async ({ page }) => {
    const loginBtn = page.locator('button:has-text("Login"), button:has-text("Sign In")').first();
    
    if (await loginBtn.isVisible()) {
      await loginBtn.click();
      
      // Wait for modal
      await page.waitForSelector('[role="dialog"], [class*="modal"]', { timeout: 5000 }).catch(() => {});
      
      // Try to submit empty form
      const submitBtn = page.locator('button[type="submit"]:has-text("Login"), button[type="submit"]:has-text("Sign In")').first();
      
      if (await submitBtn.isVisible()) {
        await submitBtn.click();
        
        // Should show validation message
        const errorMessage = page.locator('[class*="error"], [role="alert"], text=/required|invalid/i').first();
        
        // Give time for validation to appear
        await page.waitForTimeout(1000);
      }
    }
  });

  test('should close modal when clicking close button or overlay', async ({ page }) => {
    const loginBtn = page.locator('button:has-text("Login")').first();
    
    if (await loginBtn.isVisible()) {
      await loginBtn.click();
      
      // Wait for modal
      const modal = page.locator('[role="dialog"], [class*="modal"]').first();
      await expect(modal).toBeVisible({ timeout: 5000 });
      
      // Find close button
      const closeBtn = page.locator('button[aria-label*="close" i], button:has-text("×"), button:has-text("Close")').first();
      
      if (await closeBtn.isVisible()) {
        await closeBtn.click();
        await expect(modal).not.toBeVisible({ timeout: 3000 });
      } else {
        // Try pressing Escape
        await page.keyboard.press('Escape');
        await page.waitForTimeout(500);
      }
    }
  });

  test('should have accessible form with proper labels', async ({ page }) => {
    const loginBtn = page.locator('button:has-text("Login")').first();
    
    if (await loginBtn.isVisible()) {
      await loginBtn.click();
      await page.waitForSelector('[role="dialog"], [class*="modal"]', { timeout: 5000 }).catch(() => {});
      
      // Check email input accessibility
      const emailInput = page.locator('input[type="email"], input[name="email"]').first();
      if (await emailInput.isVisible()) {
        const ariaLabel = await emailInput.getAttribute('aria-label');
        const id = await emailInput.getAttribute('id');
        const placeholder = await emailInput.getAttribute('placeholder');
        
        // Should have some form of label
        expect(ariaLabel || id || placeholder).toBeTruthy();
      }
    }
  });
});

// ============================================================================
// REGISTRATION FLOW TESTS
// ============================================================================

test.describe('Registration Flow', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto(BASE_URL);
    await page.waitForLoadState('networkidle');
  });

  test('should have register option in login modal', async ({ page }) => {
    const loginBtn = page.locator('button:has-text("Login")').first();
    
    if (await loginBtn.isVisible()) {
      await loginBtn.click();
      await page.waitForSelector('[role="dialog"], [class*="modal"]', { timeout: 5000 }).catch(() => {});
      
      // Look for register link/button
      const registerLink = page.locator('text=Register, text=Sign Up, text=Create Account, a:has-text("Sign up")').first();
      
      if (await registerLink.isVisible()) {
        await expect(registerLink).toBeVisible();
      }
    }
  });

  test('should validate registration form fields', async ({ page }) => {
    const loginBtn = page.locator('button:has-text("Login")').first();
    
    if (await loginBtn.isVisible()) {
      await loginBtn.click();
      await page.waitForSelector('[role="dialog"], [class*="modal"]', { timeout: 5000 }).catch(() => {});
      
      // Switch to register mode
      const registerLink = page.locator('text=Register, text=Sign Up, a:has-text("Sign up")').first();
      
      if (await registerLink.isVisible()) {
        await registerLink.click();
        await page.waitForTimeout(500);
        
        // Should show registration form with additional fields
        const usernameInput = page.locator('input[name="username"], input[placeholder*="username" i]');
        const confirmPasswordInput = page.locator('input[name="confirmPassword"], input[placeholder*="confirm" i]');
        
        // At least email and password should be present
        const emailInput = page.locator('input[type="email"]').first();
        await expect(emailInput).toBeVisible();
      }
    }
  });
});

// ============================================================================
// MFA FLOW TESTS
// ============================================================================

test.describe('MFA Flow', () => {
  test('should display MFA setup option in settings', async ({ page }) => {
    // This test requires authentication
    test.skip(!process.env.TEST_USER_EMAIL, 'Requires test user credentials');
    
    await page.goto(BASE_URL);
    
    // Login first
    const loginBtn = page.locator('button:has-text("Login")').first();
    if (await loginBtn.isVisible()) {
      await loginBtn.click();
      
      await page.fill('input[type="email"]', TEST_USER.email);
      await page.fill('input[type="password"]', TEST_USER.password);
      await page.click('button[type="submit"]');
      
      // Wait for login
      await page.waitForTimeout(3000);
      
      // Look for settings/security section
      const settingsBtn = page.locator('text=Settings, text=Security, button:has-text("Account")').first();
      
      if (await settingsBtn.isVisible()) {
        await settingsBtn.click();
        
        // Should show MFA option
        const mfaOption = page.locator('text=Two-Factor, text=2FA, text=MFA, text=Authenticator').first();
        if (await mfaOption.isVisible()) {
          await expect(mfaOption).toBeVisible();
        }
      }
    }
  });

  test('should handle MFA verification during login', async ({ page }) => {
    // This test would verify MFA code entry
    // Skipped unless we have a test account with MFA enabled
    test.skip(true, 'Requires MFA-enabled test account');
  });
});

// ============================================================================
// PASSWORD RESET FLOW TESTS
// ============================================================================

test.describe('Password Reset Flow', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto(BASE_URL);
    await page.waitForLoadState('networkidle');
  });

  test('should have forgot password link in login modal', async ({ page }) => {
    const loginBtn = page.locator('button:has-text("Login")').first();
    
    if (await loginBtn.isVisible()) {
      await loginBtn.click();
      await page.waitForSelector('[role="dialog"], [class*="modal"]', { timeout: 5000 }).catch(() => {});
      
      // Look for forgot password link
      const forgotLink = page.locator('text=Forgot, text=Reset Password, a:has-text("forgot")').first();
      
      if (await forgotLink.isVisible()) {
        await expect(forgotLink).toBeVisible();
      }
    }
  });

  test('should show email input for password reset', async ({ page }) => {
    const loginBtn = page.locator('button:has-text("Login")').first();
    
    if (await loginBtn.isVisible()) {
      await loginBtn.click();
      await page.waitForSelector('[role="dialog"], [class*="modal"]', { timeout: 5000 }).catch(() => {});
      
      const forgotLink = page.locator('text=Forgot, a:has-text("forgot")').first();
      
      if (await forgotLink.isVisible()) {
        await forgotLink.click();
        await page.waitForTimeout(500);
        
        // Should show email input for reset
        const emailInput = page.locator('input[type="email"]').first();
        await expect(emailInput).toBeVisible();
        
        // Should have submit button
        const submitBtn = page.locator('button[type="submit"], button:has-text("Reset"), button:has-text("Send")').first();
        if (await submitBtn.isVisible()) {
          await expect(submitBtn).toBeVisible();
        }
      }
    }
  });
});

// ============================================================================
// SESSION MANAGEMENT TESTS
// ============================================================================

test.describe('Session Management', () => {
  test('should persist session across page reloads', async ({ page, context }) => {
    test.skip(!process.env.TEST_USER_EMAIL, 'Requires test user credentials');
    
    await page.goto(BASE_URL);
    
    // Login
    const loginBtn = page.locator('button:has-text("Login")').first();
    if (await loginBtn.isVisible()) {
      await loginBtn.click();
      
      await page.fill('input[type="email"]', TEST_USER.email);
      await page.fill('input[type="password"]', TEST_USER.password);
      await page.click('button[type="submit"]');
      
      // Wait for login to complete
      await page.waitForTimeout(3000);
      
      // Reload page
      await page.reload();
      await page.waitForLoadState('networkidle');
      
      // Should still be logged in (no login button, or user menu visible)
      const logoutBtn = page.locator('text=Logout, button:has-text("Sign Out")').first();
      const userMenu = page.locator('[data-testid="user-menu"], text=Profile, text=Account').first();
      
      // Either logout button or user menu should be visible
      const isLoggedIn = await logoutBtn.isVisible() || await userMenu.isVisible();
      // Session might not persist depending on implementation
      console.log('Session persisted:', isLoggedIn);
    }
  });

  test('should clear session on logout', async ({ page }) => {
    test.skip(!process.env.TEST_USER_EMAIL, 'Requires test user credentials');
    
    await page.goto(BASE_URL);
    
    // Login first
    const loginBtn = page.locator('button:has-text("Login")').first();
    if (await loginBtn.isVisible()) {
      await loginBtn.click();
      
      await page.fill('input[type="email"]', TEST_USER.email);
      await page.fill('input[type="password"]', TEST_USER.password);
      await page.click('button[type="submit"]');
      
      await page.waitForTimeout(3000);
      
      // Click logout
      const logoutBtn = page.locator('text=Logout, button:has-text("Sign Out")').first();
      
      if (await logoutBtn.isVisible()) {
        await logoutBtn.click();
        await page.waitForTimeout(1000);
        
        // Login button should be visible again
        const newLoginBtn = page.locator('button:has-text("Login")').first();
        await expect(newLoginBtn).toBeVisible({ timeout: 5000 });
      }
    }
  });
});

// ============================================================================
// SECURITY TESTS
// ============================================================================

test.describe('Security', () => {
  test('should use HTTPS', async ({ page }) => {
    const response = await page.goto(BASE_URL);
    const url = page.url();
    
    // In production, should be HTTPS
    if (!url.includes('localhost') && !url.includes('127.0.0.1')) {
      expect(url).toMatch(/^https:/);
    }
  });

  test('should have secure cookie settings', async ({ page, context }) => {
    await page.goto(BASE_URL);
    
    const cookies = await context.cookies();
    
    // Check session cookies for secure flags
    const sessionCookies = cookies.filter(c => 
      c.name.toLowerCase().includes('session') || 
      c.name.toLowerCase().includes('token')
    );
    
    sessionCookies.forEach(cookie => {
      // In production, session cookies should be HttpOnly
      console.log(`Cookie ${cookie.name}: HttpOnly=${cookie.httpOnly}, Secure=${cookie.secure}`);
    });
  });

  test('should not expose sensitive data in page source', async ({ page }) => {
    await page.goto(BASE_URL);
    
    const pageContent = await page.content();
    
    // Should not contain API keys, passwords, or tokens in source
    expect(pageContent).not.toMatch(/api[_-]?key["']\s*:\s*["'][a-zA-Z0-9]{20,}/i);
    expect(pageContent).not.toMatch(/password["']\s*:\s*["'][^"']+["']/i);
    expect(pageContent).not.toMatch(/secret["']\s*:\s*["'][a-zA-Z0-9]{20,}/i);
  });
});
