// ============================================================================
// LAYOUT TYPES
// Shared types for layout components
// ============================================================================

export type MainView = 'dashboard' | 'community' | 'equipment';

export type AuthView = 'login' | 'register' | 'forgot-password' | 'reset-password';

export interface PoolStats {
  total_miners: number;
  total_hashrate: number;
  blocks_found: number;
  pool_fee: number;
  minimum_payout: number;
  payment_interval: string;
  network: string;
  currency: string;
}

export interface MessageState {
  type: 'success' | 'error';
  text: string;
}
