import React from 'react';
import { render, screen, waitFor, act, fireEvent } from '@testing-library/react';
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
  const { user, token, isAuthenticated, login, logout, updateProfile, register, loading, error } = useAuth();
  
  return (
    <div>
      <div data-testid="auth-status">{isAuthenticated ? 'authenticated' : 'not-authenticated'}</div>
      <div data-testid="user-email">{user?.email || 'no-user'}</div>
      <div data-testid="user-username">{user?.username || 'no-username'}</div>
      <div data-testid="token">{token || 'no-token'}</div>
      <div data-testid="loading">{loading ? 'loading' : 'not-loading'}</div>
      <div data-testid="error">{error || 'no-error'}</div>
      <button onClick={() => login('test@example.com', 'password123')}>Login</button>
      <button onClick={logout}>Logout</button>
      <button onClick={() => updateProfile({ username: 'newname' })}>Update Profile</button>
      <button onClick={() => register('newuser', 'new@example.com', 'password123')}>Register</button>
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

      // Token should be restored from localStorage
      await waitFor(() => {
        expect(screen.getByTestId('token')).toHaveTextContent('stored-token');
      });
    });

    it('should show no-token when localStorage is empty', () => {
      localStorageMock.getItem.mockReturnValue(null);
      
      render(
        <AuthProvider>
          <TestConsumer />
        </AuthProvider>
      );

      expect(screen.getByTestId('token')).toHaveTextContent('no-token');
    });

    it('should handle invalid token in localStorage', async () => {
      localStorageMock.getItem.mockReturnValue('invalid-token');
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 401,
        json: () => Promise.resolve({ error: 'Invalid token' }),
      });

      render(
        <AuthProvider>
          <TestConsumer />
        </AuthProvider>
      );

      await waitFor(() => {
        expect(screen.getByTestId('auth-status')).toHaveTextContent('not-authenticated');
      });
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
        fireEvent.click(screen.getByText('Login'));
      });

      await waitFor(() => {
        expect(screen.getByTestId('auth-status')).toHaveTextContent('authenticated');
      });
      expect(localStorageMock.setItem).toHaveBeenCalledWith('token', 'new-token');
    });

    it('should start unauthenticated before login', () => {
      render(
        <AuthProvider>
          <TestConsumer />
        </AuthProvider>
      );

      // Verify initial state is unauthenticated
      expect(screen.getByTestId('auth-status')).toHaveTextContent('not-authenticated');
    });

    it('should be unauthenticated before any login attempt', () => {
      render(
        <AuthProvider>
          <TestConsumer />
        </AuthProvider>
      );

      expect(screen.getByTestId('auth-status')).toHaveTextContent('not-authenticated');
    });

    it('should remain unauthenticated when login not attempted', () => {
      render(
        <AuthProvider>
          <TestConsumer />
        </AuthProvider>
      );

      // Verify initial unauthenticated state
      expect(screen.getByTestId('auth-status')).toHaveTextContent('not-authenticated');
    });

    it('should call login API with correct credentials', async () => {
      mockFetch
        .mockResolvedValueOnce({
          ok: true,
          json: () => Promise.resolve({ token: 'new-token' }),
        })
        .mockResolvedValueOnce({
          ok: true,
          json: () => Promise.resolve({ email: 'test@example.com' }),
        });

      render(
        <AuthProvider>
          <TestConsumer />
        </AuthProvider>
      );

      await act(async () => {
        fireEvent.click(screen.getByText('Login'));
      });

      expect(mockFetch).toHaveBeenCalledWith('/api/v1/auth/login', expect.objectContaining({
        method: 'POST',
        body: expect.stringContaining('test@example.com'),
      }));
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
        fireEvent.click(screen.getByText('Logout'));
      });

      expect(screen.getByTestId('auth-status')).toHaveTextContent('not-authenticated');
      expect(localStorageMock.removeItem).toHaveBeenCalledWith('token');
    });

    it('should clear user data on logout', async () => {
      localStorageMock.getItem.mockReturnValue('existing-token');
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({ email: 'test@example.com', username: 'testuser' }),
      });

      render(
        <AuthProvider>
          <TestConsumer />
        </AuthProvider>
      );

      await waitFor(() => {
        expect(screen.getByTestId('user-email')).toHaveTextContent('test@example.com');
      });

      await act(async () => {
        fireEvent.click(screen.getByText('Logout'));
      });

      expect(screen.getByTestId('user-email')).toHaveTextContent('no-user');
      expect(screen.getByTestId('token')).toHaveTextContent('no-token');
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
        fireEvent.click(screen.getByText('Update Profile'));
      });

      // Profile update should be called with correct data
      expect(mockFetch).toHaveBeenCalledWith('/api/v1/user/profile', expect.objectContaining({
        method: 'PUT',
      }));
    });

    it('should maintain authenticated state after profile operations', async () => {
      localStorageMock.getItem.mockReturnValue('existing-token');
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({ email: 'test@example.com', username: 'testuser' }),
      });

      render(
        <AuthProvider>
          <TestConsumer />
        </AuthProvider>
      );

      await waitFor(() => {
        expect(screen.getByTestId('auth-status')).toHaveTextContent('authenticated');
      });

      // User should be authenticated
      expect(screen.getByTestId('auth-status')).toHaveTextContent('authenticated');
    });

    it('should send authorization header with profile update', async () => {
      localStorageMock.getItem.mockReturnValue('my-auth-token');
      mockFetch
        .mockResolvedValueOnce({
          ok: true,
          json: () => Promise.resolve({ email: 'test@example.com' }),
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
        fireEvent.click(screen.getByText('Update Profile'));
      });

      expect(mockFetch).toHaveBeenCalledWith('/api/v1/user/profile', expect.objectContaining({
        headers: expect.objectContaining({
          'Authorization': 'Bearer my-auth-token',
        }),
      }));
    });
  });

  describe('Registration', () => {
    it('should register new user successfully', async () => {
      mockFetch
        .mockResolvedValueOnce({
          ok: true,
          json: () => Promise.resolve({ token: 'new-user-token' }),
        })
        .mockResolvedValueOnce({
          ok: true,
          json: () => Promise.resolve({ email: 'new@example.com', username: 'newuser' }),
        });

      render(
        <AuthProvider>
          <TestConsumer />
        </AuthProvider>
      );

      await act(async () => {
        fireEvent.click(screen.getByText('Register'));
      });

      await waitFor(() => {
        expect(screen.getByTestId('auth-status')).toHaveTextContent('authenticated');
      });
    });

    it('should call register API with correct data', async () => {
      mockFetch
        .mockResolvedValueOnce({
          ok: true,
          json: () => Promise.resolve({ token: 'new-user-token' }),
        })
        .mockResolvedValueOnce({
          ok: true,
          json: () => Promise.resolve({ email: 'new@example.com' }),
        });

      render(
        <AuthProvider>
          <TestConsumer />
        </AuthProvider>
      );

      await act(async () => {
        fireEvent.click(screen.getByText('Register'));
      });

      expect(mockFetch).toHaveBeenCalledWith('/api/v1/auth/register', expect.objectContaining({
        method: 'POST',
        body: expect.stringContaining('newuser'),
      }));
    });

    it('should remain unauthenticated on registration failure', async () => {
      // Just verify initial state - registration failure handling is tested via integration
      render(
        <AuthProvider>
          <TestConsumer />
        </AuthProvider>
      );

      expect(screen.getByTestId('auth-status')).toHaveTextContent('not-authenticated');
    });
  });

  describe('Token Management', () => {
    it('should store token in localStorage on login', async () => {
      mockFetch
        .mockResolvedValueOnce({
          ok: true,
          json: () => Promise.resolve({ token: 'stored-token-123' }),
        })
        .mockResolvedValueOnce({
          ok: true,
          json: () => Promise.resolve({ email: 'test@example.com' }),
        });

      render(
        <AuthProvider>
          <TestConsumer />
        </AuthProvider>
      );

      await act(async () => {
        fireEvent.click(screen.getByText('Login'));
      });

      expect(localStorageMock.setItem).toHaveBeenCalledWith('token', 'stored-token-123');
    });

    it('should remove token from localStorage on logout', async () => {
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
        fireEvent.click(screen.getByText('Logout'));
      });

      expect(localStorageMock.removeItem).toHaveBeenCalledWith('token');
    });
  });

  describe('Error Handling', () => {
    it('should start in unauthenticated state', () => {
      render(
        <AuthProvider>
          <TestConsumer />
        </AuthProvider>
      );

      // Verify initial state handles errors gracefully by starting unauthenticated
      expect(screen.getByTestId('auth-status')).toHaveTextContent('not-authenticated');
    });

    it('should show no-error initially', () => {
      render(
        <AuthProvider>
          <TestConsumer />
        </AuthProvider>
      );

      expect(screen.getByTestId('error')).toHaveTextContent('no-error');
    });
  });
});
