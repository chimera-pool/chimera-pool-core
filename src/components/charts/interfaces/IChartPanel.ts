/**
 * Base interface for all chart panels
 * Follows Interface Segregation Principle - minimal base contract
 */
export interface IChartPanel {
  /** Unique identifier for this panel */
  id: string;
  /** Display title */
  title: string;
  /** Height in pixels */
  height?: number;
  /** Width as CSS value */
  width?: string;
  /** Category for grouping in selector */
  category: ChartCategory;
  /** Description for tooltip/help */
  description?: string;
}

/**
 * Chart categories for organizing in dropdown
 * Quadrant-based organization for main dashboard
 */
export type ChartCategory = 
  | 'pool-metrics'      // Legacy - for backward compatibility
  | 'worker-metrics'    // Legacy - for backward compatibility
  | 'earnings'          // Legacy - for backward compatibility
  | 'system'            // Admin only
  | 'alerts'            // Admin only
  // New quadrant-based categories for main dashboard
  | 'hashrate-performance'  // Q1: Hashrate & Performance metrics
  | 'workers-activity'      // Q2: Workers & Mining Activity
  | 'shares-blocks'         // Q3: Shares & Blocks
  | 'earnings-payouts';     // Q4: Earnings & Payouts

/**
 * Chart category metadata
 */
export interface IChartCategoryInfo {
  id: ChartCategory;
  label: string;
  icon: string;
  description: string;
  quadrant?: 1 | 2 | 3 | 4;  // Which quadrant this category belongs to
}

/**
 * All available chart categories with metadata
 */
export const CHART_CATEGORIES: IChartCategoryInfo[] = [
  // Legacy categories (for backward compatibility)
  {
    id: 'pool-metrics',
    label: 'Pool Metrics',
    icon: '📊',
    description: 'Overall pool performance and statistics',
  },
  {
    id: 'worker-metrics',
    label: 'Worker Metrics',
    icon: '👷',
    description: 'Individual worker and miner statistics',
  },
  {
    id: 'earnings',
    label: 'Earnings',
    icon: '💰',
    description: 'Payout and earnings data',
  },
  {
    id: 'system',
    label: 'System',
    icon: '⚙️',
    description: 'System health and performance (Admin)',
  },
  {
    id: 'alerts',
    label: 'Alerts',
    icon: '🔔',
    description: 'Alert history and monitoring (Admin)',
  },
  // New quadrant-based categories
  {
    id: 'hashrate-performance',
    label: 'Hashrate & Performance',
    icon: '⚡',
    description: 'Pool hashrate, performance metrics, and efficiency',
    quadrant: 1,
  },
  {
    id: 'workers-activity',
    label: 'Workers & Activity',
    icon: '👷',
    description: 'Worker counts, status, availability, and connections',
    quadrant: 2,
  },
  {
    id: 'shares-blocks',
    label: 'Shares & Blocks',
    icon: '🧱',
    description: 'Share submission, acceptance rates, and blocks found',
    quadrant: 3,
  },
  {
    id: 'earnings-payouts',
    label: 'Earnings & Payouts',
    icon: '💰',
    description: 'Wallet balance, pending payouts, and payout history',
    quadrant: 4,
  },
];
