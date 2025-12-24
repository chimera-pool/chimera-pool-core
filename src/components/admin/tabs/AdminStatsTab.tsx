/**
 * AdminStatsTab - Isolated Stats Tab Component
 * 
 * ISP Compliant: This component manages its own state via useAdminStatsTab hook.
 * Parent component (AdminPanel) will NOT re-render when this component's state changes.
 * 
 * Uses React.memo to prevent unnecessary re-renders from parent prop changes.
 */

import React, { memo } from 'react';
import {
  ResponsiveContainer,
  AreaChart,
  Area,
  LineChart,
  Line,
  BarChart,
  Bar,
  PieChart,
  Pie,
  Cell,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend
} from 'recharts';
import { useAdminStatsTab, TimeRange } from '../hooks/useAdminStatsTab';

// ============================================================================
// STYLES - Defined outside component to prevent recreation
// ============================================================================

const styles: { [key: string]: React.CSSProperties } = {
  section: { 
    background: 'linear-gradient(135deg, #1a1a2e 0%, #0f0f1a 100%)', 
    borderRadius: '12px', 
    padding: '24px', 
    border: '1px solid #2a2a4a', 
    marginBottom: '20px' 
  },
  header: { 
    display: 'flex', 
    justifyContent: 'space-between', 
    alignItems: 'center', 
    marginBottom: '20px', 
    flexWrap: 'wrap', 
    gap: '15px' 
  },
  title: { 
    fontSize: '1.3rem', 
    color: '#00d4ff', 
    margin: 0 
  },
  controls: { 
    display: 'flex', 
    gap: '15px', 
    alignItems: 'center', 
    flexWrap: 'wrap' 
  },
  timeSelector: { 
    display: 'flex', 
    gap: '8px', 
    flexWrap: 'wrap' 
  },
  timeBtn: { 
    padding: '6px 12px', 
    backgroundColor: '#0a0a15', 
    border: '1px solid #2a2a4a', 
    borderRadius: '6px', 
    color: '#888', 
    cursor: 'pointer', 
    fontSize: '0.85rem',
    transition: 'all 0.2s',
  },
  timeBtnActive: { 
    backgroundColor: '#00d4ff', 
    color: '#0a0a0f', 
    borderColor: '#00d4ff' 
  },
  refreshBtn: {
    padding: '8px 16px',
    backgroundColor: '#0a0a15',
    border: '1px solid #00d4ff',
    borderRadius: '6px',
    color: '#00d4ff',
    cursor: 'pointer',
    fontSize: '0.85rem',
    transition: 'all 0.2s',
  },
  autoRefreshToggle: {
    display: 'flex',
    alignItems: 'center',
    gap: '8px',
    color: '#888',
    fontSize: '0.85rem',
  },
  loading: { 
    textAlign: 'center' as const, 
    padding: '60px', 
    color: '#00d4ff' 
  },
  chartsGrid: { 
    display: 'grid', 
    gridTemplateColumns: 'repeat(auto-fit, minmax(400px, 1fr))', 
    gap: '20px' 
  },
  chartCard: { 
    backgroundColor: '#0a0a15', 
    borderRadius: '10px', 
    padding: '20px', 
    border: '1px solid #2a2a4a' 
  },
  chartTitle: { 
    color: '#e0e0e0', 
    fontSize: '1rem', 
    marginTop: 0, 
    marginBottom: '15px' 
  },
  error: {
    backgroundColor: '#2a1515',
    border: '1px solid #ff4444',
    borderRadius: '8px',
    padding: '15px',
    color: '#ff8888',
    marginBottom: '20px',
  },
};

const TIME_RANGES: { value: TimeRange; label: string }[] = [
  { value: '1h', label: '1H' },
  { value: '6h', label: '6H' },
  { value: '24h', label: '24H' },
  { value: '7d', label: '7D' },
  { value: '30d', label: '30D' },
  { value: '3m', label: '3M' },
  { value: '6m', label: '6M' },
  { value: '1y', label: '1Y' },
];

const CHART_COLORS = {
  primary: '#00d4ff',
  secondary: '#9b59b6',
  success: '#00ff88',
  warning: '#ffaa00',
  error: '#ff4444',
  text: '#888',
  border: '#2a2a4a',
};

const tooltipStyle = {
  contentStyle: { backgroundColor: '#1a1a2e', border: '1px solid #2a2a4a', borderRadius: '8px' },
  labelStyle: { color: '#00d4ff' },
};

// ============================================================================
// PROPS INTERFACE
// ============================================================================

interface AdminStatsTabProps {
  token: string;
  isActive: boolean;
}

// ============================================================================
// COMPONENT - Memoized to prevent parent re-renders from affecting this
// ============================================================================

const AdminStatsTab = memo(function AdminStatsTab({ token, isActive }: AdminStatsTabProps) {
  // All state management is isolated in this hook
  const stats = useAdminStatsTab(token, isActive);

  // Don't render anything if not active (saves resources)
  if (!isActive) return null;

  return (
    <section style={styles.section}>
      {/* Header with controls */}
      <div style={styles.header}>
        <h3 style={styles.title}>📊 Pool Statistics</h3>
        
        <div style={styles.controls}>
          {/* Time Range Selector */}
          <div style={styles.timeSelector}>
            {TIME_RANGES.map(({ value, label }) => (
              <button
                key={value}
                style={{
                  ...styles.timeBtn,
                  ...(stats.timeRange === value ? styles.timeBtnActive : {}),
                }}
                onClick={() => stats.setTimeRange(value)}
              >
                {label}
              </button>
            ))}
          </div>

          {/* Auto-refresh Toggle */}
          <label style={styles.autoRefreshToggle}>
            <input
              type="checkbox"
              checked={stats.isAutoRefreshEnabled}
              onChange={stats.toggleAutoRefresh}
            />
            Auto-refresh
          </label>

          {/* Manual Refresh Button */}
          <button
            style={styles.refreshBtn}
            onClick={stats.refresh}
            disabled={stats.isLoading}
          >
            {stats.isLoading ? '⟳ Refreshing...' : '⟳ Refresh'}
          </button>
        </div>
      </div>

      {/* Error Display */}
      {stats.error && (
        <div style={styles.error}>
          ⚠️ Error loading stats: {stats.error.message}
        </div>
      )}

      {/* Loading State */}
      {stats.isLoading && stats.hashrateData.length === 0 ? (
        <div style={styles.loading}>Loading statistics...</div>
      ) : (
        <div style={styles.chartsGrid}>
          {/* Hashrate Chart */}
          <MemoizedHashrateChart data={stats.hashrateData} />

          {/* Shares Chart */}
          <MemoizedSharesChart data={stats.sharesData} />

          {/* Miners Chart */}
          <MemoizedMinersChart data={stats.minersData} />

          {/* Acceptance Rate Chart */}
          <MemoizedAcceptanceChart data={stats.sharesData} />

          {/* Payouts Chart */}
          {stats.payoutsData.length > 0 && (
            <MemoizedPayoutsChart data={stats.payoutsData} />
          )}

          {/* Distribution Pie Chart */}
          {stats.distributionData.length > 0 && (
            <MemoizedDistributionChart data={stats.distributionData} />
          )}
        </div>
      )}
    </section>
  );
});

// ============================================================================
// MEMOIZED CHART COMPONENTS - Prevent re-renders when data hasn't changed
// ============================================================================

const MemoizedHashrateChart = memo(function HashrateChart({ data }: { data: any[] }) {
  if (data.length === 0) return null;
  
  return (
    <div style={styles.chartCard}>
      <h4 style={styles.chartTitle}>⚡ Pool Hashrate (GH/s)</h4>
      <ResponsiveContainer width="100%" height={250}>
        <AreaChart data={data}>
          <defs>
            <linearGradient id="adminHashGradient" x1="0" y1="0" x2="0" y2="1">
              <stop offset="5%" stopColor={CHART_COLORS.primary} stopOpacity={0.3}/>
              <stop offset="95%" stopColor={CHART_COLORS.primary} stopOpacity={0}/>
            </linearGradient>
          </defs>
          <CartesianGrid strokeDasharray="3 3" stroke={CHART_COLORS.border} />
          <XAxis dataKey="time" stroke={CHART_COLORS.text} fontSize={12} />
          <YAxis stroke={CHART_COLORS.text} fontSize={12} tickFormatter={(v) => `${v.toFixed(0)}`} />
          <Tooltip {...tooltipStyle} formatter={(value: number) => [`${value.toFixed(2)} GH/s`, 'Hashrate']} />
          <Area 
            type="monotone" 
            dataKey="totalGH" 
            stroke={CHART_COLORS.primary} 
            fill="url(#adminHashGradient)" 
            strokeWidth={2} 
          />
        </AreaChart>
      </ResponsiveContainer>
    </div>
  );
});

const MemoizedSharesChart = memo(function SharesChart({ data }: { data: any[] }) {
  if (data.length === 0) return null;
  
  return (
    <div style={styles.chartCard}>
      <h4 style={styles.chartTitle}>📦 Shares Submitted</h4>
      <ResponsiveContainer width="100%" height={250}>
        <BarChart data={data}>
          <CartesianGrid strokeDasharray="3 3" stroke={CHART_COLORS.border} />
          <XAxis dataKey="time" stroke={CHART_COLORS.text} fontSize={12} />
          <YAxis stroke={CHART_COLORS.text} fontSize={12} />
          <Tooltip {...tooltipStyle} />
          <Legend />
          <Bar dataKey="validShares" name="Valid" fill={CHART_COLORS.success} radius={[4, 4, 0, 0]} />
          <Bar dataKey="invalidShares" name="Invalid" fill={CHART_COLORS.error} radius={[4, 4, 0, 0]} />
        </BarChart>
      </ResponsiveContainer>
    </div>
  );
});

const MemoizedMinersChart = memo(function MinersChart({ data }: { data: any[] }) {
  if (data.length === 0) return null;
  
  return (
    <div style={styles.chartCard}>
      <h4 style={styles.chartTitle}>👷 Active Miners</h4>
      <ResponsiveContainer width="100%" height={250}>
        <LineChart data={data}>
          <CartesianGrid strokeDasharray="3 3" stroke={CHART_COLORS.border} />
          <XAxis dataKey="time" stroke={CHART_COLORS.text} fontSize={12} />
          <YAxis stroke={CHART_COLORS.text} fontSize={12} />
          <Tooltip {...tooltipStyle} />
          <Line 
            type="monotone" 
            dataKey="activeMiners" 
            stroke={CHART_COLORS.secondary} 
            strokeWidth={2} 
            dot={false} 
          />
        </LineChart>
      </ResponsiveContainer>
    </div>
  );
});

const MemoizedAcceptanceChart = memo(function AcceptanceChart({ data }: { data: any[] }) {
  if (data.length === 0) return null;
  
  return (
    <div style={styles.chartCard}>
      <h4 style={styles.chartTitle}>✅ Acceptance Rate</h4>
      <ResponsiveContainer width="100%" height={250}>
        <LineChart data={data}>
          <CartesianGrid strokeDasharray="3 3" stroke={CHART_COLORS.border} />
          <XAxis dataKey="time" stroke={CHART_COLORS.text} fontSize={12} />
          <YAxis stroke={CHART_COLORS.text} fontSize={12} domain={[90, 100]} tickFormatter={(v) => `${v}%`} />
          <Tooltip {...tooltipStyle} formatter={(value: number) => [`${value.toFixed(2)}%`, 'Rate']} />
          <Line 
            type="monotone" 
            dataKey="acceptanceRate" 
            stroke={CHART_COLORS.success} 
            strokeWidth={2} 
            dot={false} 
          />
        </LineChart>
      </ResponsiveContainer>
    </div>
  );
});

const MemoizedPayoutsChart = memo(function PayoutsChart({ data }: { data: any[] }) {
  if (data.length === 0) return null;
  
  return (
    <div style={styles.chartCard}>
      <h4 style={styles.chartTitle}>💰 Payouts</h4>
      <ResponsiveContainer width="100%" height={250}>
        <BarChart data={data}>
          <CartesianGrid strokeDasharray="3 3" stroke={CHART_COLORS.border} />
          <XAxis dataKey="time" stroke={CHART_COLORS.text} fontSize={12} />
          <YAxis stroke={CHART_COLORS.text} fontSize={12} />
          <Tooltip {...tooltipStyle} />
          <Bar dataKey="amount" name="Amount" fill={CHART_COLORS.warning} radius={[4, 4, 0, 0]} />
        </BarChart>
      </ResponsiveContainer>
    </div>
  );
});

const MemoizedDistributionChart = memo(function DistributionChart({ data }: { data: any[] }) {
  if (data.length === 0) return null;
  
  return (
    <div style={styles.chartCard}>
      <h4 style={styles.chartTitle}>📊 Hashrate Distribution</h4>
      <ResponsiveContainer width="100%" height={250}>
        <PieChart>
          <Pie
            data={data}
            dataKey="value"
            nameKey="name"
            cx="50%"
            cy="50%"
            innerRadius={60}
            outerRadius={100}
            paddingAngle={2}
          >
            {data.map((entry, index) => (
              <Cell key={`cell-${index}`} fill={entry.color} />
            ))}
          </Pie>
          <Tooltip {...tooltipStyle} />
          <Legend />
        </PieChart>
      </ResponsiveContainer>
    </div>
  );
});

export default AdminStatsTab;
