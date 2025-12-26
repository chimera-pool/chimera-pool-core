import { IChartPanel } from './IChartPanel';

/**
 * Grafana-specific panel configuration
 * Extends base IChartPanel with Grafana embed requirements
 */
export interface IGrafanaPanel extends IChartPanel {
  /** Panel source type */
  type: 'grafana';
  /** Grafana dashboard UID */
  dashboardUid: string;
  /** Panel ID within the dashboard */
  panelId: number;
  /** Organization ID (defaults to 1) */
  orgId?: number;
  /** Theme for embed */
  theme?: 'dark' | 'light';
  /** Relative time range start (e.g., 'now-24h') */
  from?: string;
  /** Relative time range end (e.g., 'now') */
  to?: string;
  /** Timezone for display */
  timezone?: string;
  /** Auto-refresh interval in seconds */
  refreshInterval?: number;
}

/**
 * Props for GrafanaEmbed component
 */
export interface IGrafanaEmbedProps {
  /** Base URL of Grafana instance */
  baseUrl: string;
  /** Panel configuration */
  panel: IGrafanaPanel;
  /** Additional CSS class */
  className?: string;
  /** Inline styles */
  style?: React.CSSProperties;
  /** Callback when iframe loads */
  onLoad?: () => void;
  /** Callback on error */
  onError?: (error: Error) => void;
}

/**
 * Build Grafana solo panel embed URL
 */
export function buildGrafanaEmbedUrl(
  baseUrl: string,
  panel: IGrafanaPanel
): string {
  const params = new URLSearchParams();
  params.set('orgId', String(panel.orgId || 1));
  params.set('panelId', String(panel.panelId));
  params.set('theme', panel.theme || 'dark');
  
  if (panel.from) params.set('from', panel.from);
  if (panel.to) params.set('to', panel.to);
  if (panel.timezone) params.set('timezone', panel.timezone);
  if (panel.refreshInterval) params.set('refresh', `${panel.refreshInterval}s`);

  return `${baseUrl}/d-solo/${panel.dashboardUid}?${params.toString()}`;
}

/**
 * Grafana connection status
 */
export interface IGrafanaStatus {
  available: boolean;
  lastCheck: Date;
  error?: string;
}
