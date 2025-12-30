import { ChartCategory } from './IChartPanel';
import { IGrafanaPanel } from './IGrafanaPanel';
import { INativeChart } from './INativeChart';

/**
 * Union type for all chart configurations
 */
export type ChartConfig = IGrafanaPanel | INativeChart;

/**
 * Chart slot configuration
 */
export interface IChartSlot {
  /** Slot identifier */
  slotId: string;
  /** Currently selected chart ID */
  selectedChartId: string;
  /** Allowed categories for this slot (empty = all) */
  allowedCategories?: ChartCategory[];
  /** Specific chart IDs allowed for this slot (overrides categories if provided) */
  allowedChartIds?: string[];
}

/**
 * Dashboard layout configuration
 */
export interface IDashboardLayout {
  /** Dashboard identifier */
  dashboardId: 'main' | 'miner' | 'admin';
  /** Number of chart slots */
  slotCount: number;
  /** Grid columns */
  columns: number;
  /** Chart slots with their configurations */
  slots: IChartSlot[];
}

/**
 * User chart preferences
 */
export interface IUserChartPreferences {
  /** User ID (or 'anonymous' for public) */
  userId: string;
  /** Per-dashboard slot selections */
  dashboards: {
    [dashboardId: string]: {
      [slotId: string]: string; // chartId
    };
  };
  /** Last updated timestamp */
  updatedAt: Date;
}

/**
 * Chart registry interface
 */
export interface IChartRegistry {
  /** Get all available charts */
  getAllCharts(): ChartConfig[];
  
  /** Get charts by category */
  getChartsByCategory(category: ChartCategory): ChartConfig[];
  
  /** Get chart by ID */
  getChartById(id: string): ChartConfig | undefined;
  
  /** Get charts available for a specific dashboard */
  getChartsForDashboard(dashboardId: string): ChartConfig[];
  
  /** Get default chart for a slot */
  getDefaultChart(dashboardId: string, slotId: string): ChartConfig | undefined;
}

/**
 * Chart selector props
 */
export interface IChartSelectorProps {
  /** Currently selected chart ID */
  selectedChartId: string;
  /** Available charts to choose from */
  availableCharts: ChartConfig[];
  /** Callback when selection changes */
  onSelect: (chartId: string) => void;
  /** Filter by categories */
  categoryFilter?: ChartCategory[];
  /** Disabled state */
  disabled?: boolean;
  /** Additional CSS class */
  className?: string;
}

/**
 * Chart slot component props
 */
export interface IChartSlotProps {
  /** Slot identifier */
  slotId: string;
  /** Dashboard this slot belongs to */
  dashboardId: 'main' | 'miner' | 'admin';
  /** Initial chart ID */
  initialChartId?: string;
  /** Allowed categories for selector */
  allowedCategories?: ChartCategory[];
  /** Specific chart IDs allowed for this slot (overrides categories) */
  allowedChartIds?: string[];
  /** Chart IDs to exclude from selection (prevents duplicates) */
  excludedChartIds?: string[];
  /** Callback when chart selection changes */
  onSelectionChange?: (chartId: string) => void;
  /** Whether selector is visible */
  showSelector?: boolean;
  /** Grafana base URL */
  grafanaBaseUrl: string;
  /** Grafana availability status (passed from parent to avoid multiple health checks) */
  grafanaAvailable?: boolean;
  /** Grafana time range start (e.g., 'now-24h') */
  grafanaTimeFrom?: string;
  /** Grafana time range end (e.g., 'now') */
  grafanaTimeTo?: string;
  /** Fallback data for native charts */
  fallbackData?: any[];
  /** Height override */
  height?: number;
  /** Additional CSS class */
  className?: string;
}
