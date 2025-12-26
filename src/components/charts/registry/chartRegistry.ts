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
  // Proxied through nginx at /grafana - serves over HTTPS to avoid mixed content
  baseUrl: '/grafana',
  orgId: 1,
  dashboards: {
    poolOverview: 'chimera-pool-overview',
    workers: 'chimera-pool-workers',
    payouts: 'chimera-pool-payouts',
    alerts: 'chimera-pool-alerts',
  },
};

/**
 * All available Grafana panels - organized by quadrant categories
 * Total: 29 panels from 4 dashboards
 */
const GRAFANA_PANELS: IGrafanaPanel[] = [
  // ============================================
  // QUADRANT 1: Hashrate & Performance (8 panels)
  // ============================================
  {
    id: 'grafana-pool-hashrate-stat',
    type: 'grafana',
    title: 'Pool Hashrate',
    category: 'hashrate-performance',
    description: 'Current total pool hashrate',
    dashboardUid: GRAFANA_CONFIG.dashboards.poolOverview,
    panelId: 1,
    height: 280,
    from: 'now-24h',
    to: 'now',
    refreshInterval: 30,
    theme: 'dark',
  },
  {
    id: 'grafana-pool-hashrate-history',
    type: 'grafana',
    title: 'Pool Hashrate History',
    category: 'hashrate-performance',
    description: 'Pool hashrate over time with mean and max',
    dashboardUid: GRAFANA_CONFIG.dashboards.poolOverview,
    panelId: 5,
    height: 280,
    from: 'now-6h',
    to: 'now',
    refreshInterval: 30,
    theme: 'dark',
  },
  {
    id: 'grafana-avg-hashrate-per-worker',
    type: 'grafana',
    title: 'Avg Hashrate per Worker',
    category: 'hashrate-performance',
    description: 'Average hashrate contribution per worker',
    dashboardUid: GRAFANA_CONFIG.dashboards.workers,
    panelId: 6,
    height: 280,
    from: 'now-6h',
    to: 'now',
    refreshInterval: 30,
    theme: 'dark',
  },
  {
    id: 'grafana-share-rejection-rate',
    type: 'grafana',
    title: 'Share Rejection Rate',
    category: 'hashrate-performance',
    description: 'Percentage of rejected shares over time',
    dashboardUid: GRAFANA_CONFIG.dashboards.poolOverview,
    panelId: 8,
    height: 280,
    from: 'now-6h',
    to: 'now',
    refreshInterval: 30,
    theme: 'dark',
  },
  {
    id: 'grafana-worker-availability',
    type: 'grafana',
    title: 'Worker Availability Rate',
    category: 'hashrate-performance',
    description: 'Percentage of workers currently online',
    dashboardUid: GRAFANA_CONFIG.dashboards.workers,
    panelId: 7,
    height: 280,
    from: 'now-6h',
    to: 'now',
    refreshInterval: 30,
    theme: 'dark',
  },
  {
    id: 'grafana-payout-processing-duration',
    type: 'grafana',
    title: 'Payout Processing Time',
    category: 'hashrate-performance',
    description: 'P95 payout processing duration',
    dashboardUid: GRAFANA_CONFIG.dashboards.payouts,
    panelId: 6,
    height: 280,
    from: 'now-24h',
    to: 'now',
    refreshInterval: 60,
    theme: 'dark',
  },

  // ============================================
  // QUADRANT 2: Workers & Activity (8 panels)
  // ============================================
  {
    id: 'grafana-active-workers-stat',
    type: 'grafana',
    title: 'Active Workers',
    category: 'workers-activity',
    description: 'Current number of online workers',
    dashboardUid: GRAFANA_CONFIG.dashboards.poolOverview,
    panelId: 2,
    height: 280,
    from: 'now-24h',
    to: 'now',
    refreshInterval: 30,
    theme: 'dark',
  },
  {
    id: 'grafana-active-workers-history',
    type: 'grafana',
    title: 'Workers History',
    category: 'workers-activity',
    description: 'Online vs offline workers over time',
    dashboardUid: GRAFANA_CONFIG.dashboards.poolOverview,
    panelId: 6,
    height: 280,
    from: 'now-6h',
    to: 'now',
    refreshInterval: 30,
    theme: 'dark',
  },
  {
    id: 'grafana-total-workers',
    type: 'grafana',
    title: 'Total Workers',
    category: 'workers-activity',
    description: 'Total registered workers',
    dashboardUid: GRAFANA_CONFIG.dashboards.workers,
    panelId: 1,
    height: 280,
    from: 'now-24h',
    to: 'now',
    refreshInterval: 30,
    theme: 'dark',
  },
  {
    id: 'grafana-online-workers',
    type: 'grafana',
    title: 'Online Workers',
    category: 'workers-activity',
    description: 'Currently connected workers',
    dashboardUid: GRAFANA_CONFIG.dashboards.workers,
    panelId: 2,
    height: 280,
    from: 'now-24h',
    to: 'now',
    refreshInterval: 30,
    theme: 'dark',
  },
  {
    id: 'grafana-offline-workers',
    type: 'grafana',
    title: 'Offline Workers',
    category: 'workers-activity',
    description: 'Currently disconnected workers',
    dashboardUid: GRAFANA_CONFIG.dashboards.workers,
    panelId: 3,
    height: 280,
    from: 'now-24h',
    to: 'now',
    refreshInterval: 30,
    theme: 'dark',
  },
  {
    id: 'grafana-unique-miners',
    type: 'grafana',
    title: 'Unique Miners',
    category: 'workers-activity',
    description: 'Distinct miner accounts',
    dashboardUid: GRAFANA_CONFIG.dashboards.workers,
    panelId: 4,
    height: 280,
    from: 'now-24h',
    to: 'now',
    refreshInterval: 30,
    theme: 'dark',
  },
  {
    id: 'grafana-worker-status-timeline',
    type: 'grafana',
    title: 'Worker Status Timeline',
    category: 'workers-activity',
    description: 'Online/offline status over time',
    dashboardUid: GRAFANA_CONFIG.dashboards.workers,
    panelId: 5,
    height: 280,
    from: 'now-6h',
    to: 'now',
    refreshInterval: 30,
    theme: 'dark',
  },
  {
    id: 'grafana-new-connections',
    type: 'grafana',
    title: 'New Connections',
    category: 'workers-activity',
    description: 'New stratum connections per hour',
    dashboardUid: GRAFANA_CONFIG.dashboards.workers,
    panelId: 8,
    height: 280,
    from: 'now-6h',
    to: 'now',
    refreshInterval: 30,
    theme: 'dark',
  },

  // ============================================
  // QUADRANT 3: Shares & Blocks (7 panels)
  // ============================================
  {
    id: 'grafana-blocks-found-24h',
    type: 'grafana',
    title: 'Blocks Found (24h)',
    category: 'shares-blocks',
    description: 'Blocks discovered in last 24 hours',
    dashboardUid: GRAFANA_CONFIG.dashboards.poolOverview,
    panelId: 3,
    height: 280,
    from: 'now-24h',
    to: 'now',
    refreshInterval: 60,
    theme: 'dark',
  },
  {
    id: 'grafana-shares-submitted',
    type: 'grafana',
    title: 'Shares Submitted',
    category: 'shares-blocks',
    description: 'Accepted vs rejected shares per 5 minutes',
    dashboardUid: GRAFANA_CONFIG.dashboards.poolOverview,
    panelId: 7,
    height: 280,
    from: 'now-6h',
    to: 'now',
    refreshInterval: 30,
    theme: 'dark',
  },
  {
    id: 'grafana-alerts-by-type',
    type: 'grafana',
    title: 'Alerts by Type',
    category: 'shares-blocks',
    description: 'Worker offline, payout sent, block found alerts',
    dashboardUid: GRAFANA_CONFIG.dashboards.alerts,
    panelId: 3,
    height: 280,
    from: 'now-24h',
    to: 'now',
    refreshInterval: 60,
    theme: 'dark',
  },
  {
    id: 'grafana-notifications-sent',
    type: 'grafana',
    title: 'Notifications Sent',
    category: 'shares-blocks',
    description: 'Email and Discord notifications per hour',
    dashboardUid: GRAFANA_CONFIG.dashboards.alerts,
    panelId: 2,
    height: 280,
    from: 'now-24h',
    to: 'now',
    refreshInterval: 60,
    theme: 'dark',
  },
  {
    id: 'grafana-total-notifications-24h',
    type: 'grafana',
    title: 'Total Notifications (24h)',
    category: 'shares-blocks',
    description: 'All notifications sent in last 24 hours',
    dashboardUid: GRAFANA_CONFIG.dashboards.alerts,
    panelId: 4,
    height: 280,
    from: 'now-24h',
    to: 'now',
    refreshInterval: 60,
    theme: 'dark',
  },
  {
    id: 'grafana-failed-notifications',
    type: 'grafana',
    title: 'Failed Notifications (24h)',
    category: 'shares-blocks',
    description: 'Failed notification deliveries',
    dashboardUid: GRAFANA_CONFIG.dashboards.alerts,
    panelId: 5,
    height: 280,
    from: 'now-24h',
    to: 'now',
    refreshInterval: 60,
    theme: 'dark',
  },
  {
    id: 'grafana-active-alerts',
    type: 'grafana',
    title: 'Active Alerts',
    category: 'shares-blocks',
    description: 'Currently firing and pending alerts',
    dashboardUid: GRAFANA_CONFIG.dashboards.alerts,
    panelId: 1,
    height: 280,
    from: 'now-24h',
    to: 'now',
    refreshInterval: 30,
    theme: 'dark',
  },

  // ============================================
  // QUADRANT 4: Earnings & Payouts (8 panels)
  // ============================================
  {
    id: 'grafana-wallet-balance',
    type: 'grafana',
    title: 'Wallet Balance (LTC)',
    category: 'earnings-payouts',
    description: 'Current pool wallet balance',
    dashboardUid: GRAFANA_CONFIG.dashboards.poolOverview,
    panelId: 4,
    height: 280,
    from: 'now-24h',
    to: 'now',
    refreshInterval: 60,
    theme: 'dark',
  },
  {
    id: 'grafana-pending-payouts',
    type: 'grafana',
    title: 'Pending Payouts',
    category: 'earnings-payouts',
    description: 'Number of payouts waiting to be processed',
    dashboardUid: GRAFANA_CONFIG.dashboards.payouts,
    panelId: 1,
    height: 280,
    from: 'now-24h',
    to: 'now',
    refreshInterval: 60,
    theme: 'dark',
  },
  {
    id: 'grafana-payouts-processed-24h',
    type: 'grafana',
    title: 'Payouts Processed (24h)',
    category: 'earnings-payouts',
    description: 'Successful payouts in last 24 hours',
    dashboardUid: GRAFANA_CONFIG.dashboards.payouts,
    panelId: 2,
    height: 280,
    from: 'now-24h',
    to: 'now',
    refreshInterval: 60,
    theme: 'dark',
  },
  {
    id: 'grafana-payouts-failed-24h',
    type: 'grafana',
    title: 'Payouts Failed (24h)',
    category: 'earnings-payouts',
    description: 'Failed payouts in last 24 hours',
    dashboardUid: GRAFANA_CONFIG.dashboards.payouts,
    panelId: 3,
    height: 280,
    from: 'now-24h',
    to: 'now',
    refreshInterval: 60,
    theme: 'dark',
  },
  {
    id: 'grafana-payouts-wallet-balance',
    type: 'grafana',
    title: 'Payout Wallet Balance',
    category: 'earnings-payouts',
    description: 'Wallet balance from payouts dashboard',
    dashboardUid: GRAFANA_CONFIG.dashboards.payouts,
    panelId: 4,
    height: 280,
    from: 'now-24h',
    to: 'now',
    refreshInterval: 60,
    theme: 'dark',
  },
  {
    id: 'grafana-payouts-by-mode',
    type: 'grafana',
    title: 'Payouts by Mode',
    category: 'earnings-payouts',
    description: 'PPLNS, SLICE, PPS payout distribution',
    dashboardUid: GRAFANA_CONFIG.dashboards.payouts,
    panelId: 5,
    height: 280,
    from: 'now-24h',
    to: 'now',
    refreshInterval: 60,
    theme: 'dark',
  },
  {
    id: 'grafana-payout-failures-by-reason',
    type: 'grafana',
    title: 'Payout Failures by Reason',
    category: 'earnings-payouts',
    description: 'Insufficient funds, invalid address, network errors',
    dashboardUid: GRAFANA_CONFIG.dashboards.payouts,
    panelId: 7,
    height: 280,
    from: 'now-24h',
    to: 'now',
    refreshInterval: 60,
    theme: 'dark',
  },
  {
    id: 'grafana-rate-limited-alerts',
    type: 'grafana',
    title: 'Rate Limited Alerts (24h)',
    category: 'earnings-payouts',
    description: 'Alerts that were rate limited',
    dashboardUid: GRAFANA_CONFIG.dashboards.alerts,
    panelId: 6,
    height: 280,
    from: 'now-24h',
    to: 'now',
    refreshInterval: 60,
    theme: 'dark',
  },

  // ============================================
  // ADMIN-ONLY: System & Alerts (legacy)
  // ============================================
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
 * Each slot has unique allowedChartIds organized by quadrant theme
 * Main dashboard: 4 quadrants with 6-8 charts each (no overlap)
 */
export const DEFAULT_LAYOUTS: Record<string, IDashboardLayout> = {
  main: {
    dashboardId: 'main',
    slotCount: 4,
    columns: 2,
    slots: [
      { 
        // QUADRANT 1: Hashrate & Performance
        slotId: 'main-1', 
        selectedChartId: 'grafana-pool-hashrate-stat', 
        allowedCategories: ['hashrate-performance'],
        allowedChartIds: [
          'grafana-pool-hashrate-stat',
          'grafana-pool-hashrate-history',
          'grafana-avg-hashrate-per-worker',
          'grafana-share-rejection-rate',
          'grafana-worker-availability',
          'grafana-payout-processing-duration',
        ]
      },
      { 
        // QUADRANT 2: Workers & Activity
        slotId: 'main-2', 
        selectedChartId: 'grafana-active-workers-stat', 
        allowedCategories: ['workers-activity'],
        allowedChartIds: [
          'grafana-active-workers-stat',
          'grafana-active-workers-history',
          'grafana-total-workers',
          'grafana-online-workers',
          'grafana-offline-workers',
          'grafana-unique-miners',
          'grafana-worker-status-timeline',
          'grafana-new-connections',
        ]
      },
      { 
        // QUADRANT 3: Shares & Blocks
        slotId: 'main-3', 
        selectedChartId: 'grafana-blocks-found-24h', 
        allowedCategories: ['shares-blocks'],
        allowedChartIds: [
          'grafana-blocks-found-24h',
          'grafana-shares-submitted',
          'grafana-alerts-by-type',
          'grafana-notifications-sent',
          'grafana-total-notifications-24h',
          'grafana-failed-notifications',
          'grafana-active-alerts',
        ]
      },
      { 
        // QUADRANT 4: Earnings & Payouts
        slotId: 'main-4', 
        selectedChartId: 'grafana-wallet-balance', 
        allowedCategories: ['earnings-payouts'],
        allowedChartIds: [
          'grafana-wallet-balance',
          'grafana-pending-payouts',
          'grafana-payouts-processed-24h',
          'grafana-payouts-failed-24h',
          'grafana-payouts-wallet-balance',
          'grafana-payouts-by-mode',
          'grafana-payout-failures-by-reason',
          'grafana-rate-limited-alerts',
        ]
      },
    ],
  },
  miner: {
    dashboardId: 'miner',
    slotCount: 4,
    columns: 2,
    slots: [
      { 
        slotId: 'miner-1', 
        selectedChartId: 'grafana-pool-hashrate-history', 
        allowedCategories: ['hashrate-performance'],
        allowedChartIds: [
          'grafana-pool-hashrate-stat',
          'grafana-pool-hashrate-history',
          'grafana-avg-hashrate-per-worker',
        ]
      },
      { 
        slotId: 'miner-2', 
        selectedChartId: 'grafana-worker-status-timeline', 
        allowedCategories: ['workers-activity'],
        allowedChartIds: [
          'grafana-worker-status-timeline',
          'grafana-online-workers',
          'grafana-offline-workers',
        ]
      },
      { 
        slotId: 'miner-3', 
        selectedChartId: 'grafana-shares-submitted', 
        allowedCategories: ['shares-blocks'],
        allowedChartIds: [
          'grafana-shares-submitted',
          'grafana-blocks-found-24h',
          'grafana-alerts-by-type',
        ]
      },
      { 
        slotId: 'miner-4', 
        selectedChartId: 'grafana-pending-payouts', 
        allowedCategories: ['earnings-payouts'],
        allowedChartIds: [
          'grafana-pending-payouts',
          'grafana-payouts-processed-24h',
          'grafana-wallet-balance',
        ]
      },
    ],
  },
  admin: {
    dashboardId: 'admin',
    slotCount: 4,
    columns: 2,
    slots: [
      { 
        slotId: 'admin-1', 
        selectedChartId: 'grafana-node-health', 
        allowedCategories: ['system'],
        allowedChartIds: ['grafana-node-health', 'grafana-stratum-connections', 'grafana-api-latency']
      },
      { 
        slotId: 'admin-2', 
        selectedChartId: 'grafana-active-workers-stat', 
        allowedCategories: ['workers-activity'],
        allowedChartIds: ['grafana-active-workers-stat', 'grafana-total-workers', 'grafana-new-connections']
      },
      { 
        slotId: 'admin-3', 
        selectedChartId: 'grafana-payout-failures-by-reason', 
        allowedCategories: ['earnings-payouts'],
        allowedChartIds: ['grafana-payout-failures-by-reason', 'grafana-payouts-by-mode', 'grafana-pending-payouts']
      },
      { 
        slotId: 'admin-4', 
        selectedChartId: 'grafana-alert-timeline', 
        allowedCategories: ['alerts'],
        allowedChartIds: ['grafana-alert-timeline', 'grafana-alert-frequency']
      },
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
      // Hashrate charts
      'grafana-pool-hashrate-stat': 'native-pool-hashrate',
      'grafana-pool-hashrate-history': 'native-pool-hashrate',
      'grafana-avg-hashrate-per-worker': 'native-pool-hashrate',
      // Worker charts
      'grafana-active-workers-stat': 'native-acceptance-rate',
      'grafana-active-workers-history': 'native-acceptance-rate',
      // Share charts
      'grafana-shares-submitted': 'native-shares-submitted',
      'grafana-share-rejection-rate': 'native-shares-submitted',
      // Earnings charts
      'grafana-wallet-balance': 'native-earnings',
      'grafana-pending-payouts': 'native-earnings',
      'grafana-payouts-processed-24h': 'native-earnings',
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
