import React from 'react';
import './CyberStatusCard.css';

export interface CyberStatusCardProps {
  label: string;
  value: string;
  status: 'healthy' | 'warning' | 'error' | 'unknown';
  icon: string;
}

export const CyberStatusCard: React.FC<CyberStatusCardProps> = ({
  label,
  value,
  status,
  icon,
}) => {
  const cardClasses = [
    'cyber-status-card',
    `cyber-status-card--${status}`,
  ].join(' ');

  const isLongValue = value.length > 20;

  return (
    <div className={cardClasses} data-testid="cyber-status-card">
      <div className="cyber-status-card__header">
        <div className="cyber-status-card__icon">{icon}</div>
        <div className="cyber-status-card__label">{label}</div>
        <div 
          className={`cyber-status-card__indicator ${status === 'healthy' ? 'cyber-pulse' : ''}`}
          data-testid="cyber-status-indicator"
        />
      </div>
      
      <div className="cyber-status-card__content">
        <div 
          className={`cyber-status-card__value ${isLongValue ? 'cyber-truncate' : ''}`}
          data-testid="cyber-status-value"
          title={isLongValue ? value : undefined}
        >
          {value}
        </div>
      </div>
      
      <div className="cyber-status-card__footer">
        <div className="cyber-status-card__timestamp">
          {new Date().toISOString()}
        </div>
      </div>
    </div>
  );
};