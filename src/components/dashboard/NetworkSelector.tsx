import React, { useState, useEffect } from 'react';

// ============================================================================
// NETWORK SELECTOR COMPONENT
// Multi-coin network selector with stats for universal mining platform
// ============================================================================

interface NetworkConfig {
  id: string;
  name: string;
  symbol: string;
  display_name: string;
  is_active: boolean;
  is_default: boolean;
  algorithm: string;
  stratum_port: number;
  explorer_url: string;
  logo_url?: string;
  min_payout_threshold: number;
  pool_fee_percent: number;
}

interface NetworkPoolStats {
  network_id: string;
  network_name: string;
  network_symbol: string;
  display_name: string;
  is_active: boolean;
  total_hashrate: number;
  active_miners: number;
  active_workers: number;
  blocks_found: number;
  network_difficulty: number;
  rpc_connected: boolean;
}

interface UserNetworkStats {
  network_id: string;
  network_name: string;
  network_symbol: string;
  display_name: string;
  is_active: boolean;
  total_hashrate: number;
  total_shares: number;
  valid_shares: number;
  blocks_found: number;
  total_earned: number;
  pending_balance: number;
  active_workers: number;
  last_active_at?: string;
  first_connected_at?: string;
}

interface AggregatedStats {
  total_networks_mined: number;
  active_networks: number;
  combined_hashrate: number;
  total_shares_all: number;
  total_blocks_all: number;
  total_earned_all: number;
  total_pending_all: number;
  total_workers_all: number;
}

interface NetworkSelectorProps {
  token?: string;
  selectedNetwork?: string;
  onNetworkChange?: (networkId: string) => void;
  showUserStats?: boolean;
  compact?: boolean;
}

const API_URL = process.env.REACT_APP_API_URL || '';

const NetworkSelector: React.FC<NetworkSelectorProps> = ({
  token,
  selectedNetwork,
  onNetworkChange,
  showUserStats = true,
  compact = false,
}) => {
  const [networks, setNetworks] = useState<NetworkConfig[]>([]);
  const [poolStats, setPoolStats] = useState<NetworkPoolStats[]>([]);
  const [userStats, setUserStats] = useState<UserNetworkStats[]>([]);
  const [aggregated, setAggregated] = useState<AggregatedStats | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [activeNetwork, setActiveNetwork] = useState<string>(selectedNetwork || '');

  useEffect(() => {
    fetchNetworks();
    fetchPoolStats();
    if (token && showUserStats) {
      fetchUserStats();
      fetchAggregatedStats();
    }
  }, [token, showUserStats]);

  const fetchNetworks = async () => {
    try {
      const res = await fetch(`${API_URL}/api/v1/networks`);
      const data = await res.json();
      if (data.success) {
        setNetworks(data.networks || []);
        if (!activeNetwork && data.networks?.length > 0) {
          const defaultNet = data.networks.find((n: NetworkConfig) => n.is_default) || data.networks[0];
          setActiveNetwork(defaultNet.id);
        }
      }
    } catch (err) {
      setError('Failed to load networks');
    }
  };

  const fetchPoolStats = async () => {
    try {
      const res = await fetch(`${API_URL}/api/v1/networks/stats`);
      const data = await res.json();
      if (data.success) {
        setPoolStats(data.stats || []);
      }
    } catch (err) {
      console.error('Failed to load pool stats:', err);
    } finally {
      setLoading(false);
    }
  };

  const fetchUserStats = async () => {
    if (!token) return;
    try {
      const res = await fetch(`${API_URL}/api/v1/user/networks/stats`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      const data = await res.json();
      if (data.success) {
        setUserStats(data.stats || []);
      }
    } catch (err) {
      console.error('Failed to load user network stats:', err);
    }
  };

  const fetchAggregatedStats = async () => {
    if (!token) return;
    try {
      const res = await fetch(`${API_URL}/api/v1/user/networks/aggregated`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      const data = await res.json();
      if (data.success) {
        setAggregated(data);
      }
    } catch (err) {
      console.error('Failed to load aggregated stats:', err);
    }
  };

  const handleNetworkSelect = (networkId: string) => {
    setActiveNetwork(networkId);
    onNetworkChange?.(networkId);
  };

  const formatHashrate = (hashrate: number): string => {
    if (hashrate >= 1e15) return `${(hashrate / 1e15).toFixed(2)} PH/s`;
    if (hashrate >= 1e12) return `${(hashrate / 1e12).toFixed(2)} TH/s`;
    if (hashrate >= 1e9) return `${(hashrate / 1e9).toFixed(2)} GH/s`;
    if (hashrate >= 1e6) return `${(hashrate / 1e6).toFixed(2)} MH/s`;
    if (hashrate >= 1e3) return `${(hashrate / 1e3).toFixed(2)} KH/s`;
    return `${hashrate.toFixed(2)} H/s`;
  };

  const getNetworkIcon = (symbol: string): string => {
    const icons: { [key: string]: string } = {
      BTC: '₿',
      LTC: 'Ł',
      BDAG: '◈',
      ETH: 'Ξ',
      XMR: 'ɱ',
      DASH: 'Đ',
      ZEC: 'ⓩ',
    };
    return icons[symbol] || '⬡';
  };

  const getPoolStatsForNetwork = (networkId: string): NetworkPoolStats | undefined => {
    return poolStats.find(s => s.network_id === networkId);
  };

  const getUserStatsForNetwork = (networkId: string): UserNetworkStats | undefined => {
    return userStats.find(s => s.network_id === networkId);
  };

  if (loading) {
    return (
      <div style={styles.loadingContainer} data-testid="network-selector-loading">
        <span style={styles.loadingSpinner}>⟳</span> Loading networks...
      </div>
    );
  }

  if (error) {
    return (
      <div style={styles.errorContainer} data-testid="network-selector-error">
        ⚠️ {error}
      </div>
    );
  }

  if (compact) {
    return (
      <div style={styles.compactContainer} data-testid="network-selector-compact">
        <select
          style={styles.compactSelect}
          value={activeNetwork}
          onChange={(e) => handleNetworkSelect(e.target.value)}
          data-testid="network-selector-dropdown"
        >
          {networks.map((network) => (
            <option key={network.id} value={network.id}>
              {getNetworkIcon(network.symbol)} {network.display_name} ({network.symbol})
            </option>
          ))}
        </select>
      </div>
    );
  }

  return (
    <div style={styles.container} data-testid="network-selector">
      {/* Aggregated Stats Header */}
      {token && aggregated && (
        <div style={styles.aggregatedHeader} data-testid="aggregated-stats">
          <h3 style={styles.aggregatedTitle}>🌐 Your Multi-Network Mining</h3>
          <div style={styles.aggregatedGrid}>
            <div style={styles.aggregatedStat}>
              <span style={styles.aggregatedLabel}>Networks</span>
              <span style={styles.aggregatedValue}>{aggregated.active_networks}/{aggregated.total_networks_mined}</span>
            </div>
            <div style={styles.aggregatedStat}>
              <span style={styles.aggregatedLabel}>Combined Hashrate</span>
              <span style={styles.aggregatedValue}>{formatHashrate(aggregated.combined_hashrate)}</span>
            </div>
            <div style={styles.aggregatedStat}>
              <span style={styles.aggregatedLabel}>Total Shares</span>
              <span style={styles.aggregatedValue}>{aggregated.total_shares_all.toLocaleString()}</span>
            </div>
            <div style={styles.aggregatedStat}>
              <span style={styles.aggregatedLabel}>Blocks Found</span>
              <span style={styles.aggregatedValue}>{aggregated.total_blocks_all}</span>
            </div>
            <div style={styles.aggregatedStat}>
              <span style={styles.aggregatedLabel}>Total Workers</span>
              <span style={styles.aggregatedValue}>{aggregated.total_workers_all}</span>
            </div>
          </div>
        </div>
      )}

      {/* Network Cards */}
      <div style={styles.networkGrid} data-testid="network-grid">
        {networks.map((network) => {
          const poolStat = getPoolStatsForNetwork(network.id);
          const userStat = getUserStatsForNetwork(network.id);
          const isSelected = activeNetwork === network.id;

          return (
            <div
              key={network.id}
              style={{
                ...styles.networkCard,
                ...(isSelected ? styles.networkCardSelected : {}),
                ...(network.is_active ? {} : styles.networkCardInactive),
              }}
              onClick={() => handleNetworkSelect(network.id)}
              data-testid={`network-card-${network.name}`}
            >
              <div style={styles.networkHeader}>
                <span style={styles.networkIcon}>{getNetworkIcon(network.symbol)}</span>
                <div style={styles.networkInfo}>
                  <span style={styles.networkName}>{network.display_name}</span>
                  <span style={styles.networkSymbol}>{network.symbol}</span>
                </div>
                {network.is_active ? (
                  <span style={styles.activeIndicator} title="Active">●</span>
                ) : (
                  <span style={styles.inactiveIndicator} title="Inactive">○</span>
                )}
              </div>

              <div style={styles.networkDetails}>
                <span style={styles.algorithmBadge}>{network.algorithm}</span>
                <span style={styles.portInfo}>Port: {network.stratum_port}</span>
              </div>

              {/* Pool Stats */}
              {poolStat && (
                <div style={styles.statsSection}>
                  <div style={styles.statRow}>
                    <span style={styles.statLabel}>Pool Hashrate</span>
                    <span style={styles.statValue}>{formatHashrate(poolStat.total_hashrate)}</span>
                  </div>
                  <div style={styles.statRow}>
                    <span style={styles.statLabel}>Miners</span>
                    <span style={styles.statValue}>{poolStat.active_miners}</span>
                  </div>
                  <div style={styles.statRow}>
                    <span style={styles.statLabel}>Workers</span>
                    <span style={styles.statValue}>{poolStat.active_workers}</span>
                  </div>
                  <div style={styles.statRow}>
                    <span style={styles.statLabel}>Blocks</span>
                    <span style={styles.statValue}>{poolStat.blocks_found}</span>
                  </div>
                </div>
              )}

              {/* User Stats */}
              {token && userStat && userStat.total_shares > 0 && (
                <div style={styles.userStatsSection}>
                  <div style={styles.userStatsHeader}>Your Stats</div>
                  <div style={styles.statRow}>
                    <span style={styles.statLabel}>Hashrate</span>
                    <span style={styles.statValueGold}>{formatHashrate(userStat.total_hashrate)}</span>
                  </div>
                  <div style={styles.statRow}>
                    <span style={styles.statLabel}>Shares</span>
                    <span style={styles.statValueGold}>{userStat.valid_shares.toLocaleString()}</span>
                  </div>
                  <div style={styles.statRow}>
                    <span style={styles.statLabel}>Pending</span>
                    <span style={styles.statValueGold}>{userStat.pending_balance.toFixed(8)}</span>
                  </div>
                </div>
              )}

              {/* Fee Info */}
              <div style={styles.feeInfo}>
                Fee: {network.pool_fee_percent}% | Min: {network.min_payout_threshold}
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
};

const styles: { [key: string]: React.CSSProperties } = {
  container: {
    padding: '20px',
  },
  loadingContainer: {
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    padding: '40px',
    color: '#D4A84B',
    fontSize: '1.1rem',
  },
  loadingSpinner: {
    marginRight: '10px',
    animation: 'spin 1s linear infinite',
  },
  errorContainer: {
    padding: '20px',
    textAlign: 'center',
    color: '#ef4444',
    background: 'rgba(239, 68, 68, 0.1)',
    borderRadius: '12px',
    border: '1px solid rgba(239, 68, 68, 0.3)',
  },
  compactContainer: {
    display: 'inline-block',
  },
  compactSelect: {
    padding: '10px 16px',
    backgroundColor: 'rgba(13, 8, 17, 0.8)',
    border: '1px solid rgba(74, 44, 90, 0.5)',
    borderRadius: '8px',
    color: '#F0EDF4',
    fontSize: '0.95rem',
    cursor: 'pointer',
  },
  aggregatedHeader: {
    background: 'linear-gradient(180deg, rgba(212, 168, 75, 0.15) 0%, rgba(45, 31, 61, 0.4) 100%)',
    borderRadius: '16px',
    padding: '20px',
    marginBottom: '24px',
    border: '1px solid rgba(212, 168, 75, 0.3)',
  },
  aggregatedTitle: {
    color: '#D4A84B',
    margin: '0 0 16px',
    fontSize: '1.1rem',
    fontWeight: 600,
  },
  aggregatedGrid: {
    display: 'grid',
    gridTemplateColumns: 'repeat(auto-fit, minmax(140px, 1fr))',
    gap: '16px',
  },
  aggregatedStat: {
    display: 'flex',
    flexDirection: 'column',
    alignItems: 'center',
    padding: '12px',
    background: 'rgba(13, 8, 17, 0.5)',
    borderRadius: '10px',
  },
  aggregatedLabel: {
    fontSize: '0.75rem',
    color: '#B8B4C8',
    textTransform: 'uppercase',
    letterSpacing: '0.5px',
    marginBottom: '6px',
  },
  aggregatedValue: {
    fontSize: '1.2rem',
    fontWeight: 700,
    color: '#F0EDF4',
  },
  networkGrid: {
    display: 'grid',
    gridTemplateColumns: 'repeat(auto-fill, minmax(280px, 1fr))',
    gap: '20px',
  },
  networkCard: {
    background: 'linear-gradient(180deg, rgba(45, 31, 61, 0.5) 0%, rgba(26, 15, 30, 0.7) 100%)',
    borderRadius: '16px',
    padding: '20px',
    border: '1px solid rgba(74, 44, 90, 0.4)',
    cursor: 'pointer',
    transition: 'all 0.2s ease',
  },
  networkCardSelected: {
    border: '2px solid #D4A84B',
    boxShadow: '0 0 20px rgba(212, 168, 75, 0.2)',
  },
  networkCardInactive: {
    opacity: 0.6,
  },
  networkHeader: {
    display: 'flex',
    alignItems: 'center',
    gap: '12px',
    marginBottom: '12px',
  },
  networkIcon: {
    fontSize: '2rem',
    color: '#D4A84B',
  },
  networkInfo: {
    flex: 1,
    display: 'flex',
    flexDirection: 'column',
  },
  networkName: {
    fontSize: '1.1rem',
    fontWeight: 600,
    color: '#F0EDF4',
  },
  networkSymbol: {
    fontSize: '0.85rem',
    color: '#B8B4C8',
  },
  activeIndicator: {
    color: '#4ade80',
    fontSize: '1rem',
  },
  inactiveIndicator: {
    color: '#6b7280',
    fontSize: '1rem',
  },
  networkDetails: {
    display: 'flex',
    gap: '10px',
    marginBottom: '16px',
  },
  algorithmBadge: {
    padding: '4px 10px',
    background: 'rgba(123, 94, 167, 0.3)',
    borderRadius: '6px',
    fontSize: '0.75rem',
    color: '#B8B4C8',
    textTransform: 'uppercase',
  },
  portInfo: {
    padding: '4px 10px',
    background: 'rgba(13, 8, 17, 0.5)',
    borderRadius: '6px',
    fontSize: '0.75rem',
    color: '#B8B4C8',
  },
  statsSection: {
    borderTop: '1px solid rgba(74, 44, 90, 0.3)',
    paddingTop: '12px',
    marginBottom: '12px',
  },
  userStatsSection: {
    borderTop: '1px solid rgba(212, 168, 75, 0.3)',
    paddingTop: '12px',
    marginBottom: '12px',
  },
  userStatsHeader: {
    fontSize: '0.75rem',
    color: '#D4A84B',
    textTransform: 'uppercase',
    letterSpacing: '0.5px',
    marginBottom: '8px',
  },
  statRow: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    padding: '4px 0',
  },
  statLabel: {
    fontSize: '0.85rem',
    color: '#B8B4C8',
  },
  statValue: {
    fontSize: '0.9rem',
    fontWeight: 600,
    color: '#F0EDF4',
  },
  statValueGold: {
    fontSize: '0.9rem',
    fontWeight: 600,
    color: '#D4A84B',
  },
  feeInfo: {
    fontSize: '0.75rem',
    color: '#7A7490',
    textAlign: 'center',
    paddingTop: '12px',
    borderTop: '1px solid rgba(74, 44, 90, 0.2)',
  },
};

export default NetworkSelector;
