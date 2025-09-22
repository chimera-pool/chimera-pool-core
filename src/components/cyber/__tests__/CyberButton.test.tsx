import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import { CyberButton } from '../CyberButton';

describe('CyberButton', () => {
  it('should render with cyber-minimal styling', () => {
    render(<CyberButton>TEST_BUTTON</CyberButton>);
    
    const button = screen.getByRole('button', { name: 'TEST_BUTTON' });
    expect(button).toBeInTheDocument();
    expect(button).toHaveClass('cyber-button');
  });

  it('should apply primary variant styling', () => {
    render(<CyberButton variant="primary">PRIMARY_BUTTON</CyberButton>);
    
    const button = screen.getByRole('button');
    expect(button).toHaveClass('cyber-button--primary');
  });

  it('should apply secondary variant styling', () => {
    render(<CyberButton variant="secondary">SECONDARY_BUTTON</CyberButton>);
    
    const button = screen.getByRole('button');
    expect(button).toHaveClass('cyber-button--secondary');
  });

  it('should show loading state with cyber animation', () => {
    render(<CyberButton loading>LOADING_BUTTON</CyberButton>);
    
    const button = screen.getByRole('button');
    expect(button).toHaveClass('cyber-button--loading');
    expect(button).toBeDisabled();
    expect(screen.getByText('PROCESSING...')).toBeInTheDocument();
  });

  it('should be disabled when disabled prop is true', () => {
    render(<CyberButton disabled>DISABLED_BUTTON</CyberButton>);
    
    const button = screen.getByRole('button');
    expect(button).toBeDisabled();
    expect(button).toHaveClass('cyber-button--disabled');
  });

  it('should handle click events', () => {
    const handleClick = jest.fn();
    render(<CyberButton onClick={handleClick}>CLICK_ME</CyberButton>);
    
    const button = screen.getByRole('button');
    fireEvent.click(button);
    
    expect(handleClick).toHaveBeenCalledTimes(1);
  });

  it('should not handle click when disabled', () => {
    const handleClick = jest.fn();
    render(<CyberButton disabled onClick={handleClick}>DISABLED_CLICK</CyberButton>);
    
    const button = screen.getByRole('button');
    fireEvent.click(button);
    
    expect(handleClick).not.toHaveBeenCalled();
  });

  it('should not handle click when loading', () => {
    const handleClick = jest.fn();
    render(<CyberButton loading onClick={handleClick}>LOADING_CLICK</CyberButton>);
    
    const button = screen.getByRole('button');
    fireEvent.click(button);
    
    expect(handleClick).not.toHaveBeenCalled();
  });

  it('should render with cyber icon when provided', () => {
    render(<CyberButton icon="🚀">ROCKET_BUTTON</CyberButton>);
    
    expect(screen.getByText('🚀')).toBeInTheDocument();
    expect(screen.getByText('ROCKET_BUTTON')).toBeInTheDocument();
  });

  it('should apply custom className', () => {
    render(<CyberButton className="custom-class">CUSTOM_BUTTON</CyberButton>);
    
    const button = screen.getByRole('button');
    expect(button).toHaveClass('cyber-button', 'custom-class');
  });
});