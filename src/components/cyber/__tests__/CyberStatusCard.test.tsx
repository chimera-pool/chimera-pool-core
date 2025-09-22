import React from 'react';
import { render, screen } from '@testing-library/react';
import { CyberStatusCard } from '../CyberStatusCard';

describe('CyberStatusCard', () => {
  it('should render with cyber-minimal styling', () => {
    render(
      <CyberStatusCard
        label="ACTIVE_ALGORITHM"
        value="BLAKE2S"
        status="healthy"
        icon="🔧"
      />
    );
    
    expect(screen.getByText('ACTIVE_ALGORITHM')).toBeInTheDocument();
    expect(screen.getByText('BLAKE2S')).toBeInTheDocument();
    expect(screen.getByText('🔧')).toBeInTheDocument();
  });

  it('should apply healthy status styling', () => {
    render(
      <CyberStatusCard
        label="TEST_LABEL"
        value="TEST_VALUE"
        status="healthy"
        icon="✅"
      />
    );
    
    const card = screen.getByTestId('cyber-status-card');
    expect(card).toHaveClass('cyber-status-card--healthy');
  });

  it('should apply warning status styling', () => {
    render(
      <CyberStatusCard
        label="TEST_LABEL"
        value="TEST_VALUE"
        status="warning"
        icon="⚠️"
      />
    );
    
    const card = screen.getByTestId('cyber-status-card');
    expect(card).toHaveClass('cyber-status-card--warning');
  });

  it('should apply error status styling', () => {
    render(
      <CyberStatusCard
        label="TEST_LABEL"
        value="TEST_VALUE"
        status="error"
        icon="❌"
      />
    );
    
    const card = screen.getByTestId('cyber-status-card');
    expect(card).toHaveClass('cyber-status-card--error');
  });

  it('should apply unknown status styling', () => {
    render(
      <CyberStatusCard
        label="TEST_LABEL"
        value="TEST_VALUE"
        status="unknown"
        icon="❓"
      />
    );
    
    const card = screen.getByTestId('cyber-status-card');
    expect(card).toHaveClass('cyber-status-card--unknown');
  });

  it('should render with cyber pulse animation for healthy status', () => {
    render(
      <CyberStatusCard
        label="HEALTHY_SERVICE"
        value="ONLINE"
        status="healthy"
        icon="💚"
      />
    );
    
    const statusIndicator = screen.getByTestId('cyber-status-indicator');
    expect(statusIndicator).toHaveClass('cyber-pulse');
  });

  it('should display formatted timestamp', () => {
    const mockDate = new Date('2023-01-01T12:00:00Z');
    jest.spyOn(global, 'Date').mockImplementation(() => mockDate as any);
    
    render(
      <CyberStatusCard
        label="TIMESTAMP_TEST"
        value="TEST_VALUE"
        status="healthy"
        icon="⏰"
      />
    );
    
    expect(screen.getByText('2023-01-01T12:00:00.000Z')).toBeInTheDocument();
    
    jest.restoreAllMocks();
  });

  it('should handle long values with truncation', () => {
    const longValue = 'VERY_LONG_VALUE_THAT_SHOULD_BE_TRUNCATED_FOR_DISPLAY';
    render(
      <CyberStatusCard
        label="LONG_VALUE_TEST"
        value={longValue}
        status="healthy"
        icon="📏"
      />
    );
    
    const valueElement = screen.getByTestId('cyber-status-value');
    expect(valueElement).toHaveClass('cyber-truncate');
  });
});