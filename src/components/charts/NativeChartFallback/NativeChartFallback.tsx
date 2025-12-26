import React from 'react';
import { 
  AreaChart, Area, LineChart, Line, BarChart, Bar,
  XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer 
} from 'recharts';
import { INativeChart, INativeChartProps, NATIVE_CHART_COLORS } from '../interfaces/INativeChart';

/**
 * Grafana-style tooltip styling
 */
const tooltipStyle = {
  contentStyle: { 
    backgroundColor: 'rgba(24, 27, 31, 0.96)', 
    border: '1px solid rgba(255, 255, 255, 0.1)', 
    borderRadius: '4px',
    boxShadow: '0 4px 12px rgba(0, 0, 0, 0.4)',
    padding: '8px 12px',
  },
  labelStyle: { 
    color: 'rgba(204, 204, 220, 0.65)', 
    fontWeight: 400, 
    fontSize: '0.75rem', 
    marginBottom: '2px' 
  },
  itemStyle: { 
    color: '#CCCCDC', 
    fontSize: '0.85rem', 
    fontWeight: 500 
  },
};

/**
 * NativeChartFallback - Recharts-based fallback when Grafana is unavailable
 * Matches Grafana dark theme styling as closely as possible
 */
export const NativeChartFallback: React.FC<INativeChartProps> = ({
  config,
  data,
  loading = false,
  error,
  className,
  style,
}) => {
  const chartHeight = config.height || 280;
  const gradientId = `gradient-${config.id}`;

  const containerStyle: React.CSSProperties = {
    width: '100%',
    height: chartHeight,
    backgroundColor: '#181B1F',
    ...style,
  };

  if (loading) {
    return (
      <div className={className} style={{ ...containerStyle, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
        <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: '8px', color: 'rgba(204, 204, 220, 0.65)' }}>
          <div style={{ 
            width: '24px', 
            height: '24px', 
            border: '2px solid rgba(204, 204, 220, 0.2)', 
            borderTopColor: config.color, 
            borderRadius: '50%', 
            animation: 'spin 1s linear infinite' 
          }} />
          <span style={{ fontSize: '0.8rem' }}>Loading...</span>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className={className} style={{ ...containerStyle, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
        <div style={{ color: '#FF6B6B', textAlign: 'center', padding: '20px' }}>
          <span style={{ display: 'block', marginBottom: '8px' }}>⚠️</span>
          <span style={{ fontSize: '0.85rem' }}>{error}</span>
        </div>
      </div>
    );
  }

  if (!data || data.length === 0) {
    return (
      <div className={className} style={{ ...containerStyle, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
        <span style={{ color: 'rgba(204, 204, 220, 0.5)', fontSize: '0.85rem' }}>No data available</span>
      </div>
    );
  }

  const commonProps = {
    data,
    margin: { top: 10, right: 16, left: 0, bottom: 0 },
  };

  const axisProps = {
    stroke: 'transparent',
    tick: { fill: 'rgba(204, 204, 220, 0.65)', fontSize: 10 },
    tickLine: false,
    axisLine: false,
  };

  const renderChart = () => {
    switch (config.chartType) {
      case 'area':
        return (
          <AreaChart {...commonProps}>
            <defs>
              <linearGradient id={gradientId} x1="0" y1="0" x2="0" y2="1">
                <stop offset="0%" stopColor={config.color} stopOpacity={config.gradientOpacity || 0.25} />
                <stop offset="100%" stopColor={config.color} stopOpacity={0.02} />
              </linearGradient>
            </defs>
            <CartesianGrid stroke="rgba(255, 255, 255, 0.06)" strokeDasharray="0" vertical={false} />
            <XAxis dataKey="time" {...axisProps} dy={8} />
            <YAxis 
              {...axisProps} 
              width={50} 
              dx={-4}
              tickFormatter={config.yAxisFormatter}
            />
            <Tooltip 
              {...tooltipStyle}
              formatter={config.tooltipFormatter ? 
                (value: number) => [config.tooltipFormatter!(value), config.title] : 
                undefined
              }
            />
            <Area
              type="monotone"
              dataKey={config.dataKey}
              stroke={config.color}
              fill={`url(#${gradientId})`}
              strokeWidth={2}
              dot={false}
              activeDot={{ r: 3, fill: config.color, stroke: '#1F2228', strokeWidth: 2 }}
            />
          </AreaChart>
        );

      case 'line':
        return (
          <LineChart {...commonProps}>
            <CartesianGrid stroke="rgba(255, 255, 255, 0.06)" strokeDasharray="0" vertical={false} />
            <XAxis dataKey="time" {...axisProps} dy={8} />
            <YAxis 
              {...axisProps} 
              width={50} 
              dx={-4}
              tickFormatter={config.yAxisFormatter}
            />
            <Tooltip 
              {...tooltipStyle}
              formatter={config.tooltipFormatter ? 
                (value: number) => [config.tooltipFormatter!(value), config.title] : 
                undefined
              }
            />
            <Line
              type="monotone"
              dataKey={config.dataKey}
              stroke={config.color}
              strokeWidth={2}
              dot={false}
              activeDot={{ r: 3, fill: config.color, stroke: '#1F2228', strokeWidth: 2 }}
            />
          </LineChart>
        );

      case 'bar':
        return (
          <BarChart {...commonProps}>
            <defs>
              <linearGradient id={gradientId} x1="0" y1="0" x2="0" y2="1">
                <stop offset="0%" stopColor={config.color} stopOpacity={0.9} />
                <stop offset="100%" stopColor={config.color} stopOpacity={0.6} />
              </linearGradient>
            </defs>
            <CartesianGrid stroke="rgba(255, 255, 255, 0.06)" strokeDasharray="0" vertical={false} />
            <XAxis dataKey="time" {...axisProps} dy={8} />
            <YAxis 
              {...axisProps} 
              width={50} 
              dx={-4}
              tickFormatter={config.yAxisFormatter}
            />
            <Tooltip 
              {...tooltipStyle}
              formatter={config.tooltipFormatter ? 
                (value: number) => [config.tooltipFormatter!(value), config.title] : 
                undefined
              }
            />
            <Bar
              dataKey={config.dataKey}
              fill={`url(#${gradientId})`}
              radius={[2, 2, 0, 0]}
            />
          </BarChart>
        );

      default:
        return null;
    }
  };

  return (
    <div className={className} style={containerStyle} data-testid="native-chart-fallback">
      <ResponsiveContainer width="100%" height={chartHeight}>
        {renderChart() as React.ReactElement}
      </ResponsiveContainer>
    </div>
  );
};

export default NativeChartFallback;
