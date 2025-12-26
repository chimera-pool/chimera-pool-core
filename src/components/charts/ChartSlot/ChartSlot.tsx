import React, { useState, useEffect, useMemo } from 'react';
import { IChartSlotProps, ChartConfig } from '../interfaces/IChartRegistry';
import { IGrafanaPanel } from '../interfaces/IGrafanaPanel';
import { GrafanaEmbed } from '../GrafanaEmbed';
import { ChartSelector } from '../ChartSelector';
import { NativeChartFallback } from '../NativeChartFallback';
import { chartRegistry } from '../registry';
import { useChartPreferences } from '../hooks/useChartPreferences';

/**
 * ChartSlot - A configurable chart slot with selector and Grafana/Native fallback
 * Combines selector, Grafana embed, and native fallback into one component
 */
export const ChartSlot: React.FC<IChartSlotProps> = ({
  slotId,
  dashboardId,
  initialChartId,
  allowedCategories,
  allowedChartIds,
  excludedChartIds = [],
  onSelectionChange,
  showSelector = true,
  grafanaBaseUrl,
  grafanaAvailable = true,
  fallbackData,
  height = 280,
  className,
}) => {
  // Get available charts for this slot - use allowedChartIds if provided (unique per slot)
  const availableCharts = useMemo(() => {
    let charts = chartRegistry.getAllCharts();
    
    // If specific chart IDs are provided, use those exclusively (unique charts per slot)
    if (allowedChartIds && allowedChartIds.length > 0) {
      charts = charts.filter(c => allowedChartIds.includes(c.id));
    } else if (allowedCategories && allowedCategories.length > 0) {
      // Fall back to category filtering if no specific IDs
      charts = charts.filter(c => allowedCategories.includes(c.category));
    }
    
    // Only show Grafana charts in selector (native are fallbacks)
    charts = charts.filter(c => c.type === 'grafana');
    
    // Exclude charts already selected in other slots (prevent duplicates)
    if (excludedChartIds.length > 0) {
      charts = charts.filter(c => !excludedChartIds.includes(c.id));
    }
    
    return charts;
  }, [allowedChartIds, allowedCategories, excludedChartIds]);

  // User preferences for chart selection
  const { getSlotSelection, setSlotSelection } = useChartPreferences();

  // Determine initial chart ID
  const defaultChartId = useMemo(() => {
    // Priority: saved preference > initial prop > registry default > first available
    const saved = getSlotSelection(dashboardId, slotId);
    if (saved) return saved;
    
    if (initialChartId) return initialChartId;
    
    const registryDefault = chartRegistry.getDefaultChart(dashboardId, slotId);
    if (registryDefault) return registryDefault.id;
    
    return availableCharts[0]?.id || '';
  }, [dashboardId, slotId, initialChartId, getSlotSelection, availableCharts]);

  const [selectedChartId, setSelectedChartId] = useState(defaultChartId);

  // Get the selected chart config
  const selectedChart = useMemo(() => {
    return chartRegistry.getChartById(selectedChartId);
  }, [selectedChartId]);

  // Use grafana availability from parent (avoids multiple health check intervals)
  const isGrafanaAvailable = grafanaAvailable;

  // Handle chart selection change
  const handleSelectChart = (chartId: string) => {
    setSelectedChartId(chartId);
    setSlotSelection(dashboardId, slotId, chartId);
    // Notify parent of selection change for duplicate prevention
    onSelectionChange?.(chartId);
  };

  // Get native fallback if Grafana unavailable
  const nativeFallback = useMemo(() => {
    if (isGrafanaAvailable || !selectedChartId) return null;
    return chartRegistry.getNativeFallback(selectedChartId);
  }, [isGrafanaAvailable, selectedChartId]);

  const containerStyle: React.CSSProperties = {
    display: 'flex',
    flexDirection: 'column',
    minHeight: height,
    backgroundColor: '#181B1F',
    borderRadius: '4px',
    border: '1px solid rgba(255, 255, 255, 0.08)',
    overflow: 'hidden',
  };

  const headerStyle: React.CSSProperties = {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    padding: '12px 16px',
    borderBottom: '1px solid rgba(255, 255, 255, 0.06)',
  };

  const titleStyle: React.CSSProperties = {
    color: 'rgba(204, 204, 220, 0.9)',
    fontSize: '0.85rem',
    fontWeight: 500,
    margin: 0,
    display: 'flex',
    alignItems: 'center',
    gap: '8px',
  };

  const statusIndicatorStyle: React.CSSProperties = {
    width: '8px',
    height: '8px',
    borderRadius: '50%',
    backgroundColor: isGrafanaAvailable ? '#73BF69' : '#FF6B6B',
  };

  const chartContainerStyle: React.CSSProperties = {
    flex: 1,
    minHeight: height - 50,
  };

  if (!selectedChart) {
    return (
      <div data-testid="chart-slot" className={className} style={containerStyle}>
        <div style={{ ...chartContainerStyle, display: 'flex', alignItems: 'center', justifyContent: 'center', color: 'rgba(204, 204, 220, 0.5)' }}>
          No chart selected
        </div>
      </div>
    );
  }

  return (
    <div data-testid="chart-slot" className={className} style={containerStyle}>
      {/* Header with title and selector */}
      <div style={headerStyle}>
        <h3 style={titleStyle}>
          <span style={statusIndicatorStyle} title={isGrafanaAvailable ? 'Grafana connected' : 'Using fallback charts'} />
          {selectedChart.title}
        </h3>
        
        {showSelector && availableCharts.length >= 1 && (
          <ChartSelector
            selectedChartId={selectedChartId}
            availableCharts={availableCharts}
            onSelect={handleSelectChart}
            categoryFilter={allowedCategories}
          />
        )}
      </div>

      {/* Chart content */}
      <div style={chartContainerStyle}>
        {isGrafanaAvailable && selectedChart.type === 'grafana' ? (
          <GrafanaEmbed
            baseUrl={grafanaBaseUrl}
            panel={selectedChart as IGrafanaPanel}
            style={{ height: '100%' }}
          />
        ) : nativeFallback ? (
          <NativeChartFallback
            config={nativeFallback}
            data={fallbackData || []}
            style={{ height: '100%' }}
          />
        ) : (
          <div style={{ 
            display: 'flex', 
            alignItems: 'center', 
            justifyContent: 'center', 
            height: '100%',
            color: 'rgba(204, 204, 220, 0.5)',
            flexDirection: 'column',
            gap: '8px',
          }}>
            <span>📊</span>
            <span>Chart unavailable</span>
            <button 
              onClick={() => window.location.reload()}
              style={{
                padding: '6px 12px',
                backgroundColor: 'rgba(245, 184, 0, 0.1)',
                border: '1px solid rgba(245, 184, 0, 0.3)',
                borderRadius: '4px',
                color: '#F5B800',
                cursor: 'pointer',
                fontSize: '0.75rem',
              }}
            >
              Retry Connection
            </button>
          </div>
        )}
      </div>
    </div>
  );
};

export default ChartSlot;
