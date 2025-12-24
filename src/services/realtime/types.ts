/**
 * Real-time Mining Data Types
 * 
 * Follows Interface Segregation Principle (ISP):
 * - Small, focused interfaces for specific data needs
 * - Components only depend on the interfaces they actually use
 * - Easy to test and mock individual pieces
 */

// ============================================================================
// BASE DATA TYPES
// ============================================================================

export interface TimeSeriesDataPoint {
  time: string;
  timestamp: Date;
}

export interface HashrateDataPoint extends TimeSeriesDataPoint {
  hashrate: number;        // Raw hashrate in H/s
  hashrateTH: number;      // Formatted in TH/s
  hashrateMH?: number;     // Formatted in MH/s (for lower rates)
}

export interface ShareDataPoint extends TimeSeriesDataPoint {
  valid: number;
  invalid: number;
  total: number;
  acceptanceRate: number;  // Percentage 0-100
}

export interface MinerDataPoint extends TimeSeriesDataPoint {
  activeMiners: number;
  totalMiners: number;
  newMiners?: number;
}

export interface EarningsDataPoint extends TimeSeriesDataPoint {
  amount: number;
  currency: string;
}

export interface DifficultyDataPoint extends TimeSeriesDataPoint {
  difficulty: number;
  networkDifficulty?: number;
}

// ============================================================================
// MINER INFO (Individual miner telemetry)
// ============================================================================

export interface MinerInfo {
  id: string;
  name: string;
  userId: number;
  hashrate: number;
  difficulty: number;
  isActive: boolean;
  lastSeen: Date;
  sharesValid: number;
  sharesInvalid: number;
  acceptanceRate: number;
  // Optional telemetry (if available from miner)
  temperature?: number;
  fanSpeed?: number;
  powerUsage?: number;
}

// ============================================================================
// POOL STATS (Current snapshot)
// ============================================================================

export interface PoolStats {
  totalHashrate: number;
  activeMiners: number;
  totalShares: number;
  validShares: number;
  invalidShares: number;
  acceptanceRate: number;
  blocksFound: number;
  lastBlockTime?: Date;
  networkDifficulty: number;
  poolDifficulty: number;
  currency: string;
  algorithm: string;
}

// ============================================================================
// USER STATS (Personal mining stats)
// ============================================================================

export interface UserStats {
  userId: number;
  totalHashrate: number;
  pendingPayout: number;
  totalEarnings: number;
  totalShares: number;
  validShares: number;
  invalidShares: number;
  acceptanceRate: number;
  activeMiners: number;
  miners: MinerInfo[];
}

// ============================================================================
// ISP-COMPLIANT INTERFACES
// Each interface serves a specific purpose
// ============================================================================

/** For components that only need current hashrate */
export interface IHashrateProvider {
  getCurrentHashrate(): number;
  getHashrateHistory(range: TimeRange): HashrateDataPoint[];
  subscribeToHashrate(callback: (hashrate: number) => void): () => void;
}

/** For components that only need share statistics */
export interface ISharesProvider {
  getCurrentShares(): { valid: number; invalid: number; rate: number };
  getSharesHistory(range: TimeRange): ShareDataPoint[];
  subscribeToShares(callback: (shares: ShareDataPoint) => void): () => void;
}

/** For components that need miner information */
export interface IMinersProvider {
  getActiveMiners(): MinerInfo[];
  getMinerCount(): number;
  getMinersHistory(range: TimeRange): MinerDataPoint[];
  subscribeToMiners(callback: (miners: MinerInfo[]) => void): () => void;
}

/** For components that need pool overview */
export interface IPoolStatsProvider {
  getPoolStats(): PoolStats;
  subscribeToPoolStats(callback: (stats: PoolStats) => void): () => void;
}

/** For components that need user-specific data */
export interface IUserStatsProvider {
  getUserStats(): UserStats | null;
  subscribeToUserStats(callback: (stats: UserStats) => void): () => void;
}

/** Combined interface for full real-time data access */
export interface IRealTimeDataProvider extends 
  IHashrateProvider, 
  ISharesProvider, 
  IMinersProvider, 
  IPoolStatsProvider,
  IUserStatsProvider {
  // Connection state
  isConnected: boolean;
  isLoading: boolean;
  error: Error | null;
  
  // Control methods
  refresh(): Promise<void>;
  setTimeRange(range: TimeRange): void;
  getTimeRange(): TimeRange;
}

// ============================================================================
// TIME RANGE TYPES
// ============================================================================

export type TimeRange = '1h' | '6h' | '24h' | '7d' | '30d' | '3m' | '6m' | '1y' | 'all';

export const TIME_RANGE_LABELS: Record<TimeRange, string> = {
  '1h': '1 Hour',
  '6h': '6 Hours',
  '24h': '24 Hours',
  '7d': '7 Days',
  '30d': '30 Days',
  '3m': '3 Months',
  '6m': '6 Months',
  '1y': '1 Year',
  'all': 'All Time',
};

export const TIME_RANGE_MS: Record<TimeRange, number> = {
  '1h': 60 * 60 * 1000,
  '6h': 6 * 60 * 60 * 1000,
  '24h': 24 * 60 * 60 * 1000,
  '7d': 7 * 24 * 60 * 60 * 1000,
  '30d': 30 * 24 * 60 * 60 * 1000,
  '3m': 90 * 24 * 60 * 60 * 1000,
  '6m': 180 * 24 * 60 * 60 * 1000,
  '1y': 365 * 24 * 60 * 60 * 1000,
  'all': Infinity,
};

// ============================================================================
// REFRESH CONFIGURATION
// ============================================================================

export interface RefreshConfig {
  interval: number;
  enabled: boolean;
}

export const DEFAULT_REFRESH_CONFIG: RefreshConfig = {
  interval: 10000, // 10 seconds
  enabled: true,
};
