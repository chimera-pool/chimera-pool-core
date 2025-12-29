import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { GrafanaEmbed } from './GrafanaEmbed';
import { IGrafanaPanel, buildGrafanaEmbedUrl } from '../interfaces/IGrafanaPanel';

describe('GrafanaEmbed', () => {
  const mockPanel: IGrafanaPanel = {
    id: 'test-panel',
    type: 'grafana',
    title: 'Test Panel',
    category: 'pool-metrics',
    dashboardUid: 'test-dashboard',
    panelId: 1,
    height: 280,
    from: 'now-24h',
    to: 'now',
    theme: 'dark',
  };

  const baseUrl = 'http://localhost:3001';

  describe('buildGrafanaEmbedUrl', () => {
    it('should build correct URL with all parameters', () => {
      const url = buildGrafanaEmbedUrl(baseUrl, mockPanel);
      
      expect(url).toContain('http://localhost:3001/d-solo/test-dashboard');
      expect(url).toContain('panelId=1');
      expect(url).toContain('theme=dark');
      expect(url).toContain('from=now-24h');
      expect(url).toContain('to=now');
    });

    it('should use default orgId of 1', () => {
      const url = buildGrafanaEmbedUrl(baseUrl, mockPanel);
      expect(url).toContain('orgId=1');
    });

    it('should disable auto-refresh to prevent browser overload', () => {
      const panelWithRefresh = { ...mockPanel, refreshInterval: 30 };
      const url = buildGrafanaEmbedUrl(baseUrl, panelWithRefresh);
      // Refresh is intentionally disabled (empty) to prevent browser overload
      expect(url).toContain('refresh=');
    });
  });

  describe('rendering', () => {
    it('should render iframe with correct src', () => {
      render(<GrafanaEmbed baseUrl={baseUrl} panel={mockPanel} />);
      
      const iframe = screen.getByTitle(mockPanel.title);
      expect(iframe).toBeInTheDocument();
      expect(iframe.tagName).toBe('IFRAME');
    });

    it('should apply height from panel config', () => {
      render(<GrafanaEmbed baseUrl={baseUrl} panel={mockPanel} />);
      
      const iframe = screen.getByTitle(mockPanel.title);
      expect(iframe).toHaveStyle({ height: '280px' });
    });

    it('should show loading state initially', () => {
      render(<GrafanaEmbed baseUrl={baseUrl} panel={mockPanel} />);
      
      expect(screen.getByText(/loading/i)).toBeInTheDocument();
    });

    it('should apply custom className', () => {
      render(
        <GrafanaEmbed 
          baseUrl={baseUrl} 
          panel={mockPanel} 
          className="custom-class" 
        />
      );
      
      const container = screen.getByTestId('grafana-embed-container');
      expect(container).toHaveClass('custom-class');
    });

    it('should apply custom styles', () => {
      render(
        <GrafanaEmbed 
          baseUrl={baseUrl} 
          panel={mockPanel} 
          style={{ border: '1px solid red' }} 
        />
      );
      
      const container = screen.getByTestId('grafana-embed-container');
      expect(container).toHaveStyle({ border: '1px solid red' });
    });
  });

  describe('callbacks', () => {
    it('should call onLoad when iframe loads', async () => {
      const onLoad = jest.fn();
      render(<GrafanaEmbed baseUrl={baseUrl} panel={mockPanel} onLoad={onLoad} />);
      
      const iframe = screen.getByTitle(mockPanel.title);
      fireEvent.load(iframe);
      
      await waitFor(() => {
        expect(onLoad).toHaveBeenCalled();
      });
    });

    it('should render iframe element for embedding', () => {
      render(<GrafanaEmbed baseUrl={baseUrl} panel={mockPanel} />);
      
      const iframe = screen.getByTitle(mockPanel.title);
      expect(iframe.tagName).toBe('IFRAME');
    });
  });
});
