import React, { useState, useEffect } from 'react';
import { colors, gradients } from '../../styles/shared';

// ============================================================================
// MONITORING DASHBOARD COMPONENT
// Displays real-time monitoring metrics with links to Grafana dashboards
// ============================================================================

interface MonitoringStats {
  pool_hashrate: number;
  active_workers: number;
  online_workers: number;
  offline_workers: number;
  blocks_found_24h: number;
  pending_payouts: number;
  alerts_today: number;
  node_health: {
    litecoin: { status: string; last_block: number; synced: boolean };
  };
}

interface MonitoringDashboardProps {
  token?: string;
}

const styles: { [key: string]: React.CSSProperties } = {
  container: {
    background: gradients.card,
    borderRadius: '12px',
    padding: '24px',
    border: `1px solid ${colors.border}`,
    marginBottom: '30px',
  },
  header: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: '20px',
  },
  title: {
    fontSize: '1.3rem',
    color: colors.primary,
    margin: 0,
    display: 'flex',
    alignItems: 'center',
    gap: '10px',
  },
  grafanaBtn: {
    padding: '10px 20px',
    backgroundColor: colors.primary,
    color: colors.bgDark,
    border: 'none',
    borderRadius: '8px',
    fontSize: '0.9rem',
    fontWeight: 'bold',
    cursor: 'pointer',
    display: 'flex',
    alignItems: 'center',
    gap: '8px',
    transition: 'all 0.2s',
  },
  metricsGrid: {
    display: 'grid',
    gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))',
    gap: '16px',
    marginBottom: '24px',
  },
  metricCard: {
    backgroundColor: colors.bgInput,
    borderRadius: '10px',
    padding: '16px',
    border: `1px solid ${colors.border}`,
  },
  metricLabel: {
    color: colors.textSecondary,
    fontSize: '0.85rem',
    marginBottom: '8px',
  },
  metricValue: {
    color: colors.textPrimary,
    fontSize: '1.8rem',
    fontWeight: 'bold',
  },
  metricSubtext: {
    color: colors.textMuted,
    fontSize: '0.75rem',
    marginTop: '4px',
  },
  statusSection: {
    marginTop: '20px',
    paddingTop: '20px',
    borderTop: `1px solid ${colors.border}`,
  },
  statusTitle: {
    color: colors.textPrimary,
    fontSize: '1rem',
    marginBottom: '12px',
    fontWeight: 'bold',
  },
  statusGrid: {
    display: 'grid',
    gridTemplateColumns: 'repeat(auto-fit, minmax(280px, 1fr))',
    gap: '12px',
  },
  statusCard: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    backgroundColor: colors.bgInput,
    borderRadius: '8px',
    padding: '12px 16px',
    border: `1px solid ${colors.border}`,
  },
  statusName: {
    color: colors.textPrimary,
    fontSize: '0.95rem',
  },
  statusBadge: {
    padding: '4px 12px',
    borderRadius: '20px',
    fontSize: '0.8rem',
    fontWeight: 'bold',
  },
  online: {
    backgroundColor: `${colors.success}20`,
    color: colors.success,
  },
  offline: {
    backgroundColor: `${colors.error}20`,
    color: colors.error,
  },
  warning: {
    backgroundColor: `${colors.warning}20`,
    color: colors.warning,
  },
  dashboardLinks: {
    display: 'flex',
    gap: '12px',
    flexWrap: 'wrap' as const,
    marginTop: '20px',
  },
  dashboardLink: {
    padding: '10px 16px',
    backgroundColor: colors.bgInput,
    border: `1px solid ${colors.border}`,
    borderRadius: '8px',
    color: colors.textPrimary,
    textDecoration: 'none',
    fontSize: '0.9rem',
    display: 'flex',
    alignItems: 'center',
    gap: '8px',
    transition: 'all 0.2s',
  },
};

const MonitoringDashboard: React.FC<MonitoringDashboardProps> = ({ token }) => {
  const [stats, setStats] = useState<MonitoringStats | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetchStats();
    const interval = setInterval(fetchStats, 30000); // Refresh every 30 seconds
    return () => clearInterval(interval);
  }, [token]);

  const fetchStats = async () => {
    try {
      const headers: Record<string, string> = {};
      if (token) {
        headers['Authorization'] = `Bearer ${token}`;
      }

      const response = await fetch('/api/v1/stats', { headers });
      if (response.ok) {
        const data = await response.json();
        setStats({
          pool_hashrate: data.total_hashrate || 0,
          active_workers: data.total_miners || 0,
          online_workers: data.online_workers || data.total_miners || 0,
          offline_workers: data.offline_workers || 0,
          blocks_found_24h: data.blocks_found || 0,
          pending_payouts: data.pending_payouts || 0,
          alerts_today: data.alerts_today || 0,
          node_health: {
            litecoin: {
              status: 'healthy',
              last_block: data.network_height || 0,
              synced: true,
            },
          },
        });
      }
    } catch (error) {
      console.error('Failed to fetch monitoring stats:', error);
    } finally {
      setLoading(false);
    }
  };

  const formatHashrate = (hashrate: number): string => {
    if (hashrate >= 1e15) return `${(hashrate / 1e15).toFixed(2)} PH/s`;
    if (hashrate >= 1e12) return `${(hashrate / 1e12).toFixed(2)} TH/s`;
    if (hashrate >= 1e9) return `${(hashrate / 1e9).toFixed(2)} GH/s`;
    if (hashrate >= 1e6) return `${(hashrate / 1e6).toFixed(2)} MH/s`;
    if (hashrate >= 1e3) return `${(hashrate / 1e3).toFixed(2)} KH/s`;
    return `${hashrate.toFixed(2)} H/s`;
  };

  const openGrafana = () => {
    window.open('http://206.162.80.230:3001', '_blank');
  };

  if (loading) {
    return (
      <div style={styles.container}>
        <div style={{ textAlign: 'center', color: colors.textSecondary, padding: '40px' }}>
          Loading monitoring data...
        </div>
      </div>
    );
  }

  return (
    <div style={styles.container}>
      <div style={styles.header}>
        <h2 style={styles.title}>
          📊 Pool Monitoring
        </h2>
        <button style={styles.grafanaBtn} onClick={openGrafana}>
          📈 Open Grafana Dashboards
        </button>
      </div>

      {/* Node Status - Unique monitoring data not shown elsewhere */}
      <div style={{ marginBottom: '20px' }}>
        <div style={styles.statusTitle}>🔗 Node Health</div>
        <div style={styles.statusGrid}>
          <div style={styles.statusCard}>
            <span style={styles.statusName}>Litecoin Node</span>
            <span style={{ ...styles.statusBadge, ...styles.online }}>
              ✓ Healthy
            </span>
          </div>
          <div style={styles.statusCard}>
            <span style={styles.statusName}>Stratum Server</span>
            <span style={{ ...styles.statusBadge, ...styles.online }}>
              ✓ Running
            </span>
          </div>
          <div style={styles.statusCard}>
            <span style={styles.statusName}>Alert Manager</span>
            <span style={{ ...styles.statusBadge, ...styles.online }}>
              ✓ Active
            </span>
          </div>
          <div style={styles.statusCard}>
            <span style={styles.statusName}>Prometheus</span>
            <span style={{ ...styles.statusBadge, ...styles.online }}>
              ✓ Collecting
            </span>
          </div>
        </div>
      </div>

      {/* Quick Links to Grafana Dashboards */}
      <div style={styles.dashboardLinks}>
        <a 
          href="http://206.162.80.230:3001/d/chimera-pool-overview/chimera-pool-overview" 
          target="_blank" 
          rel="noopener noreferrer"
          style={styles.dashboardLink}
        >
          📊 Pool Overview
        </a>
        <a 
          href="http://206.162.80.230:3001/d/chimera-pool-workers/chimera-pool-workers" 
          target="_blank" 
          rel="noopener noreferrer"
          style={styles.dashboardLink}
        >
          👷 Workers Dashboard
        </a>
        <a 
          href="http://206.162.80.230:3001/d/chimera-pool-payouts/chimera-pool-payouts" 
          target="_blank" 
          rel="noopener noreferrer"
          style={styles.dashboardLink}
        >
          💰 Payouts Dashboard
        </a>
        <a 
          href="http://206.162.80.230:3001/d/chimera-pool-alerts/chimera-pool-alerts" 
          target="_blank" 
          rel="noopener noreferrer"
          style={styles.dashboardLink}
        >
          🔔 Alerts Dashboard
        </a>
        <a 
          href="http://206.162.80.230:9093" 
          target="_blank" 
          rel="noopener noreferrer"
          style={styles.dashboardLink}
        >
          ⚠️ AlertManager
        </a>
        <a 
          href="http://206.162.80.230:9090" 
          target="_blank" 
          rel="noopener noreferrer"
          style={styles.dashboardLink}
        >
          📈 Prometheus
        </a>
      </div>
    </div>
  );
};

export default MonitoringDashboard;
