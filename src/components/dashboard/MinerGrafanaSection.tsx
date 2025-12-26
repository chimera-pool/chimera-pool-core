/**
 * MinerGrafanaSection - Grafana-embedded charts for Miner Dashboard
 * 
 * ISP Compliant: Separate component for Grafana integration
 * Shows worker-specific Grafana charts for authenticated users
 */

import React, { memo, useState, useCallback } from 'react';
import { ChartSlot } from '../charts/ChartSlot';
import { GRAFANA_CONFIG } from '../charts/registry';
import { useGrafanaHealth } from '../charts/hooks/useGrafanaHealth';
import { useChartPreferences } from '../charts/hooks/useChartPreferences';

const styles: { [key: string]: React.CSSProperties } = {
  section: {
    background: '#111217',
    borderRadius: '8px',
    padding: '16px',
    marginBottom: '20px',
  },
  header: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: '16px',
  },
  title: {
    fontSize: '1.15rem',
    color: '#F0EDF4',
    margin: 0,
    fontWeight: 600,
    display: 'flex',
    alignItems: 'center',
    gap: '8px',
  },
  status: {
    display: 'flex',
    alignItems: 'center',
    gap: '8px',
    fontSize: '0.8rem',
  },
  grid: {
    display: 'grid',
    gridTemplateColumns: 'repeat(2, 1fr)',
    gap: '16px',
  },
  toggleBtn: {
    padding: '6px 12px',
    backgroundColor: 'rgba(245, 184, 0, 0.1)',
    border: '1px solid rgba(245, 184, 0, 0.3)',
    borderRadius: '4px',
    color: '#F5B800',
    cursor: 'pointer',
    fontSize: '0.75rem',
  },
};

interface MinerGrafanaSectionProps {
  token: string;
}

export const MinerGrafanaSection = memo(function MinerGrafanaSection({ token }: MinerGrafanaSectionProps) {
  const grafanaHealth = useGrafanaHealth(GRAFANA_CONFIG.baseUrl);
  const [isExpanded, setIsExpanded] = useState(true);
  const { getSlotSelection } = useChartPreferences();

  // Track selections to prevent duplicates
  const [slotSelections, setSlotSelections] = useState<Record<string, string>>(() => ({
    'miner-1': getSlotSelection('miner', 'miner-1') || 'worker-hashrate',
    'miner-2': getSlotSelection('miner', 'miner-2') || 'worker-shares',
    'miner-3': getSlotSelection('miner', 'miner-3') || 'daily-earnings',
    'miner-4': getSlotSelection('miner', 'miner-4') || 'worker-efficiency',
  }));

  const handleSelectionChange = useCallback((slotId: string, chartId: string) => {
    setSlotSelections(prev => ({ ...prev, [slotId]: chartId }));
  }, []);

  const getExcludedForSlot = useCallback((slotId: string) => {
    return Object.entries(slotSelections)
      .filter(([id]) => id !== slotId)
      .map(([, chartId]) => chartId);
  }, [slotSelections]);

  if (!grafanaHealth.available) {
    return null; // Don't show anything if Grafana unavailable - native charts are in EquipmentPage
  }

  return (
    <div style={styles.section}>
      <div style={styles.header}>
        <h3 style={styles.title}>
          📊 Your Mining Statistics
        </h3>
        <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
          <div style={{ ...styles.status, color: '#73BF69' }}>
            <span style={{ width: '8px', height: '8px', borderRadius: '50%', backgroundColor: '#73BF69' }} />
            Live Data
          </div>
          <button 
            style={styles.toggleBtn} 
            onClick={() => setIsExpanded(!isExpanded)}
          >
            {isExpanded ? 'Collapse' : 'Expand'}
          </button>
        </div>
      </div>

      {isExpanded && (
        <div style={styles.grid}>
          <ChartSlot
            slotId="miner-1"
            dashboardId="miner"
            allowedCategories={['worker-metrics']}
            excludedChartIds={getExcludedForSlot('miner-1')}
            onSelectionChange={(id) => handleSelectionChange('miner-1', id)}
            showSelector={true}
            grafanaBaseUrl={GRAFANA_CONFIG.baseUrl}
            height={260}
          />
          <ChartSlot
            slotId="miner-2"
            dashboardId="miner"
            allowedCategories={['worker-metrics']}
            excludedChartIds={getExcludedForSlot('miner-2')}
            onSelectionChange={(id) => handleSelectionChange('miner-2', id)}
            showSelector={true}
            grafanaBaseUrl={GRAFANA_CONFIG.baseUrl}
            height={260}
          />
          <ChartSlot
            slotId="miner-3"
            dashboardId="miner"
            allowedCategories={['earnings']}
            excludedChartIds={getExcludedForSlot('miner-3')}
            onSelectionChange={(id) => handleSelectionChange('miner-3', id)}
            showSelector={true}
            grafanaBaseUrl={GRAFANA_CONFIG.baseUrl}
            height={260}
          />
          <ChartSlot
            slotId="miner-4"
            dashboardId="miner"
            allowedCategories={['worker-metrics', 'earnings']}
            excludedChartIds={getExcludedForSlot('miner-4')}
            onSelectionChange={(id) => handleSelectionChange('miner-4', id)}
            showSelector={true}
            grafanaBaseUrl={GRAFANA_CONFIG.baseUrl}
            height={260}
          />
        </div>
      )}
    </div>
  );
});

export default MinerGrafanaSection;
