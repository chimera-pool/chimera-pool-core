import React, { lazy, Suspense } from 'react';
import { LoadingSpinner } from './common/LoadingSpinner';
import { ErrorBoundary } from './common/ErrorBoundary';

// ============================================================================
// LAZY LOADED COMPONENTS WITH ERROR BOUNDARIES
// Code splitting for performance optimization
// ============================================================================

// Lazy load heavy components
const LazyMiningGraphs = lazy(() => import('./charts/MiningGraphs'));
const LazyGlobalMinerMap = lazy(() => import('./maps/GlobalMinerMap'));
const LazyUserDashboard = lazy(() => import('./dashboard/UserDashboard'));
const LazyWalletManager = lazy(() => import('./wallet/WalletManager'));
const LazyAuthModal = lazy(() => import('./auth/AuthModal'));

// Wrapper component that provides error boundary and suspense
interface LazyWrapperProps {
  children: React.ReactNode;
  componentName: string;
  loadingMessage?: string;
}

function LazyWrapper({ children, componentName, loadingMessage }: LazyWrapperProps) {
  return (
    <ErrorBoundary componentName={componentName}>
      <Suspense fallback={<LoadingSpinner message={loadingMessage || `Loading ${componentName}...`} />}>
        {children}
      </Suspense>
    </ErrorBoundary>
  );
}

// Export wrapped lazy components
export function MiningGraphsLazy(props: { token?: string; isLoggedIn: boolean }) {
  return (
    <LazyWrapper componentName="Mining Graphs" loadingMessage="Loading charts...">
      <LazyMiningGraphs {...props} />
    </LazyWrapper>
  );
}

export function GlobalMinerMapLazy() {
  return (
    <LazyWrapper componentName="Global Miner Map" loadingMessage="Loading world map...">
      <LazyGlobalMinerMap />
    </LazyWrapper>
  );
}

export function UserDashboardLazy(props: { token: string }) {
  return (
    <LazyWrapper componentName="User Dashboard" loadingMessage="Loading dashboard...">
      <LazyUserDashboard {...props} />
    </LazyWrapper>
  );
}

export function WalletManagerLazy(props: { token: string; showMessage: (type: 'success' | 'error', text: string) => void }) {
  return (
    <LazyWrapper componentName="Wallet Manager" loadingMessage="Loading wallet settings...">
      <LazyWalletManager {...props} />
    </LazyWrapper>
  );
}

export function AuthModalLazy(props: {
  view: 'login' | 'register' | 'forgot-password' | 'reset-password';
  setView: (view: 'login' | 'register' | 'forgot-password' | 'reset-password' | null) => void;
  setToken: (token: string) => void;
  showMessage: (type: 'success' | 'error', text: string) => void;
  resetToken: string | null;
}) {
  return (
    <LazyWrapper componentName="Authentication" loadingMessage="Loading...">
      <LazyAuthModal {...props} />
    </LazyWrapper>
  );
}

// Re-export the wrapper for custom usage
export { LazyWrapper };
