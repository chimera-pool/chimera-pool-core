import React, { createContext, useContext, useState, useEffect, useCallback, ReactNode } from 'react';

// Types
export interface User {
  id?: number;
  email: string;
  username: string;
  payout_address?: string;
  is_admin?: boolean;
  role?: string;
  created_at?: string;
}

export interface AuthContextType {
  user: User | null;
  token: string | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  error: string | null;
  login: (email: string, password: string) => Promise<void>;
  register: (email: string, password: string, username: string) => Promise<void>;
  logout: () => void;
  updateProfile: (data: Partial<User>) => Promise<void>;
  changePassword: (currentPassword: string, newPassword: string) => Promise<void>;
  forgotPassword: (email: string) => Promise<void>;
  resetPassword: (token: string, newPassword: string) => Promise<void>;
  clearError: () => void;
  refreshProfile: () => Promise<void>;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

interface AuthProviderProps {
  children: ReactNode;
}

export const AuthProvider: React.FC<AuthProviderProps> = ({ children }) => {
  const [user, setUser] = useState<User | null>(null);
  const [token, setToken] = useState<string | null>(() => localStorage.getItem('token'));
  const [isLoading, setIsLoading] = useState<boolean>(!!localStorage.getItem('token'));
  const [error, setError] = useState<string | null>(null);

  const isAuthenticated = !!token && !!user;

  // Fetch user profile - supports both cookie and token auth
  const fetchProfile = useCallback(async (authToken?: string) => {
    try {
      const headers: HeadersInit = {};
      // Include Authorization header if token provided (backward compatibility)
      if (authToken) {
        headers['Authorization'] = `Bearer ${authToken}`;
      }
      
      const response = await fetch('/api/v1/user/profile', {
        headers,
        credentials: 'include', // SECURITY: Include HTTP-only cookies
      });
      
      if (response.ok) {
        const data = await response.json();
        setUser(data);
        // If we got a valid response, we're authenticated
        if (!token && authToken) {
          setToken(authToken);
        }
      } else {
        // Token/cookie invalid, clear auth state
        localStorage.removeItem('token');
        setToken(null);
        setUser(null);
      }
    } catch (err) {
      console.error('Failed to fetch profile:', err);
      localStorage.removeItem('token');
      setToken(null);
      setUser(null);
    } finally {
      setIsLoading(false);
    }
  }, [token]);

  // Initialize auth state from localStorage
  useEffect(() => {
    const storedToken = localStorage.getItem('token');
    if (storedToken) {
      fetchProfile(storedToken);
    }
  }, [fetchProfile]);

  const login = useCallback(async (email: string, password: string) => {
    setIsLoading(true);
    setError(null);
    
    try {
      const response = await fetch('/api/v1/auth/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email, password }),
        credentials: 'include', // SECURITY: Accept HTTP-only cookies from server
      });

      const data = await response.json();

      if (response.ok) {
        // Store token in localStorage for backward compatibility
        // Server also sets HTTP-only cookie for enhanced security
        if (data.token) {
          localStorage.setItem('token', data.token);
          setToken(data.token);
        }
        await fetchProfile(data.token);
      } else {
        setError(data.error || 'Login failed');
        throw new Error(data.error || 'Login failed');
      }
    } catch (err) {
      setIsLoading(false);
      if (err instanceof Error && !error) {
        setError(err.message);
      }
      throw err;
    }
  }, [fetchProfile, error]);

  const register = useCallback(async (email: string, password: string, username: string) => {
    setIsLoading(true);
    setError(null);

    try {
      const response = await fetch('/api/v1/auth/register', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email, password, username })
      });

      const data = await response.json();

      if (response.ok) {
        localStorage.setItem('token', data.token);
        setToken(data.token);
        await fetchProfile(data.token);
      } else {
        setError(data.error || 'Registration failed');
        throw new Error(data.error || 'Registration failed');
      }
    } catch (err) {
      setIsLoading(false);
      if (err instanceof Error && !error) {
        setError(err.message);
      }
      throw err;
    }
  }, [fetchProfile, error]);

  const logout = useCallback(async () => {
    try {
      // Call server logout to clear HTTP-only cookie
      await fetch('/api/v1/auth/logout', {
        method: 'POST',
        credentials: 'include', // SECURITY: Include cookie to be cleared
      });
    } catch (err) {
      console.error('Logout request failed:', err);
    } finally {
      // Always clear local state regardless of server response
      localStorage.removeItem('token');
      setToken(null);
      setUser(null);
      setError(null);
    }
  }, []);

  const updateProfile = useCallback(async (data: Partial<User>) => {
    if (!token && !user) {
      throw new Error('Not authenticated');
    }

    try {
      const headers: HeadersInit = { 'Content-Type': 'application/json' };
      if (token) {
        headers['Authorization'] = `Bearer ${token}`;
      }
      
      const response = await fetch('/api/v1/user/profile', {
        method: 'PUT',
        headers,
        body: JSON.stringify(data),
        credentials: 'include', // SECURITY: Include HTTP-only cookies
      });

      const result = await response.json();

      if (response.ok) {
        setUser(prev => prev ? { ...prev, ...result.user } : null);
      } else {
        throw new Error(result.error || 'Failed to update profile');
      }
    } catch (err) {
      if (err instanceof Error) {
        setError(err.message);
      }
      throw err;
    }
  }, [token]);

  const changePassword = useCallback(async (currentPassword: string, newPassword: string) => {
    if (!token && !user) {
      throw new Error('Not authenticated');
    }

    try {
      const headers: HeadersInit = { 'Content-Type': 'application/json' };
      if (token) {
        headers['Authorization'] = `Bearer ${token}`;
      }
      
      const response = await fetch('/api/v1/user/password', {
        method: 'PUT',
        headers,
        body: JSON.stringify({ current_password: currentPassword, new_password: newPassword }),
        credentials: 'include', // SECURITY: Include HTTP-only cookies
      });

      if (!response.ok) {
        const data = await response.json();
        throw new Error(data.error || 'Failed to change password');
      }
    } catch (err) {
      if (err instanceof Error) {
        setError(err.message);
      }
      throw err;
    }
  }, [token]);

  const forgotPassword = useCallback(async (email: string) => {
    try {
      const response = await fetch('/api/v1/auth/forgot-password', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email })
      });

      if (!response.ok) {
        const data = await response.json();
        throw new Error(data.error || 'Failed to send reset email');
      }
    } catch (err) {
      if (err instanceof Error) {
        setError(err.message);
      }
      throw err;
    }
  }, []);

  const resetPassword = useCallback(async (resetToken: string, newPassword: string) => {
    try {
      const response = await fetch('/api/v1/auth/reset-password', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ token: resetToken, new_password: newPassword })
      });

      if (!response.ok) {
        const data = await response.json();
        throw new Error(data.error || 'Failed to reset password');
      }
    } catch (err) {
      if (err instanceof Error) {
        setError(err.message);
      }
      throw err;
    }
  }, []);

  const clearError = useCallback(() => {
    setError(null);
  }, []);

  const refreshProfile = useCallback(async () => {
    if (token) {
      await fetchProfile(token);
    }
  }, [token, fetchProfile]);

  const value: AuthContextType = {
    user,
    token,
    isAuthenticated,
    isLoading,
    error,
    login,
    register,
    logout,
    updateProfile,
    changePassword,
    forgotPassword,
    resetPassword,
    clearError,
    refreshProfile,
  };

  return (
    <AuthContext.Provider value={value}>
      {children}
    </AuthContext.Provider>
  );
};

export const useAuth = (): AuthContextType => {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
};

export default AuthContext;
