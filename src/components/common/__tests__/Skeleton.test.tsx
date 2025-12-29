import React from 'react';
import { render, screen } from '@testing-library/react';
import {
  SkeletonLine,
  SkeletonCircle,
  SkeletonCard,
  SkeletonStatCard,
  SkeletonChart,
  SkeletonTable,
  SkeletonTableRow,
  SkeletonDashboard,
} from '../Skeleton';

describe('Skeleton Components', () => {
  describe('SkeletonLine', () => {
    it('should render with default props', () => {
      render(<SkeletonLine />);
      const skeleton = screen.getByRole('status');
      expect(skeleton).toBeInTheDocument();
      expect(skeleton).toHaveAttribute('aria-label', 'Loading');
    });

    it('should apply custom width and height', () => {
      render(<SkeletonLine width={200} height={20} />);
      const skeleton = screen.getByRole('status');
      expect(skeleton).toHaveStyle({ width: '200px', height: '20px' });
    });

    it('should accept string width values', () => {
      render(<SkeletonLine width="50%" />);
      const skeleton = screen.getByRole('status');
      expect(skeleton).toHaveStyle({ width: '50%' });
    });

    it('should apply custom styles', () => {
      render(<SkeletonLine style={{ marginTop: '10px' }} />);
      const skeleton = screen.getByRole('status');
      expect(skeleton).toHaveStyle({ marginTop: '10px' });
    });
  });

  describe('SkeletonCircle', () => {
    it('should render as a circle', () => {
      render(<SkeletonCircle size={50} />);
      const skeleton = screen.getByRole('status');
      expect(skeleton).toHaveStyle({ 
        width: '50px', 
        height: '50px', 
        borderRadius: '50%' 
      });
    });

    it('should use default size of 40', () => {
      render(<SkeletonCircle />);
      const skeleton = screen.getByRole('status');
      expect(skeleton).toHaveStyle({ width: '40px', height: '40px' });
    });
  });

  describe('SkeletonCard', () => {
    it('should render with default dimensions', () => {
      render(<SkeletonCard />);
      const skeleton = screen.getByRole('status');
      expect(skeleton).toHaveStyle({ width: '100%', height: '120px' });
    });

    it('should apply custom border radius', () => {
      render(<SkeletonCard borderRadius={16} />);
      const skeleton = screen.getByRole('status');
      expect(skeleton).toHaveStyle({ borderRadius: '16px' });
    });
  });

  describe('SkeletonStatCard', () => {
    it('should render stat card placeholder', () => {
      render(<SkeletonStatCard />);
      const skeleton = screen.getByLabelText('Loading statistic');
      expect(skeleton).toBeInTheDocument();
    });

    it('should hide icon when showIcon is false', () => {
      const { container } = render(<SkeletonStatCard showIcon={false} />);
      // With showIcon=false, there should be fewer child elements
      const children = container.querySelector('[aria-label="Loading statistic"]')?.children;
      expect(children?.length).toBe(2); // Only 2 lines, no circle
    });
  });

  describe('SkeletonChart', () => {
    it('should render chart placeholder', () => {
      render(<SkeletonChart />);
      const skeleton = screen.getByLabelText('Loading chart');
      expect(skeleton).toBeInTheDocument();
    });

    it('should hide header when showHeader is false', () => {
      const { container } = render(<SkeletonChart showHeader={false} />);
      const skeleton = container.querySelector('[role="status"]');
      // Without header, should have fewer children
      expect(skeleton?.children.length).toBe(1);
    });
  });

  describe('SkeletonTable', () => {
    it('should render table placeholder', () => {
      render(<SkeletonTable />);
      const skeleton = screen.getByLabelText('Loading table');
      expect(skeleton).toBeInTheDocument();
    });

    it('should render correct number of rows', () => {
      render(<SkeletonTable rows={3} columns={4} />);
      const rows = screen.getAllByRole('row');
      expect(rows.length).toBe(3);
    });
  });

  describe('SkeletonTableRow', () => {
    it('should render row with columns', () => {
      render(<SkeletonTableRow columns={5} />);
      const row = screen.getByRole('row');
      expect(row.children.length).toBe(5);
    });
  });

  describe('SkeletonDashboard', () => {
    it('should render complete dashboard skeleton', () => {
      render(<SkeletonDashboard />);
      const skeleton = screen.getByLabelText('Loading dashboard');
      expect(skeleton).toBeInTheDocument();
    });

    it('should render multiple stat cards', () => {
      const { container } = render(<SkeletonDashboard />);
      const statCards = container.querySelectorAll('[aria-label="Loading statistic"]');
      expect(statCards.length).toBe(6);
    });
  });

  describe('Animation variants', () => {
    it('should support pulse animation', () => {
      render(<SkeletonLine animation="pulse" data-testid="pulse" />);
      const skeleton = screen.getByRole('status');
      expect(skeleton).toBeInTheDocument();
    });

    it('should support wave animation', () => {
      render(<SkeletonLine animation="wave" data-testid="wave" />);
      const skeleton = screen.getByRole('status');
      expect(skeleton).toBeInTheDocument();
    });

    it('should support no animation', () => {
      render(<SkeletonLine animation="none" data-testid="none" />);
      const skeleton = screen.getByRole('status');
      expect(skeleton).toBeInTheDocument();
    });
  });

  describe('Accessibility', () => {
    it('should have proper ARIA attributes', () => {
      render(<SkeletonLine />);
      const skeleton = screen.getByRole('status');
      expect(skeleton).toHaveAttribute('aria-label', 'Loading');
    });

    it('should be focusable for screen readers', () => {
      render(<SkeletonStatCard />);
      const skeleton = screen.getByLabelText('Loading statistic');
      expect(skeleton).toBeInTheDocument();
    });
  });
});
