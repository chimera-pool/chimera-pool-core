import React from 'react';

// ============================================================================
// STAT CARD COMPONENT
// Extracted from App.tsx for modular architecture
// Displays pool statistics with live indicators
// ============================================================================

interface StatCardProps {
  label: string;
  value: string | number;
}

const styles = {
  statCard: {
    background: 'linear-gradient(180deg, rgba(45, 31, 61, 0.8) 0%, rgba(26, 15, 30, 0.9) 100%)',
    borderRadius: '14px',
    padding: '20px',
    border: '1px solid #4A2C5A',
    textAlign: 'center' as const,
    boxShadow: '0 4px 24px rgba(0, 0, 0, 0.3)',
    transition: 'all 0.2s ease',
  },
  statLabel: {
    fontSize: '0.75rem',
    color: '#B8B4C8',
    margin: '0 0 8px',
    textTransform: 'uppercase' as const,
    letterSpacing: '0.08em',
    fontWeight: 500,
  },
  statValue: {
    fontSize: '1.5rem',
    color: '#D4A84B',
    margin: 0,
    fontWeight: 700,
    letterSpacing: '-0.02em',
  },
};

export function StatCard({ label, value }: StatCardProps) {
  const isLiveData = ['Active Miners', 'Pool Hashrate'].includes(label);
  const isHashrate = label === 'Pool Hashrate';

  return (
    <div 
      style={styles.statCard} 
      className="stat-card-enhanced"
      data-testid={`stat-card-${label.toLowerCase().replace(/\s+/g, '-')}`}
    >
      <h3 style={styles.statLabel}>
        {isLiveData && <span className="live-indicator" />}
        {label}
      </h3>
      <p 
        style={styles.statValue} 
        className={`stat-value-glow ${isHashrate ? 'hashrate-value' : ''}`}
      >
        {value}
      </p>
    </div>
  );
}

export default StatCard;
