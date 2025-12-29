import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import { ChartSlot } from './ChartSlot';

// Mock the child components
jest.mock('../GrafanaEmbed', () => ({
  GrafanaEmbed: ({ panel }: any) => (
    <div data-testid="grafana-embed">{panel.title}</div>
  ),
}));

jest.mock('../ChartSelector', () => ({
  ChartSelector: ({ onSelect, selectedChartId }: any) => (
    <select 
      data-testid="chart-selector-mock"
      value={selectedChartId}
      onChange={(e) => onSelect(e.target.value)}
    >
      <option value="grafana-pool-hashrate">Pool Hashrate</option>
      <option value="grafana-shares-submitted">Shares</option>
    </select>
  ),
}));

jest.mock('../registry', () => ({
  chartRegistry: {
    getAllCharts: () => [
      { id: 'grafana-pool-hashrate', type: 'grafana', title: 'Pool Hashrate', category: 'pool-metrics' },
      { id: 'grafana-shares-submitted', type: 'grafana', title: 'Shares', category: 'pool-metrics' },
    ],
    getChartById: (id: string) => ({
      id,
      type: 'grafana',
      title: id === 'grafana-pool-hashrate' ? 'Pool Hashrate' : 'Shares',
      category: 'pool-metrics',
      dashboardUid: 'test',
      panelId: 1,
    }),
    getChartsForDashboard: () => [
      { id: 'grafana-pool-hashrate', type: 'grafana', title: 'Pool Hashrate', category: 'pool-metrics' },
      { id: 'grafana-shares-submitted', type: 'grafana', title: 'Shares', category: 'pool-metrics' },
    ],
    getDefaultChart: () => ({
      id: 'grafana-pool-hashrate',
      type: 'grafana',
      title: 'Pool Hashrate',
      category: 'pool-metrics',
    }),
    getNativeFallback: () => ({
      id: 'native-pool-hashrate',
      type: 'native',
      title: 'Pool Hashrate (Fallback)',
      category: 'pool-metrics',
    }),
  },
  GRAFANA_CONFIG: {
    baseUrl: 'http://localhost:3001',
  },
}));

describe('ChartSlot', () => {
  const defaultProps = {
    slotId: 'main-1',
    dashboardId: 'main' as const,
    grafanaBaseUrl: 'http://localhost:3001',
  };

  beforeEach(() => {
    localStorage.clear();
    jest.clearAllMocks();
  });

  describe('rendering', () => {
    it('should render chart slot container', () => {
      render(<ChartSlot {...defaultProps} />);
      expect(screen.getByTestId('chart-slot')).toBeInTheDocument();
    });

    it('should render selector when showSelector is true', () => {
      render(<ChartSlot {...defaultProps} showSelector={true} />);
      expect(screen.getByTestId('chart-selector-mock')).toBeInTheDocument();
    });

    it('should hide selector when showSelector is false', () => {
      render(<ChartSlot {...defaultProps} showSelector={false} />);
      expect(screen.queryByTestId('chart-selector-mock')).not.toBeInTheDocument();
    });

    it('should render Grafana embed by default', () => {
      render(<ChartSlot {...defaultProps} />);
      expect(screen.getByTestId('grafana-embed')).toBeInTheDocument();
    });
  });

  describe('chart selection', () => {
    it('should use initialChartId when provided', () => {
      render(
        <ChartSlot 
          {...defaultProps} 
          initialChartId="grafana-shares-submitted"
          showSelector={true}
        />
      );
      
      const selector = screen.getByTestId('chart-selector-mock');
      expect(selector).toHaveValue('grafana-shares-submitted');
    });

    it('should update chart when selection changes', () => {
      render(<ChartSlot {...defaultProps} showSelector={true} />);
      
      const selector = screen.getByTestId('chart-selector-mock');
      fireEvent.change(selector, { target: { value: 'grafana-shares-submitted' } });
      
      expect(screen.getByTestId('grafana-embed')).toHaveTextContent('Shares');
    });
  });

  describe('preferences persistence', () => {
    it('should save selection to localStorage', () => {
      render(<ChartSlot {...defaultProps} showSelector={true} />);
      
      const selector = screen.getByTestId('chart-selector-mock');
      fireEvent.change(selector, { target: { value: 'grafana-shares-submitted' } });
      
      const saved = localStorage.getItem('chimera-chart-preferences');
      expect(saved).toBeTruthy();
      const parsed = JSON.parse(saved!);
      expect(parsed.dashboards.main['main-1']).toBe('grafana-shares-submitted');
    });
  });

  describe('styling', () => {
    it('should apply custom className', () => {
      render(<ChartSlot {...defaultProps} className="custom-slot" />);
      expect(screen.getByTestId('chart-slot')).toHaveClass('custom-slot');
    });

    it('should apply custom height', () => {
      render(<ChartSlot {...defaultProps} height={400} />);
      const slot = screen.getByTestId('chart-slot');
      expect(slot).toHaveStyle({ minHeight: '400px' });
    });
  });
});
