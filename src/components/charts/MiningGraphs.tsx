import React, { useState, useEffect, useCallback } from 'react';
import { useAutoRefresh, REFRESH_INTERVALS } from '../../hooks/useAutoRefresh';
import { 
  LineChart, Line, AreaChart, Area, BarChart, Bar, 
  XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer 
} from 'recharts';
import { colors, gradients, shadows, transitions } from '../../styles/shared';
import { useDashboardGraphs } from '../../services/realtime/useRealTimeData';

// ============================================================================
// MINING GRAPHS COMPONENT - GRAFANA-QUALITY ELITE DESIGN
// Professional dark theme with smooth gradients and premium styling
// ============================================================================

export type TimeRange = '1h' | '6h' | '24h' | '7d' | '30d' | '3m' | '6m' | '1y' | 'all';
export type ViewMode = 'pool' | 'personal';

export interface MiningGraphsProps {
  token?: string;
  isLoggedIn: boolean;
}

// Grafana-inspired chart colors
const chartColors = {
  gold: '#D4A84B',
  goldLight: '#E8C171',
  purple: '#7B5EA7',
  purpleLight: '#9B7EC7',
  green: '#4ADE80',
  greenDark: '#22C55E',
  coral: '#C45C5C',
  coralLight: '#E07777',
  blue: '#60A5FA',
  silver: '#B8B4C8',
  gridLine: 'rgba(74, 44, 90, 0.3)',
  axisText: '#7A7490',
  background: '#0D0811',
  cardBg: 'rgba(45, 31, 61, 0.4)',
};

const styles: { [key: string]: React.CSSProperties } = {
  section: {
    background: 'linear-gradient(180deg, rgba(45, 31, 61, 0.6) 0%, rgba(26, 15, 30, 0.8) 100%)',
    borderRadius: '16px',
    padding: '24px',
    border: '1px solid #4A2C5A',
    marginBottom: '24px',
    boxShadow: '0 4px 24px rgba(0, 0, 0, 0.3)',
  },
  header: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: '24px',
    flexWrap: 'wrap' as const,
    gap: '16px',
  },
  title: {
    fontSize: '1.15rem',
    color: '#F0EDF4',
    margin: 0,
    fontWeight: 600,
    letterSpacing: '0.01em',
  },
  timeSelector: {
    display: 'flex',
    gap: '4px',
    flexWrap: 'wrap' as const,
  },
  timeBtn: {
    padding: '6px 12px',
    backgroundColor: 'rgba(31, 20, 40, 0.8)',
    border: '1px solid #4A2C5A',
    borderRadius: '6px',
    color: '#7A7490',
    cursor: 'pointer',
    fontSize: '0.8rem',
    fontWeight: 500,
    transition: 'all 0.15s ease',
  },
  timeBtnActive: {
    background: 'linear-gradient(135deg, #D4A84B 0%, #B8923A 100%)',
    color: '#1A0F1E',
    borderColor: '#D4A84B',
    boxShadow: '0 0 12px rgba(212, 168, 75, 0.3)',
  },
  viewToggle: {
    display: 'flex',
    backgroundColor: 'rgba(31, 20, 40, 0.8)',
    borderRadius: '10px',
    padding: '3px',
    border: '1px solid #4A2C5A',
  },
  viewBtn: {
    padding: '8px 16px',
    border: 'none',
    borderRadius: '8px',
    cursor: 'pointer',
    fontSize: '0.85rem',
    fontWeight: 500,
    transition: 'all 0.15s ease',
    backgroundColor: 'transparent',
    color: '#B8B4C8',
  },
  loading: {
    textAlign: 'center',
    padding: '80px',
    color: '#D4A84B',
    fontSize: '0.95rem',
  },
  chartsGrid: {
    display: 'grid',
    gridTemplateColumns: 'repeat(auto-fit, minmax(420px, 1fr))',
    gap: '20px',
  },
  chartCard: {
    background: 'linear-gradient(180deg, rgba(13, 8, 17, 0.9) 0%, rgba(26, 15, 30, 0.95) 100%)',
    borderRadius: '12px',
    padding: '20px',
    border: '1px solid rgba(74, 44, 90, 0.5)',
    boxShadow: '0 2px 16px rgba(0, 0, 0, 0.3)',
  },
  chartTitle: {
    color: '#F0EDF4',
    fontSize: '0.95rem',
    fontWeight: 600,
    margin: '0 0 16px',
    display: 'flex',
    alignItems: 'center',
    gap: '8px',
  },
  chartTitleIcon: {
    width: '18px',
    height: '18px',
    borderRadius: '4px',
    display: 'inline-flex',
    alignItems: 'center',
    justifyContent: 'center',
    fontSize: '0.7rem',
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
  { value: 'all', label: 'All' },
];

// Grafana-style tooltip
const tooltipStyle = {
  contentStyle: { 
    backgroundColor: 'rgba(13, 8, 17, 0.95)', 
    border: '1px solid #4A2C5A', 
    borderRadius: '8px',
    boxShadow: '0 4px 20px rgba(0, 0, 0, 0.5)',
    padding: '12px 16px',
  },
  labelStyle: { color: '#D4A84B', fontWeight: 600, marginBottom: '4px' },
  itemStyle: { color: '#F0EDF4', fontSize: '0.9rem' },
};

export function MiningGraphs({ token, isLoggedIn }: MiningGraphsProps) {
  // Use unified real-time data for pool-wide statistics
  const dashboardData = useDashboardGraphs();
  
  const [viewMode, setViewMode] = useState<ViewMode>(isLoggedIn ? 'personal' : 'pool');
  const [hashrateData, setHashrateData] = useState<any[]>([]);
  const [sharesData, setSharesData] = useState<any[]>([]);
  const [earningsData, setEarningsData] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [refreshInterval, setRefreshInterval] = useState(REFRESH_INTERVALS.FAST);

  // Use unified time range from context for pool view
  const timeRange = dashboardData.timeRange;
  const setTimeRange = dashboardData.setTimeRange;

  // Auto-refresh controls from unified context
  const autoRefresh = {
    isActive: dashboardData.isAutoRefreshEnabled,
    toggle: dashboardData.toggleAutoRefresh,
    refresh: dashboardData.refresh,
    isRefreshing: dashboardData.isLoading,
    nextRefreshIn: 10, // Default countdown - actual timing managed by context
  };

  // Direct fetch for pool data - fallback that always works
  const fetchPoolData = async () => {
    setLoading(true);
    try {
      const [hashRes, sharesRes, minersRes] = await Promise.all([
        fetch(`/api/v1/pool/stats/hashrate?range=${timeRange}`),
        fetch(`/api/v1/pool/stats/shares?range=${timeRange}`),
        fetch(`/api/v1/pool/stats/miners?range=${timeRange}`)
      ]);

      if (hashRes.ok) {
        const data = await hashRes.json();
        setHashrateData((data.data || []).map((d: any) => ({
          time: new Date(d.time).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
          hashrateTH: (d.hashrate || 0) / 1e12,
          hashrateMH: (d.hashrate || 0) / 1e6,
        })));
      }
      if (sharesRes.ok) {
        const data = await sharesRes.json();
        setSharesData((data.data || []).map((d: any) => {
          const valid = d.validShares || d.valid || 0;
          const invalid = d.invalidShares || d.invalid || 0;
          const total = valid + invalid;
          return {
            time: new Date(d.time).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
            validShares: valid,
            invalidShares: invalid,
            acceptanceRate: total > 0 ? (valid / total) * 100 : 100,
          };
        }));
      }
    } catch (error) {
      console.error('Failed to fetch pool data:', error);
    } finally {
      setLoading(false);
    }
  };

  // Fetch pool data when in pool view mode
  useEffect(() => {
    if (viewMode === 'pool') {
      fetchPoolData();
    }
  }, [viewMode, timeRange]);

  // Fetch personal data when in personal view mode
  useEffect(() => {
    if (viewMode === 'personal' && token) {
      fetchPersonalData();
    }
  }, [timeRange, viewMode, token]);

  const generateMockPoolData = (type: string) => {
    const now = new Date();
    const data = [];
    for (let i = 23; i >= 0; i--) {
      const time = new Date(now.getTime() - i * 3600000);
      if (type === 'hashrate') {
        data.push({
          time: time.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
          hashrateMH: 150 + Math.random() * 50
        });
      } else {
        const valid = Math.floor(50000 + Math.random() * 10000);
        const invalid = Math.floor(valid * 0.005);
        data.push({
          time: time.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
          validShares: valid,
          invalidShares: invalid,
          acceptanceRate: (valid / (valid + invalid)) * 100
        });
      }
    }
    return data;
  };

  // Fetch personal data (user-specific stats) - pool data comes from unified context
  const fetchPersonalData = async () => {
    if (!token) return;
    
    setLoading(true);
    try {
      const headers = { 'Authorization': `Bearer ${token}` };
      const [hashRes, sharesRes, earningsRes] = await Promise.all([
        fetch(`/api/v1/user/stats/hashrate?range=${timeRange}`, { headers }),
        fetch(`/api/v1/user/stats/shares?range=${timeRange}`, { headers }),
        fetch(`/api/v1/user/stats/earnings?range=${timeRange === '1h' || timeRange === '6h' ? '24h' : timeRange}`, { headers })
      ]);

      if (hashRes.ok) {
        const data = await hashRes.json();
        setHashrateData(data.data?.map((d: any) => ({
          ...d,
          time: new Date(d.time).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
          hashrateMH: d.hashrate / 1000000,
          hashrateTH: d.hashrate / 1000000000000
        })) || []);
      }
      if (sharesRes.ok) {
        const data = await sharesRes.json();
        setSharesData(data.data?.map((d: any) => ({
          ...d,
          time: new Date(d.time).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
          validShares: d.valid || d.validShares || 0,
          invalidShares: d.invalid || d.invalidShares || 0,
          acceptanceRate: d.acceptanceRate || d.acceptance_rate || 100
        })) || []);
      }
      if (earningsRes.ok) {
        const data = await earningsRes.json();
        setEarningsData(data.data?.map((d: any) => ({
          ...d,
          time: new Date(d.time).toLocaleDateString([], { month: 'short', day: 'numeric' })
        })) || []);
      }
    } catch (error) {
      console.error('Failed to fetch personal graph data:', error);
    } finally {
      setLoading(false);
    }
  };

  return (
    <section style={styles.section}>
      <div style={styles.header}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '16px', flexWrap: 'wrap' as const }}>
          <h2 style={styles.title}>{viewMode === 'pool' ? 'Pool' : 'Personal'} Mining Statistics</h2>
          {isLoggedIn && (
            <div style={styles.viewToggle}>
              <button
                style={{
                  ...styles.viewBtn,
                  background: viewMode === 'pool' ? 'linear-gradient(135deg, #D4A84B 0%, #B8923A 100%)' : 'transparent',
                  color: viewMode === 'pool' ? '#1A0F1E' : '#B8B4C8',
                  fontWeight: viewMode === 'pool' ? 600 : 500,
                }}
                onClick={() => setViewMode('pool')}
              >
                Pool
              </button>
              <button
                style={{
                  ...styles.viewBtn,
                  background: viewMode === 'personal' ? 'linear-gradient(135deg, #D4A84B 0%, #B8923A 100%)' : 'transparent',
                  color: viewMode === 'personal' ? '#1A0F1E' : '#B8B4C8',
                  fontWeight: viewMode === 'personal' ? 600 : 500,
                }}
                onClick={() => setViewMode('personal')}
              >
                Personal
              </button>
            </div>
          )}
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: '12px', flexWrap: 'wrap' as const }}>
          <div style={styles.timeSelector}>
            {TIME_RANGES.map(({ value, label }) => (
              <button
                key={value}
                style={{
                  ...styles.timeBtn,
                  ...(timeRange === value ? styles.timeBtnActive : {})
                }}
                onClick={() => setTimeRange(value)}
              >
                {label}
              </button>
            ))}
          </div>
          
          {/* Auto-Refresh Controls - Grafana-style */}
          <div style={{ display: 'flex', alignItems: 'center', gap: '6px', padding: '4px 10px', backgroundColor: 'rgba(31, 20, 40, 0.8)', borderRadius: '8px', border: '1px solid #4A2C5A' }}>
            <span style={{ 
              display: 'inline-flex', 
              alignItems: 'center', 
              gap: '4px',
              color: autoRefresh.isActive ? chartColors.coral : '#7A7490', 
              fontSize: '0.75rem',
              fontWeight: 600,
            }}>
              <span style={{ 
                width: '6px', 
                height: '6px', 
                borderRadius: '50%', 
                backgroundColor: autoRefresh.isActive ? chartColors.coral : '#7A7490',
                boxShadow: autoRefresh.isActive ? '0 0 8px rgba(196, 92, 92, 0.6)' : 'none',
                animation: autoRefresh.isActive ? 'pulse 1.5s infinite' : 'none',
              }} />
              {autoRefresh.isActive ? 'LIVE' : 'PAUSED'}
            </span>
            <button
              onClick={autoRefresh.toggle}
              style={{
                padding: '4px 10px',
                backgroundColor: autoRefresh.isActive ? 'rgba(196, 92, 92, 0.15)' : 'rgba(74, 44, 90, 0.5)',
                border: `1px solid ${autoRefresh.isActive ? 'rgba(196, 92, 92, 0.3)' : '#4A2C5A'}`,
                borderRadius: '4px',
                color: autoRefresh.isActive ? chartColors.coral : '#B8B4C8',
                cursor: 'pointer',
                fontSize: '0.75rem',
                fontWeight: 500,
              }}
            >
              {autoRefresh.isActive ? 'Pause' : 'Start'}
            </button>
            <button
              onClick={() => autoRefresh.refresh()}
              disabled={autoRefresh.isRefreshing}
              style={{
                padding: '4px 8px',
                background: 'linear-gradient(135deg, #D4A84B 0%, #B8923A 100%)',
                border: 'none',
                borderRadius: '4px',
                color: '#1A0F1E',
                cursor: 'pointer',
                fontSize: '0.75rem',
                fontWeight: 600,
                opacity: autoRefresh.isRefreshing ? 0.5 : 1,
              }}
            >
              ↻
            </button>
            {autoRefresh.isActive && (
              <span style={{ color: '#7A7490', fontSize: '0.7rem', fontWeight: 500 }}>
                {autoRefresh.nextRefreshIn}s
              </span>
            )}
          </div>
        </div>
      </div>

      {loading ? (
        <div style={styles.loading}>
          <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: '12px' }}>
            <div style={{ width: '40px', height: '40px', border: '3px solid #4A2C5A', borderTopColor: '#D4A84B', borderRadius: '50%', animation: 'spin 1s linear infinite' }} />
            <span>Loading charts...</span>
          </div>
        </div>
      ) : (
        <div style={styles.chartsGrid}>
          {/* Hashrate Chart - Grafana Style */}
          <div style={styles.chartCard}>
            <h3 style={styles.chartTitle}>
              <span style={{ ...styles.chartTitleIcon, backgroundColor: 'rgba(212, 168, 75, 0.2)', color: chartColors.gold }}>⚡</span>
              Hashrate History
            </h3>
            <ResponsiveContainer width="100%" height={220}>
              <AreaChart data={hashrateData} margin={{ top: 5, right: 10, left: 0, bottom: 5 }}>
                <defs>
                  <linearGradient id="hashGradient" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="0%" stopColor={chartColors.gold} stopOpacity={0.4}/>
                    <stop offset="50%" stopColor={chartColors.gold} stopOpacity={0.15}/>
                    <stop offset="100%" stopColor={chartColors.gold} stopOpacity={0}/>
                  </linearGradient>
                </defs>
                <CartesianGrid strokeDasharray="0" stroke={chartColors.gridLine} vertical={false} />
                <XAxis 
                  dataKey="time" 
                  stroke={chartColors.axisText} 
                  fontSize={11} 
                  tickLine={false}
                  axisLine={{ stroke: chartColors.gridLine }}
                />
                <YAxis 
                  stroke={chartColors.axisText} 
                  fontSize={11} 
                  tickFormatter={(v) => `${v.toFixed(1)}TH/s`} 
                  tickLine={false}
                  axisLine={false}
                  width={70}
                />
                <Tooltip {...tooltipStyle} formatter={(value: number) => [`${value.toFixed(2)} TH/s`, 'Hashrate']} />
                <Area 
                  type="monotoneX" 
                  dataKey="hashrateTH" 
                  stroke={chartColors.gold} 
                  fill="url(#hashGradient)" 
                  strokeWidth={2}
                  dot={false}
                  activeDot={{ r: 4, fill: chartColors.gold, stroke: '#1A0F1E', strokeWidth: 2 }}
                />
              </AreaChart>
            </ResponsiveContainer>
          </div>

          {/* Shares Chart - Grafana Style */}
          <div style={styles.chartCard}>
            <h3 style={styles.chartTitle}>
              <span style={{ ...styles.chartTitleIcon, backgroundColor: 'rgba(74, 222, 128, 0.2)', color: chartColors.green }}>▤</span>
              Shares Submitted
            </h3>
            <ResponsiveContainer width="100%" height={220}>
              <BarChart data={sharesData} margin={{ top: 5, right: 10, left: 0, bottom: 5 }}>
                <defs>
                  <linearGradient id="validGradient" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="0%" stopColor={chartColors.green} stopOpacity={0.9}/>
                    <stop offset="100%" stopColor={chartColors.greenDark} stopOpacity={0.7}/>
                  </linearGradient>
                  <linearGradient id="invalidGradient" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="0%" stopColor={chartColors.coral} stopOpacity={0.9}/>
                    <stop offset="100%" stopColor={chartColors.coral} stopOpacity={0.7}/>
                  </linearGradient>
                </defs>
                <CartesianGrid strokeDasharray="0" stroke={chartColors.gridLine} vertical={false} />
                <XAxis 
                  dataKey="time" 
                  stroke={chartColors.axisText} 
                  fontSize={11}
                  tickLine={false}
                  axisLine={{ stroke: chartColors.gridLine }}
                />
                <YAxis 
                  stroke={chartColors.axisText} 
                  fontSize={11}
                  tickLine={false}
                  axisLine={false}
                  width={50}
                />
                <Tooltip {...tooltipStyle} />
                <Legend 
                  wrapperStyle={{ paddingTop: '10px' }}
                  formatter={(value) => <span style={{ color: '#B8B4C8', fontSize: '0.8rem' }}>{value}</span>}
                />
                <Bar dataKey="validShares" name="Valid" fill="url(#validGradient)" radius={[3, 3, 0, 0]} />
                <Bar dataKey="invalidShares" name="Invalid" fill="url(#invalidGradient)" radius={[3, 3, 0, 0]} />
              </BarChart>
            </ResponsiveContainer>
          </div>

          {/* Acceptance Rate Chart - Grafana Style */}
          <div style={styles.chartCard}>
            <h3 style={styles.chartTitle}>
              <span style={{ ...styles.chartTitleIcon, backgroundColor: 'rgba(74, 222, 128, 0.2)', color: chartColors.green }}>✓</span>
              Acceptance Rate
            </h3>
            <ResponsiveContainer width="100%" height={220}>
              <AreaChart data={sharesData} margin={{ top: 5, right: 10, left: 0, bottom: 5 }}>
                <defs>
                  <linearGradient id="acceptGradient" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="0%" stopColor={chartColors.green} stopOpacity={0.3}/>
                    <stop offset="100%" stopColor={chartColors.green} stopOpacity={0}/>
                  </linearGradient>
                </defs>
                <CartesianGrid strokeDasharray="0" stroke={chartColors.gridLine} vertical={false} />
                <XAxis 
                  dataKey="time" 
                  stroke={chartColors.axisText} 
                  fontSize={11}
                  tickLine={false}
                  axisLine={{ stroke: chartColors.gridLine }}
                />
                <YAxis 
                  stroke={chartColors.axisText} 
                  fontSize={11} 
                  domain={[90, 100]} 
                  tickFormatter={(v) => `${v}%`}
                  tickLine={false}
                  axisLine={false}
                  width={45}
                />
                <Tooltip {...tooltipStyle} formatter={(value: number) => [`${value.toFixed(2)}%`, 'Rate']} />
                <Area 
                  type="monotoneX" 
                  dataKey="acceptanceRate" 
                  stroke={chartColors.green} 
                  fill="url(#acceptGradient)"
                  strokeWidth={2} 
                  dot={false}
                  activeDot={{ r: 4, fill: chartColors.green, stroke: '#1A0F1E', strokeWidth: 2 }}
                />
              </AreaChart>
            </ResponsiveContainer>
          </div>

          {/* Earnings Chart - Grafana Style */}
          <div style={styles.chartCard}>
            <h3 style={styles.chartTitle}>
              <span style={{ ...styles.chartTitleIcon, backgroundColor: 'rgba(123, 94, 167, 0.2)', color: chartColors.purple }}>◈</span>
              Earnings History
            </h3>
            <ResponsiveContainer width="100%" height={220}>
              <AreaChart data={earningsData} margin={{ top: 5, right: 10, left: 0, bottom: 5 }}>
                <defs>
                  <linearGradient id="earnGradient" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="0%" stopColor={chartColors.purple} stopOpacity={0.4}/>
                    <stop offset="50%" stopColor={chartColors.purple} stopOpacity={0.15}/>
                    <stop offset="100%" stopColor={chartColors.purple} stopOpacity={0}/>
                  </linearGradient>
                </defs>
                <CartesianGrid strokeDasharray="0" stroke={chartColors.gridLine} vertical={false} />
                <XAxis 
                  dataKey="time" 
                  stroke={chartColors.axisText} 
                  fontSize={11}
                  tickLine={false}
                  axisLine={{ stroke: chartColors.gridLine }}
                />
                <YAxis 
                  stroke={chartColors.axisText} 
                  fontSize={11} 
                  tickFormatter={(v) => `${v.toFixed(2)}`}
                  tickLine={false}
                  axisLine={false}
                  width={50}
                />
                <Tooltip 
                  {...tooltipStyle}
                  labelStyle={{ color: chartColors.purple, fontWeight: 600 }}
                  formatter={(value: number) => [`${value.toFixed(4)} BDAG`, 'Cumulative']} 
                />
                <Area 
                  type="monotoneX" 
                  dataKey="cumulative" 
                  stroke={chartColors.purple} 
                  fill="url(#earnGradient)" 
                  strokeWidth={2}
                  dot={false}
                  activeDot={{ r: 4, fill: chartColors.purple, stroke: '#1A0F1E', strokeWidth: 2 }}
                />
              </AreaChart>
            </ResponsiveContainer>
          </div>
        </div>
      )}
    </section>
  );
}

export default MiningGraphs;
