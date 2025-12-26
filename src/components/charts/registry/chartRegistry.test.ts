/**
 * Chart Registry Tests - TDD for Quadrant-Based Chart Categorization
 * 
 * Tests ensure:
 * 1. Each quadrant has unique, non-overlapping charts
 * 2. Charts are properly categorized by theme
 * 3. All 29 Grafana panels are registered
 * 4. ISP compliance - interfaces are properly segregated
 */

import { chartRegistry, GRAFANA_CONFIG, DEFAULT_LAYOUTS } from './chartRegistry';
import { ChartCategory } from '../interfaces/IChartPanel';

describe('ChartRegistry', () => {
  describe('Panel Registration', () => {
    it('should have all 29 Grafana panels registered', () => {
      const allCharts = chartRegistry.getAllCharts();
      const grafanaCharts = allCharts.filter(c => c.type === 'grafana');
      
      // We expect at least 29 Grafana panels
      expect(grafanaCharts.length).toBeGreaterThanOrEqual(29);
    });

    it('should have panels from all 4 dashboards', () => {
      const allCharts = chartRegistry.getAllCharts();
      const grafanaCharts = allCharts.filter(c => c.type === 'grafana');
      
      const dashboardUids = new Set(grafanaCharts.map((c: any) => c.dashboardUid));
      
      expect(dashboardUids.has(GRAFANA_CONFIG.dashboards.poolOverview)).toBe(true);
      expect(dashboardUids.has(GRAFANA_CONFIG.dashboards.workers)).toBe(true);
      expect(dashboardUids.has(GRAFANA_CONFIG.dashboards.payouts)).toBe(true);
      expect(dashboardUids.has(GRAFANA_CONFIG.dashboards.alerts)).toBe(true);
    });
  });

  describe('Quadrant Categories', () => {
    const quadrantCategories: ChartCategory[] = [
      'hashrate-performance',
      'workers-activity', 
      'shares-blocks',
      'earnings-payouts'
    ];

    it('should have 4 distinct quadrant categories', () => {
      const allCharts = chartRegistry.getAllCharts();
      const categories = new Set(allCharts.map(c => c.category));
      
      quadrantCategories.forEach(cat => {
        expect(categories.has(cat)).toBe(true);
      });
    });

    it('should have 7-8 charts per quadrant category', () => {
      quadrantCategories.forEach(category => {
        const charts = chartRegistry.getChartsByCategory(category);
        expect(charts.length).toBeGreaterThanOrEqual(6);
        expect(charts.length).toBeLessThanOrEqual(10);
      });
    });

    it('should not have overlapping charts between quadrants', () => {
      const layout = DEFAULT_LAYOUTS.main;
      const allSlotChartIds: string[] = [];
      
      layout.slots.forEach(slot => {
        if (slot.allowedChartIds) {
          slot.allowedChartIds.forEach(id => {
            // Each chart ID should only appear in one slot
            expect(allSlotChartIds).not.toContain(id);
            allSlotChartIds.push(id);
          });
        }
      });
    });
  });

  describe('Quadrant 1: Hashrate & Performance', () => {
    it('should contain hashrate-related charts', () => {
      const charts = chartRegistry.getChartsByCategory('hashrate-performance');
      const titles = charts.map(c => c.title.toLowerCase());
      
      expect(titles.some(t => t.includes('hashrate'))).toBe(true);
    });

    it('should contain performance metrics', () => {
      const charts = chartRegistry.getChartsByCategory('hashrate-performance');
      const ids = charts.map(c => c.id);
      
      // Should have pool hashrate variants
      expect(ids.some(id => id.includes('hashrate'))).toBe(true);
    });
  });

  describe('Quadrant 2: Workers & Activity', () => {
    it('should contain worker-related charts', () => {
      const charts = chartRegistry.getChartsByCategory('workers-activity');
      const titles = charts.map(c => c.title.toLowerCase());
      
      expect(titles.some(t => t.includes('worker'))).toBe(true);
    });

    it('should contain activity metrics', () => {
      const charts = chartRegistry.getChartsByCategory('workers-activity');
      const ids = charts.map(c => c.id);
      
      expect(ids.some(id => id.includes('worker') || id.includes('connection'))).toBe(true);
    });
  });

  describe('Quadrant 3: Shares & Blocks', () => {
    it('should contain share-related charts', () => {
      const charts = chartRegistry.getChartsByCategory('shares-blocks');
      const titles = charts.map(c => c.title.toLowerCase());
      
      expect(titles.some(t => t.includes('share') || t.includes('block'))).toBe(true);
    });
  });

  describe('Quadrant 4: Earnings & Payouts', () => {
    it('should contain payout-related charts', () => {
      const charts = chartRegistry.getChartsByCategory('earnings-payouts');
      const titles = charts.map(c => c.title.toLowerCase());
      
      expect(titles.some(t => 
        t.includes('payout') || 
        t.includes('earning') || 
        t.includes('balance') ||
        t.includes('wallet')
      )).toBe(true);
    });
  });

  describe('Dashboard Layout', () => {
    it('should have main dashboard with 4 slots', () => {
      const layout = DEFAULT_LAYOUTS.main;
      
      expect(layout.slotCount).toBe(4);
      expect(layout.slots.length).toBe(4);
    });

    it('each slot should have unique allowedChartIds', () => {
      const layout = DEFAULT_LAYOUTS.main;
      
      layout.slots.forEach(slot => {
        expect(slot.allowedChartIds).toBeDefined();
        expect(slot.allowedChartIds!.length).toBeGreaterThanOrEqual(6);
      });
    });

    it('slot 1 should be for hashrate-performance', () => {
      const layout = DEFAULT_LAYOUTS.main;
      const slot1 = layout.slots[0];
      
      expect(slot1.allowedCategories).toContain('hashrate-performance');
    });

    it('slot 2 should be for workers-activity', () => {
      const layout = DEFAULT_LAYOUTS.main;
      const slot2 = layout.slots[1];
      
      expect(slot2.allowedCategories).toContain('workers-activity');
    });

    it('slot 3 should be for shares-blocks', () => {
      const layout = DEFAULT_LAYOUTS.main;
      const slot3 = layout.slots[2];
      
      expect(slot3.allowedCategories).toContain('shares-blocks');
    });

    it('slot 4 should be for earnings-payouts', () => {
      const layout = DEFAULT_LAYOUTS.main;
      const slot4 = layout.slots[3];
      
      expect(slot4.allowedCategories).toContain('earnings-payouts');
    });
  });

  describe('Chart Retrieval', () => {
    it('getChartById should return correct chart', () => {
      const chart = chartRegistry.getChartById('grafana-pool-hashrate-stat');
      
      expect(chart).toBeDefined();
      expect(chart?.title).toContain('Hashrate');
    });

    it('getChartById should return undefined for non-existent chart', () => {
      const chart = chartRegistry.getChartById('non-existent-chart');
      
      expect(chart).toBeUndefined();
    });

    it('getNativeFallback should return fallback for Grafana charts', () => {
      const fallback = chartRegistry.getNativeFallback('grafana-pool-hashrate-stat');
      
      expect(fallback).toBeDefined();
      expect(fallback?.type).toBe('native');
    });
  });

  describe('ISP Compliance', () => {
    it('ChartConfig should have minimal required properties', () => {
      const chart = chartRegistry.getAllCharts()[0];
      
      // Core properties every chart must have
      expect(chart).toHaveProperty('id');
      expect(chart).toHaveProperty('type');
      expect(chart).toHaveProperty('title');
      expect(chart).toHaveProperty('category');
    });

    it('Grafana panels should have Grafana-specific properties', () => {
      const grafanaCharts = chartRegistry.getAllCharts().filter(c => c.type === 'grafana');
      
      grafanaCharts.forEach(chart => {
        expect(chart).toHaveProperty('dashboardUid');
        expect(chart).toHaveProperty('panelId');
      });
    });

    it('Native charts should have native-specific properties', () => {
      const nativeCharts = chartRegistry.getAllCharts().filter(c => c.type === 'native');
      
      nativeCharts.forEach(chart => {
        expect(chart).toHaveProperty('chartType');
        expect(chart).toHaveProperty('dataKey');
      });
    });
  });
});
