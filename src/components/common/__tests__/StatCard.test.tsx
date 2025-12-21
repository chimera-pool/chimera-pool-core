import React from 'react';
import { render, screen } from '@testing-library/react';
import { StatCard } from '../StatCard';

describe('StatCard', () => {
  it('should render label and value', () => {
    render(<StatCard label="Total Miners" value={42} />);
    
    expect(screen.getByText('Total Miners')).toBeInTheDocument();
    expect(screen.getByText('42')).toBeInTheDocument();
  });

  it('should render string values', () => {
    render(<StatCard label="Hashrate" value="1.5 TH/s" />);
    
    expect(screen.getByText('Hashrate')).toBeInTheDocument();
    expect(screen.getByText('1.5 TH/s')).toBeInTheDocument();
  });

  it('should render with icon when provided', () => {
    render(<StatCard label="Status" value="Active" icon="⚡" />);
    
    expect(screen.getByText('⚡')).toBeInTheDocument();
    expect(screen.getByText('Status')).toBeInTheDocument();
    expect(screen.getByText('Active')).toBeInTheDocument();
  });

  it('should render positive trend indicator', () => {
    render(
      <StatCard 
        label="Earnings" 
        value="100 BDAG" 
        trend={{ value: 5.5, isPositive: true }} 
      />
    );
    
    expect(screen.getByText(/5\.5%/)).toBeInTheDocument();
    expect(screen.getByText(/↑/)).toBeInTheDocument();
  });

  it('should render negative trend indicator', () => {
    render(
      <StatCard 
        label="Difficulty" 
        value="1000" 
        trend={{ value: 2.3, isPositive: false }} 
      />
    );
    
    expect(screen.getByText(/2\.3%/)).toBeInTheDocument();
    expect(screen.getByText(/↓/)).toBeInTheDocument();
  });

  it('should apply custom color when provided', () => {
    const { container } = render(
      <StatCard label="Custom" value="Test" color="#ff0000" />
    );
    
    const valueElement = container.querySelector('p');
    expect(valueElement).toHaveStyle({ color: '#ff0000' });
  });

  it('should handle zero values', () => {
    render(<StatCard label="Pending" value={0} />);
    
    expect(screen.getByText('0')).toBeInTheDocument();
  });

  it('should handle large numbers', () => {
    render(<StatCard label="Shares" value={1000000} />);
    
    expect(screen.getByText('1000000')).toBeInTheDocument();
  });
});
