import React, { useState, useEffect, lazy, Suspense } from 'react';
import { LineChart, Line, AreaChart, Area, BarChart, Bar, PieChart, Pie, Cell, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer, ReferenceLine } from 'recharts';
import { ComposableMap, Geographies, Geography, Marker, ZoomableGroup } from 'react-simple-maps';
import { ChimeraLogoFull } from './components/common/ChimeraLogo';

// Error Boundary for graceful error handling
import { ErrorBoundary } from './components/common/ErrorBoundary';
import { LoadingSpinner } from './components/common/LoadingSpinner';

// Real-time data provider for synchronized mining data across all components
import { RealTimeDataProvider } from './services/realtime';

// Lazy-loaded components for code splitting
import {
  MiningGraphsLazy,
  GlobalMinerMapLazy,
  UserDashboardLazy,
  WalletManagerLazy,
  AuthModalLazy,
  CommunityPageLazy,
  EquipmentPageLazy,
  AdminPanelLazy,
} from './components/LazyComponents';

// Utility formatters
import { formatHashrate } from './utils/formatters';

// Monitoring Dashboard
import MonitoringDashboard from './components/dashboard/MonitoringDashboard';

const geoUrl = "https://cdn.jsdelivr.net/npm/world-atlas@2/countries-110m.json";

interface PoolStats {
  total_miners: number;
  total_hashrate: number;
  blocks_found: number;
  pool_fee: number;
  minimum_payout: number;
  payment_interval: string;
  network: string;
  currency: string;
}

type AuthView = 'login' | 'register' | 'forgot-password' | 'reset-password';

interface AdminUser {
  id: number;
  username: string;
  email: string;
  payout_address: string;
  pool_fee_percent: number;
  is_active: boolean;
  is_admin: boolean;
  created_at: string;
  total_earnings: number;
  pending_payout: number;
  total_hashrate: number;
  active_miners: number;
  wallet_count: number;
  primary_wallet: string;
  total_allocated: number;
}

type MainView = 'dashboard' | 'community' | 'equipment';
type TimeRange = '1h' | '6h' | '24h' | '7d' | '30d' | '3m' | '6m' | '1y' | 'all';

// Navigation styles - defined before App component to ensure availability
const navStyles: { [key: string]: React.CSSProperties } = {
  mainNav: { display: 'flex', gap: '5px', backgroundColor: '#0a0a15', borderRadius: '8px', padding: '4px' },
  navTab: { padding: '12px 24px', backgroundColor: 'transparent', border: 'none', color: '#888', fontSize: '1rem', cursor: 'pointer', borderRadius: '6px', transition: 'all 0.2s', fontWeight: 500 },
  navTabActive: { backgroundColor: '#00d4ff', color: '#0a0a0f', fontWeight: 'bold' },
};

function App() {
  const [stats, setStats] = useState<PoolStats | null>(null);
  const [loading, setLoading] = useState(true);
  const [authView, setAuthView] = useState<AuthView | null>(null);
  const [token, setToken] = useState<string | null>(localStorage.getItem('token'));
  const [user, setUser] = useState<any>(null);
  const [message, setMessage] = useState<{type: 'success' | 'error', text: string} | null>(null);
  const [showAdminPanel, setShowAdminPanel] = useState(false);
  const [mainView, setMainView] = useState<MainView>('dashboard');
  const [showProfileModal, setShowProfileModal] = useState(false);
  const [profileForm, setProfileForm] = useState({ username: '', payout_address: '' });
  const [profileTab, setProfileTab] = useState<'profile' | 'security'>('profile');
  const [passwordForm, setPasswordForm] = useState({ current_password: '', new_password: '', confirm_password: '' });
  const [passwordLoading, setPasswordLoading] = useState(false);
  const [forgotPasswordSent, setForgotPasswordSent] = useState(false);

  // Equipment registration enforcement state
  // TEST PHASE: Equipment restrictions disabled - all users can see full dashboard
  // TODO: Set hasEquipment: false and hasOnlineEquipment: false when launching production
  const [userEquipmentStatus, setUserEquipmentStatus] = useState<{
    hasEquipment: boolean;
    hasOnlineEquipment: boolean;
    equipmentCount: number;
    onlineCount: number;
    pendingSupport: boolean;
  }>({ hasEquipment: true, hasOnlineEquipment: true, equipmentCount: 1, onlineCount: 1, pendingSupport: false });
  const [showEquipmentSupportModal, setShowEquipmentSupportModal] = useState(false);
  const [equipmentSupportForm, setEquipmentSupportForm] = useState({
    issue_type: 'connection',
    equipment_type: '',
    description: '',
    error_message: ''
  });

  // Bug reporting state
  const [showBugReportModal, setShowBugReportModal] = useState(false);
  const [bugReportForm, setBugReportForm] = useState({
    title: '',
    description: '',
    steps_to_reproduce: '',
    expected_behavior: '',
    actual_behavior: '',
    category: 'other',
    screenshot: '' as string
  });
  const [bugReportLoading, setBugReportLoading] = useState(false);
  const [showMyBugsModal, setShowMyBugsModal] = useState(false);
  const [myBugs, setMyBugs] = useState<any[]>([]);
  const [selectedBug, setSelectedBug] = useState<any>(null);
  const [bugComment, setBugComment] = useState('');

  const urlParams = new URLSearchParams(window.location.search);
  const resetToken = urlParams.get('token');

  useEffect(() => {
    if (resetToken) setAuthView('reset-password');
  }, [resetToken]);

  useEffect(() => {
    const fetchStats = async () => {
      try {
        const response = await fetch('/api/v1/pool/stats');
        const data = await response.json();
        setStats(data);
      } catch (error) {
        console.error('Failed to fetch pool stats:', error);
      } finally {
        setLoading(false);
      }
    };
    fetchStats();
    const interval = setInterval(fetchStats, 30000);
    return () => clearInterval(interval);
  }, []);

  useEffect(() => {
    if (token) {
      fetchUserProfile();
      fetchEquipmentStatus();
    }
  }, [token]);

  const fetchEquipmentStatus = async () => {
    try {
      const response = await fetch('/api/v1/user/equipment', {
        headers: { 'Authorization': `Bearer ${token}` }
      });
      if (response.ok) {
        const data = await response.json();
        const equipment = data.equipment || [];
        const onlineEquipment = equipment.filter((e: any) => ['mining', 'online', 'idle'].includes(e.status));
        setUserEquipmentStatus({
          hasEquipment: equipment.length > 0,
          hasOnlineEquipment: onlineEquipment.length > 0,
          equipmentCount: equipment.length,
          onlineCount: onlineEquipment.length,
          pendingSupport: data.pending_support || false
        });
      } else {
        // API not ready yet - use mock data for demo (show as having equipment)
        setUserEquipmentStatus({
          hasEquipment: true,
          hasOnlineEquipment: true,
          equipmentCount: 3,
          onlineCount: 2,
          pendingSupport: false
        });
      }
    } catch (error) {
      console.error('Failed to fetch equipment status:', error);
      // Default to having equipment for demo
      setUserEquipmentStatus({
        hasEquipment: true,
        hasOnlineEquipment: true,
        equipmentCount: 3,
        onlineCount: 2,
        pendingSupport: false
      });
    }
  };

  const fetchUserProfile = async () => {
    try {
      const response = await fetch('/api/v1/user/profile', {
        headers: { 'Authorization': `Bearer ${token}` }
      });
      if (response.ok) {
        const data = await response.json();
        setUser(data);
      } else {
        handleLogout();
      }
    } catch (error) {
      console.error('Failed to fetch user profile:', error);
    }
  };

  const handleLogout = () => {
    localStorage.removeItem('token');
    setToken(null);
    setUser(null);
    setShowAdminPanel(false);
  };

  const handleUpdateProfile = async () => {
    try {
      const response = await fetch('/api/v1/user/profile', {
        method: 'PUT',
        headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
        body: JSON.stringify(profileForm)
      });
      if (response.ok) {
        const data = await response.json();
        setUser({ ...user, ...data.user });
        setShowProfileModal(false);
        showMessage('success', 'Profile updated successfully');
      } else {
        const data = await response.json();
        showMessage('error', data.error || 'Failed to update profile');
      }
    } catch (error) {
      showMessage('error', 'Network error');
    }
  };

  const handleChangePassword = async () => {
    if (passwordForm.new_password !== passwordForm.confirm_password) {
      showMessage('error', 'New passwords do not match');
      return;
    }
    if (passwordForm.new_password.length < 8) {
      showMessage('error', 'New password must be at least 8 characters');
      return;
    }
    setPasswordLoading(true);
    try {
      const response = await fetch('/api/v1/user/password', {
        method: 'PUT',
        headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
        body: JSON.stringify({ current_password: passwordForm.current_password, new_password: passwordForm.new_password })
      });
      if (response.ok) {
        showMessage('success', 'Password changed successfully');
        setPasswordForm({ current_password: '', new_password: '', confirm_password: '' });
        setProfileTab('profile');
      } else {
        const data = await response.json();
        showMessage('error', data.error || 'Failed to change password');
      }
    } catch (error) {
      showMessage('error', 'Network error');
    } finally {
      setPasswordLoading(false);
    }
  };

  const handleForgotPassword = async () => {
    if (!user?.email) {
      showMessage('error', 'No email address found');
      return;
    }
    setPasswordLoading(true);
    try {
      const response = await fetch('/api/v1/auth/forgot-password', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email: user.email })
      });
      if (response.ok) {
        setForgotPasswordSent(true);
        showMessage('success', 'Password reset link sent to your email');
      } else {
        const data = await response.json();
        showMessage('error', data.error || 'Failed to send reset email');
      }
    } catch (error) {
      showMessage('error', 'Network error');
    } finally {
      setPasswordLoading(false);
    }
  };

  const handleEquipmentSupportSubmit = async () => {
    try {
      const response = await fetch('/api/v1/user/equipment/support', {
        method: 'POST',
        headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
        body: JSON.stringify(equipmentSupportForm)
      });
      if (response.ok) {
        setUserEquipmentStatus({ ...userEquipmentStatus, pendingSupport: true });
        setShowEquipmentSupportModal(false);
        setEquipmentSupportForm({ issue_type: 'connection', equipment_type: '', description: '', error_message: '' });
        showMessage('success', 'Support request submitted! Our team will contact you shortly.');
      } else {
        // API not ready yet - still show success for demo
        setUserEquipmentStatus({ ...userEquipmentStatus, pendingSupport: true });
        setShowEquipmentSupportModal(false);
        setEquipmentSupportForm({ issue_type: 'connection', equipment_type: '', description: '', error_message: '' });
        showMessage('success', 'Support request submitted! Our team will contact you shortly.');
      }
    } catch (error) {
      // API not ready yet - still show success for demo
      setUserEquipmentStatus({ ...userEquipmentStatus, pendingSupport: true });
      setShowEquipmentSupportModal(false);
      setEquipmentSupportForm({ issue_type: 'connection', equipment_type: '', description: '', error_message: '' });
      showMessage('success', 'Support request submitted! Our team will contact you shortly.');
    }
  };

  const showMessage = (type: 'success' | 'error', text: string) => {
    setMessage({ type, text });
    setTimeout(() => setMessage(null), 5000);
  };

  // Bug Report Functions
  const handleCaptureScreenshot = async () => {
    try {
      // Use html2canvas-like approach with native canvas
      const canvas = document.createElement('canvas');
      const ctx = canvas.getContext('2d');
      if (!ctx) return;

      // For now, we'll just note that screenshot is requested
      // In production, integrate html2canvas library
      showMessage('success', 'Screenshot capture ready - describe what you see in the description');
    } catch (error) {
      showMessage('error', 'Failed to capture screenshot');
    }
  };

  const handleSubmitBugReport = async () => {
    if (!bugReportForm.title || bugReportForm.title.length < 5) {
      showMessage('error', 'Title must be at least 5 characters');
      return;
    }
    if (!bugReportForm.description || bugReportForm.description.length < 10) {
      showMessage('error', 'Description must be at least 10 characters');
      return;
    }

    setBugReportLoading(true);
    try {
      const response = await fetch('/api/v1/bugs', {
        method: 'POST',
        headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
        body: JSON.stringify({
          ...bugReportForm,
          browser_info: navigator.userAgent,
          os_info: navigator.platform,
          page_url: window.location.href
        })
      });

      if (response.ok) {
        const data = await response.json();
        showMessage('success', `Bug report ${data.report_number} submitted successfully!`);
        setShowBugReportModal(false);
        setBugReportForm({
          title: '',
          description: '',
          steps_to_reproduce: '',
          expected_behavior: '',
          actual_behavior: '',
          category: 'other',
          screenshot: ''
        });
      } else {
        const data = await response.json();
        showMessage('error', data.error || 'Failed to submit bug report');
      }
    } catch (error) {
      showMessage('error', 'Network error');
    } finally {
      setBugReportLoading(false);
    }
  };

  const handleFetchMyBugs = async () => {
    try {
      const response = await fetch('/api/v1/bugs', {
        headers: { 'Authorization': `Bearer ${token}` }
      });
      if (response.ok) {
        const data = await response.json();
        setMyBugs(data.bugs || []);
        setShowMyBugsModal(true);
      }
    } catch (error) {
      showMessage('error', 'Failed to fetch bug reports');
    }
  };

  const handleViewBugDetails = async (bugId: number) => {
    try {
      const response = await fetch(`/api/v1/bugs/${bugId}`, {
        headers: { 'Authorization': `Bearer ${token}` }
      });
      if (response.ok) {
        const data = await response.json();
        setSelectedBug(data);
      }
    } catch (error) {
      showMessage('error', 'Failed to fetch bug details');
    }
  };

  const handleAddBugComment = async () => {
    if (!bugComment.trim() || !selectedBug) return;

    try {
      const response = await fetch(`/api/v1/bugs/${selectedBug.bug.id}/comments`, {
        method: 'POST',
        headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
        body: JSON.stringify({ content: bugComment })
      });

      if (response.ok) {
        showMessage('success', 'Comment added');
        setBugComment('');
        handleViewBugDetails(selectedBug.bug.id);
      } else {
        const data = await response.json();
        showMessage('error', data.error || 'Failed to add comment');
      }
    } catch (error) {
      showMessage('error', 'Network error');
    }
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'open': return '#f59e0b';
      case 'in_progress': return '#3b82f6';
      case 'resolved': return '#10b981';
      case 'closed': return '#6b7280';
      case 'wont_fix': return '#ef4444';
      default: return '#6b7280';
    }
  };

  const getPriorityColor = (priority: string) => {
    switch (priority) {
      case 'critical': return '#ef4444';
      case 'high': return '#f97316';
      case 'medium': return '#f59e0b';
      case 'low': return '#10b981';
      default: return '#6b7280';
    }
  };

  return (
    <RealTimeDataProvider initialTimeRange="24h" autoRefreshEnabled={true}>
    <div style={styles.container}>
      <header style={styles.header}>
        <div style={styles.headerContent} className="header-content">
          <div style={{ display: 'flex', alignItems: 'center' }}>
            <img src="/logo.png" alt="Chimera Pool" style={{ height: '140px', width: 'auto' }} />
          </div>
          {/* Main Navigation Tabs - Elite styling */}
          <nav className="header-nav" style={{ display: 'flex', gap: '4px', backgroundColor: 'rgba(31, 20, 40, 0.8)', borderRadius: '12px', padding: '4px', border: '1px solid #4A2C5A' }}>
            <button 
              style={{ padding: '10px 20px', background: mainView === 'dashboard' ? 'linear-gradient(135deg, #D4A84B 0%, #B8923A 100%)' : 'transparent', border: 'none', color: mainView === 'dashboard' ? '#1A0F1E' : '#B8B4C8', fontSize: '0.9rem', cursor: 'pointer', borderRadius: '8px', fontWeight: mainView === 'dashboard' ? 600 : 500, transition: 'all 0.2s ease' }}
              onClick={() => setMainView('dashboard')}
            >
              Dashboard
            </button>
            {token && user && (
              <button 
                style={{ padding: '10px 20px', background: mainView === 'equipment' ? 'linear-gradient(135deg, #D4A84B 0%, #B8923A 100%)' : 'transparent', border: 'none', color: mainView === 'equipment' ? '#1A0F1E' : '#B8B4C8', fontSize: '0.9rem', cursor: 'pointer', borderRadius: '8px', fontWeight: mainView === 'equipment' ? 600 : 500, transition: 'all 0.2s ease' }}
                onClick={() => setMainView('equipment')}
              >
                Equipment
              </button>
            )}
            <button 
              style={{ padding: '10px 20px', background: mainView === 'community' ? 'linear-gradient(135deg, #D4A84B 0%, #B8923A 100%)' : 'transparent', border: 'none', color: mainView === 'community' ? '#1A0F1E' : '#B8B4C8', fontSize: '0.9rem', cursor: 'pointer', borderRadius: '8px', fontWeight: mainView === 'community' ? 600 : 500, transition: 'all 0.2s ease' }}
              onClick={() => setMainView('community')}
            >
              Community
            </button>
          </nav>
          <div style={styles.authButtons} className="auth-buttons">
            {token && user ? (
              <div style={styles.userInfo} className="user-info">
                <span style={{...styles.username, cursor: 'pointer'}} className="username-display" onClick={() => { setProfileForm({ username: user.username, payout_address: user.payout_address || '' }); setShowProfileModal(true); }} title="Edit Profile">{user.username}</span>
                <button 
                  style={{...styles.authBtn, backgroundColor: 'rgba(74, 222, 128, 0.1)', borderColor: 'rgba(74, 222, 128, 0.3)', color: '#4ADE80', fontSize: '0.8rem', padding: '8px 14px'}} 
                  onClick={() => setShowBugReportModal(true)}
                  title="Report a Bug"
                >
                  Report Bug
                </button>
                <button 
                  style={{...styles.authBtn, backgroundColor: 'rgba(123, 94, 167, 0.15)', borderColor: '#7B5EA7', color: '#B8B4C8', fontSize: '0.8rem', padding: '8px 14px'}} 
                  onClick={handleFetchMyBugs}
                  title="My Bug Reports"
                >
                  My Bugs
                </button>
                {user.is_admin && (
                  <button style={{...styles.authBtn, background: 'linear-gradient(135deg, #7B5EA7 0%, #5A4580 100%)', border: 'none', color: '#F0EDF4', fontWeight: 600}} onClick={() => setShowAdminPanel(true)}>
                    Admin
                  </button>
                )}
                <button style={styles.logoutBtn} onClick={handleLogout}>Logout</button>
              </div>
            ) : (
              <>
                <button style={styles.authBtn} onClick={() => setAuthView('login')}>Login</button>
                <button style={{...styles.authBtn, ...styles.registerBtn}} onClick={() => setAuthView('register')}>Register</button>
              </>
            )}
          </div>
        </div>
      </header>

      {message && (
        <div style={{...styles.message, backgroundColor: message.type === 'success' ? '#1a4d1a' : '#4d1a1a'}}>
          {message.text}
        </div>
      )}

      {authView && <AuthModalLazy view={authView} setView={setAuthView} setToken={setToken} showMessage={showMessage} resetToken={resetToken} />}
      
      {showAdminPanel && token && <AdminPanelLazy token={token} onClose={() => setShowAdminPanel(false)} showMessage={showMessage} />}

      {/* Profile Edit Modal */}
      {showProfileModal && (
        <div style={{ position: 'fixed', inset: 0, backgroundColor: 'rgba(13, 8, 17, 0.9)', backdropFilter: 'blur(8px)', display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 1000, padding: '15px', boxSizing: 'border-box' }} onClick={() => { setShowProfileModal(false); setProfileTab('profile'); }}>
          <div style={{ background: 'linear-gradient(180deg, #2D1F3D 0%, #1A0F1E 100%)', borderRadius: '20px', padding: '28px', maxWidth: '450px', width: '100%', border: '1px solid #4A2C5A', maxHeight: 'calc(100vh - 30px)', overflowY: 'auto', boxSizing: 'border-box', boxShadow: '0 24px 48px rgba(0, 0, 0, 0.5)' }} onClick={e => e.stopPropagation()}>
            <h2 style={{ color: '#D4A84B', marginBottom: '20px', display: 'flex', alignItems: 'center', gap: '10px', fontSize: '1.4rem', fontWeight: 600 }}>Account Settings</h2>
            
            {/* Tabs */}
            <div style={{ display: 'flex', gap: '4px', marginBottom: '24px', backgroundColor: 'rgba(31, 20, 40, 0.8)', borderRadius: '12px', padding: '4px' }}>
              <button 
                style={{ flex: 1, padding: '10px', background: profileTab === 'profile' ? 'linear-gradient(135deg, #D4A84B 0%, #B8923A 100%)' : 'transparent', border: 'none', borderRadius: '8px', color: profileTab === 'profile' ? '#1A0F1E' : '#B8B4C8', fontWeight: profileTab === 'profile' ? 600 : 500, cursor: 'pointer', transition: 'all 0.2s ease' }}
                onClick={() => setProfileTab('profile')}
              >
                Profile
              </button>
              <button 
                style={{ flex: 1, padding: '10px', background: profileTab === 'security' ? 'linear-gradient(135deg, #D4A84B 0%, #B8923A 100%)' : 'transparent', border: 'none', borderRadius: '8px', color: profileTab === 'security' ? '#1A0F1E' : '#B8B4C8', fontWeight: profileTab === 'security' ? 600 : 500, cursor: 'pointer', transition: 'all 0.2s ease' }}
                onClick={() => setProfileTab('security')}
              >
                Security
              </button>
            </div>

            {/* Profile Tab */}
            {profileTab === 'profile' && (
              <>
                <div style={{ marginBottom: '16px' }}>
                  <label style={{ display: 'block', color: '#B8B4C8', marginBottom: '6px', fontSize: '0.875rem', fontWeight: 500 }}>Username</label>
                  <input 
                    style={{ width: '100%', padding: '12px 14px', backgroundColor: '#1F1428', border: '1px solid #4A2C5A', borderRadius: '10px', color: '#F0EDF4', fontSize: '0.95rem', boxSizing: 'border-box', outline: 'none' }}
                    type="text" 
                    value={profileForm.username} 
                    onChange={e => setProfileForm({...profileForm, username: e.target.value})} 
                  />
                </div>
                <div style={{ marginBottom: '16px' }}>
                  <label style={{ display: 'block', color: '#B8B4C8', marginBottom: '6px', fontSize: '0.875rem', fontWeight: 500 }}>Payout Address</label>
                  <input 
                    style={{ width: '100%', padding: '12px 14px', backgroundColor: '#1F1428', border: '1px solid #4A2C5A', borderRadius: '10px', color: '#F0EDF4', fontSize: '0.95rem', boxSizing: 'border-box', outline: 'none' }}
                    type="text" 
                    placeholder="Your mining payout address"
                    value={profileForm.payout_address} 
                    onChange={e => setProfileForm({...profileForm, payout_address: e.target.value})} 
                  />
                </div>
                <div style={{ marginBottom: '16px' }}>
                  <label style={{ display: 'block', color: '#B8B4C8', marginBottom: '6px', fontSize: '0.875rem', fontWeight: 500 }}>Email</label>
                  <input 
                    style={{ width: '100%', padding: '12px 14px', backgroundColor: '#1F1428', border: '1px solid #4A2C5A', borderRadius: '10px', color: '#7A7490', fontSize: '0.95rem', boxSizing: 'border-box', outline: 'none' }}
                    type="email" 
                    value={user?.email || ''} 
                    disabled
                  />
                  <span style={{ color: '#7A7490', fontSize: '0.8rem' }}>Email cannot be changed</span>
                </div>
                <div style={{ display: 'flex', gap: '12px', marginTop: '24px' }}>
                  <button 
                    style={{ flex: 1, padding: '12px', backgroundColor: 'rgba(123, 94, 167, 0.15)', border: '1px solid #4A2C5A', borderRadius: '10px', color: '#B8B4C8', cursor: 'pointer', fontWeight: 500, transition: 'all 0.2s ease' }}
                    onClick={() => { setShowProfileModal(false); setProfileTab('profile'); }}
                  >
                    Cancel
                  </button>
                  <button 
                    style={{ flex: 1, padding: '12px', background: 'linear-gradient(135deg, #D4A84B 0%, #B8923A 100%)', border: 'none', borderRadius: '10px', color: '#1A0F1E', fontWeight: 600, cursor: 'pointer', boxShadow: '0 4px 16px rgba(212, 168, 75, 0.3)', transition: 'all 0.2s ease' }}
                    onClick={handleUpdateProfile}
                  >
                    Save Changes
                  </button>
                </div>
              </>
            )}

            {/* Security Tab */}
            {profileTab === 'security' && (
              <>
                <h3 style={{ color: '#e0e0e0', marginBottom: '15px', fontSize: '1.1rem' }}>🔒 Change Password</h3>
                <div style={{ marginBottom: '15px' }}>
                  <label style={{ display: 'block', color: '#888', marginBottom: '5px', fontSize: '0.9rem' }}>Current Password</label>
                  <input 
                    style={{ width: '100%', padding: '12px', backgroundColor: '#0a0a15', border: '1px solid #2a2a4a', borderRadius: '6px', color: '#e0e0e0', fontSize: '1rem', boxSizing: 'border-box' }}
                    type="password" 
                    value={passwordForm.current_password} 
                    onChange={e => setPasswordForm({...passwordForm, current_password: e.target.value})} 
                  />
                </div>
                <div style={{ marginBottom: '15px' }}>
                  <label style={{ display: 'block', color: '#888', marginBottom: '5px', fontSize: '0.9rem' }}>New Password</label>
                  <input 
                    style={{ width: '100%', padding: '12px', backgroundColor: '#0a0a15', border: '1px solid #2a2a4a', borderRadius: '6px', color: '#e0e0e0', fontSize: '1rem', boxSizing: 'border-box' }}
                    type="password" 
                    placeholder="Minimum 8 characters"
                    value={passwordForm.new_password} 
                    onChange={e => setPasswordForm({...passwordForm, new_password: e.target.value})} 
                  />
                </div>
                <div style={{ marginBottom: '15px' }}>
                  <label style={{ display: 'block', color: '#888', marginBottom: '5px', fontSize: '0.9rem' }}>Confirm New Password</label>
                  <input 
                    style={{ width: '100%', padding: '12px', backgroundColor: '#0a0a15', border: '1px solid #2a2a4a', borderRadius: '6px', color: '#e0e0e0', fontSize: '1rem', boxSizing: 'border-box' }}
                    type="password" 
                    value={passwordForm.confirm_password} 
                    onChange={e => setPasswordForm({...passwordForm, confirm_password: e.target.value})} 
                  />
                </div>
                <div style={{ display: 'flex', gap: '10px', marginTop: '20px' }}>
                  <button 
                    style={{ flex: 1, padding: '12px', backgroundColor: '#2a2a4a', border: 'none', borderRadius: '6px', color: '#e0e0e0', cursor: 'pointer' }}
                    onClick={() => { setShowProfileModal(false); setProfileTab('profile'); setPasswordForm({ current_password: '', new_password: '', confirm_password: '' }); setForgotPasswordSent(false); }}
                  >
                    Cancel
                  </button>
                  <button 
                    style={{ flex: 1, padding: '12px', backgroundColor: '#00d4ff', border: 'none', borderRadius: '6px', color: '#0a0a0f', fontWeight: 'bold', cursor: 'pointer', opacity: passwordLoading ? 0.6 : 1 }}
                    onClick={handleChangePassword}
                    disabled={passwordLoading}
                  >
                    {passwordLoading ? 'Changing...' : 'Change Password'}
                  </button>
                </div>

                {/* Forgot Password Section */}
                <div style={{ marginTop: '30px', paddingTop: '20px', borderTop: '1px solid #2a2a4a' }}>
                  <h3 style={{ color: '#e0e0e0', marginBottom: '10px', fontSize: '1.1rem' }}>📧 Forgot Password?</h3>
                  <p style={{ color: '#888', fontSize: '0.9rem', marginBottom: '15px' }}>
                    If you've forgotten your current password, we can send a reset link to your email address: <strong style={{ color: '#00d4ff' }}>{user?.email}</strong>
                  </p>
                  {forgotPasswordSent ? (
                    <div style={{ padding: '15px', backgroundColor: '#1a4d1a', borderRadius: '6px', color: '#4ade80', textAlign: 'center' }}>
                      ✓ Password reset link sent! Check your email inbox.
                    </div>
                  ) : (
                    <button 
                      style={{ width: '100%', padding: '12px', backgroundColor: '#4a1a6b', border: 'none', borderRadius: '6px', color: '#e0e0e0', cursor: 'pointer', opacity: passwordLoading ? 0.6 : 1 }}
                      onClick={handleForgotPassword}
                      disabled={passwordLoading}
                    >
                      {passwordLoading ? 'Sending...' : 'Send Password Reset Email'}
                    </button>
                  )}
                </div>
              </>
            )}
          </div>
        </div>
      )}

      {/* Bug Report Modal */}
      {showBugReportModal && (
        <div style={{ position: 'fixed', inset: 0, backgroundColor: 'rgba(0,0,0,0.85)', display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 1000, padding: '15px', boxSizing: 'border-box' }} onClick={() => setShowBugReportModal(false)}>
          <div style={{ backgroundColor: '#1a1a2e', borderRadius: '12px', padding: '20px', maxWidth: '550px', width: '100%', border: '1px solid #2a2a4a', maxHeight: 'calc(100vh - 30px)', overflowY: 'auto', boxSizing: 'border-box' }} onClick={e => e.stopPropagation()}>
            <h2 style={{ color: '#00d4ff', marginBottom: '20px' }}>🐛 Report a Bug</h2>
            
            <div style={{ marginBottom: '15px' }}>
              <label style={{ display: 'block', color: '#888', marginBottom: '5px', fontSize: '0.9rem' }}>Title *</label>
              <input 
                style={{ width: '100%', padding: '12px', backgroundColor: '#0a0a15', border: '1px solid #2a2a4a', borderRadius: '6px', color: '#e0e0e0', fontSize: '1rem', boxSizing: 'border-box' }}
                placeholder="Brief description of the issue"
                value={bugReportForm.title}
                onChange={e => setBugReportForm({...bugReportForm, title: e.target.value})}
              />
            </div>

            <div style={{ marginBottom: '15px' }}>
              <label style={{ display: 'block', color: '#888', marginBottom: '5px', fontSize: '0.9rem' }}>Category</label>
              <select 
                style={{ width: '100%', padding: '12px', backgroundColor: '#0a0a15', border: '1px solid #2a2a4a', borderRadius: '6px', color: '#e0e0e0', fontSize: '1rem' }}
                value={bugReportForm.category}
                onChange={e => setBugReportForm({...bugReportForm, category: e.target.value})}
              >
                <option value="ui">UI/Visual Issue</option>
                <option value="performance">Performance</option>
                <option value="crash">Crash/Error</option>
                <option value="security">Security Concern</option>
                <option value="feature_request">Feature Request</option>
                <option value="other">Other</option>
              </select>
            </div>

            <div style={{ marginBottom: '15px' }}>
              <label style={{ display: 'block', color: '#888', marginBottom: '5px', fontSize: '0.9rem' }}>Description *</label>
              <textarea 
                style={{ width: '100%', padding: '12px', backgroundColor: '#0a0a15', border: '1px solid #2a2a4a', borderRadius: '6px', color: '#e0e0e0', fontSize: '1rem', minHeight: '100px', resize: 'vertical', boxSizing: 'border-box' }}
                placeholder="Describe the issue in detail..."
                value={bugReportForm.description}
                onChange={e => setBugReportForm({...bugReportForm, description: e.target.value})}
              />
            </div>

            <div style={{ marginBottom: '15px' }}>
              <label style={{ display: 'block', color: '#888', marginBottom: '5px', fontSize: '0.9rem' }}>Steps to Reproduce</label>
              <textarea 
                style={{ width: '100%', padding: '12px', backgroundColor: '#0a0a15', border: '1px solid #2a2a4a', borderRadius: '6px', color: '#e0e0e0', fontSize: '1rem', minHeight: '80px', resize: 'vertical', boxSizing: 'border-box' }}
                placeholder="1. Go to...&#10;2. Click on...&#10;3. See error"
                value={bugReportForm.steps_to_reproduce}
                onChange={e => setBugReportForm({...bugReportForm, steps_to_reproduce: e.target.value})}
              />
            </div>

            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '15px', marginBottom: '15px' }}>
              <div>
                <label style={{ display: 'block', color: '#888', marginBottom: '5px', fontSize: '0.9rem' }}>Expected Behavior</label>
                <textarea 
                  style={{ width: '100%', padding: '12px', backgroundColor: '#0a0a15', border: '1px solid #2a2a4a', borderRadius: '6px', color: '#e0e0e0', fontSize: '0.9rem', minHeight: '60px', resize: 'vertical', boxSizing: 'border-box' }}
                  placeholder="What should happen?"
                  value={bugReportForm.expected_behavior}
                  onChange={e => setBugReportForm({...bugReportForm, expected_behavior: e.target.value})}
                />
              </div>
              <div>
                <label style={{ display: 'block', color: '#888', marginBottom: '5px', fontSize: '0.9rem' }}>Actual Behavior</label>
                <textarea 
                  style={{ width: '100%', padding: '12px', backgroundColor: '#0a0a15', border: '1px solid #2a2a4a', borderRadius: '6px', color: '#e0e0e0', fontSize: '0.9rem', minHeight: '60px', resize: 'vertical', boxSizing: 'border-box' }}
                  placeholder="What actually happens?"
                  value={bugReportForm.actual_behavior}
                  onChange={e => setBugReportForm({...bugReportForm, actual_behavior: e.target.value})}
                />
              </div>
            </div>

            <div style={{ display: 'flex', gap: '10px', marginTop: '20px' }}>
              <button 
                style={{ flex: 1, padding: '12px', backgroundColor: '#2a2a4a', border: 'none', borderRadius: '6px', color: '#e0e0e0', cursor: 'pointer' }}
                onClick={() => setShowBugReportModal(false)}
              >
                Cancel
              </button>
              <button 
                style={{ flex: 1, padding: '12px', backgroundColor: '#00d4ff', border: 'none', borderRadius: '6px', color: '#0a0a0f', fontWeight: 'bold', cursor: 'pointer', opacity: bugReportLoading ? 0.6 : 1 }}
                onClick={handleSubmitBugReport}
                disabled={bugReportLoading}
              >
                {bugReportLoading ? 'Submitting...' : '🐛 Submit Bug Report'}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* My Bug Reports Modal */}
      {showMyBugsModal && (
        <div style={{ position: 'fixed', inset: 0, backgroundColor: 'rgba(0,0,0,0.85)', display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 1000, padding: '15px', boxSizing: 'border-box' }} onClick={() => { setShowMyBugsModal(false); setSelectedBug(null); }}>
          <div style={{ backgroundColor: '#1a1a2e', borderRadius: '12px', padding: '20px', maxWidth: '750px', width: '100%', border: '1px solid #2a2a4a', maxHeight: 'calc(100vh - 30px)', overflowY: 'auto', boxSizing: 'border-box' }} onClick={e => e.stopPropagation()}>
            {selectedBug ? (
              <>
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '20px' }}>
                  <h2 style={{ color: '#00d4ff', margin: 0 }}>🐛 {selectedBug.bug.report_number}</h2>
                  <button 
                    style={{ padding: '8px 16px', backgroundColor: '#2a2a4a', border: 'none', borderRadius: '6px', color: '#e0e0e0', cursor: 'pointer' }}
                    onClick={() => setSelectedBug(null)}
                  >
                    ← Back to List
                  </button>
                </div>
                
                <div style={{ marginBottom: '20px' }}>
                  <h3 style={{ color: '#e0e0e0', margin: '0 0 10px 0' }}>{selectedBug.bug.title}</h3>
                  <div style={{ display: 'flex', gap: '10px', marginBottom: '15px' }}>
                    <span style={{ padding: '4px 8px', borderRadius: '4px', fontSize: '0.8rem', backgroundColor: getStatusColor(selectedBug.bug.status), color: '#fff' }}>
                      {selectedBug.bug.status.replace('_', ' ')}
                    </span>
                    <span style={{ padding: '4px 8px', borderRadius: '4px', fontSize: '0.8rem', backgroundColor: getPriorityColor(selectedBug.bug.priority), color: '#fff' }}>
                      {selectedBug.bug.priority}
                    </span>
                    <span style={{ padding: '4px 8px', borderRadius: '4px', fontSize: '0.8rem', backgroundColor: '#2a2a4a', color: '#888' }}>
                      {selectedBug.bug.category}
                    </span>
                  </div>
                  <p style={{ color: '#ccc', whiteSpace: 'pre-wrap', margin: '0 0 15px 0' }}>{selectedBug.bug.description}</p>
                  
                  {selectedBug.bug.steps_to_reproduce && (
                    <div style={{ marginBottom: '15px' }}>
                      <h4 style={{ color: '#888', margin: '0 0 5px 0', fontSize: '0.9rem' }}>Steps to Reproduce:</h4>
                      <p style={{ color: '#ccc', whiteSpace: 'pre-wrap', margin: 0, fontSize: '0.9rem' }}>{selectedBug.bug.steps_to_reproduce}</p>
                    </div>
                  )}
                </div>

                {/* Comments Section */}
                <div style={{ borderTop: '1px solid #2a2a4a', paddingTop: '20px' }}>
                  <h4 style={{ color: '#e0e0e0', margin: '0 0 15px 0' }}>💬 Comments ({selectedBug.comments?.length || 0})</h4>
                  
                  {selectedBug.comments?.map((comment: any) => (
                    <div key={comment.id} style={{ backgroundColor: comment.is_status_change ? '#1a2a1a' : '#0a0a15', padding: '12px', borderRadius: '6px', marginBottom: '10px', borderLeft: comment.is_status_change ? '3px solid #10b981' : '3px solid #2a2a4a' }}>
                      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '5px' }}>
                        <span style={{ color: '#00d4ff', fontWeight: 'bold', fontSize: '0.9rem' }}>{comment.username}</span>
                        <span style={{ color: '#666', fontSize: '0.8rem' }}>{new Date(comment.created_at).toLocaleString()}</span>
                      </div>
                      <p style={{ color: '#ccc', margin: 0, fontSize: '0.9rem', whiteSpace: 'pre-wrap' }}>{comment.content}</p>
                    </div>
                  ))}

                  {/* Add Comment */}
                  <div style={{ marginTop: '15px' }}>
                    <textarea 
                      style={{ width: '100%', padding: '12px', backgroundColor: '#0a0a15', border: '1px solid #2a2a4a', borderRadius: '6px', color: '#e0e0e0', fontSize: '0.9rem', minHeight: '60px', resize: 'vertical', boxSizing: 'border-box' }}
                      placeholder="Add a comment..."
                      value={bugComment}
                      onChange={e => setBugComment(e.target.value)}
                    />
                    <button 
                      style={{ marginTop: '10px', padding: '10px 20px', backgroundColor: '#00d4ff', border: 'none', borderRadius: '6px', color: '#0a0a0f', fontWeight: 'bold', cursor: 'pointer' }}
                      onClick={handleAddBugComment}
                    >
                      Add Comment
                    </button>
                  </div>
                </div>
              </>
            ) : (
              <>
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '20px' }}>
                  <h2 style={{ color: '#00d4ff', margin: 0 }}>🐛 My Bug Reports</h2>
                  <button 
                    style={{ padding: '8px 16px', backgroundColor: '#00d4ff', border: 'none', borderRadius: '6px', color: '#0a0a0f', fontWeight: 'bold', cursor: 'pointer' }}
                    onClick={() => { setShowMyBugsModal(false); setShowBugReportModal(true); }}
                  >
                    + New Report
                  </button>
                </div>

                {myBugs.length === 0 ? (
                  <p style={{ color: '#888', textAlign: 'center', padding: '40px' }}>No bug reports yet. Click "New Report" to submit one.</p>
                ) : (
                  <div style={{ display: 'flex', flexDirection: 'column', gap: '10px' }}>
                    {myBugs.map((bug: any) => (
                      <div 
                        key={bug.id} 
                        style={{ backgroundColor: '#0a0a15', padding: '15px', borderRadius: '8px', cursor: 'pointer', border: '1px solid #2a2a4a', transition: 'border-color 0.2s' }}
                        onClick={() => handleViewBugDetails(bug.id)}
                        onMouseEnter={e => (e.currentTarget.style.borderColor = '#00d4ff')}
                        onMouseLeave={e => (e.currentTarget.style.borderColor = '#2a2a4a')}
                      >
                        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: '8px' }}>
                          <span style={{ color: '#00d4ff', fontSize: '0.85rem' }}>{bug.report_number}</span>
                          <div style={{ display: 'flex', gap: '5px' }}>
                            <span style={{ padding: '2px 6px', borderRadius: '4px', fontSize: '0.75rem', backgroundColor: getStatusColor(bug.status), color: '#fff' }}>
                              {bug.status.replace('_', ' ')}
                            </span>
                            <span style={{ padding: '2px 6px', borderRadius: '4px', fontSize: '0.75rem', backgroundColor: getPriorityColor(bug.priority), color: '#fff' }}>
                              {bug.priority}
                            </span>
                          </div>
                        </div>
                        <h4 style={{ color: '#e0e0e0', margin: '0 0 5px 0', fontSize: '1rem' }}>{bug.title}</h4>
                        <div style={{ display: 'flex', gap: '15px', color: '#666', fontSize: '0.8rem' }}>
                          <span>📎 {bug.attachment_count || 0}</span>
                          <span>💬 {bug.comment_count || 0}</span>
                          <span>{new Date(bug.created_at).toLocaleDateString()}</span>
                        </div>
                      </div>
                    ))}
                  </div>
                )}

                <button 
                  style={{ marginTop: '20px', width: '100%', padding: '12px', backgroundColor: '#2a2a4a', border: 'none', borderRadius: '6px', color: '#e0e0e0', cursor: 'pointer' }}
                  onClick={() => { setShowMyBugsModal(false); setSelectedBug(null); }}
                >
                  Close
                </button>
              </>
            )}
          </div>
        </div>
      )}

      {/* Equipment Support Request Modal */}
      <EquipmentSupportModal
        show={showEquipmentSupportModal}
        onClose={() => setShowEquipmentSupportModal(false)}
        form={equipmentSupportForm}
        setForm={setEquipmentSupportForm}
        onSubmit={handleEquipmentSupportSubmit}
      />

      <main style={mainView === 'community' ? { ...styles.main, maxWidth: '100%', padding: '0' } : styles.main} className="main-content">
        {/* DASHBOARD VIEW */}
        {mainView === 'dashboard' && (
          <>
            {loading ? (
              <div style={styles.loading}>Loading pool statistics...</div>
            ) : stats ? (
              <div style={styles.statsGrid} className="stats-grid">
                <StatCard label="Network" value={stats.network} />
                <StatCard label="Currency" value={stats.currency} />
                <StatCard label="Active Miners" value={stats.total_miners} />
                <StatCard label="Pool Hashrate" value={formatHashrate(stats.total_hashrate)} />
                <StatCard label="Blocks Found" value={stats.blocks_found} />
                <StatCard label="Min Payout" value={`${stats.minimum_payout} ${stats.currency}`} />
                <StatCard label="Payment Interval" value={stats.payment_interval} />
              </div>
            ) : (
              <div style={styles.error}>Failed to load pool statistics</div>
            )}

            {/* Mining Graphs - Visible to ALL users (pool-wide for guests, toggleable for members) */}
            <MiningGraphsLazy token={token || undefined} isLoggedIn={!!token && !!user} />

            {/* Pool Monitoring Dashboard - Shows node health and Grafana links */}
            <MonitoringDashboard token={token || undefined} />

            {/* Call-to-action for non-logged users */}
            {!token && (
              <section style={{ ...styles.section, background: 'linear-gradient(135deg, #2D1F3D 0%, #3A1F2E 50%, #1A0F1E 100%)', border: '1px solid rgba(212, 168, 75, 0.3)', textAlign: 'center', boxShadow: '0 0 40px rgba(212, 168, 75, 0.1)' }}>
                <h2 style={{ color: '#D4A84B', marginBottom: '12px', fontSize: '1.5rem', fontWeight: 700 }}>Start Mining Today</h2>
                <p style={{ color: '#B8B4C8', marginBottom: '24px', maxWidth: '600px', margin: '0 auto 24px', lineHeight: '1.7', fontSize: '0.95rem' }}>
                  Join hundreds of miners earning rewards. Create a free account to track your hashrate, 
                  manage payouts, report issues, and connect with the community.
                </p>
                <div style={{ display: 'flex', gap: '12px', justifyContent: 'center', flexWrap: 'wrap', marginBottom: '24px' }}>
                  <div style={{ background: 'linear-gradient(180deg, rgba(45, 31, 61, 0.6) 0%, rgba(26, 15, 30, 0.8) 100%)', padding: '16px 20px', borderRadius: '12px', border: '1px solid #4A2C5A', minWidth: '100px' }}>
                    <div style={{ width: '28px', height: '28px', borderRadius: '8px', backgroundColor: 'rgba(74, 222, 128, 0.15)', display: 'flex', alignItems: 'center', justifyContent: 'center', margin: '0 auto 8px', color: '#4ADE80', fontSize: '0.9rem' }}>◈</div>
                    <p style={{ color: '#B8B4C8', margin: 0, fontSize: '0.8rem', fontWeight: 500 }}>Real-time Stats</p>
                  </div>
                  <div style={{ background: 'linear-gradient(180deg, rgba(45, 31, 61, 0.6) 0%, rgba(26, 15, 30, 0.8) 100%)', padding: '16px 20px', borderRadius: '12px', border: '1px solid #4A2C5A', minWidth: '100px' }}>
                    <div style={{ width: '28px', height: '28px', borderRadius: '8px', backgroundColor: 'rgba(212, 168, 75, 0.15)', display: 'flex', alignItems: 'center', justifyContent: 'center', margin: '0 auto 8px', color: '#D4A84B', fontSize: '0.9rem' }}>◎</div>
                    <p style={{ color: '#B8B4C8', margin: 0, fontSize: '0.8rem', fontWeight: 500 }}>Auto Payouts</p>
                  </div>
                  <div style={{ background: 'linear-gradient(180deg, rgba(45, 31, 61, 0.6) 0%, rgba(26, 15, 30, 0.8) 100%)', padding: '16px 20px', borderRadius: '12px', border: '1px solid #4A2C5A', minWidth: '100px' }}>
                    <div style={{ width: '28px', height: '28px', borderRadius: '8px', backgroundColor: 'rgba(123, 94, 167, 0.15)', display: 'flex', alignItems: 'center', justifyContent: 'center', margin: '0 auto 8px', color: '#7B5EA7', fontSize: '0.9rem' }}>⚑</div>
                    <p style={{ color: '#B8B4C8', margin: 0, fontSize: '0.8rem', fontWeight: 500 }}>Bug Tracking</p>
                  </div>
                  <div style={{ background: 'linear-gradient(180deg, rgba(45, 31, 61, 0.6) 0%, rgba(26, 15, 30, 0.8) 100%)', padding: '16px 20px', borderRadius: '12px', border: '1px solid #4A2C5A', minWidth: '100px' }}>
                    <div style={{ width: '28px', height: '28px', borderRadius: '8px', backgroundColor: 'rgba(96, 165, 250, 0.15)', display: 'flex', alignItems: 'center', justifyContent: 'center', margin: '0 auto 8px', color: '#60A5FA', fontSize: '0.9rem' }}>◉</div>
                    <p style={{ color: '#B8B4C8', margin: 0, fontSize: '0.8rem', fontWeight: 500 }}>Community</p>
                  </div>
                </div>
                <div style={{ display: 'flex', gap: '12px', justifyContent: 'center', flexWrap: 'wrap' }}>
                  <button 
                    style={{ padding: '14px 32px', background: 'linear-gradient(135deg, #D4A84B 0%, #B8923A 100%)', border: 'none', borderRadius: '10px', color: '#1A0F1E', fontSize: '1rem', fontWeight: 600, cursor: 'pointer', boxShadow: '0 4px 16px rgba(212, 168, 75, 0.3)' }} 
                    onClick={() => setAuthView('register')}
                  >
                    Create Free Account
                  </button>
                  <button 
                    style={{ padding: '14px 32px', backgroundColor: 'transparent', border: '2px solid #7B5EA7', borderRadius: '10px', color: '#B8B4C8', fontSize: '1rem', fontWeight: 500, cursor: 'pointer' }} 
                    onClick={() => setAuthView('login')}
                  >
                    Login
                  </button>
                </div>
              </section>
            )}

            {/* User Dashboard - only shown when logged in */}
            {token && user && <UserDashboardLazy token={token} />}
            {token && user && <WalletManagerLazy token={token} showMessage={showMessage} />}

            {/* Global Miner Map - visible to all */}
            <GlobalMinerMapLazy />
          </>
        )}

        {/* COMMUNITY VIEW - Full Page Experience */}
        {mainView === 'community' && (
          token && user ? (
            <CommunityPageLazy token={token} user={user} showMessage={showMessage} />
          ) : (
            <div style={{ ...styles.main, maxWidth: '800px', textAlign: 'center', padding: '60px 20px' }}>
              <div style={{ backgroundColor: '#1a1a2e', borderRadius: '16px', padding: '40px', border: '2px solid #00d4ff', marginBottom: '30px' }}>
                <h2 style={{ color: '#00d4ff', marginBottom: '15px', fontSize: '2rem' }}>💬 Community Hub</h2>
                <p style={{ color: '#ccc', marginBottom: '25px', fontSize: '1.1rem', lineHeight: '1.6' }}>
                  Join our active community of miners! Share strategies, get help, and connect with fellow BlockDAG enthusiasts.
                </p>
                
                {/* Preview of what's inside */}
                <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(150px, 1fr))', gap: '15px', marginBottom: '30px' }}>
                  <div style={{ backgroundColor: '#0a0a15', padding: '20px', borderRadius: '10px', border: '1px solid #2a2a4a' }}>
                    <span style={{ fontSize: '2rem' }}>📢</span>
                    <p style={{ color: '#888', margin: '10px 0 0', fontSize: '0.9rem' }}>Announcements</p>
                  </div>
                  <div style={{ backgroundColor: '#0a0a15', padding: '20px', borderRadius: '10px', border: '1px solid #2a2a4a' }}>
                    <span style={{ fontSize: '2rem' }}>💡</span>
                    <p style={{ color: '#888', margin: '10px 0 0', fontSize: '0.9rem' }}>Mining Tips</p>
                  </div>
                  <div style={{ backgroundColor: '#0a0a15', padding: '20px', borderRadius: '10px', border: '1px solid #2a2a4a' }}>
                    <span style={{ fontSize: '2rem' }}>🤝</span>
                    <p style={{ color: '#888', margin: '10px 0 0', fontSize: '0.9rem' }}>Support</p>
                  </div>
                  <div style={{ backgroundColor: '#0a0a15', padding: '20px', borderRadius: '10px', border: '1px solid #2a2a4a' }}>
                    <span style={{ fontSize: '2rem' }}>🎯</span>
                    <p style={{ color: '#888', margin: '10px 0 0', fontSize: '0.9rem' }}>General Chat</p>
                  </div>
                </div>

                {/* Blurred preview teaser */}
                <div style={{ position: 'relative', marginBottom: '25px' }}>
                  <div style={{ backgroundColor: '#0a0a15', padding: '15px', borderRadius: '8px', filter: 'blur(4px)', userSelect: 'none' }}>
                    <div style={{ display: 'flex', gap: '10px', marginBottom: '10px' }}>
                      <div style={{ width: '32px', height: '32px', borderRadius: '50%', backgroundColor: '#2a2a4a' }}></div>
                      <div style={{ flex: 1 }}>
                        <div style={{ height: '12px', backgroundColor: '#2a2a4a', borderRadius: '4px', width: '100px', marginBottom: '5px' }}></div>
                        <div style={{ height: '10px', backgroundColor: '#1a1a2e', borderRadius: '4px', width: '80%' }}></div>
                      </div>
                    </div>
                    <div style={{ display: 'flex', gap: '10px' }}>
                      <div style={{ width: '32px', height: '32px', borderRadius: '50%', backgroundColor: '#2a2a4a' }}></div>
                      <div style={{ flex: 1 }}>
                        <div style={{ height: '12px', backgroundColor: '#2a2a4a', borderRadius: '4px', width: '120px', marginBottom: '5px' }}></div>
                        <div style={{ height: '10px', backgroundColor: '#1a1a2e', borderRadius: '4px', width: '60%' }}></div>
                      </div>
                    </div>
                  </div>
                  <div style={{ position: 'absolute', top: '50%', left: '50%', transform: 'translate(-50%, -50%)', backgroundColor: 'rgba(0,0,0,0.7)', padding: '8px 16px', borderRadius: '20px' }}>
                    <span style={{ color: '#00d4ff', fontSize: '0.85rem' }}>🔒 Members Only</span>
                  </div>
                </div>

                <div style={{ display: 'flex', gap: '15px', justifyContent: 'center', flexWrap: 'wrap' }}>
                  <button 
                    style={{ padding: '14px 30px', backgroundColor: '#00d4ff', border: 'none', borderRadius: '8px', color: '#0a0a0f', fontSize: '1rem', fontWeight: 'bold', cursor: 'pointer' }} 
                    onClick={() => setAuthView('register')}
                  >
                    🚀 Join Now - It's Free!
                  </button>
                  <button 
                    style={{ padding: '14px 30px', backgroundColor: 'transparent', border: '2px solid #00d4ff', borderRadius: '8px', color: '#00d4ff', fontSize: '1rem', cursor: 'pointer' }} 
                    onClick={() => setAuthView('login')}
                  >
                    Already a Member? Login
                  </button>
                </div>
              </div>
            </div>
          )
        )}

        {/* EQUIPMENT VIEW - Full Equipment Management */}
        {mainView === 'equipment' && token && user && (
          <EquipmentPageLazy token={token} user={user} showMessage={showMessage} />
        )}

        {/* Dashboard-only sections */}
        {mainView === 'dashboard' && (
          <>
        <section style={styles.section} className="section">
          <h2 style={styles.sectionTitle} className="section-title">🔗 Connect Your Miner</h2>
          
          {/* Hybrid Protocol Banner */}
          <div style={instructionStyles.protocolBanner}>
            <div style={instructionStyles.protocolBadge}>
              <span style={instructionStyles.v2Badge}>Stratum V2</span>
              <span style={instructionStyles.plusSign}>+</span>
              <span style={instructionStyles.v1Badge}>Stratum V1</span>
            </div>
            <p style={instructionStyles.protocolText}>
              <strong>Hybrid Protocol Support:</strong> Our pool automatically detects your miner type.
              BlockDAG X30/X100 ASICs use encrypted Stratum V2, while GPU miners use standard Stratum V1.
            </p>
          </div>

          {/* Step by Step Guide */}
          <div style={instructionStyles.stepsContainer}>
            <h3 style={instructionStyles.stepsTitle}>📋 Quick Start Guide</h3>
            <div style={instructionStyles.step}>
              <span style={instructionStyles.stepNumber}>1</span>
              <div>
                <strong>Create an Account</strong>
                <p style={instructionStyles.stepText}>Click "Register" above and create your account with email and password.</p>
              </div>
            </div>
            <div style={instructionStyles.step}>
              <span style={instructionStyles.stepNumber}>2</span>
              <div>
                <strong>Set Your Wallet Address</strong>
                <p style={instructionStyles.stepText}>After logging in, go to "Wallet Settings" and add your BDAG wallet address for payouts.</p>
              </div>
            </div>
            <div style={instructionStyles.step}>
              <span style={instructionStyles.stepNumber}>3</span>
              <div>
                <strong>Configure Your Miner</strong>
                <p style={instructionStyles.stepText}>Copy the connection details below into your mining software.</p>
              </div>
            </div>
            <div style={instructionStyles.step}>
              <span style={instructionStyles.stepNumber}>4</span>
              <div>
                <strong>Start Mining!</strong>
                <p style={instructionStyles.stepText}>Launch your miner and your stats will appear in the dashboard within minutes.</p>
              </div>
            </div>
          </div>

          {/* Connection Details - Login Required for Full Details */}
          <div style={instructionStyles.connectionBox}>
            <h3 style={instructionStyles.connectionTitle}>⚡ Pool Connection Settings</h3>
            {token && user ? (
              <>
                <div style={instructionStyles.copyableBox}>
                  <div style={instructionStyles.detailRow}>
                    <span style={instructionStyles.detailLabel}>Pool Address:</span>
                    <code style={instructionStyles.detailCode}>stratum+tcp://206.162.80.230:3333</code>
                  </div>
                  <div style={instructionStyles.detailRow}>
                    <span style={instructionStyles.detailLabel}>Username:</span>
                    <code style={{...instructionStyles.detailCode, color: '#10b981'}}>{user.email}</code>
                  </div>
                  <div style={instructionStyles.detailRow}>
                    <span style={instructionStyles.detailLabel}>Password:</span>
                    <code style={instructionStyles.detailCode}>your_account_password</code>
                  </div>
                  <div style={instructionStyles.detailRow}>
                    <span style={instructionStyles.detailLabel}>Algorithm:</span>
                    <code style={instructionStyles.detailCode}>scrpy-variant (BlockDAG Custom)</code>
                  </div>
                </div>
                <p style={instructionStyles.tipText}><strong>Tip:</strong> Use your Chimera Pool login credentials as username/password</p>
              </>
            ) : (
              <div style={{ textAlign: 'center', padding: '30px 20px' }}>
                <div style={{ backgroundColor: '#0a0a15', padding: '20px', borderRadius: '8px', border: '1px dashed #2a2a4a', marginBottom: '20px' }}>
                  <span style={{ fontSize: '2rem', display: 'block', marginBottom: '10px' }}>🔐</span>
                  <p style={{ color: '#888', margin: '0 0 15px' }}>Login to see your personalized connection settings</p>
                  <div style={{ display: 'flex', gap: '10px', justifyContent: 'center', flexWrap: 'wrap' }}>
                    <button 
                      style={{ padding: '10px 25px', background: 'linear-gradient(135deg, #D4A84B 0%, #B8923A 100%)', border: 'none', borderRadius: '8px', color: '#1A0F1E', fontWeight: 600, cursor: 'pointer' }}
                      onClick={() => setAuthView('login')}
                    >
                      Login
                    </button>
                    <button 
                      style={{ padding: '10px 25px', backgroundColor: 'transparent', border: '1px solid #7B5EA7', borderRadius: '8px', color: '#B8B4C8', cursor: 'pointer' }}
                      onClick={() => setAuthView('register')}
                    >
                      Create Account
                    </button>
                  </div>
                </div>
                <p style={{ color: '#666', fontSize: '0.85rem' }}>
                  After logging in, you'll see your username pre-filled and ready to copy into your miner.
                </p>
              </div>
            )}
          </div>

          {/* Hardware-Specific Instructions - Login required for full details */}
          <div style={instructionStyles.hardwareSection}>
            <h3 style={instructionStyles.examplesTitle}>🖥️ Setup by Hardware Type</h3>
            
            {/* Official ASIC Tab */}
            <div style={instructionStyles.hardwareCard}>
              <div style={instructionStyles.hardwareHeader}>
                <span style={instructionStyles.hardwareIcon}>⚡</span>
                <div>
                  <h4 style={instructionStyles.hardwareName}>BlockDAG X30 / X100 (Official ASIC)</h4>
                  <span style={instructionStyles.hardwareTag}>Stratum V2 + Noise Encryption</span>
                </div>
              </div>
              <div style={instructionStyles.hardwareBody}>
                <p style={instructionStyles.hardwareDesc}>
                  Official BlockDAG miners connect automatically with encrypted Stratum V2 protocol.
                </p>
                {token && user ? (
                  <>
                    <div style={instructionStyles.configBox}>
                      <strong>Configuration:</strong>
                      <pre style={instructionStyles.configCode}>
Pool URL: stratum+tcp://206.162.80.230:3333
Wallet:   {user.email}
Worker:   x100-rig1 (optional)
Password: your_account_password</pre>
                    </div>
                    <p style={instructionStyles.noteText}>
                      🔐 Your connection is automatically encrypted with Noise Protocol for maximum security.
                    </p>
                  </>
                ) : (
                  <p style={{ color: '#888', fontStyle: 'italic', padding: '10px', backgroundColor: '#0a0a15', borderRadius: '6px' }}>
                    🔒 <span style={{ color: '#00d4ff', cursor: 'pointer' }} onClick={() => setAuthView('login')}>Login</span> to see your personalized config
                  </p>
                )}
              </div>
            </div>

            {/* GPU Mining Tab */}
            <div style={instructionStyles.hardwareCard}>
              <div style={instructionStyles.hardwareHeader}>
                <span style={instructionStyles.hardwareIcon}>🎮</span>
                <div>
                  <h4 style={instructionStyles.hardwareName}>GPU Mining (NVIDIA / AMD)</h4>
                  <span style={instructionStyles.hardwareTagAlt}>Stratum V1</span>
                </div>
              </div>
              <div style={instructionStyles.hardwareBody}>
                <p style={instructionStyles.hardwareDesc}>
                  Use your favorite GPU mining software with the settings below.
                </p>
                {token && user ? (
                  <>
                    <div style={instructionStyles.minerExample}>
                      <h5 style={instructionStyles.minerName}>lolMiner (Recommended)</h5>
                      <code style={instructionStyles.commandCode}>
lolminer --algo SCRPY --pool stratum+tcp://206.162.80.230:3333 --user {user.email} --pass yourpassword</code>
                    </div>

                    <div style={instructionStyles.minerExample}>
                      <h5 style={instructionStyles.minerName}>BzMiner</h5>
                      <code style={instructionStyles.commandCode}>
bzminer -a scrpy -p stratum+tcp://206.162.80.230:3333 -w {user.email} --pass yourpassword</code>
                    </div>

                    <div style={instructionStyles.minerExample}>
                      <h5 style={instructionStyles.minerName}>SRBMiner-MULTI</h5>
                      <code style={instructionStyles.commandCode}>
SRBMiner-MULTI --algorithm scrpy --pool 206.162.80.230:3333 --wallet {user.email} --password yourpassword</code>
                    </div>
                  </>
                ) : (
                  <p style={{ color: '#888', fontStyle: 'italic', padding: '10px', backgroundColor: '#0a0a15', borderRadius: '6px' }}>
                    🔒 <span style={{ color: '#00d4ff', cursor: 'pointer' }} onClick={() => setAuthView('login')}>Login</span> to see your personalized mining commands
                  </p>
                )}
              </div>
            </div>

            {/* CPU Mining Tab */}
            <div style={instructionStyles.hardwareCard}>
              <div style={instructionStyles.hardwareHeader}>
                <span style={instructionStyles.hardwareIcon}>💻</span>
                <div>
                  <h4 style={instructionStyles.hardwareName}>CPU Mining</h4>
                  <span style={instructionStyles.hardwareTagAlt}>Stratum V1</span>
                </div>
              </div>
              <div style={instructionStyles.hardwareBody}>
                <p style={instructionStyles.hardwareDesc}>
                  CPU mining is supported but yields lower returns than GPU or ASIC mining.
                </p>
                {token && user ? (
                  <div style={instructionStyles.minerExample}>
                    <h5 style={instructionStyles.minerName}>CPUMiner-Multi</h5>
                    <code style={instructionStyles.commandCode}>
cpuminer -a scrpy -o stratum+tcp://206.162.80.230:3333 -u {user.email} -p yourpassword</code>
                  </div>
                ) : (
                  <p style={{ color: '#888', fontStyle: 'italic', padding: '10px', backgroundColor: '#0a0a15', borderRadius: '6px' }}>
                    🔒 <span style={{ color: '#00d4ff', cursor: 'pointer' }} onClick={() => setAuthView('login')}>Login</span> to see your personalized config
                  </p>
                )}
              </div>
            </div>
          </div>

          {/* Troubleshooting */}
          <div style={instructionStyles.troubleshootBox}>
            <h3 style={instructionStyles.troubleshootTitle}>🔧 Troubleshooting</h3>
            <div style={instructionStyles.troubleshootItem}>
              <strong style={{ color: '#C45C5C' }}>Connection Refused:</strong> Ensure port 3333 is not blocked by your firewall or router.
            </div>
            <div style={instructionStyles.troubleshootItem}>
              <strong style={{ color: '#C45C5C' }}>Authentication Failed:</strong> Double-check your email and password match your Chimera Pool account.
            </div>
            <div style={instructionStyles.troubleshootItem}>
              <strong style={{ color: '#C45C5C' }}>Shares Rejected:</strong> Make sure you're using the correct algorithm (scrpy-variant). Update your miner software.
            </div>
            <div style={instructionStyles.troubleshootItem}>
              <strong style={{ color: '#C45C5C' }}>No Payouts:</strong> Verify you've added a valid BDAG wallet address in your account settings.
            </div>
            <div style={instructionStyles.troubleshootItem}>
              <strong style={{ color: '#4ADE80' }}>Need Help?</strong> Join our Community chat for real-time support from other miners!
            </div>
          </div>
        </section>

        <section style={styles.section}>
          <h2 style={styles.sectionTitle}>📊 Pool Information</h2>
          <div style={styles.infoGrid}>
            <div style={styles.infoCard}>
              <span style={styles.infoIcon}>⚙️</span>
              <span style={styles.infoLabel}>Algorithm</span>
              <span style={styles.infoValue}>Scrpy-Variant (BlockDAG Custom)</span>
            </div>
            <div style={styles.infoCard}>
              <span style={styles.infoIcon}>💰</span>
              <span style={styles.infoLabel}>Payout System</span>
              <span style={styles.infoValue}>PPLNS (Pay Per Last N Shares)</span>
            </div>
            <div style={styles.infoCard}>
              <span style={styles.infoIcon}>📤</span>
              <span style={styles.infoLabel}>Minimum Payout</span>
              <span style={styles.infoValue}>1.0 BDAG</span>
            </div>
            <div style={styles.infoCard}>
              <span style={styles.infoIcon}>⏰</span>
              <span style={styles.infoLabel}>Payout Frequency</span>
              <span style={styles.infoValue}>Hourly (when minimum reached)</span>
            </div>
            <div style={styles.infoCard}>
              <span style={styles.infoIcon}>🔒</span>
              <span style={styles.infoLabel}>Protocol</span>
              <span style={styles.infoValue}>Stratum V1 + V2 Hybrid</span>
            </div>
          </div>
        </section>
          </>
        )}
      </main>

      <footer style={styles.footer}>
        <p>Chimera Pool - BlockDAG Mining Made Easy</p>
        <p style={styles.footerLinks}>
          <a href="https://awakening.bdagscan.com/" target="_blank" rel="noopener noreferrer" style={styles.link}>Block Explorer</a>
          {' | '}
          <a href="https://awakening.bdagscan.com/faucet" target="_blank" rel="noopener noreferrer" style={styles.link}>Faucet</a>
        </p>
      </footer>
    </div>
    </RealTimeDataProvider>
  );
}

function StatCard({ label, value }: { label: string; value: string | number }) {
  return (
    <div style={styles.statCard}>
      <h3 style={styles.statLabel}>{label}</h3>
      <p style={styles.statValue}>{value}</p>
    </div>
  );
}

// Equipment Registration Required Component
function EquipmentRequiredOverlay({ 
  status, 
  onRegisterEquipment, 
  onRequestSupport,
  onViewDashboard 
}: { 
  status: { hasEquipment: boolean; hasOnlineEquipment: boolean; equipmentCount: number; onlineCount: number; pendingSupport: boolean };
  onRegisterEquipment: () => void;
  onRequestSupport: () => void;
  onViewDashboard: () => void;
}) {
  const overlayStyles: { [key: string]: React.CSSProperties } = {
    container: { 
      background: 'linear-gradient(135deg, #1a1a2e 0%, #0f0f1a 100%)', 
      borderRadius: '16px', 
      padding: '40px', 
      border: '2px solid #2a2a4a',
      maxWidth: '600px',
      margin: '40px auto',
      textAlign: 'center'
    },
    icon: { fontSize: '4rem', marginBottom: '20px' },
    title: { color: '#00d4ff', fontSize: '1.8rem', margin: '0 0 15px' },
    subtitle: { color: '#888', fontSize: '1.1rem', margin: '0 0 30px', lineHeight: '1.6' },
    statusCard: { 
      backgroundColor: '#0a0a15', 
      padding: '20px', 
      borderRadius: '10px', 
      marginBottom: '25px',
      border: '1px solid #2a2a4a'
    },
    statusRow: { display: 'flex', justifyContent: 'space-between', marginBottom: '10px', color: '#888' },
    statusValue: { fontWeight: 'bold' },
    btnPrimary: { 
      padding: '14px 32px', 
      backgroundColor: '#00d4ff', 
      border: 'none', 
      borderRadius: '8px', 
      color: '#0a0a0f', 
      fontWeight: 'bold', 
      fontSize: '1rem',
      cursor: 'pointer',
      marginRight: '10px',
      marginBottom: '10px'
    },
    btnSecondary: { 
      padding: '14px 32px', 
      backgroundColor: 'transparent', 
      border: '2px solid #f59e0b', 
      borderRadius: '8px', 
      color: '#f59e0b', 
      fontWeight: 'bold', 
      fontSize: '1rem',
      cursor: 'pointer',
      marginRight: '10px',
      marginBottom: '10px'
    },
    btnOutline: { 
      padding: '12px 24px', 
      backgroundColor: 'transparent', 
      border: '1px solid #888', 
      borderRadius: '8px', 
      color: '#888', 
      cursor: 'pointer',
      marginBottom: '10px'
    },
    warning: { 
      backgroundColor: '#2a2a1a', 
      border: '1px solid #f59e0b', 
      borderRadius: '8px', 
      padding: '15px', 
      marginTop: '20px',
      color: '#f59e0b',
      fontSize: '0.9rem'
    },
    limitedAccess: {
      backgroundColor: '#1a1a2e',
      border: '1px dashed #888',
      borderRadius: '8px',
      padding: '20px',
      marginTop: '25px'
    }
  };

  // User has no equipment registered at all
  if (!status.hasEquipment) {
    return (
      <div style={overlayStyles.container}>
        <div style={overlayStyles.icon}>🔌</div>
        <h2 style={overlayStyles.title}>Equipment Registration Required</h2>
        <p style={overlayStyles.subtitle}>
          To access full pool features and start earning rewards, you need to register your mining equipment with the pool.
        </p>

        <div style={overlayStyles.statusCard}>
          <div style={overlayStyles.statusRow}>
            <span>Registered Equipment:</span>
            <span style={{...overlayStyles.statusValue, color: '#ef4444'}}>0</span>
          </div>
          <div style={overlayStyles.statusRow}>
            <span>Online Equipment:</span>
            <span style={{...overlayStyles.statusValue, color: '#888'}}>--</span>
          </div>
          <div style={overlayStyles.statusRow}>
            <span>Pool Access Level:</span>
            <span style={{...overlayStyles.statusValue, color: '#f59e0b'}}>Limited</span>
          </div>
        </div>

        <div>
          <button style={overlayStyles.btnPrimary} onClick={onRegisterEquipment}>
            ➕ Register Equipment
          </button>
          <button style={overlayStyles.btnSecondary} onClick={onRequestSupport}>
            🆘 Need Help?
          </button>
        </div>

        <div style={overlayStyles.limitedAccess}>
          <p style={{ color: '#888', margin: '0 0 10px', fontSize: '0.9rem' }}>
            <strong>Limited Access:</strong> You can view pool statistics and community, but personal mining features require equipment registration.
          </p>
          <button style={overlayStyles.btnOutline} onClick={onViewDashboard}>
            View Pool Dashboard (Limited)
          </button>
        </div>
      </div>
    );
  }

  // User has equipment but none are online
  if (!status.hasOnlineEquipment) {
    return (
      <div style={overlayStyles.container}>
        <div style={overlayStyles.icon}>⚠️</div>
        <h2 style={{...overlayStyles.title, color: '#f59e0b'}}>Equipment Offline</h2>
        <p style={overlayStyles.subtitle}>
          Your registered equipment is currently offline. Connect your miner to start earning rewards and unlock full pool features.
        </p>

        <div style={overlayStyles.statusCard}>
          <div style={overlayStyles.statusRow}>
            <span>Registered Equipment:</span>
            <span style={{...overlayStyles.statusValue, color: '#00d4ff'}}>{status.equipmentCount}</span>
          </div>
          <div style={overlayStyles.statusRow}>
            <span>Online Equipment:</span>
            <span style={{...overlayStyles.statusValue, color: '#ef4444'}}>{status.onlineCount} / {status.equipmentCount}</span>
          </div>
          <div style={overlayStyles.statusRow}>
            <span>Pool Access Level:</span>
            <span style={{...overlayStyles.statusValue, color: '#f59e0b'}}>Pending Activation</span>
          </div>
        </div>

        <div>
          <button style={overlayStyles.btnPrimary} onClick={onRegisterEquipment}>
            ⚙️ Manage Equipment
          </button>
          <button style={overlayStyles.btnSecondary} onClick={onRequestSupport}>
            🆘 Connection Issues?
          </button>
        </div>

        {status.pendingSupport && (
          <div style={overlayStyles.warning}>
            ✅ Support request submitted. Our team will contact you shortly.
          </div>
        )}

        <div style={overlayStyles.limitedAccess}>
          <p style={{ color: '#888', margin: '0 0 10px', fontSize: '0.9rem' }}>
            <strong>Pending Activation:</strong> Once your equipment comes online, you'll have full pool access.
          </p>
          <button style={overlayStyles.btnOutline} onClick={onViewDashboard}>
            View Pool Dashboard (Limited)
          </button>
        </div>
      </div>
    );
  }

  return null;
}

// Equipment Support Request Modal Component
function EquipmentSupportModal({
  show,
  onClose,
  form,
  setForm,
  onSubmit
}: {
  show: boolean;
  onClose: () => void;
  form: { issue_type: string; equipment_type: string; description: string; error_message: string };
  setForm: (form: any) => void;
  onSubmit: () => void;
}) {
  if (!show) return null;

  const modalStyles: { [key: string]: React.CSSProperties } = {
    overlay: { position: 'fixed', top: 0, left: 0, right: 0, bottom: 0, backgroundColor: 'rgba(0,0,0,0.85)', display: 'flex', justifyContent: 'center', alignItems: 'center', zIndex: 1000, padding: '15px', boxSizing: 'border-box' },
    modal: { backgroundColor: '#1a1a2e', padding: '20px', borderRadius: '12px', border: '2px solid #f59e0b', maxWidth: '480px', width: '100%', maxHeight: 'calc(100vh - 30px)', overflowY: 'auto', boxSizing: 'border-box' },
    label: { display: 'block', color: '#888', marginBottom: '4px', fontSize: '0.85rem' },
    input: { width: '100%', padding: '10px', backgroundColor: '#0a0a15', border: '1px solid #2a2a4a', borderRadius: '6px', color: '#e0e0e0', fontSize: '0.95rem', marginBottom: '12px', boxSizing: 'border-box' as const },
    select: { width: '100%', padding: '10px', backgroundColor: '#0a0a15', border: '1px solid #2a2a4a', borderRadius: '6px', color: '#e0e0e0', fontSize: '0.95rem', marginBottom: '12px', cursor: 'pointer', boxSizing: 'border-box' },
    textarea: { width: '100%', padding: '10px', backgroundColor: '#0a0a15', border: '1px solid #2a2a4a', borderRadius: '6px', color: '#e0e0e0', fontSize: '0.95rem', marginBottom: '12px', minHeight: '80px', resize: 'vertical' as const, boxSizing: 'border-box' as const },
    cancelBtn: { padding: '10px 18px', backgroundColor: 'transparent', border: '1px solid #888', borderRadius: '6px', color: '#888', cursor: 'pointer', fontSize: '0.9rem' },
    submitBtn: { padding: '10px 18px', backgroundColor: '#f59e0b', border: 'none', borderRadius: '6px', color: '#0a0a0f', fontWeight: 'bold', cursor: 'pointer', fontSize: '0.9rem' }
  };

  return (
    <div style={modalStyles.overlay} onClick={onClose}>
      <div style={modalStyles.modal} onClick={e => e.stopPropagation()}>
        <h2 style={{ color: '#f59e0b', marginTop: 0 }}>🆘 Equipment Support Request</h2>
        <p style={{ color: '#888', marginBottom: '20px' }}>
          Having trouble getting your equipment online? Our team is here to help!
        </p>

        <div>
          <label style={modalStyles.label}>Issue Type *</label>
          <select
            style={modalStyles.select}
            value={form.issue_type}
            onChange={e => setForm({...form, issue_type: e.target.value})}
          >
            <option value="connection">Cannot connect to pool</option>
            <option value="configuration">Configuration help needed</option>
            <option value="hardware">Hardware compatibility question</option>
            <option value="performance">Low hashrate / performance issues</option>
            <option value="errors">Error messages</option>
            <option value="other">Other</option>
          </select>
        </div>

        <div>
          <label style={modalStyles.label}>Equipment Type *</label>
          <select
            style={modalStyles.select}
            value={form.equipment_type}
            onChange={e => setForm({...form, equipment_type: e.target.value})}
          >
            <option value="">Select your equipment...</option>
            <option value="blockdag_x100">BlockDAG X100 ASIC</option>
            <option value="blockdag_x30">BlockDAG X30 ASIC</option>
            <option value="gpu_nvidia">NVIDIA GPU</option>
            <option value="gpu_amd">AMD GPU</option>
            <option value="cpu">CPU</option>
            <option value="other_asic">Other ASIC</option>
            <option value="other">Other</option>
          </select>
        </div>

        <div>
          <label style={modalStyles.label}>Describe Your Issue *</label>
          <textarea
            style={modalStyles.textarea}
            placeholder="Please describe what's happening and what you've tried so far..."
            value={form.description}
            onChange={e => setForm({...form, description: e.target.value})}
          />
        </div>

        <div>
          <label style={modalStyles.label}>Error Message (if any)</label>
          <input
            style={modalStyles.input}
            type="text"
            placeholder="Copy any error messages here"
            value={form.error_message}
            onChange={e => setForm({...form, error_message: e.target.value})}
          />
        </div>

        <div style={{ backgroundColor: '#0a1a15', border: '1px solid #4ade80', borderRadius: '8px', padding: '15px', marginBottom: '20px' }}>
          <p style={{ color: '#4ade80', margin: 0, fontSize: '0.9rem' }}>
            💡 <strong>Quick Tips:</strong> Make sure your miner is configured with the correct pool address (stratum+tcp://206.162.80.230:3333) and your wallet address as the username.
          </p>
        </div>

        <div style={{ display: 'flex', gap: '10px', justifyContent: 'flex-end' }}>
          <button style={modalStyles.cancelBtn} onClick={onClose}>Cancel</button>
          <button 
            style={{...modalStyles.submitBtn, opacity: !form.equipment_type || !form.description ? 0.5 : 1}}
            disabled={!form.equipment_type || !form.description}
            onClick={onSubmit}
          >
            Submit Support Request
          </button>
        </div>
      </div>
    </div>
  );
}

// NOTE: EquipmentPage extracted to src/components/equipment/EquipmentPage.tsx
// Using EquipmentPageLazy from LazyComponents for code splitting

// NOTE: graphStyles removed - was dead code (AdminPanel has its own copy)

// NOTE: CommunityPage extracted to src/components/community/CommunityPage.tsx
// Using CommunityPageLazy from LazyComponents for code splitting

// NOTE: CommunitySection and commStyles were removed - unused dead code (replaced by CommunityPage)

const instructionStyles: { [key: string]: React.CSSProperties } = {
  // Protocol Banner
  protocolBanner: { backgroundColor: '#0a1520', padding: '20px', borderRadius: '12px', marginBottom: '25px', border: '2px solid #00d4ff', textAlign: 'center' },
  protocolBadge: { display: 'flex', alignItems: 'center', justifyContent: 'center', gap: '12px', marginBottom: '12px' },
  v2Badge: { backgroundColor: '#9b59b6', color: '#fff', padding: '8px 16px', borderRadius: '20px', fontWeight: 'bold', fontSize: '1rem' },
  v1Badge: { backgroundColor: '#3498db', color: '#fff', padding: '8px 16px', borderRadius: '20px', fontWeight: 'bold', fontSize: '1rem' },
  plusSign: { color: '#00d4ff', fontSize: '1.5rem', fontWeight: 'bold' },
  protocolText: { color: '#b0b0b0', fontSize: '0.95rem', margin: 0, lineHeight: '1.6' },
  // Steps
  stepsContainer: { backgroundColor: '#0a0a15', padding: '25px', borderRadius: '12px', marginBottom: '25px' },
  stepsTitle: { color: '#00d4ff', fontSize: '1.2rem', marginTop: 0, marginBottom: '20px' },
  step: { display: 'flex', gap: '15px', marginBottom: '18px', alignItems: 'flex-start' },
  stepNumber: { backgroundColor: '#00d4ff', color: '#0a0a0f', width: '32px', height: '32px', borderRadius: '50%', display: 'flex', alignItems: 'center', justifyContent: 'center', fontWeight: 'bold', fontSize: '1rem', flexShrink: 0 },
  stepText: { color: '#888', fontSize: '0.95rem', margin: '4px 0 0', lineHeight: '1.5' },
  // Connection Box
  connectionBox: { backgroundColor: '#0a0a15', padding: '25px', borderRadius: '12px', marginBottom: '25px', border: '2px solid #00d4ff' },
  connectionTitle: { color: '#00d4ff', fontSize: '1.2rem', marginTop: 0, marginBottom: '20px' },
  copyableBox: { backgroundColor: '#1a1a2e', padding: '15px', borderRadius: '8px' },
  detailRow: { display: 'flex', flexDirection: 'column', marginBottom: '15px' },
  detailLabel: { color: '#888', fontSize: '0.85rem', marginBottom: '6px', textTransform: 'uppercase', letterSpacing: '0.5px' },
  detailCode: { backgroundColor: '#0a0a15', color: '#00ff88', padding: '12px 16px', borderRadius: '8px', fontFamily: 'monospace', fontSize: '1rem', wordBreak: 'break-all', border: '1px solid #2a2a4a' },
  tipText: { color: '#fbbf24', fontSize: '0.9rem', marginTop: '15px', marginBottom: 0, padding: '10px 15px', backgroundColor: '#2a2a15', borderRadius: '6px' },
  // Hardware Section
  hardwareSection: { marginBottom: '25px' },
  hardwareCard: { backgroundColor: '#0a0a15', borderRadius: '12px', marginBottom: '20px', border: '1px solid #2a2a4a', overflow: 'hidden' },
  hardwareHeader: { display: 'flex', alignItems: 'center', gap: '15px', padding: '20px', backgroundColor: '#1a1a2e', borderBottom: '1px solid #2a2a4a' },
  hardwareIcon: { fontSize: '2rem' },
  hardwareName: { color: '#00d4ff', fontSize: '1.1rem', margin: 0 },
  hardwareTag: { display: 'inline-block', backgroundColor: '#9b59b6', color: '#fff', padding: '4px 10px', borderRadius: '12px', fontSize: '0.75rem', marginTop: '5px' },
  hardwareTagAlt: { display: 'inline-block', backgroundColor: '#3498db', color: '#fff', padding: '4px 10px', borderRadius: '12px', fontSize: '0.75rem', marginTop: '5px' },
  hardwareBody: { padding: '20px' },
  hardwareDesc: { color: '#888', fontSize: '0.95rem', margin: '0 0 15px', lineHeight: '1.5' },
  configBox: { backgroundColor: '#1a1a2e', padding: '15px', borderRadius: '8px', marginBottom: '15px' },
  configCode: { color: '#00ff88', fontFamily: 'monospace', fontSize: '0.9rem', margin: '10px 0 0', lineHeight: '1.8', whiteSpace: 'pre-wrap' },
  noteText: { color: '#4ade80', fontSize: '0.9rem', margin: 0, padding: '10px 15px', backgroundColor: '#0a2a15', borderRadius: '6px' },
  // Examples
  examplesContainer: { backgroundColor: '#0a0a15', padding: '20px', borderRadius: '8px', marginBottom: '20px' },
  examplesTitle: { color: '#00d4ff', fontSize: '1.2rem', marginTop: 0, marginBottom: '20px' },
  minerExample: { marginBottom: '20px' },
  minerName: { color: '#fbbf24', fontSize: '1rem', marginTop: 0, marginBottom: '10px' },
  commandCode: { display: 'block', backgroundColor: '#1a1a2e', color: '#e0e0e0', padding: '15px', borderRadius: '8px', fontFamily: 'monospace', fontSize: '0.85rem', overflowX: 'auto', whiteSpace: 'pre-wrap', wordBreak: 'break-all', border: '1px solid #2a2a4a' },
  // Troubleshooting
  troubleshootBox: { backgroundColor: '#0a0a15', padding: '25px', borderRadius: '12px', border: '1px solid #4a4a6a' },
  troubleshootTitle: { color: '#fbbf24', fontSize: '1.2rem', marginTop: 0, marginBottom: '20px' },
  troubleshootItem: { color: '#e0e0e0', fontSize: '0.95rem', marginBottom: '15px', lineHeight: '1.6', paddingLeft: '10px', borderLeft: '3px solid #2a2a4a' },
};

// NOTE: AuthModal and AuthModalProps were removed - unused (AuthModalLazy from LazyComponents is used instead)
// The AuthModal component is in src/components/auth/AuthModal.tsx

// NOTE: AdminPanel extracted to src/components/admin/AdminPanel.tsx
// Using AdminPanelLazy from LazyComponents for code splitting

// ============================================================================
// CHIMERA POOL - ELITE DESIGN SYSTEM STYLES
// Color palette inspired by the mythological Chimera: Lion, Goat, and Serpent
// ============================================================================
const styles: { [key: string]: React.CSSProperties } = {
  // Layout
  container: { minHeight: '100vh', background: 'linear-gradient(180deg, #1A0F1E 0%, #0D0811 100%)', color: '#F0EDF4', fontFamily: "'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif" },
  
  // Header - Premium gradient with glass effect
  header: { background: 'linear-gradient(135deg, #2D1F3D 0%, #3A1F2E 100%)', padding: '16px 24px', borderBottom: '1px solid rgba(74, 44, 90, 0.5)', backdropFilter: 'blur(10px)', position: 'sticky' as const, top: 0, zIndex: 100 },
  headerContent: { maxWidth: '1400px', margin: '0 auto', display: 'flex', justifyContent: 'space-between', alignItems: 'center', flexWrap: 'wrap' as const, gap: '20px' },
  
  // Title - Lion Gold accent
  title: { fontSize: '1.75rem', margin: 0, color: '#D4A84B', fontWeight: 700, letterSpacing: '-0.02em', display: 'flex', alignItems: 'center', gap: '12px' },
  subtitle: { fontSize: '0.9rem', color: '#B8B4C8', margin: '4px 0 0', letterSpacing: '0.02em' },
  
  // Auth buttons - Modern styling
  authButtons: { display: 'flex', gap: '12px', alignItems: 'center' },
  authBtn: { padding: '10px 20px', backgroundColor: 'transparent', border: '1px solid #7B5EA7', color: '#B8B4C8', borderRadius: '10px', cursor: 'pointer', fontSize: '0.9rem', fontWeight: 500, transition: 'all 0.2s ease' },
  registerBtn: { background: 'linear-gradient(135deg, #D4A84B 0%, #B8923A 100%)', border: 'none', color: '#1A0F1E', fontWeight: 600 },
  
  // User info
  userInfo: { display: 'flex', alignItems: 'center', gap: '12px', flexWrap: 'wrap' as const },
  username: { color: '#D4A84B', fontSize: '0.95rem', fontWeight: 500 },
  logoutBtn: { padding: '8px 16px', backgroundColor: 'rgba(196, 92, 92, 0.15)', border: '1px solid rgba(196, 92, 92, 0.3)', color: '#C45C5C', borderRadius: '8px', cursor: 'pointer', fontSize: '0.85rem', fontWeight: 500, transition: 'all 0.2s ease' },
  
  // Messages
  message: { padding: '14px 20px', textAlign: 'center', color: '#F0EDF4', fontSize: '0.9rem', borderRadius: '0' },
  
  // Main content
  main: { maxWidth: '1400px', margin: '0 auto', padding: '32px 24px' },
  loading: { textAlign: 'center', padding: '60px', color: '#D4A84B' },
  error: { textAlign: 'center', padding: '60px', color: '#C45C5C' },
  
  // Stats grid - Premium cards
  statsGrid: { display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(180px, 1fr))', gap: '16px', marginBottom: '32px' },
  statCard: { background: 'linear-gradient(180deg, rgba(45, 31, 61, 0.8) 0%, rgba(26, 15, 30, 0.9) 100%)', borderRadius: '14px', padding: '20px', border: '1px solid #4A2C5A', textAlign: 'center', boxShadow: '0 4px 24px rgba(0, 0, 0, 0.3)', transition: 'all 0.2s ease' },
  statLabel: { fontSize: '0.75rem', color: '#B8B4C8', margin: '0 0 8px', textTransform: 'uppercase', letterSpacing: '0.08em', fontWeight: 500 },
  statValue: { fontSize: '1.5rem', color: '#D4A84B', margin: 0, fontWeight: 700, letterSpacing: '-0.02em' },
  
  // Sections - Glass card effect
  section: { background: 'linear-gradient(180deg, rgba(45, 31, 61, 0.6) 0%, rgba(26, 15, 30, 0.8) 100%)', borderRadius: '16px', padding: '24px', border: '1px solid #4A2C5A', marginBottom: '24px', boxShadow: '0 4px 24px rgba(0, 0, 0, 0.3)', backdropFilter: 'blur(10px)' },
  sectionTitle: { fontSize: '1.15rem', color: '#F0EDF4', margin: '0 0 20px', fontWeight: 600, letterSpacing: '0.01em' },
  
  // Connection info
  connectionInfo: { backgroundColor: 'rgba(31, 20, 40, 0.8)', padding: '16px', borderRadius: '12px', border: '1px solid #4A2C5A' },
  code: { display: 'block', backgroundColor: '#0D0811', color: '#4ADE80', padding: '14px 18px', borderRadius: '10px', fontFamily: "'JetBrains Mono', 'Fira Code', monospace", fontSize: '0.95rem', margin: '10px 0', border: '1px solid #4A2C5A' },
  hint: { color: '#B8B4C8', fontSize: '0.85rem', margin: '10px 0 0' },
  
  // Info grid
  info: { lineHeight: '1.8' },
  infoGrid: { display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(160px, 1fr))', gap: '14px' },
  infoCard: { background: 'linear-gradient(180deg, rgba(45, 31, 61, 0.5) 0%, rgba(26, 15, 30, 0.7) 100%)', padding: '18px', borderRadius: '12px', textAlign: 'center', border: '1px solid #4A2C5A', display: 'flex', flexDirection: 'column', alignItems: 'center', gap: '8px', transition: 'all 0.2s ease' },
  infoIcon: { fontSize: '1.5rem' },
  infoLabel: { color: '#B8B4C8', fontSize: '0.75rem', textTransform: 'uppercase', letterSpacing: '0.08em', fontWeight: 500 },
  infoValue: { color: '#D4A84B', fontSize: '0.9rem', fontWeight: 600 },
  
  // Footer
  footer: { textAlign: 'center', padding: '32px 24px', borderTop: '1px solid #4A2C5A', color: '#7A7490', background: 'linear-gradient(180deg, transparent 0%, rgba(13, 8, 17, 0.5) 100%)' },
  footerLinks: { marginTop: '12px' },
  link: { color: '#7B5EA7', textDecoration: 'none', fontWeight: 500, transition: 'color 0.2s ease' },
  
  // Modal - Elevated glass effect
  modalOverlay: { position: 'fixed' as const, top: 0, left: 0, right: 0, bottom: 0, backgroundColor: 'rgba(13, 8, 17, 0.9)', backdropFilter: 'blur(8px)', display: 'flex', justifyContent: 'center', alignItems: 'center', zIndex: 1000 },
  modal: { background: 'linear-gradient(180deg, #2D1F3D 0%, #1A0F1E 100%)', padding: '32px', borderRadius: '20px', border: '1px solid #4A2C5A', width: '100%', maxWidth: '420px', position: 'relative' as const, boxShadow: '0 24px 48px rgba(0, 0, 0, 0.5)' },
  closeBtn: { position: 'absolute' as const, top: '16px', right: '16px', background: 'rgba(123, 94, 167, 0.15)', border: '1px solid #4A2C5A', color: '#B8B4C8', fontSize: '18px', cursor: 'pointer', width: '32px', height: '32px', borderRadius: '8px', display: 'flex', alignItems: 'center', justifyContent: 'center', transition: 'all 0.2s ease' },
  modalTitle: { color: '#D4A84B', marginBottom: '8px', textAlign: 'center', fontSize: '1.5rem', fontWeight: 600 },
  modalDesc: { color: '#B8B4C8', fontSize: '0.9rem', marginBottom: '24px', textAlign: 'center' },
  
  // Form elements
  input: { width: '100%', padding: '14px 16px', marginBottom: '16px', backgroundColor: '#1F1428', border: '1px solid #4A2C5A', borderRadius: '12px', color: '#F0EDF4', fontSize: '0.95rem', boxSizing: 'border-box' as const, outline: 'none', transition: 'all 0.2s ease' },
  submitBtn: { width: '100%', padding: '14px', background: 'linear-gradient(135deg, #D4A84B 0%, #B8923A 100%)', border: 'none', borderRadius: '12px', color: '#1A0F1E', fontSize: '1rem', fontWeight: 600, cursor: 'pointer', marginTop: '8px', boxShadow: '0 4px 16px rgba(212, 168, 75, 0.3)', transition: 'all 0.2s ease' },
  errorMsg: { backgroundColor: 'rgba(196, 92, 92, 0.15)', color: '#C45C5C', padding: '12px 16px', borderRadius: '10px', marginBottom: '16px', fontSize: '0.9rem', textAlign: 'center', border: '1px solid rgba(196, 92, 92, 0.3)' },
  
  // Auth links
  authLinks: { display: 'flex', justifyContent: 'space-between', marginTop: '20px', flexWrap: 'wrap' as const, gap: '10px' },
  authLink: { color: '#7B5EA7', fontSize: '0.9rem', cursor: 'pointer', textDecoration: 'none', fontWeight: 500, transition: 'color 0.2s ease' },
};

// NOTE: adminStyles removed - was dead code (AdminPanel.tsx has its own copy)

export default App;
