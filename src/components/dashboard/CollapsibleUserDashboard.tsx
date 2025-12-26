/**
 * Collapsible User Mining Dashboard
 * Auto-collapses when user has no active mining equipment
 * Expands automatically when equipment is detected mining
 */

import React, { useState, useEffect } from 'react';
import { colors, gradients } from '../../styles/shared';
import {
  IUserEquipmentStatus,
  IUserDashboardVisibility,
  getDashboardVisibility,
} from './interfaces/IUserMiningDashboard';
import { UserDashboard } from './UserDashboard';

export interface CollapsibleUserDashboardProps {
  token: string;
  equipmentStatus: IUserEquipmentStatus;
  isLoggedIn: boolean;
}

const styles: { [key: string]: React.CSSProperties } = {
  container: {
    marginBottom: '30px',
  },
  collapsedSection: {
    background: 'linear-gradient(180deg, rgba(45, 31, 61, 0.5) 0%, rgba(26, 15, 30, 0.7) 100%)',
    borderRadius: '16px',
    padding: '24px 28px',
    border: '1px solid rgba(74, 44, 90, 0.5)',
    cursor: 'pointer',
    transition: 'all 0.3s ease',
    boxShadow: '0 4px 20px rgba(0, 0, 0, 0.2)',
  },
  collapsedHeader: {
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'space-between',
    gap: '16px',
  },
  collapsedTitle: {
    display: 'flex',
    alignItems: 'center',
    gap: '12px',
    margin: 0,
  },
  titleText: {
    fontSize: '1.25rem',
    color: '#F0EDF4',
    margin: 0,
    fontWeight: 700,
    letterSpacing: '0.01em',
  },
  statusBadge: {
    display: 'inline-flex',
    alignItems: 'center',
    gap: '6px',
    padding: '4px 12px',
    borderRadius: '20px',
    fontSize: '0.75rem',
    fontWeight: 600,
    textTransform: 'uppercase' as const,
  },
  activeBadge: {
    backgroundColor: 'rgba(74, 222, 128, 0.15)',
    color: '#4ADE80',
    border: '1px solid rgba(74, 222, 128, 0.3)',
    boxShadow: '0 0 8px rgba(74, 222, 128, 0.2)',
  },
  inactiveBadge: {
    backgroundColor: 'rgba(156, 163, 175, 0.15)',
    color: '#9CA3AF',
    border: '1px solid rgba(156, 163, 175, 0.3)',
  },
  loadingBadge: {
    backgroundColor: 'rgba(96, 165, 250, 0.15)',
    color: '#60A5FA',
    border: '1px solid rgba(96, 165, 250, 0.3)',
  },
  collapsedMessage: {
    color: '#B8B4C8',
    fontSize: '0.95rem',
    margin: '16px 0 0',
    lineHeight: '1.6',
  },
  expandIcon: {
    fontSize: '1.2rem',
    color: colors.textSecondary,
    transition: 'transform 0.3s ease',
  },
  expandIconRotated: {
    transform: 'rotate(180deg)',
  },
  expandedContent: {
    overflow: 'hidden',
    transition: 'max-height 0.3s ease, opacity 0.3s ease',
  },
  noEquipmentActions: {
    display: 'flex',
    gap: '12px',
    marginTop: '16px',
    flexWrap: 'wrap' as const,
  },
  actionButton: {
    padding: '10px 20px',
    borderRadius: '8px',
    fontSize: '0.85rem',
    fontWeight: 500,
    cursor: 'pointer',
    transition: 'all 0.2s ease',
  },
  primaryButton: {
    background: 'linear-gradient(135deg, #D4A84B 0%, #B8923A 100%)',
    border: 'none',
    color: '#1A0F1E',
    boxShadow: '0 2px 12px rgba(212, 168, 75, 0.3)',
  },
  secondaryButton: {
    backgroundColor: 'transparent',
    border: '1px solid rgba(74, 44, 90, 0.5)',
    color: '#B8B4C8',
  },
  equipmentSummary: {
    display: 'flex',
    gap: '20px',
    marginTop: '12px',
    flexWrap: 'wrap' as const,
  },
  summaryItem: {
    display: 'flex',
    alignItems: 'center',
    gap: '8px',
    color: colors.textSecondary,
    fontSize: '0.85rem',
  },
  summaryIcon: {
    fontSize: '1rem',
  },
  summaryValue: {
    fontWeight: 600,
    color: colors.textPrimary,
  },
};

export const CollapsibleUserDashboard: React.FC<CollapsibleUserDashboardProps> = ({
  token,
  equipmentStatus,
  isLoggedIn,
}) => {
  const visibility = getDashboardVisibility(equipmentStatus, isLoggedIn);
  const [isExpanded, setIsExpanded] = useState(visibility.isExpanded);

  // Auto-expand when active equipment is detected
  useEffect(() => {
    if (equipmentStatus.hasActiveEquipment && !isExpanded) {
      setIsExpanded(true);
    }
    // Auto-collapse when no active equipment (unless manually expanded)
    if (!equipmentStatus.hasActiveEquipment && !equipmentStatus.isLoading && visibility.canToggle) {
      // Keep expanded if user manually toggled, otherwise auto-collapse
    }
  }, [equipmentStatus.hasActiveEquipment, equipmentStatus.isLoading]);

  // Don't render if shouldn't show
  if (!visibility.shouldShow) {
    return null;
  }

  const handleToggle = () => {
    if (visibility.canToggle) {
      setIsExpanded(!isExpanded);
    }
  };

  const getStatusBadge = () => {
    if (equipmentStatus.isLoading) {
      return (
        <span style={{ ...styles.statusBadge, ...styles.loadingBadge }}>
          <span className="pulse-dot">●</span> Loading
        </span>
      );
    }
    if (equipmentStatus.hasActiveEquipment) {
      return (
        <span style={{ ...styles.statusBadge, ...styles.activeBadge }}>
          <span>●</span> {equipmentStatus.activeEquipmentCount} Active
        </span>
      );
    }
    if (equipmentStatus.hasEquipment) {
      return (
        <span style={{ ...styles.statusBadge, ...styles.inactiveBadge }}>
          <span>○</span> {equipmentStatus.totalEquipmentCount} Offline
        </span>
      );
    }
    return (
      <span style={{ ...styles.statusBadge, ...styles.inactiveBadge }}>
        <span>○</span> No Equipment
      </span>
    );
  };

  // Collapsed view
  if (!isExpanded) {
    return (
      <div style={styles.container} data-testid="collapsible-user-dashboard">
        <div
          style={{
            ...styles.collapsedSection,
            ...(visibility.canToggle ? { cursor: 'pointer' } : { cursor: 'default' }),
          }}
          onClick={handleToggle}
          role={visibility.canToggle ? 'button' : undefined}
          tabIndex={visibility.canToggle ? 0 : undefined}
          onKeyDown={(e) => {
            if (visibility.canToggle && (e.key === 'Enter' || e.key === ' ')) {
              handleToggle();
            }
          }}
        >
          <div style={styles.collapsedHeader}>
            <div style={styles.collapsedTitle}>
              <h2 style={styles.titleText}>📈 Your Mining Dashboard</h2>
              {getStatusBadge()}
            </div>
            {visibility.canToggle && (
              <span
                style={{
                  ...styles.expandIcon,
                  ...(isExpanded ? styles.expandIconRotated : {}),
                }}
              >
                ▼
              </span>
            )}
          </div>

          {visibility.collapsedMessage && (
            <p style={styles.collapsedMessage}>{visibility.collapsedMessage}</p>
          )}

          {/* Show equipment summary when collapsed but has equipment */}
          {equipmentStatus.hasEquipment && !equipmentStatus.hasActiveEquipment && (
            <div style={styles.equipmentSummary}>
              <div style={styles.summaryItem}>
                <span style={styles.summaryIcon}>⛏️</span>
                <span>
                  <span style={styles.summaryValue}>{equipmentStatus.totalEquipmentCount}</span> devices registered
                </span>
              </div>
              <div style={styles.summaryItem}>
                <span style={styles.summaryIcon}>🔴</span>
                <span>
                  <span style={styles.summaryValue}>{equipmentStatus.totalEquipmentCount - equipmentStatus.activeEquipmentCount}</span> offline
                </span>
              </div>
            </div>
          )}

          {/* Show action buttons when no equipment */}
          {!equipmentStatus.hasEquipment && !equipmentStatus.isLoading && (
            <div style={styles.noEquipmentActions}>
              <a
                href="#connect-miner"
                style={{ ...styles.actionButton, ...styles.primaryButton, textDecoration: 'none' }}
                onClick={(e) => e.stopPropagation()}
              >
                🔗 How to Connect Miner
              </a>
              <a
                href="#equipment"
                style={{ ...styles.actionButton, ...styles.secondaryButton, textDecoration: 'none' }}
                onClick={(e) => e.stopPropagation()}
              >
                📋 Equipment Guide
              </a>
            </div>
          )}
        </div>
      </div>
    );
  }

  // Expanded view - render full UserDashboard
  return (
    <div style={styles.container} data-testid="collapsible-user-dashboard">
      {/* Collapsible header */}
      <div
        style={{
          ...styles.collapsedSection,
          borderBottomLeftRadius: 0,
          borderBottomRightRadius: 0,
          borderBottom: 'none',
          marginBottom: 0,
        }}
        onClick={handleToggle}
        role="button"
        tabIndex={0}
        onKeyDown={(e) => {
          if (e.key === 'Enter' || e.key === ' ') {
            handleToggle();
          }
        }}
      >
        <div style={styles.collapsedHeader}>
          <div style={styles.collapsedTitle}>
            <h2 style={styles.titleText}>📈 Your Mining Dashboard</h2>
            {getStatusBadge()}
          </div>
          <span style={{ ...styles.expandIcon, ...styles.expandIconRotated }}>▼</span>
        </div>
      </div>

      {/* Expanded content */}
      <div
        style={{
          background: gradients.card,
          borderRadius: '0 0 12px 12px',
          border: `1px solid ${colors.border}`,
          borderTop: 'none',
        }}
      >
        <UserDashboard token={token} />
      </div>
    </div>
  );
};

export default CollapsibleUserDashboard;
