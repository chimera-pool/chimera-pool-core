import React, { useState, useEffect } from 'react';

interface ProfileModalProps {
  isOpen: boolean;
  onClose: () => void;
  token: string;
  user: any;
  showMessage: (type: 'success' | 'error', text: string) => void;
  onUserUpdate: (user: any) => void;
}

type ProfileTab = 'profile' | 'security';

const styles = {
  overlay: {
    position: 'fixed' as const,
    top: 0,
    left: 0,
    right: 0,
    bottom: 0,
    backgroundColor: 'rgba(13, 8, 17, 0.95)',
    backdropFilter: 'blur(8px)',
    display: 'flex',
    justifyContent: 'center',
    alignItems: 'center',
    zIndex: 2000,
    padding: '20px',
  },
  modal: {
    background: 'linear-gradient(180deg, #2D1F3D 0%, #1A0F1E 100%)',
    borderRadius: '16px',
    padding: '28px',
    width: '100%',
    maxWidth: '500px',
    border: '1px solid rgba(74, 44, 90, 0.4)',
    boxShadow: '0 24px 48px rgba(0, 0, 0, 0.5)',
  },
  header: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: '20px',
  },
  title: {
    color: '#D4A84B',
    margin: 0,
    fontSize: '1.4rem',
    fontWeight: 600,
  },
  closeBtn: {
    background: 'none',
    border: 'none',
    color: '#B8B4C8',
    fontSize: '24px',
    cursor: 'pointer',
  },
  tabs: {
    display: 'flex',
    gap: '10px',
    marginBottom: '20px',
    borderBottom: '1px solid rgba(74, 44, 90, 0.4)',
    paddingBottom: '10px',
  },
  tab: {
    padding: '10px 20px',
    backgroundColor: 'transparent',
    border: 'none',
    color: '#B8B4C8',
    cursor: 'pointer',
    borderRadius: '8px',
    fontSize: '0.95rem',
  },
  tabActive: {
    backgroundColor: 'rgba(212, 168, 75, 0.15)',
    color: '#D4A84B',
  },
  formGroup: {
    marginBottom: '16px',
  },
  label: {
    display: 'block',
    color: '#B8B4C8',
    marginBottom: '8px',
    fontSize: '0.9rem',
    fontWeight: 500,
  },
  input: {
    width: '100%',
    padding: '14px',
    backgroundColor: 'rgba(26, 15, 30, 0.8)',
    border: '1px solid #4A2C5A',
    borderRadius: '10px',
    color: '#F0EDF4',
    fontSize: '1rem',
    boxSizing: 'border-box' as const,
  },
  saveBtn: {
    width: '100%',
    padding: '14px',
    background: 'linear-gradient(135deg, #D4A84B 0%, #B8923A 100%)',
    border: 'none',
    borderRadius: '10px',
    color: '#1A0F1E',
    fontWeight: 600,
    cursor: 'pointer',
    fontSize: '1rem',
    marginTop: '10px',
  },
  divider: {
    height: '1px',
    backgroundColor: 'rgba(74, 44, 90, 0.4)',
    margin: '20px 0',
  },
  forgotLink: {
    backgroundColor: 'transparent',
    border: 'none',
    color: '#7B5EA7',
    cursor: 'pointer',
    fontSize: '0.9rem',
    padding: 0,
    marginTop: '10px',
  },
  successMessage: {
    backgroundColor: 'rgba(74, 222, 128, 0.15)',
    border: '1px solid rgba(74, 222, 128, 0.3)',
    borderRadius: '8px',
    padding: '12px',
    color: '#4ade80',
    marginBottom: '15px',
  },
};

export function ProfileModal({ isOpen, onClose, token, user, showMessage, onUserUpdate }: ProfileModalProps) {
  const [activeTab, setActiveTab] = useState<ProfileTab>('profile');
  const [profileForm, setProfileForm] = useState({ username: '', payout_address: '' });
  const [passwordForm, setPasswordForm] = useState({ current_password: '', new_password: '', confirm_password: '' });
  const [passwordLoading, setPasswordLoading] = useState(false);
  const [forgotPasswordSent, setForgotPasswordSent] = useState(false);

  useEffect(() => {
    if (user && isOpen) {
      setProfileForm({
        username: user.username || '',
        payout_address: user.payout_address || '',
      });
    }
  }, [user, isOpen]);

  const handleSaveProfile = async () => {
    try {
      const response = await fetch('/api/v1/user/profile', {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`,
        },
        body: JSON.stringify({
          username: profileForm.username,
          payout_address: profileForm.payout_address,
        }),
      });

      if (response.ok) {
        const data = await response.json();
        showMessage('success', 'Profile updated successfully');
        onUserUpdate({ ...user, ...data.user });
      } else {
        const data = await response.json();
        showMessage('error', data.error || 'Failed to update profile');
      }
    } catch (error) {
      showMessage('error', 'Network error. Please try again.');
    }
  };

  const handleChangePassword = async () => {
    if (passwordForm.new_password !== passwordForm.confirm_password) {
      showMessage('error', 'New passwords do not match');
      return;
    }
    if (passwordForm.new_password.length < 8) {
      showMessage('error', 'Password must be at least 8 characters');
      return;
    }

    setPasswordLoading(true);
    try {
      const response = await fetch('/api/v1/user/password', {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`,
        },
        body: JSON.stringify({
          current_password: passwordForm.current_password,
          new_password: passwordForm.new_password,
        }),
      });

      if (response.ok) {
        showMessage('success', 'Password changed successfully');
        setPasswordForm({ current_password: '', new_password: '', confirm_password: '' });
      } else {
        const data = await response.json();
        showMessage('error', data.error || 'Failed to change password');
      }
    } catch (error) {
      showMessage('error', 'Network error. Please try again.');
    } finally {
      setPasswordLoading(false);
    }
  };

  const handleForgotPassword = async () => {
    if (!user?.email) {
      showMessage('error', 'No email associated with account');
      return;
    }

    try {
      const response = await fetch('/api/v1/auth/forgot-password', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email: user.email }),
      });

      if (response.ok) {
        setForgotPasswordSent(true);
      } else {
        const data = await response.json();
        showMessage('error', data.error || 'Failed to send reset email');
      }
    } catch (error) {
      showMessage('error', 'Network error. Please try again.');
    }
  };

  if (!isOpen) return null;

  return (
    <div style={styles.overlay} onClick={onClose}>
      <div style={styles.modal} onClick={e => e.stopPropagation()}>
        <div style={styles.header}>
          <h2 style={styles.title}>⚙️ Account Settings</h2>
          <button style={styles.closeBtn} onClick={onClose}>×</button>
        </div>

        <div style={styles.tabs}>
          <button
            style={{ ...styles.tab, ...(activeTab === 'profile' ? styles.tabActive : {}) }}
            onClick={() => setActiveTab('profile')}
          >
            👤 Profile
          </button>
          <button
            style={{ ...styles.tab, ...(activeTab === 'security' ? styles.tabActive : {}) }}
            onClick={() => setActiveTab('security')}
          >
            🔒 Security
          </button>
        </div>

        {activeTab === 'profile' && (
          <>
            <div style={styles.formGroup}>
              <label style={styles.label}>Username</label>
              <input
                style={styles.input}
                value={profileForm.username}
                onChange={e => setProfileForm({ ...profileForm, username: e.target.value })}
                placeholder="Enter username"
              />
            </div>

            <div style={styles.formGroup}>
              <label style={styles.label}>Payout Address</label>
              <input
                style={styles.input}
                value={profileForm.payout_address}
                onChange={e => setProfileForm({ ...profileForm, payout_address: e.target.value })}
                placeholder="Your wallet address"
              />
            </div>

            <button style={styles.saveBtn} onClick={handleSaveProfile}>
              💾 Save Profile
            </button>
          </>
        )}

        {activeTab === 'security' && (
          <>
            <div style={styles.formGroup}>
              <label style={styles.label}>Current Password</label>
              <input
                style={styles.input}
                type="password"
                value={passwordForm.current_password}
                onChange={e => setPasswordForm({ ...passwordForm, current_password: e.target.value })}
                placeholder="Enter current password"
              />
            </div>

            <div style={styles.formGroup}>
              <label style={styles.label}>New Password</label>
              <input
                style={styles.input}
                type="password"
                value={passwordForm.new_password}
                onChange={e => setPasswordForm({ ...passwordForm, new_password: e.target.value })}
                placeholder="Enter new password (min 8 characters)"
              />
            </div>

            <div style={styles.formGroup}>
              <label style={styles.label}>Confirm New Password</label>
              <input
                style={styles.input}
                type="password"
                value={passwordForm.confirm_password}
                onChange={e => setPasswordForm({ ...passwordForm, confirm_password: e.target.value })}
                placeholder="Confirm new password"
              />
            </div>

            <button
              style={{ ...styles.saveBtn, opacity: passwordLoading ? 0.7 : 1 }}
              onClick={handleChangePassword}
              disabled={passwordLoading}
            >
              {passwordLoading ? 'Changing...' : '🔐 Change Password'}
            </button>

            <div style={styles.divider} />

            {forgotPasswordSent ? (
              <div style={styles.successMessage}>
                ✅ Password reset email sent! Check your inbox.
              </div>
            ) : (
              <button style={styles.forgotLink} onClick={handleForgotPassword}>
                Forgot your password? Click here to reset via email
              </button>
            )}
          </>
        )}
      </div>
    </div>
  );
}

export default ProfileModal;
