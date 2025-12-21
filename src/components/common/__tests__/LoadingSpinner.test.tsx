import React from 'react';
import { render, screen } from '@testing-library/react';
import { LoadingSpinner } from '../LoadingSpinner';

describe('LoadingSpinner', () => {
  it('should render with default message', () => {
    render(<LoadingSpinner />);
    
    expect(screen.getByText('Loading...')).toBeInTheDocument();
  });

  it('should render with custom message', () => {
    render(<LoadingSpinner message="Fetching data..." />);
    
    expect(screen.getByText('Fetching data...')).toBeInTheDocument();
  });

  it('should render without message when empty string provided', () => {
    const { container } = render(<LoadingSpinner message="" />);
    
    // Should only have the spinner, no text
    expect(container.querySelectorAll('p').length).toBe(0);
  });

  it('should apply small size variant', () => {
    const { container } = render(<LoadingSpinner size="small" />);
    
    const spinner = container.querySelector('div > div');
    expect(spinner).toHaveStyle({ width: '24px', height: '24px' });
  });

  it('should apply medium size variant (default)', () => {
    const { container } = render(<LoadingSpinner size="medium" />);
    
    const spinner = container.querySelector('div > div');
    expect(spinner).toHaveStyle({ width: '40px', height: '40px' });
  });

  it('should apply large size variant', () => {
    const { container } = render(<LoadingSpinner size="large" />);
    
    const spinner = container.querySelector('div > div');
    expect(spinner).toHaveStyle({ width: '60px', height: '60px' });
  });

  it('should render in fullScreen mode', () => {
    const { container } = render(<LoadingSpinner fullScreen />);
    
    const wrapper = container.firstChild as HTMLElement;
    expect(wrapper).toHaveStyle({ position: 'fixed' });
  });

  it('should not be fullScreen by default', () => {
    const { container } = render(<LoadingSpinner />);
    
    const wrapper = container.firstChild as HTMLElement;
    expect(wrapper).not.toHaveStyle({ position: 'fixed' });
  });

  it('should have animation keyframes', () => {
    const { container } = render(<LoadingSpinner />);
    
    // Check that style element with keyframes is present
    const styleElement = container.querySelector('style');
    expect(styleElement).toBeInTheDocument();
    expect(styleElement?.textContent).toContain('chimera-spin');
  });
});
