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
 */
export type ChartCategory = 
  | 'pool-metrics'
  | 'worker-metrics'
  | 'earnings'
  | 'system'
  | 'alerts';

/**
 * Chart category metadata
 */
export interface IChartCategoryInfo {
  id: ChartCategory;
  label: string;
  icon: string;
  description: string;
}

/**
 * All available chart categories with metadata
 */
export const CHART_CATEGORIES: IChartCategoryInfo[] = [
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
];
