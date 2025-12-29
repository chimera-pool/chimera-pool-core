import React from 'react';

// ============================================================================
// SKELETON LOADING COMPONENTS
// Provides visual loading placeholders following Interface Segregation Principle
// Each skeleton type is independently usable and composable
// ============================================================================

/** Base skeleton props following ISP */
interface SkeletonBaseProps {
  className?: string;
  style?: React.CSSProperties;
  /** Animation type: pulse (default), wave, or none */
  animation?: 'pulse' | 'wave' | 'none';
}

/** Skeleton line props for text placeholders */
interface SkeletonLineProps extends SkeletonBaseProps {
  width?: string | number;
  height?: string | number;
}

/** Skeleton card props for card placeholders */
interface SkeletonCardProps extends SkeletonBaseProps {
  width?: string | number;
  height?: string | number;
  borderRadius?: string | number;
}

/** Skeleton stat card props for dashboard stat placeholders */
interface SkeletonStatCardProps extends SkeletonBaseProps {
  showIcon?: boolean;
}

/** Skeleton chart props for chart placeholders */
interface SkeletonChartProps extends SkeletonBaseProps {
  height?: string | number;
  showHeader?: boolean;
}

/** Skeleton table props for table placeholders */
interface SkeletonTableProps extends SkeletonBaseProps {
  rows?: number;
  columns?: number;
}

// CSS keyframes for animations (injected once)
const injectStyles = (() => {
  let injected = false;
  return () => {
    if (injected || typeof document === 'undefined') return;
    injected = true;
    
    const style = document.createElement('style');
    style.textContent = `
      @keyframes skeleton-pulse {
        0%, 100% { opacity: 0.4; }
        50% { opacity: 0.8; }
      }
      @keyframes skeleton-wave {
        0% { background-position: -200% 0; }
        100% { background-position: 200% 0; }
      }
    `;
    document.head.appendChild(style);
  };
})();

const baseStyles: React.CSSProperties = {
  backgroundColor: 'rgba(255, 255, 255, 0.1)',
  borderRadius: '4px',
};

const getAnimationStyle = (animation: 'pulse' | 'wave' | 'none'): React.CSSProperties => {
  injectStyles();
  
  switch (animation) {
    case 'pulse':
      return { animation: 'skeleton-pulse 1.5s ease-in-out infinite' };
    case 'wave':
      return {
        background: 'linear-gradient(90deg, rgba(255,255,255,0.1) 25%, rgba(255,255,255,0.2) 50%, rgba(255,255,255,0.1) 75%)',
        backgroundSize: '200% 100%',
        animation: 'skeleton-wave 1.5s ease-in-out infinite',
      };
    case 'none':
    default:
      return {};
  }
};

/** Basic skeleton line for text placeholders */
export function SkeletonLine({ 
  width = '100%', 
  height = 16, 
  animation = 'pulse',
  className,
  style 
}: SkeletonLineProps) {
  return (
    <div
      className={className}
      style={{
        ...baseStyles,
        ...getAnimationStyle(animation),
        width: typeof width === 'number' ? `${width}px` : width,
        height: typeof height === 'number' ? `${height}px` : height,
        ...style,
      }}
      role="status"
      aria-label="Loading"
    />
  );
}

/** Skeleton circle for avatar placeholders */
export function SkeletonCircle({ 
  size = 40, 
  animation = 'pulse',
  className,
  style 
}: SkeletonBaseProps & { size?: number }) {
  return (
    <div
      className={className}
      style={{
        ...baseStyles,
        ...getAnimationStyle(animation),
        width: size,
        height: size,
        borderRadius: '50%',
        ...style,
      }}
      role="status"
      aria-label="Loading"
    />
  );
}

/** Skeleton card for card placeholders */
export function SkeletonCard({ 
  width = '100%', 
  height = 120, 
  borderRadius = 8,
  animation = 'pulse',
  className,
  style 
}: SkeletonCardProps) {
  return (
    <div
      className={className}
      style={{
        ...baseStyles,
        ...getAnimationStyle(animation),
        width: typeof width === 'number' ? `${width}px` : width,
        height: typeof height === 'number' ? `${height}px` : height,
        borderRadius: typeof borderRadius === 'number' ? `${borderRadius}px` : borderRadius,
        ...style,
      }}
      role="status"
      aria-label="Loading"
    />
  );
}

/** Skeleton stat card for dashboard statistics */
export function SkeletonStatCard({ 
  showIcon = true,
  animation = 'pulse',
  className,
  style 
}: SkeletonStatCardProps) {
  return (
    <div
      className={className}
      style={{
        backgroundColor: 'rgba(255, 255, 255, 0.05)',
        borderRadius: '12px',
        padding: '20px',
        display: 'flex',
        flexDirection: 'column',
        gap: '12px',
        minWidth: '140px',
        ...style,
      }}
      role="status"
      aria-label="Loading statistic"
    >
      {showIcon && (
        <SkeletonCircle size={32} animation={animation} />
      )}
      <SkeletonLine width="60%" height={14} animation={animation} />
      <SkeletonLine width="80%" height={24} animation={animation} />
    </div>
  );
}

/** Skeleton chart for chart placeholders */
export function SkeletonChart({ 
  height = 300,
  showHeader = true,
  animation = 'pulse',
  className,
  style 
}: SkeletonChartProps) {
  return (
    <div
      className={className}
      style={{
        backgroundColor: 'rgba(255, 255, 255, 0.03)',
        borderRadius: '12px',
        padding: '20px',
        ...style,
      }}
      role="status"
      aria-label="Loading chart"
    >
      {showHeader && (
        <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '16px' }}>
          <SkeletonLine width={150} height={20} animation={animation} />
          <SkeletonLine width={100} height={32} animation={animation} style={{ borderRadius: '6px' }} />
        </div>
      )}
      <SkeletonCard 
        height={typeof height === 'number' ? height - (showHeader ? 68 : 0) : height} 
        animation={animation}
        borderRadius={8}
      />
    </div>
  );
}

/** Skeleton table row */
export function SkeletonTableRow({ 
  columns = 4,
  animation = 'pulse',
  className,
  style 
}: SkeletonBaseProps & { columns?: number }) {
  return (
    <div
      className={className}
      style={{
        display: 'grid',
        gridTemplateColumns: `repeat(${columns}, 1fr)`,
        gap: '16px',
        padding: '12px 0',
        borderBottom: '1px solid rgba(255, 255, 255, 0.05)',
        ...style,
      }}
      role="row"
    >
      {Array.from({ length: columns }).map((_, i) => (
        <SkeletonLine 
          key={i} 
          width={`${60 + Math.random() * 30}%`} 
          height={16} 
          animation={animation}
        />
      ))}
    </div>
  );
}

/** Skeleton table for table placeholders */
export function SkeletonTable({ 
  rows = 5,
  columns = 4,
  animation = 'pulse',
  className,
  style 
}: SkeletonTableProps) {
  return (
    <div
      className={className}
      style={{
        backgroundColor: 'rgba(255, 255, 255, 0.03)',
        borderRadius: '12px',
        padding: '20px',
        ...style,
      }}
      role="status"
      aria-label="Loading table"
    >
      {/* Header */}
      <div
        style={{
          display: 'grid',
          gridTemplateColumns: `repeat(${columns}, 1fr)`,
          gap: '16px',
          padding: '12px 0',
          borderBottom: '1px solid rgba(255, 255, 255, 0.1)',
          marginBottom: '8px',
        }}
      >
        {Array.from({ length: columns }).map((_, i) => (
          <SkeletonLine key={i} width="70%" height={14} animation={animation} />
        ))}
      </div>
      
      {/* Rows */}
      {Array.from({ length: rows }).map((_, i) => (
        <SkeletonTableRow key={i} columns={columns} animation={animation} />
      ))}
    </div>
  );
}

/** Dashboard skeleton - combines multiple skeleton types */
export function SkeletonDashboard({ animation = 'pulse' }: SkeletonBaseProps) {
  return (
    <div role="status" aria-label="Loading dashboard">
      {/* Stats row */}
      <div style={{ 
        display: 'grid', 
        gridTemplateColumns: 'repeat(auto-fit, minmax(140px, 1fr))', 
        gap: '16px',
        marginBottom: '24px'
      }}>
        {Array.from({ length: 6 }).map((_, i) => (
          <SkeletonStatCard key={i} animation={animation} />
        ))}
      </div>
      
      {/* Charts grid */}
      <div style={{ 
        display: 'grid', 
        gridTemplateColumns: 'repeat(auto-fit, minmax(300px, 1fr))', 
        gap: '16px',
        marginBottom: '24px'
      }}>
        <SkeletonChart animation={animation} />
        <SkeletonChart animation={animation} />
      </div>
      
      {/* Table */}
      <SkeletonTable rows={5} columns={4} animation={animation} />
    </div>
  );
}

export default {
  Line: SkeletonLine,
  Circle: SkeletonCircle,
  Card: SkeletonCard,
  StatCard: SkeletonStatCard,
  Chart: SkeletonChart,
  Table: SkeletonTable,
  TableRow: SkeletonTableRow,
  Dashboard: SkeletonDashboard,
};
