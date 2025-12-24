/**
 * Graph Data Transformation Utilities
 * Follows Interface Segregation Principle - separate interfaces for different data types
 */

// Re-export formatHashrate from formatters.ts to maintain backward compatibility
export { formatHashrate } from './formatters';

// Interfaces for raw API responses (ISP)
export interface IRawHashrateData {
  time: string;
  hashrate?: number;
  totalHashrate?: number;
}

export interface IRawSharesData {
  time: string;
  validShares: number;
  invalidShares: number;
  totalShares?: number;
  acceptanceRate?: number;
}

export interface IRawMinersData {
  time: string;
  activeMiners: number;
  uniqueUsers?: number;
}

// Interfaces for transformed chart data (ISP)
export interface IChartHashrateData {
  time: string;
  hashrateMH: number;
  hashrateTH: number;
  hashrate: number;
}

export interface IChartSharesData {
  time: string;
  validShares: number;
  invalidShares: number;
  totalShares: number;
  acceptanceRate: number;
}

export interface IChartMinersData {
  time: string;
  activeMiners: number;
  uniqueUsers: number;
}

// Transformer interfaces (ISP)
export interface IHashrateTransformer {
  transform(data: IRawHashrateData[]): IChartHashrateData[];
}

export interface ISharesTransformer {
  transform(data: IRawSharesData[]): IChartSharesData[];
}

export interface IMinersTransformer {
  transform(data: IRawMinersData[]): IChartMinersData[];
}

/**
 * Transforms raw hashrate data from API to chart-ready format
 * Handles both 'hashrate' and 'totalHashrate' field names
 */
export function transformHashrateData(data: IRawHashrateData[]): IChartHashrateData[] {
  if (!data || !Array.isArray(data)) return [];
  
  return data.map(d => {
    const rawHashrate = d.hashrate || d.totalHashrate || 0;
    return {
      time: formatTime(d.time),
      hashrate: rawHashrate,
      hashrateMH: rawHashrate / 1_000_000,
      hashrateTH: rawHashrate / 1_000_000_000_000,
    };
  });
}

/**
 * Transforms raw shares data from API to chart-ready format
 */
export function transformSharesData(data: IRawSharesData[]): IChartSharesData[] {
  if (!data || !Array.isArray(data)) return [];
  
  return data.map(d => {
    const total = d.totalShares || (d.validShares + d.invalidShares);
    const acceptanceRate = d.acceptanceRate || (total > 0 ? (d.validShares / total) * 100 : 0);
    
    return {
      time: formatTime(d.time),
      validShares: d.validShares,
      invalidShares: d.invalidShares,
      totalShares: total,
      acceptanceRate,
    };
  });
}

/**
 * Transforms raw miners data from API to chart-ready format
 */
export function transformMinersData(data: IRawMinersData[]): IChartMinersData[] {
  if (!data || !Array.isArray(data)) return [];
  
  return data.map(d => ({
    time: formatTime(d.time),
    activeMiners: d.activeMiners,
    uniqueUsers: d.uniqueUsers || d.activeMiners,
  }));
}

/**
 * Formats ISO timestamp to display time
 */
export function formatTime(isoString: string): string {
  try {
    return new Date(isoString).toLocaleTimeString([], { 
      hour: '2-digit', 
      minute: '2-digit' 
    });
  } catch {
    return isoString;
  }
}

