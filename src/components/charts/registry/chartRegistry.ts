import { 
  ChartConfig, 
  IChartRegistry, 
  IDashboardLayout,
  IChartSlot 
} from '../interfaces/IChartRegistry';
import { ChartCategory } from '../interfaces/IChartPanel';
import { IGrafanaPanel } from '../interfaces/IGrafanaPanel';
import { INativeChart, NATIVE_CHART_COLORS } from '../interfaces/INativeChart';

/**
 * Grafana configuration for Chimera Pool
 */
export const GRAFANA_CONFIG = {
  // Direct HTTP URL - works in Chrome, Edge blocks due to mixed content
  baseUrl: 'http://206.162.80.230:3001',
  orgId: 1,
  dashboards: {
    poolOverview: 'chimera-pool-overview',
    workers: 'chimera-pool-workers',
    payouts: 'chimera-pool-payouts',
    alerts: 'chimera-pool-alerts',
  },
};

/**
 * All available Grafana panels
 */
const GRAFANA_PANELS: IGrafanaPanel[] = [
  // Pool Metrics
  {
    id: 'grafana-pool-hashrate',
    type: 'grafana',
    title: 'Pool Hashrate',
    category: 'pool-metrics',
    description: 'Total pool hashrate over time',
    dashboardUid: GRAFANA_CONFIG.dashboards.poolOverview,
    panelId: 1,
    height: 280,
    from: 'now-24h',
    to: 'now',
    refreshInterval: 30,
    theme: 'dark',
  },
  {
    id: 'grafana-pool-hashrate-7d',
    type: 'grafana',
    title: 'Pool Hashrate (7 Days)',
    category: 'pool-metrics',
    description: 'Total pool hashrate over 7 days',
    dashboardUid: GRAFANA_CONFIG.dashboards.poolOverview,
    panelId: 1,
    height: 280,
    from: 'now-7d',
    to: 'now',
    refreshInterval: 60,
    theme: 'dark',
  },
  {
    id: 'grafana-shares-submitted',
    type: 'grafana',
    title: 'Shares Submitted',
    category: 'pool-metrics',
    description: 'Valid and invalid shares over time',
    dashboardUid: GRAFANA_CONFIG.dashboards.poolOverview,
    panelId: 2,
    height: 280,
    from: 'now-24h',
    to: 'now',
    refreshInterval: 30,
    theme: 'dark',
  },
  {
    id: 'grafana-active-miners',
    type: 'grafana',
    title: 'Active Miners',
    category: 'pool-metrics',
    description: 'Number of active miners over time',
    dashboardUid: GRAFANA_CONFIG.dashboards.poolOverview,
    panelId: 3,
    height: 280,
    from: 'now-24h',
    to: 'now',
    refreshInterval: 30,
    theme: 'dark',
  },
  {
    id: 'grafana-block-history',
    type: 'grafana',
    title: 'Block History',
    category: 'pool-metrics',
    description: 'Blocks found by the pool',
    dashboardUid: GRAFANA_CONFIG.dashboards.poolOverview,
    panelId: 4,
    height: 280,
    from: 'now-7d',
    to: 'now',
    refreshInterval: 60,
    theme: 'dark',
  },
  {
    id: 'grafana-network-difficulty',
    type: 'grafana',
    title: 'Network Difficulty',
    category: 'pool-metrics',
    description: 'Network difficulty over time',
    dashboardUid: GRAFANA_CONFIG.dashboards.poolOverview,
    panelId: 5,
    height: 280,
    from: 'now-24h',
    to: 'now',
    refreshInterval: 60,
    theme: 'dark',
  },

  // Worker Metrics
  {
    id: 'grafana-worker-hashrate',
    type: 'grafana',
    title: 'Worker Hashrate',
    category: 'worker-metrics',
    description: 'Individual worker hashrate',
    dashboardUid: GRAFANA_CONFIG.dashboards.workers,
    panelId: 1,
    height: 280,
    from: 'now-24h',
    to: 'now',
    refreshInterval: 30,
    theme: 'dark',
  },
  {
    id: 'grafana-worker-shares',
    type: 'grafana',
    title: 'Worker Shares',
    category: 'worker-metrics',
    description: 'Worker share submission rate',
    dashboardUid: GRAFANA_CONFIG.dashboards.workers,
    panelId: 2,
    height: 280,
    from: 'now-24h',
    to: 'now',
    refreshInterval: 30,
    theme: 'dark',
  },
  {
    id: 'grafana-worker-status',
    type: 'grafana',
    title: 'Worker Status',
    category: 'worker-metrics',
    description: 'Online/offline worker status',
    dashboardUid: GRAFANA_CONFIG.dashboards.workers,
    panelId: 3,
    height: 280,
    from: 'now-24h',
    to: 'now',
    refreshInterval: 30,
    theme: 'dark',
  },
  {
    id: 'grafana-worker-efficiency',
    type: 'grafana',
    title: 'Worker Efficiency',
    category: 'worker-metrics',
    description: 'Share acceptance rate by worker',
    dashboardUid: GRAFANA_CONFIG.dashboards.workers,
    panelId: 4,
    height: 280,
    from: 'now-24h',
    to: 'now',
    refreshInterval: 30,
    theme: 'dark',
  },

  // Earnings
  {
    id: 'grafana-earnings-cumulative',
    type: 'grafana',
    title: 'Cumulative Earnings',
    category: 'earnings',
    description: 'Total earnings over time',
    dashboardUid: GRAFANA_CONFIG.dashboards.payouts,
    panelId: 1,
    height: 280,
    from: 'now-30d',
    to: 'now',
    refreshInterval: 60,
    theme: 'dark',
  },
  {
    id: 'grafana-earnings-daily',
    type: 'grafana',
    title: 'Daily Earnings',
    category: 'earnings',
    description: 'Earnings per day',
    dashboardUid: GRAFANA_CONFIG.dashboards.payouts,
    panelId: 2,
    height: 280,
    from: 'now-30d',
    to: 'now',
    refreshInterval: 60,
    theme: 'dark',
  },
  {
    id: 'grafana-payout-history',
    type: 'grafana',
    title: 'Payout History',
    category: 'earnings',
    description: 'Historical payouts',
    dashboardUid: GRAFANA_CONFIG.dashboards.payouts,
    panelId: 3,
    height: 280,
    from: 'now-90d',
    to: 'now',
    refreshInterval: 300,
    theme: 'dark',
  },
  {
    id: 'grafana-pending-balance',
    type: 'grafana',
    title: 'Pending Balance',
    category: 'earnings',
    description: 'Unpaid balance over time',
    dashboardUid: GRAFANA_CONFIG.dashboards.payouts,
    panelId: 4,
    height: 280,
    from: 'now-7d',
    to: 'now',
    refreshInterval: 60,
    theme: 'dark',
  },

  // System (Admin)
  {
    id: 'grafana-node-health',
    type: 'grafana',
    title: 'Node Health',
    category: 'system',
    description: 'Blockchain node health status',
    dashboardUid: GRAFANA_CONFIG.dashboards.poolOverview,
    panelId: 6,
    height: 280,
    from: 'now-24h',
    to: 'now',
    refreshInterval: 30,
    theme: 'dark',
  },
  {
    id: 'grafana-stratum-connections',
    type: 'grafana',
    title: 'Stratum Connections',
    category: 'system',
    description: 'Active stratum connections',
    dashboardUid: GRAFANA_CONFIG.dashboards.poolOverview,
    panelId: 7,
    height: 280,
    from: 'now-24h',
    to: 'now',
    refreshInterval: 30,
    theme: 'dark',
  },
  {
    id: 'grafana-api-latency',
    type: 'grafana',
    title: 'API Latency',
    category: 'system',
    description: 'API response times',
    dashboardUid: GRAFANA_CONFIG.dashboards.poolOverview,
    panelId: 8,
    height: 280,
    from: 'now-24h',
    to: 'now',
    refreshInterval: 30,
    theme: 'dark',
  },

  // Alerts (Admin)
  {
    id: 'grafana-alert-timeline',
    type: 'grafana',
    title: 'Alert Timeline',
    category: 'alerts',
    description: 'Alert history over time',
    dashboardUid: GRAFANA_CONFIG.dashboards.alerts,
    panelId: 1,
    height: 280,
    from: 'now-7d',
    to: 'now',
    refreshInterval: 60,
    theme: 'dark',
  },
  {
    id: 'grafana-alert-frequency',
    type: 'grafana',
    title: 'Alert Frequency',
    category: 'alerts',
    description: 'Alerts by type and frequency',
    dashboardUid: GRAFANA_CONFIG.dashboards.alerts,
    panelId: 2,
    height: 280,
    from: 'now-7d',
    to: 'now',
    refreshInterval: 60,
    theme: 'dark',
  },
];

/**
 * Native chart fallbacks (used when Grafana unavailable)
 */
const NATIVE_CHARTS: INativeChart[] = [
  {
    id: 'native-pool-hashrate',
    type: 'native',
    title: 'Pool Hashrate',
    category: 'pool-metrics',
    description: 'Total pool hashrate (fallback)',
    chartType: 'area',
    dataKey: 'hashrateTH',
    color: NATIVE_CHART_COLORS.gold,
    gradientOpacity: 0.25,
    height: 280,
    apiEndpoint: '/api/v1/pool/stats/hashrate',
    yAxisFormatter: (v: number) => `${v.toFixed(1)} TH/s`,
    tooltipFormatter: (v: number) => `${v.toFixed(2)} TH/s`,
  },
  {
    id: 'native-shares-submitted',
    type: 'native',
    title: 'Shares Submitted',
    category: 'pool-metrics',
    description: 'Valid and invalid shares (fallback)',
    chartType: 'area',
    dataKey: 'validShares',
    color: NATIVE_CHART_COLORS.green,
    colorSecondary: NATIVE_CHART_COLORS.coral,
    gradientOpacity: 0.25,
    height: 280,
    apiEndpoint: '/api/v1/pool/stats/shares',
  },
  {
    id: 'native-acceptance-rate',
    type: 'native',
    title: 'Acceptance Rate',
    category: 'pool-metrics',
    description: 'Share acceptance rate (fallback)',
    chartType: 'area',
    dataKey: 'acceptanceRate',
    color: NATIVE_CHART_COLORS.blue,
    gradientOpacity: 0.25,
    height: 280,
    apiEndpoint: '/api/v1/pool/stats/shares',
    yAxisFormatter: (v: number) => `${v}%`,
    tooltipFormatter: (v: number) => `${v.toFixed(2)}%`,
  },
  {
    id: 'native-earnings',
    type: 'native',
    title: 'Cumulative Earnings',
    category: 'earnings',
    description: 'Earnings over time (fallback)',
    chartType: 'area',
    dataKey: 'cumulative',
    color: NATIVE_CHART_COLORS.purple,
    gradientOpacity: 0.25,
    height: 280,
    apiEndpoint: '/api/v1/pool/stats/earnings',
    tooltipFormatter: (v: number) => `${v.toFixed(4)} LTC`,
  },
];

/**
 * All available charts combined
 */
const ALL_CHARTS: ChartConfig[] = [...GRAFANA_PANELS, ...NATIVE_CHARTS];

/**
 * Default dashboard layouts
 */
export const DEFAULT_LAYOUTS: Record<string, IDashboardLayout> = {
  main: {
    dashboardId: 'main',
    slotCount: 4,
    columns: 2,
    slots: [
      { slotId: 'main-1', selectedChartId: 'grafana-pool-hashrate', allowedCategories: ['pool-metrics'] },
      { slotId: 'main-2', selectedChartId: 'grafana-shares-submitted', allowedCategories: ['pool-metrics'] },
      { slotId: 'main-3', selectedChartId: 'grafana-active-miners', allowedCategories: ['pool-metrics'] },
      { slotId: 'main-4', selectedChartId: 'grafana-block-history', allowedCategories: ['pool-metrics', 'earnings'] },
    ],
  },
  miner: {
    dashboardId: 'miner',
    slotCount: 4,
    columns: 2,
    slots: [
      { slotId: 'miner-1', selectedChartId: 'grafana-worker-hashrate', allowedCategories: ['worker-metrics'] },
      { slotId: 'miner-2', selectedChartId: 'grafana-worker-shares', allowedCategories: ['worker-metrics'] },
      { slotId: 'miner-3', selectedChartId: 'grafana-earnings-cumulative', allowedCategories: ['earnings'] },
      { slotId: 'miner-4', selectedChartId: 'grafana-worker-status', allowedCategories: ['worker-metrics'] },
    ],
  },
  admin: {
    dashboardId: 'admin',
    slotCount: 4,
    columns: 2,
    slots: [
      { slotId: 'admin-1', selectedChartId: 'grafana-node-health', allowedCategories: ['system'] },
      { slotId: 'admin-2', selectedChartId: 'grafana-stratum-connections', allowedCategories: ['system'] },
      { slotId: 'admin-3', selectedChartId: 'grafana-payout-history', allowedCategories: ['earnings'] },
      { slotId: 'admin-4', selectedChartId: 'grafana-alert-timeline', allowedCategories: ['alerts'] },
    ],
  },
};

/**
 * Chart Registry Implementation
 */
class ChartRegistryImpl implements IChartRegistry {
  private charts: ChartConfig[] = ALL_CHARTS;
  private layouts: Record<string, IDashboardLayout> = DEFAULT_LAYOUTS;

  getAllCharts(): ChartConfig[] {
    return this.charts;
  }

  getChartsByCategory(category: ChartCategory): ChartConfig[] {
    return this.charts.filter(chart => chart.category === category);
  }

  getChartById(id: string): ChartConfig | undefined {
    return this.charts.find(chart => chart.id === id);
  }

  getChartsForDashboard(dashboardId: string): ChartConfig[] {
    const layout = this.layouts[dashboardId];
    if (!layout) return this.charts;

    // Get all allowed categories for this dashboard
    const allowedCategories = new Set<ChartCategory>();
    layout.slots.forEach(slot => {
      slot.allowedCategories?.forEach(cat => allowedCategories.add(cat));
    });

    if (allowedCategories.size === 0) return this.charts;
    return this.charts.filter(chart => allowedCategories.has(chart.category));
  }

  getDefaultChart(dashboardId: string, slotId: string): ChartConfig | undefined {
    const layout = this.layouts[dashboardId];
    if (!layout) return undefined;

    const slot = layout.slots.find(s => s.slotId === slotId);
    if (!slot) return undefined;

    return this.getChartById(slot.selectedChartId);
  }

  getLayout(dashboardId: string): IDashboardLayout | undefined {
    return this.layouts[dashboardId];
  }

  getNativeFallback(grafanaChartId: string): INativeChart | undefined {
    // Map Grafana chart to native fallback
    const mappings: Record<string, string> = {
      'grafana-pool-hashrate': 'native-pool-hashrate',
      'grafana-pool-hashrate-7d': 'native-pool-hashrate',
      'grafana-shares-submitted': 'native-shares-submitted',
      'grafana-active-miners': 'native-acceptance-rate',
      'grafana-earnings-cumulative': 'native-earnings',
      'grafana-earnings-daily': 'native-earnings',
    };

    const fallbackId = mappings[grafanaChartId];
    if (!fallbackId) return NATIVE_CHARTS[0]; // Default fallback

    return NATIVE_CHARTS.find(c => c.id === fallbackId);
  }
}

/**
 * Singleton chart registry instance
 */
export const chartRegistry = new ChartRegistryImpl();
