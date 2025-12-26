// Chimera Pool - Chart Components
// ============================================================================
// GRAFANA INTEGRATION WITH NATIVE FALLBACK
// ============================================================================

// Core components
export { GrafanaEmbed } from './GrafanaEmbed';
export { ChartSelector } from './ChartSelector';
export { ChartSlot } from './ChartSlot';
export { NativeChartFallback } from './NativeChartFallback';
export { GrafanaDashboard } from './GrafanaDashboard';
export type { GrafanaDashboardProps } from './GrafanaDashboard';

// Legacy component (kept as additional fallback)
export { MiningGraphs } from './MiningGraphs';
export type { MiningGraphsProps, TimeRange, ViewMode } from './MiningGraphs';

// Interfaces
export * from './interfaces';

// Registry
export { chartRegistry, GRAFANA_CONFIG, DEFAULT_LAYOUTS } from './registry';

// Hooks
export { useChartPreferences, useGrafanaHealth } from './hooks';
