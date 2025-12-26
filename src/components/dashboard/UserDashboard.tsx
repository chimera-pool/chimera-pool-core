import React, { useState, useEffect } from 'react';
import { colors, gradients } from '../../styles/shared';
import { formatHashrate } from '../../utils/formatters';
import { useUserDashboard } from '../../services/realtime/useRealTimeData';
import { PayoutSettings } from './PayoutSettings';
import NotificationSettings from './NotificationSettings';
import MonitoringDashboard from './MonitoringDashboard';

// ============================================================================
// USER DASHBOARD COMPONENT
// Displays user's mining statistics, miners, and payouts
// ============================================================================

export interface Miner {
  id: number;
  name: string;
  hashrate: number;
  is_active: boolean;
  last_seen: string;
  shares_submitted: number;
  valid_shares: number;
  invalid_shares: number;
}

export interface UserStats {
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

export interface UserDashboardProps {
  token: string;
}

const styles: { [key: string]: React.CSSProperties } = {
  section: {
    background: gradients.card,
    borderRadius: '12px',
    padding: '24px',
    border: `1px solid ${colors.border}`,
    marginBottom: '30px',
  },
  sectionTitle: {
    fontSize: '1.3rem',
    color: colors.primary,
    margin: '0 0 16px',
  },
  loading: {
    textAlign: 'center',
    padding: '40px',
    color: colors.primary,
  },
  statsRow: {
    display: 'grid',
    gridTemplateColumns: 'repeat(auto-fit, minmax(150px, 1fr))',
    gap: '15px',
    marginBottom: '25px',
  },
  statBox: {
    backgroundColor: colors.bgInput,
    padding: '20px',
    borderRadius: '8px',
    textAlign: 'center',
    border: `1px solid ${colors.border}`,
  },
  statLabel: {
    display: 'block',
    color: colors.textSecondary,
    fontSize: '0.85rem',
    marginBottom: '8px',
    textTransform: 'uppercase' as const,
  },
  statValue: {
    display: 'block',
    color: colors.primary,
    fontSize: '1.4rem',
    fontWeight: 'bold',
  },
  subTitle: {
    color: colors.primary,
    fontSize: '1.1rem',
    marginTop: '25px',
    marginBottom: '15px',
  },
  emptyState: {
    backgroundColor: colors.bgInput,
    padding: '30px',
    borderRadius: '8px',
    textAlign: 'center',
    color: colors.textSecondary,
  },
  tableWrapper: {
    overflowX: 'auto' as const,
    backgroundColor: colors.bgInput,
    borderRadius: '8px',
  },
  table: {
    width: '100%',
    borderCollapse: 'collapse' as const,
  },
  th: {
    padding: '12px 15px',
    textAlign: 'left' as const,
    borderBottom: `2px solid ${colors.border}`,
    color: colors.primary,
    fontSize: '0.8rem',
    textTransform: 'uppercase' as const,
    whiteSpace: 'nowrap' as const,
  },
  tr: {
    borderBottom: `1px solid ${colors.bgCard}`,
  },
  td: {
    padding: '12px 15px',
    color: colors.textPrimary,
    fontSize: '0.9rem',
  },
  online: {
    color: colors.success,
  },
  offline: {
    color: colors.textSecondary,
  },
};

export function UserDashboard({ token }: UserDashboardProps) {
  // Unified real-time data from context
  const dashboardData = useUserDashboard();
  
  const [stats, setStats] = useState<UserStats | null>(null);
  const [loading, setLoading] = useState(true);

  // Set auth token in context on mount
  useEffect(() => {
    dashboardData.setAuthToken(token);
  }, [token, dashboardData.setAuthToken]);

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

  const getSuccessRateColor = (rate: number): string => {
    if (rate >= 95) return colors.success;
    if (rate >= 80) return colors.warning;
    return colors.error;
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
    <section style={styles.section}>
      <h2 style={styles.sectionTitle}>📈 Your Mining Dashboard</h2>
      
      {/* Summary Stats */}
      <div style={styles.statsRow}>
        <div style={styles.statBox}>
          <span style={styles.statLabel}>Total Hashrate</span>
          <span style={styles.statValue}>{formatHashrate(stats.total_hashrate)}</span>
        </div>
        <div style={styles.statBox}>
          <span style={styles.statLabel}>Active Miners</span>
          <span style={styles.statValue}>
            {stats.miners.filter(m => m.is_active).length} / {stats.miners.length}
          </span>
        </div>
        <div style={styles.statBox}>
          <span style={styles.statLabel}>Total Shares</span>
          <span style={styles.statValue}>{stats.total_shares.toLocaleString()}</span>
        </div>
        <div style={styles.statBox}>
          <span style={styles.statLabel}>Success Rate</span>
          <span style={{ ...styles.statValue, color: getSuccessRateColor(stats.success_rate) }}>
            {stats.success_rate.toFixed(2)}%
          </span>
        </div>
      </div>

      {/* Miners Table */}
      <h3 style={styles.subTitle}>⛏️ Your Miners ({stats.miners.length})</h3>
      {stats.miners.length === 0 ? (
        <div style={styles.emptyState}>
          <p>No miners connected yet. Connect your miner using the stratum URL below!</p>
        </div>
      ) : (
        <div style={styles.tableWrapper}>
          <table style={styles.table}>
            <thead>
              <tr>
                <th style={styles.th}>Miner Name</th>
                <th style={styles.th}>Status</th>
                <th style={styles.th}>Hashrate</th>
                <th style={styles.th}>Valid Shares</th>
                <th style={styles.th}>Invalid Shares</th>
                <th style={styles.th}>Success Rate</th>
                <th style={styles.th}>Last Seen</th>
              </tr>
            </thead>
            <tbody>
              {stats.miners.map(miner => {
                const minerSuccessRate = miner.shares_submitted > 0 
                  ? ((miner.valid_shares || 0) / miner.shares_submitted) * 100 
                  : 0;
                return (
                  <tr key={miner.id} style={styles.tr}>
                    <td style={styles.td}>{miner.name}</td>
                    <td style={styles.td}>
                      <span style={miner.is_active ? styles.online : styles.offline}>
                        {miner.is_active ? '🟢 Online' : '🔴 Offline'}
                      </span>
                    </td>
                    <td style={styles.td}>{formatHashrate(miner.hashrate)}</td>
                    <td style={styles.td}>{(miner.valid_shares || 0).toLocaleString()}</td>
                    <td style={styles.td}>{(miner.invalid_shares || 0).toLocaleString()}</td>
                    <td style={styles.td}>
                      <span style={{ color: getSuccessRateColor(minerSuccessRate) }}>
                        {minerSuccessRate.toFixed(2)}%
                      </span>
                    </td>
                    <td style={styles.td}>{new Date(miner.last_seen).toLocaleString()}</td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}

      {/* Monitoring Dashboard */}
      <MonitoringDashboard token={token} />

      {/* Payout Settings */}
      <PayoutSettings token={token} />

      {/* Notification Settings */}
      <NotificationSettings token={token} />

      {/* Recent Payouts */}
      {stats.recent_payouts.length > 0 && (
        <>
          <h3 style={styles.subTitle}>💰 Recent Payouts</h3>
          <div style={styles.tableWrapper}>
            <table style={styles.table}>
              <thead>
                <tr>
                  <th style={styles.th}>Amount</th>
                  <th style={styles.th}>Status</th>
                  <th style={styles.th}>TX Hash</th>
                  <th style={styles.th}>Date</th>
                </tr>
              </thead>
              <tbody>
                {stats.recent_payouts.slice(0, 5).map((payout: any, idx: number) => (
                  <tr key={idx} style={styles.tr}>
                    <td style={styles.td}>{payout.amount} BDAG</td>
                    <td style={styles.td}>
                      <span style={payout.status === 'completed' ? styles.online : styles.offline}>
                        {payout.status}
                      </span>
                    </td>
                    <td style={styles.td}>
                      {payout.tx_hash ? `${payout.tx_hash.slice(0, 10)}...` : '-'}
                    </td>
                    <td style={styles.td}>{new Date(payout.created_at).toLocaleString()}</td>
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

export default UserDashboard;
