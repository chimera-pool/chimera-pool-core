import React from 'react';
import './CyberButton.css';

export interface CyberButtonProps {
  children: React.ReactNode;
  variant?: 'primary' | 'secondary';
  loading?: boolean;
  disabled?: boolean;
  onClick?: () => void;
  icon?: string;
  className?: string;
}

export const CyberButton: React.FC<CyberButtonProps> = ({
  children,
  variant = 'primary',
  loading = false,
  disabled = false,
  onClick,
  icon,
  className = '',
}) => {
  const handleClick = () => {
    if (!disabled && !loading && onClick) {
      onClick();
    }
  };

  const buttonClasses = [
    'cyber-button',
    `cyber-button--${variant}`,
    loading && 'cyber-button--loading',
    disabled && 'cyber-button--disabled',
    className,
  ].filter(Boolean).join(' ');

  return (
    <button
      className={buttonClasses}
      onClick={handleClick}
      disabled={disabled || loading}
    >
      {icon && <span className="cyber-button__icon">{icon}</span>}
      <span className="cyber-button__text">
        {loading ? 'PROCESSING...' : children}
      </span>
    </button>
  );
};