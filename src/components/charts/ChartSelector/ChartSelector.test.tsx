import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import { ChartSelector } from './ChartSelector';
import { ChartConfig } from '../interfaces/IChartRegistry';
import { IGrafanaPanel } from '../interfaces/IGrafanaPanel';

describe('ChartSelector', () => {
  const mockCharts: ChartConfig[] = [
    {
      id: 'chart-1',
      type: 'grafana',
      title: 'Pool Hashrate',
      category: 'pool-metrics',
      dashboardUid: 'test',
      panelId: 1,
    } as IGrafanaPanel,
    {
      id: 'chart-2',
      type: 'grafana',
      title: 'Shares Submitted',
      category: 'pool-metrics',
      dashboardUid: 'test',
      panelId: 2,
    } as IGrafanaPanel,
    {
      id: 'chart-3',
      type: 'grafana',
      title: 'Worker Hashrate',
      category: 'worker-metrics',
      dashboardUid: 'test',
      panelId: 3,
    } as IGrafanaPanel,
  ];

  const mockOnSelect = jest.fn();

  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('rendering', () => {
    it('should render dropdown with selected chart', () => {
      render(
        <ChartSelector
          selectedChartId="chart-1"
          availableCharts={mockCharts}
          onSelect={mockOnSelect}
        />
      );

      expect(screen.getByRole('combobox')).toBeInTheDocument();
      expect(screen.getByDisplayValue('Pool Hashrate')).toBeInTheDocument();
    });

    it('should display all available charts as options', () => {
      render(
        <ChartSelector
          selectedChartId="chart-1"
          availableCharts={mockCharts}
          onSelect={mockOnSelect}
        />
      );

      const select = screen.getByRole('combobox');
      expect(select.children).toHaveLength(mockCharts.length);
    });

    it('should filter charts by category when categoryFilter provided', () => {
      render(
        <ChartSelector
          selectedChartId="chart-1"
          availableCharts={mockCharts}
          onSelect={mockOnSelect}
          categoryFilter={['pool-metrics']}
        />
      );

      const select = screen.getByRole('combobox');
      // Should only show 2 pool-metrics charts
      expect(select.children).toHaveLength(2);
    });
  });

  describe('interactions', () => {
    it('should call onSelect when selection changes', () => {
      render(
        <ChartSelector
          selectedChartId="chart-1"
          availableCharts={mockCharts}
          onSelect={mockOnSelect}
        />
      );

      const select = screen.getByRole('combobox');
      fireEvent.change(select, { target: { value: 'chart-2' } });

      expect(mockOnSelect).toHaveBeenCalledWith('chart-2');
    });

    it('should be disabled when disabled prop is true', () => {
      render(
        <ChartSelector
          selectedChartId="chart-1"
          availableCharts={mockCharts}
          onSelect={mockOnSelect}
          disabled={true}
        />
      );

      expect(screen.getByRole('combobox')).toBeDisabled();
    });
  });

  describe('styling', () => {
    it('should apply custom className', () => {
      render(
        <ChartSelector
          selectedChartId="chart-1"
          availableCharts={mockCharts}
          onSelect={mockOnSelect}
          className="custom-selector"
        />
      );

      expect(screen.getByTestId('chart-selector')).toHaveClass('custom-selector');
    });
  });
});
