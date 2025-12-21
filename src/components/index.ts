// ============================================================================
// CHIMERA POOL - COMPONENT EXPORTS
// Centralized exports for all extracted components
// ============================================================================

// Common Components
export { StatCard, ErrorBoundary, LoadingSpinner } from './common';
export type { StatCardProps, LoadingSpinnerProps } from './common';

// Auth Components
export { AuthModal } from './auth';
export type { AuthModalProps, AuthView } from './auth';

// Dashboard Components
export { UserDashboard } from './dashboard';
export type { UserDashboardProps, UserStats, Miner } from './dashboard';

// Chart Components
export { MiningGraphs } from './charts';
export type { MiningGraphsProps, TimeRange, ViewMode } from './charts';

// Map Components
export { GlobalMinerMap } from './maps';
export type { MinerLocation, LocationStats } from './maps';

// Wallet Components
export { WalletManager } from './wallet';
export type { WalletManagerProps, UserWallet, WalletSummary } from './wallet';
