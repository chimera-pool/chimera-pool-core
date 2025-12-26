import React, { useState, useEffect, useCallback } from 'react';
import { ChartSlot } from '../ChartSlot';
import { MiningGraphs } from '../MiningGraphs';
import { chartRegistry, GRAFANA_CONFIG, DEFAULT_LAYOUTS } from '../registry';
import { useGrafanaHealth } from '../hooks/useGrafanaHealth';
import { useChartPreferences } from '../hooks/useChartPreferences';
import { ChartCategory } from '../interfaces/IChartPanel';

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

  return (
    <div style={containerStyle}>
      <div style={headerStyle}>
        <h2 style={titleStyle}>📊 Pool Mining Statistics</h2>
        <div style={{ display: 'flex', alignItems: 'center', gap: '16px' }}>
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
            key={slot.slotId}
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
            height={280}
          />
        ))}
      </div>
    </div>
  );
};

export default GrafanaDashboard;
