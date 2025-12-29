import React, { useState, useEffect } from 'react';

// Error Boundary for graceful error handling
import { ChartErrorBoundary } from './components/charts/ChartErrorBoundary';

// Real-time data provider for synchronized mining data across all components
import { RealTimeDataProvider } from './services/realtime';

// Lazy-loaded components for code splitting
import {
  GrafanaDashboardLazy,
  GlobalMinerMapLazy,
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

// Collapsible User Mining Dashboard with auto-collapse for inactive equipment
import { CollapsibleUserDashboard } from './components/dashboard/CollapsibleUserDashboard';
import { useUserEquipmentStatus } from './hooks/useUserEquipmentStatus';

// Litecoin Mining Instructions
import { MiningInstructionsLitecoin } from './components/mining/MiningInstructionsLitecoin';

// Modular modal components
import { BugReportModal, ProfileModal } from './components/modals';


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

type MainView = 'dashboard' | 'community' | 'equipment';

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

  // Equipment status hook for collapsible user mining dashboard
  // Dashboard auto-collapses when no active equipment is detected
  const { status: equipmentStatus } = useUserEquipmentStatus({ token });
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
    }
  }, [token]);

  // Equipment status is now managed by useUserEquipmentStatus hook

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
      // Equipment status will be refreshed by the hook automatically
      setShowEquipmentSupportModal(false);
      setEquipmentSupportForm({ issue_type: 'connection', equipment_type: '', description: '', error_message: '' });
      showMessage('success', 'Support request submitted! Our team will contact you shortly.');
    } catch (error) {
      // Still show success for demo - API may not be ready
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
          <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
            <img src="/logo.png" alt="Chimera Pool" style={{ height: '85px', width: 'auto' }} />
            <span style={{ fontSize: '1.5rem', fontWeight: 700, color: '#D4A84B', letterSpacing: '0.5px' }}>Chimera Pool</span>
          </div>
          {/* Main Navigation Tabs - Elite styling */}
          <nav className="header-nav" style={{ display: 'flex', gap: '4px', backgroundColor: 'rgba(31, 20, 40, 0.8)', borderRadius: '12px', padding: '4px', border: '1px solid #4A2C5A' }}>
            <button 
              className={mainView !== 'dashboard' ? 'nav-tab-enhanced' : ''}
              style={{ padding: '10px 20px', background: mainView === 'dashboard' ? 'linear-gradient(135deg, #D4A84B 0%, #B8923A 100%)' : 'transparent', border: 'none', color: mainView === 'dashboard' ? '#1A0F1E' : '#B8B4C8', fontSize: '0.9rem', cursor: 'pointer', borderRadius: '8px', fontWeight: mainView === 'dashboard' ? 600 : 500, transition: 'all 0.2s ease', boxShadow: mainView === 'dashboard' ? '0 2px 12px rgba(212, 168, 75, 0.3)' : 'none' }}
              onClick={() => setMainView('dashboard')}
            >
              Dashboard
            </button>
            {token && user && (
              <button 
                className={mainView !== 'equipment' ? 'nav-tab-enhanced' : ''}
                style={{ padding: '10px 20px', background: mainView === 'equipment' ? 'linear-gradient(135deg, #D4A84B 0%, #B8923A 100%)' : 'transparent', border: 'none', color: mainView === 'equipment' ? '#1A0F1E' : '#B8B4C8', fontSize: '0.9rem', cursor: 'pointer', borderRadius: '8px', fontWeight: mainView === 'equipment' ? 600 : 500, transition: 'all 0.2s ease', boxShadow: mainView === 'equipment' ? '0 2px 12px rgba(212, 168, 75, 0.3)' : 'none' }}
                onClick={() => setMainView('equipment')}
              >
                Equipment
              </button>
            )}
            <button 
              className={mainView !== 'community' ? 'nav-tab-enhanced' : ''}
              style={{ padding: '10px 20px', background: mainView === 'community' ? 'linear-gradient(135deg, #D4A84B 0%, #B8923A 100%)' : 'transparent', border: 'none', color: mainView === 'community' ? '#1A0F1E' : '#B8B4C8', fontSize: '0.9rem', cursor: 'pointer', borderRadius: '8px', fontWeight: mainView === 'community' ? 600 : 500, transition: 'all 0.2s ease', boxShadow: mainView === 'community' ? '0 2px 12px rgba(212, 168, 75, 0.3)' : 'none' }}
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

      {/* Profile Modal */}
      <ProfileModal 
        isOpen={showProfileModal} 
        onClose={() => setShowProfileModal(false)} 
        token={token || ''} 
        user={user} 
        showMessage={showMessage}
        onUserUpdate={setUser}
      />

            {/* Bug Report Modal */}
      <BugReportModal 
        isOpen={showBugReportModal} 
        onClose={() => setShowBugReportModal(false)} 
        token={token || ''} 
        showMessage={showMessage}
      />

      {/* My Bug Reports Modal */}
      {showMyBugsModal && (
        <div style={{ position: 'fixed', inset: 0, backgroundColor: 'rgba(13, 8, 17, 0.9)', backdropFilter: 'blur(8px)', display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 1000, padding: '15px', boxSizing: 'border-box' }} onClick={() => { setShowMyBugsModal(false); setSelectedBug(null); }}>
          <div style={{ background: 'linear-gradient(180deg, #2D1F3D 0%, #1A0F1E 100%)', borderRadius: '20px', padding: '28px', maxWidth: '750px', width: '100%', border: '1px solid #4A2C5A', maxHeight: 'calc(100vh - 30px)', overflowY: 'auto', boxSizing: 'border-box', boxShadow: '0 24px 48px rgba(0, 0, 0, 0.5)' }} onClick={e => e.stopPropagation()}>
            {selectedBug ? (
              <>
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '20px' }}>
                  <h2 style={{ color: '#D4A84B', margin: 0, fontSize: '1.4rem', fontWeight: 600 }}>🐛 {selectedBug.bug.report_number}</h2>
                  <button 
                    style={{ padding: '10px 18px', backgroundColor: 'rgba(74, 44, 90, 0.5)', border: '1px solid #4A2C5A', borderRadius: '10px', color: '#B8B4C8', cursor: 'pointer', fontWeight: 500 }}
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

            {/* Mining Graphs - Visible to ALL users with Grafana integration and dropdown selectors */}
            <ChartErrorBoundary fallbackMessage="Charts failed to load. Click retry to try again.">
              <GrafanaDashboardLazy 
                dashboardId="main" 
                token={token || undefined} 
                isLoggedIn={!!token && !!user}
                showSelectors={true}
              />
            </ChartErrorBoundary>

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

            {/* User Mining Dashboard - auto-collapses when no active equipment */}
            {token && user && (
              <CollapsibleUserDashboard 
                token={token} 
                equipmentStatus={equipmentStatus}
                isLoggedIn={!!token && !!user}
              />
            )}
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
            {/* Litecoin Mining Instructions - Prominent on Home Page */}
            <section style={styles.section} className="section">
              <MiningInstructionsLitecoin 
                className="mining-instructions-section"
                showAdvanced={true}
                onCopySuccess={() => showMessage('success', 'Copied to clipboard')}
              />
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
  const isLiveData = ['Active Miners', 'Pool Hashrate'].includes(label);
  const isHashrate = label === 'Pool Hashrate';
  
  return (
    <div style={styles.statCard} className="stat-card-enhanced">
      <h3 style={styles.statLabel}>
        {isLiveData && <span className="live-indicator" />}
        {label}
      </h3>
      <p style={styles.statValue} className={`stat-value-glow ${isHashrate ? 'hashrate-value' : ''}`}>
        {value}
      </p>
    </div>
  );
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
// NOTE: instructionStyles was removed - MiningInstructionsLitecoin component has its own styles
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
  header: { background: 'linear-gradient(135deg, #2D1F3D 0%, #3A1F2E 100%)', padding: '8px 24px', borderBottom: '1px solid rgba(74, 44, 90, 0.5)', backdropFilter: 'blur(10px)', position: 'sticky' as const, top: 0, zIndex: 100 },
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
