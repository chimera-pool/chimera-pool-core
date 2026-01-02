import React, { useState, useEffect } from 'react';
import {
  ResponsiveContainer,
  AreaChart,
  Area,
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ReferenceLine
} from 'recharts';
// Equipment Management Page Component
interface Equipment {
  id: string;
  name: string;
  type: string;
  status: string;
  worker_name: string;
  model: string;
  current_hashrate: number;
  average_hashrate: number;
  temperature: number;
  power_usage: number;
  latency: number;
  shares_accepted: number;
  shares_rejected: number;
  uptime: number;
  last_seen: string;
  total_earnings: number;
  payout_splits: PayoutSplit[];
  // Uptime/Downtime tracking
  connected_at: string;
  total_connection_time: number; // seconds since first connection
  total_downtime: number; // seconds of downtime
  downtime_incidents: number;
  last_downtime_start?: string;
  last_downtime_end?: string;
}

interface EquipmentSettings {
  id: string;
  name: string;
  worker_name: string;
  power_limit: number;
  target_temperature: number;
  auto_restart: boolean;
  notification_email: boolean;
  notification_offline_threshold: number; // minutes before alert
  difficulty_mode: 'auto' | 'fixed';
  fixed_difficulty?: number;
  // New elite controls
  network_id: string;
  power_mode: 'performance' | 'balanced' | 'efficiency';
  miner_group_id?: string;
  schedule_enabled: boolean;
  schedule?: MiningSchedule;
}

interface MiningSchedule {
  monday: { enabled: boolean; start: string; end: string };
  tuesday: { enabled: boolean; start: string; end: string };
  wednesday: { enabled: boolean; start: string; end: string };
  thursday: { enabled: boolean; start: string; end: string };
  friday: { enabled: boolean; start: string; end: string };
  saturday: { enabled: boolean; start: string; end: string };
  sunday: { enabled: boolean; start: string; end: string };
}

interface NetworkConfig {
  id: string;
  name: string;
  symbol: string;
  algorithm: string;
  is_active: boolean;
}

interface MinerGroup {
  id: string;
  name: string;
  color: string;
  description?: string;
}

interface PayoutSplit {
  id: string;
  wallet_address: string;
  percentage: number;
  label: string;
  is_active: boolean;
}

interface EquipmentWallet {
  id: string;
  wallet_address: string;
  label: string;
  is_primary: boolean;
  currency: string;
  // Enhanced wallet controls
  wallet_type: 'hot' | 'cold' | 'exchange' | 'staking';
  status: 'active' | 'inactive' | 'locked';
  min_payout_threshold: number;
  allocation_percent: number;
}

function EquipmentPage({ token, user, showMessage }: { token: string; user: any; showMessage: (type: 'success' | 'error', text: string) => void }) {
  const [equipment, setEquipment] = useState<Equipment[]>([]);
  const [wallets, setWallets] = useState<EquipmentWallet[]>([]);
  const [loading, setLoading] = useState(true);
  const [activeTab, setActiveTab] = useState<'equipment' | 'wallets' | 'alerts' | 'networks' | 'groups'>('equipment');
  const [selectedEquipment, setSelectedEquipment] = useState<Equipment | null>(null);
  const [showAddWalletModal, setShowAddWalletModal] = useState(false);
  const [newWallet, setNewWallet] = useState({ address: '', label: '', is_primary: false, wallet_type: 'hot' as const, min_payout_threshold: 0.01 });
  const [editingSplits, setEditingSplits] = useState<{ equipmentId: string; splits: PayoutSplit[] } | null>(null);
  const [showSettingsModal, setShowSettingsModal] = useState<Equipment | null>(null);
  const [showChartsModal, setShowChartsModal] = useState<Equipment | null>(null);
  const [equipmentSettings, setEquipmentSettings] = useState<EquipmentSettings | null>(null);
  const [chartTimeRange, setChartTimeRange] = useState<'1h' | '6h' | '24h' | '7d' | '30d'>('24h');
  
  // Elite control state
  const [networks, setNetworks] = useState<NetworkConfig[]>([
    { id: 'litecoin', name: 'Litecoin', symbol: 'LTC', algorithm: 'Scrypt', is_active: true },
    { id: 'blockdag', name: 'BlockDAG', symbol: 'BDAG', algorithm: 'KHeavyHash', is_active: false },
    { id: 'dogecoin', name: 'Dogecoin', symbol: 'DOGE', algorithm: 'Scrypt', is_active: false },
  ]);
  const [minerGroups, setMinerGroups] = useState<MinerGroup[]>([
    { id: 'default', name: 'Default', color: '#00d4ff', description: 'All miners' },
  ]);
  const [showAddGroupModal, setShowAddGroupModal] = useState(false);
  const [newGroup, setNewGroup] = useState({ name: '', color: '#D4A84B', description: '' });
  const [settingsTab, setSettingsTab] = useState<'general' | 'network' | 'wallet' | 'schedule'>('general');

  useEffect(() => {
    fetchData();
  }, []);

  const fetchData = async () => {
    setLoading(true);
    try {
      const headers = { 'Authorization': `Bearer ${token}` };
      const [eqRes, walletRes] = await Promise.all([
        fetch('/api/v1/user/equipment', { headers }),
        fetch('/api/v1/user/wallets', { headers })
      ]);

      if (eqRes.ok) {
        const data = await eqRes.json();
        // Map API response to Equipment interface
        const mappedEquipment = (data.equipment || []).map((eq: any) => ({
          id: eq.id,
          name: eq.name,
          type: eq.type,
          status: eq.status,
          worker_name: eq.worker_name,
          model: eq.model || getModelFromType(eq.type),
          current_hashrate: eq.current_hashrate,
          average_hashrate: eq.average_hashrate,
          temperature: eq.temperature || 0,
          power_usage: eq.power_usage || 0,
          latency: eq.latency || 0,
          shares_accepted: eq.shares_accepted,
          shares_rejected: eq.shares_rejected,
          uptime: eq.uptime,
          last_seen: eq.last_seen,
          total_earnings: eq.total_earnings,
          payout_splits: [{ id: 's1', wallet_address: user?.payout_address || '', percentage: 100, label: 'Primary', is_active: true }],
          connected_at: eq.connected_at,
          total_connection_time: eq.total_connection_time,
          total_downtime: eq.total_downtime || 0,
          downtime_incidents: eq.downtime_incidents || 0,
        }));
        setEquipment(mappedEquipment.length > 0 ? mappedEquipment : []);
      } else {
        setEquipment([]);
      }

      if (walletRes.ok) {
        const data = await walletRes.json();
        setWallets(data.wallets || []);
      }
    } catch (error) {
      console.error('Failed to fetch equipment data:', error);
      setEquipment([]);
    } finally {
      setLoading(false);
    }
  };

  // Helper to determine model name from equipment type
  const getModelFromType = (type: string): string => {
    switch (type) {
      case 'blockdag_x100': return 'BlockDAG X100';
      case 'blockdag_x30': return 'BlockDAG X30';
      case 'asic': return 'ASIC Miner';
      case 'gpu': return 'GPU Rig';
      default: return 'Mining Equipment';
    }
  };

  // Generate mock equipment for demo
  const generateMockEquipment = (): Equipment[] => {
    const now = Date.now();
    return [
      {
        id: 'eq-001',
        name: 'Main X100 ASIC',
        type: 'blockdag_x100',
        status: 'mining',
        worker_name: 'x100-main',
        model: 'BlockDAG X100',
        current_hashrate: 150000000,
        average_hashrate: 148000000,
        temperature: 62.5,
        power_usage: 1200,
        latency: 25.3,
        shares_accepted: 15420,
        shares_rejected: 45,
        uptime: 432000,
        last_seen: new Date().toISOString(),
        total_earnings: 125.5,
        payout_splits: [{ id: 's1', wallet_address: user.payout_address || 'bdag1...', percentage: 100, label: 'Primary', is_active: true }],
        connected_at: new Date(now - 518400000).toISOString(), // 6 days ago
        total_connection_time: 518400, // 6 days in seconds
        total_downtime: 3600, // 1 hour downtime
        downtime_incidents: 1,
        last_downtime_start: new Date(now - 172800000).toISOString(),
        last_downtime_end: new Date(now - 169200000).toISOString()
      },
      {
        id: 'eq-002',
        name: 'GPU Rig #1',
        type: 'gpu',
        status: 'mining',
        worker_name: 'gpu-rig1',
        model: 'RTX 4090 x4',
        current_hashrate: 45000000,
        average_hashrate: 44500000,
        temperature: 68.2,
        power_usage: 1400,
        latency: 32.1,
        shares_accepted: 8920,
        shares_rejected: 28,
        uptime: 259200,
        last_seen: new Date().toISOString(),
        total_earnings: 45.2,
        payout_splits: [{ id: 's2', wallet_address: user.payout_address || 'bdag1...', percentage: 100, label: 'Primary', is_active: true }],
        connected_at: new Date(now - 345600000).toISOString(), // 4 days ago
        total_connection_time: 345600, // 4 days in seconds
        total_downtime: 7200, // 2 hours downtime
        downtime_incidents: 3,
        last_downtime_start: new Date(now - 86400000).toISOString(),
        last_downtime_end: new Date(now - 82800000).toISOString()
      },
      {
        id: 'eq-003',
        name: 'Backup X30',
        type: 'blockdag_x30',
        status: 'offline',
        worker_name: 'x30-backup',
        model: 'BlockDAG X30',
        current_hashrate: 0,
        average_hashrate: 35000000,
        temperature: 0,
        power_usage: 0,
        latency: 0,
        shares_accepted: 5200,
        shares_rejected: 15,
        uptime: 0,
        last_seen: new Date(now - 86400000).toISOString(),
        total_earnings: 22.8,
        payout_splits: [{ id: 's3', wallet_address: user.payout_address || 'bdag1...', percentage: 100, label: 'Primary', is_active: true }],
        connected_at: new Date(now - 604800000).toISOString(), // 7 days ago
        total_connection_time: 604800, // 7 days in seconds
        total_downtime: 86400, // 24 hours downtime (currently offline)
        downtime_incidents: 2,
        last_downtime_start: new Date(now - 86400000).toISOString(),
        last_downtime_end: undefined // Currently offline
      }
    ];
  };

  // Format uptime percentage
  const formatUptimePercent = (eq: Equipment) => {
    if (eq.total_connection_time === 0) return '0%';
    const uptimeSeconds = eq.total_connection_time - eq.total_downtime;
    return ((uptimeSeconds / eq.total_connection_time) * 100).toFixed(2) + '%';
  };

  // Format duration in human readable form
  const formatDuration = (seconds: number) => {
    if (seconds < 60) return `${seconds}s`;
    if (seconds < 3600) return `${Math.floor(seconds / 60)}m`;
    if (seconds < 86400) return `${Math.floor(seconds / 3600)}h ${Math.floor((seconds % 3600) / 60)}m`;
    const days = Math.floor(seconds / 86400);
    const hours = Math.floor((seconds % 86400) / 3600);
    return `${days}d ${hours}h`;
  };

  // Open settings modal for equipment
  const openSettingsModal = (eq: Equipment) => {
    setShowSettingsModal(eq);
    setSettingsTab('general');
    const defaultSchedule = {
      monday: { enabled: true, start: '00:00', end: '23:59' },
      tuesday: { enabled: true, start: '00:00', end: '23:59' },
      wednesday: { enabled: true, start: '00:00', end: '23:59' },
      thursday: { enabled: true, start: '00:00', end: '23:59' },
      friday: { enabled: true, start: '00:00', end: '23:59' },
      saturday: { enabled: true, start: '00:00', end: '23:59' },
      sunday: { enabled: true, start: '00:00', end: '23:59' },
    };
    setEquipmentSettings({
      id: eq.id,
      name: eq.name,
      worker_name: eq.worker_name,
      power_limit: eq.power_usage,
      target_temperature: 70,
      auto_restart: true,
      notification_email: true,
      notification_offline_threshold: 5,
      difficulty_mode: 'auto',
      fixed_difficulty: undefined,
      // Elite controls
      network_id: 'litecoin',
      power_mode: 'performance',
      miner_group_id: 'default',
      schedule_enabled: false,
      schedule: defaultSchedule,
    });
  };

  // Generate mock chart data for equipment
  const generateChartData = (eq: Equipment, range: string) => {
    const points = range === '1h' ? 12 : range === '6h' ? 36 : range === '24h' ? 48 : range === '7d' ? 168 : 720;
    const data = [];
    const baseHashrate = eq.average_hashrate;
    const now = Date.now();
    for (let i = points; i >= 0; i--) {
      const timestamp = new Date(now - (i * (range === '1h' ? 300000 : range === '6h' ? 600000 : range === '24h' ? 1800000 : range === '7d' ? 3600000 : 3600000)));
      const variance = (Math.random() - 0.5) * 0.1 * baseHashrate;
      data.push({
        time: timestamp.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
        hashrate: Math.max(0, baseHashrate + variance),
        temperature: eq.temperature > 0 ? eq.temperature + (Math.random() - 0.5) * 5 : 0,
        power: eq.power_usage > 0 ? eq.power_usage + (Math.random() - 0.5) * 100 : 0
      });
    }
    return data;
  };

  const formatHashrate = (h: number) => {
    if (h >= 1e12) return `${(h / 1e12).toFixed(2)} TH/s`;
    if (h >= 1e9) return `${(h / 1e9).toFixed(2)} GH/s`;
    if (h >= 1e6) return `${(h / 1e6).toFixed(2)} MH/s`;
    if (h >= 1e3) return `${(h / 1e3).toFixed(2)} KH/s`;
    return `${h.toFixed(2)} H/s`;
  };

  const formatUptime = (seconds: number) => {
    const days = Math.floor(seconds / 86400);
    const hours = Math.floor((seconds % 86400) / 3600);
    if (days > 0) return `${days}d ${hours}h`;
    return `${hours}h`;
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'mining': return '#4ade80';
      case 'online': return '#00d4ff';
      case 'idle': return '#f59e0b';
      case 'offline': return '#888';
      case 'error': return '#ef4444';
      default: return '#888';
    }
  };

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'mining': return '⛏️';
      case 'online': return '🟢';
      case 'idle': return '💤';
      case 'offline': return '⚫';
      case 'error': return '🔴';
      default: return '❓';
    }
  };

  const getTypeIcon = (type: string) => {
    switch (type) {
      case 'blockdag_x100': return '⚡';
      case 'blockdag_x30': return '⚡';
      case 'gpu': return '🎮';
      case 'cpu': return '💻';
      case 'asic': return '🔲';
      default: return '⚙️';
    }
  };

  const totalStats = {
    totalEquipment: equipment.length,
    online: equipment.filter(e => ['mining', 'online', 'idle'].includes(e.status)).length,
    offline: equipment.filter(e => e.status === 'offline').length,
    errors: equipment.filter(e => e.status === 'error').length,
    totalHashrate: equipment.reduce((sum, e) => sum + e.current_hashrate, 0),
    totalEarnings: equipment.reduce((sum, e) => sum + e.total_earnings, 0),
    avgLatency: equipment.filter(e => e.latency > 0).reduce((sum, e, _, arr) => sum + e.latency / arr.length, 0)
  };

  return (
    <div style={{ padding: '20px', maxWidth: '1400px', margin: '0 auto' }}>
      {/* Header */}
      <div style={{ marginBottom: '30px' }}>
        <h1 style={{ color: '#D4A84B', margin: '0 0 10px', fontSize: '1.8rem', fontWeight: 700 }}>⚙️ Equipment Control Center</h1>
        <p style={{ color: '#B8B4C8', margin: 0 }}>Manage your mining hardware, monitor performance, and configure payouts</p>
      </div>

      {/* Stats Overview */}
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(180px, 1fr))', gap: '15px', marginBottom: '30px' }}>
        <div style={eqStyles.statCard}>
          <div style={eqStyles.statIcon}>🖥️</div>
          <div style={eqStyles.statValue}>{totalStats.totalEquipment}</div>
          <div style={eqStyles.statLabel}>Total Equipment</div>
        </div>
        <div style={{...eqStyles.statCard, borderColor: '#4ade80'}}>
          <div style={eqStyles.statIcon}>✅</div>
          <div style={{...eqStyles.statValue, color: '#4ade80'}}>{totalStats.online}</div>
          <div style={eqStyles.statLabel}>Online</div>
        </div>
        <div style={{...eqStyles.statCard, borderColor: '#888'}}>
          <div style={eqStyles.statIcon}>⚫</div>
          <div style={{...eqStyles.statValue, color: '#888'}}>{totalStats.offline}</div>
          <div style={eqStyles.statLabel}>Offline</div>
        </div>
        <div style={eqStyles.statCard}>
          <div style={eqStyles.statIcon}>⚡</div>
          <div style={eqStyles.statValue}>{formatHashrate(totalStats.totalHashrate)}</div>
          <div style={eqStyles.statLabel}>Total Hashrate</div>
        </div>
        <div style={eqStyles.statCard}>
          <div style={eqStyles.statIcon}>💰</div>
          <div style={eqStyles.statValue}>{totalStats.totalEarnings.toFixed(2)}</div>
          <div style={eqStyles.statLabel}>Total Earnings (BDAG)</div>
        </div>
        <div style={eqStyles.statCard}>
          <div style={eqStyles.statIcon}>📡</div>
          <div style={eqStyles.statValue}>{totalStats.avgLatency.toFixed(1)} ms</div>
          <div style={eqStyles.statLabel}>Avg Latency</div>
        </div>
      </div>

      {/* Tabs */}
      <div style={{ display: 'flex', gap: '8px', marginBottom: '20px', borderBottom: '2px solid #2a2a4a', paddingBottom: '10px', flexWrap: 'wrap' }}>
        <button
          style={{...eqStyles.tab, ...(activeTab === 'equipment' ? eqStyles.tabActive : {})}}
          onClick={() => setActiveTab('equipment')}
          data-testid="equipment-tab"
        >
          🖥️ Equipment ({equipment.length})
        </button>
        <button
          style={{...eqStyles.tab, ...(activeTab === 'wallets' ? eqStyles.tabActive : {})}}
          onClick={() => setActiveTab('wallets')}
          data-testid="wallets-tab"
        >
          💼 Wallets ({wallets.length})
        </button>
        <button
          style={{...eqStyles.tab, ...(activeTab === 'networks' ? eqStyles.tabActive : {})}}
          onClick={() => setActiveTab('networks')}
          data-testid="networks-tab"
        >
          🌐 Networks ({networks.filter(n => n.is_active).length})
        </button>
        <button
          style={{...eqStyles.tab, ...(activeTab === 'groups' ? eqStyles.tabActive : {})}}
          onClick={() => setActiveTab('groups')}
          data-testid="groups-tab"
        >
          📁 Groups ({minerGroups.length})
        </button>
        <button
          style={{...eqStyles.tab, ...(activeTab === 'alerts' ? eqStyles.tabActive : {})}}
          onClick={() => setActiveTab('alerts')}
          data-testid="alerts-tab"
        >
          🔔 Alerts
        </button>
      </div>

      {loading ? (
        <div style={{ textAlign: 'center', padding: '60px', color: '#00d4ff' }}>Loading equipment...</div>
      ) : (
        <>
          {/* Equipment Tab */}
          {activeTab === 'equipment' && (
            <div style={{ display: 'grid', gap: '15px' }}>
              {equipment.map(eq => (
                <div 
                  key={eq.id} 
                  style={{...eqStyles.equipmentCard, borderLeftColor: getStatusColor(eq.status)}}
                  onClick={() => setSelectedEquipment(selectedEquipment?.id === eq.id ? null : eq)}
                >
                  <div style={eqStyles.equipmentHeader}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                      <span style={{ fontSize: '1.5rem' }}>{getTypeIcon(eq.type)}</span>
                      <div>
                        <h3 style={{ color: '#e0e0e0', margin: 0, fontSize: '1.1rem' }}>{eq.name}</h3>
                        <p style={{ color: '#888', margin: '2px 0 0', fontSize: '0.85rem' }}>{eq.model} • {eq.worker_name}</p>
                      </div>
                    </div>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '15px' }}>
                      <span style={{ color: getStatusColor(eq.status), fontWeight: 'bold', display: 'flex', alignItems: 'center', gap: '5px' }}>
                        {getStatusIcon(eq.status)} {eq.status.toUpperCase()}
                      </span>
                      <span style={{ color: '#888', cursor: 'pointer' }}>{selectedEquipment?.id === eq.id ? '▲' : '▼'}</span>
                    </div>
                  </div>

                  {/* Quick Stats Row */}
                  <div style={eqStyles.quickStats}>
                    <div style={eqStyles.quickStat}>
                      <span style={eqStyles.quickLabel}>Hashrate</span>
                      <span style={eqStyles.quickValue}>{formatHashrate(eq.current_hashrate)}</span>
                    </div>
                    <div style={eqStyles.quickStat}>
                      <span style={eqStyles.quickLabel}>Temp</span>
                      <span style={{...eqStyles.quickValue, color: eq.temperature > 80 ? '#ef4444' : eq.temperature > 70 ? '#f59e0b' : '#4ade80'}}>
                        {eq.temperature > 0 ? `${eq.temperature}°C` : '--'}
                      </span>
                    </div>
                    <div style={eqStyles.quickStat}>
                      <span style={eqStyles.quickLabel}>Power</span>
                      <span style={eqStyles.quickValue}>{eq.power_usage > 0 ? `${eq.power_usage}W` : '--'}</span>
                    </div>
                    <div style={eqStyles.quickStat}>
                      <span style={eqStyles.quickLabel}>Latency</span>
                      <span style={{...eqStyles.quickValue, color: eq.latency > 50 ? '#f59e0b' : '#4ade80'}}>
                        {eq.latency > 0 ? `${eq.latency.toFixed(1)}ms` : '--'}
                      </span>
                    </div>
                    <div style={eqStyles.quickStat}>
                      <span style={eqStyles.quickLabel}>Shares</span>
                      <span style={eqStyles.quickValue}>{eq.shares_accepted.toLocaleString()}</span>
                    </div>
                    <div style={eqStyles.quickStat}>
                      <span style={eqStyles.quickLabel}>Uptime</span>
                      <span style={eqStyles.quickValue}>{eq.uptime > 0 ? formatUptime(eq.uptime) : '--'}</span>
                    </div>
                  </div>

                  {/* Expanded Details */}
                  {selectedEquipment?.id === eq.id && (
                    <div style={eqStyles.expandedDetails}>
                      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))', gap: '20px' }}>
                        {/* Performance */}
                        <div style={eqStyles.detailSection}>
                          <h4 style={eqStyles.detailTitle}>📊 Performance</h4>
                          <div style={eqStyles.detailRow}>
                            <span>Current Hashrate:</span>
                            <span style={{ color: '#00d4ff' }}>{formatHashrate(eq.current_hashrate)}</span>
                          </div>
                          <div style={eqStyles.detailRow}>
                            <span>Average Hashrate:</span>
                            <span>{formatHashrate(eq.average_hashrate)}</span>
                          </div>
                          <div style={eqStyles.detailRow}>
                            <span>Efficiency:</span>
                            <span>{eq.power_usage > 0 ? `${(eq.current_hashrate / eq.power_usage / 1000).toFixed(2)} MH/W` : '--'}</span>
                          </div>
                          <div style={eqStyles.detailRow}>
                            <span>Acceptance Rate:</span>
                            <span style={{ color: '#4ade80' }}>
                              {((eq.shares_accepted / (eq.shares_accepted + eq.shares_rejected)) * 100).toFixed(2)}%
                            </span>
                          </div>
                        </div>

                        {/* Hardware */}
                        <div style={eqStyles.detailSection}>
                          <h4 style={eqStyles.detailTitle}>🔧 Hardware</h4>
                          <div style={eqStyles.detailRow}>
                            <span>Temperature:</span>
                            <span style={{ color: eq.temperature > 80 ? '#ef4444' : '#4ade80' }}>
                              {eq.temperature > 0 ? `${eq.temperature}°C` : 'N/A'}
                            </span>
                          </div>
                          <div style={eqStyles.detailRow}>
                            <span>Power Usage:</span>
                            <span>{eq.power_usage > 0 ? `${eq.power_usage}W` : 'N/A'}</span>
                          </div>
                          <div style={eqStyles.detailRow}>
                            <span>Network Latency:</span>
                            <span style={{ color: eq.latency > 50 ? '#f59e0b' : '#4ade80' }}>
                              {eq.latency > 0 ? `${eq.latency.toFixed(1)}ms` : 'N/A'}
                            </span>
                          </div>
                          <div style={eqStyles.detailRow}>
                            <span>Last Seen:</span>
                            <span>{new Date(eq.last_seen).toLocaleString()}</span>
                          </div>
                        </div>

                        {/* Uptime/Downtime */}
                        <div style={eqStyles.detailSection}>
                          <h4 style={eqStyles.detailTitle}>⏱️ Uptime & Availability</h4>
                          <div style={eqStyles.detailRow}>
                            <span>Connected Since:</span>
                            <span>{new Date(eq.connected_at).toLocaleDateString()}</span>
                          </div>
                          <div style={eqStyles.detailRow}>
                            <span>Total Connection Time:</span>
                            <span>{formatDuration(eq.total_connection_time)}</span>
                          </div>
                          <div style={eqStyles.detailRow}>
                            <span>Total Downtime:</span>
                            <span style={{ color: eq.total_downtime > 3600 ? '#ef4444' : '#4ade80' }}>
                              {formatDuration(eq.total_downtime)}
                            </span>
                          </div>
                          <div style={eqStyles.detailRow}>
                            <span>Uptime Percentage:</span>
                            <span style={{ color: parseFloat(formatUptimePercent(eq)) > 99 ? '#4ade80' : parseFloat(formatUptimePercent(eq)) > 95 ? '#f59e0b' : '#ef4444', fontWeight: 'bold' }}>
                              {formatUptimePercent(eq)}
                            </span>
                          </div>
                          <div style={eqStyles.detailRow}>
                            <span>Downtime Incidents:</span>
                            <span style={{ color: eq.downtime_incidents > 3 ? '#ef4444' : '#888' }}>{eq.downtime_incidents}</span>
                          </div>
                          {eq.last_downtime_start && (
                            <div style={eqStyles.detailRow}>
                              <span>Last Downtime:</span>
                              <span style={{ fontSize: '0.8rem' }}>
                                {new Date(eq.last_downtime_start).toLocaleString()}
                                {eq.last_downtime_end ? ` → ${new Date(eq.last_downtime_end).toLocaleTimeString()}` : ' (ongoing)'}
                              </span>
                            </div>
                          )}
                          {/* Uptime Bar */}
                          <div style={{ marginTop: '10px' }}>
                            <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '4px' }}>
                              <span style={{ fontSize: '0.75rem', color: '#888' }}>Availability</span>
                              <span style={{ fontSize: '0.75rem', color: '#888' }}>{formatUptimePercent(eq)}</span>
                            </div>
                            <div style={{ height: '8px', backgroundColor: '#2a2a4a', borderRadius: '4px', overflow: 'hidden' }}>
                              <div style={{
                                height: '100%',
                                width: formatUptimePercent(eq),
                                backgroundColor: parseFloat(formatUptimePercent(eq)) > 99 ? '#4ade80' : parseFloat(formatUptimePercent(eq)) > 95 ? '#f59e0b' : '#ef4444',
                                borderRadius: '4px',
                                transition: 'width 0.3s'
                              }} />
                            </div>
                          </div>
                        </div>

                        {/* Earnings */}
                        <div style={eqStyles.detailSection}>
                          <h4 style={eqStyles.detailTitle}>💰 Earnings</h4>
                          <div style={eqStyles.detailRow}>
                            <span>Total Earnings:</span>
                            <span style={{ color: '#9b59b6' }}>{eq.total_earnings.toFixed(4)} BDAG</span>
                          </div>
                          <div style={eqStyles.detailRow}>
                            <span>Shares Accepted:</span>
                            <span style={{ color: '#4ade80' }}>{eq.shares_accepted.toLocaleString()}</span>
                          </div>
                          <div style={eqStyles.detailRow}>
                            <span>Shares Rejected:</span>
                            <span style={{ color: '#ef4444' }}>{eq.shares_rejected.toLocaleString()}</span>
                          </div>
                        </div>

                        {/* Payout Splits */}
                        <div style={eqStyles.detailSection}>
                          <h4 style={eqStyles.detailTitle}>💼 Payout Distribution</h4>
                          {eq.payout_splits.map(split => (
                            <div key={split.id} style={eqStyles.detailRow}>
                              <span>{split.label || 'Wallet'}:</span>
                              <span>{split.percentage}% → {split.wallet_address.slice(0, 10)}...</span>
                            </div>
                          ))}
                          <button
                            style={eqStyles.editSplitsBtn}
                            onClick={(e) => { e.stopPropagation(); setEditingSplits({ equipmentId: eq.id, splits: [...eq.payout_splits] }); }}
                          >
                            ✏️ Edit Payout Splits
                          </button>
                        </div>
                      </div>

                      {/* Action Buttons */}
                      <div style={{ display: 'flex', gap: '10px', marginTop: '20px', flexWrap: 'wrap' }}>
                        <button style={eqStyles.actionBtn} onClick={(e) => { e.stopPropagation(); setShowChartsModal(eq); }}>📊 View Charts</button>
                        <button style={eqStyles.actionBtn} onClick={(e) => { e.stopPropagation(); showMessage('success', `Restart signal sent to ${eq.name}`); }}>🔄 Restart</button>
                        <button style={{...eqStyles.actionBtn, borderColor: '#f59e0b', color: '#f59e0b'}} onClick={(e) => { e.stopPropagation(); openSettingsModal(eq); }}>⚙️ Settings</button>
                        <button style={{...eqStyles.actionBtn, borderColor: '#ef4444', color: '#ef4444'}} onClick={(e) => { e.stopPropagation(); showMessage('error', 'Remove equipment from Account Settings'); }}>🗑️ Remove</button>
                      </div>
                    </div>
                  )}
                </div>
              ))}
            </div>
          )}

          {/* Wallets Tab */}
          {activeTab === 'wallets' && (
            <div>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '20px' }}>
                <p style={{ color: '#888', margin: 0 }}>Manage your payout wallet addresses</p>
                <button
                  style={eqStyles.addBtn}
                  onClick={() => setShowAddWalletModal(true)}
                >
                  ➕ Add Wallet
                </button>
              </div>

              {wallets.length === 0 ? (
                <div style={{ textAlign: 'center', padding: '40px', backgroundColor: '#0a0a15', borderRadius: '8px', border: '1px solid #2a2a4a' }}>
                  <p style={{ color: '#888', marginBottom: '20px' }}>No wallets configured yet. Add a wallet to receive payouts.</p>
                  <button style={eqStyles.addBtn} onClick={() => setShowAddWalletModal(true)}>➕ Add Your First Wallet</button>
                </div>
              ) : (
                <div style={{ display: 'grid', gap: '12px' }}>
                  {wallets.map(wallet => (
                    <div key={wallet.id} style={eqStyles.walletCard}>
                      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', flexWrap: 'wrap', gap: '10px' }}>
                        <div style={{ minWidth: 0, flex: 1 }}>
                          <h3 style={{ color: '#e0e0e0', margin: '0 0 4px', display: 'flex', alignItems: 'center', gap: '6px', fontSize: '0.95rem', flexWrap: 'wrap' }}>
                            {wallet.label || 'Unnamed Wallet'}
                            {wallet.is_primary && <span style={eqStyles.primaryBadge}>PRIMARY</span>}
                          </h3>
                          <code style={{ color: '#00d4ff', fontSize: '0.8rem', wordBreak: 'break-all' }}>{wallet.wallet_address}</code>
                        </div>
                        <div style={{ display: 'flex', gap: '8px', flexShrink: 0 }}>
                          {!wallet.is_primary && (
                            <button style={eqStyles.smallBtn}>Set Primary</button>
                          )}
                          <button style={{...eqStyles.smallBtn, borderColor: '#ef4444', color: '#ef4444'}}>Remove</button>
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>
          )}

          {/* Networks Tab */}
          {activeTab === 'networks' && (
            <div data-testid="networks-content">
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '20px' }}>
                <div>
                  <h3 style={{ color: '#D4A84B', margin: '0 0 4px' }}>Network Configuration</h3>
                  <p style={{ color: '#888', margin: 0, fontSize: '0.9rem' }}>Select which networks your equipment can mine on</p>
                </div>
              </div>
              
              <div style={{ display: 'grid', gap: '12px' }}>
                {networks.map(network => (
                  <div key={network.id} style={{
                    ...eqStyles.walletCard,
                    borderLeft: `4px solid ${network.is_active ? '#4ade80' : '#666'}`,
                    opacity: network.is_active ? 1 : 0.7
                  }}>
                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', flexWrap: 'wrap', gap: '12px' }}>
                      <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                        <div style={{ 
                          width: '48px', height: '48px', borderRadius: '12px', 
                          background: 'linear-gradient(135deg, rgba(212, 168, 75, 0.2) 0%, rgba(123, 94, 167, 0.2) 100%)',
                          display: 'flex', alignItems: 'center', justifyContent: 'center',
                          fontSize: '1.5rem'
                        }}>
                          {network.symbol === 'LTC' ? '🪙' : network.symbol === 'BDAG' ? '⚡' : '🔷'}
                        </div>
                        <div>
                          <h4 style={{ color: '#e0e0e0', margin: '0 0 4px', fontSize: '1.1rem' }}>{network.name}</h4>
                          <div style={{ display: 'flex', gap: '12px', fontSize: '0.85rem', color: '#888' }}>
                            <span>Symbol: <span style={{ color: '#D4A84B' }}>{network.symbol}</span></span>
                            <span>Algorithm: <span style={{ color: '#7B5EA7' }}>{network.algorithm}</span></span>
                          </div>
                        </div>
                      </div>
                      <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                        <span style={{ 
                          padding: '6px 12px', borderRadius: '20px', fontSize: '0.8rem', fontWeight: 600,
                          backgroundColor: network.is_active ? 'rgba(74, 222, 128, 0.2)' : 'rgba(136, 136, 136, 0.2)',
                          color: network.is_active ? '#4ade80' : '#888'
                        }}>
                          {network.is_active ? '● Active' : '○ Inactive'}
                        </span>
                        <button 
                          style={{...eqStyles.smallBtn, borderColor: network.is_active ? '#ef4444' : '#4ade80', color: network.is_active ? '#ef4444' : '#4ade80'}}
                          onClick={() => {
                            setNetworks(networks.map(n => n.id === network.id ? {...n, is_active: !n.is_active} : n));
                            showMessage('success', `${network.name} ${network.is_active ? 'disabled' : 'enabled'}`);
                          }}
                        >
                          {network.is_active ? 'Disable' : 'Enable'}
                        </button>
                      </div>
                    </div>
                    {network.is_active && (
                      <div style={{ marginTop: '16px', paddingTop: '16px', borderTop: '1px solid rgba(74, 44, 90, 0.3)' }}>
                        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(150px, 1fr))', gap: '12px' }}>
                          <div style={{ background: 'rgba(13, 8, 17, 0.6)', padding: '12px', borderRadius: '8px' }}>
                            <div style={{ color: '#888', fontSize: '0.75rem', marginBottom: '4px' }}>EQUIPMENT MINING</div>
                            <div style={{ color: '#D4A84B', fontSize: '1.1rem', fontWeight: 600 }}>{equipment.length}</div>
                          </div>
                          <div style={{ background: 'rgba(13, 8, 17, 0.6)', padding: '12px', borderRadius: '8px' }}>
                            <div style={{ color: '#888', fontSize: '0.75rem', marginBottom: '4px' }}>POOL FEE</div>
                            <div style={{ color: '#4ade80', fontSize: '1.1rem', fontWeight: 600 }}>1.0%</div>
                          </div>
                          <div style={{ background: 'rgba(13, 8, 17, 0.6)', padding: '12px', borderRadius: '8px' }}>
                            <div style={{ color: '#888', fontSize: '0.75rem', marginBottom: '4px' }}>MIN PAYOUT</div>
                            <div style={{ color: '#00d4ff', fontSize: '1.1rem', fontWeight: 600 }}>0.01 {network.symbol}</div>
                          </div>
                        </div>
                      </div>
                    )}
                  </div>
                ))}
              </div>
              
              <div style={{ marginTop: '24px', padding: '16px', background: 'rgba(212, 168, 75, 0.1)', borderRadius: '12px', border: '1px solid rgba(212, 168, 75, 0.3)' }}>
                <h4 style={{ color: '#D4A84B', margin: '0 0 8px', fontSize: '0.95rem' }}>💡 Multi-Network Mining</h4>
                <p style={{ color: '#B8B4C8', margin: 0, fontSize: '0.85rem' }}>
                  Enable multiple networks to automatically switch based on profitability, or assign specific equipment to specific networks in the Equipment Settings.
                </p>
              </div>
            </div>
          )}

          {/* Groups Tab */}
          {activeTab === 'groups' && (
            <div data-testid="groups-content">
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '20px' }}>
                <div>
                  <h3 style={{ color: '#D4A84B', margin: '0 0 4px' }}>Miner Groups</h3>
                  <p style={{ color: '#888', margin: 0, fontSize: '0.9rem' }}>Organize your equipment into logical groups for easier management</p>
                </div>
                <button style={eqStyles.addBtn} onClick={() => setShowAddGroupModal(true)}>➕ Create Group</button>
              </div>
              
              <div style={{ display: 'grid', gap: '12px' }}>
                {minerGroups.map(group => (
                  <div key={group.id} style={{
                    ...eqStyles.walletCard,
                    borderLeft: `4px solid ${group.color}`
                  }}>
                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', flexWrap: 'wrap', gap: '12px' }}>
                      <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                        <div style={{ 
                          width: '40px', height: '40px', borderRadius: '10px', 
                          backgroundColor: group.color + '30',
                          display: 'flex', alignItems: 'center', justifyContent: 'center',
                          color: group.color, fontSize: '1.2rem', fontWeight: 700
                        }}>
                          {group.name.charAt(0).toUpperCase()}
                        </div>
                        <div>
                          <h4 style={{ color: '#e0e0e0', margin: '0 0 4px', fontSize: '1rem' }}>{group.name}</h4>
                          {group.description && <p style={{ color: '#888', margin: 0, fontSize: '0.85rem' }}>{group.description}</p>}
                        </div>
                      </div>
                      <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                        <span style={{ color: '#888', fontSize: '0.9rem' }}>
                          {equipment.length} equipment
                        </span>
                        {group.id !== 'default' && (
                          <button 
                            style={{...eqStyles.smallBtn, borderColor: '#ef4444', color: '#ef4444'}}
                            onClick={() => {
                              setMinerGroups(minerGroups.filter(g => g.id !== group.id));
                              showMessage('success', `Group "${group.name}" deleted`);
                            }}
                          >
                            Delete
                          </button>
                        )}
                      </div>
                    </div>
                  </div>
                ))}
              </div>
              
              <div style={{ marginTop: '24px', padding: '16px', background: 'rgba(123, 94, 167, 0.1)', borderRadius: '12px', border: '1px solid rgba(123, 94, 167, 0.3)' }}>
                <h4 style={{ color: '#7B5EA7', margin: '0 0 8px', fontSize: '0.95rem' }}>📁 Group Benefits</h4>
                <ul style={{ color: '#B8B4C8', margin: 0, paddingLeft: '20px', fontSize: '0.85rem' }}>
                  <li>Apply settings to multiple equipment at once</li>
                  <li>View aggregated statistics per group</li>
                  <li>Set group-specific wallet allocations</li>
                  <li>Schedule mining times per group</li>
                </ul>
              </div>
            </div>
          )}

          {/* Alerts Tab */}
          {activeTab === 'alerts' && (
            <div style={{ textAlign: 'center', padding: '60px', backgroundColor: '#0a0a15', borderRadius: '8px', border: '1px solid #2a2a4a' }}>
              <span style={{ fontSize: '3rem' }}>🔔</span>
              <h3 style={{ color: '#e0e0e0', margin: '20px 0 10px' }}>No Active Alerts</h3>
              <p style={{ color: '#888' }}>You'll be notified here when equipment goes offline or experiences issues.</p>
            </div>
          )}
        </>
      )}

      {/* Add Wallet Modal */}
      {showAddWalletModal && (
        <div style={eqStyles.modalOverlay} onClick={() => setShowAddWalletModal(false)}>
          <div style={eqStyles.modal} onClick={e => e.stopPropagation()}>
            <h2 style={{ color: '#00d4ff', marginTop: 0 }}>➕ Add New Wallet</h2>
            <div style={{ marginBottom: '15px' }}>
              <label style={eqStyles.label}>Wallet Address *</label>
              <input
                style={eqStyles.input}
                type="text"
                placeholder="bdag1..."
                value={newWallet.address}
                onChange={e => setNewWallet({...newWallet, address: e.target.value})}
              />
            </div>
            <div style={{ marginBottom: '15px' }}>
              <label style={eqStyles.label}>Label (optional)</label>
              <input
                style={eqStyles.input}
                type="text"
                placeholder="e.g., Cold Storage, Trading"
                value={newWallet.label}
                onChange={e => setNewWallet({...newWallet, label: e.target.value})}
              />
            </div>
            <div style={{ marginBottom: '20px' }}>
              <label style={{ display: 'flex', alignItems: 'center', gap: '10px', color: '#888', cursor: 'pointer' }}>
                <input
                  type="checkbox"
                  checked={newWallet.is_primary}
                  onChange={e => setNewWallet({...newWallet, is_primary: e.target.checked})}
                />
                Set as primary wallet
              </label>
            </div>
            <div style={{ display: 'flex', gap: '10px', justifyContent: 'flex-end' }}>
              <button style={eqStyles.cancelBtn} onClick={() => setShowAddWalletModal(false)}>Cancel</button>
              <button style={eqStyles.saveBtn} onClick={() => { showMessage('success', 'Wallet added successfully'); setShowAddWalletModal(false); }}>Add Wallet</button>
            </div>
          </div>
        </div>
      )}

      {/* Edit Payout Splits Modal */}
      {editingSplits && (
        <div style={eqStyles.modalOverlay} onClick={() => setEditingSplits(null)}>
          <div style={{...eqStyles.modal, maxWidth: '500px'}} onClick={e => e.stopPropagation()}>
            <h2 style={{ color: '#00d4ff', marginTop: 0 }}>💼 Configure Payout Splits</h2>
            <p style={{ color: '#888', marginBottom: '20px' }}>Split earnings from this equipment to multiple wallets. Total must equal 100%.</p>
            
            {editingSplits.splits.map((split, idx) => (
              <div key={split.id} style={{ display: 'flex', gap: '8px', marginBottom: '10px', alignItems: 'center', flexWrap: 'wrap' }}>
                <input
                  style={{...eqStyles.input, flex: '1 1 150px', marginBottom: 0, minWidth: 0}}
                  type="text"
                  placeholder="Wallet address"
                  value={split.wallet_address}
                  onChange={e => {
                    const newSplits = [...editingSplits.splits];
                    newSplits[idx].wallet_address = e.target.value;
                    setEditingSplits({...editingSplits, splits: newSplits});
                  }}
                />
                <div style={{ display: 'flex', alignItems: 'center', gap: '4px' }}>
                  <input
                    style={{...eqStyles.input, width: '60px', marginBottom: 0, textAlign: 'center'}}
                    type="number"
                    min="1"
                    max="100"
                    value={split.percentage}
                    onChange={e => {
                      const newSplits = [...editingSplits.splits];
                      newSplits[idx].percentage = parseInt(e.target.value) || 0;
                      setEditingSplits({...editingSplits, splits: newSplits});
                    }}
                  />
                  <span style={{ color: '#888', fontSize: '0.9rem' }}>%</span>
                  {editingSplits.splits.length > 1 && (
                    <button
                      style={{ padding: '4px 8px', backgroundColor: 'transparent', border: 'none', color: '#ef4444', cursor: 'pointer', fontSize: '1rem' }}
                      onClick={() => {
                        const newSplits = editingSplits.splits.filter((_, i) => i !== idx);
                        setEditingSplits({...editingSplits, splits: newSplits});
                      }}
                    >
                      ✕
                    </button>
                  )}
                </div>
              </div>
            ))}

            <button
              style={{ ...eqStyles.smallBtn, marginTop: '10px', marginBottom: '20px' }}
              onClick={() => {
                setEditingSplits({
                  ...editingSplits,
                  splits: [...editingSplits.splits, { id: `new-${Date.now()}`, wallet_address: '', percentage: 0, label: '', is_active: true }]
                });
              }}
            >
              ➕ Add Split
            </button>

            <div style={{ backgroundColor: '#0a0a15', padding: '10px 15px', borderRadius: '6px', marginBottom: '20px' }}>
              <span style={{ color: '#888' }}>Total: </span>
              <span style={{ color: editingSplits.splits.reduce((sum, s) => sum + s.percentage, 0) === 100 ? '#4ade80' : '#ef4444', fontWeight: 'bold' }}>
                {editingSplits.splits.reduce((sum, s) => sum + s.percentage, 0)}%
              </span>
              {editingSplits.splits.reduce((sum, s) => sum + s.percentage, 0) !== 100 && (
                <span style={{ color: '#ef4444', marginLeft: '10px' }}>(Must equal 100%)</span>
              )}
            </div>

            <div style={{ display: 'flex', gap: '10px', justifyContent: 'flex-end' }}>
              <button style={eqStyles.cancelBtn} onClick={() => setEditingSplits(null)}>Cancel</button>
              <button 
                style={{...eqStyles.saveBtn, opacity: editingSplits.splits.reduce((sum, s) => sum + s.percentage, 0) !== 100 ? 0.5 : 1}}
                disabled={editingSplits.splits.reduce((sum, s) => sum + s.percentage, 0) !== 100}
                onClick={() => { showMessage('success', 'Payout splits updated'); setEditingSplits(null); }}
              >
                Save Splits
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Equipment Settings Modal - Elite Tabbed Interface */}
      {showSettingsModal && equipmentSettings && (
        <div style={eqStyles.modalOverlay} onClick={() => setShowSettingsModal(null)}>
          <div style={{...eqStyles.modal, maxWidth: '600px'}} onClick={e => e.stopPropagation()}>
            <h2 style={{ color: '#D4A84B', marginTop: 0, fontSize: '1.3rem' }}>⚙️ Equipment Settings</h2>
            <p style={{ color: '#888', marginBottom: '15px', fontSize: '0.9rem' }}>Configure {showSettingsModal.name}</p>
            
            {/* Settings Tabs */}
            <div style={{ display: 'flex', gap: '4px', marginBottom: '20px', borderBottom: '2px solid #2a2a4a', paddingBottom: '8px' }}>
              {(['general', 'network', 'wallet', 'schedule'] as const).map(tab => (
                <button
                  key={tab}
                  style={{
                    padding: '8px 16px', backgroundColor: settingsTab === tab ? 'rgba(212, 168, 75, 0.2)' : 'transparent',
                    border: 'none', borderBottom: settingsTab === tab ? '2px solid #D4A84B' : '2px solid transparent',
                    color: settingsTab === tab ? '#D4A84B' : '#888', cursor: 'pointer', fontSize: '0.85rem', fontWeight: 500,
                    borderRadius: '6px 6px 0 0', transition: 'all 0.2s'
                  }}
                  onClick={() => setSettingsTab(tab)}
                >
                  {tab === 'general' && '🔧 General'}
                  {tab === 'network' && '🌐 Network'}
                  {tab === 'wallet' && '💼 Wallet'}
                  {tab === 'schedule' && '📅 Schedule'}
                </button>
              ))}
            </div>
            
            {/* General Tab */}
            {settingsTab === 'general' && (
              <div style={{ display: 'grid', gap: '12px' }}>
                <div>
                  <label style={eqStyles.label}>Equipment Name</label>
                  <input style={eqStyles.input} type="text" value={equipmentSettings.name}
                    onChange={e => setEquipmentSettings({...equipmentSettings, name: e.target.value})} />
                </div>
                <div>
                  <label style={eqStyles.label}>Worker Name</label>
                  <input style={eqStyles.input} type="text" value={equipmentSettings.worker_name}
                    onChange={e => setEquipmentSettings({...equipmentSettings, worker_name: e.target.value})} />
                </div>
                <div>
                  <label style={eqStyles.label}>Miner Group</label>
                  <select style={{...eqStyles.input, cursor: 'pointer'}} value={equipmentSettings.miner_group_id}
                    onChange={e => setEquipmentSettings({...equipmentSettings, miner_group_id: e.target.value})}>
                    {minerGroups.map(g => <option key={g.id} value={g.id}>{g.name}</option>)}
                  </select>
                </div>
                <div>
                  <label style={eqStyles.label}>Power Mode</label>
                  <select style={{...eqStyles.input, cursor: 'pointer'}} value={equipmentSettings.power_mode}
                    onChange={e => setEquipmentSettings({...equipmentSettings, power_mode: e.target.value as any})}>
                    <option value="performance">🚀 Performance (Max Hashrate)</option>
                    <option value="balanced">⚖️ Balanced (Recommended)</option>
                    <option value="efficiency">🌱 Efficiency (Lower Power)</option>
                  </select>
                </div>
                <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '15px' }}>
                  <div>
                    <label style={eqStyles.label}>Power Limit (W)</label>
                    <input style={eqStyles.input} type="number" value={equipmentSettings.power_limit}
                      onChange={e => setEquipmentSettings({...equipmentSettings, power_limit: parseInt(e.target.value) || 0})} />
                  </div>
                  <div>
                    <label style={eqStyles.label}>Target Temp (°C)</label>
                    <input style={eqStyles.input} type="number" value={equipmentSettings.target_temperature}
                      onChange={e => setEquipmentSettings({...equipmentSettings, target_temperature: parseInt(e.target.value) || 70})} />
                  </div>
                </div>
                <div>
                  <label style={eqStyles.label}>Difficulty Mode</label>
                  <select style={{...eqStyles.input, cursor: 'pointer'}} value={equipmentSettings.difficulty_mode}
                    onChange={e => setEquipmentSettings({...equipmentSettings, difficulty_mode: e.target.value as 'auto' | 'fixed'})}>
                    <option value="auto">Auto (Recommended)</option>
                    <option value="fixed">Fixed Difficulty</option>
                  </select>
                </div>
                {equipmentSettings.difficulty_mode === 'fixed' && (
                  <div>
                    <label style={eqStyles.label}>Fixed Difficulty</label>
                    <input style={eqStyles.input} type="number" value={equipmentSettings.fixed_difficulty || ''}
                      onChange={e => setEquipmentSettings({...equipmentSettings, fixed_difficulty: parseInt(e.target.value) || undefined})}
                      placeholder="e.g., 1000000" />
                  </div>
                )}
                <div style={{ borderTop: '1px solid #2a2a4a', paddingTop: '15px', marginTop: '5px' }}>
                  <h4 style={{ color: '#00d4ff', margin: '0 0 15px', fontSize: '0.95rem' }}>🔔 Notifications</h4>
                  <label style={{ display: 'flex', alignItems: 'center', gap: '10px', color: '#888', cursor: 'pointer', marginBottom: '10px' }}>
                    <input type="checkbox" checked={equipmentSettings.auto_restart}
                      onChange={e => setEquipmentSettings({...equipmentSettings, auto_restart: e.target.checked})} />
                    Auto-restart on error
                  </label>
                  <label style={{ display: 'flex', alignItems: 'center', gap: '10px', color: '#888', cursor: 'pointer', marginBottom: '10px' }}>
                    <input type="checkbox" checked={equipmentSettings.notification_email}
                      onChange={e => setEquipmentSettings({...equipmentSettings, notification_email: e.target.checked})} />
                    Email notifications when offline
                  </label>
                  <div style={{ display: 'flex', alignItems: 'center', gap: '10px', color: '#888' }}>
                    <span>Alert after</span>
                    <input style={{...eqStyles.input, width: '80px', marginBottom: 0, textAlign: 'center'}} type="number" min="1" max="60"
                      value={equipmentSettings.notification_offline_threshold}
                      onChange={e => setEquipmentSettings({...equipmentSettings, notification_offline_threshold: parseInt(e.target.value) || 5})} />
                    <span>minutes offline</span>
                  </div>
                </div>
              </div>
            )}

            {/* Network Tab */}
            {settingsTab === 'network' && (
              <div style={{ display: 'grid', gap: '16px' }}>
                <div>
                  <label style={eqStyles.label}>Mining Network</label>
                  <p style={{ color: '#666', fontSize: '0.8rem', margin: '0 0 8px' }}>Select which network this equipment should mine on</p>
                  <div style={{ display: 'grid', gap: '8px' }}>
                    {networks.filter(n => n.is_active).map(network => (
                      <label key={network.id} style={{
                        display: 'flex', alignItems: 'center', gap: '12px', padding: '12px 16px',
                        background: equipmentSettings.network_id === network.id ? 'rgba(212, 168, 75, 0.15)' : 'rgba(13, 8, 17, 0.6)',
                        border: equipmentSettings.network_id === network.id ? '1px solid #D4A84B' : '1px solid rgba(74, 44, 90, 0.3)',
                        borderRadius: '10px', cursor: 'pointer', transition: 'all 0.2s'
                      }}>
                        <input type="radio" name="network" value={network.id} checked={equipmentSettings.network_id === network.id}
                          onChange={e => setEquipmentSettings({...equipmentSettings, network_id: e.target.value})}
                          style={{ accentColor: '#D4A84B' }} />
                        <div style={{ flex: 1 }}>
                          <div style={{ color: '#e0e0e0', fontWeight: 500 }}>{network.name} ({network.symbol})</div>
                          <div style={{ color: '#888', fontSize: '0.8rem' }}>Algorithm: {network.algorithm}</div>
                        </div>
                        {network.symbol === 'LTC' ? '🪙' : network.symbol === 'BDAG' ? '⚡' : '🔷'}
                      </label>
                    ))}
                  </div>
                </div>
                <div style={{ padding: '12px', background: 'rgba(74, 222, 128, 0.1)', borderRadius: '8px', border: '1px solid rgba(74, 222, 128, 0.3)' }}>
                  <div style={{ color: '#4ade80', fontWeight: 500, marginBottom: '4px' }}>💡 Auto-Switch Available</div>
                  <p style={{ color: '#888', fontSize: '0.85rem', margin: 0 }}>
                    Enable "Auto-Switch" in the Networks tab to automatically switch to the most profitable network.
                  </p>
                </div>
              </div>
            )}

            {/* Wallet Tab */}
            {settingsTab === 'wallet' && (
              <div style={{ display: 'grid', gap: '16px' }}>
                <div>
                  <label style={eqStyles.label}>Payout Wallet Assignment</label>
                  <p style={{ color: '#666', fontSize: '0.8rem', margin: '0 0 8px' }}>Choose where this equipment's earnings go</p>
                  <div style={{ display: 'grid', gap: '8px' }}>
                    {wallets.length > 0 ? wallets.map(wallet => (
                      <label key={wallet.id} style={{
                        display: 'flex', alignItems: 'center', gap: '12px', padding: '12px 16px',
                        background: 'rgba(13, 8, 17, 0.6)', border: '1px solid rgba(74, 44, 90, 0.3)',
                        borderRadius: '10px', cursor: 'pointer'
                      }}>
                        <input type="checkbox" defaultChecked={wallet.is_primary} style={{ accentColor: '#D4A84B' }} />
                        <div style={{ flex: 1 }}>
                          <div style={{ color: '#e0e0e0', fontWeight: 500 }}>{wallet.label || 'Unnamed Wallet'}</div>
                          <code style={{ color: '#00d4ff', fontSize: '0.75rem' }}>{wallet.wallet_address.slice(0, 20)}...</code>
                        </div>
                        <input type="number" min="0" max="100" defaultValue={100} style={{
                          width: '60px', padding: '6px', background: 'rgba(13, 8, 17, 0.8)', border: '1px solid rgba(74, 44, 90, 0.5)',
                          borderRadius: '6px', color: '#D4A84B', textAlign: 'center', fontSize: '0.9rem'
                        }} />
                        <span style={{ color: '#888' }}>%</span>
                      </label>
                    )) : (
                      <div style={{ textAlign: 'center', padding: '20px', color: '#888' }}>
                        No wallets configured. Add wallets in the Wallets tab.
                      </div>
                    )}
                  </div>
                </div>
                <button style={{...eqStyles.smallBtn, width: '100%'}} onClick={() => { setShowSettingsModal(null); setActiveTab('wallets'); }}>
                  ➕ Manage Wallets
                </button>
              </div>
            )}

            {/* Schedule Tab */}
            {settingsTab === 'schedule' && equipmentSettings.schedule && (
              <div style={{ display: 'grid', gap: '16px' }}>
                <label style={{ display: 'flex', alignItems: 'center', gap: '10px', color: '#e0e0e0', cursor: 'pointer' }}>
                  <input type="checkbox" checked={equipmentSettings.schedule_enabled}
                    onChange={e => setEquipmentSettings({...equipmentSettings, schedule_enabled: e.target.checked})}
                    style={{ accentColor: '#D4A84B' }} />
                  <span style={{ fontWeight: 500 }}>Enable Mining Schedule</span>
                </label>
                {equipmentSettings.schedule_enabled && (
                  <div style={{ display: 'grid', gap: '8px' }}>
                    {(['monday', 'tuesday', 'wednesday', 'thursday', 'friday', 'saturday', 'sunday'] as const).map(day => (
                      <div key={day} style={{
                        display: 'flex', alignItems: 'center', gap: '12px', padding: '10px 12px',
                        background: 'rgba(13, 8, 17, 0.6)', borderRadius: '8px', border: '1px solid rgba(74, 44, 90, 0.3)'
                      }}>
                        <input type="checkbox" checked={equipmentSettings.schedule![day].enabled}
                          onChange={e => setEquipmentSettings({
                            ...equipmentSettings,
                            schedule: {...equipmentSettings.schedule!, [day]: {...equipmentSettings.schedule![day], enabled: e.target.checked}}
                          })}
                          style={{ accentColor: '#D4A84B' }} />
                        <span style={{ width: '80px', color: equipmentSettings.schedule![day].enabled ? '#e0e0e0' : '#666', textTransform: 'capitalize' }}>
                          {day}
                        </span>
                        <input type="time" value={equipmentSettings.schedule![day].start}
                          disabled={!equipmentSettings.schedule![day].enabled}
                          onChange={e => setEquipmentSettings({
                            ...equipmentSettings,
                            schedule: {...equipmentSettings.schedule!, [day]: {...equipmentSettings.schedule![day], start: e.target.value}}
                          })}
                          style={{ padding: '4px 8px', background: 'rgba(13, 8, 17, 0.8)', border: '1px solid rgba(74, 44, 90, 0.5)',
                            borderRadius: '6px', color: '#e0e0e0', fontSize: '0.85rem' }} />
                        <span style={{ color: '#666' }}>to</span>
                        <input type="time" value={equipmentSettings.schedule![day].end}
                          disabled={!equipmentSettings.schedule![day].enabled}
                          onChange={e => setEquipmentSettings({
                            ...equipmentSettings,
                            schedule: {...equipmentSettings.schedule!, [day]: {...equipmentSettings.schedule![day], end: e.target.value}}
                          })}
                          style={{ padding: '4px 8px', background: 'rgba(13, 8, 17, 0.8)', border: '1px solid rgba(74, 44, 90, 0.5)',
                            borderRadius: '6px', color: '#e0e0e0', fontSize: '0.85rem' }} />
                      </div>
                    ))}
                  </div>
                )}
                <div style={{ padding: '12px', background: 'rgba(212, 168, 75, 0.1)', borderRadius: '8px', border: '1px solid rgba(212, 168, 75, 0.3)' }}>
                  <div style={{ color: '#D4A84B', fontWeight: 500, marginBottom: '4px' }}>⚡ Power Savings</div>
                  <p style={{ color: '#888', fontSize: '0.85rem', margin: 0 }}>
                    Schedule mining during off-peak electricity hours to reduce costs. Equipment will automatically pause outside scheduled times.
                  </p>
                </div>
              </div>
            )}

            <div style={{ display: 'flex', gap: '10px', justifyContent: 'flex-end', marginTop: '20px' }}>
              <button style={eqStyles.cancelBtn} onClick={() => setShowSettingsModal(null)}>Cancel</button>
              <button style={eqStyles.saveBtn} onClick={() => { showMessage('success', 'Settings saved'); setShowSettingsModal(null); }}>Save Settings</button>
            </div>
          </div>
        </div>
      )}

      {/* Equipment Charts Modal */}
      {showChartsModal && (
        <div style={eqStyles.modalOverlay} onClick={() => setShowChartsModal(null)}>
          <div style={{...eqStyles.modal, maxWidth: '850px', padding: '15px'}} onClick={e => e.stopPropagation()}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: '15px', flexWrap: 'wrap', gap: '10px' }}>
              <div>
                <h2 style={{ color: '#00d4ff', margin: 0, fontSize: '1.2rem' }}>📊 {showChartsModal.name}</h2>
                <p style={{ color: '#888', margin: '3px 0 0', fontSize: '0.85rem' }}>{showChartsModal.model} • {showChartsModal.worker_name}</p>
              </div>
              <button style={{ background: 'none', border: 'none', color: '#888', fontSize: '1.3rem', cursor: 'pointer', padding: '0' }} onClick={() => setShowChartsModal(null)}>✕</button>
            </div>

            {/* Time Range Selector */}
            <div style={{ display: 'flex', gap: '6px', marginBottom: '15px', flexWrap: 'wrap' }}>
              {(['1h', '6h', '24h', '7d', '30d'] as const).map(range => (
                <button
                  key={range}
                  style={{
                    padding: '6px 12px',
                    backgroundColor: chartTimeRange === range ? '#00d4ff' : '#0a0a15',
                    border: '1px solid #2a2a4a',
                    borderRadius: '6px',
                    color: chartTimeRange === range ? '#0a0a0f' : '#888',
                    cursor: 'pointer',
                    fontWeight: chartTimeRange === range ? 'bold' : 'normal',
                    fontSize: '0.85rem'
                  }}
                  onClick={() => setChartTimeRange(range)}
                >
                  {range.toUpperCase()}
                </button>
              ))}
            </div>

            {/* Charts Grid */}
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(280px, 1fr))', gap: '12px' }}>
              {/* Hashrate Chart */}
              <div style={{ backgroundColor: '#0a0a15', borderRadius: '10px', padding: '20px', border: '1px solid #2a2a4a' }}>
                <h3 style={{ color: '#e0e0e0', margin: '0 0 15px', fontSize: '1rem' }}>⚡ Hashrate History</h3>
                <ResponsiveContainer width="100%" height={200}>
                  <AreaChart data={generateChartData(showChartsModal, chartTimeRange)}>
                    <defs>
                      <linearGradient id="eqHashGradient" x1="0" y1="0" x2="0" y2="1">
                        <stop offset="5%" stopColor="#00d4ff" stopOpacity={0.4}/>
                        <stop offset="95%" stopColor="#00d4ff" stopOpacity={0}/>
                      </linearGradient>
                    </defs>
                    <CartesianGrid strokeDasharray="3 3" stroke="#2a2a4a" />
                    <XAxis dataKey="time" stroke="#666" tick={{ fontSize: 10 }} />
                    <YAxis stroke="#666" tick={{ fontSize: 10 }} tickFormatter={(v) => formatHashrate(v)} />
                    <Tooltip 
                      contentStyle={{ backgroundColor: '#1a1a2e', border: '1px solid #2a2a4a' }}
                      formatter={(value: number) => [formatHashrate(value), 'Hashrate']}
                    />
                    <Area type="monotone" dataKey="hashrate" stroke="#00d4ff" fill="url(#eqHashGradient)" strokeWidth={2} />
                  </AreaChart>
                </ResponsiveContainer>
              </div>

              {/* Temperature Chart */}
              <div style={{ backgroundColor: '#0a0a15', borderRadius: '10px', padding: '20px', border: '1px solid #2a2a4a' }}>
                <h3 style={{ color: '#e0e0e0', margin: '0 0 15px', fontSize: '1rem' }}>🌡️ Temperature History</h3>
                <ResponsiveContainer width="100%" height={200}>
                  <LineChart data={generateChartData(showChartsModal, chartTimeRange)}>
                    <CartesianGrid strokeDasharray="3 3" stroke="#2a2a4a" />
                    <XAxis dataKey="time" stroke="#666" tick={{ fontSize: 10 }} />
                    <YAxis stroke="#666" tick={{ fontSize: 10 }} domain={[40, 90]} tickFormatter={(v) => `${v}°C`} />
                    <Tooltip 
                      contentStyle={{ backgroundColor: '#1a1a2e', border: '1px solid #2a2a4a' }}
                      formatter={(value: number) => [`${value.toFixed(1)}°C`, 'Temperature']}
                    />
                    <Line type="monotone" dataKey="temperature" stroke="#ef4444" strokeWidth={2} dot={false} />
                    <ReferenceLine y={80} stroke="#f59e0b" strokeDasharray="5 5" label={{ value: 'Warning', fill: '#f59e0b', fontSize: 10 }} />
                  </LineChart>
                </ResponsiveContainer>
              </div>

              {/* Power Usage Chart */}
              <div style={{ backgroundColor: '#0a0a15', borderRadius: '10px', padding: '20px', border: '1px solid #2a2a4a' }}>
                <h3 style={{ color: '#e0e0e0', margin: '0 0 15px', fontSize: '1rem' }}>⚡ Power Consumption</h3>
                <ResponsiveContainer width="100%" height={200}>
                  <AreaChart data={generateChartData(showChartsModal, chartTimeRange)}>
                    <defs>
                      <linearGradient id="eqPowerGradient" x1="0" y1="0" x2="0" y2="1">
                        <stop offset="5%" stopColor="#f59e0b" stopOpacity={0.4}/>
                        <stop offset="95%" stopColor="#f59e0b" stopOpacity={0}/>
                      </linearGradient>
                    </defs>
                    <CartesianGrid strokeDasharray="3 3" stroke="#2a2a4a" />
                    <XAxis dataKey="time" stroke="#666" tick={{ fontSize: 10 }} />
                    <YAxis stroke="#666" tick={{ fontSize: 10 }} tickFormatter={(v) => `${v}W`} />
                    <Tooltip 
                      contentStyle={{ backgroundColor: '#1a1a2e', border: '1px solid #2a2a4a' }}
                      formatter={(value: number) => [`${value.toFixed(0)}W`, 'Power']}
                    />
                    <Area type="monotone" dataKey="power" stroke="#f59e0b" fill="url(#eqPowerGradient)" strokeWidth={2} />
                  </AreaChart>
                </ResponsiveContainer>
              </div>

              {/* Summary Stats */}
              <div style={{ backgroundColor: '#0a0a15', borderRadius: '10px', padding: '20px', border: '1px solid #2a2a4a' }}>
                <h3 style={{ color: '#e0e0e0', margin: '0 0 15px', fontSize: '1rem' }}>📈 Summary ({chartTimeRange.toUpperCase()})</h3>
                <div style={{ display: 'grid', gap: '12px' }}>
                  <div style={{ display: 'flex', justifyContent: 'space-between', color: '#888' }}>
                    <span>Current Hashrate:</span>
                    <span style={{ color: '#00d4ff', fontWeight: 'bold' }}>{formatHashrate(showChartsModal.current_hashrate)}</span>
                  </div>
                  <div style={{ display: 'flex', justifyContent: 'space-between', color: '#888' }}>
                    <span>Average Hashrate:</span>
                    <span style={{ color: '#e0e0e0' }}>{formatHashrate(showChartsModal.average_hashrate)}</span>
                  </div>
                  <div style={{ display: 'flex', justifyContent: 'space-between', color: '#888' }}>
                    <span>Current Temp:</span>
                    <span style={{ color: showChartsModal.temperature > 80 ? '#ef4444' : '#4ade80' }}>{showChartsModal.temperature > 0 ? `${showChartsModal.temperature}°C` : 'N/A'}</span>
                  </div>
                  <div style={{ display: 'flex', justifyContent: 'space-between', color: '#888' }}>
                    <span>Power Usage:</span>
                    <span style={{ color: '#f59e0b' }}>{showChartsModal.power_usage > 0 ? `${showChartsModal.power_usage}W` : 'N/A'}</span>
                  </div>
                  <div style={{ display: 'flex', justifyContent: 'space-between', color: '#888' }}>
                    <span>Efficiency:</span>
                    <span style={{ color: '#9b59b6' }}>{showChartsModal.power_usage > 0 ? `${(showChartsModal.current_hashrate / showChartsModal.power_usage / 1000).toFixed(2)} MH/W` : 'N/A'}</span>
                  </div>
                  <div style={{ display: 'flex', justifyContent: 'space-between', color: '#888' }}>
                    <span>Uptime:</span>
                    <span style={{ color: '#4ade80', fontWeight: 'bold' }}>{formatUptimePercent(showChartsModal)}</span>
                  </div>
                </div>
              </div>
            </div>

            <div style={{ marginTop: '20px', textAlign: 'center' }}>
              <p style={{ color: '#666', fontSize: '0.85rem' }}>Charts update automatically when equipment is connected</p>
            </div>
          </div>
        </div>
      )}

      {/* Add Group Modal */}
      {showAddGroupModal && (
        <div style={eqStyles.modalOverlay} onClick={() => setShowAddGroupModal(false)}>
          <div style={eqStyles.modal} onClick={e => e.stopPropagation()}>
            <h2 style={{ color: '#D4A84B', marginTop: 0 }}>📁 Create Miner Group</h2>
            <p style={{ color: '#888', marginBottom: '20px', fontSize: '0.9rem' }}>Organize your equipment into logical groups</p>
            
            <div style={{ marginBottom: '15px' }}>
              <label style={eqStyles.label}>Group Name *</label>
              <input
                style={eqStyles.input}
                type="text"
                placeholder="e.g., Basement Rigs, Office ASICs"
                value={newGroup.name}
                onChange={e => setNewGroup({...newGroup, name: e.target.value})}
              />
            </div>
            
            <div style={{ marginBottom: '15px' }}>
              <label style={eqStyles.label}>Description (optional)</label>
              <input
                style={eqStyles.input}
                type="text"
                placeholder="Brief description of this group"
                value={newGroup.description}
                onChange={e => setNewGroup({...newGroup, description: e.target.value})}
              />
            </div>
            
            <div style={{ marginBottom: '20px' }}>
              <label style={eqStyles.label}>Group Color</label>
              <div style={{ display: 'flex', gap: '8px', flexWrap: 'wrap' }}>
                {['#D4A84B', '#00d4ff', '#4ade80', '#ef4444', '#f59e0b', '#7B5EA7', '#ec4899', '#06b6d4'].map(color => (
                  <button
                    key={color}
                    style={{
                      width: '36px', height: '36px', borderRadius: '8px', backgroundColor: color,
                      border: newGroup.color === color ? '3px solid #fff' : '2px solid transparent',
                      cursor: 'pointer', transition: 'all 0.2s'
                    }}
                    onClick={() => setNewGroup({...newGroup, color})}
                  />
                ))}
              </div>
            </div>
            
            <div style={{ display: 'flex', gap: '10px', justifyContent: 'flex-end' }}>
              <button style={eqStyles.cancelBtn} onClick={() => setShowAddGroupModal(false)}>Cancel</button>
              <button 
                style={{...eqStyles.saveBtn, opacity: !newGroup.name ? 0.5 : 1}}
                disabled={!newGroup.name}
                onClick={() => {
                  setMinerGroups([...minerGroups, { id: `group-${Date.now()}`, ...newGroup }]);
                  showMessage('success', `Group "${newGroup.name}" created`);
                  setNewGroup({ name: '', color: '#D4A84B', description: '' });
                  setShowAddGroupModal(false);
                }}
              >
                Create Group
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

const eqStyles: { [key: string]: React.CSSProperties } = {
  // Phase 3 Enhanced Stat Cards with live indicators
  statCard: { 
    background: 'linear-gradient(180deg, rgba(45, 31, 61, 0.8) 0%, rgba(26, 15, 30, 0.9) 100%)', 
    padding: '20px', 
    borderRadius: '14px', 
    border: '1px solid rgba(74, 44, 90, 0.5)', 
    textAlign: 'center', 
    boxShadow: '0 4px 24px rgba(0, 0, 0, 0.3)', 
    transition: 'all 0.25s ease',
    position: 'relative',
    overflow: 'hidden'
  },
  statIcon: { fontSize: '1.5rem', marginBottom: '10px', filter: 'drop-shadow(0 2px 4px rgba(0,0,0,0.3))' },
  statValue: { fontSize: '1.5rem', fontWeight: 700, color: '#D4A84B', marginBottom: '6px', textShadow: '0 2px 4px rgba(0, 0, 0, 0.3)', letterSpacing: '-0.02em' },
  statLabel: { color: '#B8B4C8', fontSize: '0.75rem', textTransform: 'uppercase', letterSpacing: '0.08em', fontWeight: 500 },
  
  // Phase 3 Enhanced Tab Navigation
  tab: { 
    padding: '14px 24px', 
    backgroundColor: 'transparent', 
    border: 'none', 
    borderBottom: '3px solid transparent', 
    color: '#B8B4C8', 
    cursor: 'pointer', 
    fontSize: '0.95rem', 
    transition: 'all 0.25s ease', 
    fontWeight: 500,
    position: 'relative'
  },
  tabActive: { 
    color: '#D4A84B', 
    borderBottomColor: '#D4A84B',
    textShadow: '0 0 20px rgba(212, 168, 75, 0.5)'
  },
  
  // Phase 3 Enhanced Equipment Cards
  equipmentCard: { 
    background: 'linear-gradient(180deg, rgba(45, 31, 61, 0.6) 0%, rgba(26, 15, 30, 0.8) 100%)', 
    borderRadius: '16px', 
    padding: '20px', 
    border: '1px solid rgba(74, 44, 90, 0.4)', 
    borderLeft: '4px solid #7A7490', 
    cursor: 'pointer', 
    transition: 'all 0.3s ease', 
    boxShadow: '0 4px 20px rgba(0, 0, 0, 0.2)',
    position: 'relative'
  },
  equipmentHeader: { display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '16px', flexWrap: 'wrap', gap: '12px' },
  quickStats: { display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(90px, 1fr))', gap: '12px' },
  quickStat: { 
    background: 'rgba(13, 8, 17, 0.7)', 
    padding: '12px', 
    borderRadius: '10px', 
    textAlign: 'center', 
    border: '1px solid rgba(74, 44, 90, 0.3)',
    transition: 'all 0.2s ease'
  },
  quickLabel: { display: 'block', color: '#B8B4C8', fontSize: '0.7rem', marginBottom: '6px', textTransform: 'uppercase', letterSpacing: '0.05em', fontWeight: 500 },
  quickValue: { display: 'block', color: '#F0EDF4', fontWeight: 600, fontSize: '0.95rem' },
  
  // Phase 3 Enhanced Detail Sections
  expandedDetails: { marginTop: '20px', paddingTop: '20px', borderTop: '1px solid rgba(74, 44, 90, 0.4)' },
  detailSection: { 
    background: 'rgba(13, 8, 17, 0.7)', 
    padding: '16px', 
    borderRadius: '12px', 
    border: '1px solid rgba(74, 44, 90, 0.3)',
    transition: 'all 0.2s ease'
  },
  detailTitle: { color: '#D4A84B', margin: '0 0 14px', fontSize: '1rem', fontWeight: 600, display: 'flex', alignItems: 'center', gap: '8px' },
  detailRow: { display: 'flex', justifyContent: 'space-between', marginBottom: '10px', color: '#B8B4C8', fontSize: '0.85rem', flexWrap: 'wrap', gap: '8px' },
  
  // Action Buttons
  editSplitsBtn: { marginTop: '12px', padding: '10px 16px', backgroundColor: 'transparent', border: '1px solid #7B5EA7', borderRadius: '10px', color: '#7B5EA7', cursor: 'pointer', fontSize: '0.85rem', transition: 'all 0.2s' },
  actionBtn: { padding: '12px 20px', backgroundColor: 'transparent', border: '1px solid #D4A84B', borderRadius: '10px', color: '#D4A84B', cursor: 'pointer', fontSize: '0.9rem', fontWeight: 500, transition: 'all 0.25s ease' },
  
  // Wallet Cards
  walletCard: { background: 'linear-gradient(180deg, rgba(45, 31, 61, 0.6) 0%, rgba(26, 15, 30, 0.8) 100%)', padding: '20px', borderRadius: '14px', border: '1px solid rgba(74, 44, 90, 0.4)', transition: 'all 0.2s ease' },
  primaryBadge: { background: 'linear-gradient(135deg, #4ADE80 0%, #22C55E 100%)', color: '#1A0F1E', fontSize: '0.7rem', padding: '4px 10px', borderRadius: '8px', fontWeight: 700, textTransform: 'uppercase', letterSpacing: '0.03em' },
  smallBtn: { padding: '8px 14px', backgroundColor: 'transparent', border: '1px solid #D4A84B', borderRadius: '8px', color: '#D4A84B', cursor: 'pointer', fontSize: '0.85rem', transition: 'all 0.2s' },
  addBtn: { padding: '12px 24px', background: 'linear-gradient(135deg, #D4A84B 0%, #B8923A 100%)', border: 'none', borderRadius: '10px', color: '#1A0F1E', fontWeight: 600, cursor: 'pointer', fontSize: '0.95rem', boxShadow: '0 4px 12px rgba(212, 168, 75, 0.3)', transition: 'all 0.25s ease' },
  
  // Modals
  modalOverlay: { position: 'fixed', top: 0, left: 0, right: 0, bottom: 0, backgroundColor: 'rgba(13, 8, 17, 0.95)', backdropFilter: 'blur(12px)', display: 'flex', justifyContent: 'center', alignItems: 'center', zIndex: 1000, padding: '20px', boxSizing: 'border-box' },
  modal: { background: 'linear-gradient(180deg, #2D1F3D 0%, #1A0F1E 100%)', padding: '30px', borderRadius: '18px', border: '1px solid rgba(212, 168, 75, 0.3)', maxWidth: '480px', width: '100%', maxHeight: 'calc(100vh - 40px)', overflowY: 'auto', boxSizing: 'border-box', boxShadow: '0 32px 64px rgba(0, 0, 0, 0.6)' },
  label: { display: 'block', color: '#B8B4C8', marginBottom: '8px', fontSize: '0.9rem', fontWeight: 500 },
  input: { width: '100%', padding: '14px 16px', backgroundColor: 'rgba(13, 8, 17, 0.8)', border: '1px solid rgba(74, 44, 90, 0.5)', borderRadius: '12px', color: '#F0EDF4', fontSize: '1rem', marginBottom: '16px', boxSizing: 'border-box', transition: 'border-color 0.2s, box-shadow 0.2s' },
  cancelBtn: { padding: '14px 24px', backgroundColor: 'transparent', border: '1px solid rgba(74, 44, 90, 0.5)', borderRadius: '12px', color: '#B8B4C8', cursor: 'pointer', fontSize: '0.95rem', transition: 'all 0.2s' },
  saveBtn: { padding: '14px 24px', background: 'linear-gradient(135deg, #D4A84B 0%, #B8923A 100%)', border: 'none', borderRadius: '12px', color: '#1A0F1E', fontWeight: 600, cursor: 'pointer', fontSize: '0.95rem', boxShadow: '0 4px 12px rgba(212, 168, 75, 0.3)', transition: 'all 0.25s ease' },
};

export default EquipmentPage;
