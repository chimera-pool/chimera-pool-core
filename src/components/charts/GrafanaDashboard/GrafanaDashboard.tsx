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
  // Default to native fallback charts - more reliable cross-browser
  const [useFallback, setUseFallback] = useState(true);
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

  // Always use native fallback for better cross-browser compatibility
  // Grafana iframe embedding has issues with subpath proxying
  useEffect(() => {
    // Keep useFallback true - native charts work better
    setUseFallback(true);
  }, []);

  const containerStyle: React.CSSProperties = {
    background: '#111217',
    borderRadius: '8px',
    padding: '16px',
    marginBottom: '20px',
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
    fontSize: '1.15rem',
    color: '#F0EDF4',
    margin: 0,
    fontWeight: 600,
  };

  const statusStyle: React.CSSProperties = {
    display: 'flex',
    alignItems: 'center',
    gap: '8px',
    fontSize: '0.8rem',
    color: grafanaHealth.available ? '#73BF69' : '#FF6B6B',
  };

  const gridStyle: React.CSSProperties = {
    display: 'grid',
    gridTemplateColumns: `repeat(${layout?.columns || 2}, 1fr)`,
    gap: '16px',
  };

  const fallbackButtonStyle: React.CSSProperties = {
    padding: '6px 12px',
    backgroundColor: useFallback ? 'rgba(115, 191, 105, 0.1)' : 'rgba(245, 184, 0, 0.1)',
    border: `1px solid ${useFallback ? 'rgba(115, 191, 105, 0.3)' : 'rgba(245, 184, 0, 0.3)'}`,
    borderRadius: '4px',
    color: useFallback ? '#73BF69' : '#F5B800',
    cursor: 'pointer',
    fontSize: '0.75rem',
    display: 'flex',
    alignItems: 'center',
    gap: '6px',
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
          {!grafanaHealth.available && (
            <button 
              style={fallbackButtonStyle}
              onClick={() => setUseFallback(true)}
            >
              Use Fallback Charts
            </button>
          )}
        </div>
      </div>

      <div style={gridStyle}>
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
            height={280}
          />
        ))}
      </div>
    </div>
  );
};

export default GrafanaDashboard;
