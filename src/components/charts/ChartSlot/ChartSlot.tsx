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
  grafanaTimeFrom,
  grafanaTimeTo,
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

  // Get category icon for the selected chart
  const getCategoryIcon = (category: string): string => {
    const icons: Record<string, string> = {
      'hashrate-performance': '⚡',
      'workers-activity': '👷',
      'shares-blocks': '📊',
      'earnings-payouts': '💰',
      'pool-metrics': '📈',
      'worker-metrics': '🖥️',
      'earnings': '💎',
      'system': '⚙️',
      'alerts': '🔔',
    };
    return icons[category] || '📊';
  };

  const [isHovered, setIsHovered] = useState(false);
  const [isLoading, setIsLoading] = useState(true);

  // Simulate loading state for shimmer effect
  useEffect(() => {
    const timer = setTimeout(() => setIsLoading(false), 800);
    return () => clearTimeout(timer);
  }, [selectedChartId]);

  const containerStyle: React.CSSProperties = {
    display: 'flex',
    flexDirection: 'column',
    minHeight: height,
    background: 'linear-gradient(180deg, rgba(45, 31, 61, 0.6) 0%, rgba(26, 15, 30, 0.8) 100%)',
    borderRadius: '16px',
    border: isHovered ? '1px solid rgba(212, 168, 75, 0.4)' : '1px solid rgba(74, 44, 90, 0.5)',
    overflow: 'hidden',
    transition: 'all 0.3s ease',
    boxShadow: isHovered 
      ? '0 8px 32px rgba(212, 168, 75, 0.15), 0 4px 16px rgba(0, 0, 0, 0.3)' 
      : '0 4px 20px rgba(0, 0, 0, 0.2)',
    transform: isHovered ? 'translateY(-2px)' : 'translateY(0)',
  };

  const headerStyle: React.CSSProperties = {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    padding: '16px 20px',
    borderBottom: '1px solid rgba(74, 44, 90, 0.4)',
    background: 'linear-gradient(90deg, rgba(45, 31, 61, 0.5) 0%, rgba(26, 15, 30, 0.6) 100%)',
  };

  const titleStyle: React.CSSProperties = {
    color: '#F0EDF4',
    fontSize: '1rem',
    fontWeight: 700,
    margin: 0,
    display: 'flex',
    alignItems: 'center',
    gap: '12px',
    textShadow: '0 2px 4px rgba(0, 0, 0, 0.3)',
  };

  const categoryIconStyle: React.CSSProperties = {
    fontSize: '1.2rem',
    opacity: 1,
    filter: 'drop-shadow(0 2px 4px rgba(0, 0, 0, 0.3))',
  };

  const shimmerStyle: React.CSSProperties = {
    position: 'absolute',
    top: 0,
    left: 0,
    right: 0,
    bottom: 0,
    background: 'linear-gradient(90deg, transparent 0%, rgba(212, 168, 75, 0.1) 50%, transparent 100%)',
    animation: 'shimmer 1.5s infinite',
    pointerEvents: 'none',
  };

  const tooltipStyle: React.CSSProperties = {
    position: 'relative',
    cursor: 'help',
  };

  const statusIndicatorStyle: React.CSSProperties = {
    width: '8px',
    height: '8px',
    borderRadius: '50%',
    backgroundColor: isGrafanaAvailable ? '#4ADE80' : '#FF6B6B',
    boxShadow: isGrafanaAvailable ? '0 0 8px rgba(74, 222, 128, 0.5)' : '0 0 8px rgba(255, 107, 107, 0.5)',
    animation: isGrafanaAvailable ? 'pulse-glow 2s ease-in-out infinite' : 'none',
  };

  const chartContainerStyle: React.CSSProperties = {
    flex: 1,
    minHeight: height - 50,
  };

  if (!selectedChart) {
    return (
      <div 
        data-testid="chart-slot" 
        className={className} 
        style={containerStyle}
        onMouseEnter={() => setIsHovered(true)}
        onMouseLeave={() => setIsHovered(false)}
      >
        <div style={{ ...chartContainerStyle, display: 'flex', alignItems: 'center', justifyContent: 'center', color: 'rgba(204, 204, 220, 0.5)' }}>
          No chart selected
        </div>
      </div>
    );
  }

  return (
    <div 
      data-testid="chart-slot" 
      className={className} 
      style={containerStyle}
      onMouseEnter={() => setIsHovered(true)}
      onMouseLeave={() => setIsHovered(false)}
    >
      {/* Shimmer loading overlay */}
      {isLoading && (
        <style>
          {`
            @keyframes shimmer {
              0% { transform: translateX(-100%); }
              100% { transform: translateX(100%); }
            }
          `}
        </style>
      )}
      {isLoading && <div style={shimmerStyle} data-testid="chart-loading-shimmer" />}
      {/* Header with title and selector */}
      <div style={headerStyle}>
        <h3 style={titleStyle}>
          <span style={statusIndicatorStyle} title={isGrafanaAvailable ? 'Grafana connected' : 'Using fallback charts'} />
          <span style={categoryIconStyle}>{getCategoryIcon(selectedChart.category)}</span>
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
            panel={{
              ...(selectedChart as IGrafanaPanel),
              from: grafanaTimeFrom || (selectedChart as IGrafanaPanel).from,
              to: grafanaTimeTo || (selectedChart as IGrafanaPanel).to,
            }}
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
