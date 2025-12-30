import React, { useState, useEffect, useCallback } from 'react';
import { ChartSlot } from '../ChartSlot';
import { MiningGraphs } from '../MiningGraphs';
import { chartRegistry, GRAFANA_CONFIG, DEFAULT_LAYOUTS } from '../registry';
import { useGrafanaHealth } from '../hooks/useGrafanaHealth';
import { useChartPreferences } from '../hooks/useChartPreferences';
import { ChartCategory } from '../interfaces/IChartPanel';

// Time range options for Grafana
type GrafanaTimeRange = '1h' | '6h' | '24h' | '7d' | '30d' | '3m' | '6m' | '1y' | 'all';

const TIME_RANGE_OPTIONS: { value: GrafanaTimeRange; label: string; from: string }[] = [
  { value: '1h', label: '1H', from: 'now-1h' },
  { value: '6h', label: '6H', from: 'now-6h' },
  { value: '24h', label: '24H', from: 'now-24h' },
  { value: '7d', label: '7D', from: 'now-7d' },
  { value: '30d', label: '30D', from: 'now-30d' },
  { value: '3m', label: '3M', from: 'now-90d' },
  { value: '6m', label: '6M', from: 'now-180d' },
  { value: '1y', label: '1Y', from: 'now-1y' },
  { value: 'all', label: 'All', from: 'now-5y' },
];

/**
 * Props for GrafanaDashboard
 */
export interface GrafanaDashboardProps {
  dashboardId: 'main' | 'miner' | 'admin';
  token?: string;
  isLoggedIn?: boolean;
  showSelectors?: boolean;
  fallbackData?: {
    hashrate?: any[];
    shares?: any[];
    earnings?: any[];
  };
}

/**
 * GrafanaDashboard - Displays chart slots with Grafana panels or native fallback
 * Replaces MiningGraphs with selectable Grafana-embedded charts
 */
export const GrafanaDashboard: React.FC<GrafanaDashboardProps> = ({
  dashboardId,
  token,
  isLoggedIn = false,
  showSelectors = true,
  fallbackData,
}) => {
  const grafanaHealth = useGrafanaHealth(GRAFANA_CONFIG.baseUrl);
  // Default to Grafana charts - native charts are fallback only
  const [useFallback, setUseFallback] = useState(false);
  const { getSlotSelection } = useChartPreferences();
  
  // Time range state for Grafana panels
  const [timeRange, setTimeRange] = useState<GrafanaTimeRange>('24h');
  const currentTimeOption = TIME_RANGE_OPTIONS.find(t => t.value === timeRange) || TIME_RANGE_OPTIONS[2];

  // Get layout configuration for this dashboard
  const layout = DEFAULT_LAYOUTS[dashboardId];

  // Track selected charts across all slots to prevent duplicates
  const [slotSelections, setSlotSelections] = useState<Record<string, string>>(() => {
    const initial: Record<string, string> = {};
    layout?.slots.forEach(slot => {
      const saved = getSlotSelection(dashboardId, slot.slotId);
      initial[slot.slotId] = saved || slot.selectedChartId || '';
    });
    return initial;
  });

  // Handle chart selection change from a slot
  const handleSlotSelectionChange = useCallback((slotId: string, chartId: string) => {
    setSlotSelections(prev => ({ ...prev, [slotId]: chartId }));
  }, []);

  // Get excluded chart IDs for a specific slot (all other slots' selections)
  const getExcludedChartsForSlot = useCallback((slotId: string): string[] => {
    return Object.entries(slotSelections)
      .filter(([id, chartId]) => id !== slotId && chartId)
      .map(([, chartId]) => chartId);
  }, [slotSelections]);

  // Only fall back to native charts if Grafana is unavailable
  useEffect(() => {
    if (!grafanaHealth.available) {
      setUseFallback(true);
    }
  }, [grafanaHealth.available]);

  const containerStyle: React.CSSProperties = {
    background: 'linear-gradient(180deg, rgba(45, 31, 61, 0.5) 0%, rgba(26, 15, 30, 0.7) 100%)',
    borderRadius: '16px',
    padding: '20px',
    marginBottom: '24px',
    border: '1px solid rgba(74, 44, 90, 0.4)',
    boxShadow: '0 4px 24px rgba(0, 0, 0, 0.3)',
  };

  const headerStyle: React.CSSProperties = {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: '16px',
    flexWrap: 'wrap',
    gap: '12px',
  };

  const titleStyle: React.CSSProperties = {
    fontSize: '1.2rem',
    color: '#F0EDF4',
    margin: 0,
    fontWeight: 700,
    letterSpacing: '0.01em',
  };

  const statusStyle: React.CSSProperties = {
    display: 'flex',
    alignItems: 'center',
    gap: '8px',
    fontSize: '0.8rem',
    color: grafanaHealth.available ? '#4ADE80' : '#FF6B6B',
    fontWeight: 500,
  };

  // Responsive grid - uses CSS class for media query support
  const gridClassName = 'grafana-dashboard-grid';

  const fallbackButtonStyle: React.CSSProperties = {
    padding: '8px 14px',
    backgroundColor: useFallback ? 'rgba(74, 222, 128, 0.1)' : 'rgba(212, 168, 75, 0.1)',
    border: `1px solid ${useFallback ? 'rgba(74, 222, 128, 0.3)' : 'rgba(212, 168, 75, 0.3)'}`,
    borderRadius: '8px',
    color: useFallback ? '#4ADE80' : '#D4A84B',
    cursor: 'pointer',
    fontSize: '0.75rem',
    fontWeight: 500,
    display: 'flex',
    alignItems: 'center',
    gap: '8px',
    transition: 'all 0.2s ease',
  };

  // If using fallback mode, render the legacy MiningGraphs component
  if (useFallback && dashboardId === 'main') {
    return (
      <div style={containerStyle}>
        <div style={headerStyle}>
          <h2 style={titleStyle}>📊 Pool Mining Statistics</h2>
          <button 
            style={fallbackButtonStyle}
            onClick={() => {
              setUseFallback(false);
              grafanaHealth.refresh();
            }}
          >
            <span style={{ width: '8px', height: '8px', borderRadius: '50%', backgroundColor: '#73BF69' }} />
            Switch to Grafana Charts
          </button>
        </div>
        <MiningGraphs token={token} isLoggedIn={isLoggedIn} />
      </div>
    );
  }

  const timeRangeBtnStyle: React.CSSProperties = {
    padding: '6px 10px',
    backgroundColor: 'rgba(31, 20, 40, 0.8)',
    border: '1px solid #4A2C5A',
    borderRadius: '6px',
    color: '#B8B4C8',
    cursor: 'pointer',
    fontSize: '0.75rem',
    fontWeight: 500,
    transition: 'all 0.15s ease',
  };

  const timeRangeBtnActiveStyle: React.CSSProperties = {
    padding: '6px 10px',
    backgroundColor: '#D4A84B',
    background: 'linear-gradient(135deg, #D4A84B 0%, #B8923A 100%)',
    border: '1px solid #D4A84B',
    borderRadius: '6px',
    color: '#1A0F1E',
    cursor: 'pointer',
    fontSize: '0.75rem',
    fontWeight: 600,
    boxShadow: '0 0 12px rgba(212, 168, 75, 0.4)',
    transition: 'all 0.15s ease',
  };

  return (
    <div style={containerStyle}>
      <div style={headerStyle}>
        <h2 style={titleStyle}>📊 Pool Mining Statistics</h2>
        <div style={{ display: 'flex', alignItems: 'center', gap: '16px', flexWrap: 'wrap' }}>
          {/* Time Range Selector */}
          <div style={{ display: 'flex', gap: '4px', flexWrap: 'wrap' }} data-testid="grafana-time-selector">
            {TIME_RANGE_OPTIONS.map((option) => (
              <button
                key={option.value}
                style={timeRange === option.value ? timeRangeBtnActiveStyle : timeRangeBtnStyle}
                onClick={() => setTimeRange(option.value)}
                data-testid={`time-range-btn-${option.value}`}
              >
                {option.label}
              </button>
            ))}
          </div>
          <div style={statusStyle}>
            <span style={{ 
              width: '8px', 
              height: '8px', 
              borderRadius: '50%', 
              backgroundColor: grafanaHealth.available ? '#73BF69' : '#FF6B6B' 
            }} />
            {grafanaHealth.available ? 'Grafana Connected' : 'Connecting...'}
          </div>
          <button 
            style={fallbackButtonStyle}
            onClick={() => setUseFallback(true)}
          >
            <span style={{ width: '8px', height: '8px', borderRadius: '50%', backgroundColor: '#73BF69' }} />
            Switch to Native Charts
          </button>
        </div>
      </div>

      <div className={gridClassName}>
        {layout?.slots.map((slot, index) => (
          <ChartSlot
            key={`${slot.slotId}-${timeRange}`}
            slotId={slot.slotId}
            dashboardId={dashboardId}
            initialChartId={slotSelections[slot.slotId] || slot.selectedChartId}
            allowedCategories={slot.allowedCategories as ChartCategory[]}
            allowedChartIds={slot.allowedChartIds}
            excludedChartIds={getExcludedChartsForSlot(slot.slotId)}
            onSelectionChange={(chartId) => handleSlotSelectionChange(slot.slotId, chartId)}
            showSelector={showSelectors}
            grafanaBaseUrl={GRAFANA_CONFIG.baseUrl}
            grafanaAvailable={grafanaHealth.available}
            grafanaTimeFrom={currentTimeOption.from}
            grafanaTimeTo="now"
            height={280}
          />
        ))}
      </div>
    </div>
  );
};

export default GrafanaDashboard;
