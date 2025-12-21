import React from 'react';
import { colors } from '../../styles/shared';

// ============================================================================
// LOADING SPINNER COMPONENT
// Reusable loading indicator with size variants
// ============================================================================

export interface LoadingSpinnerProps {
  size?: 'small' | 'medium' | 'large';
  message?: string;
  fullScreen?: boolean;
}

const sizes = {
  small: { spinner: 24, border: 3, fontSize: '0.85rem' },
  medium: { spinner: 40, border: 4, fontSize: '1rem' },
  large: { spinner: 60, border: 5, fontSize: '1.1rem' },
};

const keyframes = `
  @keyframes chimera-spin {
    0% { transform: rotate(0deg); }
    100% { transform: rotate(360deg); }
  }
`;

export function LoadingSpinner({ 
  size = 'medium', 
  message = 'Loading...', 
  fullScreen = false 
}: LoadingSpinnerProps) {
  const sizeConfig = sizes[size];
  
  const containerStyle: React.CSSProperties = fullScreen ? {
    position: 'fixed',
    top: 0,
    left: 0,
    right: 0,
    bottom: 0,
    display: 'flex',
    flexDirection: 'column',
    justifyContent: 'center',
    alignItems: 'center',
    backgroundColor: 'rgba(10, 10, 15, 0.9)',
    zIndex: 9999,
  } : {
    display: 'flex',
    flexDirection: 'column',
    justifyContent: 'center',
    alignItems: 'center',
    padding: '40px 20px',
  };

  const spinnerStyle: React.CSSProperties = {
    width: sizeConfig.spinner,
    height: sizeConfig.spinner,
    border: `${sizeConfig.border}px solid ${colors.border}`,
    borderTop: `${sizeConfig.border}px solid ${colors.primary}`,
    borderRadius: '50%',
    animation: 'chimera-spin 1s linear infinite',
  };

  const messageStyle: React.CSSProperties = {
    marginTop: '16px',
    color: colors.primary,
    fontSize: sizeConfig.fontSize,
  };

  return (
    <>
      <style>{keyframes}</style>
      <div style={containerStyle}>
        <div style={spinnerStyle} />
        {message && <p style={messageStyle}>{message}</p>}
      </div>
    </>
  );
}

export default LoadingSpinner;
