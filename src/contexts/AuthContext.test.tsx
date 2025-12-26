import React from 'react';
import { render, screen, waitFor, act } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { AuthProvider, useAuth } from './AuthContext';

// Mock fetch globally
const mockFetch = jest.fn();
global.fetch = mockFetch;

// Mock localStorage
const localStorageMock = {
  getItem: jest.fn(),
  setItem: jest.fn(),
  removeItem: jest.fn(),
  clear: jest.fn(),
};
Object.defineProperty(window, 'localStorage', { value: localStorageMock });

// Test component that uses the auth context
const TestConsumer: React.FC = () => {
  const { user, token, isAuthenticated, login, logout, updateProfile } = useAuth();
  
  return (
    <div>
      <div data-testid="auth-status">{isAuthenticated ? 'authenticated' : 'not-authenticated'}</div>
      <div data-testid="user-email">{user?.email || 'no-user'}</div>
      <div data-testid="token">{token || 'no-token'}</div>
      <button onClick={() => login('test@example.com', 'password123')}>Login</button>
      <button onClick={logout}>Logout</button>
      <button onClick={() => updateProfile({ username: 'newname' })}>Update Profile</button>
    </div>
  );
};

describe('AuthContext', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    localStorageMock.getItem.mockReturnValue(null);
  });

  describe('Initial State', () => {
    it('should start unauthenticated when no token in localStorage', () => {
      render(
        <AuthProvider>
          <TestConsumer />
        </AuthProvider>
      );

      expect(screen.getByTestId('auth-status')).toHaveTextContent('not-authenticated');
      expect(screen.getByTestId('user-email')).toHaveTextContent('no-user');
    });

    it('should restore token from localStorage and fetch profile', async () => {
      localStorageMock.getItem.mockReturnValue('stored-token');
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({ email: 'stored@example.com', username: 'storeduser' }),
      });

      render(
        <AuthProvider>
          <TestConsumer />
        </AuthProvider>
      );

      await waitFor(() => {
        expect(screen.getByTestId('auth-status')).toHaveTextContent('authenticated');
      });
      expect(screen.getByTestId('user-email')).toHaveTextContent('stored@example.com');
    });
  });

  describe('Login', () => {
    it('should authenticate user on successful login', async () => {
      mockFetch
        .mockResolvedValueOnce({
          ok: true,
          json: () => Promise.resolve({ token: 'new-token' }),
        })
        .mockResolvedValueOnce({
          ok: true,
          json: () => Promise.resolve({ email: 'test@example.com', username: 'testuser' }),
        });

      render(
        <AuthProvider>
          <TestConsumer />
        </AuthProvider>
      );

      await act(async () => {
        await userEvent.click(screen.getByText('Login'));
      });

      await waitFor(() => {
        expect(screen.getByTestId('auth-status')).toHaveTextContent('authenticated');
      });
      expect(localStorageMock.setItem).toHaveBeenCalledWith('token', 'new-token');
    });

    it('should throw error on failed login', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        json: () => Promise.resolve({ error: 'Invalid credentials' }),
      });

      const consoleSpy = jest.spyOn(console, 'error').mockImplementation();

      render(
        <AuthProvider>
          <TestConsumer />
        </AuthProvider>
      );

      await act(async () => {
        await userEvent.click(screen.getByText('Login'));
      });

      expect(screen.getByTestId('auth-status')).toHaveTextContent('not-authenticated');
      consoleSpy.mockRestore();
    });
  });

  describe('Logout', () => {
    it('should clear auth state on logout', async () => {
      localStorageMock.getItem.mockReturnValue('existing-token');
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({ email: 'test@example.com' }),
      });

      render(
        <AuthProvider>
          <TestConsumer />
        </AuthProvider>
      );

      await waitFor(() => {
        expect(screen.getByTestId('auth-status')).toHaveTextContent('authenticated');
      });

      await act(async () => {
        await userEvent.click(screen.getByText('Logout'));
      });

      expect(screen.getByTestId('auth-status')).toHaveTextContent('not-authenticated');
      expect(localStorageMock.removeItem).toHaveBeenCalledWith('token');
    });
  });

  describe('Update Profile', () => {
    it('should update user data on successful profile update', async () => {
      localStorageMock.getItem.mockReturnValue('existing-token');
      mockFetch
        .mockResolvedValueOnce({
          ok: true,
          json: () => Promise.resolve({ email: 'test@example.com', username: 'oldname' }),
        })
        .mockResolvedValueOnce({
          ok: true,
          json: () => Promise.resolve({ user: { email: 'test@example.com', username: 'newname' } }),
        });

      render(
        <AuthProvider>
          <TestConsumer />
        </AuthProvider>
      );

      await waitFor(() => {
        expect(screen.getByTestId('auth-status')).toHaveTextContent('authenticated');
      });

      await act(async () => {
        await userEvent.click(screen.getByText('Update Profile'));
      });

      // Profile update should be called with correct data
      expect(mockFetch).toHaveBeenCalledWith('/api/v1/user/profile', expect.objectContaining({
        method: 'PUT',
      }));
    });
  });
});
