/**
 * AdminGrafanaSection - Grafana-embedded charts for Admin Panel
 * 
 * ISP Compliant: Separate component for Grafana integration
 * Uses ChartSlot components with admin-specific chart options
 */

import React, { memo, useState, useCallback } from 'react';
import { ChartSlot } from '../../charts/ChartSlot';
import { GRAFANA_CONFIG } from '../../charts/registry';
import { useGrafanaHealth } from '../../charts/hooks/useGrafanaHealth';
import { useChartPreferences } from '../../charts/hooks/useChartPreferences';

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

interface AdminGrafanaSectionProps {
  token: string;
}

export const AdminGrafanaSection = memo(function AdminGrafanaSection({ token }: AdminGrafanaSectionProps) {
  const grafanaHealth = useGrafanaHealth(GRAFANA_CONFIG.baseUrl);
  const [isExpanded, setIsExpanded] = useState(true);
  const { getSlotSelection } = useChartPreferences();

  // Track selections to prevent duplicates
  const [slotSelections, setSlotSelections] = useState<Record<string, string>>(() => ({
    'admin-1': getSlotSelection('admin', 'admin-1') || 'system-cpu',
    'admin-2': getSlotSelection('admin', 'admin-2') || 'system-memory',
    'admin-3': getSlotSelection('admin', 'admin-3') || 'cumulative-earnings',
    'admin-4': getSlotSelection('admin', 'admin-4') || 'alert-history',
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
    return (
      <div style={styles.section}>
        <div style={styles.header}>
          <h3 style={styles.title}>📊 Grafana Charts</h3>
          <div style={{ ...styles.status, color: '#FF6B6B' }}>
            <span style={{ width: '8px', height: '8px', borderRadius: '50%', backgroundColor: '#FF6B6B' }} />
            Grafana Unavailable
            <button style={styles.toggleBtn} onClick={grafanaHealth.refresh}>
              Retry
            </button>
          </div>
        </div>
        <p style={{ color: 'rgba(204, 204, 220, 0.5)', textAlign: 'center', padding: '20px' }}>
          Native charts are displayed in the Stats tab above.
        </p>
      </div>
    );
  }

  return (
    <div style={styles.section}>
      <div style={styles.header}>
        <h3 style={styles.title}>
          📊 Grafana Admin Charts
        </h3>
        <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
          <div style={{ ...styles.status, color: '#73BF69' }}>
            <span style={{ width: '8px', height: '8px', borderRadius: '50%', backgroundColor: '#73BF69' }} />
            Connected
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
            slotId="admin-1"
            dashboardId="admin"
            allowedCategories={['system']}
            excludedChartIds={getExcludedForSlot('admin-1')}
            onSelectionChange={(id) => handleSelectionChange('admin-1', id)}
            showSelector={true}
            grafanaBaseUrl={GRAFANA_CONFIG.baseUrl}
            grafanaAvailable={grafanaHealth.available}
            height={260}
          />
          <ChartSlot
            slotId="admin-2"
            dashboardId="admin"
            allowedCategories={['system']}
            excludedChartIds={getExcludedForSlot('admin-2')}
            onSelectionChange={(id) => handleSelectionChange('admin-2', id)}
            showSelector={true}
            grafanaBaseUrl={GRAFANA_CONFIG.baseUrl}
            grafanaAvailable={grafanaHealth.available}
            height={260}
          />
          <ChartSlot
            slotId="admin-3"
            dashboardId="admin"
            allowedCategories={['earnings', 'pool-metrics']}
            excludedChartIds={getExcludedForSlot('admin-3')}
            onSelectionChange={(id) => handleSelectionChange('admin-3', id)}
            showSelector={true}
            grafanaBaseUrl={GRAFANA_CONFIG.baseUrl}
            grafanaAvailable={grafanaHealth.available}
            height={260}
          />
          <ChartSlot
            slotId="admin-4"
            dashboardId="admin"
            allowedCategories={['alerts']}
            excludedChartIds={getExcludedForSlot('admin-4')}
            onSelectionChange={(id) => handleSelectionChange('admin-4', id)}
            showSelector={true}
            grafanaBaseUrl={GRAFANA_CONFIG.baseUrl}
            grafanaAvailable={grafanaHealth.available}
            height={260}
          />
        </div>
      )}
    </div>
  );
});

export default AdminGrafanaSection;
