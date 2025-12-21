import React from 'react';
import { colors, gradients } from '../../styles/shared';

// ============================================================================
// STAT CARD COMPONENT
// Reusable statistics display card with consistent styling
// ============================================================================

export interface StatCardProps {
  label: string;
  value: string | number;
  icon?: string;
  trend?: {
    value: number;
    isPositive: boolean;
  };
  color?: string;
}

const styles: { [key: string]: React.CSSProperties } = {
  card: {
    background: gradients.card,
    borderRadius: '12px',
    padding: '24px',
    border: `1px solid ${colors.border}`,
    textAlign: 'center',
    transition: 'transform 0.2s, box-shadow 0.2s',
  },
  icon: {
    fontSize: '1.5rem',
    marginBottom: '8px',
  },
  label: {
    fontSize: '0.9rem',
    color: colors.textSecondary,
    margin: '0 0 8px',
    textTransform: 'uppercase' as const,
    letterSpacing: '0.5px',
  },
  value: {
    fontSize: '1.5rem',
    color: colors.primary,
    margin: 0,
    fontWeight: 'bold',
  },
  trend: {
    fontSize: '0.8rem',
    marginTop: '8px',
  },
  trendPositive: {
    color: colors.success,
  },
  trendNegative: {
    color: colors.error,
  },
};

export function StatCard({ label, value, icon, trend, color }: StatCardProps) {
  return (
    <div style={styles.card}>
      {icon && <div style={styles.icon}>{icon}</div>}
      <h3 style={styles.label}>{label}</h3>
      <p style={{ ...styles.value, color: color || colors.primary }}>{value}</p>
      {trend && (
        <div style={{
          ...styles.trend,
          ...(trend.isPositive ? styles.trendPositive : styles.trendNegative)
        }}>
          {trend.isPositive ? '↑' : '↓'} {Math.abs(trend.value).toFixed(1)}%
        </div>
      )}
    </div>
  );
}

export default StatCard;
