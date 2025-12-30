import React from 'react';
import { MainView } from './types';

// ============================================================================
// APP HEADER COMPONENT
// Extracted from App.tsx for modular architecture
// Handles navigation, branding, and auth buttons
// ============================================================================

interface AppHeaderProps {
  mainView: MainView;
  setMainView: (view: MainView) => void;
  token: string | null;
  user: any;
  onLogin: () => void;
  onRegister: () => void;
  onLogout: () => void;
  onOpenProfile: () => void;
  onOpenBugReport: () => void;
  onOpenMyBugs: () => void;
  onOpenAdmin: () => void;
}

const styles = {
  header: {
    background: 'linear-gradient(135deg, #2D1F3D 0%, #3A1F2E 100%)',
    padding: '8px 24px',
    borderBottom: '1px solid rgba(74, 44, 90, 0.5)',
    backdropFilter: 'blur(10px)',
    position: 'sticky' as const,
    top: 0,
    zIndex: 100,
  },
  headerContent: {
    maxWidth: '1400px',
    margin: '0 auto',
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    flexWrap: 'wrap' as const,
    gap: '20px',
  },
  authButtons: {
    display: 'flex',
    gap: '12px',
    alignItems: 'center',
  },
  authBtn: {
    padding: '10px 20px',
    backgroundColor: 'transparent',
    border: '1px solid #7B5EA7',
    color: '#B8B4C8',
    borderRadius: '10px',
    cursor: 'pointer',
    fontSize: '0.9rem',
    fontWeight: 500,
    transition: 'all 0.2s ease',
  },
  registerBtn: {
    background: 'linear-gradient(135deg, #D4A84B 0%, #B8923A 100%)',
    border: 'none',
    color: '#1A0F1E',
    fontWeight: 600,
  },
  userInfo: {
    display: 'flex',
    alignItems: 'center',
    gap: '12px',
    flexWrap: 'wrap' as const,
  },
  username: {
    color: '#D4A84B',
    fontSize: '0.95rem',
    fontWeight: 500,
    cursor: 'pointer',
  },
  logoutBtn: {
    padding: '8px 16px',
    backgroundColor: 'rgba(196, 92, 92, 0.15)',
    border: '1px solid rgba(196, 92, 92, 0.3)',
    color: '#C45C5C',
    borderRadius: '8px',
    cursor: 'pointer',
    fontSize: '0.85rem',
    fontWeight: 500,
    transition: 'all 0.2s ease',
  },
};

export function AppHeader({
  mainView,
  setMainView,
  token,
  user,
  onLogin,
  onRegister,
  onLogout,
  onOpenProfile,
  onOpenBugReport,
  onOpenMyBugs,
  onOpenAdmin,
}: AppHeaderProps) {
  const getNavButtonStyle = (view: MainView) => ({
    padding: '10px 20px',
    background: mainView === view 
      ? 'linear-gradient(135deg, #D4A84B 0%, #B8923A 100%)' 
      : 'transparent',
    border: 'none',
    color: mainView === view ? '#1A0F1E' : '#B8B4C8',
    fontSize: '0.9rem',
    cursor: 'pointer',
    borderRadius: '8px',
    fontWeight: mainView === view ? 600 : 500,
    transition: 'all 0.2s ease',
    boxShadow: mainView === view ? '0 2px 12px rgba(212, 168, 75, 0.3)' : 'none',
  });

  return (
    <header style={styles.header}>
      <div style={styles.headerContent} className="header-content">
        {/* Logo and Brand */}
        <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
          <img 
            src="/logo.png" 
            alt="Chimera Pool" 
            style={{ height: '85px', width: 'auto' }} 
          />
          <span style={{ 
            fontSize: '1.5rem', 
            fontWeight: 700, 
            color: '#D4A84B', 
            letterSpacing: '0.5px' 
          }}>
            Chimera Pool
          </span>
        </div>

        {/* Main Navigation */}
        <nav 
          className="header-nav" 
          style={{ 
            display: 'flex', 
            gap: '4px', 
            backgroundColor: 'rgba(31, 20, 40, 0.8)', 
            borderRadius: '12px', 
            padding: '4px', 
            border: '1px solid #4A2C5A' 
          }}
          data-testid="main-navigation"
        >
          <button
            className={mainView !== 'dashboard' ? 'nav-tab-enhanced' : ''}
            style={getNavButtonStyle('dashboard')}
            onClick={() => setMainView('dashboard')}
            data-testid="nav-dashboard-btn"
          >
            Dashboard
          </button>
          {token && user && (
            <button
              className={mainView !== 'equipment' ? 'nav-tab-enhanced' : ''}
              style={getNavButtonStyle('equipment')}
              onClick={() => setMainView('equipment')}
              data-testid="nav-equipment-btn"
            >
              Equipment
            </button>
          )}
          <button
            className={mainView !== 'community' ? 'nav-tab-enhanced' : ''}
            style={getNavButtonStyle('community')}
            onClick={() => setMainView('community')}
            data-testid="nav-community-btn"
          >
            Community
          </button>
        </nav>

        {/* Auth Buttons */}
        <div style={styles.authButtons} className="auth-buttons">
          {token && user ? (
            <div style={styles.userInfo} className="user-info">
              <span
                style={styles.username}
                className="username-display"
                onClick={onOpenProfile}
                title="Edit Profile"
                data-testid="header-username-btn"
              >
                {user.username}
              </span>
              <button
                style={{
                  ...styles.authBtn,
                  backgroundColor: 'rgba(74, 222, 128, 0.1)',
                  borderColor: 'rgba(74, 222, 128, 0.3)',
                  color: '#4ADE80',
                  fontSize: '0.8rem',
                  padding: '8px 14px',
                }}
                onClick={onOpenBugReport}
                title="Report a Bug"
                data-testid="header-report-bug-btn"
              >
                Report Bug
              </button>
              <button
                style={{
                  ...styles.authBtn,
                  backgroundColor: 'rgba(123, 94, 167, 0.15)',
                  borderColor: '#7B5EA7',
                  color: '#B8B4C8',
                  fontSize: '0.8rem',
                  padding: '8px 14px',
                }}
                onClick={onOpenMyBugs}
                title="My Bug Reports"
                data-testid="header-my-bugs-btn"
              >
                My Bugs
              </button>
              {user.is_admin && (
                <button
                  style={{
                    ...styles.authBtn,
                    background: 'linear-gradient(135deg, #7B5EA7 0%, #5A4580 100%)',
                    border: 'none',
                    color: '#F0EDF4',
                    fontWeight: 600,
                  }}
                  onClick={onOpenAdmin}
                  data-testid="header-admin-btn"
                >
                  Admin
                </button>
              )}
              <button 
                style={styles.logoutBtn} 
                onClick={onLogout}
                data-testid="header-logout-btn"
              >
                Logout
              </button>
            </div>
          ) : (
            <>
              <button 
                style={styles.authBtn} 
                onClick={onLogin}
                data-testid="header-login-btn"
              >
                Login
              </button>
              <button
                style={{ ...styles.authBtn, ...styles.registerBtn }}
                onClick={onRegister}
                data-testid="header-register-btn"
              >
                Register
              </button>
            </>
          )}
        </div>
      </div>
    </header>
  );
}

export default AppHeader;
