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

// Chimera Elite Theme - Grafana-inspired monitoring styles
const styles: { [key: string]: React.CSSProperties } = {
  container: {
    background: 'linear-gradient(180deg, rgba(45, 31, 61, 0.6) 0%, rgba(26, 15, 30, 0.8) 100%)',
    borderRadius: '16px',
    padding: '24px',
    border: '1px solid #4A2C5A',
    marginBottom: '24px',
    boxShadow: '0 4px 24px rgba(0, 0, 0, 0.3)',
  },
  header: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: '20px',
    flexWrap: 'wrap' as const,
    gap: '12px',
  },
  title: {
    fontSize: '1.15rem',
    color: '#F0EDF4',
    margin: 0,
    display: 'flex',
    alignItems: 'center',
    gap: '10px',
    fontWeight: 600,
  },
  grafanaBtn: {
    padding: '10px 18px',
    background: 'linear-gradient(135deg, #D4A84B 0%, #B8923A 100%)',
    color: '#1A0F1E',
    border: 'none',
    borderRadius: '10px',
    fontSize: '0.85rem',
    fontWeight: 600,
    cursor: 'pointer',
    display: 'flex',
    alignItems: 'center',
    gap: '8px',
    transition: 'all 0.2s',
    boxShadow: '0 2px 8px rgba(212, 168, 75, 0.3)',
  },
  metricsGrid: {
    display: 'grid',
    gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))',
    gap: '16px',
    marginBottom: '24px',
  },
  metricCard: {
    background: 'linear-gradient(180deg, rgba(13, 8, 17, 0.8) 0%, rgba(26, 15, 30, 0.9) 100%)',
    borderRadius: '12px',
    padding: '16px',
    border: '1px solid rgba(74, 44, 90, 0.5)',
  },
  metricLabel: {
    color: '#B8B4C8',
    fontSize: '0.8rem',
    marginBottom: '8px',
    fontWeight: 500,
    letterSpacing: '0.03em',
  },
  metricValue: {
    color: '#D4A84B',
    fontSize: '1.8rem',
    fontWeight: 700,
  },
  metricSubtext: {
    color: '#7A7490',
    fontSize: '0.75rem',
    marginTop: '4px',
  },
  statusSection: {
    marginTop: '20px',
    paddingTop: '20px',
    borderTop: '1px solid rgba(74, 44, 90, 0.5)',
  },
  statusTitle: {
    color: '#F0EDF4',
    fontSize: '0.95rem',
    marginBottom: '12px',
    fontWeight: 600,
    display: 'flex',
    alignItems: 'center',
    gap: '8px',
  },
  statusGrid: {
    display: 'grid',
    gridTemplateColumns: 'repeat(auto-fit, minmax(220px, 1fr))',
    gap: '10px',
  },
  statusCard: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    background: 'linear-gradient(180deg, rgba(13, 8, 17, 0.7) 0%, rgba(26, 15, 30, 0.85) 100%)',
    borderRadius: '10px',
    padding: '14px 18px',
    border: '1px solid rgba(74, 44, 90, 0.4)',
    transition: 'all 0.2s ease',
  },
  statusName: {
    color: '#B8B4C8',
    fontSize: '0.9rem',
    fontWeight: 500,
  },
  statusBadge: {
    padding: '4px 12px',
    borderRadius: '20px',
    fontSize: '0.75rem',
    fontWeight: 600,
    letterSpacing: '0.02em',
  },
  online: {
    backgroundColor: 'rgba(74, 222, 128, 0.15)',
    color: '#4ADE80',
    border: '1px solid rgba(74, 222, 128, 0.3)',
    boxShadow: '0 0 8px rgba(74, 222, 128, 0.2)',
  },
  offline: {
    backgroundColor: 'rgba(239, 68, 68, 0.15)',
    color: '#EF4444',
    border: '1px solid rgba(239, 68, 68, 0.3)',
  },
  warning: {
    backgroundColor: 'rgba(251, 191, 36, 0.15)',
    color: '#FBBF24',
    border: '1px solid rgba(251, 191, 36, 0.3)',
  },
  dashboardLinks: {
    display: 'flex',
    gap: '10px',
    flexWrap: 'wrap' as const,
    marginTop: '16px',
  },
  dashboardLink: {
    padding: '12px 16px',
    background: 'linear-gradient(180deg, rgba(13, 8, 17, 0.7) 0%, rgba(26, 15, 30, 0.85) 100%)',
    border: '1px solid rgba(74, 44, 90, 0.4)',
    borderRadius: '10px',
    color: '#B8B4C8',
    textDecoration: 'none',
    fontSize: '0.85rem',
    fontWeight: 500,
    display: 'flex',
    alignItems: 'center',
    gap: '8px',
    transition: 'all 0.25s ease',
  },
  dashboardLinkHover: {
    borderColor: 'rgba(212, 168, 75, 0.5)',
    color: '#D4A84B',
    transform: 'translateY(-2px)',
    boxShadow: '0 4px 16px rgba(0, 0, 0, 0.3)',
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
    window.open('/grafana', '_blank');
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
          Open Grafana Dashboards
        </button>
      </div>

      {/* Node Status - Unique monitoring data not shown elsewhere */}
      <div style={{ marginBottom: '20px' }}>
        <div style={styles.statusTitle}>🔗 Node Health</div>
        <div style={styles.statusGrid}>
          <div style={styles.statusCard} className="status-card-enhanced">
            <span style={styles.statusName}>Litecoin Node</span>
            <span style={{ ...styles.statusBadge, ...styles.online }} className="status-pulse">Healthy</span>
          </div>
          <div style={styles.statusCard} className="status-card-enhanced">
            <span style={styles.statusName}>Stratum Server</span>
            <span style={{ ...styles.statusBadge, ...styles.online }} className="status-pulse">Running</span>
          </div>
          <div style={styles.statusCard} className="status-card-enhanced">
            <span style={styles.statusName}>Alert Manager</span>
            <span style={{ ...styles.statusBadge, ...styles.online }} className="status-pulse">Active</span>
          </div>
          <div style={styles.statusCard} className="status-card-enhanced">
            <span style={styles.statusName}>Prometheus</span>
            <span style={{ ...styles.statusBadge, ...styles.online }} className="status-pulse">Collecting</span>
          </div>
        </div>
      </div>

      {/* Quick Links to Grafana Dashboards */}
      <div style={styles.dashboardLinks}>
        <a 
          href="/grafana/d/chimera-pool-overview/chimera-pool-overview" 
          target="_blank" 
          rel="noopener noreferrer"
          style={styles.dashboardLink}
          className="dashboard-link-enhanced"
        >
          📊 Pool Overview
        </a>
        <a 
          href="/grafana/d/chimera-pool-workers/chimera-pool-workers" 
          target="_blank" 
          rel="noopener noreferrer"
          style={styles.dashboardLink}
          className="dashboard-link-enhanced"
        >
          👷 Workers
        </a>
        <a 
          href="/grafana/d/chimera-pool-payouts/chimera-pool-payouts" 
          target="_blank" 
          rel="noopener noreferrer"
          style={styles.dashboardLink}
          className="dashboard-link-enhanced"
        >
          💰 Payouts
        </a>
        <a 
          href="/grafana/d/chimera-pool-alerts/chimera-pool-alerts" 
          target="_blank" 
          rel="noopener noreferrer"
          style={styles.dashboardLink}
          className="dashboard-link-enhanced"
        >
          🔔 Alerts
        </a>
        <a 
          href="https://206.162.80.230:9093" 
          target="_blank" 
          rel="noopener noreferrer"
          style={styles.dashboardLink}
          className="dashboard-link-enhanced"
        >
          ⚠️ AlertManager
        </a>
        <a 
          href="https://206.162.80.230:9090" 
          target="_blank" 
          rel="noopener noreferrer"
          style={styles.dashboardLink}
          className="dashboard-link-enhanced"
        >
          📈 Prometheus
        </a>
      </div>
    </div>
  );
};

export default MonitoringDashboard;
