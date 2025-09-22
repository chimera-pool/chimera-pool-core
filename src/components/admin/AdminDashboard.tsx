import React, { useState, useEffect } from 'react';
import './AdminDashboard.css';

interface SystemHealth {
  status: string;
  services: Record<string, string>;
  version: string;
  timestamp: string;
}

interface PoolStats {
  activeMiners: number;
  totalHashrate: number;
  blocksFound: number;
  totalShares: number;
}

export const AdminDashboard: React.FC = () => {
  const [systemHealth, setSystemHealth] = useState<SystemHealth | null>(null);
  const [poolStats, setPoolStats] = useState<PoolStats | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetchSystemHealth();
    fetchPoolStats();
    
    // Refresh data every 30 seconds
    const interval = setInterval(() => {
      fetchSystemHealth();
      fetchPoolStats();
    }, 30000);

    return () => clearInterval(interval);
  }, []);

  const fetchSystemHealth = async () => {
    try {
      const response = await fetch('/api/admin/system/health');
      if (response.ok) {
        const data = await response.json();
        setSystemHealth(data);
      }
    } catch (error) {
      console.error('Failed to fetch system health:', error);
    }
  };

  const fetchPoolStats = async () => {
    try {
      const response = await fetch('/api/admin/pool/stats');
      if (response.ok) {
        const data = await response.json();
        setPoolStats(data);
      }
    } catch (error) {
      console.error('Failed to fetch pool stats:', error);
    } finally {
      setLoading(false);
    }
  };

  if (loading) {
    return (
      <div className="admin-dashboard">
        <div className="cyber-loading">
          <div className="cyber-spinner"></div>
          <span>LOADING_ADMIN_INTERFACE...</span>
        </div>
      </div>
    );
  }

  return (
    <div className="admin-dashboard">
      <div className="cyber-header">
        <h1 className="cyber-title">
          <span className="cyber-bracket">[</span>
          ADMIN_CONTROL_PANEL
          <span className="cyber-bracket">]</span>
        </h1>
        <div className="cyber-timestamp">
          {new Date().toISOString()}
        </div>
      </div>

      <div className="admin-grid">
        {/* System Health Panel */}
        <div className="cyber-panel">
          <div className="cyber-panel-header">
            <h3 className="cyber-panel-title">
              <span className="cyber-icon">🔧</span>
              SYSTEM_HEALTH
            </h3>
          </div>
          <div className="cyber-panel-content">
            {systemHealth ? (
              <>
                <div className={`health-status ${systemHealth.status}`}>
                  STATUS: {systemHealth.status.toUpperCase()}
                </div>
                <div className="services-grid">
                  {Object.entries(systemHealth.services).map(([service, status]) => (
                    <div key={service} className="service-item">
                      <span className="service-name">{service}</span>
                      <span className={`service-status ${status}`}>
                        {status.toUpperCase()}
                      </span>
                    </div>
                  ))}
                </div>
                <div className="system-info">
                  <div>Version: {systemHealth.version}</div>
                  <div>Last Check: {new Date(systemHealth.timestamp).toLocaleTimeString()}</div>
                </div>
              </>
            ) : (
              <div className="error-message">SYSTEM_HEALTH_UNAVAILABLE</div>
            )}
          </div>
        </div>

        {/* Pool Statistics Panel */}
        <div className="cyber-panel">
          <div className="cyber-panel-header">
            <h3 className="cyber-panel-title">
              <span className="cyber-icon">⛏️</span>
              POOL_STATISTICS
            </h3>
          </div>
          <div className="cyber-panel-content">
            {poolStats ? (
              <div className="stats-grid">
                <div className="stat-item">
                  <div className="stat-label">ACTIVE_MINERS</div>
                  <div className="stat-value">{poolStats.activeMiners.toLocaleString()}</div>
                </div>
                <div className="stat-item">
                  <div className="stat-label">TOTAL_HASHRATE</div>
                  <div className="stat-value">{(poolStats.totalHashrate / 1000000).toFixed(2)} MH/s</div>
                </div>
                <div className="stat-item">
                  <div className="stat-label">BLOCKS_FOUND</div>
                  <div className="stat-value">{poolStats.blocksFound.toLocaleString()}</div>
                </div>
                <div className="stat-item">
                  <div className="stat-label">TOTAL_SHARES</div>
                  <div className="stat-value">{poolStats.totalShares.toLocaleString()}</div>
                </div>
              </div>
            ) : (
              <div className="error-message">POOL_STATS_UNAVAILABLE</div>
            )}
          </div>
        </div>

        {/* Algorithm Management Panel */}
        <div className="cyber-panel">
          <div className="cyber-panel-header">
            <h3 className="cyber-panel-title">
              <span className="cyber-icon">🔄</span>
              ALGORITHM_MANAGEMENT
            </h3>
          </div>
          <div className="cyber-panel-content">
            <div className="algorithm-controls">
              <button className="cyber-button primary">
                VIEW_ACTIVE_ALGORITHM
              </button>
              <button className="cyber-button secondary">
                STAGE_NEW_ALGORITHM
              </button>
              <button className="cyber-button warning">
                DEPLOY_STAGED_ALGORITHM
              </button>
            </div>
          </div>
        </div>

        {/* User Management Panel */}
        <div className="cyber-panel">
          <div className="cyber-panel-header">
            <h3 className="cyber-panel-title">
              <span className="cyber-icon">👥</span>
              USER_MANAGEMENT
            </h3>
          </div>
          <div className="cyber-panel-content">
            <div className="user-controls">
              <button className="cyber-button primary">
                VIEW_ALL_USERS
              </button>
              <button className="cyber-button secondary">
                MANAGE_PERMISSIONS
              </button>
              <button className="cyber-button warning">
                SECURITY_AUDIT
              </button>
            </div>
          </div>
        </div>

        {/* Monitoring Panel */}
        <div className="cyber-panel">
          <div className="cyber-panel-header">
            <h3 className="cyber-panel-title">
              <span className="cyber-icon">📊</span>
              MONITORING
            </h3>
          </div>
          <div className="cyber-panel-content">
            <div className="monitoring-controls">
              <button className="cyber-button primary">
                VIEW_METRICS
              </button>
              <button className="cyber-button secondary">
                CONFIGURE_ALERTS
              </button>
              <button className="cyber-button info">
                EXPORT_LOGS
              </button>
            </div>
          </div>
        </div>

        {/* Backup & Recovery Panel */}
        <div className="cyber-panel">
          <div className="cyber-panel-header">
            <h3 className="cyber-panel-title">
              <span className="cyber-icon">💾</span>
              BACKUP_RECOVERY
            </h3>
          </div>
          <div className="cyber-panel-content">
            <div className="backup-controls">
              <button className="cyber-button primary">
                CREATE_BACKUP
              </button>
              <button className="cyber-button secondary">
                RESTORE_BACKUP
              </button>
              <button className="cyber-button info">
                BACKUP_STATUS
              </button>
            </div>
          </div>
        </div>
      </div>

      {/* Cyber background effects */}
      <div className="cyber-background-effects">
        <div className="cyber-grid-overlay"></div>
        <div className="cyber-scan-lines"></div>
      </div>
    </div>
  );
};

export default AdminDashboard;