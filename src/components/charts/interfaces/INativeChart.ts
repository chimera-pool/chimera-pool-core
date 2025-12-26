import { IChartPanel } from './IChartPanel';

/**
 * Native chart configuration (Recharts fallback)
 * Used when Grafana is unavailable
 */
export interface INativeChart extends IChartPanel {
  /** Chart source type */
  type: 'native';
  /** Type of chart to render */
  chartType: 'area' | 'line' | 'bar';
  /** Data key for primary metric */
  dataKey: string;
  /** Primary color */
  color: string;
  /** Secondary color (for gradients) */
  colorSecondary?: string;
  /** Gradient opacity (0-1) */
  gradientOpacity?: number;
  /** Y-axis formatter */
  yAxisFormatter?: (value: number) => string;
  /** Tooltip formatter */
  tooltipFormatter?: (value: number) => string;
  /** API endpoint to fetch data */
  apiEndpoint: string;
}

/**
 * Props for native chart component
 */
export interface INativeChartProps {
  /** Chart configuration */
  config: INativeChart;
  /** Chart data */
  data: any[];
  /** Loading state */
  loading?: boolean;
  /** Error message */
  error?: string;
  /** Additional CSS class */
  className?: string;
  /** Inline styles */
  style?: React.CSSProperties;
}

/**
 * Native chart color palette matching Grafana dark theme
 */
export const NATIVE_CHART_COLORS = {
  gold: '#F5B800',
  green: '#73BF69',
  blue: '#5794F2',
  purple: '#B877D9',
  coral: '#FF6B6B',
  silver: '#CCCCDC',
};
