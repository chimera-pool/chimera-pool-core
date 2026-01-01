import React, { useState, useEffect } from 'react';
import {
  ResponsiveContainer,
  BarChart,
  Bar,
  PieChart,
  Pie,
  Cell,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend
} from 'recharts';
import { formatHashrate } from '../../../utils/formatters';

interface AdminMinersTabProps {
  token: string;
  isActive: boolean;
  showMessage: (type: 'success' | 'error', text: string) => void;
}

const graphStyles: { [key: string]: React.CSSProperties } = {
  chartCard: { backgroundColor: '#0a0a15', borderRadius: '10px', padding: '20px', border: '1px solid #2a2a4a' },
  chartTitle: { color: '#e0e0e0', fontSize: '1rem', marginTop: 0, marginBottom: '15px' },
};

const styles: { [key: string]: React.CSSProperties } = {
  container: { padding: '20px' },
  loading: { padding: '40px', textAlign: 'center', color: '#D4A84B' },
  tableContainer: { overflowX: 'auto' },
  table: { width: '100%', borderCollapse: 'collapse' },
  th: { padding: '14px', textAlign: 'left', borderBottom: '2px solid rgba(74, 44, 90, 0.5)', color: '#D4A84B', fontSize: '0.8rem', textTransform: 'uppercase', letterSpacing: '0.03em', fontWeight: 600 },
  tr: { borderBottom: '1px solid rgba(74, 44, 90, 0.3)', transition: 'background 0.2s' },
  td: { padding: '14px', color: '#F0EDF4' },
  activeBadge: { background: 'rgba(74, 222, 128, 0.15)', color: '#4ade80', padding: '4px 10px', borderRadius: '6px', fontSize: '0.8rem', border: '1px solid rgba(74, 222, 128, 0.3)' },
  inactiveBadge: { background: 'rgba(248, 113, 113, 0.15)', color: '#f87171', padding: '4px 10px', borderRadius: '6px', fontSize: '0.8rem', border: '1px solid rgba(248, 113, 113, 0.3)' },
  actionBtn: { background: 'none', border: 'none', cursor: 'pointer', fontSize: '1.1rem', padding: '4px 8px', transition: 'transform 0.2s' },
  pagination: { display: 'flex', justifyContent: 'center', alignItems: 'center', gap: '20px', padding: '20px' },
  pageBtn: { padding: '10px 18px', background: 'rgba(74, 44, 90, 0.4)', border: 'none', borderRadius: '8px', color: '#F0EDF4', cursor: 'pointer', transition: 'all 0.2s' },
  pageInfo: { color: '#B8B4C8' },
  searchBar: { padding: '16px 0' },
  searchInput: { width: '100%', padding: '12px 16px', backgroundColor: 'rgba(13, 8, 17, 0.8)', border: '1px solid rgba(74, 44, 90, 0.5)', borderRadius: '10px', color: '#F0EDF4', fontSize: '1rem', boxSizing: 'border-box' as const },
  algoCard: { background: 'rgba(13, 8, 17, 0.6)', padding: '20px', borderRadius: '12px', border: '1px solid rgba(74, 44, 90, 0.4)' },
};

export function AdminMinersTab({ token, isActive, showMessage }: AdminMinersTabProps) {
  const [allMiners, setAllMiners] = useState<any[]>([]);
  const [minersLoading, setMinersLoading] = useState(false);
  const [minerSearch, setMinerSearch] = useState('');
  const [minerPage, setMinerPage] = useState(1);
  const [minerTotal, setMinerTotal] = useState(0);
  const [activeMinersOnly, setActiveMinersOnly] = useState(false);
  const [selectedMiner, setSelectedMiner] = useState<any>(null);
  const [selectedUserMiners, setSelectedUserMiners] = useState<any>(null);

  useEffect(() => {
    if (isActive) {
      fetchAllMiners();
    }
  }, [isActive, minerPage, minerSearch, activeMinersOnly]);

  const fetchAllMiners = async () => {
    setMinersLoading(true);
    try {
      const params = new URLSearchParams({
        page: minerPage.toString(),
        limit: '20',
        ...(minerSearch && { search: minerSearch }),
        ...(activeMinersOnly && { active_only: 'true' })
      });
      const response = await fetch(`/api/v1/admin/monitoring/miners?${params}`, {
        headers: { 'Authorization': `Bearer ${token}` }
      });
      if (response.ok) {
        const data = await response.json();
        setAllMiners(data.miners || []);
        setMinerTotal(data.total || 0);
      }
    } catch (error) {
      console.error('Failed to fetch miners:', error);
    } finally {
      setMinersLoading(false);
    }
  };

  const fetchMinerDetail = async (minerId: number) => {
    try {
      const response = await fetch(`/api/v1/admin/monitoring/miners/${minerId}`, {
        headers: { 'Authorization': `Bearer ${token}` }
      });
      if (response.ok) {
        const data = await response.json();
        setSelectedMiner(data);
      }
    } catch (error) {
      console.error('Failed to fetch miner detail:', error);
    }
  };

  if (!isActive) return null;

  return (
    <div style={styles.container}>
      {selectedMiner ? (
        // Miner Detail View
        <div>
          <button 
            style={{ marginBottom: '20px', padding: '8px 16px', backgroundColor: '#2a2a4a', border: 'none', borderRadius: '6px', color: '#e0e0e0', cursor: 'pointer' }}
            onClick={() => setSelectedMiner(null)}
          >
            ← Back to Miners List
          </button>

          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(300px, 1fr))', gap: '20px' }}>
            {/* Miner Info Card */}
            <div style={{ ...styles.algoCard, borderColor: selectedMiner.is_active ? '#4ade80' : '#f87171' }}>
              <h3 style={{ color: '#00d4ff', marginTop: 0 }}>⛏️ {selectedMiner.name}</h3>
              <p><strong>User:</strong> {selectedMiner.username} (ID: {selectedMiner.user_id})</p>
              <p><strong>IP Address:</strong> {selectedMiner.address || 'Unknown'}</p>
              <p><strong>Status:</strong> <span style={selectedMiner.is_active ? styles.activeBadge : styles.inactiveBadge}>{selectedMiner.is_active ? '🟢 Active' : '🔴 Offline'}</span></p>
              <p><strong>Connection:</strong> {selectedMiner.connection_duration}</p>
              <p><strong>Uptime (24h):</strong> {selectedMiner.uptime_percent?.toFixed(1)}%</p>
              <p><strong>Last Seen:</strong> {new Date(selectedMiner.last_seen).toLocaleString()}</p>
            </div>

            {/* Hashrate Card */}
            <div style={styles.algoCard}>
              <h3 style={{ color: '#9b59b6', marginTop: 0 }}>⚡ Performance</h3>
              <p><strong>Reported Hashrate:</strong> {formatHashrate(selectedMiner.hashrate)}</p>
              <p><strong>Effective Hashrate:</strong> {formatHashrate(selectedMiner.performance?.effective_hashrate || 0)}</p>
              <p><strong>Efficiency:</strong> {selectedMiner.performance?.efficiency_percent?.toFixed(1) || 0}%</p>
              <p><strong>Shares/Min:</strong> {selectedMiner.performance?.shares_per_minute?.toFixed(2) || 0}</p>
              <p><strong>Avg Share Time:</strong> {selectedMiner.performance?.avg_share_time_seconds?.toFixed(1) || 0}s</p>
              <p><strong>Est. Daily Shares:</strong> {selectedMiner.performance?.estimated_daily_shares || 0}</p>
            </div>

            {/* Share Stats Card */}
            <div style={styles.algoCard}>
              <h3 style={{ color: '#4ade80', marginTop: 0 }}>📊 Share Statistics</h3>
              <p><strong>Total Shares:</strong> {selectedMiner.share_stats?.total_shares || 0}</p>
              <p><strong>Valid:</strong> <span style={{ color: '#4ade80' }}>{selectedMiner.share_stats?.valid_shares || 0}</span></p>
              <p><strong>Invalid:</strong> <span style={{ color: '#f87171' }}>{selectedMiner.share_stats?.invalid_shares || 0}</span></p>
              <p><strong>Acceptance Rate:</strong> {selectedMiner.share_stats?.acceptance_rate?.toFixed(2) || 0}%</p>
              <p><strong>Last Hour:</strong> {selectedMiner.share_stats?.last_hour || 0}</p>
              <p><strong>Last 24h:</strong> {selectedMiner.share_stats?.last_24_hours || 0}</p>
              <p><strong>Avg Difficulty:</strong> {selectedMiner.share_stats?.avg_difficulty?.toFixed(4) || 0}</p>
            </div>

            {/* Troubleshooting Card */}
            <div style={{ ...styles.algoCard, borderColor: '#f59e0b' }}>
              <h3 style={{ color: '#f59e0b', marginTop: 0 }}>🔧 Troubleshooting</h3>
              {selectedMiner.share_stats?.acceptance_rate < 95 && (
                <div style={{ backgroundColor: '#4d2a1a', padding: '10px', borderRadius: '6px', marginBottom: '10px' }}>
                  <strong style={{ color: '#f87171' }}>⚠️ Low Acceptance Rate</strong>
                  <p style={{ margin: '5px 0 0', color: '#fbbf24', fontSize: '0.9rem' }}>
                    {(100 - (selectedMiner.share_stats?.acceptance_rate || 0)).toFixed(1)}% of shares are invalid. Check miner configuration.
                  </p>
                </div>
              )}
              {!selectedMiner.is_active && (
                <div style={{ backgroundColor: '#4d1a1a', padding: '10px', borderRadius: '6px', marginBottom: '10px' }}>
                  <strong style={{ color: '#f87171' }}>🔴 Miner Offline</strong>
                  <p style={{ margin: '5px 0 0', color: '#fbbf24', fontSize: '0.9rem' }}>
                    Last seen: {new Date(selectedMiner.last_seen).toLocaleString()}
                  </p>
                </div>
              )}
              {selectedMiner.performance?.efficiency_percent < 80 && selectedMiner.performance?.efficiency_percent > 0 && (
                <div style={{ backgroundColor: '#4d3a1a', padding: '10px', borderRadius: '6px', marginBottom: '10px' }}>
                  <strong style={{ color: '#fbbf24' }}>⚡ Low Efficiency</strong>
                  <p style={{ margin: '5px 0 0', color: '#fbbf24', fontSize: '0.9rem' }}>
                    Effective hashrate is only {selectedMiner.performance?.efficiency_percent?.toFixed(1)}% of reported. Possible network issues.
                  </p>
                </div>
              )}
              {selectedMiner.share_stats?.acceptance_rate >= 95 && selectedMiner.is_active && (
                <div style={{ backgroundColor: '#1a4d1a', padding: '10px', borderRadius: '6px' }}>
                  <strong style={{ color: '#4ade80' }}>✅ Miner Healthy</strong>
                  <p style={{ margin: '5px 0 0', color: '#4ade80', fontSize: '0.9rem' }}>
                    No issues detected. Miner is operating normally.
                  </p>
                </div>
              )}
            </div>
          </div>

          {/* Visual Charts Section */}
          <h3 style={{ color: '#00d4ff', marginTop: '30px' }}>📊 Visual Analytics</h3>
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(350px, 1fr))', gap: '20px', marginBottom: '30px' }}>
            
            {/* Share Distribution Pie Chart */}
            <div style={graphStyles.chartCard}>
              <h4 style={graphStyles.chartTitle}>Share Distribution</h4>
              <ResponsiveContainer width="100%" height={250}>
                <PieChart>
                  <Pie
                    data={[
                      { name: 'Valid', value: selectedMiner.share_stats?.valid_shares || 0, color: '#4ade80' },
                      { name: 'Invalid', value: selectedMiner.share_stats?.invalid_shares || 0, color: '#f87171' }
                    ]}
                    cx="50%"
                    cy="50%"
                    innerRadius={60}
                    outerRadius={90}
                    paddingAngle={5}
                    dataKey="value"
                    label={({ name, percent }) => `${name}: ${(percent * 100).toFixed(1)}%`}
                  >
                    <Cell fill="#4ade80" />
                    <Cell fill="#f87171" />
                  </Pie>
                  <Tooltip 
                    contentStyle={{ backgroundColor: '#1a1a2e', border: '1px solid #2a2a4a', borderRadius: '8px' }}
                    formatter={(value: number) => [value, 'Shares']}
                  />
                  <Legend />
                </PieChart>
              </ResponsiveContainer>
            </div>

            {/* Share Timeline Bar Chart */}
            <div style={graphStyles.chartCard}>
              <h4 style={graphStyles.chartTitle}>Recent Share Activity</h4>
              <ResponsiveContainer width="100%" height={250}>
                <BarChart data={
                  (selectedMiner.recent_shares || []).slice(0, 10).reverse().map((share: any, idx: number) => ({
                    name: `#${idx + 1}`,
                    difficulty: share.difficulty,
                    valid: share.is_valid ? share.difficulty : 0,
                    invalid: !share.is_valid ? share.difficulty : 0
                  }))
                }>
                  <CartesianGrid strokeDasharray="3 3" stroke="#2a2a4a" />
                  <XAxis dataKey="name" stroke="#888" fontSize={12} />
                  <YAxis stroke="#888" fontSize={12} />
                  <Tooltip 
                    contentStyle={{ backgroundColor: '#1a1a2e', border: '1px solid #2a2a4a', borderRadius: '8px' }}
                    formatter={(value: number) => [value.toFixed(4), 'Difficulty']}
                  />
                  <Bar dataKey="valid" stackId="a" fill="#4ade80" name="Valid" />
                  <Bar dataKey="invalid" stackId="a" fill="#f87171" name="Invalid" />
                </BarChart>
              </ResponsiveContainer>
            </div>

            {/* Performance Gauge */}
            <div style={graphStyles.chartCard}>
              <h4 style={graphStyles.chartTitle}>Efficiency Breakdown</h4>
              <ResponsiveContainer width="100%" height={250}>
                <BarChart
                  layout="vertical"
                  data={[
                    { name: 'Acceptance', value: selectedMiner.share_stats?.acceptance_rate || 0, fill: '#4ade80' },
                    { name: 'Efficiency', value: selectedMiner.performance?.efficiency_percent || 0, fill: '#00d4ff' },
                    { name: 'Uptime (24h)', value: selectedMiner.uptime_percent || 0, fill: '#9b59b6' }
                  ]}
                  margin={{ left: 20, right: 30 }}
                >
                  <CartesianGrid strokeDasharray="3 3" stroke="#2a2a4a" />
                  <XAxis type="number" domain={[0, 100]} stroke="#888" fontSize={12} tickFormatter={(v) => `${v}%`} />
                  <YAxis type="category" dataKey="name" stroke="#888" fontSize={12} width={80} />
                  <Tooltip 
                    contentStyle={{ backgroundColor: '#1a1a2e', border: '1px solid #2a2a4a', borderRadius: '8px' }}
                    formatter={(value: number) => [`${value.toFixed(1)}%`, 'Value']}
                  />
                  <Bar dataKey="value" radius={[0, 4, 4, 0]}>
                    {[
                      { name: 'Acceptance', fill: '#4ade80' },
                      { name: 'Efficiency', fill: '#00d4ff' },
                      { name: 'Uptime', fill: '#9b59b6' }
                    ].map((entry, index) => (
                      <Cell key={`cell-${index}`} fill={entry.fill} />
                    ))}
                  </Bar>
                </BarChart>
              </ResponsiveContainer>
            </div>

            {/* Hashrate Comparison */}
            <div style={graphStyles.chartCard}>
              <h4 style={graphStyles.chartTitle}>Hashrate Analysis</h4>
              <ResponsiveContainer width="100%" height={250}>
                <BarChart
                  data={[
                    { name: 'Reported', value: selectedMiner.hashrate || 0 },
                    { name: 'Effective', value: selectedMiner.performance?.effective_hashrate || 0 }
                  ]}
                >
                  <CartesianGrid strokeDasharray="3 3" stroke="#2a2a4a" />
                  <XAxis dataKey="name" stroke="#888" fontSize={12} />
                  <YAxis stroke="#888" fontSize={12} tickFormatter={(v) => formatHashrate(v)} />
                  <Tooltip 
                    contentStyle={{ backgroundColor: '#1a1a2e', border: '1px solid #2a2a4a', borderRadius: '8px' }}
                    formatter={(value: number) => [formatHashrate(value), 'Hashrate']}
                  />
                  <Bar dataKey="value" fill="#9b59b6" radius={[4, 4, 0, 0]} />
                </BarChart>
              </ResponsiveContainer>
            </div>
          </div>

          {/* Recent Shares Table */}
          <h3 style={{ color: '#00d4ff', marginTop: '30px' }}>📋 Recent Shares</h3>
          <div style={styles.tableContainer}>
            <table style={styles.table}>
              <thead>
                <tr>
                  <th style={styles.th}>Time</th>
                  <th style={styles.th}>Difficulty</th>
                  <th style={styles.th}>Status</th>
                  <th style={styles.th}>Nonce</th>
                </tr>
              </thead>
              <tbody>
                {(selectedMiner.recent_shares || []).map((share: any) => (
                  <tr key={share.id} style={styles.tr}>
                    <td style={styles.td}>{share.time_since}</td>
                    <td style={styles.td}>{share.difficulty?.toFixed(4)}</td>
                    <td style={styles.td}>
                      <span style={share.is_valid ? styles.activeBadge : styles.inactiveBadge}>
                        {share.is_valid ? '✓ Valid' : '✗ Invalid'}
                      </span>
                    </td>
                    <td style={styles.td}><code style={{ fontSize: '0.8rem' }}>{share.nonce?.substring(0, 16)}...</code></td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      ) : selectedUserMiners ? (
        // User Miners Summary View
        <div>
          <button 
            style={{ marginBottom: '20px', padding: '8px 16px', backgroundColor: '#2a2a4a', border: 'none', borderRadius: '6px', color: '#e0e0e0', cursor: 'pointer' }}
            onClick={() => setSelectedUserMiners(null)}
          >
            ← Back to Miners List
          </button>

          <div style={{ ...styles.algoCard, marginBottom: '20px' }}>
            <h3 style={{ color: '#00d4ff', marginTop: 0 }}>👤 {selectedUserMiners.username}'s Mining Overview</h3>
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(150px, 1fr))', gap: '15px', marginTop: '15px' }}>
              <div style={{ textAlign: 'center' }}>
                <div style={{ fontSize: '2rem', color: '#9b59b6' }}>{selectedUserMiners.total_miners}</div>
                <div style={{ color: '#888' }}>Total Miners</div>
              </div>
              <div style={{ textAlign: 'center' }}>
                <div style={{ fontSize: '2rem', color: '#4ade80' }}>{selectedUserMiners.active_miners}</div>
                <div style={{ color: '#888' }}>Active</div>
              </div>
              <div style={{ textAlign: 'center' }}>
                <div style={{ fontSize: '2rem', color: '#f87171' }}>{selectedUserMiners.inactive_miners}</div>
                <div style={{ color: '#888' }}>Offline</div>
              </div>
              <div style={{ textAlign: 'center' }}>
                <div style={{ fontSize: '2rem', color: '#00d4ff' }}>{formatHashrate(selectedUserMiners.total_hashrate)}</div>
                <div style={{ color: '#888' }}>Total Hashrate</div>
              </div>
              <div style={{ textAlign: 'center' }}>
                <div style={{ fontSize: '2rem', color: '#fbbf24' }}>{selectedUserMiners.total_shares_24h}</div>
                <div style={{ color: '#888' }}>Shares (24h)</div>
              </div>
            </div>
          </div>

          <h3 style={{ color: '#00d4ff' }}>⛏️ Miners</h3>
          <div style={{ display: 'flex', flexDirection: 'column', gap: '10px' }}>
            {(selectedUserMiners.miners || []).map((miner: any) => (
              <div 
                key={miner.id} 
                style={{ ...styles.algoCard, cursor: 'pointer', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}
                onClick={() => fetchMinerDetail(miner.id)}
              >
                <div>
                  <strong style={{ color: '#e0e0e0' }}>{miner.name}</strong>
                  <span style={{ marginLeft: '10px', ...miner.is_active ? styles.activeBadge : styles.inactiveBadge }}>
                    {miner.is_active ? 'Online' : 'Offline'}
                  </span>
                </div>
                <div style={{ display: 'flex', gap: '20px', color: '#888' }}>
                  <span>⚡ {formatHashrate(miner.hashrate)}</span>
                  <span>📊 {miner.shares_24h} shares</span>
                  <span>✓ {miner.valid_percent?.toFixed(1)}%</span>
                </div>
              </div>
            ))}
          </div>
        </div>
      ) : (
        // All Miners List View
        <>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '20px', flexWrap: 'wrap', gap: '15px' }}>
            <div>
              <h3 style={{ color: '#9b59b6', margin: 0 }}>⛏️ Miner Monitoring</h3>
              <p style={{ color: '#888', margin: '5px 0 0', fontSize: '0.9rem' }}>
                {minerTotal} miners • View detailed performance and troubleshooting info
              </p>
            </div>
            <div style={{ display: 'flex', gap: '10px', alignItems: 'center' }}>
              <label style={{ color: '#888', display: 'flex', alignItems: 'center', gap: '5px' }}>
                <input 
                  type="checkbox" 
                  checked={activeMinersOnly} 
                  onChange={e => { setActiveMinersOnly(e.target.checked); setMinerPage(1); }}
                />
                Active only
              </label>
              <button 
                style={{ padding: '8px 16px', backgroundColor: '#00d4ff', border: 'none', borderRadius: '6px', color: '#0a0a0f', fontWeight: 'bold', cursor: 'pointer' }}
                onClick={fetchAllMiners}
              >
                🔄 Refresh
              </button>
            </div>
          </div>

          <div style={styles.searchBar}>
            <input 
              style={styles.searchInput} 
              type="text" 
              placeholder="Search miners by name..." 
              value={minerSearch} 
              onChange={e => { setMinerSearch(e.target.value); setMinerPage(1); }} 
            />
          </div>

          {minersLoading ? (
            <div style={styles.loading}>Loading miners...</div>
          ) : allMiners.length === 0 ? (
            <div style={{ textAlign: 'center', padding: '40px', color: '#666' }}>
              <p style={{ fontSize: '1.2rem', margin: '0 0 10px' }}>No miners found</p>
              <p style={{ margin: 0 }}>No miners are currently registered in the pool.</p>
            </div>
          ) : (
            <>
              <div style={styles.tableContainer}>
                <table style={styles.table}>
                  <thead>
                    <tr>
                      <th style={styles.th}>Miner Name</th>
                      <th style={styles.th}>IP Address</th>
                      <th style={styles.th}>Hashrate</th>
                      <th style={styles.th}>Shares (24h)</th>
                      <th style={styles.th}>Valid %</th>
                      <th style={styles.th}>Status</th>
                      <th style={styles.th}>Last Seen</th>
                      <th style={styles.th}>Actions</th>
                    </tr>
                  </thead>
                  <tbody>
                    {allMiners.map((miner: any) => (
                      <tr key={miner.id} style={styles.tr}>
                        <td style={styles.td}><strong>{miner.name}</strong></td>
                        <td style={styles.td}>{miner.address || 'Unknown'}</td>
                        <td style={styles.td}>{formatHashrate(miner.hashrate)}</td>
                        <td style={styles.td}>{miner.shares_24h}</td>
                        <td style={styles.td}>
                          <span style={{ color: miner.valid_percent >= 95 ? '#4ade80' : miner.valid_percent >= 80 ? '#fbbf24' : '#f87171' }}>
                            {miner.valid_percent?.toFixed(1)}%
                          </span>
                        </td>
                        <td style={styles.td}>
                          <span style={miner.is_active ? styles.activeBadge : styles.inactiveBadge}>
                            {miner.is_active ? '🟢 Online' : '🔴 Offline'}
                          </span>
                        </td>
                        <td style={styles.td}>{new Date(miner.last_seen).toLocaleString()}</td>
                        <td style={styles.td}>
                          <button 
                            style={{ ...styles.actionBtn, backgroundColor: '#1a3a4a', borderRadius: '4px' }} 
                            onClick={() => fetchMinerDetail(miner.id)}
                            title="View Details"
                          >
                            🔍
                          </button>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>

              <div style={styles.pagination}>
                <button style={styles.pageBtn} disabled={minerPage <= 1} onClick={() => setMinerPage(p => p - 1)}>← Prev</button>
                <span style={styles.pageInfo}>Page {minerPage} of {Math.ceil(minerTotal / 20)} ({minerTotal} miners)</span>
                <button style={styles.pageBtn} disabled={minerPage >= Math.ceil(minerTotal / 20)} onClick={() => setMinerPage(p => p + 1)}>Next →</button>
              </div>
            </>
          )}
        </>
      )}
    </div>
  );
}

export default AdminMinersTab;
