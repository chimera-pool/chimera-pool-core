// ============================================================================
// EQUIPMENT UTILITIES
// Utility functions for equipment management
// ============================================================================

import { Equipment } from './types';

export const formatHashrate = (h: number): string => {
  if (h >= 1e12) return `${(h / 1e12).toFixed(2)} TH/s`;
  if (h >= 1e9) return `${(h / 1e9).toFixed(2)} GH/s`;
  if (h >= 1e6) return `${(h / 1e6).toFixed(2)} MH/s`;
  if (h >= 1e3) return `${(h / 1e3).toFixed(2)} KH/s`;
  return `${h.toFixed(2)} H/s`;
};

export const formatUptime = (seconds: number): string => {
  const days = Math.floor(seconds / 86400);
  const hours = Math.floor((seconds % 86400) / 3600);
  if (days > 0) return `${days}d ${hours}h`;
  return `${hours}h`;
};

export const formatDuration = (seconds: number): string => {
  if (seconds < 60) return `${seconds}s`;
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m`;
  if (seconds < 86400) return `${Math.floor(seconds / 3600)}h ${Math.floor((seconds % 3600) / 60)}m`;
  const days = Math.floor(seconds / 86400);
  const hours = Math.floor((seconds % 86400) / 3600);
  return `${days}d ${hours}h`;
};

export const formatUptimePercent = (eq: Equipment): string => {
  if (eq.total_connection_time === 0) return '0%';
  const uptimeSeconds = eq.total_connection_time - eq.total_downtime;
  return ((uptimeSeconds / eq.total_connection_time) * 100).toFixed(2) + '%';
};

export const getStatusColor = (status: string): string => {
  switch (status) {
    case 'mining': return '#4ade80';
    case 'online': return '#00d4ff';
    case 'idle': return '#f59e0b';
    case 'offline': return '#888';
    case 'error': return '#ef4444';
    default: return '#888';
  }
};

export const getStatusIcon = (status: string): string => {
  switch (status) {
    case 'mining': return '⛏️';
    case 'online': return '🟢';
    case 'idle': return '💤';
    case 'offline': return '⚫';
    case 'error': return '🔴';
    default: return '❓';
  }
};

export const getTypeIcon = (type: string): string => {
  switch (type) {
    case 'blockdag_x100': return '⚡';
    case 'blockdag_x30': return '⚡';
    case 'gpu': return '🎮';
    case 'cpu': return '💻';
    case 'asic': return '🔲';
    default: return '⚙️';
  }
};

export const generateChartData = (eq: Equipment, range: string) => {
  const points = range === '1h' ? 12 : range === '6h' ? 36 : range === '24h' ? 48 : range === '7d' ? 168 : 720;
  const data = [];
  const baseHashrate = eq.average_hashrate;
  const now = Date.now();
  
  for (let i = points; i >= 0; i--) {
    const multiplier = range === '1h' ? 300000 : range === '6h' ? 600000 : range === '24h' ? 1800000 : 3600000;
    const timestamp = new Date(now - (i * multiplier));
    const variance = (Math.random() - 0.5) * 0.1 * baseHashrate;
    data.push({
      time: timestamp.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
      hashrate: Math.max(0, baseHashrate + variance),
      temperature: eq.temperature > 0 ? eq.temperature + (Math.random() - 0.5) * 5 : 0,
      power: eq.power_usage > 0 ? eq.power_usage + (Math.random() - 0.5) * 100 : 0,
    });
  }
  return data;
};

export const generateMockEquipment = (user: any): Equipment[] => {
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
      payout_splits: [{ id: 's1', wallet_address: user?.payout_address || 'bdag1...', percentage: 100, label: 'Primary', is_active: true }],
      connected_at: new Date(now - 518400000).toISOString(),
      total_connection_time: 518400,
      total_downtime: 3600,
      downtime_incidents: 1,
      last_downtime_start: new Date(now - 172800000).toISOString(),
      last_downtime_end: new Date(now - 169200000).toISOString(),
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
      payout_splits: [{ id: 's2', wallet_address: user?.payout_address || 'bdag1...', percentage: 100, label: 'Primary', is_active: true }],
      connected_at: new Date(now - 345600000).toISOString(),
      total_connection_time: 345600,
      total_downtime: 7200,
      downtime_incidents: 3,
      last_downtime_start: new Date(now - 86400000).toISOString(),
      last_downtime_end: new Date(now - 82800000).toISOString(),
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
      payout_splits: [{ id: 's3', wallet_address: user?.payout_address || 'bdag1...', percentage: 100, label: 'Primary', is_active: true }],
      connected_at: new Date(now - 604800000).toISOString(),
      total_connection_time: 604800,
      total_downtime: 86400,
      downtime_incidents: 2,
      last_downtime_start: new Date(now - 86400000).toISOString(),
      last_downtime_end: undefined,
    },
  ];
};

export const calculateEquipmentStats = (equipment: Equipment[]) => ({
  totalEquipment: equipment.length,
  online: equipment.filter(e => ['mining', 'online', 'idle'].includes(e.status)).length,
  offline: equipment.filter(e => e.status === 'offline').length,
  errors: equipment.filter(e => e.status === 'error').length,
  totalHashrate: equipment.reduce((sum, e) => sum + e.current_hashrate, 0),
  totalEarnings: equipment.reduce((sum, e) => sum + e.total_earnings, 0),
  avgLatency: equipment.filter(e => e.latency > 0).reduce((sum, e, _, arr) => sum + e.latency / arr.length, 0),
});
