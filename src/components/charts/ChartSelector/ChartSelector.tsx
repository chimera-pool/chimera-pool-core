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
      'pool-metrics': [],
      'worker-metrics': [],
      'earnings': [],
      'system': [],
      'alerts': [],
    };

    filteredCharts.forEach(chart => {
      grouped[chart.category].push(chart);
    });

    return grouped;
  }, [filteredCharts]);

  const handleChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
    onSelect(e.target.value);
  };

  const selectStyle: React.CSSProperties = {
    backgroundColor: 'rgba(24, 27, 31, 0.9)',
    color: '#CCCCDC',
    border: '1px solid rgba(255, 255, 255, 0.1)',
    borderRadius: '4px',
    padding: '6px 28px 6px 10px',
    fontSize: '0.8rem',
    cursor: disabled ? 'not-allowed' : 'pointer',
    outline: 'none',
    appearance: 'none',
    backgroundImage: `url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' viewBox='0 0 24 24' fill='none' stroke='%23CCCCDC' stroke-width='2'%3E%3Cpath d='M6 9l6 6 6-6'/%3E%3C/svg%3E")`,
    backgroundRepeat: 'no-repeat',
    backgroundPosition: 'right 8px center',
    minWidth: '160px',
    opacity: disabled ? 0.5 : 1,
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
