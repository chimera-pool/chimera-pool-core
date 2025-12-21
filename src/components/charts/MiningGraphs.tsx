import React, { useState, useEffect } from 'react';
import { 
  LineChart, Line, AreaChart, Area, BarChart, Bar, 
  XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer 
} from 'recharts';
import { colors, gradients } from '../../styles/shared';

// ============================================================================
// MINING GRAPHS COMPONENT
// Displays hashrate, shares, acceptance rate, and earnings charts
// Supports both pool-wide and personal view modes
// ============================================================================

export type TimeRange = '1h' | '6h' | '24h' | '7d' | '30d' | '3m' | '6m' | '1y' | 'all';
export type ViewMode = 'pool' | 'personal';

export interface MiningGraphsProps {
  token?: string;
  isLoggedIn: boolean;
}

const styles: { [key: string]: React.CSSProperties } = {
  section: {
    background: gradients.card,
    borderRadius: '12px',
    padding: '24px',
    border: `1px solid ${colors.border}`,
    marginBottom: '20px',
  },
  header: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: '20px',
    flexWrap: 'wrap' as const,
    gap: '15px',
  },
  title: {
    fontSize: '1.3rem',
    color: colors.primary,
    margin: 0,
  },
  timeSelector: {
    display: 'flex',
    gap: '5px',
    flexWrap: 'wrap' as const,
  },
  timeBtn: {
    padding: '6px 12px',
    backgroundColor: colors.bgInput,
    border: `1px solid ${colors.border}`,
    borderRadius: '6px',
    color: colors.textSecondary,
    cursor: 'pointer',
    fontSize: '0.85rem',
    transition: 'all 0.2s',
  },
  timeBtnActive: {
    backgroundColor: colors.primary,
    color: colors.bgDark,
    borderColor: colors.primary,
  },
  viewToggle: {
    display: 'flex',
    backgroundColor: colors.bgInput,
    borderRadius: '8px',
    padding: '4px',
    border: `1px solid ${colors.border}`,
  },
  viewBtn: {
    padding: '8px 16px',
    border: 'none',
    borderRadius: '6px',
    cursor: 'pointer',
    fontSize: '0.85rem',
    transition: 'all 0.2s',
  },
  loading: {
    textAlign: 'center',
    padding: '60px',
    color: colors.primary,
  },
  chartsGrid: {
    display: 'grid',
    gridTemplateColumns: 'repeat(auto-fit, minmax(400px, 1fr))',
    gap: '20px',
  },
  chartCard: {
    backgroundColor: colors.bgInput,
    borderRadius: '10px',
    padding: '20px',
    border: `1px solid ${colors.border}`,
  },
  chartTitle: {
    color: colors.primary,
    fontSize: '1rem',
    margin: '0 0 15px',
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

const tooltipStyle = {
  contentStyle: { 
    backgroundColor: colors.bgCard, 
    border: `1px solid ${colors.border}`, 
    borderRadius: '8px' 
  },
  labelStyle: { color: colors.primary },
};

export function MiningGraphs({ token, isLoggedIn }: MiningGraphsProps) {
  const [timeRange, setTimeRange] = useState<TimeRange>('24h');
  const [viewMode, setViewMode] = useState<ViewMode>(isLoggedIn ? 'personal' : 'pool');
  const [hashrateData, setHashrateData] = useState<any[]>([]);
  const [sharesData, setSharesData] = useState<any[]>([]);
  const [earningsData, setEarningsData] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetchAllData();
  }, [timeRange, viewMode]);

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

  const fetchAllData = async () => {
    setLoading(true);
    try {
      if (viewMode === 'personal' && token) {
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
            hashrateMH: d.hashrate / 1000000
          })) || []);
        }
        if (sharesRes.ok) {
          const data = await sharesRes.json();
          setSharesData(data.data?.map((d: any) => ({
            ...d,
            time: new Date(d.time).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
          })) || []);
        }
        if (earningsRes.ok) {
          const data = await earningsRes.json();
          setEarningsData(data.data?.map((d: any) => ({
            ...d,
            time: new Date(d.time).toLocaleDateString([], { month: 'short', day: 'numeric' })
          })) || []);
        }
      } else {
        const [hashRes, sharesRes] = await Promise.all([
          fetch(`/api/v1/pool/stats/hashrate?range=${timeRange}`),
          fetch(`/api/v1/pool/stats/shares?range=${timeRange}`)
        ]);

        if (hashRes.ok) {
          const data = await hashRes.json();
          setHashrateData(data.data?.map((d: any) => ({
            ...d,
            time: new Date(d.time).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
            hashrateMH: d.hashrate / 1000000000
          })) || generateMockPoolData('hashrate'));
        } else {
          setHashrateData(generateMockPoolData('hashrate'));
        }
        if (sharesRes.ok) {
          const data = await sharesRes.json();
          setSharesData(data.data?.map((d: any) => ({
            ...d,
            time: new Date(d.time).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
          })) || generateMockPoolData('shares'));
        } else {
          setSharesData(generateMockPoolData('shares'));
        }
        setEarningsData([]);
      }
    } catch (error) {
      console.error('Failed to fetch graph data:', error);
      if (viewMode === 'pool') {
        setHashrateData(generateMockPoolData('hashrate'));
        setSharesData(generateMockPoolData('shares'));
      }
    } finally {
      setLoading(false);
    }
  };

  return (
    <section style={styles.section}>
      <div style={styles.header}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '15px', flexWrap: 'wrap' as const }}>
          <h2 style={styles.title}>📊 {viewMode === 'pool' ? 'Pool' : 'My'} Mining Statistics</h2>
          {isLoggedIn && (
            <div style={styles.viewToggle}>
              <button
                style={{
                  ...styles.viewBtn,
                  backgroundColor: viewMode === 'pool' ? colors.primary : 'transparent',
                  color: viewMode === 'pool' ? colors.bgDark : colors.textSecondary,
                  fontWeight: viewMode === 'pool' ? 'bold' : 'normal',
                }}
                onClick={() => setViewMode('pool')}
              >
                🌐 Pool
              </button>
              <button
                style={{
                  ...styles.viewBtn,
                  backgroundColor: viewMode === 'personal' ? colors.primary : 'transparent',
                  color: viewMode === 'personal' ? colors.bgDark : colors.textSecondary,
                  fontWeight: viewMode === 'personal' ? 'bold' : 'normal',
                }}
                onClick={() => setViewMode('personal')}
              >
                👤 Personal
              </button>
            </div>
          )}
        </div>
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
      </div>

      {loading ? (
        <div style={styles.loading}>Loading charts...</div>
      ) : (
        <div style={styles.chartsGrid}>
          {/* Hashrate Chart */}
          <div style={styles.chartCard}>
            <h3 style={styles.chartTitle}>⚡ Hashrate History</h3>
            <ResponsiveContainer width="100%" height={250}>
              <AreaChart data={hashrateData}>
                <defs>
                  <linearGradient id="hashGradient" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor={colors.primary} stopOpacity={0.3}/>
                    <stop offset="95%" stopColor={colors.primary} stopOpacity={0}/>
                  </linearGradient>
                </defs>
                <CartesianGrid strokeDasharray="3 3" stroke={colors.border} />
                <XAxis dataKey="time" stroke={colors.textSecondary} fontSize={12} />
                <YAxis stroke={colors.textSecondary} fontSize={12} tickFormatter={(v) => `${v.toFixed(0)} MH/s`} />
                <Tooltip {...tooltipStyle} formatter={(value: number) => [`${value.toFixed(2)} MH/s`, 'Hashrate']} />
                <Area type="monotone" dataKey="hashrateMH" stroke={colors.primary} fill="url(#hashGradient)" strokeWidth={2} />
              </AreaChart>
            </ResponsiveContainer>
          </div>

          {/* Shares Chart */}
          <div style={styles.chartCard}>
            <h3 style={styles.chartTitle}>📦 Shares Submitted</h3>
            <ResponsiveContainer width="100%" height={250}>
              <BarChart data={sharesData}>
                <CartesianGrid strokeDasharray="3 3" stroke={colors.border} />
                <XAxis dataKey="time" stroke={colors.textSecondary} fontSize={12} />
                <YAxis stroke={colors.textSecondary} fontSize={12} />
                <Tooltip {...tooltipStyle} />
                <Legend />
                <Bar dataKey="validShares" name="Valid" fill={colors.success} radius={[4, 4, 0, 0]} />
                <Bar dataKey="invalidShares" name="Invalid" fill={colors.error} radius={[4, 4, 0, 0]} />
              </BarChart>
            </ResponsiveContainer>
          </div>

          {/* Acceptance Rate Chart */}
          <div style={styles.chartCard}>
            <h3 style={styles.chartTitle}>✅ Acceptance Rate</h3>
            <ResponsiveContainer width="100%" height={250}>
              <LineChart data={sharesData}>
                <CartesianGrid strokeDasharray="3 3" stroke={colors.border} />
                <XAxis dataKey="time" stroke={colors.textSecondary} fontSize={12} />
                <YAxis stroke={colors.textSecondary} fontSize={12} domain={[90, 100]} tickFormatter={(v) => `${v}%`} />
                <Tooltip {...tooltipStyle} formatter={(value: number) => [`${value.toFixed(2)}%`, 'Rate']} />
                <Line type="monotone" dataKey="acceptanceRate" stroke={colors.success} strokeWidth={2} dot={false} />
              </LineChart>
            </ResponsiveContainer>
          </div>

          {/* Earnings Chart */}
          <div style={styles.chartCard}>
            <h3 style={styles.chartTitle}>💰 Earnings History</h3>
            <ResponsiveContainer width="100%" height={250}>
              <AreaChart data={earningsData}>
                <defs>
                  <linearGradient id="earnGradient" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="#9b59b6" stopOpacity={0.3}/>
                    <stop offset="95%" stopColor="#9b59b6" stopOpacity={0}/>
                  </linearGradient>
                </defs>
                <CartesianGrid strokeDasharray="3 3" stroke={colors.border} />
                <XAxis dataKey="time" stroke={colors.textSecondary} fontSize={12} />
                <YAxis stroke={colors.textSecondary} fontSize={12} tickFormatter={(v) => `${v.toFixed(2)}`} />
                <Tooltip 
                  {...tooltipStyle}
                  labelStyle={{ color: '#9b59b6' }}
                  formatter={(value: number) => [`${value.toFixed(4)} BDAG`, 'Cumulative']} 
                />
                <Area type="monotone" dataKey="cumulative" stroke="#9b59b6" fill="url(#earnGradient)" strokeWidth={2} />
              </AreaChart>
            </ResponsiveContainer>
          </div>
        </div>
      )}
    </section>
  );
}

export default MiningGraphs;
