/**
 * Real-Time Data Service
 * 
 * Unified service for fetching and managing real-time mining data
 * Implements ISP-compliant interfaces for flexible consumption
 */

import {
  TimeRange,
  HashrateDataPoint,
  ShareDataPoint,
  MinerDataPoint,
  MinerInfo,
  PoolStats,
  UserStats,
  IRealTimeDataProvider,
  DEFAULT_REFRESH_CONFIG,
} from './types';

type Callback<T> = (data: T) => void;

export class RealTimeDataService implements IRealTimeDataProvider {
  // State
  private _isConnected: boolean = false;
  private _isLoading: boolean = false;
  private _error: Error | null = null;
  private _timeRange: TimeRange = '24h';
  private _authToken: string | null = null;

  // Cached data
  private _poolStats: PoolStats | null = null;
  private _userStats: UserStats | null = null;
  private _hashrateHistory: HashrateDataPoint[] = [];
  private _sharesHistory: ShareDataPoint[] = [];
  private _minersHistory: MinerDataPoint[] = [];
  private _activeMiners: MinerInfo[] = [];

  // Subscribers
  private _hashrateSubscribers: Set<Callback<number>> = new Set();
  private _sharesSubscribers: Set<Callback<ShareDataPoint>> = new Set();
  private _minersSubscribers: Set<Callback<MinerInfo[]>> = new Set();
  private _poolStatsSubscribers: Set<Callback<PoolStats>> = new Set();
  private _userStatsSubscribers: Set<Callback<UserStats>> = new Set();

  // Auto-refresh
  private _refreshInterval: NodeJS.Timeout | null = null;

  // ============================================================================
  // GETTERS
  // ============================================================================

  get isConnected(): boolean {
    return this._isConnected;
  }

  get isLoading(): boolean {
    return this._isLoading;
  }

  get error(): Error | null {
    return this._error;
  }

  // ============================================================================
  // IHashrateProvider Implementation
  // ============================================================================

  getCurrentHashrate(): number {
    return this._poolStats?.totalHashrate ?? 0;
  }

  getHashrateHistory(range: TimeRange): HashrateDataPoint[] {
    return this._hashrateHistory;
  }

  subscribeToHashrate(callback: Callback<number>): () => void {
    this._hashrateSubscribers.add(callback);
    return () => this._hashrateSubscribers.delete(callback);
  }

  async fetchHashrateHistory(range: TimeRange): Promise<HashrateDataPoint[]> {
    try {
      const response = await fetch(`/api/v1/pool/stats/hashrate?range=${range}`, {
        headers: this._getHeaders(),
      });

      if (!response.ok) {
        throw new Error(`Failed to fetch hashrate history: ${response.status}`);
      }

      const data = await response.json();
      this._hashrateHistory = (data.data || []).map((d: any) => ({
        time: this.formatTime(d.time),
        timestamp: new Date(d.time),
        hashrate: d.hashrate || d.totalHashrate || 0,
        hashrateTH: this.formatHashrateToTH(d.hashrate || d.totalHashrate || 0),
        hashrateMH: (d.hashrate || d.totalHashrate || 0) / 1000000,
      }));

      return this._hashrateHistory;
    } catch (error) {
      this._error = error as Error;
      return [];
    }
  }

  // ============================================================================
  // ISharesProvider Implementation
  // ============================================================================

  getCurrentShares(): { valid: number; invalid: number; rate: number } {
    const valid = this._poolStats?.validShares ?? 0;
    const invalid = this._poolStats?.invalidShares ?? 0;
    return {
      valid,
      invalid,
      rate: this.calculateAcceptanceRate(valid, invalid),
    };
  }

  getSharesHistory(range: TimeRange): ShareDataPoint[] {
    return this._sharesHistory;
  }

  subscribeToShares(callback: Callback<ShareDataPoint>): () => void {
    this._sharesSubscribers.add(callback);
    return () => this._sharesSubscribers.delete(callback);
  }

  async fetchSharesHistory(range: TimeRange): Promise<ShareDataPoint[]> {
    try {
      const response = await fetch(`/api/v1/pool/stats/shares?range=${range}`, {
        headers: this._getHeaders(),
      });

      if (!response.ok) {
        throw new Error(`Failed to fetch shares history: ${response.status}`);
      }

      const data = await response.json();
      this._sharesHistory = (data.data || []).map((d: any) => {
        // API returns validShares/invalidShares (camelCase)
        const valid = d.validShares || d.valid || d.valid_shares || 0;
        const invalid = d.invalidShares || d.invalid || d.invalid_shares || 0;
        return {
          time: this.formatTime(d.time),
          timestamp: new Date(d.time),
          valid,
          invalid,
          total: valid + invalid,
          acceptanceRate: this.calculateAcceptanceRate(valid, invalid),
        };
      });

      return this._sharesHistory;
    } catch (error) {
      this._error = error as Error;
      return [];
    }
  }

  // ============================================================================
  // IMinersProvider Implementation
  // ============================================================================

  getActiveMiners(): MinerInfo[] {
    return this._activeMiners;
  }

  getMinerCount(): number {
    return this._poolStats?.activeMiners ?? 0;
  }

  getMinersHistory(range: TimeRange): MinerDataPoint[] {
    return this._minersHistory;
  }

  subscribeToMiners(callback: Callback<MinerInfo[]>): () => void {
    this._minersSubscribers.add(callback);
    return () => this._minersSubscribers.delete(callback);
  }

  async fetchMinersHistory(range: TimeRange): Promise<MinerDataPoint[]> {
    try {
      const response = await fetch(`/api/v1/pool/stats/miners?range=${range}`, {
        headers: this._getHeaders(),
      });

      if (!response.ok) {
        throw new Error(`Failed to fetch miners history: ${response.status}`);
      }

      const data = await response.json();
      this._minersHistory = (data.data || []).map((d: any) => ({
        time: this.formatTime(d.time),
        timestamp: new Date(d.time),
        activeMiners: d.active_miners || d.activeMiners || 0,
        totalMiners: d.total_miners || d.totalMiners || 0,
        newMiners: d.new_miners || d.newMiners,
      }));

      return this._minersHistory;
    } catch (error) {
      this._error = error as Error;
      return [];
    }
  }

  // ============================================================================
  // IPoolStatsProvider Implementation
  // ============================================================================

  getPoolStats(): PoolStats {
    return this._poolStats || {
      totalHashrate: 0,
      activeMiners: 0,
      totalShares: 0,
      validShares: 0,
      invalidShares: 0,
      acceptanceRate: 100,
      blocksFound: 0,
      networkDifficulty: 0,
      poolDifficulty: 0,
      currency: 'LTC',
      algorithm: 'Scrypt',
    };
  }

  subscribeToPoolStats(callback: Callback<PoolStats>): () => void {
    this._poolStatsSubscribers.add(callback);
    return () => this._poolStatsSubscribers.delete(callback);
  }

  // ============================================================================
  // IUserStatsProvider Implementation
  // ============================================================================

  getUserStats(): UserStats | null {
    if (!this._authToken) return null;
    return this._userStats;
  }

  subscribeToUserStats(callback: Callback<UserStats>): () => void {
    this._userStatsSubscribers.add(callback);
    return () => this._userStatsSubscribers.delete(callback);
  }

  // ============================================================================
  // Control Methods
  // ============================================================================

  setAuthToken(token: string | null): void {
    this._authToken = token;
  }

  async refresh(): Promise<void> {
    this._isLoading = true;
    this._error = null;

    try {
      // Fetch pool stats
      const statsResponse = await fetch('/api/v1/pool/stats', {
        headers: this._getHeaders(),
      });

      if (statsResponse.ok) {
        const data = await statsResponse.json();
        const validShares = data.valid_shares || 0;
        const invalidShares = data.invalid_shares || 0;

        this._poolStats = {
          totalHashrate: data.total_hashrate || 0,
          activeMiners: data.active_miners || data.total_miners || 0,
          totalShares: data.total_shares || validShares + invalidShares,
          validShares,
          invalidShares,
          acceptanceRate: this.calculateAcceptanceRate(validShares, invalidShares),
          blocksFound: data.blocks_found || 0,
          lastBlockTime: data.last_block_time ? new Date(data.last_block_time) : undefined,
          networkDifficulty: data.network_difficulty || 0,
          poolDifficulty: data.pool_difficulty || data.difficulty || 0,
          currency: data.currency || 'LTC',
          algorithm: data.algorithm || 'Scrypt',
        };

        // Notify pool stats subscribers
        this._poolStatsSubscribers.forEach(cb => cb(this._poolStats!));
        
        // Notify hashrate subscribers
        this._hashrateSubscribers.forEach(cb => cb(this._poolStats!.totalHashrate));
      }

      // Fetch miners if available
      try {
        const minersResponse = await fetch('/api/v1/pool/miners', {
          headers: this._getHeaders(),
        });

        if (minersResponse.ok) {
          const data = await minersResponse.json();
          this._activeMiners = (data.miners || []).map((m: any) => ({
            id: m.id?.toString() || m.miner_id?.toString() || '',
            name: m.name || m.miner_name || 'Unknown',
            userId: m.user_id || 0,
            hashrate: m.hashrate || 0,
            difficulty: m.difficulty || 0,
            isActive: m.is_active ?? true,
            lastSeen: new Date(m.last_seen || Date.now()),
            sharesValid: m.shares_valid || m.valid_shares || 0,
            sharesInvalid: m.shares_invalid || m.invalid_shares || 0,
            acceptanceRate: this.calculateAcceptanceRate(
              m.shares_valid || m.valid_shares || 0,
              m.shares_invalid || m.invalid_shares || 0
            ),
          }));

          // Notify miners subscribers
          this._minersSubscribers.forEach(cb => cb(this._activeMiners));
        }
      } catch {
        // Miners endpoint might not exist, that's okay
      }

      // Fetch user stats if authenticated
      if (this._authToken) {
        try {
          const userResponse = await fetch('/api/v1/user/stats', {
            headers: this._getHeaders(),
          });

          if (userResponse.ok) {
            const data = await userResponse.json();
            this._userStats = {
              userId: data.user_id || 0,
              totalHashrate: data.total_hashrate || 0,
              pendingPayout: data.pending_payout || 0,
              totalEarnings: data.total_earnings || 0,
              totalShares: data.total_shares || 0,
              validShares: data.valid_shares || 0,
              invalidShares: data.invalid_shares || 0,
              acceptanceRate: this.calculateAcceptanceRate(
                data.valid_shares || 0,
                data.invalid_shares || 0
              ),
              activeMiners: data.active_miners || 0,
              miners: (data.miners || []).map((m: any) => ({
                id: m.id?.toString() || '',
                name: m.name || 'Unknown',
                userId: data.user_id || 0,
                hashrate: m.hashrate || 0,
                difficulty: m.difficulty || 0,
                isActive: m.is_active ?? true,
                lastSeen: new Date(m.last_seen || Date.now()),
                sharesValid: m.valid_shares || 0,
                sharesInvalid: m.invalid_shares || 0,
                acceptanceRate: this.calculateAcceptanceRate(
                  m.valid_shares || 0,
                  m.invalid_shares || 0
                ),
              })),
            };

            // Notify user stats subscribers
            this._userStatsSubscribers.forEach(cb => cb(this._userStats!));
          }
        } catch {
          // User stats might fail if not authenticated properly
        }
      }

      // Notify shares subscribers with latest data
      const currentShares = this.getCurrentShares();
      this._sharesSubscribers.forEach(cb => cb({
        time: this.formatTime(new Date().toISOString()),
        timestamp: new Date(),
        valid: currentShares.valid,
        invalid: currentShares.invalid,
        total: currentShares.valid + currentShares.invalid,
        acceptanceRate: currentShares.rate,
      }));

      this._isConnected = true;
    } catch (error) {
      this._error = error as Error;
      this._isConnected = false;
    } finally {
      this._isLoading = false;
    }
  }

  async setTimeRange(range: TimeRange): Promise<void> {
    this._timeRange = range;
    await this.refresh();
  }

  getTimeRange(): TimeRange {
    return this._timeRange;
  }

  // ============================================================================
  // Auto-Refresh Control
  // ============================================================================

  startAutoRefresh(intervalMs: number = DEFAULT_REFRESH_CONFIG.interval): void {
    this.stopAutoRefresh();
    this._refreshInterval = setInterval(() => {
      this.refresh();
    }, intervalMs);
  }

  stopAutoRefresh(): void {
    if (this._refreshInterval) {
      clearInterval(this._refreshInterval);
      this._refreshInterval = null;
    }
  }

  // ============================================================================
  // Utility Methods
  // ============================================================================

  formatHashrateToTH(hashrate: number): number {
    return hashrate / 1000000000000; // H/s to TH/s
  }

  calculateAcceptanceRate(valid: number, invalid: number): number {
    const total = valid + invalid;
    if (total === 0) return 100; // No shares = 100% (no failures)
    return (valid / total) * 100;
  }

  formatTime(isoTime: string): string {
    const date = new Date(isoTime);
    return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
  }

  dispose(): void {
    this.stopAutoRefresh();
    this._hashrateSubscribers.clear();
    this._sharesSubscribers.clear();
    this._minersSubscribers.clear();
    this._poolStatsSubscribers.clear();
    this._userStatsSubscribers.clear();
    this._isConnected = false;
  }

  // ============================================================================
  // Private Helpers
  // ============================================================================

  private _getHeaders(): Record<string, string> {
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
    };
    if (this._authToken) {
      headers['Authorization'] = `Bearer ${this._authToken}`;
    }
    return headers;
  }
}

// Singleton instance for app-wide usage
let _instance: RealTimeDataService | null = null;

export function getRealTimeDataService(): RealTimeDataService {
  if (!_instance) {
    _instance = new RealTimeDataService();
  }
  return _instance;
}

export default RealTimeDataService;
