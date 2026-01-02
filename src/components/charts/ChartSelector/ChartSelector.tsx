import React, { useMemo } from 'react';
import { IChartSelectorProps, ChartConfig } from '../interfaces/IChartRegistry';
import { ChartCategory, CHART_CATEGORIES } from '../interfaces/IChartPanel';

/**
 * ChartSelector - Dropdown to select which chart to display
 * Supports category filtering and groups charts by category
 */
export const ChartSelector: React.FC<IChartSelectorProps> = ({
  selectedChartId,
  availableCharts,
  onSelect,
  categoryFilter,
  disabled = false,
  className,
}) => {
  // Filter charts by category if filter provided
  const filteredCharts = useMemo(() => {
    if (!categoryFilter || categoryFilter.length === 0) {
      return availableCharts;
    }
    return availableCharts.filter(chart => 
      categoryFilter.includes(chart.category)
    );
  }, [availableCharts, categoryFilter]);

  // Group charts by category for optgroup display
  const chartsByCategory = useMemo(() => {
    const grouped: Record<ChartCategory, ChartConfig[]> = {
      // Legacy categories
      'pool-metrics': [],
      'worker-metrics': [],
      'earnings': [],
      'system': [],
      'alerts': [],
      // New quadrant categories
      'hashrate-performance': [],
      'workers-activity': [],
      'shares-blocks': [],
      'earnings-payouts': [],
    };

    filteredCharts.forEach(chart => {
      if (grouped[chart.category]) {
        grouped[chart.category].push(chart);
      }
    });

    return grouped;
  }, [filteredCharts]);

  const handleChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
    onSelect(e.target.value);
  };

  const selectStyle: React.CSSProperties = {
    backgroundColor: 'rgba(26, 15, 30, 0.95)',
    color: '#F0EDF4',
    border: '1px solid rgba(212, 168, 75, 0.4)',
    borderRadius: '10px',
    padding: '10px 36px 10px 14px',
    fontSize: '0.85rem',
    fontWeight: 500,
    cursor: disabled ? 'not-allowed' : 'pointer',
    outline: 'none',
    appearance: 'none',
    backgroundImage: `url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='14' height='14' viewBox='0 0 24 24' fill='none' stroke='%23D4A84B' stroke-width='2.5'%3E%3Cpath d='M6 9l6 6 6-6'/%3E%3C/svg%3E")`,
    backgroundRepeat: 'no-repeat',
    backgroundPosition: 'right 12px center',
    minWidth: '180px',
    opacity: disabled ? 0.5 : 1,
    transition: 'all 0.25s ease',
    boxShadow: '0 4px 12px rgba(0, 0, 0, 0.25), inset 0 1px 0 rgba(255, 255, 255, 0.05)',
    textShadow: '0 1px 2px rgba(0, 0, 0, 0.3)',
  };

  const getCategoryLabel = (categoryId: ChartCategory): string => {
    const category = CHART_CATEGORIES.find(c => c.id === categoryId);
    return category ? `${category.icon} ${category.label}` : categoryId;
  };

  // Check if we should use optgroups (multiple categories present)
  const categoriesWithCharts = Object.entries(chartsByCategory)
    .filter(([_, charts]) => charts.length > 0);
  const useOptGroups = categoriesWithCharts.length > 1;

  return (
    <div data-testid="chart-selector" className={className}>
      <select
        value={selectedChartId}
        onChange={handleChange}
        disabled={disabled}
        style={selectStyle}
      >
        {useOptGroups ? (
          // Render with optgroups when multiple categories
          categoriesWithCharts.map(([category, charts]) => (
            <optgroup 
              key={category} 
              label={getCategoryLabel(category as ChartCategory)}
            >
              {charts.map(chart => (
                <option key={chart.id} value={chart.id}>
                  {chart.title}
                </option>
              ))}
            </optgroup>
          ))
        ) : (
          // Render flat list when single category
          filteredCharts.map(chart => (
            <option key={chart.id} value={chart.id}>
              {chart.title}
            </option>
          ))
        )}
      </select>
    </div>
  );
};

export default ChartSelector;
