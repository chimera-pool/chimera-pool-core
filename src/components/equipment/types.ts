// ============================================================================
// EQUIPMENT TYPES
// Type definitions for equipment management components
// ============================================================================

export interface Equipment {
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
  connected_at: string;
  total_connection_time: number;
  total_downtime: number;
  downtime_incidents: number;
  last_downtime_start?: string;
  last_downtime_end?: string;
}

export interface EquipmentSettings {
  id: string;
  name: string;
  worker_name: string;
  power_limit: number;
  target_temperature: number;
  auto_restart: boolean;
  notification_email: boolean;
  notification_offline_threshold: number;
  difficulty_mode: 'auto' | 'fixed';
  fixed_difficulty?: number;
}

export interface PayoutSplit {
  id: string;
  wallet_address: string;
  percentage: number;
  label: string;
  is_active: boolean;
}

export interface EquipmentWallet {
  id: string;
  wallet_address: string;
  label: string;
  is_primary: boolean;
  currency: string;
}

export interface EquipmentStats {
  totalEquipment: number;
  online: number;
  offline: number;
  errors: number;
  totalHashrate: number;
  totalEarnings: number;
  avgLatency: number;
}

export type EquipmentTab = 'equipment' | 'wallets' | 'alerts';
export type ChartTimeRange = '1h' | '6h' | '24h' | '7d' | '30d';
