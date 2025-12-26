/**
 * MinerGrafanaSection - Grafana-embedded charts for Miner Dashboard
 * 
 * ISP Compliant: Separate component for Grafana integration
 * Shows worker-specific Grafana charts for authenticated users
 */

import React, { memo, useState } from 'react';
import { ChartSlot } from '../charts/ChartSlot';
import { GRAFANA_CONFIG } from '../charts/registry';
import { useGrafanaHealth } from '../charts/hooks/useGrafanaHealth';

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
            showSelector={true}
            grafanaBaseUrl={GRAFANA_CONFIG.baseUrl}
            height={260}
          />
          <ChartSlot
            slotId="miner-2"
            dashboardId="miner"
            allowedCategories={['worker-metrics']}
            showSelector={true}
            grafanaBaseUrl={GRAFANA_CONFIG.baseUrl}
            height={260}
          />
          <ChartSlot
            slotId="miner-3"
            dashboardId="miner"
            allowedCategories={['earnings']}
            showSelector={true}
            grafanaBaseUrl={GRAFANA_CONFIG.baseUrl}
            height={260}
          />
          <ChartSlot
            slotId="miner-4"
            dashboardId="miner"
            allowedCategories={['worker-metrics', 'earnings']}
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
