import React, { Component, ErrorInfo, ReactNode } from 'react';

interface Props {
  children: ReactNode;
  fallbackMessage?: string;
  onError?: (error: Error, errorInfo: ErrorInfo) => void;
  onRetry?: () => void;
}

interface State {
  hasError: boolean;
  error: Error | null;
}

/**
 * ChartErrorBoundary - Catches errors in chart components and prevents page crash
 * Provides a fallback UI with retry option
 */
export class ChartErrorBoundary extends Component<Props, State> {
  constructor(props: Props) {
    super(props);
    this.state = { hasError: false, error: null };
  }

  static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, errorInfo: ErrorInfo): void {
    console.error('ChartErrorBoundary caught an error:', error, errorInfo);
    this.props.onError?.(error, errorInfo);
  }

  handleRetry = (): void => {
    this.setState({ hasError: false, error: null });
    this.props.onRetry?.();
  };

  render(): ReactNode {
    if (this.state.hasError) {
      return (
        <div style={styles.container}>
          <div style={styles.content}>
            <span style={styles.icon}>⚠️</span>
            <h3 style={styles.title}>Chart Error</h3>
            <p style={styles.message}>
              {this.props.fallbackMessage || 'Something went wrong loading the charts.'}
            </p>
            {this.state.error && (
              <p style={styles.errorDetail}>
                {this.state.error.message}
              </p>
            )}
            <button onClick={this.handleRetry} style={styles.retryButton}>
              Try Again
            </button>
          </div>
        </div>
      );
    }

    return this.props.children;
  }
}

const styles: Record<string, React.CSSProperties> = {
  container: {
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    minHeight: '300px',
    backgroundColor: '#181B1F',
    borderRadius: '8px',
    padding: '24px',
  },
  content: {
    textAlign: 'center',
    maxWidth: '400px',
  },
  icon: {
    fontSize: '48px',
    display: 'block',
    marginBottom: '16px',
  },
  title: {
    color: '#F0EDF4',
    fontSize: '1.25rem',
    margin: '0 0 8px 0',
  },
  message: {
    color: 'rgba(204, 204, 220, 0.7)',
    fontSize: '0.9rem',
    margin: '0 0 8px 0',
  },
  errorDetail: {
    color: '#FF6B6B',
    fontSize: '0.75rem',
    margin: '0 0 16px 0',
    padding: '8px',
    backgroundColor: 'rgba(255, 107, 107, 0.1)',
    borderRadius: '4px',
    wordBreak: 'break-word',
  },
  retryButton: {
    padding: '10px 20px',
    backgroundColor: 'rgba(245, 184, 0, 0.1)',
    border: '1px solid rgba(245, 184, 0, 0.3)',
    borderRadius: '4px',
    color: '#F5B800',
    cursor: 'pointer',
    fontSize: '0.85rem',
    fontWeight: 500,
  },
};

export default ChartErrorBoundary;
