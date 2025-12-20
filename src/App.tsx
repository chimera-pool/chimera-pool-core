import React, { useState, useEffect } from 'react';
import { LineChart, Line, AreaChart, Area, BarChart, Bar, PieChart, Pie, Cell, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from 'recharts';
import { ComposableMap, Geographies, Geography, Marker, ZoomableGroup } from 'react-simple-maps';

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

type MainView = 'dashboard' | 'community';

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
    if (token) fetchUserProfile();
  }, [token]);

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

  const showMessage = (type: 'success' | 'error', text: string) => {
    setMessage({ type, text });
    setTimeout(() => setMessage(null), 5000);
  };

  return (
    <div style={styles.container}>
      <header style={styles.header}>
        <div style={styles.headerContent}>
          <div>
            <h1 style={styles.title}>⛏️ Chimera Pool</h1>
            <p style={styles.subtitle}>BlockDAG Mining Pool</p>
          </div>
          {/* Main Navigation Tabs - Always visible */}
          <nav style={{ display: 'flex', gap: '5px', backgroundColor: '#0a0a15', borderRadius: '8px', padding: '4px', border: '2px solid #00d4ff' }}>
            <button 
              style={{ padding: '12px 24px', backgroundColor: mainView === 'dashboard' ? '#00d4ff' : 'transparent', border: 'none', color: mainView === 'dashboard' ? '#0a0a0f' : '#888', fontSize: '1rem', cursor: 'pointer', borderRadius: '6px', fontWeight: mainView === 'dashboard' ? 'bold' : 500 }}
              onClick={() => setMainView('dashboard')}
            >
              📊 Dashboard
            </button>
            <button 
              style={{ padding: '12px 24px', backgroundColor: mainView === 'community' ? '#00d4ff' : 'transparent', border: 'none', color: mainView === 'community' ? '#0a0a0f' : '#888', fontSize: '1rem', cursor: 'pointer', borderRadius: '6px', fontWeight: mainView === 'community' ? 'bold' : 500 }}
              onClick={() => setMainView('community')}
            >
              💬 Community
            </button>
          </nav>
          <div style={styles.authButtons}>
            {token && user ? (
              <div style={styles.userInfo}>
                <span style={styles.username}>👤 {user.username}</span>
                {user.is_admin && (
                  <button style={{...styles.authBtn, backgroundColor: '#4a1a6b', borderColor: '#9b59b6'}} onClick={() => setShowAdminPanel(true)}>
                    🛡️ Admin
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

      {authView && <AuthModal view={authView} setView={setAuthView} setToken={setToken} showMessage={showMessage} resetToken={resetToken} />}
      
      {showAdminPanel && token && <AdminPanel token={token} onClose={() => setShowAdminPanel(false)} showMessage={showMessage} />}

      <main style={mainView === 'community' ? { ...styles.main, maxWidth: '100%', padding: '0' } : styles.main}>
        {/* DASHBOARD VIEW */}
        {mainView === 'dashboard' && (
          <>
            {loading ? (
              <div style={styles.loading}>Loading pool statistics...</div>
            ) : stats ? (
              <div style={styles.statsGrid}>
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

            {/* User Dashboard - only shown when logged in */}
            {token && user && <UserDashboard token={token} />}
            {token && user && <UserMiningGraphs token={token} />}
            {token && user && <WalletManager token={token} showMessage={showMessage} />}

            {/* Global Miner Map - visible to all */}
            <GlobalMinerMap />
          </>
        )}

        {/* COMMUNITY VIEW - Full Page Experience */}
        {mainView === 'community' && (
          token && user ? (
            <CommunityPage token={token} user={user} showMessage={showMessage} />
          ) : (
            <div style={{ ...styles.main, textAlign: 'center', padding: '80px 20px' }}>
              <h2 style={{ color: '#00d4ff', marginBottom: '20px' }}>💬 Community</h2>
              <p style={{ color: '#888', marginBottom: '30px' }}>Please log in to access the community features.</p>
              <button style={{ ...styles.authBtn, ...styles.registerBtn }} onClick={() => setAuthView('login')}>
                Login to Join
              </button>
            </div>
          )
        )}

        {/* Dashboard-only sections */}
        {mainView === 'dashboard' && (
          <>
        <section style={styles.section}>
          <h2 style={styles.sectionTitle}>🔗 Connect Your Miner</h2>
          
          {/* Hybrid Protocol Banner */}
          <div style={instructionStyles.protocolBanner}>
            <div style={instructionStyles.protocolBadge}>
              <span style={instructionStyles.v2Badge}>✨ Stratum V2</span>
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

          {/* Connection Details - Simplified */}
          <div style={instructionStyles.connectionBox}>
            <h3 style={instructionStyles.connectionTitle}>⚡ Pool Connection Settings</h3>
            <div style={instructionStyles.copyableBox}>
              <div style={instructionStyles.detailRow}>
                <span style={instructionStyles.detailLabel}>Pool Address:</span>
                <code style={instructionStyles.detailCode}>stratum+tcp://206.162.80.230:3333</code>
              </div>
              <div style={instructionStyles.detailRow}>
                <span style={instructionStyles.detailLabel}>Username:</span>
                <code style={instructionStyles.detailCode}>your_email@example.com</code>
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
            <p style={instructionStyles.tipText}>💡 <strong>Tip:</strong> Use your Chimera Pool login credentials as username/password</p>
          </div>

          {/* Hardware-Specific Instructions */}
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
                <div style={instructionStyles.configBox}>
                  <strong>Configuration:</strong>
                  <pre style={instructionStyles.configCode}>
Pool URL: stratum+tcp://206.162.80.230:3333
Wallet:   your_email@example.com
Worker:   x100-rig1 (optional)
Password: your_account_password</pre>
                </div>
                <p style={instructionStyles.noteText}>
                  🔐 Your connection is automatically encrypted with Noise Protocol for maximum security.
                </p>
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
                
                <div style={instructionStyles.minerExample}>
                  <h5 style={instructionStyles.minerName}>🔹 lolMiner (Recommended)</h5>
                  <code style={instructionStyles.commandCode}>
lolminer --algo SCRPY --pool stratum+tcp://206.162.80.230:3333 --user your@email.com --pass yourpassword</code>
                </div>

                <div style={instructionStyles.minerExample}>
                  <h5 style={instructionStyles.minerName}>🔹 BzMiner</h5>
                  <code style={instructionStyles.commandCode}>
bzminer -a scrpy -p stratum+tcp://206.162.80.230:3333 -w your@email.com --pass yourpassword</code>
                </div>

                <div style={instructionStyles.minerExample}>
                  <h5 style={instructionStyles.minerName}>🔹 SRBMiner-MULTI</h5>
                  <code style={instructionStyles.commandCode}>
SRBMiner-MULTI --algorithm scrpy --pool 206.162.80.230:3333 --wallet your@email.com --password yourpassword</code>
                </div>
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
                <div style={instructionStyles.minerExample}>
                  <h5 style={instructionStyles.minerName}>🔹 CPUMiner-Multi</h5>
                  <code style={instructionStyles.commandCode}>
cpuminer -a scrpy -o stratum+tcp://206.162.80.230:3333 -u your@email.com -p yourpassword</code>
                </div>
              </div>
            </div>
          </div>

          {/* Troubleshooting */}
          <div style={instructionStyles.troubleshootBox}>
            <h3 style={instructionStyles.troubleshootTitle}>🔧 Troubleshooting</h3>
            <div style={instructionStyles.troubleshootItem}>
              <strong>❌ Connection Refused:</strong> Ensure port 3333 is not blocked by your firewall or router.
            </div>
            <div style={instructionStyles.troubleshootItem}>
              <strong>❌ Authentication Failed:</strong> Double-check your email and password match your Chimera Pool account.
            </div>
            <div style={instructionStyles.troubleshootItem}>
              <strong>❌ Shares Rejected:</strong> Make sure you're using the correct algorithm (scrpy-variant). Update your miner software.
            </div>
            <div style={instructionStyles.troubleshootItem}>
              <strong>❌ No Payouts:</strong> Verify you've added a valid BDAG wallet address in your account settings.
            </div>
            <div style={instructionStyles.troubleshootItem}>
              <strong>💡 Need Help?</strong> Join our Community chat for real-time support from other miners!
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

// User Dashboard Component - shows miners, hashrate, shares, payouts
interface UserDashboardProps {
  token: string;
}

interface Miner {
  id: number;
  name: string;
  hashrate: number;
  is_active: boolean;
  last_seen: string;
  shares_submitted: number;
  valid_shares: number;
  invalid_shares: number;
}

interface UserStats {
  total_hashrate: number;
  total_earnings: number;
  pending_payout: number;
  total_shares: number;
  valid_shares: number;
  invalid_shares: number;
  success_rate: number;
  miners: Miner[];
  recent_payouts: any[];
}

function UserDashboard({ token }: UserDashboardProps) {
  const [stats, setStats] = useState<UserStats | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetchUserStats();
    const interval = setInterval(fetchUserStats, 30000);
    return () => clearInterval(interval);
  }, [token]);

  const fetchUserStats = async () => {
    try {
      const [minersRes, payoutsRes] = await Promise.all([
        fetch('/api/v1/user/miners', { headers: { 'Authorization': `Bearer ${token}` } }),
        fetch('/api/v1/user/payouts', { headers: { 'Authorization': `Bearer ${token}` } })
      ]);

      const miners = minersRes.ok ? await minersRes.json() : { miners: [] };
      const payouts = payoutsRes.ok ? await payoutsRes.json() : { payouts: [] };

      const minerList: Miner[] = miners.miners || [];
      const totalHashrate = minerList.reduce((sum: number, m: Miner) => sum + (m.is_active ? m.hashrate : 0), 0);
      const totalShares = minerList.reduce((sum: number, m: Miner) => sum + (m.shares_submitted || 0), 0);
      const validShares = minerList.reduce((sum: number, m: Miner) => sum + (m.valid_shares || 0), 0);
      const invalidShares = minerList.reduce((sum: number, m: Miner) => sum + (m.invalid_shares || 0), 0);
      const successRate = totalShares > 0 ? (validShares / totalShares) * 100 : 0;

      setStats({
        total_hashrate: totalHashrate,
        total_earnings: 0,
        pending_payout: 0,
        total_shares: totalShares,
        valid_shares: validShares,
        invalid_shares: invalidShares,
        success_rate: successRate,
        miners: minerList,
        recent_payouts: payouts.payouts || []
      });
    } catch (error) {
      console.error('Failed to fetch user stats:', error);
    } finally {
      setLoading(false);
    }
  };

  if (loading) {
    return (
      <section style={styles.section}>
        <h2 style={styles.sectionTitle}>📈 Your Mining Dashboard</h2>
        <div style={styles.loading}>Loading your mining stats...</div>
      </section>
    );
  }

  if (!stats) {
    return null;
  }

  return (
    <section style={{...styles.section, marginBottom: '30px'}}>
      <h2 style={styles.sectionTitle}>📈 Your Mining Dashboard</h2>
      
      {/* Summary Stats */}
      <div style={dashStyles.statsRow}>
        <div style={dashStyles.statBox}>
          <span style={dashStyles.statLabel}>Total Hashrate</span>
          <span style={dashStyles.statValue}>{formatHashrate(stats.total_hashrate)}</span>
        </div>
        <div style={dashStyles.statBox}>
          <span style={dashStyles.statLabel}>Active Miners</span>
          <span style={dashStyles.statValue}>{stats.miners.filter(m => m.is_active).length} / {stats.miners.length}</span>
        </div>
        <div style={dashStyles.statBox}>
          <span style={dashStyles.statLabel}>Total Shares</span>
          <span style={dashStyles.statValue}>{stats.total_shares.toLocaleString()}</span>
        </div>
        <div style={dashStyles.statBox}>
          <span style={dashStyles.statLabel}>Success Rate</span>
          <span style={{...dashStyles.statValue, color: stats.success_rate >= 95 ? '#4ade80' : stats.success_rate >= 80 ? '#fbbf24' : '#f87171'}}>
            {stats.success_rate.toFixed(2)}%
          </span>
        </div>
      </div>

      {/* Miners Table */}
      <h3 style={dashStyles.subTitle}>⛏️ Your Miners ({stats.miners.length})</h3>
      {stats.miners.length === 0 ? (
        <div style={dashStyles.emptyState}>
          <p>No miners connected yet. Connect your miner using the stratum URL below!</p>
        </div>
      ) : (
        <div style={dashStyles.tableWrapper}>
          <table style={dashStyles.table}>
            <thead>
              <tr>
                <th style={dashStyles.th}>Miner Name</th>
                <th style={dashStyles.th}>Status</th>
                <th style={dashStyles.th}>Hashrate</th>
                <th style={dashStyles.th}>Valid Shares</th>
                <th style={dashStyles.th}>Invalid Shares</th>
                <th style={dashStyles.th}>Success Rate</th>
                <th style={dashStyles.th}>Last Seen</th>
              </tr>
            </thead>
            <tbody>
              {stats.miners.map(miner => {
                const minerSuccessRate = miner.shares_submitted > 0 
                  ? ((miner.valid_shares || 0) / miner.shares_submitted) * 100 
                  : 0;
                return (
                  <tr key={miner.id} style={dashStyles.tr}>
                    <td style={dashStyles.td}>{miner.name}</td>
                    <td style={dashStyles.td}>
                      <span style={miner.is_active ? dashStyles.online : dashStyles.offline}>
                        {miner.is_active ? '🟢 Online' : '🔴 Offline'}
                      </span>
                    </td>
                    <td style={dashStyles.td}>{formatHashrate(miner.hashrate)}</td>
                    <td style={dashStyles.td}>{(miner.valid_shares || 0).toLocaleString()}</td>
                    <td style={dashStyles.td}>{(miner.invalid_shares || 0).toLocaleString()}</td>
                    <td style={dashStyles.td}>
                      <span style={{color: minerSuccessRate >= 95 ? '#4ade80' : minerSuccessRate >= 80 ? '#fbbf24' : '#f87171'}}>
                        {minerSuccessRate.toFixed(2)}%
                      </span>
                    </td>
                    <td style={dashStyles.td}>{new Date(miner.last_seen).toLocaleString()}</td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}

      {/* Recent Payouts */}
      {stats.recent_payouts.length > 0 && (
        <>
          <h3 style={dashStyles.subTitle}>💰 Recent Payouts</h3>
          <div style={dashStyles.tableWrapper}>
            <table style={dashStyles.table}>
              <thead>
                <tr>
                  <th style={dashStyles.th}>Amount</th>
                  <th style={dashStyles.th}>Status</th>
                  <th style={dashStyles.th}>TX Hash</th>
                  <th style={dashStyles.th}>Date</th>
                </tr>
              </thead>
              <tbody>
                {stats.recent_payouts.slice(0, 5).map((payout: any, idx: number) => (
                  <tr key={idx} style={dashStyles.tr}>
                    <td style={dashStyles.td}>{payout.amount} BDAG</td>
                    <td style={dashStyles.td}>
                      <span style={payout.status === 'completed' ? dashStyles.online : dashStyles.offline}>
                        {payout.status}
                      </span>
                    </td>
                    <td style={dashStyles.td}>
                      {payout.tx_hash ? `${payout.tx_hash.slice(0, 10)}...` : '-'}
                    </td>
                    <td style={dashStyles.td}>{new Date(payout.created_at).toLocaleString()}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </>
      )}
    </section>
  );
}

const dashStyles: { [key: string]: React.CSSProperties } = {
  statsRow: { display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(150px, 1fr))', gap: '15px', marginBottom: '25px' },
  statBox: { backgroundColor: '#0a0a15', padding: '20px', borderRadius: '8px', textAlign: 'center', border: '1px solid #2a2a4a' },
  statLabel: { display: 'block', color: '#888', fontSize: '0.85rem', marginBottom: '8px', textTransform: 'uppercase' },
  statValue: { display: 'block', color: '#00d4ff', fontSize: '1.4rem', fontWeight: 'bold' },
  subTitle: { color: '#00d4ff', fontSize: '1.1rem', marginTop: '25px', marginBottom: '15px' },
  emptyState: { backgroundColor: '#0a0a15', padding: '30px', borderRadius: '8px', textAlign: 'center', color: '#888' },
  tableWrapper: { overflowX: 'auto', backgroundColor: '#0a0a15', borderRadius: '8px' },
  table: { width: '100%', borderCollapse: 'collapse' },
  th: { padding: '12px 15px', textAlign: 'left', borderBottom: '2px solid #2a2a4a', color: '#00d4ff', fontSize: '0.8rem', textTransform: 'uppercase', whiteSpace: 'nowrap' },
  tr: { borderBottom: '1px solid #1a1a2e' },
  td: { padding: '12px 15px', color: '#e0e0e0', fontSize: '0.9rem' },
  online: { color: '#4ade80' },
  offline: { color: '#888' },
};

// User Mining Graphs Component
type TimeRange = '1h' | '6h' | '24h' | '7d' | '30d' | '3m' | '6m' | '1y' | 'all';

function UserMiningGraphs({ token }: { token: string }) {
  const [timeRange, setTimeRange] = useState<TimeRange>('24h');
  const [hashrateData, setHashrateData] = useState<any[]>([]);
  const [sharesData, setSharesData] = useState<any[]>([]);
  const [earningsData, setEarningsData] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetchAllData();
  }, [timeRange]);

  const fetchAllData = async () => {
    setLoading(true);
    try {
      const headers = { 'Authorization': `Bearer ${token}` };
      const [hashRes, sharesRes, earningsRes] = await Promise.all([
        fetch(`/api/v1/user/stats/hashrate?range=${timeRange}`, { headers }),
        fetch(`/api/v1/user/stats/shares?range=${timeRange}`, { headers }),
        fetch(`/api/v1/user/stats/earnings?range=${timeRange === '1h' || timeRange === '6h' ? '24h' : timeRange}`, { headers })
      ]);

      if (hashRes.ok) {
        const data = await hashRes.json();
        setHashrateData(data.data?.map((d: any) => ({
          ...d,
          time: new Date(d.time).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
          hashrateMH: d.hashrate / 1000000
        })) || []);
      }
      if (sharesRes.ok) {
        const data = await sharesRes.json();
        setSharesData(data.data?.map((d: any) => ({
          ...d,
          time: new Date(d.time).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
        })) || []);
      }
      if (earningsRes.ok) {
        const data = await earningsRes.json();
        setEarningsData(data.data?.map((d: any) => ({
          ...d,
          time: new Date(d.time).toLocaleDateString([], { month: 'short', day: 'numeric' })
        })) || []);
      }
    } catch (error) {
      console.error('Failed to fetch graph data:', error);
    } finally {
      setLoading(false);
    }
  };

  const COLORS = ['#00d4ff', '#9b59b6', '#4ade80', '#f59e0b', '#ef4444'];

  return (
    <section style={graphStyles.section}>
      <div style={graphStyles.header}>
        <h2 style={graphStyles.title}>📊 Mining Statistics</h2>
        <div style={graphStyles.timeSelector}>
          {([
            { value: '1h', label: '1H' },
            { value: '6h', label: '6H' },
            { value: '24h', label: '24H' },
            { value: '7d', label: '7D' },
            { value: '30d', label: '30D' },
            { value: '3m', label: '3M' },
            { value: '6m', label: '6M' },
            { value: '1y', label: '1Y' },
            { value: 'all', label: 'All' }
          ] as { value: TimeRange; label: string }[]).map(({ value, label }) => (
            <button
              key={value}
              style={{...graphStyles.timeBtn, ...(timeRange === value ? graphStyles.timeBtnActive : {})}}
              onClick={() => setTimeRange(value)}
            >
              {label}
            </button>
          ))}
        </div>
      </div>

      {loading ? (
        <div style={graphStyles.loading}>Loading charts...</div>
      ) : (
        <div style={graphStyles.chartsGrid}>
          {/* Hashrate Chart */}
          <div style={graphStyles.chartCard}>
            <h3 style={graphStyles.chartTitle}>⚡ Hashrate History</h3>
            <ResponsiveContainer width="100%" height={250}>
              <AreaChart data={hashrateData}>
                <defs>
                  <linearGradient id="hashGradient" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="#00d4ff" stopOpacity={0.3}/>
                    <stop offset="95%" stopColor="#00d4ff" stopOpacity={0}/>
                  </linearGradient>
                </defs>
                <CartesianGrid strokeDasharray="3 3" stroke="#2a2a4a" />
                <XAxis dataKey="time" stroke="#888" fontSize={12} />
                <YAxis stroke="#888" fontSize={12} tickFormatter={(v) => `${v.toFixed(0)} MH/s`} />
                <Tooltip 
                  contentStyle={{ backgroundColor: '#1a1a2e', border: '1px solid #2a2a4a', borderRadius: '8px' }}
                  labelStyle={{ color: '#00d4ff' }}
                  formatter={(value: number) => [`${value.toFixed(2)} MH/s`, 'Hashrate']}
                />
                <Area type="monotone" dataKey="hashrateMH" stroke="#00d4ff" fill="url(#hashGradient)" strokeWidth={2} />
              </AreaChart>
            </ResponsiveContainer>
          </div>

          {/* Shares Chart */}
          <div style={graphStyles.chartCard}>
            <h3 style={graphStyles.chartTitle}>📦 Shares Submitted</h3>
            <ResponsiveContainer width="100%" height={250}>
              <BarChart data={sharesData}>
                <CartesianGrid strokeDasharray="3 3" stroke="#2a2a4a" />
                <XAxis dataKey="time" stroke="#888" fontSize={12} />
                <YAxis stroke="#888" fontSize={12} />
                <Tooltip 
                  contentStyle={{ backgroundColor: '#1a1a2e', border: '1px solid #2a2a4a', borderRadius: '8px' }}
                  labelStyle={{ color: '#00d4ff' }}
                />
                <Legend />
                <Bar dataKey="validShares" name="Valid" fill="#4ade80" radius={[4, 4, 0, 0]} />
                <Bar dataKey="invalidShares" name="Invalid" fill="#ef4444" radius={[4, 4, 0, 0]} />
              </BarChart>
            </ResponsiveContainer>
          </div>

          {/* Acceptance Rate Chart */}
          <div style={graphStyles.chartCard}>
            <h3 style={graphStyles.chartTitle}>✅ Acceptance Rate</h3>
            <ResponsiveContainer width="100%" height={250}>
              <LineChart data={sharesData}>
                <CartesianGrid strokeDasharray="3 3" stroke="#2a2a4a" />
                <XAxis dataKey="time" stroke="#888" fontSize={12} />
                <YAxis stroke="#888" fontSize={12} domain={[90, 100]} tickFormatter={(v) => `${v}%`} />
                <Tooltip 
                  contentStyle={{ backgroundColor: '#1a1a2e', border: '1px solid #2a2a4a', borderRadius: '8px' }}
                  labelStyle={{ color: '#00d4ff' }}
                  formatter={(value: number) => [`${value.toFixed(2)}%`, 'Rate']}
                />
                <Line type="monotone" dataKey="acceptanceRate" stroke="#4ade80" strokeWidth={2} dot={false} />
              </LineChart>
            </ResponsiveContainer>
          </div>

          {/* Earnings Chart */}
          <div style={graphStyles.chartCard}>
            <h3 style={graphStyles.chartTitle}>💰 Earnings History</h3>
            <ResponsiveContainer width="100%" height={250}>
              <AreaChart data={earningsData}>
                <defs>
                  <linearGradient id="earnGradient" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="#9b59b6" stopOpacity={0.3}/>
                    <stop offset="95%" stopColor="#9b59b6" stopOpacity={0}/>
                  </linearGradient>
                </defs>
                <CartesianGrid strokeDasharray="3 3" stroke="#2a2a4a" />
                <XAxis dataKey="time" stroke="#888" fontSize={12} />
                <YAxis stroke="#888" fontSize={12} tickFormatter={(v) => `${v.toFixed(2)}`} />
                <Tooltip 
                  contentStyle={{ backgroundColor: '#1a1a2e', border: '1px solid #2a2a4a', borderRadius: '8px' }}
                  labelStyle={{ color: '#9b59b6' }}
                  formatter={(value: number) => [`${value.toFixed(4)} BDAG`, 'Cumulative']}
                />
                <Area type="monotone" dataKey="cumulative" stroke="#9b59b6" fill="url(#earnGradient)" strokeWidth={2} />
              </AreaChart>
            </ResponsiveContainer>
          </div>
        </div>
      )}
    </section>
  );
}

const graphStyles: { [key: string]: React.CSSProperties } = {
  section: { background: 'linear-gradient(135deg, #1a1a2e 0%, #0f0f1a 100%)', borderRadius: '12px', padding: '24px', border: '1px solid #2a2a4a', marginBottom: '20px' },
  header: { display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '20px', flexWrap: 'wrap', gap: '15px' },
  title: { fontSize: '1.3rem', color: '#00d4ff', margin: 0 },
  timeSelector: { display: 'flex', gap: '8px', flexWrap: 'wrap' },
  timeBtn: { padding: '6px 12px', backgroundColor: '#0a0a15', border: '1px solid #2a2a4a', borderRadius: '6px', color: '#888', cursor: 'pointer', fontSize: '0.85rem' },
  timeBtnActive: { backgroundColor: '#00d4ff', color: '#0a0a0f', borderColor: '#00d4ff' },
  loading: { textAlign: 'center', padding: '60px', color: '#00d4ff' },
  chartsGrid: { display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(400px, 1fr))', gap: '20px' },
  chartCard: { backgroundColor: '#0a0a15', borderRadius: '10px', padding: '20px', border: '1px solid #2a2a4a' },
  chartTitle: { color: '#e0e0e0', fontSize: '1rem', marginTop: 0, marginBottom: '15px' },
};

// Global Miner Map Component
interface MinerLocation {
  city: string;
  country: string;
  countryCode: string;
  continent: string;
  lat: number;
  lng: number;
  minerCount: number;
  hashrate: number;
  activeCount: number;
  isActive: boolean;
}

interface LocationStats {
  totalMiners: number;
  totalCountries: number;
  activeMiners: number;
  topCountries: { country: string; countryCode: string; minerCount: number; hashrate: number }[];
  continentBreakdown: { continent: string; minerCount: number; hashrate: number }[];
}

function GlobalMinerMap() {
  const [locations, setLocations] = useState<MinerLocation[]>([]);
  const [stats, setStats] = useState<LocationStats | null>(null);
  const [loading, setLoading] = useState(true);
  const [hoveredLocation, setHoveredLocation] = useState<MinerLocation | null>(null);
  const [tooltipPos, setTooltipPos] = useState({ x: 0, y: 0 });

  useEffect(() => {
    fetchData();
    const interval = setInterval(fetchData, 30000);
    return () => clearInterval(interval);
  }, []);

  const fetchData = async () => {
    try {
      const [locRes, statsRes] = await Promise.all([
        fetch('/api/v1/miners/locations'),
        fetch('/api/v1/miners/locations/stats')
      ]);

      if (locRes.ok) {
        const data = await locRes.json();
        setLocations(data.locations || []);
      }
      if (statsRes.ok) {
        const data = await statsRes.json();
        setStats(data);
      }
    } catch (error) {
      console.error('Failed to fetch miner locations:', error);
    } finally {
      setLoading(false);
    }
  };

  const getMarkerSize = (count: number) => {
    if (count >= 10) return 12;
    if (count >= 5) return 9;
    if (count >= 2) return 7;
    return 5;
  };

  const handleMouseEnter = (location: MinerLocation, e: React.MouseEvent) => {
    setHoveredLocation(location);
    setTooltipPos({ x: e.clientX, y: e.clientY });
  };

  const CONTINENT_COLORS: { [key: string]: string } = {
    'North America': '#00d4ff',
    'South America': '#4ade80',
    'Europe': '#9b59b6',
    'Asia': '#f59e0b',
    'Africa': '#ef4444',
    'Oceania': '#3b82f6',
    'Unknown': '#888888'
  };

  return (
    <section style={mapStyles.section}>
      <div style={mapStyles.header}>
        <h2 style={mapStyles.title}>🌍 Global Miner Network</h2>
        <div style={mapStyles.statsRow}>
          <div style={mapStyles.statBadge}>
            <span style={mapStyles.statNumber}>{stats?.totalMiners || 0}</span>
            <span style={mapStyles.statLabel}>Total Miners</span>
          </div>
          <div style={mapStyles.statBadge}>
            <span style={mapStyles.statNumber}>{stats?.activeMiners || 0}</span>
            <span style={mapStyles.statLabel}>Active</span>
          </div>
          <div style={mapStyles.statBadge}>
            <span style={mapStyles.statNumber}>{stats?.totalCountries || 0}</span>
            <span style={mapStyles.statLabel}>Countries</span>
          </div>
        </div>
      </div>

      {loading ? (
        <div style={mapStyles.loading}>Loading global miner network...</div>
      ) : (
        <div style={mapStyles.mapContainer}>
          <div style={mapStyles.mapWrapper}>
            <ComposableMap
              projection="geoMercator"
              projectionConfig={{ scale: 120, center: [0, 20] }}
              style={{ width: '100%', height: '100%', backgroundColor: '#0a0a15' }}
            >
              <ZoomableGroup>
                <Geographies geography={geoUrl}>
                  {({ geographies }) =>
                    geographies.map((geo) => (
                      <Geography
                        key={geo.rsmKey}
                        geography={geo}
                        fill="#1a1a2e"
                        stroke="#2a2a4a"
                        strokeWidth={0.5}
                        style={{
                          default: { outline: 'none' },
                          hover: { fill: '#2a2a4a', outline: 'none' },
                          pressed: { outline: 'none' }
                        }}
                      />
                    ))
                  }
                </Geographies>
                {locations.map((location, idx) => (
                  <Marker
                    key={idx}
                    coordinates={[location.lng, location.lat]}
                    onMouseEnter={(e) => handleMouseEnter(location, e as any)}
                    onMouseLeave={() => setHoveredLocation(null)}
                  >
                    <circle
                      r={getMarkerSize(location.minerCount)}
                      fill={location.isActive ? '#00d4ff' : '#888888'}
                      fillOpacity={0.8}
                      stroke={location.isActive ? '#00d4ff' : '#888888'}
                      strokeWidth={2}
                      strokeOpacity={0.4}
                      style={{ cursor: 'pointer' }}
                    >
                      {location.isActive && (
                        <animate
                          attributeName="r"
                          from={getMarkerSize(location.minerCount)}
                          to={getMarkerSize(location.minerCount) + 3}
                          dur="1.5s"
                          repeatCount="indefinite"
                        />
                      )}
                    </circle>
                  </Marker>
                ))}
              </ZoomableGroup>
            </ComposableMap>

            {hoveredLocation && (
              <div style={{
                ...mapStyles.tooltip,
                left: tooltipPos.x + 10,
                top: tooltipPos.y - 60
              }}>
                <div style={mapStyles.tooltipCity}>{hoveredLocation.city}</div>
                <div style={mapStyles.tooltipCountry}>{hoveredLocation.country}</div>
                <div style={mapStyles.tooltipStats}>
                  <span>⛏️ {hoveredLocation.minerCount} miners</span>
                  <span>⚡ {formatHashrate(hoveredLocation.hashrate)}</span>
                  <span>{hoveredLocation.activeCount} active</span>
                </div>
              </div>
            )}
          </div>

          <div style={mapStyles.sidebar}>
            <div style={mapStyles.sidebarSection}>
              <h4 style={mapStyles.sidebarTitle}>🏆 Top Countries</h4>
              {stats?.topCountries?.slice(0, 5).map((country, idx) => (
                <div key={idx} style={mapStyles.countryRow}>
                  <span style={mapStyles.countryRank}>#{idx + 1}</span>
                  <span style={mapStyles.countryName}>{country.country}</span>
                  <span style={mapStyles.countryMiners}>{country.minerCount}</span>
                </div>
              ))}
            </div>

            <div style={mapStyles.sidebarSection}>
              <h4 style={mapStyles.sidebarTitle}>🌐 By Continent</h4>
              {stats?.continentBreakdown?.map((cont, idx) => (
                <div key={idx} style={mapStyles.continentRow}>
                  <span style={{ ...mapStyles.continentDot, backgroundColor: CONTINENT_COLORS[cont.continent] || '#888' }}></span>
                  <span style={mapStyles.continentName}>{cont.continent}</span>
                  <span style={mapStyles.continentMiners}>{cont.minerCount}</span>
                </div>
              ))}
            </div>
          </div>
        </div>
      )}
    </section>
  );
}

const mapStyles: { [key: string]: React.CSSProperties } = {
  section: { background: 'linear-gradient(135deg, #1a1a2e 0%, #0f0f1a 100%)', borderRadius: '12px', padding: '24px', border: '1px solid #2a2a4a', marginBottom: '20px' },
  header: { display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '20px', flexWrap: 'wrap', gap: '15px' },
  title: { fontSize: '1.3rem', color: '#00d4ff', margin: 0 },
  statsRow: { display: 'flex', gap: '15px' },
  statBadge: { display: 'flex', flexDirection: 'column', alignItems: 'center', backgroundColor: '#0a0a15', padding: '10px 20px', borderRadius: '8px', border: '1px solid #2a2a4a' },
  statNumber: { color: '#00d4ff', fontSize: '1.4rem', fontWeight: 'bold' },
  statLabel: { color: '#888', fontSize: '0.75rem', textTransform: 'uppercase' },
  loading: { textAlign: 'center', padding: '100px', color: '#00d4ff' },
  mapContainer: { display: 'flex', gap: '20px', flexWrap: 'wrap' },
  mapWrapper: { flex: '1 1 600px', height: '400px', backgroundColor: '#0a0a15', borderRadius: '10px', border: '1px solid #2a2a4a', overflow: 'hidden', position: 'relative' },
  tooltip: { position: 'fixed', backgroundColor: '#1a1a2e', border: '1px solid #00d4ff', borderRadius: '8px', padding: '12px', zIndex: 1000, pointerEvents: 'none' },
  tooltipCity: { color: '#00d4ff', fontWeight: 'bold', fontSize: '1rem' },
  tooltipCountry: { color: '#888', fontSize: '0.85rem', marginBottom: '8px' },
  tooltipStats: { display: 'flex', flexDirection: 'column', gap: '4px', color: '#e0e0e0', fontSize: '0.85rem' },
  sidebar: { flex: '0 0 250px', display: 'flex', flexDirection: 'column', gap: '20px' },
  sidebarSection: { backgroundColor: '#0a0a15', borderRadius: '10px', padding: '15px', border: '1px solid #2a2a4a' },
  sidebarTitle: { color: '#00d4ff', fontSize: '0.95rem', margin: '0 0 12px 0' },
  countryRow: { display: 'flex', alignItems: 'center', gap: '10px', padding: '6px 0', borderBottom: '1px solid #1a1a2e' },
  countryRank: { color: '#888', fontSize: '0.8rem', width: '25px' },
  countryName: { flex: 1, color: '#e0e0e0', fontSize: '0.9rem' },
  countryMiners: { color: '#00d4ff', fontWeight: 'bold' },
  continentRow: { display: 'flex', alignItems: 'center', gap: '10px', padding: '6px 0' },
  continentDot: { width: '10px', height: '10px', borderRadius: '50%' },
  continentName: { flex: 1, color: '#e0e0e0', fontSize: '0.9rem' },
  continentMiners: { color: '#888' },
};

// Community Section Component
interface Channel { id: number; name: string; description: string; type: string; isReadOnly: boolean; adminOnlyPost: boolean; }
interface ChannelCategory { id: number; name: string; channels: Channel[]; }
interface ChatMessage { id: number; content: string; isEdited: boolean; createdAt: string; user: { id: number; username: string; badgeIcon: string; badgeColor: string; }; replyToId?: number; }
interface ForumCategory { id: number; name: string; description: string; icon: string; postCount: number; }
interface ForumPost { id: number; title: string; preview: string; tags: string[]; viewCount: number; replyCount: number; upvotes: number; isPinned: boolean; isLocked: boolean; createdAt: string; author: { id: number; username: string; badgeIcon: string; }; }
interface OnlineUser { id: number; username: string; status: string; badgeIcon: string; }

// Full-Page Community Component
function CommunityPage({ token, user, showMessage }: { token: string; user: any; showMessage: (type: 'success' | 'error', text: string) => void }) {
  const [activeView, setActiveView] = useState<'chat' | 'forums' | 'leaderboard'>('chat');
  const [categories, setCategories] = useState<ChannelCategory[]>([]);
  const [selectedChannel, setSelectedChannel] = useState<Channel | null>(null);
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [newMessage, setNewMessage] = useState('');
  const [onlineUsers, setOnlineUsers] = useState<OnlineUser[]>([]);
  const [collapsedCategories, setCollapsedCategories] = useState<Set<number>>(new Set());
  const [loading, setLoading] = useState(true);
  const [leaderboard, setLeaderboard] = useState<any[]>([]);
  const [leaderboardType, setLeaderboardType] = useState('hashrate');
  
  // Admin channel management state
  const [showCreateChannel, setShowCreateChannel] = useState(false);
  const [showCreateCategory, setShowCreateCategory] = useState(false);
  const [channelForm, setChannelForm] = useState({ name: '', description: '', category_id: '', type: 'text', is_read_only: false, admin_only_post: false });
  const [categoryForm, setCategoryForm] = useState({ name: '', description: '' });

  const isAdmin = user?.is_admin || user?.role === 'admin' || user?.role === 'super_admin';
  const isModerator = isAdmin || user?.role === 'moderator';

  const fetchChannels = async () => {
    console.log('fetchChannels called');
    try {
      const res = await fetch('/api/v1/community/channels', { headers: { Authorization: `Bearer ${token}` } });
      console.log('fetchChannels response:', res.status);
      if (res.ok) {
        const data = await res.json();
        console.log('fetchChannels data:', data);
        setCategories(data.categories || []);
        if (data.categories?.length > 0 && data.categories[0].channels?.length > 0) {
          setSelectedChannel(data.categories[0].channels[0]);
        }
      } else {
        console.error('fetchChannels failed:', res.status, await res.text());
      }
    } catch (e) { console.error('fetchChannels error:', e); }
    setLoading(false);
  };

  useEffect(() => {
    fetchChannels();
    fetchOnlineUsers();
    const interval = setInterval(fetchOnlineUsers, 30000);
    return () => clearInterval(interval);
  }, []);

  useEffect(() => {
    if (selectedChannel) fetchMessages();
  }, [selectedChannel]);

  useEffect(() => {
    if (activeView === 'leaderboard') fetchLeaderboard();
  }, [activeView, leaderboardType]);

  const fetchMessages = async () => {
    if (!selectedChannel) return;
    try {
      const res = await fetch(`/api/v1/community/channels/${selectedChannel.id}/messages`, { headers: { Authorization: `Bearer ${token}` } });
      if (res.ok) {
        const data = await res.json();
        setMessages(data.messages || []);
      }
    } catch (e) { console.error(e); }
  };

  const fetchOnlineUsers = async () => {
    try {
      const res = await fetch('/api/v1/community/online-users', { headers: { Authorization: `Bearer ${token}` } });
      if (res.ok) {
        const data = await res.json();
        setOnlineUsers(data.users || []);
      }
    } catch (e) { console.error(e); }
  };

  const fetchLeaderboard = async () => {
    try {
      const res = await fetch(`/api/v1/community/leaderboard?type=${leaderboardType}`, { headers: { Authorization: `Bearer ${token}` } });
      if (res.ok) {
        const data = await res.json();
        setLeaderboard(data.leaderboard || []);
      }
    } catch (e) { console.error(e); }
  };

  const sendMessage = async () => {
    if (!newMessage.trim() || !selectedChannel) return;
    try {
      const res = await fetch(`/api/v1/community/channels/${selectedChannel.id}/messages`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
        body: JSON.stringify({ content: newMessage })
      });
      if (res.ok) {
        setNewMessage('');
        fetchMessages();
      }
    } catch (e) { console.error(e); }
  };

  const toggleCategory = (catId: number) => {
    const newCollapsed = new Set(collapsedCategories);
    if (newCollapsed.has(catId)) newCollapsed.delete(catId);
    else newCollapsed.add(catId);
    setCollapsedCategories(newCollapsed);
  };

  const handleCreateChannel = async () => {
    if (!channelForm.name || !channelForm.category_id) {
      showMessage('error', 'Channel name and category are required');
      return;
    }
    try {
      const response = await fetch('/api/v1/admin/community/channels', {
        method: 'POST',
        headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
        body: JSON.stringify(channelForm)
      });
      if (response.ok) {
        showMessage('success', 'Channel created successfully');
        setShowCreateChannel(false);
        setChannelForm({ name: '', description: '', category_id: '', type: 'text', is_read_only: false, admin_only_post: false });
        fetchChannels();
      } else {
        const data = await response.json();
        showMessage('error', data.error || 'Failed to create channel');
      }
    } catch (error) {
      showMessage('error', 'Network error');
    }
  };

  const handleCreateCategory = async () => {
    if (!categoryForm.name) {
      showMessage('error', 'Category name is required');
      return;
    }
    try {
      const response = await fetch('/api/v1/admin/community/channel-categories', {
        method: 'POST',
        headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
        body: JSON.stringify(categoryForm)
      });
      if (response.ok) {
        showMessage('success', 'Category created successfully');
        setShowCreateCategory(false);
        setCategoryForm({ name: '', description: '' });
        fetchChannels();
      } else {
        const data = await response.json();
        showMessage('error', data.error || 'Failed to create category');
      }
    } catch (error) {
      showMessage('error', 'Network error');
    }
  };

  const formatTime = (dateStr: string) => {
    const date = new Date(dateStr);
    return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
  };

  if (loading) return <div style={communityPageStyles.loading}>Loading community...</div>;

  return (
    <div style={communityPageStyles.pageContainer}>
      {/* Left Sidebar - Channels */}
      <div style={communityPageStyles.leftSidebar}>
        <div style={communityPageStyles.sidebarHeader}>
          <span>💬 Channels</span>
          {isModerator && (
            <div style={{ display: 'flex', gap: '5px' }}>
              <button style={communityPageStyles.addBtn} onClick={() => setShowCreateCategory(true)} title="New Category">📁+</button>
              <button style={communityPageStyles.addBtn} onClick={() => { setShowCreateChannel(true); setChannelForm({ ...channelForm, category_id: categories[0]?.id?.toString() || '' }); }} title="New Channel">💬+</button>
            </div>
          )}
        </div>
        
        {categories.length === 0 ? (
          <div style={communityPageStyles.emptyState}>
            <p>No channels yet.</p>
            {isModerator && <p style={{ color: '#00d4ff', fontSize: '0.85rem' }}>Click 📁+ to create a category first.</p>}
          </div>
        ) : (
          categories.map(cat => (
            <div key={cat.id} style={communityPageStyles.category}>
              <div style={communityPageStyles.categoryHeader} onClick={() => toggleCategory(cat.id)}>
                <span>{collapsedCategories.has(cat.id) ? '▶' : '▼'}</span>
                <span style={communityPageStyles.categoryName}>{cat.name}</span>
              </div>
              {!collapsedCategories.has(cat.id) && cat.channels?.map(ch => (
                <div
                  key={ch.id}
                  onClick={() => setSelectedChannel(ch)}
                  style={{ ...communityPageStyles.channel, ...(selectedChannel?.id === ch.id ? communityPageStyles.channelActive : {}) }}
                >
                  <span style={communityPageStyles.channelHash}>#</span>
                  <span>{ch.name}</span>
                  {ch.type === 'announcement' && <span style={communityPageStyles.channelBadge}>📢</span>}
                  {ch.type === 'regional' && <span style={communityPageStyles.channelBadge}>🌍</span>}
                </div>
              ))}
            </div>
          ))
        )}

        {/* Online Users */}
        <div style={communityPageStyles.onlineSection}>
          <div style={communityPageStyles.onlineHeader}>Online — {onlineUsers.length}</div>
          {onlineUsers.slice(0, 15).map(u => (
            <div key={u.id} style={communityPageStyles.onlineUser}>
              <span style={communityPageStyles.onlineIndicator}></span>
              <span>{u.badgeIcon}</span>
              <span style={communityPageStyles.onlineUsername}>{u.username}</span>
            </div>
          ))}
        </div>
      </div>

      {/* Main Content Area */}
      <div style={communityPageStyles.mainContent}>
        {/* Secondary Navigation */}
        <div style={communityPageStyles.secondaryNav}>
          {[
            { key: 'chat', label: '💬 Chat' },
            { key: 'forums', label: '📋 Forums' },
            { key: 'leaderboard', label: '🏆 Leaderboard' },
          ].map(tab => (
            <button
              key={tab.key}
              onClick={() => setActiveView(tab.key as any)}
              style={{ ...communityPageStyles.secondaryTab, ...(activeView === tab.key ? communityPageStyles.secondaryTabActive : {}) }}
            >
              {tab.label}
            </button>
          ))}
        </div>

        {/* Chat View */}
        {activeView === 'chat' && (
          <div style={communityPageStyles.chatContainer}>
            <div style={communityPageStyles.chatHeader}>
              <span style={communityPageStyles.chatChannelName}># {selectedChannel?.name || 'Select a channel'}</span>
              <span style={communityPageStyles.chatChannelDesc}>{selectedChannel?.description}</span>
            </div>

            <div style={communityPageStyles.messagesContainer}>
              {messages.length === 0 ? (
                <div style={communityPageStyles.noMessages}>
                  <p>No messages yet. Be the first to say something!</p>
                </div>
              ) : (
                messages.map(msg => (
                  <div key={msg.id} style={communityPageStyles.message}>
                    <div style={communityPageStyles.messageHeader}>
                      <span style={{ ...communityPageStyles.messageBadge, color: msg.user.badgeColor }}>{msg.user.badgeIcon}</span>
                      <span style={communityPageStyles.messageUsername}>{msg.user.username}</span>
                      <span style={communityPageStyles.messageTime}>{formatTime(msg.createdAt)}</span>
                    </div>
                    <div style={communityPageStyles.messageContent}>{msg.content}</div>
                  </div>
                ))
              )}
            </div>

            <div style={communityPageStyles.inputContainer}>
              <input
                style={communityPageStyles.messageInput}
                type="text"
                placeholder={selectedChannel ? `Message #${selectedChannel.name}` : 'Select a channel...'}
                value={newMessage}
                onChange={e => setNewMessage(e.target.value)}
                onKeyPress={e => e.key === 'Enter' && sendMessage()}
                disabled={!selectedChannel || selectedChannel.isReadOnly}
              />
              <button style={communityPageStyles.sendBtn} onClick={sendMessage} disabled={!selectedChannel}>
                Send
              </button>
            </div>
          </div>
        )}

        {/* Leaderboard View */}
        {activeView === 'leaderboard' && (
          <div style={communityPageStyles.leaderboardContainer}>
            <div style={communityPageStyles.leaderboardHeader}>
              <h3>🏆 Mining Leaderboard</h3>
              <select 
                style={communityPageStyles.leaderboardSelect}
                value={leaderboardType}
                onChange={e => setLeaderboardType(e.target.value)}
              >
                <option value="hashrate">Hashrate</option>
                <option value="shares">Shares</option>
                <option value="blocks">Blocks Found</option>
              </select>
            </div>
            <div style={communityPageStyles.leaderboardList}>
              {leaderboard.map((entry, idx) => (
                <div key={entry.id} style={communityPageStyles.leaderboardEntry}>
                  <span style={communityPageStyles.leaderboardRank}>#{idx + 1}</span>
                  <span style={communityPageStyles.leaderboardBadge}>{entry.badgeIcon}</span>
                  <span style={communityPageStyles.leaderboardName}>{entry.username}</span>
                  <span style={communityPageStyles.leaderboardValue}>{entry.value}</span>
                </div>
              ))}
            </div>
          </div>
        )}

        {/* Forums View */}
        {activeView === 'forums' && (
          <div style={communityPageStyles.forumsContainer}>
            <h3 style={{ color: '#00d4ff', margin: '0 0 20px' }}>📋 Forums</h3>
            <p style={{ color: '#888' }}>Forums coming soon...</p>
          </div>
        )}
      </div>

      {/* Create Channel Modal */}
      {showCreateChannel && (
        <div style={communityPageStyles.modalOverlay} onClick={() => setShowCreateChannel(false)}>
          <div style={communityPageStyles.modal} onClick={e => e.stopPropagation()}>
            <h3 style={communityPageStyles.modalTitle}>Create New Channel</h3>
            <div style={communityPageStyles.formGroup}>
              <label style={communityPageStyles.label}>Channel Name *</label>
              <input style={communityPageStyles.input} type="text" placeholder="e.g., general-chat" value={channelForm.name} onChange={e => setChannelForm({...channelForm, name: e.target.value})} />
            </div>
            <div style={communityPageStyles.formGroup}>
              <label style={communityPageStyles.label}>Description</label>
              <input style={communityPageStyles.input} type="text" placeholder="What's this channel for?" value={channelForm.description} onChange={e => setChannelForm({...channelForm, description: e.target.value})} />
            </div>
            <div style={communityPageStyles.formGroup}>
              <label style={communityPageStyles.label}>Category *</label>
              <select style={communityPageStyles.select} value={channelForm.category_id} onChange={e => setChannelForm({...channelForm, category_id: e.target.value})}>
                <option value="">Select a category</option>
                {categories.map(cat => <option key={cat.id} value={cat.id}>{cat.name}</option>)}
              </select>
            </div>
            <div style={communityPageStyles.formGroup}>
              <label style={communityPageStyles.label}>Channel Type</label>
              <select style={communityPageStyles.select} value={channelForm.type} onChange={e => setChannelForm({...channelForm, type: e.target.value})}>
                <option value="text">💬 Text</option>
                <option value="announcement">📢 Announcement</option>
                <option value="regional">🌍 Regional</option>
              </select>
            </div>
            <div style={communityPageStyles.modalActions}>
              <button style={communityPageStyles.cancelBtn} onClick={() => setShowCreateChannel(false)}>Cancel</button>
              <button style={communityPageStyles.submitBtn} onClick={handleCreateChannel}>Create</button>
            </div>
          </div>
        </div>
      )}

      {/* Create Category Modal */}
      {showCreateCategory && (
        <div style={communityPageStyles.modalOverlay} onClick={() => setShowCreateCategory(false)}>
          <div style={communityPageStyles.modal} onClick={e => e.stopPropagation()}>
            <h3 style={communityPageStyles.modalTitle}>Create New Category</h3>
            <div style={communityPageStyles.formGroup}>
              <label style={communityPageStyles.label}>Category Name *</label>
              <input style={communityPageStyles.input} type="text" placeholder="e.g., General, Mining Talk" value={categoryForm.name} onChange={e => setCategoryForm({...categoryForm, name: e.target.value})} />
            </div>
            <div style={communityPageStyles.formGroup}>
              <label style={communityPageStyles.label}>Description</label>
              <input style={communityPageStyles.input} type="text" placeholder="What topics belong here?" value={categoryForm.description} onChange={e => setCategoryForm({...categoryForm, description: e.target.value})} />
            </div>
            <div style={communityPageStyles.modalActions}>
              <button style={communityPageStyles.cancelBtn} onClick={() => setShowCreateCategory(false)}>Cancel</button>
              <button style={communityPageStyles.submitBtn} onClick={handleCreateCategory}>Create</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

const communityPageStyles: { [key: string]: React.CSSProperties } = {
  pageContainer: { display: 'flex', height: 'calc(100vh - 100px)', backgroundColor: '#0a0a0f' },
  leftSidebar: { width: '260px', backgroundColor: '#1a1a2e', borderRight: '1px solid #2a2a4a', display: 'flex', flexDirection: 'column', overflowY: 'auto' },
  sidebarHeader: { padding: '15px', borderBottom: '1px solid #2a2a4a', color: '#00d4ff', fontWeight: 'bold', fontSize: '1.1rem', display: 'flex', justifyContent: 'space-between', alignItems: 'center' },
  addBtn: { background: 'none', border: 'none', cursor: 'pointer', fontSize: '1rem', padding: '4px 8px', borderRadius: '4px', color: '#888' },
  emptyState: { padding: '20px', textAlign: 'center', color: '#666' },
  category: { marginBottom: '5px' },
  categoryHeader: { display: 'flex', alignItems: 'center', gap: '8px', padding: '8px 15px', color: '#888', fontSize: '0.85rem', cursor: 'pointer', textTransform: 'uppercase' },
  categoryName: { fontWeight: 'bold' },
  channel: { display: 'flex', alignItems: 'center', gap: '8px', padding: '8px 15px 8px 25px', color: '#888', cursor: 'pointer', borderRadius: '4px', margin: '2px 8px' },
  channelActive: { backgroundColor: '#2a2a4a', color: '#e0e0e0' },
  channelHash: { color: '#666' },
  channelBadge: { marginLeft: 'auto', fontSize: '0.8rem' },
  onlineSection: { marginTop: 'auto', borderTop: '1px solid #2a2a4a', padding: '10px' },
  onlineHeader: { padding: '8px', color: '#888', fontSize: '0.8rem', textTransform: 'uppercase' },
  onlineUser: { display: 'flex', alignItems: 'center', gap: '8px', padding: '4px 8px', fontSize: '0.9rem' },
  onlineIndicator: { width: '8px', height: '8px', borderRadius: '50%', backgroundColor: '#4ade80' },
  onlineUsername: { color: '#e0e0e0' },
  mainContent: { flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' },
  secondaryNav: { display: 'flex', gap: '5px', padding: '10px 20px', borderBottom: '1px solid #2a2a4a', backgroundColor: '#1a1a2e' },
  secondaryTab: { padding: '10px 20px', backgroundColor: 'transparent', border: 'none', color: '#888', fontSize: '0.95rem', cursor: 'pointer', borderRadius: '6px' },
  secondaryTabActive: { backgroundColor: '#2a2a4a', color: '#00d4ff' },
  chatContainer: { flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' },
  chatHeader: { padding: '15px 20px', borderBottom: '1px solid #2a2a4a', backgroundColor: '#1a1a2e' },
  chatChannelName: { color: '#e0e0e0', fontWeight: 'bold', fontSize: '1.1rem' },
  chatChannelDesc: { color: '#888', fontSize: '0.9rem', marginLeft: '15px' },
  messagesContainer: { flex: 1, overflowY: 'auto', padding: '20px' },
  noMessages: { textAlign: 'center', color: '#666', padding: '40px' },
  message: { marginBottom: '20px' },
  messageHeader: { display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '4px' },
  messageBadge: { fontSize: '1.1rem' },
  messageUsername: { color: '#00d4ff', fontWeight: 'bold' },
  messageTime: { color: '#666', fontSize: '0.8rem' },
  messageContent: { color: '#e0e0e0', paddingLeft: '28px' },
  inputContainer: { display: 'flex', gap: '10px', padding: '15px 20px', borderTop: '1px solid #2a2a4a', backgroundColor: '#1a1a2e' },
  messageInput: { flex: 1, padding: '12px 16px', backgroundColor: '#0a0a15', border: '1px solid #2a2a4a', borderRadius: '8px', color: '#e0e0e0', fontSize: '1rem' },
  sendBtn: { padding: '12px 24px', backgroundColor: '#00d4ff', border: 'none', borderRadius: '8px', color: '#0a0a0f', fontWeight: 'bold', cursor: 'pointer' },
  leaderboardContainer: { padding: '20px', overflowY: 'auto' },
  leaderboardHeader: { display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '20px', color: '#00d4ff' },
  leaderboardSelect: { padding: '8px 16px', backgroundColor: '#1a1a2e', border: '1px solid #2a2a4a', borderRadius: '6px', color: '#e0e0e0' },
  leaderboardList: { display: 'flex', flexDirection: 'column', gap: '10px' },
  leaderboardEntry: { display: 'flex', alignItems: 'center', gap: '15px', padding: '15px', backgroundColor: '#1a1a2e', borderRadius: '8px', border: '1px solid #2a2a4a' },
  leaderboardRank: { color: '#f59e0b', fontWeight: 'bold', width: '40px' },
  leaderboardBadge: { fontSize: '1.2rem' },
  leaderboardName: { flex: 1, color: '#e0e0e0' },
  leaderboardValue: { color: '#00d4ff', fontWeight: 'bold' },
  forumsContainer: { padding: '20px' },
  loading: { display: 'flex', justifyContent: 'center', alignItems: 'center', height: 'calc(100vh - 100px)', color: '#00d4ff', fontSize: '1.2rem' },
  modalOverlay: { position: 'fixed', top: 0, left: 0, right: 0, bottom: 0, backgroundColor: 'rgba(0,0,0,0.8)', display: 'flex', justifyContent: 'center', alignItems: 'center', zIndex: 1000 },
  modal: { backgroundColor: '#1a1a2e', padding: '30px', borderRadius: '12px', border: '1px solid #2a2a4a', width: '100%', maxWidth: '450px' },
  modalTitle: { color: '#00d4ff', margin: '0 0 20px' },
  formGroup: { marginBottom: '15px' },
  label: { display: 'block', color: '#888', marginBottom: '5px', fontSize: '0.9rem' },
  input: { width: '100%', padding: '10px 12px', backgroundColor: '#0a0a15', border: '1px solid #2a2a4a', borderRadius: '6px', color: '#e0e0e0', fontSize: '1rem', boxSizing: 'border-box' },
  select: { width: '100%', padding: '10px 12px', backgroundColor: '#0a0a15', border: '1px solid #2a2a4a', borderRadius: '6px', color: '#e0e0e0', fontSize: '1rem' },
  modalActions: { display: 'flex', gap: '10px', marginTop: '20px' },
  cancelBtn: { flex: 1, padding: '10px', backgroundColor: '#2a2a4a', border: 'none', borderRadius: '6px', color: '#e0e0e0', cursor: 'pointer' },
  submitBtn: { flex: 1, padding: '10px', backgroundColor: '#00d4ff', border: 'none', borderRadius: '6px', color: '#0a0a0f', fontWeight: 'bold', cursor: 'pointer' },
};

function CommunitySection({ token, user }: { token: string; user: any }) {
  const [activeView, setActiveView] = useState<'chat' | 'forums' | 'leaderboard' | 'profile'>('chat');
  const [categories, setCategories] = useState<ChannelCategory[]>([]);
  const [selectedChannel, setSelectedChannel] = useState<Channel | null>(null);
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [newMessage, setNewMessage] = useState('');
  const [forums, setForums] = useState<ForumCategory[]>([]);
  const [selectedForum, setSelectedForum] = useState<ForumCategory | null>(null);
  const [forumPosts, setForumPosts] = useState<ForumPost[]>([]);
  const [onlineUsers, setOnlineUsers] = useState<OnlineUser[]>([]);
  const [collapsedCategories, setCollapsedCategories] = useState<Set<number>>(new Set());
  const [loading, setLoading] = useState(true);
  const [leaderboard, setLeaderboard] = useState<any[]>([]);
  const [leaderboardType, setLeaderboardType] = useState('hashrate');

  const fetchChannels = async () => {
    console.log('[CommunitySection] fetchChannels called');
    try {
      const res = await fetch('/api/v1/community/channels', { headers: { Authorization: `Bearer ${token}` } });
      console.log('[CommunitySection] fetchChannels response:', res.status);
      if (res.ok) {
        const data = await res.json();
        console.log('[CommunitySection] fetchChannels data:', data);
        setCategories(data.categories || []);
        if (data.categories?.length > 0 && data.categories[0].channels?.length > 0) {
          setSelectedChannel(data.categories[0].channels[0]);
        }
      } else {
        console.error('[CommunitySection] fetchChannels failed:', res.status);
      }
    } catch (e) { console.error('[CommunitySection] fetchChannels error:', e); }
    setLoading(false);
  };

  useEffect(() => {
    console.log('[CommunitySection] mounted');
    fetchChannels();
    fetchOnlineUsers();
    const interval = setInterval(fetchOnlineUsers, 30000);
    return () => clearInterval(interval);
  }, []);

  useEffect(() => {
    if (selectedChannel) fetchMessages();
  }, [selectedChannel]);

  useEffect(() => {
    if (activeView === 'forums') fetchForums();
    if (activeView === 'leaderboard') fetchLeaderboard();
  }, [activeView, leaderboardType]);

  const fetchMessages = async () => {
    if (!selectedChannel) return;
    try {
      const res = await fetch(`/api/v1/community/channels/${selectedChannel.id}/messages`, { headers: { Authorization: `Bearer ${token}` } });
      if (res.ok) {
        const data = await res.json();
        setMessages(data.messages || []);
      }
    } catch (e) { console.error(e); }
  };

  const fetchOnlineUsers = async () => {
    try {
      const res = await fetch('/api/v1/community/online-users', { headers: { Authorization: `Bearer ${token}` } });
      if (res.ok) {
        const data = await res.json();
        setOnlineUsers(data.users || []);
      }
    } catch (e) { console.error(e); }
  };

  const fetchForums = async () => {
    try {
      const res = await fetch('/api/v1/community/forums', { headers: { Authorization: `Bearer ${token}` } });
      if (res.ok) {
        const data = await res.json();
        setForums(data.forums || []);
      }
    } catch (e) { console.error(e); }
  };

  const fetchForumPosts = async (forumId: number) => {
    try {
      const res = await fetch(`/api/v1/community/forums/${forumId}/posts`, { headers: { Authorization: `Bearer ${token}` } });
      if (res.ok) {
        const data = await res.json();
        setForumPosts(data.posts || []);
      }
    } catch (e) { console.error(e); }
  };

  const fetchLeaderboard = async () => {
    try {
      const res = await fetch(`/api/v1/community/leaderboard?type=${leaderboardType}`, { headers: { Authorization: `Bearer ${token}` } });
      if (res.ok) {
        const data = await res.json();
        setLeaderboard(data.leaderboard || []);
      }
    } catch (e) { console.error(e); }
  };

  const sendMessage = async () => {
    if (!newMessage.trim() || !selectedChannel) return;
    try {
      const res = await fetch(`/api/v1/community/channels/${selectedChannel.id}/messages`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
        body: JSON.stringify({ content: newMessage })
      });
      if (res.ok) {
        setNewMessage('');
        fetchMessages();
      }
    } catch (e) { console.error(e); }
  };

  const toggleCategory = (catId: number) => {
    const newCollapsed = new Set(collapsedCategories);
    if (newCollapsed.has(catId)) newCollapsed.delete(catId);
    else newCollapsed.add(catId);
    setCollapsedCategories(newCollapsed);
  };

  const formatTime = (dateStr: string) => {
    const date = new Date(dateStr);
    return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
  };

  if (loading) return <div style={commStyles.loading}>Loading community...</div>;

  return (
    <section style={commStyles.section}>
      <div style={commStyles.header}>
        <h2 style={commStyles.title}>💬 Community</h2>
        <div style={commStyles.viewTabs}>
          {[
            { key: 'chat', label: '💬 Chat', icon: '💬' },
            { key: 'forums', label: '📋 Forums', icon: '📋' },
            { key: 'leaderboard', label: '🏆 Leaderboard', icon: '🏆' },
          ].map(tab => (
            <button
              key={tab.key}
              onClick={() => setActiveView(tab.key as any)}
              style={{ ...commStyles.viewTab, ...(activeView === tab.key ? commStyles.viewTabActive : {}) }}
            >
              {tab.label}
            </button>
          ))}
        </div>
      </div>

      <div style={commStyles.container}>
        {activeView === 'chat' && (
          <>
            {/* Channel Sidebar */}
            <div style={commStyles.sidebar}>
              <div style={commStyles.sidebarHeader}>Channels</div>
              {categories.map(cat => (
                <div key={cat.id} style={commStyles.category}>
                  <div style={commStyles.categoryHeader} onClick={() => toggleCategory(cat.id)}>
                    <span>{collapsedCategories.has(cat.id) ? '▶' : '▼'}</span>
                    <span style={commStyles.categoryName}>{cat.name}</span>
                  </div>
                  {!collapsedCategories.has(cat.id) && cat.channels.map(ch => (
                    <div
                      key={ch.id}
                      onClick={() => setSelectedChannel(ch)}
                      style={{ ...commStyles.channel, ...(selectedChannel?.id === ch.id ? commStyles.channelActive : {}) }}
                    >
                      <span style={commStyles.channelHash}>#</span>
                      <span>{ch.name}</span>
                      {ch.type === 'announcement' && <span style={commStyles.channelBadge}>📢</span>}
                    </div>
                  ))}
                </div>
              ))}

              <div style={commStyles.onlineSection}>
                <div style={commStyles.onlineHeader}>Online — {onlineUsers.length}</div>
                {onlineUsers.slice(0, 10).map(u => (
                  <div key={u.id} style={commStyles.onlineUser}>
                    <span style={commStyles.onlineIndicator}></span>
                    <span>{u.badgeIcon}</span>
                    <span style={commStyles.onlineUsername}>{u.username}</span>
                  </div>
                ))}
              </div>
            </div>

            {/* Chat Area */}
            <div style={commStyles.chatArea}>
              <div style={commStyles.chatHeader}>
                <span style={commStyles.chatChannelName}># {selectedChannel?.name || 'Select a channel'}</span>
                <span style={commStyles.chatChannelDesc}>{selectedChannel?.description}</span>
              </div>

              <div style={commStyles.messagesContainer}>
                {messages.map(msg => (
                  <div key={msg.id} style={commStyles.message}>
                    <div style={commStyles.messageHeader}>
                      <span style={{ ...commStyles.messageBadge, color: msg.user.badgeColor }}>{msg.user.badgeIcon}</span>
                      <span style={commStyles.messageUsername}>{msg.user.username}</span>
                      <span style={commStyles.messageTime}>{formatTime(msg.createdAt)}</span>
                      {msg.isEdited && <span style={commStyles.messageEdited}>(edited)</span>}
                    </div>
                    <div style={commStyles.messageContent}>{msg.content}</div>
                  </div>
                ))}
                {messages.length === 0 && (
                  <div style={commStyles.noMessages}>No messages yet. Start the conversation!</div>
                )}
              </div>

              <div style={commStyles.inputArea}>
                <input
                  type="text"
                  value={newMessage}
                  onChange={(e) => setNewMessage(e.target.value)}
                  onKeyPress={(e) => e.key === 'Enter' && sendMessage()}
                  placeholder={`Message #${selectedChannel?.name || 'channel'}...`}
                  style={commStyles.messageInput}
                  disabled={selectedChannel?.isReadOnly || selectedChannel?.adminOnlyPost}
                />
                <button onClick={sendMessage} style={commStyles.sendBtn} disabled={!newMessage.trim()}>
                  Send
                </button>
              </div>
            </div>
          </>
        )}

        {activeView === 'forums' && (
          <div style={commStyles.forumsContainer}>
            {!selectedForum ? (
              <div style={commStyles.forumsList}>
                <h3 style={commStyles.forumsTitle}>📋 Forum Categories</h3>
                {forums.map(forum => (
                  <div
                    key={forum.id}
                    style={commStyles.forumCard}
                    onClick={() => { setSelectedForum(forum); fetchForumPosts(forum.id); }}
                  >
                    <span style={commStyles.forumIcon}>{forum.icon}</span>
                    <div style={commStyles.forumInfo}>
                      <div style={commStyles.forumName}>{forum.name}</div>
                      <div style={commStyles.forumDesc}>{forum.description}</div>
                    </div>
                    <div style={commStyles.forumCount}>{forum.postCount} posts</div>
                  </div>
                ))}
              </div>
            ) : (
              <div style={commStyles.forumPosts}>
                <button onClick={() => setSelectedForum(null)} style={commStyles.backBtn}>← Back to Forums</button>
                <h3 style={commStyles.forumPostsTitle}>{selectedForum.icon} {selectedForum.name}</h3>
                {forumPosts.map(post => (
                  <div key={post.id} style={commStyles.postCard}>
                    <div style={commStyles.postHeader}>
                      {post.isPinned && <span style={commStyles.pinBadge}>📌</span>}
                      {post.isLocked && <span style={commStyles.lockBadge}>🔒</span>}
                      <span style={commStyles.postTitle}>{post.title}</span>
                    </div>
                    <div style={commStyles.postPreview}>{post.preview}</div>
                    <div style={commStyles.postMeta}>
                      <span>{post.author.badgeIcon} {post.author.username}</span>
                      <span>👁 {post.viewCount}</span>
                      <span>💬 {post.replyCount}</span>
                      <span>👍 {post.upvotes}</span>
                    </div>
                  </div>
                ))}
                {forumPosts.length === 0 && <div style={commStyles.noPosts}>No posts in this forum yet.</div>}
              </div>
            )}
          </div>
        )}

        {activeView === 'leaderboard' && (
          <div style={commStyles.leaderboardContainer}>
            <div style={commStyles.leaderboardTabs}>
              {['hashrate', 'blocks', 'forum'].map(type => (
                <button
                  key={type}
                  onClick={() => setLeaderboardType(type)}
                  style={{ ...commStyles.lbTab, ...(leaderboardType === type ? commStyles.lbTabActive : {}) }}
                >
                  {type === 'hashrate' ? '⚡ Hashrate' : type === 'blocks' ? '🎯 Blocks' : '💬 Forum'}
                </button>
              ))}
            </div>
            <div style={commStyles.leaderboardList}>
              {leaderboard.map((entry, idx) => (
                <div key={entry.userId} style={{ ...commStyles.lbEntry, ...(idx < 3 ? commStyles.lbTop3 : {}) }}>
                  <span style={commStyles.lbRank}>
                    {idx === 0 ? '🥇' : idx === 1 ? '🥈' : idx === 2 ? '🥉' : `#${entry.rank}`}
                  </span>
                  <span style={commStyles.lbBadge}>{entry.badgeIcon}</span>
                  <span style={commStyles.lbUsername}>{entry.username}</span>
                  <span style={commStyles.lbScore}>
                    {leaderboardType === 'hashrate' ? formatHashrate(entry.score) : entry.score}
                  </span>
                </div>
              ))}
              {leaderboard.length === 0 && <div style={commStyles.noData}>No leaderboard data yet.</div>}
            </div>
          </div>
        )}
      </div>
    </section>
  );
}

const commStyles: { [key: string]: React.CSSProperties } = {
  section: { background: 'linear-gradient(135deg, #1a1a2e 0%, #0f0f1a 100%)', borderRadius: '12px', border: '1px solid #2a2a4a', marginBottom: '20px', overflow: 'hidden' },
  header: { display: 'flex', justifyContent: 'space-between', alignItems: 'center', padding: '20px 24px', borderBottom: '1px solid #2a2a4a', flexWrap: 'wrap', gap: '15px' },
  title: { fontSize: '1.3rem', color: '#00d4ff', margin: 0 },
  viewTabs: { display: 'flex', gap: '8px' },
  viewTab: { padding: '8px 16px', backgroundColor: '#0a0a15', border: '1px solid #2a2a4a', borderRadius: '6px', color: '#888', cursor: 'pointer', fontSize: '0.9rem' },
  viewTabActive: { backgroundColor: '#00d4ff', color: '#0a0a0f', borderColor: '#00d4ff' },
  loading: { textAlign: 'center', padding: '60px', color: '#00d4ff' },
  container: { display: 'flex', minHeight: '500px' },
  sidebar: { width: '240px', backgroundColor: '#0a0a15', borderRight: '1px solid #2a2a4a', display: 'flex', flexDirection: 'column' },
  sidebarHeader: { padding: '12px 16px', color: '#888', fontSize: '0.75rem', textTransform: 'uppercase', fontWeight: 'bold' },
  category: { marginBottom: '8px' },
  categoryHeader: { display: 'flex', alignItems: 'center', gap: '8px', padding: '4px 16px', color: '#888', cursor: 'pointer', fontSize: '0.8rem' },
  categoryName: { textTransform: 'uppercase', fontWeight: 'bold' },
  channel: { display: 'flex', alignItems: 'center', gap: '6px', padding: '6px 16px 6px 24px', color: '#888', cursor: 'pointer', fontSize: '0.9rem' },
  channelActive: { backgroundColor: '#2a2a4a', color: '#e0e0e0' },
  channelHash: { color: '#555', fontWeight: 'bold' },
  channelBadge: { marginLeft: 'auto', fontSize: '0.75rem' },
  onlineSection: { marginTop: 'auto', borderTop: '1px solid #2a2a4a', padding: '12px 0' },
  onlineHeader: { padding: '8px 16px', color: '#4ade80', fontSize: '0.75rem', textTransform: 'uppercase' },
  onlineUser: { display: 'flex', alignItems: 'center', gap: '8px', padding: '4px 16px', fontSize: '0.85rem' },
  onlineIndicator: { width: '8px', height: '8px', borderRadius: '50%', backgroundColor: '#4ade80' },
  onlineUsername: { color: '#e0e0e0' },
  chatArea: { flex: 1, display: 'flex', flexDirection: 'column' },
  chatHeader: { padding: '12px 20px', borderBottom: '1px solid #2a2a4a', display: 'flex', alignItems: 'center', gap: '12px' },
  chatChannelName: { color: '#e0e0e0', fontWeight: 'bold', fontSize: '1rem' },
  chatChannelDesc: { color: '#888', fontSize: '0.85rem' },
  messagesContainer: { flex: 1, padding: '16px 20px', overflowY: 'auto', maxHeight: '400px' },
  message: { marginBottom: '16px' },
  messageHeader: { display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '4px' },
  messageBadge: { fontSize: '1rem' },
  messageUsername: { color: '#00d4ff', fontWeight: 'bold', fontSize: '0.9rem' },
  messageTime: { color: '#555', fontSize: '0.75rem' },
  messageEdited: { color: '#555', fontSize: '0.7rem', fontStyle: 'italic' },
  messageContent: { color: '#e0e0e0', fontSize: '0.95rem', lineHeight: 1.4, paddingLeft: '28px' },
  noMessages: { textAlign: 'center', color: '#555', padding: '40px' },
  inputArea: { display: 'flex', gap: '10px', padding: '16px 20px', borderTop: '1px solid #2a2a4a' },
  messageInput: { flex: 1, padding: '12px 16px', backgroundColor: '#0a0a15', border: '1px solid #2a2a4a', borderRadius: '8px', color: '#e0e0e0', fontSize: '0.95rem' },
  sendBtn: { padding: '12px 24px', backgroundColor: '#00d4ff', border: 'none', borderRadius: '8px', color: '#0a0a0f', fontWeight: 'bold', cursor: 'pointer' },
  forumsContainer: { flex: 1, padding: '24px' },
  forumsList: { maxWidth: '800px', margin: '0 auto' },
  forumsTitle: { color: '#00d4ff', marginBottom: '20px' },
  forumCard: { display: 'flex', alignItems: 'center', gap: '16px', padding: '16px 20px', backgroundColor: '#0a0a15', borderRadius: '10px', border: '1px solid #2a2a4a', marginBottom: '12px', cursor: 'pointer' },
  forumIcon: { fontSize: '1.5rem' },
  forumInfo: { flex: 1 },
  forumName: { color: '#e0e0e0', fontWeight: 'bold', marginBottom: '4px' },
  forumDesc: { color: '#888', fontSize: '0.85rem' },
  forumCount: { color: '#00d4ff', fontSize: '0.85rem' },
  forumPosts: { maxWidth: '800px', margin: '0 auto' },
  backBtn: { padding: '8px 16px', backgroundColor: 'transparent', border: '1px solid #2a2a4a', borderRadius: '6px', color: '#888', cursor: 'pointer', marginBottom: '16px' },
  forumPostsTitle: { color: '#00d4ff', marginBottom: '20px' },
  postCard: { padding: '16px 20px', backgroundColor: '#0a0a15', borderRadius: '10px', border: '1px solid #2a2a4a', marginBottom: '12px' },
  postHeader: { display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '8px' },
  pinBadge: { fontSize: '0.9rem' },
  lockBadge: { fontSize: '0.9rem' },
  postTitle: { color: '#e0e0e0', fontWeight: 'bold', fontSize: '1.05rem' },
  postPreview: { color: '#888', fontSize: '0.9rem', marginBottom: '12px' },
  postMeta: { display: 'flex', gap: '16px', color: '#555', fontSize: '0.85rem' },
  noPosts: { textAlign: 'center', color: '#555', padding: '40px' },
  leaderboardContainer: { flex: 1, padding: '24px' },
  leaderboardTabs: { display: 'flex', gap: '8px', marginBottom: '20px', justifyContent: 'center' },
  lbTab: { padding: '10px 20px', backgroundColor: '#0a0a15', border: '1px solid #2a2a4a', borderRadius: '8px', color: '#888', cursor: 'pointer' },
  lbTabActive: { backgroundColor: '#00d4ff', color: '#0a0a0f', borderColor: '#00d4ff' },
  leaderboardList: { maxWidth: '600px', margin: '0 auto' },
  lbEntry: { display: 'flex', alignItems: 'center', gap: '12px', padding: '12px 16px', backgroundColor: '#0a0a15', borderRadius: '8px', marginBottom: '8px', border: '1px solid #2a2a4a' },
  lbTop3: { borderColor: '#fbbf24' },
  lbRank: { width: '40px', textAlign: 'center', fontSize: '1.1rem' },
  lbBadge: { fontSize: '1.2rem' },
  lbUsername: { flex: 1, color: '#e0e0e0', fontWeight: 'bold' },
  lbScore: { color: '#00d4ff', fontWeight: 'bold' },
  noData: { textAlign: 'center', color: '#555', padding: '40px' },
};

// Wallet Manager Component
interface UserWallet {
  id: number;
  address: string;
  label: string;
  percentage: number;
  is_primary: boolean;
  is_active: boolean;
  created_at: string;
}

interface WalletSummary {
  total_wallets: number;
  active_wallets: number;
  total_percentage: number;
  remaining_percentage: number;
  has_primary_wallet: boolean;
}

function WalletManager({ token, showMessage }: { token: string; showMessage: (type: 'success' | 'error', text: string) => void }) {
  const [wallets, setWallets] = useState<UserWallet[]>([]);
  const [summary, setSummary] = useState<WalletSummary | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [showAddForm, setShowAddForm] = useState(false);
  const [editingWallet, setEditingWallet] = useState<UserWallet | null>(null);
  const [newWallet, setNewWallet] = useState({ address: '', label: '', percentage: 100, is_primary: false });

  useEffect(() => {
    fetchWallets();
  }, [token]);

  const fetchWallets = async () => {
    try {
      const res = await fetch('/api/v1/user/wallets', {
        headers: { 'Authorization': `Bearer ${token}` }
      });
      if (res.ok) {
        const data = await res.json();
        setWallets(data.wallets || []);
        setSummary(data.summary || null);
      }
    } catch (error) {
      console.error('Failed to fetch wallets:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleAddWallet = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newWallet.address.trim()) return;
    
    setSaving(true);
    try {
      const res = await fetch('/api/v1/user/wallets', {
        method: 'POST',
        headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
        body: JSON.stringify(newWallet)
      });
      
      if (res.ok) {
        showMessage('success', 'Wallet added successfully!');
        setNewWallet({ address: '', label: '', percentage: summary?.remaining_percentage || 100, is_primary: false });
        setShowAddForm(false);
        fetchWallets();
      } else {
        const data = await res.json();
        showMessage('error', data.error || 'Failed to add wallet');
      }
    } catch (error) {
      showMessage('error', 'Network error. Please try again.');
    } finally {
      setSaving(false);
    }
  };

  const handleUpdateWallet = async (wallet: UserWallet) => {
    setSaving(true);
    try {
      const res = await fetch(`/api/v1/user/wallets/${wallet.id}`, {
        method: 'PUT',
        headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
        body: JSON.stringify(wallet)
      });
      
      if (res.ok) {
        showMessage('success', 'Wallet updated successfully!');
        setEditingWallet(null);
        fetchWallets();
      } else {
        const data = await res.json();
        showMessage('error', data.error || 'Failed to update wallet');
      }
    } catch (error) {
      showMessage('error', 'Network error. Please try again.');
    } finally {
      setSaving(false);
    }
  };

  const handleDeleteWallet = async (walletId: number) => {
    if (!window.confirm('Are you sure you want to delete this wallet?')) return;
    
    try {
      const res = await fetch(`/api/v1/user/wallets/${walletId}`, {
        method: 'DELETE',
        headers: { 'Authorization': `Bearer ${token}` }
      });
      
      if (res.ok) {
        showMessage('success', 'Wallet deleted successfully!');
        fetchWallets();
      } else {
        const data = await res.json();
        showMessage('error', data.error || 'Failed to delete wallet');
      }
    } catch (error) {
      showMessage('error', 'Network error. Please try again.');
    }
  };

  if (loading) {
    return (
      <section style={styles.section}>
        <h2 style={styles.sectionTitle}>💰 Wallet Settings</h2>
        <div style={styles.loading}>Loading wallet settings...</div>
      </section>
    );
  }

  return (
    <section style={{...styles.section, marginBottom: '30px'}}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '20px' }}>
        <h2 style={{...styles.sectionTitle, margin: 0}}>💰 Multi-Wallet Payout Settings</h2>
        <button 
          style={walletStyles.addBtn} 
          onClick={() => { setShowAddForm(true); setNewWallet({ ...newWallet, percentage: summary?.remaining_percentage || 100 }); }}
          disabled={summary?.remaining_percentage === 0}
        >
          + Add Wallet
        </button>
      </div>

      {/* Summary Bar */}
      {summary && (
        <div style={walletStyles.summaryBar}>
          <div style={walletStyles.summaryItem}>
            <span style={walletStyles.summaryLabel}>Active Wallets</span>
            <span style={walletStyles.summaryValue}>{summary.active_wallets}</span>
          </div>
          <div style={walletStyles.summaryItem}>
            <span style={walletStyles.summaryLabel}>Allocated</span>
            <span style={walletStyles.summaryValue}>{summary.total_percentage.toFixed(1)}%</span>
          </div>
          <div style={walletStyles.summaryItem}>
            <span style={walletStyles.summaryLabel}>Remaining</span>
            <span style={{...walletStyles.summaryValue, color: summary.remaining_percentage > 0 ? '#fbbf24' : '#4ade80'}}>
              {summary.remaining_percentage.toFixed(1)}%
            </span>
          </div>
          <div style={walletStyles.progressBar}>
            <div style={{...walletStyles.progressFill, width: `${summary.total_percentage}%`}}></div>
          </div>
        </div>
      )}

      {/* Add Wallet Form */}
      {showAddForm && (
        <div style={walletStyles.formContainer}>
          <h3 style={walletStyles.formTitle}>Add New Wallet</h3>
          <form onSubmit={handleAddWallet} style={walletStyles.form}>
            <div style={walletStyles.formRow}>
              <div style={walletStyles.formGroup}>
                <label style={walletStyles.label}>Wallet Address *</label>
                <input
                  style={walletStyles.input}
                  type="text"
                  value={newWallet.address}
                  onChange={(e) => setNewWallet({...newWallet, address: e.target.value})}
                  placeholder="0x..."
                  required
                />
              </div>
              <div style={walletStyles.formGroup}>
                <label style={walletStyles.label}>Label</label>
                <input
                  style={walletStyles.input}
                  type="text"
                  value={newWallet.label}
                  onChange={(e) => setNewWallet({...newWallet, label: e.target.value})}
                  placeholder="e.g., Main, Hardware, Exchange"
                />
              </div>
            </div>
            <div style={walletStyles.formRow}>
              <div style={walletStyles.formGroup}>
                <label style={walletStyles.label}>Payout Percentage *</label>
                <div style={walletStyles.percentageInput}>
                  <input
                    style={{...walletStyles.input, width: '100px'}}
                    type="number"
                    min="0.01"
                    max={summary?.remaining_percentage || 100}
                    step="0.01"
                    value={newWallet.percentage}
                    onChange={(e) => setNewWallet({...newWallet, percentage: parseFloat(e.target.value) || 0})}
                    required
                  />
                  <span style={walletStyles.percentSign}>%</span>
                </div>
                <p style={walletStyles.hint}>Available: {summary?.remaining_percentage.toFixed(2)}%</p>
              </div>
              <div style={walletStyles.formGroup}>
                <label style={walletStyles.checkboxLabel}>
                  <input
                    type="checkbox"
                    checked={newWallet.is_primary}
                    onChange={(e) => setNewWallet({...newWallet, is_primary: e.target.checked})}
                  />
                  Set as Primary Wallet
                </label>
              </div>
            </div>
            <div style={walletStyles.formActions}>
              <button type="button" style={walletStyles.cancelBtn} onClick={() => setShowAddForm(false)}>Cancel</button>
              <button type="submit" style={walletStyles.saveBtn} disabled={saving}>{saving ? 'Adding...' : 'Add Wallet'}</button>
            </div>
          </form>
        </div>
      )}

      {/* Wallets List */}
      {wallets.length === 0 ? (
        <div style={walletStyles.emptyState}>
          <p>No wallets configured yet.</p>
          <p style={{ color: '#888', fontSize: '0.9rem' }}>Add a wallet to receive mining payouts.</p>
        </div>
      ) : (
        <div style={walletStyles.walletsList}>
          {wallets.map((wallet) => (
            <div key={wallet.id} style={{...walletStyles.walletCard, opacity: wallet.is_active ? 1 : 0.6}}>
              {editingWallet?.id === wallet.id ? (
                <div style={walletStyles.editForm}>
                  <input
                    style={walletStyles.input}
                    type="text"
                    value={editingWallet.address}
                    onChange={(e) => setEditingWallet({...editingWallet, address: e.target.value})}
                    placeholder="Wallet address"
                  />
                  <input
                    style={{...walletStyles.input, width: '150px'}}
                    type="text"
                    value={editingWallet.label}
                    onChange={(e) => setEditingWallet({...editingWallet, label: e.target.value})}
                    placeholder="Label"
                  />
                  <div style={walletStyles.percentageInput}>
                    <input
                      style={{...walletStyles.input, width: '80px'}}
                      type="number"
                      min="0.01"
                      max="100"
                      step="0.01"
                      value={editingWallet.percentage}
                      onChange={(e) => setEditingWallet({...editingWallet, percentage: parseFloat(e.target.value) || 0})}
                    />
                    <span style={walletStyles.percentSign}>%</span>
                  </div>
                  <button style={walletStyles.saveBtn} onClick={() => handleUpdateWallet(editingWallet)} disabled={saving}>Save</button>
                  <button style={walletStyles.cancelBtn} onClick={() => setEditingWallet(null)}>Cancel</button>
                </div>
              ) : (
                <>
                  <div style={walletStyles.walletMain}>
                    <div style={walletStyles.walletInfo}>
                      <div style={walletStyles.walletHeader}>
                        <span style={walletStyles.walletLabel}>{wallet.label || 'Wallet'}</span>
                        {wallet.is_primary && <span style={walletStyles.primaryBadge}>⭐ Primary</span>}
                        {!wallet.is_active && <span style={walletStyles.inactiveBadge}>Inactive</span>}
                      </div>
                      <code style={walletStyles.addressCode}>
                        {wallet.address.slice(0, 12)}...{wallet.address.slice(-10)}
                      </code>
                    </div>
                    <div style={walletStyles.walletPercentage}>
                      <span style={walletStyles.percentageValue}>{wallet.percentage.toFixed(1)}%</span>
                      <span style={walletStyles.percentageLabel}>of payouts</span>
                    </div>
                  </div>
                  <div style={walletStyles.walletActions}>
                    <button style={walletStyles.editBtn} onClick={() => setEditingWallet({...wallet})}>✏️</button>
                    <button style={walletStyles.deleteBtn} onClick={() => handleDeleteWallet(wallet.id)}>🗑️</button>
                  </div>
                </>
              )}
            </div>
          ))}
        </div>
      )}

      {/* Payout Split Preview */}
      {wallets.length > 1 && summary && summary.total_percentage === 100 && (
        <div style={walletStyles.previewBox}>
          <h4 style={walletStyles.previewTitle}>📊 Payout Split Preview (Example: 10 BDAG)</h4>
          <div style={walletStyles.previewList}>
            {wallets.filter(w => w.is_active).map((wallet) => (
              <div key={wallet.id} style={walletStyles.previewItem}>
                <span>{wallet.label || 'Wallet'}</span>
                <span style={walletStyles.previewAmount}>{(10 * wallet.percentage / 100).toFixed(4)} BDAG</span>
              </div>
            ))}
          </div>
        </div>
      )}
    </section>
  );
}

const walletStyles: { [key: string]: React.CSSProperties } = {
  formContainer: { backgroundColor: '#0a0a15', padding: '20px', borderRadius: '8px', marginBottom: '20px', border: '1px solid #2a2a4a' },
  formTitle: { color: '#00d4ff', margin: '0 0 15px', fontSize: '1.1rem' },
  form: { display: 'flex', flexDirection: 'column', gap: '15px' },
  formRow: { display: 'flex', gap: '20px', flexWrap: 'wrap' },
  formGroup: { flex: 1, minWidth: '200px' },
  formActions: { display: 'flex', gap: '10px', justifyContent: 'flex-end', marginTop: '10px' },
  label: { display: 'block', color: '#888', fontSize: '0.85rem', textTransform: 'uppercase', marginBottom: '6px' },
  input: { width: '100%', padding: '10px 14px', backgroundColor: '#1a1a2e', border: '1px solid #2a2a4a', borderRadius: '6px', color: '#e0e0e0', fontSize: '0.95rem', boxSizing: 'border-box' },
  percentageInput: { display: 'flex', alignItems: 'center', gap: '8px' },
  percentSign: { color: '#888', fontSize: '1rem' },
  checkboxLabel: { display: 'flex', alignItems: 'center', gap: '8px', color: '#e0e0e0', cursor: 'pointer', marginTop: '20px' },
  addBtn: { padding: '10px 20px', backgroundColor: '#00d4ff', border: 'none', borderRadius: '6px', color: '#0a0a0f', fontWeight: 'bold', cursor: 'pointer', fontSize: '0.9rem' },
  saveBtn: { padding: '10px 20px', backgroundColor: '#00d4ff', border: 'none', borderRadius: '6px', color: '#0a0a0f', fontWeight: 'bold', cursor: 'pointer' },
  cancelBtn: { padding: '10px 20px', backgroundColor: '#2a2a4a', border: 'none', borderRadius: '6px', color: '#e0e0e0', cursor: 'pointer' },
  hint: { color: '#666', fontSize: '0.8rem', marginTop: '4px' },
  summaryBar: { display: 'flex', gap: '20px', alignItems: 'center', backgroundColor: '#0a0a15', padding: '15px 20px', borderRadius: '8px', marginBottom: '20px', flexWrap: 'wrap' },
  summaryItem: { display: 'flex', flexDirection: 'column', gap: '4px' },
  summaryLabel: { color: '#888', fontSize: '0.75rem', textTransform: 'uppercase' },
  summaryValue: { color: '#00d4ff', fontSize: '1.3rem', fontWeight: 'bold' },
  progressBar: { flex: 1, minWidth: '200px', height: '8px', backgroundColor: '#2a2a4a', borderRadius: '4px', overflow: 'hidden' },
  progressFill: { height: '100%', backgroundColor: '#00d4ff', borderRadius: '4px', transition: 'width 0.3s ease' },
  walletsList: { display: 'flex', flexDirection: 'column', gap: '12px' },
  walletCard: { display: 'flex', alignItems: 'center', justifyContent: 'space-between', backgroundColor: '#0a0a15', padding: '16px 20px', borderRadius: '8px', border: '1px solid #2a2a4a' },
  walletMain: { display: 'flex', alignItems: 'center', gap: '30px', flex: 1 },
  walletInfo: { display: 'flex', flexDirection: 'column', gap: '6px' },
  walletHeader: { display: 'flex', alignItems: 'center', gap: '10px' },
  walletLabel: { color: '#e0e0e0', fontWeight: 'bold', fontSize: '1rem' },
  walletPercentage: { display: 'flex', flexDirection: 'column', alignItems: 'flex-end', gap: '2px' },
  percentageValue: { color: '#00d4ff', fontSize: '1.5rem', fontWeight: 'bold' },
  percentageLabel: { color: '#666', fontSize: '0.75rem' },
  walletActions: { display: 'flex', gap: '8px' },
  editBtn: { background: 'none', border: 'none', cursor: 'pointer', fontSize: '1.1rem', padding: '6px' },
  deleteBtn: { background: 'none', border: 'none', cursor: 'pointer', fontSize: '1.1rem', padding: '6px' },
  editForm: { display: 'flex', gap: '10px', alignItems: 'center', flexWrap: 'wrap', width: '100%' },
  addressCode: { fontFamily: 'monospace', color: '#888', fontSize: '0.85rem' },
  primaryBadge: { backgroundColor: '#4a3a1a', color: '#fbbf24', padding: '3px 8px', borderRadius: '4px', fontSize: '0.75rem' },
  inactiveBadge: { backgroundColor: '#4a1a1a', color: '#f87171', padding: '3px 8px', borderRadius: '4px', fontSize: '0.75rem' },
  emptyState: { textAlign: 'center', padding: '40px', color: '#666', backgroundColor: '#0a0a15', borderRadius: '8px' },
  previewBox: { backgroundColor: '#0a0a15', padding: '20px', borderRadius: '8px', marginTop: '20px', border: '1px dashed #2a2a4a' },
  previewTitle: { color: '#888', margin: '0 0 15px', fontSize: '0.95rem' },
  previewList: { display: 'flex', flexDirection: 'column', gap: '8px' },
  previewItem: { display: 'flex', justifyContent: 'space-between', alignItems: 'center', padding: '8px 12px', backgroundColor: '#1a1a2e', borderRadius: '4px' },
  previewAmount: { color: '#00d4ff', fontWeight: 'bold', fontFamily: 'monospace' },
};

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

interface AuthModalProps {
  view: AuthView;
  setView: (view: AuthView | null) => void;
  setToken: (token: string | null) => void;
  showMessage: (type: 'success' | 'error', text: string) => void;
  resetToken: string | null;
}

function AuthModal({ view, setView, setToken, showMessage, resetToken }: AuthModalProps) {
  const [formData, setFormData] = useState({ username: '', email: '', password: '', confirmPassword: '', newPassword: '' });
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setFormData({ ...formData, [e.target.name]: e.target.value });
    setError('');
  };

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
        body: JSON.stringify({ username: formData.username, email: formData.email, password: formData.password })
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
    <div style={styles.modalOverlay} onClick={closeModal}>
      <div style={styles.modal} onClick={(e) => e.stopPropagation()}>
        <button style={styles.closeBtn} onClick={closeModal}>×</button>
        
        {view === 'login' && (
          <form onSubmit={handleLogin}>
            <h2 style={styles.modalTitle}>Login</h2>
            {error && <div style={styles.errorMsg}>{error}</div>}
            <input style={styles.input} type="email" name="email" placeholder="Email Address" value={formData.email} onChange={handleChange} required />
            <input style={styles.input} type="password" name="password" placeholder="Password" value={formData.password} onChange={handleChange} required />
            <button style={styles.submitBtn} type="submit" disabled={loading}>{loading ? 'Logging in...' : 'Login'}</button>
            <div style={styles.authLinks}>
              <span style={styles.authLink} onClick={() => setView('forgot-password')}>Forgot Password?</span>
              <span style={styles.authLink} onClick={() => setView('register')}>Create Account</span>
            </div>
          </form>
        )}

        {view === 'register' && (
          <form onSubmit={handleRegister}>
            <h2 style={styles.modalTitle}>Create Account</h2>
            {error && <div style={styles.errorMsg}>{error}</div>}
            <input style={styles.input} type="text" name="username" placeholder="Username" value={formData.username} onChange={handleChange} required />
            <input style={styles.input} type="email" name="email" placeholder="Email" value={formData.email} onChange={handleChange} required />
            <input style={styles.input} type="password" name="password" placeholder="Password (min 8 characters)" value={formData.password} onChange={handleChange} minLength={8} required />
            <input style={styles.input} type="password" name="confirmPassword" placeholder="Confirm Password" value={formData.confirmPassword} onChange={handleChange} required />
            <button style={styles.submitBtn} type="submit" disabled={loading}>{loading ? 'Creating...' : 'Create Account'}</button>
            <div style={styles.authLinks}>
              <span style={styles.authLink} onClick={() => setView('login')}>Already have an account? Login</span>
            </div>
          </form>
        )}

        {view === 'forgot-password' && (
          <form onSubmit={handleForgotPassword}>
            <h2 style={styles.modalTitle}>Reset Password</h2>
            <p style={styles.modalDesc}>Enter your email address and we'll send you a link to reset your password.</p>
            {error && <div style={styles.errorMsg}>{error}</div>}
            <input style={styles.input} type="email" name="email" placeholder="Email Address" value={formData.email} onChange={handleChange} required />
            <button style={styles.submitBtn} type="submit" disabled={loading}>{loading ? 'Sending...' : 'Send Reset Link'}</button>
            <div style={styles.authLinks}>
              <span style={styles.authLink} onClick={() => setView('login')}>Back to Login</span>
            </div>
          </form>
        )}

        {view === 'reset-password' && (
          <form onSubmit={handleResetPassword}>
            <h2 style={styles.modalTitle}>Set New Password</h2>
            <p style={styles.modalDesc}>Enter your new password below.</p>
            {error && <div style={styles.errorMsg}>{error}</div>}
            <input style={styles.input} type="password" name="newPassword" placeholder="New Password (min 8 characters)" value={formData.newPassword} onChange={handleChange} minLength={8} required />
            <input style={styles.input} type="password" name="confirmPassword" placeholder="Confirm New Password" value={formData.confirmPassword} onChange={handleChange} required />
            <button style={styles.submitBtn} type="submit" disabled={loading}>{loading ? 'Resetting...' : 'Reset Password'}</button>
          </form>
        )}
      </div>
    </div>
  );
}

interface AdminPanelProps {
  token: string;
  onClose: () => void;
  showMessage: (type: 'success' | 'error', text: string) => void;
}

function AdminPanel({ token, onClose, showMessage }: AdminPanelProps) {
  const [activeTab, setActiveTab] = useState<'users' | 'stats' | 'algorithm' | 'community' | 'roles'>('users');
  const [users, setUsers] = useState<AdminUser[]>([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState('');
  const [page, setPage] = useState(1);
  const [totalCount, setTotalCount] = useState(0);
  const [selectedUser, setSelectedUser] = useState<any>(null);
  const [editingUser, setEditingUser] = useState<AdminUser | null>(null);
  const [editForm, setEditForm] = useState({ pool_fee_percent: '', payout_address: '', is_active: true, is_admin: false });
  const pageSize = 10;

  // Role management state
  const [moderators, setModerators] = useState<any[]>([]);
  const [admins, setAdmins] = useState<any[]>([]);
  const [rolesLoading, setRolesLoading] = useState(false);
  const [roleChangeUser, setRoleChangeUser] = useState<any>(null);
  const [newRole, setNewRole] = useState('');

  // Pool statistics state
  const [poolStatsRange, setPoolStatsRange] = useState<TimeRange>('24h');
  const [poolHashrateData, setPoolHashrateData] = useState<any[]>([]);
  const [poolSharesData, setPoolSharesData] = useState<any[]>([]);
  const [poolMinersData, setPoolMinersData] = useState<any[]>([]);
  const [poolPayoutsData, setPoolPayoutsData] = useState<any[]>([]);
  const [poolDistribution, setPoolDistribution] = useState<any[]>([]);
  const [statsLoading, setStatsLoading] = useState(false);

  // Algorithm settings state
  const [algorithmData, setAlgorithmData] = useState<any>(null);
  const [algorithmForm, setAlgorithmForm] = useState({
    algorithm: '',
    algorithm_variant: '',
    difficulty_target: '',
    block_time: '',
    stratum_port: '',
    algorithm_params: ''
  });
  const [savingAlgorithm, setSavingAlgorithm] = useState(false);

  // Community/Channel management state
  const [channels, setChannels] = useState<any[]>([]);
  const [categories, setCategories] = useState<any[]>([]);
  const [channelsLoading, setChannelsLoading] = useState(false);
  const [showCreateChannel, setShowCreateChannel] = useState(false);
  const [showCreateCategory, setShowCreateCategory] = useState(false);
  const [editingChannel, setEditingChannel] = useState<any>(null);
  const [channelForm, setChannelForm] = useState({ name: '', description: '', category_id: '', type: 'text', is_read_only: false, admin_only_post: false });
  const [categoryForm, setCategoryForm] = useState({ name: '', description: '' });

  useEffect(() => { fetchUsers(); }, [page, search]);
  useEffect(() => { if (activeTab === 'algorithm') fetchAlgorithmSettings(); }, [activeTab]);
  useEffect(() => { if (activeTab === 'stats') fetchPoolStats(); }, [activeTab, poolStatsRange]);
  useEffect(() => { if (activeTab === 'community') fetchChannelsAndCategories(); }, [activeTab]);
  useEffect(() => { if (activeTab === 'roles') fetchRoles(); }, [activeTab]);

  const fetchRoles = async () => {
    setRolesLoading(true);
    try {
      const headers = { 'Authorization': `Bearer ${token}` };
      const [modsRes, adminsRes] = await Promise.all([
        fetch('/api/v1/admin/moderators', { headers }),
        fetch('/api/v1/admin/admins', { headers })
      ]);
      if (modsRes.ok) {
        const data = await modsRes.json();
        setModerators(data.moderators || []);
      }
      if (adminsRes.ok) {
        const data = await adminsRes.json();
        setAdmins(data.admins || []);
      }
    } catch (error) {
      console.error('Failed to fetch roles:', error);
    } finally {
      setRolesLoading(false);
    }
  };

  const handleChangeRole = async () => {
    if (!roleChangeUser || !newRole) return;
    try {
      const response = await fetch(`/api/v1/admin/users/${roleChangeUser.id}/role`, {
        method: 'PUT',
        headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
        body: JSON.stringify({ role: newRole })
      });
      if (response.ok) {
        showMessage('success', `User role changed to ${newRole}`);
        setRoleChangeUser(null);
        setNewRole('');
        fetchRoles();
        fetchUsers();
      } else {
        const data = await response.json();
        showMessage('error', data.error || 'Failed to change role');
      }
    } catch (error) {
      showMessage('error', 'Network error');
    }
  };

  const fetchChannelsAndCategories = async () => {
    setChannelsLoading(true);
    try {
      const headers = { 'Authorization': `Bearer ${token}` };
      const [channelsRes, categoriesRes] = await Promise.all([
        fetch('/api/v1/community/channels', { headers }),
        fetch('/api/v1/community/channel-categories', { headers })
      ]);
      if (channelsRes.ok) {
        const data = await channelsRes.json();
        setChannels(data.channels || []);
      }
      if (categoriesRes.ok) {
        const data = await categoriesRes.json();
        setCategories(data.categories || []);
      }
    } catch (error) {
      console.error('Failed to fetch channels:', error);
    } finally {
      setChannelsLoading(false);
    }
  };

  const handleCreateChannel = async () => {
    if (!channelForm.name || !channelForm.category_id) {
      showMessage('error', 'Channel name and category are required');
      return;
    }
    try {
      const response = await fetch('/api/v1/admin/community/channels', {
        method: 'POST',
        headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
        body: JSON.stringify(channelForm)
      });
      if (response.ok) {
        showMessage('success', 'Channel created successfully');
        setShowCreateChannel(false);
        setChannelForm({ name: '', description: '', category_id: '', type: 'text', is_read_only: false, admin_only_post: false });
        fetchChannelsAndCategories();
      } else {
        const data = await response.json();
        showMessage('error', data.error || 'Failed to create channel');
      }
    } catch (error) {
      showMessage('error', 'Network error');
    }
  };

  const handleUpdateChannel = async () => {
    if (!editingChannel || !channelForm.name) {
      showMessage('error', 'Channel name is required');
      return;
    }
    try {
      const response = await fetch(`/api/v1/admin/community/channels/${editingChannel.id}`, {
        method: 'PUT',
        headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
        body: JSON.stringify(channelForm)
      });
      if (response.ok) {
        showMessage('success', 'Channel updated successfully');
        setEditingChannel(null);
        setChannelForm({ name: '', description: '', category_id: '', type: 'text', is_read_only: false, admin_only_post: false });
        fetchChannelsAndCategories();
      } else {
        const data = await response.json();
        showMessage('error', data.error || 'Failed to update channel');
      }
    } catch (error) {
      showMessage('error', 'Network error');
    }
  };

  const handleDeleteChannel = async (channelId: string) => {
    if (!window.confirm('Are you sure you want to delete this channel?')) return;
    try {
      const response = await fetch(`/api/v1/admin/community/channels/${channelId}`, {
        method: 'DELETE',
        headers: { 'Authorization': `Bearer ${token}` }
      });
      if (response.ok) {
        showMessage('success', 'Channel deleted');
        fetchChannelsAndCategories();
      } else {
        const data = await response.json();
        showMessage('error', data.error || 'Failed to delete channel');
      }
    } catch (error) {
      showMessage('error', 'Network error');
    }
  };

  const handleCreateCategory = async () => {
    if (!categoryForm.name) {
      showMessage('error', 'Category name is required');
      return;
    }
    try {
      const response = await fetch('/api/v1/admin/community/channel-categories', {
        method: 'POST',
        headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
        body: JSON.stringify(categoryForm)
      });
      if (response.ok) {
        showMessage('success', 'Category created successfully');
        setShowCreateCategory(false);
        setCategoryForm({ name: '', description: '' });
        fetchChannelsAndCategories();
      } else {
        const data = await response.json();
        showMessage('error', data.error || 'Failed to create category');
      }
    } catch (error) {
      showMessage('error', 'Network error');
    }
  };

  const handleDeleteCategory = async (categoryId: string) => {
    if (!window.confirm('Are you sure you want to delete this category? All channels in this category will be deleted.')) return;
    try {
      const response = await fetch(`/api/v1/admin/community/channel-categories/${categoryId}`, {
        method: 'DELETE',
        headers: { 'Authorization': `Bearer ${token}` }
      });
      if (response.ok) {
        showMessage('success', 'Category deleted');
        fetchChannelsAndCategories();
      } else {
        const data = await response.json();
        showMessage('error', data.error || 'Failed to delete category');
      }
    } catch (error) {
      showMessage('error', 'Network error');
    }
  };

  const openEditChannel = (channel: any) => {
    setEditingChannel(channel);
    setChannelForm({
      name: channel.name,
      description: channel.description || '',
      category_id: channel.category_id,
      type: channel.type || 'text',
      is_read_only: channel.is_read_only || false,
      admin_only_post: channel.admin_only_post || false
    });
  };

  const fetchPoolStats = async () => {
    setStatsLoading(true);
    try {
      const headers = { 'Authorization': `Bearer ${token}` };
      const [hashRes, sharesRes, minersRes, payoutsRes, distRes] = await Promise.all([
        fetch(`/api/v1/admin/stats/hashrate?range=${poolStatsRange}`, { headers }),
        fetch(`/api/v1/admin/stats/shares?range=${poolStatsRange}`, { headers }),
        fetch(`/api/v1/admin/stats/miners?range=${poolStatsRange}`, { headers }),
        fetch(`/api/v1/admin/stats/payouts?range=${poolStatsRange}`, { headers }),
        fetch('/api/v1/admin/stats/distribution', { headers })
      ]);

      if (hashRes.ok) {
        const data = await hashRes.json();
        setPoolHashrateData(data.data?.map((d: any) => ({
          ...d,
          time: new Date(d.time).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
          totalGH: d.totalHashrate / 1000000000
        })) || []);
      }
      if (sharesRes.ok) {
        const data = await sharesRes.json();
        setPoolSharesData(data.data?.map((d: any) => ({
          ...d,
          time: new Date(d.time).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
        })) || []);
      }
      if (minersRes.ok) {
        const data = await minersRes.json();
        setPoolMinersData(data.data?.map((d: any) => ({
          ...d,
          time: new Date(d.time).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
        })) || []);
      }
      if (payoutsRes.ok) {
        const data = await payoutsRes.json();
        setPoolPayoutsData(data.data?.map((d: any) => ({
          ...d,
          time: new Date(d.time).toLocaleDateString([], { month: 'short', day: 'numeric' })
        })) || []);
      }
      if (distRes.ok) {
        const data = await distRes.json();
        setPoolDistribution(data.distribution || []);
      }
    } catch (error) {
      console.error('Failed to fetch pool stats:', error);
    } finally {
      setStatsLoading(false);
    }
  };

  const fetchAlgorithmSettings = async () => {
    try {
      const response = await fetch('/api/v1/admin/algorithm', {
        headers: { 'Authorization': `Bearer ${token}` }
      });
      if (response.ok) {
        const data = await response.json();
        setAlgorithmData(data);
        setAlgorithmForm({
          algorithm: data.algorithm?.algorithm?.value || '',
          algorithm_variant: data.algorithm?.algorithm_variant?.value || '',
          difficulty_target: data.algorithm?.difficulty_target?.value || '',
          block_time: data.algorithm?.block_time?.value || '',
          stratum_port: data.algorithm?.stratum_port?.value || '',
          algorithm_params: data.algorithm?.algorithm_params?.value || '{}'
        });
      }
    } catch (error) {
      console.error('Failed to fetch algorithm settings:', error);
    }
  };

  const handleSaveAlgorithm = async () => {
    setSavingAlgorithm(true);
    try {
      const response = await fetch('/api/v1/admin/algorithm', {
        method: 'PUT',
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json'
        },
        body: JSON.stringify(algorithmForm)
      });
      if (response.ok) {
        const data = await response.json();
        showMessage('success', data.message + (data.note ? ` Note: ${data.note}` : ''));
        fetchAlgorithmSettings();
      } else {
        const data = await response.json();
        showMessage('error', data.error || 'Failed to update algorithm settings');
      }
    } catch (error) {
      showMessage('error', 'Network error');
    } finally {
      setSavingAlgorithm(false);
    }
  };

  const fetchUsers = async () => {
    setLoading(true);
    try {
      const params = new URLSearchParams({ page: String(page), page_size: String(pageSize) });
      if (search) params.append('search', search);
      const response = await fetch(`/api/v1/admin/users?${params}`, { headers: { 'Authorization': `Bearer ${token}` } });
      if (response.ok) {
        const data = await response.json();
        setUsers(data.users || []);
        setTotalCount(data.total_count || 0);
      } else if (response.status === 403) {
        showMessage('error', 'Admin access required');
        onClose();
      }
    } catch (error) {
      console.error('Failed to fetch users:', error);
    } finally {
      setLoading(false);
    }
  };

  const fetchUserDetail = async (userId: number) => {
    try {
      const response = await fetch(`/api/v1/admin/users/${userId}`, { headers: { 'Authorization': `Bearer ${token}` } });
      if (response.ok) setSelectedUser(await response.json());
    } catch (error) {
      console.error('Failed to fetch user details:', error);
    }
  };

  const handleEditUser = (user: AdminUser) => {
    setEditingUser(user);
    setEditForm({ pool_fee_percent: user.pool_fee_percent?.toString() || '', payout_address: user.payout_address || '', is_active: user.is_active, is_admin: user.is_admin });
  };

  const handleSaveUser = async () => {
    if (!editingUser) return;
    const updates: any = {};
    if (editForm.pool_fee_percent !== '') {
      const fee = parseFloat(editForm.pool_fee_percent);
      if (isNaN(fee) || fee < 0 || fee > 100) { showMessage('error', 'Pool fee must be between 0 and 100'); return; }
      updates.pool_fee_percent = fee;
    }
    if (editForm.payout_address) updates.payout_address = editForm.payout_address;
    updates.is_active = editForm.is_active;
    updates.is_admin = editForm.is_admin;

    try {
      const response = await fetch(`/api/v1/admin/users/${editingUser.id}`, {
        method: 'PUT',
        headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
        body: JSON.stringify(updates)
      });
      if (response.ok) { showMessage('success', 'User updated successfully'); setEditingUser(null); fetchUsers(); }
      else { const data = await response.json(); showMessage('error', data.error || 'Failed to update user'); }
    } catch (error) { showMessage('error', 'Network error'); }
  };

  const handleDeleteUser = async (userId: number) => {
    if (!window.confirm('Are you sure you want to deactivate this user?')) return;
    try {
      const response = await fetch(`/api/v1/admin/users/${userId}`, { method: 'DELETE', headers: { 'Authorization': `Bearer ${token}` } });
      if (response.ok) { showMessage('success', 'User deactivated'); fetchUsers(); }
      else { const data = await response.json(); showMessage('error', data.error || 'Failed to deactivate user'); }
    } catch (error) { showMessage('error', 'Network error'); }
  };

  const totalPages = Math.ceil(totalCount / pageSize);

  return (
    <div style={adminStyles.overlay} onClick={onClose}>
      <div style={adminStyles.panel} onClick={e => e.stopPropagation()}>
        <div style={adminStyles.header}>
          <h2 style={adminStyles.title}>🛡️ Admin Panel</h2>
          <button style={adminStyles.closeBtn} onClick={onClose}>×</button>
        </div>

        {/* Tabs */}
        <div style={adminStyles.tabs}>
          <button 
            style={{...adminStyles.tab, ...(activeTab === 'users' ? adminStyles.tabActive : {})}} 
            onClick={() => setActiveTab('users')}
          >
            👥 User Management
          </button>
          <button 
            style={{...adminStyles.tab, ...(activeTab === 'stats' ? adminStyles.tabActive : {})}} 
            onClick={() => setActiveTab('stats')}
          >
            📊 Pool Statistics
          </button>
          <button 
            style={{...adminStyles.tab, ...(activeTab === 'algorithm' ? adminStyles.tabActive : {})}} 
            onClick={() => setActiveTab('algorithm')}
          >
            ⚙️ Algorithm Settings
          </button>
          <button 
            style={{...adminStyles.tab, ...(activeTab === 'community' ? adminStyles.tabActive : {})}} 
            onClick={() => setActiveTab('community')}
          >
            💬 Community Channels
          </button>
          <button 
            style={{...adminStyles.tab, ...(activeTab === 'roles' ? adminStyles.tabActive : {})}} 
            onClick={() => setActiveTab('roles')}
          >
            👑 Role Management
          </button>
        </div>

        {/* User Management Tab */}
        {activeTab === 'users' && (
          <>
            <div style={adminStyles.searchBar}>
              <input style={adminStyles.searchInput} type="text" placeholder="Search users by username or email..." value={search} onChange={e => { setSearch(e.target.value); setPage(1); }} />
            </div>

            {loading ? (
          <div style={adminStyles.loading}>Loading users...</div>
        ) : (
          <>
            <div style={adminStyles.tableContainer}>
              <table style={adminStyles.table}>
                <thead>
                  <tr>
                    <th style={adminStyles.th}>Username</th>
                    <th style={adminStyles.th}>Email</th>
                    <th style={adminStyles.th}>Wallets</th>
                    <th style={adminStyles.th}>Hashrate</th>
                    <th style={adminStyles.th}>Earnings</th>
                    <th style={adminStyles.th}>Fee %</th>
                    <th style={adminStyles.th}>Status</th>
                    <th style={adminStyles.th}>Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {users.map(user => (
                    <tr key={user.id} style={adminStyles.tr}>
                      <td style={adminStyles.td}><span style={user.is_admin ? adminStyles.adminBadge : {}}>{user.username} {user.is_admin && '👑'}</span></td>
                      <td style={adminStyles.td}>{user.email}</td>
                      <td style={adminStyles.td}>
                        <div style={{ display: 'flex', flexDirection: 'column', gap: '2px' }}>
                          <span style={{ color: user.wallet_count > 1 ? '#00d4ff' : '#888', fontWeight: user.wallet_count > 1 ? 'bold' : 'normal' }}>
                            {user.wallet_count || 0} wallet{user.wallet_count !== 1 ? 's' : ''}
                          </span>
                          {user.primary_wallet && (
                            <span style={{ fontSize: '0.75rem', color: '#666', maxWidth: '120px', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }} title={user.primary_wallet}>
                              {user.primary_wallet.substring(0, 12)}...
                            </span>
                          )}
                          {user.wallet_count > 1 && (
                            <span style={{ fontSize: '0.7rem', backgroundColor: '#1a3a4a', color: '#00d4ff', padding: '1px 4px', borderRadius: '3px', display: 'inline-block', width: 'fit-content' }}>
                              Split: {user.total_allocated?.toFixed(0) || 0}%
                            </span>
                          )}
                        </div>
                      </td>
                      <td style={adminStyles.td}>{formatHashrate(user.total_hashrate)}</td>
                      <td style={adminStyles.td}>{user.total_earnings.toFixed(4)}</td>
                      <td style={adminStyles.td}>{user.pool_fee_percent || 'Default'}</td>
                      <td style={adminStyles.td}><span style={user.is_active ? adminStyles.activeBadge : adminStyles.inactiveBadge}>{user.is_active ? 'Active' : 'Inactive'}</span></td>
                      <td style={adminStyles.td}>
                        <button style={adminStyles.actionBtn} onClick={() => fetchUserDetail(user.id)}>👁️</button>
                        <button style={adminStyles.actionBtn} onClick={() => handleEditUser(user)}>✏️</button>
                        <button style={{...adminStyles.actionBtn, opacity: 0.7}} onClick={() => handleDeleteUser(user.id)}>🗑️</button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>

            <div style={adminStyles.pagination}>
              <button style={adminStyles.pageBtn} disabled={page <= 1} onClick={() => setPage(p => p - 1)}>← Prev</button>
              <span style={adminStyles.pageInfo}>Page {page} of {totalPages} ({totalCount} users)</span>
              <button style={adminStyles.pageBtn} disabled={page >= totalPages} onClick={() => setPage(p => p + 1)}>Next →</button>
            </div>
          </>
        )}

        {editingUser && (
          <div style={adminStyles.editModal}>
            <h3 style={adminStyles.editTitle}>Edit User: {editingUser.username}</h3>
            <div style={adminStyles.formGroup}>
              <label style={adminStyles.label}>Pool Fee % (leave empty for default)</label>
              <input style={adminStyles.formInput} type="number" min="0" max="100" step="0.1" placeholder="e.g., 1.5" value={editForm.pool_fee_percent} onChange={e => setEditForm({...editForm, pool_fee_percent: e.target.value})} />
            </div>
            <div style={adminStyles.formGroup}>
              <label style={adminStyles.label}>Payout Address</label>
              <input style={adminStyles.formInput} type="text" placeholder="0x..." value={editForm.payout_address} onChange={e => setEditForm({...editForm, payout_address: e.target.value})} />
            </div>
            <div style={adminStyles.formGroup}>
              <label style={adminStyles.checkboxLabel}><input type="checkbox" checked={editForm.is_active} onChange={e => setEditForm({...editForm, is_active: e.target.checked})} /> Active</label>
              <label style={adminStyles.checkboxLabel}><input type="checkbox" checked={editForm.is_admin} onChange={e => setEditForm({...editForm, is_admin: e.target.checked})} /> Admin</label>
            </div>
            <div style={adminStyles.editActions}>
              <button style={adminStyles.cancelBtn} onClick={() => setEditingUser(null)}>Cancel</button>
              <button style={adminStyles.saveBtn} onClick={handleSaveUser}>Save Changes</button>
            </div>
          </div>
        )}

        {selectedUser && (
          <div style={adminStyles.detailModal}>
            <button style={adminStyles.closeDetailBtn} onClick={() => setSelectedUser(null)}>×</button>
            <h3 style={adminStyles.detailTitle}>User Details: {selectedUser.user.username}</h3>
            <div style={adminStyles.detailCard}>
              <p><strong>Email:</strong> {selectedUser.user.email}</p>
              <p><strong>Payout Address:</strong> {selectedUser.user.payout_address || 'Not set'}</p>
              <p><strong>Pool Fee:</strong> {selectedUser.user.pool_fee_percent || 'Default'}%</p>
              <p><strong>Total Earnings:</strong> {selectedUser.user.total_earnings}</p>
              <p><strong>Pending Payout:</strong> {selectedUser.user.pending_payout}</p>
              <p><strong>Blocks Found:</strong> {selectedUser.user.blocks_found}</p>
            </div>

            {/* Wallet Configuration Section */}
            <h4 style={adminStyles.subTitle}>💰 Wallet Configuration ({selectedUser.wallets?.length || 0} wallets)</h4>
            {selectedUser.wallet_summary && (
              <div style={{ ...adminStyles.detailCard, backgroundColor: '#0a1520', borderColor: '#00d4ff' }}>
                <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '10px' }}>
                  <span>Total Allocated:</span>
                  <span style={{ color: selectedUser.wallet_summary.total_allocated >= 100 ? '#4ade80' : '#fbbf24', fontWeight: 'bold' }}>
                    {selectedUser.wallet_summary.total_allocated?.toFixed(1)}%
                  </span>
                </div>
                {selectedUser.wallet_summary.has_multiple_wallets && (
                  <div style={{ backgroundColor: '#1a3a4a', padding: '8px 12px', borderRadius: '6px', marginBottom: '10px' }}>
                    <span style={{ color: '#00d4ff', fontSize: '0.9rem' }}>⚡ Multi-wallet split payments enabled</span>
                  </div>
                )}
                {selectedUser.wallet_summary.remaining_percent > 0 && (
                  <div style={{ backgroundColor: '#4d3a1a', padding: '8px 12px', borderRadius: '6px' }}>
                    <span style={{ color: '#fbbf24', fontSize: '0.9rem' }}>⚠️ {selectedUser.wallet_summary.remaining_percent?.toFixed(1)}% unallocated</span>
                  </div>
                )}
              </div>
            )}
            {selectedUser.wallets?.length > 0 ? (
              <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
                {selectedUser.wallets.map((wallet: any) => (
                  <div key={wallet.id} style={{ 
                    ...adminStyles.detailCard, 
                    display: 'flex', 
                    justifyContent: 'space-between', 
                    alignItems: 'center',
                    borderColor: wallet.is_primary ? '#9b59b6' : '#2a2a4a',
                    backgroundColor: wallet.is_active ? '#0a0a15' : '#1a1a1a',
                    opacity: wallet.is_active ? 1 : 0.6
                  }}>
                    <div style={{ flex: 1 }}>
                      <div style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '4px' }}>
                        {wallet.is_primary && <span style={{ backgroundColor: '#4d1a4d', color: '#d946ef', padding: '2px 6px', borderRadius: '4px', fontSize: '0.7rem' }}>PRIMARY</span>}
                        {!wallet.is_active && <span style={{ backgroundColor: '#4d1a1a', color: '#ef4444', padding: '2px 6px', borderRadius: '4px', fontSize: '0.7rem' }}>INACTIVE</span>}
                        {wallet.label && <span style={{ color: '#888', fontSize: '0.85rem' }}>{wallet.label}</span>}
                      </div>
                      <p style={{ margin: 0, fontFamily: 'monospace', fontSize: '0.85rem', color: '#00d4ff', wordBreak: 'break-all' }}>
                        {wallet.address}
                      </p>
                    </div>
                    <div style={{ 
                      backgroundColor: '#1a1a2e', 
                      padding: '8px 16px', 
                      borderRadius: '8px', 
                      marginLeft: '15px',
                      textAlign: 'center',
                      minWidth: '80px'
                    }}>
                      <div style={{ fontSize: '1.2rem', fontWeight: 'bold', color: '#4ade80' }}>{wallet.percentage}%</div>
                      <div style={{ fontSize: '0.7rem', color: '#888' }}>allocation</div>
                    </div>
                  </div>
                ))}
              </div>
            ) : (
              <div style={{ ...adminStyles.detailCard, textAlign: 'center', color: '#666' }}>
                <p>No wallets configured. User is using legacy payout address.</p>
              </div>
            )}

            <h4 style={adminStyles.subTitle}>Share Statistics</h4>
            <div style={adminStyles.detailCard}>
              <p><strong>Total Shares:</strong> {selectedUser.shares_stats.total_shares}</p>
              <p><strong>Valid:</strong> {selectedUser.shares_stats.valid_shares}</p>
              <p><strong>Invalid:</strong> {selectedUser.shares_stats.invalid_shares}</p>
              <p><strong>Last 24h:</strong> {selectedUser.shares_stats.last_24_hours}</p>
            </div>
            <h4 style={adminStyles.subTitle}>Miners ({selectedUser.miners?.length || 0})</h4>
            {selectedUser.miners?.map((m: any) => (
              <div key={m.id} style={adminStyles.minerRow}>
                <span>{m.name}</span>
                <span>{formatHashrate(m.hashrate)}</span>
                <span style={m.is_active ? adminStyles.activeBadge : adminStyles.inactiveBadge}>{m.is_active ? 'Online' : 'Offline'}</span>
              </div>
            ))}
          </div>
        )}
          </>
        )}

        {/* Pool Statistics Tab */}
        {activeTab === 'stats' && (
          <div style={adminStyles.algorithmContainer}>
            <div style={adminStyles.algoHeader}>
              <h3 style={adminStyles.algoTitle}>📊 Pool Statistics Dashboard</h3>
              <div style={graphStyles.timeSelector}>
                {([
                  { value: '1h', label: '1H' },
                  { value: '6h', label: '6H' },
                  { value: '24h', label: '24H' },
                  { value: '7d', label: '7D' },
                  { value: '30d', label: '30D' },
                  { value: '3m', label: '3M' },
                  { value: '6m', label: '6M' },
                  { value: '1y', label: '1Y' },
                  { value: 'all', label: 'All' }
                ] as { value: TimeRange; label: string }[]).map(({ value, label }) => (
                  <button
                    key={value}
                    style={{...graphStyles.timeBtn, ...(poolStatsRange === value ? graphStyles.timeBtnActive : {})}}
                    onClick={() => setPoolStatsRange(value)}
                  >
                    {label}
                  </button>
                ))}
              </div>
            </div>

            {statsLoading ? (
              <div style={graphStyles.loading}>Loading pool statistics...</div>
            ) : (
              <div style={graphStyles.chartsGrid}>
                {/* Pool Hashrate Chart */}
                <div style={graphStyles.chartCard}>
                  <h3 style={graphStyles.chartTitle}>⚡ Pool Hashrate</h3>
                  <ResponsiveContainer width="100%" height={250}>
                    <AreaChart data={poolHashrateData}>
                      <defs>
                        <linearGradient id="poolHashGradient" x1="0" y1="0" x2="0" y2="1">
                          <stop offset="5%" stopColor="#9b59b6" stopOpacity={0.3}/>
                          <stop offset="95%" stopColor="#9b59b6" stopOpacity={0}/>
                        </linearGradient>
                      </defs>
                      <CartesianGrid strokeDasharray="3 3" stroke="#2a2a4a" />
                      <XAxis dataKey="time" stroke="#888" fontSize={12} />
                      <YAxis stroke="#888" fontSize={12} tickFormatter={(v) => `${v.toFixed(1)} GH/s`} />
                      <Tooltip 
                        contentStyle={{ backgroundColor: '#1a1a2e', border: '1px solid #2a2a4a', borderRadius: '8px' }}
                        labelStyle={{ color: '#9b59b6' }}
                        formatter={(value: number) => [`${value.toFixed(2)} GH/s`, 'Pool Hashrate']}
                      />
                      <Area type="monotone" dataKey="totalGH" stroke="#9b59b6" fill="url(#poolHashGradient)" strokeWidth={2} />
                    </AreaChart>
                  </ResponsiveContainer>
                </div>

                {/* Active Miners Chart */}
                <div style={graphStyles.chartCard}>
                  <h3 style={graphStyles.chartTitle}>👥 Active Miners</h3>
                  <ResponsiveContainer width="100%" height={250}>
                    <LineChart data={poolMinersData}>
                      <CartesianGrid strokeDasharray="3 3" stroke="#2a2a4a" />
                      <XAxis dataKey="time" stroke="#888" fontSize={12} />
                      <YAxis stroke="#888" fontSize={12} />
                      <Tooltip 
                        contentStyle={{ backgroundColor: '#1a1a2e', border: '1px solid #2a2a4a', borderRadius: '8px' }}
                        labelStyle={{ color: '#00d4ff' }}
                      />
                      <Legend />
                      <Line type="monotone" dataKey="activeMiners" name="Active Miners" stroke="#00d4ff" strokeWidth={2} dot={false} />
                      <Line type="monotone" dataKey="uniqueUsers" name="Unique Users" stroke="#4ade80" strokeWidth={2} dot={false} />
                    </LineChart>
                  </ResponsiveContainer>
                </div>

                {/* Pool Shares Chart */}
                <div style={graphStyles.chartCard}>
                  <h3 style={graphStyles.chartTitle}>📦 Pool Shares</h3>
                  <ResponsiveContainer width="100%" height={250}>
                    <BarChart data={poolSharesData}>
                      <CartesianGrid strokeDasharray="3 3" stroke="#2a2a4a" />
                      <XAxis dataKey="time" stroke="#888" fontSize={12} />
                      <YAxis stroke="#888" fontSize={12} />
                      <Tooltip 
                        contentStyle={{ backgroundColor: '#1a1a2e', border: '1px solid #2a2a4a', borderRadius: '8px' }}
                        labelStyle={{ color: '#00d4ff' }}
                      />
                      <Legend />
                      <Bar dataKey="validShares" name="Valid" fill="#4ade80" radius={[4, 4, 0, 0]} />
                      <Bar dataKey="invalidShares" name="Invalid" fill="#ef4444" radius={[4, 4, 0, 0]} />
                    </BarChart>
                  </ResponsiveContainer>
                </div>

                {/* Pool Payouts Chart */}
                <div style={graphStyles.chartCard}>
                  <h3 style={graphStyles.chartTitle}>💰 Pool Payouts</h3>
                  <ResponsiveContainer width="100%" height={250}>
                    <AreaChart data={poolPayoutsData}>
                      <defs>
                        <linearGradient id="payoutGradient" x1="0" y1="0" x2="0" y2="1">
                          <stop offset="5%" stopColor="#f59e0b" stopOpacity={0.3}/>
                          <stop offset="95%" stopColor="#f59e0b" stopOpacity={0}/>
                        </linearGradient>
                      </defs>
                      <CartesianGrid strokeDasharray="3 3" stroke="#2a2a4a" />
                      <XAxis dataKey="time" stroke="#888" fontSize={12} />
                      <YAxis stroke="#888" fontSize={12} tickFormatter={(v) => `${v.toFixed(0)}`} />
                      <Tooltip 
                        contentStyle={{ backgroundColor: '#1a1a2e', border: '1px solid #2a2a4a', borderRadius: '8px' }}
                        labelStyle={{ color: '#f59e0b' }}
                        formatter={(value: number) => [`${value.toFixed(2)} BDAG`, 'Cumulative Paid']}
                      />
                      <Area type="monotone" dataKey="cumulative" stroke="#f59e0b" fill="url(#payoutGradient)" strokeWidth={2} />
                    </AreaChart>
                  </ResponsiveContainer>
                </div>

                {/* Hashrate Distribution Pie Chart */}
                <div style={{...graphStyles.chartCard, gridColumn: '1 / -1'}}>
                  <h3 style={graphStyles.chartTitle}>🥧 Hashrate Distribution (Top Miners)</h3>
                  <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-around', flexWrap: 'wrap' }}>
                    <ResponsiveContainer width={400} height={300}>
                      <PieChart>
                        <Pie
                          data={poolDistribution}
                          dataKey="percentage"
                          nameKey="username"
                          cx="50%"
                          cy="50%"
                          outerRadius={100}
                          label={({ username, percentage }) => `${username}: ${percentage.toFixed(1)}%`}
                        >
                          {poolDistribution.map((entry, index) => (
                            <Cell key={`cell-${index}`} fill={['#00d4ff', '#9b59b6', '#4ade80', '#f59e0b', '#ef4444', '#3b82f6', '#ec4899'][index % 7]} />
                          ))}
                        </Pie>
                        <Tooltip 
                          contentStyle={{ backgroundColor: '#1a1a2e', border: '1px solid #2a2a4a', borderRadius: '8px' }}
                          formatter={(value: number, name: string) => [`${value.toFixed(2)}%`, name]}
                        />
                      </PieChart>
                    </ResponsiveContainer>
                    <div style={{ padding: '20px' }}>
                      <h4 style={{ color: '#00d4ff', marginBottom: '15px' }}>Top Contributors</h4>
                      {poolDistribution.slice(0, 5).map((user, idx) => (
                        <div key={idx} style={{ display: 'flex', justifyContent: 'space-between', gap: '40px', padding: '8px 0', borderBottom: '1px solid #2a2a4a' }}>
                          <span style={{ color: ['#00d4ff', '#9b59b6', '#4ade80', '#f59e0b', '#ef4444'][idx] }}>{user.username}</span>
                          <span style={{ color: '#888' }}>{formatHashrate(user.hashrate)}</span>
                          <span style={{ color: '#e0e0e0' }}>{user.percentage.toFixed(1)}%</span>
                        </div>
                      ))}
                    </div>
                  </div>
                </div>
              </div>
            )}
          </div>
        )}

        {/* Algorithm Settings Tab */}
        {activeTab === 'algorithm' && (
          <div style={adminStyles.algorithmContainer}>
            <div style={adminStyles.algoHeader}>
              <h3 style={adminStyles.algoTitle}>⚙️ Mining Algorithm Configuration</h3>
              <p style={adminStyles.algoDesc}>
                Configure the mining algorithm for the pool. BlockDAG uses a custom Scrpy-variant algorithm.
                When BlockDAG releases their official algorithm specification, paste it below.
              </p>
            </div>

            {/* Custom Algorithm Notice */}
            <div style={{ backgroundColor: '#1a2a3a', padding: '20px', borderRadius: '12px', marginBottom: '25px', border: '2px solid #00d4ff' }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: '12px', marginBottom: '12px' }}>
                <span style={{ fontSize: '1.5rem' }}>🔷</span>
                <h4 style={{ color: '#00d4ff', margin: 0 }}>BlockDAG Custom Algorithm</h4>
              </div>
              <p style={{ color: '#b0b0b0', margin: 0, lineHeight: '1.6' }}>
                This pool supports BlockDAG's proprietary Scrpy-variant algorithm. When BlockDAG releases 
                the official algorithm specification or updates, use the <strong>"Custom Algorithm Code"</strong> section 
                below to paste the algorithm definition. The pool will automatically apply the new algorithm.
              </p>
            </div>

            <div style={adminStyles.algoGrid}>
              <div style={adminStyles.algoCard}>
                <label style={adminStyles.algoLabel}>Algorithm Type</label>
                <select 
                  style={adminStyles.algoSelect}
                  value={algorithmForm.algorithm}
                  onChange={e => setAlgorithmForm({...algorithmForm, algorithm: e.target.value})}
                >
                  <option value="scrpy-variant">Scrpy-Variant (BlockDAG Custom)</option>
                  <option value="scrypt">Scrypt</option>
                  <option value="sha256">SHA-256</option>
                  <option value="blake3">Blake3</option>
                  <option value="ethash">Ethash</option>
                  <option value="kawpow">KawPow</option>
                  <option value="custom">Custom (Define Below)</option>
                  {algorithmData?.supported_algorithms?.map((algo: any) => (
                    <option key={algo.id} value={algo.id}>{algo.name}</option>
                  ))}
                </select>
                <p style={adminStyles.algoHint}>
                  Select "Custom" to define a new algorithm from BlockDAG specifications
                </p>
              </div>

              <div style={adminStyles.algoCard}>
                <label style={adminStyles.algoLabel}>Algorithm Variant / Version</label>
                <input 
                  style={adminStyles.algoInput}
                  type="text"
                  value={algorithmForm.algorithm_variant}
                  onChange={e => setAlgorithmForm({...algorithmForm, algorithm_variant: e.target.value})}
                  placeholder="e.g., scrpy-v1.0, blockdag-mainnet"
                />
                <p style={adminStyles.algoHint}>Version identifier for the algorithm variant</p>
              </div>

              <div style={adminStyles.algoCard}>
                <label style={adminStyles.algoLabel}>Base Difficulty</label>
                <input 
                  style={adminStyles.algoInput}
                  type="text"
                  value={algorithmForm.difficulty_target}
                  onChange={e => setAlgorithmForm({...algorithmForm, difficulty_target: e.target.value})}
                  placeholder="e.g., 1.0"
                />
                <p style={adminStyles.algoHint}>Starting difficulty for share validation</p>
              </div>

              <div style={adminStyles.algoCard}>
                <label style={adminStyles.algoLabel}>Target Block Time (seconds)</label>
                <input 
                  style={adminStyles.algoInput}
                  type="text"
                  value={algorithmForm.block_time}
                  onChange={e => setAlgorithmForm({...algorithmForm, block_time: e.target.value})}
                  placeholder="e.g., 10"
                />
                <p style={adminStyles.algoHint}>Expected time between blocks</p>
              </div>

              <div style={adminStyles.algoCard}>
                <label style={adminStyles.algoLabel}>Stratum Port</label>
                <input 
                  style={adminStyles.algoInput}
                  type="text"
                  value={algorithmForm.stratum_port}
                  onChange={e => setAlgorithmForm({...algorithmForm, stratum_port: e.target.value})}
                  placeholder="e.g., 3333"
                />
                <p style={adminStyles.algoHint}>Port for miner connections</p>
              </div>

              <div style={{...adminStyles.algoCard, gridColumn: '1 / -1'}}>
                <label style={adminStyles.algoLabel}>Algorithm Parameters (JSON)</label>
                <textarea 
                  style={adminStyles.algoTextarea}
                  value={algorithmForm.algorithm_params}
                  onChange={e => setAlgorithmForm({...algorithmForm, algorithm_params: e.target.value})}
                  placeholder='{"N": 1024, "r": 1, "p": 1, "keyLen": 32}'
                  rows={4}
                />
                <p style={adminStyles.algoHint}>Scrypt parameters: N (CPU/memory cost), r (block size), p (parallelization), keyLen (output length)</p>
              </div>
            </div>

            {/* Custom Algorithm Code Section */}
            <div style={{ backgroundColor: '#0a1015', padding: '25px', borderRadius: '12px', marginTop: '25px', border: '2px dashed #9b59b6' }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: '12px', marginBottom: '15px' }}>
                <span style={{ fontSize: '1.5rem' }}>📝</span>
                <h4 style={{ color: '#9b59b6', margin: 0 }}>Custom Algorithm Code (BlockDAG Official)</h4>
              </div>
              <p style={{ color: '#888', marginBottom: '15px', lineHeight: '1.6' }}>
                When BlockDAG releases their official algorithm specification, paste the complete algorithm definition below.
                This supports Go code, JSON configuration, or algorithm pseudocode that will be compiled into the mining validator.
              </p>
              
              <div style={{ marginBottom: '15px' }}>
                <label style={{ display: 'block', color: '#00d4ff', marginBottom: '8px', fontSize: '0.9rem', textTransform: 'uppercase' }}>
                  Algorithm Name / Identifier
                </label>
                <input 
                  style={{ ...adminStyles.algoInput, backgroundColor: '#1a1a2e', border: '1px solid #9b59b6' }}
                  type="text"
                  placeholder="e.g., blockdag-scrpy-v2, bdag-mainnet-algo"
                  defaultValue="scrpy-variant"
                />
              </div>

              <div style={{ marginBottom: '15px' }}>
                <label style={{ display: 'block', color: '#00d4ff', marginBottom: '8px', fontSize: '0.9rem', textTransform: 'uppercase' }}>
                  Custom Algorithm Code / Specification
                </label>
                <textarea 
                  style={{ 
                    width: '100%', 
                    minHeight: '300px', 
                    backgroundColor: '#0a0a15', 
                    border: '1px solid #9b59b6', 
                    borderRadius: '8px', 
                    color: '#00ff88', 
                    fontFamily: 'monospace', 
                    fontSize: '0.9rem', 
                    padding: '15px',
                    boxSizing: 'border-box',
                    lineHeight: '1.6',
                    resize: 'vertical'
                  }}
                  placeholder={`// Paste BlockDAG's official algorithm specification here
// Example format:

{
  "algorithm": "scrpy-variant",
  "version": "1.0.0",
  "parameters": {
    "N": 1024,
    "r": 1,
    "p": 1,
    "keyLen": 32,
    "salt": "BlockDAG",
    "hashFunction": "sha256"
  },
  "validation": {
    "targetBits": 24,
    "difficultyAdjustment": "DAA",
    "blockTimeTarget": 10
  },
  "customCode": "// Go implementation or pseudocode"
}`}
                />
              </div>

              <div style={{ display: 'flex', gap: '10px', flexWrap: 'wrap' }}>
                <button 
                  style={{ 
                    padding: '12px 24px', 
                    backgroundColor: '#9b59b6', 
                    border: 'none', 
                    borderRadius: '8px', 
                    color: '#fff', 
                    fontWeight: 'bold', 
                    cursor: 'pointer',
                    display: 'flex',
                    alignItems: 'center',
                    gap: '8px'
                  }}
                >
                  ✅ Validate Algorithm
                </button>
                <button 
                  style={{ 
                    padding: '12px 24px', 
                    backgroundColor: '#1a4d4d', 
                    border: '1px solid #4ade80', 
                    borderRadius: '8px', 
                    color: '#4ade80', 
                    fontWeight: 'bold', 
                    cursor: 'pointer',
                    display: 'flex',
                    alignItems: 'center',
                    gap: '8px'
                  }}
                >
                  🧪 Test with Sample Block
                </button>
                <button 
                  style={{ 
                    padding: '12px 24px', 
                    backgroundColor: 'transparent', 
                    border: '1px solid #888', 
                    borderRadius: '8px', 
                    color: '#888', 
                    cursor: 'pointer' 
                  }}
                >
                  📋 Load from Clipboard
                </button>
              </div>

              <div style={{ marginTop: '15px', padding: '12px', backgroundColor: '#1a2a1a', borderRadius: '6px', border: '1px solid #4ade80' }}>
                <p style={{ margin: 0, color: '#4ade80', fontSize: '0.9rem' }}>
                  💡 <strong>Tip:</strong> After pasting the algorithm, click "Validate Algorithm" to check for syntax errors, 
                  then "Test with Sample Block" to verify it produces valid hashes before saving.
                </p>
              </div>
            </div>

            <div style={adminStyles.algoActions}>
              <button 
                style={adminStyles.algoSaveBtn}
                onClick={handleSaveAlgorithm}
                disabled={savingAlgorithm}
              >
                {savingAlgorithm ? 'Saving...' : '💾 Save Algorithm Settings'}
              </button>
            </div>

            <div style={adminStyles.algoWarning}>
              <strong>⚠️ Important:</strong> After changing algorithm settings, you may need to restart the stratum server 
              for changes to take effect. Notify miners before making algorithm changes as they may need to update their mining software.
            </div>

            {/* Hardware Difficulty Tiers Info */}
            <div style={{ backgroundColor: '#0a0a15', padding: '20px', borderRadius: '12px', marginTop: '20px', border: '1px solid #2a2a4a' }}>
              <h4 style={{ color: '#00d4ff', margin: '0 0 15px' }}>🖥️ Hardware-Aware Difficulty Tiers</h4>
              <p style={{ color: '#888', marginBottom: '15px', fontSize: '0.9rem' }}>
                The pool automatically detects miner hardware and applies appropriate difficulty levels:
              </p>
              <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(150px, 1fr))', gap: '10px' }}>
                <div style={{ backgroundColor: '#1a1a2e', padding: '12px', borderRadius: '8px', textAlign: 'center' }}>
                  <span style={{ display: 'block', fontSize: '1.2rem', marginBottom: '5px' }}>💻</span>
                  <span style={{ color: '#888', fontSize: '0.8rem' }}>CPU</span>
                  <span style={{ display: 'block', color: '#00d4ff', fontWeight: 'bold' }}>Base × 1</span>
                </div>
                <div style={{ backgroundColor: '#1a1a2e', padding: '12px', borderRadius: '8px', textAlign: 'center' }}>
                  <span style={{ display: 'block', fontSize: '1.2rem', marginBottom: '5px' }}>🎮</span>
                  <span style={{ color: '#888', fontSize: '0.8rem' }}>GPU</span>
                  <span style={{ display: 'block', color: '#00d4ff', fontWeight: 'bold' }}>Base × 16</span>
                </div>
                <div style={{ backgroundColor: '#1a1a2e', padding: '12px', borderRadius: '8px', textAlign: 'center' }}>
                  <span style={{ display: 'block', fontSize: '1.2rem', marginBottom: '5px' }}>🔧</span>
                  <span style={{ color: '#888', fontSize: '0.8rem' }}>FPGA</span>
                  <span style={{ display: 'block', color: '#00d4ff', fontWeight: 'bold' }}>Base × 64</span>
                </div>
                <div style={{ backgroundColor: '#1a1a2e', padding: '12px', borderRadius: '8px', textAlign: 'center' }}>
                  <span style={{ display: 'block', fontSize: '1.2rem', marginBottom: '5px' }}>⚡</span>
                  <span style={{ color: '#888', fontSize: '0.8rem' }}>ASIC</span>
                  <span style={{ display: 'block', color: '#00d4ff', fontWeight: 'bold' }}>Base × 256</span>
                </div>
                <div style={{ backgroundColor: '#1a2a3a', padding: '12px', borderRadius: '8px', textAlign: 'center', border: '1px solid #9b59b6' }}>
                  <span style={{ display: 'block', fontSize: '1.2rem', marginBottom: '5px' }}>🔷</span>
                  <span style={{ color: '#9b59b6', fontSize: '0.8rem' }}>X30/X100</span>
                  <span style={{ display: 'block', color: '#9b59b6', fontWeight: 'bold' }}>Base × 1024</span>
                </div>
              </div>
            </div>
          </div>
        )}

        {/* Community Channels Tab */}
        {activeTab === 'community' && (
          <div style={adminStyles.algorithmContainer}>
            <div style={adminStyles.algoHeader}>
              <h3 style={adminStyles.algoTitle}>💬 Community Channel Management</h3>
              <p style={adminStyles.algoDesc}>
                Create and manage community chat channels. Organize channels into categories for better navigation.
              </p>
            </div>

            <div style={{ display: 'flex', gap: '10px', marginBottom: '20px' }}>
              <button 
                style={{ ...adminStyles.algoSaveBtn, padding: '12px 24px', fontSize: '1rem' }}
                onClick={() => setShowCreateCategory(true)}
              >
                ➕ New Category
              </button>
              <button 
                style={{ ...adminStyles.algoSaveBtn, padding: '12px 24px', fontSize: '1rem', backgroundColor: '#00d4ff' }}
                onClick={() => { setShowCreateChannel(true); setEditingChannel(null); setChannelForm({ name: '', description: '', category_id: categories[0]?.id || '', type: 'text', is_read_only: false, admin_only_post: false }); }}
              >
                ➕ New Channel
              </button>
            </div>

            {channelsLoading ? (
              <div style={adminStyles.loading}>Loading channels...</div>
            ) : (
              <div>
                {categories.length === 0 ? (
                  <div style={{ textAlign: 'center', padding: '40px', color: '#888' }}>
                    <p>No categories yet. Create a category first, then add channels to it.</p>
                  </div>
                ) : (
                  categories.map((category: any) => (
                    <div key={category.id} style={{ marginBottom: '25px', backgroundColor: '#0a0a15', borderRadius: '8px', border: '1px solid #2a2a4a', overflow: 'hidden' }}>
                      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', padding: '15px 20px', backgroundColor: '#1a1a2e', borderBottom: '1px solid #2a2a4a' }}>
                        <div>
                          <h4 style={{ margin: 0, color: '#9b59b6' }}>📁 {category.name}</h4>
                          {category.description && <p style={{ margin: '5px 0 0', color: '#888', fontSize: '0.9rem' }}>{category.description}</p>}
                        </div>
                        <button 
                          style={{ ...adminStyles.actionBtn, color: '#ef4444' }}
                          onClick={() => handleDeleteCategory(category.id)}
                          title="Delete Category"
                        >
                          🗑️
                        </button>
                      </div>
                      <div style={{ padding: '15px 20px' }}>
                        {channels.filter((ch: any) => ch.category_id === category.id).length === 0 ? (
                          <p style={{ color: '#666', fontStyle: 'italic', margin: 0 }}>No channels in this category</p>
                        ) : (
                          channels.filter((ch: any) => ch.category_id === category.id).map((channel: any) => (
                            <div key={channel.id} style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', padding: '12px 15px', backgroundColor: '#1a1a2e', borderRadius: '6px', marginBottom: '8px' }}>
                              <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                                <span style={{ fontSize: '1.2rem' }}>
                                  {channel.type === 'announcement' ? '📢' : channel.type === 'regional' ? '🌍' : '💬'}
                                </span>
                                <div>
                                  <span style={{ color: '#00d4ff', fontWeight: 'bold' }}># {channel.name}</span>
                                  {channel.description && <p style={{ margin: '4px 0 0', color: '#888', fontSize: '0.85rem' }}>{channel.description}</p>}
                                  <div style={{ display: 'flex', gap: '8px', marginTop: '4px' }}>
                                    {channel.is_read_only && <span style={{ backgroundColor: '#4d3a1a', color: '#fbbf24', padding: '2px 6px', borderRadius: '4px', fontSize: '0.7rem' }}>Read-Only</span>}
                                    {channel.admin_only_post && <span style={{ backgroundColor: '#4d1a4d', color: '#d946ef', padding: '2px 6px', borderRadius: '4px', fontSize: '0.7rem' }}>Admin Post Only</span>}
                                  </div>
                                </div>
                              </div>
                              <div style={{ display: 'flex', gap: '8px' }}>
                                <button style={adminStyles.actionBtn} onClick={() => openEditChannel(channel)} title="Edit">✏️</button>
                                <button style={{ ...adminStyles.actionBtn, color: '#ef4444' }} onClick={() => handleDeleteChannel(channel.id)} title="Delete">🗑️</button>
                              </div>
                            </div>
                          ))
                        )}
                      </div>
                    </div>
                  ))
                )}
              </div>
            )}

            {/* Create/Edit Channel Modal */}
            {(showCreateChannel || editingChannel) && (
              <div style={adminStyles.editModal}>
                <h3 style={adminStyles.editTitle}>{editingChannel ? 'Edit Channel' : 'Create New Channel'}</h3>
                <div style={adminStyles.formGroup}>
                  <label style={adminStyles.label}>Channel Name *</label>
                  <input 
                    style={adminStyles.formInput} 
                    type="text" 
                    placeholder="e.g., general-chat" 
                    value={channelForm.name} 
                    onChange={e => setChannelForm({...channelForm, name: e.target.value})} 
                  />
                </div>
                <div style={adminStyles.formGroup}>
                  <label style={adminStyles.label}>Description</label>
                  <input 
                    style={adminStyles.formInput} 
                    type="text" 
                    placeholder="What's this channel for?" 
                    value={channelForm.description} 
                    onChange={e => setChannelForm({...channelForm, description: e.target.value})} 
                  />
                </div>
                <div style={adminStyles.formGroup}>
                  <label style={adminStyles.label}>Category *</label>
                  <select 
                    style={adminStyles.algoSelect} 
                    value={channelForm.category_id} 
                    onChange={e => setChannelForm({...channelForm, category_id: e.target.value})}
                  >
                    <option value="">Select a category</option>
                    {categories.map((cat: any) => (
                      <option key={cat.id} value={cat.id}>{cat.name}</option>
                    ))}
                  </select>
                </div>
                <div style={adminStyles.formGroup}>
                  <label style={adminStyles.label}>Channel Type</label>
                  <select 
                    style={adminStyles.algoSelect} 
                    value={channelForm.type} 
                    onChange={e => setChannelForm({...channelForm, type: e.target.value})}
                  >
                    <option value="text">💬 Text Channel</option>
                    <option value="announcement">📢 Announcement Channel</option>
                    <option value="regional">🌍 Regional Channel</option>
                  </select>
                </div>
                <div style={adminStyles.formGroup}>
                  <label style={adminStyles.checkboxLabel}>
                    <input type="checkbox" checked={channelForm.is_read_only} onChange={e => setChannelForm({...channelForm, is_read_only: e.target.checked})} />
                    Read-Only (users can view but not post)
                  </label>
                  <label style={adminStyles.checkboxLabel}>
                    <input type="checkbox" checked={channelForm.admin_only_post} onChange={e => setChannelForm({...channelForm, admin_only_post: e.target.checked})} />
                    Admin-Only Posting
                  </label>
                </div>
                <div style={adminStyles.editActions}>
                  <button style={adminStyles.cancelBtn} onClick={() => { setShowCreateChannel(false); setEditingChannel(null); }}>Cancel</button>
                  <button style={adminStyles.saveBtn} onClick={editingChannel ? handleUpdateChannel : handleCreateChannel}>
                    {editingChannel ? 'Save Changes' : 'Create Channel'}
                  </button>
                </div>
              </div>
            )}

            {/* Create Category Modal */}
            {showCreateCategory && (
              <div style={adminStyles.editModal}>
                <h3 style={adminStyles.editTitle}>Create New Category</h3>
                <div style={adminStyles.formGroup}>
                  <label style={adminStyles.label}>Category Name *</label>
                  <input 
                    style={adminStyles.formInput} 
                    type="text" 
                    placeholder="e.g., General, Mining Talk, Support" 
                    value={categoryForm.name} 
                    onChange={e => setCategoryForm({...categoryForm, name: e.target.value})} 
                  />
                </div>
                <div style={adminStyles.formGroup}>
                  <label style={adminStyles.label}>Description</label>
                  <input 
                    style={adminStyles.formInput} 
                    type="text" 
                    placeholder="What topics belong in this category?" 
                    value={categoryForm.description} 
                    onChange={e => setCategoryForm({...categoryForm, description: e.target.value})} 
                  />
                </div>
                <div style={adminStyles.editActions}>
                  <button style={adminStyles.cancelBtn} onClick={() => setShowCreateCategory(false)}>Cancel</button>
                  <button style={adminStyles.saveBtn} onClick={handleCreateCategory}>Create Category</button>
                </div>
              </div>
            )}
          </div>
        )}

        {/* Role Management Tab */}
        {activeTab === 'roles' && (
          <div style={adminStyles.algorithmContainer}>
            <div style={adminStyles.algoHeader}>
              <h3 style={adminStyles.algoTitle}>👑 Role Management</h3>
              <p style={adminStyles.algoDesc}>
                Manage user roles and permissions. Promote users to moderators or admins.
              </p>
            </div>

            {rolesLoading ? (
              <div style={adminStyles.loading}>Loading roles...</div>
            ) : (
              <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(400px, 1fr))', gap: '20px' }}>
                {/* Admins Section */}
                <div style={{ backgroundColor: '#0a0a15', borderRadius: '8px', border: '1px solid #9b59b6', overflow: 'hidden' }}>
                  <div style={{ padding: '15px 20px', backgroundColor: '#1a1a2e', borderBottom: '1px solid #9b59b6' }}>
                    <h4 style={{ margin: 0, color: '#9b59b6', display: 'flex', alignItems: 'center', gap: '8px' }}>
                      👑 Administrators ({admins.length})
                    </h4>
                    <p style={{ margin: '5px 0 0', color: '#888', fontSize: '0.85rem' }}>
                      Full access to all pool settings and user management
                    </p>
                  </div>
                  <div style={{ padding: '15px 20px', maxHeight: '300px', overflowY: 'auto' }}>
                    {admins.length === 0 ? (
                      <p style={{ color: '#666', fontStyle: 'italic', margin: 0 }}>No administrators assigned</p>
                    ) : (
                      admins.map((admin: any) => (
                        <div key={admin.id} style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', padding: '10px 12px', backgroundColor: '#1a1a2e', borderRadius: '6px', marginBottom: '8px' }}>
                          <div>
                            <span style={{ color: '#9b59b6', fontWeight: 'bold' }}>{admin.username}</span>
                            <p style={{ margin: '2px 0 0', color: '#888', fontSize: '0.8rem' }}>{admin.email}</p>
                            <span style={{ backgroundColor: admin.role === 'super_admin' ? '#4d1a4d' : '#2a2a4a', color: admin.role === 'super_admin' ? '#d946ef' : '#9b59b6', padding: '2px 6px', borderRadius: '4px', fontSize: '0.7rem' }}>
                              {admin.role === 'super_admin' ? '⭐ Super Admin' : 'Admin'}
                            </span>
                          </div>
                          <button 
                            style={{ ...adminStyles.actionBtn, color: '#fbbf24' }}
                            onClick={() => { setRoleChangeUser(admin); setNewRole('user'); }}
                            title="Change Role"
                          >
                            ✏️
                          </button>
                        </div>
                      ))
                    )}
                  </div>
                </div>

                {/* Moderators Section */}
                <div style={{ backgroundColor: '#0a0a15', borderRadius: '8px', border: '1px solid #00d4ff', overflow: 'hidden' }}>
                  <div style={{ padding: '15px 20px', backgroundColor: '#1a1a2e', borderBottom: '1px solid #00d4ff' }}>
                    <h4 style={{ margin: 0, color: '#00d4ff', display: 'flex', alignItems: 'center', gap: '8px' }}>
                      🛡️ Moderators ({moderators.length})
                    </h4>
                    <p style={{ margin: '5px 0 0', color: '#888', fontSize: '0.85rem' }}>
                      Can manage community channels, mute users, and view reports
                    </p>
                  </div>
                  <div style={{ padding: '15px 20px', maxHeight: '300px', overflowY: 'auto' }}>
                    {moderators.length === 0 ? (
                      <p style={{ color: '#666', fontStyle: 'italic', margin: 0 }}>No moderators assigned</p>
                    ) : (
                      moderators.map((mod: any) => (
                        <div key={mod.id} style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', padding: '10px 12px', backgroundColor: '#1a1a2e', borderRadius: '6px', marginBottom: '8px' }}>
                          <div>
                            <span style={{ color: '#00d4ff', fontWeight: 'bold' }}>{mod.username}</span>
                            <p style={{ margin: '2px 0 0', color: '#888', fontSize: '0.8rem' }}>{mod.email}</p>
                          </div>
                          <button 
                            style={{ ...adminStyles.actionBtn, color: '#fbbf24' }}
                            onClick={() => { setRoleChangeUser(mod); setNewRole('user'); }}
                            title="Change Role"
                          >
                            ✏️
                          </button>
                        </div>
                      ))
                    )}
                  </div>
                </div>
              </div>
            )}

            {/* Promote User Section */}
            <div style={{ marginTop: '25px', backgroundColor: '#0a0a15', borderRadius: '8px', border: '1px solid #4ade80', padding: '20px' }}>
              <h4 style={{ margin: '0 0 15px', color: '#4ade80' }}>➕ Promote a User</h4>
              <p style={{ color: '#888', fontSize: '0.9rem', marginBottom: '15px' }}>
                Search for a user from the User Management tab, then return here to promote them.
              </p>
              <div style={{ display: 'flex', gap: '10px', flexWrap: 'wrap' }}>
                <button 
                  style={{ ...adminStyles.algoSaveBtn, backgroundColor: '#00d4ff' }}
                  onClick={() => setActiveTab('users')}
                >
                  👥 Go to User Management
                </button>
              </div>
            </div>

            {/* Role Hierarchy Info */}
            <div style={{ marginTop: '25px', backgroundColor: '#1a1a2e', borderRadius: '8px', padding: '20px' }}>
              <h4 style={{ margin: '0 0 15px', color: '#fbbf24' }}>📋 Role Hierarchy</h4>
              <div style={{ display: 'grid', gap: '12px' }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                  <span style={{ backgroundColor: '#4d1a4d', color: '#d946ef', padding: '4px 10px', borderRadius: '4px', fontSize: '0.85rem', minWidth: '120px', textAlign: 'center' }}>⭐ Super Admin</span>
                  <span style={{ color: '#888', fontSize: '0.9rem' }}>Full access, can promote/demote admins</span>
                </div>
                <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                  <span style={{ backgroundColor: '#2a1a4a', color: '#9b59b6', padding: '4px 10px', borderRadius: '4px', fontSize: '0.85rem', minWidth: '120px', textAlign: 'center' }}>👑 Admin</span>
                  <span style={{ color: '#888', fontSize: '0.9rem' }}>Pool settings, ban users, promote moderators</span>
                </div>
                <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                  <span style={{ backgroundColor: '#1a3a4a', color: '#00d4ff', padding: '4px 10px', borderRadius: '4px', fontSize: '0.85rem', minWidth: '120px', textAlign: 'center' }}>🛡️ Moderator</span>
                  <span style={{ color: '#888', fontSize: '0.9rem' }}>Manage channels, mute users, view reports</span>
                </div>
                <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                  <span style={{ backgroundColor: '#2a2a4a', color: '#888', padding: '4px 10px', borderRadius: '4px', fontSize: '0.85rem', minWidth: '120px', textAlign: 'center' }}>👤 User</span>
                  <span style={{ color: '#888', fontSize: '0.9rem' }}>Standard mining pool access</span>
                </div>
              </div>
            </div>

            {/* Role Change Modal */}
            {roleChangeUser && (
              <div style={adminStyles.editModal}>
                <h3 style={adminStyles.editTitle}>Change Role: {roleChangeUser.username}</h3>
                <p style={{ color: '#888', marginBottom: '15px' }}>
                  Current role: <strong style={{ color: '#00d4ff' }}>{roleChangeUser.role || 'user'}</strong>
                </p>
                <div style={adminStyles.formGroup}>
                  <label style={adminStyles.label}>New Role</label>
                  <select 
                    style={adminStyles.algoSelect} 
                    value={newRole} 
                    onChange={e => setNewRole(e.target.value)}
                  >
                    <option value="">Select a role</option>
                    <option value="user">👤 User</option>
                    <option value="moderator">🛡️ Moderator</option>
                    <option value="admin">👑 Admin</option>
                    <option value="super_admin">⭐ Super Admin</option>
                  </select>
                </div>
                <div style={adminStyles.editActions}>
                  <button style={adminStyles.cancelBtn} onClick={() => { setRoleChangeUser(null); setNewRole(''); }}>Cancel</button>
                  <button style={adminStyles.saveBtn} onClick={handleChangeRole} disabled={!newRole}>Change Role</button>
                </div>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
}

function formatHashrate(hashrate: number): string {
  if (hashrate >= 1e12) return (hashrate / 1e12).toFixed(2) + ' TH/s';
  if (hashrate >= 1e9) return (hashrate / 1e9).toFixed(2) + ' GH/s';
  if (hashrate >= 1e6) return (hashrate / 1e6).toFixed(2) + ' MH/s';
  if (hashrate >= 1e3) return (hashrate / 1e3).toFixed(2) + ' KH/s';
  return hashrate.toFixed(2) + ' H/s';
}

const styles: { [key: string]: React.CSSProperties } = {
  container: { minHeight: '100vh', backgroundColor: '#0a0a0f', color: '#e0e0e0', fontFamily: "'Segoe UI', Tahoma, Geneva, Verdana, sans-serif" },
  header: { background: 'linear-gradient(135deg, #1a1a2e 0%, #16213e 100%)', padding: '20px', borderBottom: '2px solid #00d4ff' },
  headerContent: { maxWidth: '1200px', margin: '0 auto', display: 'flex', justifyContent: 'space-between', alignItems: 'center', flexWrap: 'wrap' as const, gap: '20px' },
  title: { fontSize: '2.5rem', margin: 0, color: '#00d4ff', textShadow: '0 0 20px rgba(0, 212, 255, 0.5)' },
  subtitle: { fontSize: '1.2rem', color: '#888', margin: '10px 0 0' },
  authButtons: { display: 'flex', gap: '10px', alignItems: 'center' },
  authBtn: { padding: '10px 20px', backgroundColor: 'transparent', border: '1px solid #00d4ff', color: '#00d4ff', borderRadius: '6px', cursor: 'pointer', fontSize: '0.9rem' },
  registerBtn: { backgroundColor: '#00d4ff', color: '#0a0a0f' },
  userInfo: { display: 'flex', alignItems: 'center', gap: '15px' },
  username: { color: '#00d4ff', fontSize: '1rem' },
  logoutBtn: { padding: '8px 16px', backgroundColor: '#4d1a1a', border: 'none', color: '#ff6666', borderRadius: '6px', cursor: 'pointer', fontSize: '0.9rem' },
  message: { padding: '15px 20px', textAlign: 'center', color: '#fff', fontSize: '0.95rem' },
  main: { maxWidth: '1200px', margin: '0 auto', padding: '40px 20px' },
  loading: { textAlign: 'center', padding: '40px', color: '#00d4ff' },
  error: { textAlign: 'center', padding: '40px', color: '#ff4444' },
  statsGrid: { display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))', gap: '20px', marginBottom: '40px' },
  statCard: { background: 'linear-gradient(135deg, #1a1a2e 0%, #0f0f1a 100%)', borderRadius: '12px', padding: '24px', border: '1px solid #2a2a4a', textAlign: 'center' },
  statLabel: { fontSize: '0.9rem', color: '#888', margin: '0 0 8px', textTransform: 'uppercase' },
  statValue: { fontSize: '1.5rem', color: '#00d4ff', margin: 0, fontWeight: 'bold' },
  section: { background: 'linear-gradient(135deg, #1a1a2e 0%, #0f0f1a 100%)', borderRadius: '12px', padding: '24px', border: '1px solid #2a2a4a', marginBottom: '20px' },
  sectionTitle: { fontSize: '1.3rem', color: '#00d4ff', margin: '0 0 16px' },
  connectionInfo: { backgroundColor: '#0a0a15', padding: '16px', borderRadius: '8px' },
  code: { display: 'block', backgroundColor: '#000', color: '#00ff88', padding: '12px 16px', borderRadius: '6px', fontFamily: 'monospace', fontSize: '1.1rem', margin: '8px 0' },
  hint: { color: '#888', fontSize: '0.9rem', margin: '8px 0 0' },
  info: { lineHeight: '1.8' },
  infoGrid: { display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(180px, 1fr))', gap: '15px' },
  infoCard: { backgroundColor: '#0a0a15', padding: '20px', borderRadius: '10px', textAlign: 'center', border: '1px solid #2a2a4a', display: 'flex', flexDirection: 'column', alignItems: 'center', gap: '8px' },
  infoIcon: { fontSize: '1.8rem' },
  infoLabel: { color: '#888', fontSize: '0.85rem', textTransform: 'uppercase', letterSpacing: '0.5px' },
  infoValue: { color: '#00d4ff', fontSize: '0.95rem', fontWeight: 'bold' },
  footer: { textAlign: 'center', padding: '30px 20px', borderTop: '1px solid #2a2a4a', color: '#666' },
  footerLinks: { marginTop: '10px' },
  link: { color: '#00d4ff', textDecoration: 'none' },
  modalOverlay: { position: 'fixed' as const, top: 0, left: 0, right: 0, bottom: 0, backgroundColor: 'rgba(0, 0, 0, 0.8)', display: 'flex', justifyContent: 'center', alignItems: 'center', zIndex: 1000 },
  modal: { backgroundColor: '#1a1a2e', padding: '40px', borderRadius: '12px', border: '1px solid #2a2a4a', width: '100%', maxWidth: '400px', position: 'relative' as const },
  closeBtn: { position: 'absolute' as const, top: '15px', right: '15px', background: 'none', border: 'none', color: '#888', fontSize: '24px', cursor: 'pointer' },
  modalTitle: { color: '#00d4ff', marginBottom: '20px', textAlign: 'center' },
  modalDesc: { color: '#888', fontSize: '0.9rem', marginBottom: '20px', textAlign: 'center' },
  input: { width: '100%', padding: '12px 16px', marginBottom: '15px', backgroundColor: '#0a0a15', border: '1px solid #2a2a4a', borderRadius: '6px', color: '#e0e0e0', fontSize: '1rem', boxSizing: 'border-box' as const },
  submitBtn: { width: '100%', padding: '12px', backgroundColor: '#00d4ff', border: 'none', borderRadius: '6px', color: '#0a0a0f', fontSize: '1rem', fontWeight: 'bold', cursor: 'pointer', marginTop: '10px' },
  errorMsg: { backgroundColor: '#4d1a1a', color: '#ff6666', padding: '10px', borderRadius: '6px', marginBottom: '15px', fontSize: '0.9rem', textAlign: 'center' },
  authLinks: { display: 'flex', justifyContent: 'space-between', marginTop: '20px', flexWrap: 'wrap' as const, gap: '10px' },
  authLink: { color: '#00d4ff', fontSize: '0.9rem', cursor: 'pointer', textDecoration: 'underline' },
};

const adminStyles: { [key: string]: React.CSSProperties } = {
  overlay: { position: 'fixed', top: 0, left: 0, right: 0, bottom: 0, backgroundColor: 'rgba(0,0,0,0.9)', display: 'flex', justifyContent: 'center', alignItems: 'flex-start', padding: '20px', zIndex: 2000, overflowY: 'auto' },
  panel: { backgroundColor: '#1a1a2e', borderRadius: '12px', width: '100%', maxWidth: '1200px', maxHeight: '90vh', overflow: 'auto', position: 'relative' },
  header: { display: 'flex', justifyContent: 'space-between', alignItems: 'center', padding: '20px', borderBottom: '1px solid #2a2a4a' },
  title: { color: '#9b59b6', margin: 0, fontSize: '1.5rem' },
  closeBtn: { background: 'none', border: 'none', color: '#888', fontSize: '28px', cursor: 'pointer' },
  tabs: { display: 'flex', borderBottom: '1px solid #2a2a4a', padding: '0 20px' },
  tab: { padding: '15px 25px', backgroundColor: 'transparent', border: 'none', color: '#888', fontSize: '1rem', cursor: 'pointer', borderBottom: '3px solid transparent', marginBottom: '-1px' },
  tabActive: { color: '#9b59b6', borderBottomColor: '#9b59b6' },
  searchBar: { padding: '15px 20px' },
  searchInput: { width: '100%', padding: '12px', backgroundColor: '#0a0a15', border: '1px solid #2a2a4a', borderRadius: '6px', color: '#e0e0e0', fontSize: '1rem', boxSizing: 'border-box' as const },
  loading: { padding: '40px', textAlign: 'center', color: '#00d4ff' },
  tableContainer: { overflowX: 'auto', padding: '0 20px' },
  table: { width: '100%', borderCollapse: 'collapse' },
  th: { padding: '12px', textAlign: 'left', borderBottom: '2px solid #2a2a4a', color: '#00d4ff', fontSize: '0.85rem', textTransform: 'uppercase' },
  tr: { borderBottom: '1px solid #2a2a4a' },
  td: { padding: '12px', color: '#e0e0e0' },
  adminBadge: { color: '#f1c40f' },
  activeBadge: { backgroundColor: '#1a4d1a', color: '#4ade80', padding: '4px 8px', borderRadius: '4px', fontSize: '0.8rem' },
  inactiveBadge: { backgroundColor: '#4d1a1a', color: '#f87171', padding: '4px 8px', borderRadius: '4px', fontSize: '0.8rem' },
  actionBtn: { background: 'none', border: 'none', cursor: 'pointer', fontSize: '1.1rem', padding: '4px 8px' },
  pagination: { display: 'flex', justifyContent: 'center', alignItems: 'center', gap: '20px', padding: '20px' },
  pageBtn: { padding: '8px 16px', backgroundColor: '#2a2a4a', border: 'none', borderRadius: '4px', color: '#e0e0e0', cursor: 'pointer' },
  pageInfo: { color: '#888' },
  editModal: { position: 'absolute', top: '50%', left: '50%', transform: 'translate(-50%, -50%)', backgroundColor: '#0a0a15', padding: '30px', borderRadius: '12px', border: '1px solid #9b59b6', minWidth: '400px', zIndex: 10 },
  editTitle: { color: '#9b59b6', marginTop: 0 },
  formGroup: { marginBottom: '15px' },
  label: { display: 'block', color: '#888', marginBottom: '5px', fontSize: '0.9rem' },
  formInput: { width: '100%', padding: '10px', backgroundColor: '#1a1a2e', border: '1px solid #2a2a4a', borderRadius: '4px', color: '#e0e0e0', boxSizing: 'border-box' as const },
  checkboxLabel: { display: 'inline-flex', alignItems: 'center', gap: '8px', color: '#e0e0e0', marginRight: '20px' },
  editActions: { display: 'flex', gap: '10px', marginTop: '20px' },
  cancelBtn: { flex: 1, padding: '10px', backgroundColor: '#2a2a4a', border: 'none', borderRadius: '4px', color: '#e0e0e0', cursor: 'pointer' },
  saveBtn: { flex: 1, padding: '10px', backgroundColor: '#9b59b6', border: 'none', borderRadius: '4px', color: '#fff', cursor: 'pointer' },
  detailModal: { position: 'absolute', top: '10px', right: '10px', bottom: '10px', width: '400px', backgroundColor: '#0a0a15', borderRadius: '8px', border: '1px solid #2a2a4a', padding: '20px', overflowY: 'auto' },
  closeDetailBtn: { position: 'absolute', top: '10px', right: '10px', background: 'none', border: 'none', color: '#888', fontSize: '24px', cursor: 'pointer' },
  detailTitle: { color: '#00d4ff', marginTop: 0 },
  detailCard: { backgroundColor: '#1a1a2e', padding: '15px', borderRadius: '8px', marginBottom: '15px' },
  subTitle: { color: '#00d4ff', marginTop: '20px', marginBottom: '10px' },
  minerRow: { display: 'flex', justifyContent: 'space-between', padding: '8px', borderBottom: '1px solid #2a2a4a', backgroundColor: '#1a1a2e', marginBottom: '5px', borderRadius: '4px' },
  algorithmContainer: { padding: '20px' },
  algoHeader: { marginBottom: '25px' },
  algoTitle: { color: '#9b59b6', marginTop: 0, marginBottom: '10px' },
  algoDesc: { color: '#888', margin: 0 },
  algoGrid: { display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(280px, 1fr))', gap: '20px', marginBottom: '25px' },
  algoCard: { backgroundColor: '#0a0a15', padding: '20px', borderRadius: '8px', border: '1px solid #2a2a4a' },
  algoLabel: { display: 'block', color: '#00d4ff', marginBottom: '10px', fontSize: '0.9rem', textTransform: 'uppercase' },
  algoSelect: { width: '100%', padding: '12px', backgroundColor: '#1a1a2e', border: '1px solid #2a2a4a', borderRadius: '6px', color: '#e0e0e0', fontSize: '1rem' },
  algoInput: { width: '100%', padding: '12px', backgroundColor: '#1a1a2e', border: '1px solid #2a2a4a', borderRadius: '6px', color: '#e0e0e0', fontSize: '1rem', boxSizing: 'border-box' as const },
  algoTextarea: { width: '100%', padding: '12px', backgroundColor: '#1a1a2e', border: '1px solid #2a2a4a', borderRadius: '6px', color: '#e0e0e0', fontSize: '0.9rem', fontFamily: 'monospace', resize: 'vertical' as const, boxSizing: 'border-box' as const },
  algoHint: { color: '#666', fontSize: '0.85rem', margin: '8px 0 0' },
  algoActions: { textAlign: 'center', marginBottom: '20px' },
  algoSaveBtn: { padding: '15px 40px', backgroundColor: '#9b59b6', border: 'none', borderRadius: '8px', color: '#fff', fontSize: '1.1rem', fontWeight: 'bold', cursor: 'pointer' },
  algoWarning: { backgroundColor: '#4d3a1a', border: '1px solid #f59e0b', borderRadius: '8px', padding: '15px', color: '#fbbf24', fontSize: '0.9rem', lineHeight: '1.5' },
};

export default App;
