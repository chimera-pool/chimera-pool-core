import React, { Component, ErrorInfo, ReactNode } from 'react';
import { colors } from '../../styles/shared';

// ============================================================================
// ERROR BOUNDARY COMPONENT
// Catches JavaScript errors in child component tree and displays fallback UI
// World-class error handling with recovery options
// ============================================================================

interface ErrorBoundaryProps {
  children: ReactNode;
  fallback?: ReactNode;
  onError?: (error: Error, errorInfo: ErrorInfo) => void;
  componentName?: string;
}

interface ErrorBoundaryState {
  hasError: boolean;
  error: Error | null;
  errorInfo: ErrorInfo | null;
}

const styles: { [key: string]: React.CSSProperties } = {
  container: {
    padding: '40px 20px',
    textAlign: 'center',
    backgroundColor: '#1a1a2e',
    borderRadius: '12px',
    border: `1px solid ${colors.error}`,
    margin: '20px',
  },
  icon: {
    fontSize: '3rem',
    marginBottom: '16px',
  },
  title: {
    color: colors.error,
    fontSize: '1.3rem',
    marginBottom: '12px',
  },
  message: {
    color: colors.textSecondary,
    marginBottom: '20px',
    maxWidth: '500px',
    margin: '0 auto 20px',
    lineHeight: '1.6',
  },
  errorDetails: {
    backgroundColor: '#0a0a15',
    padding: '16px',
    borderRadius: '8px',
    textAlign: 'left' as const,
    marginBottom: '20px',
    maxHeight: '150px',
    overflow: 'auto',
    fontFamily: 'monospace',
    fontSize: '0.85rem',
    color: colors.error,
  },
  actions: {
    display: 'flex',
    gap: '12px',
    justifyContent: 'center',
    flexWrap: 'wrap' as const,
  },
  retryBtn: {
    padding: '10px 24px',
    backgroundColor: colors.primary,
    border: 'none',
    borderRadius: '6px',
    color: colors.bgDark,
    fontWeight: 'bold',
    cursor: 'pointer',
    fontSize: '0.9rem',
  },
  reportBtn: {
    padding: '10px 24px',
    backgroundColor: 'transparent',
    border: `1px solid ${colors.textSecondary}`,
    borderRadius: '6px',
    color: colors.textSecondary,
    cursor: 'pointer',
    fontSize: '0.9rem',
  },
};

export class ErrorBoundary extends Component<ErrorBoundaryProps, ErrorBoundaryState> {
  constructor(props: ErrorBoundaryProps) {
    super(props);
    this.state = {
      hasError: false,
      error: null,
      errorInfo: null,
    };
  }

  static getDerivedStateFromError(error: Error): Partial<ErrorBoundaryState> {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, errorInfo: ErrorInfo): void {
    this.setState({ errorInfo });
    
    // Call optional error callback
    if (this.props.onError) {
      this.props.onError(error, errorInfo);
    }

    // Log to console in development
    console.error('ErrorBoundary caught an error:', error, errorInfo);
  }

  handleRetry = (): void => {
    this.setState({ hasError: false, error: null, errorInfo: null });
  };

  handleReport = (): void => {
    // In production, this would send error to logging service
    const { error, errorInfo } = this.state;
    const errorReport = {
      message: error?.message,
      stack: error?.stack,
      componentStack: errorInfo?.componentStack,
      componentName: this.props.componentName,
      timestamp: new Date().toISOString(),
      userAgent: navigator.userAgent,
    };
    
    console.log('Error Report:', errorReport);
    alert('Error report generated. Check console for details.');
  };

  render(): ReactNode {
    if (this.state.hasError) {
      // Custom fallback provided
      if (this.props.fallback) {
        return this.props.fallback;
      }

      // Default fallback UI
      return (
        <div style={styles.container}>
          <div style={styles.icon}>⚠️</div>
          <h2 style={styles.title}>Something went wrong</h2>
          <p style={styles.message}>
            {this.props.componentName 
              ? `The ${this.props.componentName} component encountered an error.`
              : 'An unexpected error occurred in this section.'
            }
            {' '}You can try refreshing or report this issue.
          </p>
          
          {this.state.error && (
            <div style={styles.errorDetails}>
              <strong>Error:</strong> {this.state.error.message}
            </div>
          )}
          
          <div style={styles.actions}>
            <button style={styles.retryBtn} onClick={this.handleRetry}>
              🔄 Try Again
            </button>
            <button style={styles.reportBtn} onClick={this.handleReport}>
              🐛 Report Issue
            </button>
          </div>
        </div>
      );
    }

    return this.props.children;
  }
}

export default ErrorBoundary;
