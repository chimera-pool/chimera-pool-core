import React, { useState, useCallback, memo } from 'react';
import { colors } from '../../styles/shared';

// ============================================================================
// AUTH MODAL COMPONENT
// Handles login, registration, forgot password, and password reset flows
// ============================================================================

export type AuthView = 'login' | 'register' | 'forgot-password' | 'reset-password';

export interface AuthModalProps {
  view: AuthView;
  setView: (view: AuthView | null) => void;
  setToken: (token: string) => void;
  showMessage: (type: 'success' | 'error', text: string) => void;
  resetToken: string | null;
}

const styles: { [key: string]: React.CSSProperties } = {
  modalOverlay: {
    position: 'fixed',
    top: 0,
    left: 0,
    right: 0,
    bottom: 0,
    backgroundColor: 'rgba(0, 0, 0, 0.8)',
    display: 'flex',
    justifyContent: 'center',
    alignItems: 'center',
    zIndex: 1000,
  },
  modal: {
    backgroundColor: colors.bgCard,
    padding: '40px',
    borderRadius: '12px',
    border: `1px solid ${colors.border}`,
    width: '100%',
    maxWidth: '400px',
    position: 'relative',
  },
  closeBtn: {
    position: 'absolute',
    top: '15px',
    right: '15px',
    background: 'none',
    border: 'none',
    color: colors.textSecondary,
    fontSize: '24px',
    cursor: 'pointer',
  },
  modalTitle: {
    color: colors.primary,
    marginBottom: '20px',
    textAlign: 'center',
  },
  modalDesc: {
    color: colors.textSecondary,
    fontSize: '0.9rem',
    marginBottom: '20px',
    textAlign: 'center',
  },
  input: {
    width: '100%',
    padding: '12px 16px',
    marginBottom: '15px',
    backgroundColor: colors.bgInput,
    border: `1px solid ${colors.border}`,
    borderRadius: '6px',
    color: colors.textPrimary,
    fontSize: '1rem',
    boxSizing: 'border-box',
  },
  submitBtn: {
    width: '100%',
    padding: '12px',
    backgroundColor: colors.primary,
    border: 'none',
    borderRadius: '6px',
    color: colors.bgDark,
    fontSize: '1rem',
    fontWeight: 'bold',
    cursor: 'pointer',
    marginTop: '10px',
  },
  errorMsg: {
    backgroundColor: '#4d1a1a',
    color: colors.error,
    padding: '10px',
    borderRadius: '6px',
    marginBottom: '15px',
    fontSize: '0.9rem',
    textAlign: 'center',
  },
  authLinks: {
    display: 'flex',
    justifyContent: 'space-between',
    marginTop: '20px',
    flexWrap: 'wrap',
    gap: '10px',
  },
  authLink: {
    color: colors.primary,
    fontSize: '0.9rem',
    cursor: 'pointer',
    textDecoration: 'underline',
  },
  passwordWrapper: {
    position: 'relative',
  },
  passwordToggle: {
    position: 'absolute',
    right: '10px',
    top: '50%',
    transform: 'translateY(-50%)',
    background: 'none',
    border: 'none',
    color: colors.textSecondary,
    cursor: 'pointer',
    fontSize: '14px',
  },
};

// PasswordInput component - memoized to prevent re-renders
interface PasswordInputProps {
  name: string;
  placeholder: string;
  value: string;
  show: boolean;
  onToggle: () => void;
  onChange: (e: React.ChangeEvent<HTMLInputElement>) => void;
  minLength?: number;
}

const PasswordInput = memo(function PasswordInput({
  name,
  placeholder,
  value,
  show,
  onToggle,
  onChange,
  minLength
}: PasswordInputProps) {
  return (
    <div style={styles.passwordWrapper}>
      <input
        style={styles.input}
        type={show ? 'text' : 'password'}
        name={name}
        placeholder={placeholder}
        value={value}
        onChange={onChange}
        minLength={minLength}
        required
      />
      <button
        type="button"
        onClick={onToggle}
        style={styles.passwordToggle}
      >
        {show ? '🙈' : '👁️'}
      </button>
    </div>
  );
});

export function AuthModal({ view, setView, setToken, showMessage, resetToken }: AuthModalProps) {
  const [formData, setFormData] = useState({
    username: '',
    email: '',
    password: '',
    confirmPassword: '',
    newPassword: ''
  });
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [showConfirmPassword, setShowConfirmPassword] = useState(false);

  const handleChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    const { name, value } = e.target;
    setFormData(prev => ({ ...prev, [name]: value }));
    setError('');
  }, []);

  const toggleShowPassword = useCallback(() => {
    setShowPassword(prev => !prev);
  }, []);

  const toggleShowConfirmPassword = useCallback(() => {
    setShowConfirmPassword(prev => !prev);
  }, []);

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError('');
    try {
      const response = await fetch('/api/v1/auth/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email: formData.email, password: formData.password })
      });
      const data = await response.json();
      if (response.ok) {
        localStorage.setItem('token', data.token);
        setToken(data.token);
        setView(null);
        showMessage('success', 'Login successful!');
      } else {
        setError(data.error || 'Login failed');
      }
    } catch (err) {
      setError('Network error. Please try again.');
    } finally {
      setLoading(false);
    }
  };

  const handleRegister = async (e: React.FormEvent) => {
    e.preventDefault();
    if (formData.password !== formData.confirmPassword) {
      setError('Passwords do not match');
      return;
    }
    setLoading(true);
    setError('');
    try {
      const response = await fetch('/api/v1/auth/register', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          username: formData.username,
          email: formData.email,
          password: formData.password
        })
      });
      const data = await response.json();
      if (response.ok) {
        showMessage('success', 'Registration successful! Please login.');
        setView('login');
      } else {
        setError(data.error || 'Registration failed');
      }
    } catch (err) {
      setError('Network error. Please try again.');
    } finally {
      setLoading(false);
    }
  };

  const handleForgotPassword = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError('');
    try {
      const response = await fetch('/api/v1/auth/forgot-password', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email: formData.email })
      });
      const data = await response.json();
      if (response.ok) {
        showMessage('success', data.message);
        setView(null);
      } else {
        setError(data.error || 'Request failed');
      }
    } catch (err) {
      setError('Network error. Please try again.');
    } finally {
      setLoading(false);
    }
  };

  const handleResetPassword = async (e: React.FormEvent) => {
    e.preventDefault();
    if (formData.newPassword !== formData.confirmPassword) {
      setError('Passwords do not match');
      return;
    }
    setLoading(true);
    setError('');
    try {
      const response = await fetch('/api/v1/auth/reset-password', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ token: resetToken, new_password: formData.newPassword })
      });
      const data = await response.json();
      if (response.ok) {
        showMessage('success', 'Password reset successful! Please login.');
        window.history.replaceState({}, document.title, window.location.pathname);
        setView('login');
      } else {
        setError(data.error || 'Reset failed');
      }
    } catch (err) {
      setError('Network error. Please try again.');
    } finally {
      setLoading(false);
    }
  };

  const closeModal = () => {
    if (view === 'reset-password' && resetToken) {
      window.history.replaceState({}, document.title, window.location.pathname);
    }
    setView(null);
  };


  return (
    <div style={styles.modalOverlay as React.CSSProperties} onClick={closeModal}>
      <div style={styles.modal as React.CSSProperties} onClick={(e) => e.stopPropagation()}>
        <button style={styles.closeBtn as React.CSSProperties} onClick={closeModal}>×</button>
        
        {view === 'login' && (
          <form onSubmit={handleLogin}>
            <h2 style={styles.modalTitle as React.CSSProperties}>Login</h2>
            {error && <div style={styles.errorMsg as React.CSSProperties}>{error}</div>}
            <input
              style={styles.input}
              type="email"
              name="email"
              placeholder="Email Address"
              value={formData.email}
              onChange={handleChange}
              required
            />
            <PasswordInput
              name="password"
              placeholder="Password"
              value={formData.password}
              show={showPassword}
              onToggle={toggleShowPassword}
              onChange={handleChange}
            />
            <button style={styles.submitBtn} type="submit" disabled={loading}>
              {loading ? 'Logging in...' : 'Login'}
            </button>
            <div style={styles.authLinks as React.CSSProperties}>
              <span style={styles.authLink} onClick={() => setView('forgot-password')}>
                Forgot Password?
              </span>
              <span style={styles.authLink} onClick={() => setView('register')}>
                Create Account
              </span>
            </div>
          </form>
        )}

        {view === 'register' && (
          <form onSubmit={handleRegister}>
            <h2 style={styles.modalTitle as React.CSSProperties}>Create Account</h2>
            {error && <div style={styles.errorMsg as React.CSSProperties}>{error}</div>}
            <input
              style={styles.input}
              type="text"
              name="username"
              placeholder="Username"
              value={formData.username}
              onChange={handleChange}
              required
            />
            <input
              style={styles.input}
              type="email"
              name="email"
              placeholder="Email"
              value={formData.email}
              onChange={handleChange}
              required
            />
            <PasswordInput
              name="password"
              placeholder="Password (min 8 characters)"
              value={formData.password}
              show={showPassword}
              onToggle={toggleShowPassword}
              onChange={handleChange}
              minLength={8}
            />
            <PasswordInput
              name="confirmPassword"
              placeholder="Confirm Password"
              value={formData.confirmPassword}
              show={showConfirmPassword}
              onToggle={toggleShowConfirmPassword}
              onChange={handleChange}
            />
            <button style={styles.submitBtn} type="submit" disabled={loading}>
              {loading ? 'Creating...' : 'Create Account'}
            </button>
            <div style={styles.authLinks as React.CSSProperties}>
              <span style={styles.authLink} onClick={() => setView('login')}>
                Already have an account? Login
              </span>
            </div>
          </form>
        )}

        {view === 'forgot-password' && (
          <form onSubmit={handleForgotPassword}>
            <h2 style={styles.modalTitle as React.CSSProperties}>Reset Password</h2>
            <p style={styles.modalDesc as React.CSSProperties}>
              Enter your email address and we'll send you a link to reset your password.
            </p>
            {error && <div style={styles.errorMsg as React.CSSProperties}>{error}</div>}
            <input
              style={styles.input}
              type="email"
              name="email"
              placeholder="Email Address"
              value={formData.email}
              onChange={handleChange}
              required
            />
            <button style={styles.submitBtn} type="submit" disabled={loading}>
              {loading ? 'Sending...' : 'Send Reset Link'}
            </button>
            <div style={styles.authLinks as React.CSSProperties}>
              <span style={styles.authLink} onClick={() => setView('login')}>
                Back to Login
              </span>
            </div>
          </form>
        )}

        {view === 'reset-password' && (
          <form onSubmit={handleResetPassword}>
            <h2 style={styles.modalTitle as React.CSSProperties}>Set New Password</h2>
            <p style={styles.modalDesc as React.CSSProperties}>
              Enter your new password below.
            </p>
            {error && <div style={styles.errorMsg as React.CSSProperties}>{error}</div>}
            <input
              style={styles.input}
              type="password"
              name="newPassword"
              placeholder="New Password (min 8 characters)"
              value={formData.newPassword}
              onChange={handleChange}
              minLength={8}
              required
            />
            <input
              style={styles.input}
              type="password"
              name="confirmPassword"
              placeholder="Confirm New Password"
              value={formData.confirmPassword}
              onChange={handleChange}
              required
            />
            <button style={styles.submitBtn} type="submit" disabled={loading}>
              {loading ? 'Resetting...' : 'Reset Password'}
            </button>
          </form>
        )}
      </div>
    </div>
  );
}

export default AuthModal;
