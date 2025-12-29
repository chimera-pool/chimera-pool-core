// Interface Segregation Principle - Minimal, focused interfaces
export interface IMiningInstructionsProps {
  className?: string;
  showAdvanced?: boolean;
  onCopySuccess?: (text: string) => void;
}

export interface IMiningConnection {
  url: string;
  port: number;
  protocol: 'stratum+tcp' | 'stratum+ssl';
  algorithm: string;
  network: string;
  currency: string;
}

export interface IMinerConfig {
  name: string;
  command: string;
  example: string;
  supported: boolean;
}

export interface ITroubleshootingTip {
  issue: string;
  solution: string;
  priority: 'high' | 'medium' | 'low';
}

// Network type for multi-chain support
export type NetworkType = 'litecoin' | 'bitcoin' | 'ethereum' | 'dogecoin' | 'blockdag';

// Connection quality type
export type ConnectionQuality = 'excellent' | 'good' | 'poor' | 'offline';

// Miner interface for MinerStatusMonitor
export interface IMiner {
  id: string;
  address: string;
  hashrate: number;
  status: 'online' | 'offline';
  lastSeen: string;
  shares: { accepted: number; rejected: number };
  algorithm: string;
  network: NetworkType;
}

// Props for MinerStatusMonitor component
export interface IMinerStatusMonitorProps {
  className?: string;
  websocketUrl?: string;
  apiUrl?: string;
  network?: NetworkType;
}

export interface IMiningInstructionsState {
  loading: boolean;
  stats: any;
  copied: string | null;
  expandedSection: string | null;
}

export interface IMiningInstructionsActions {
  copyToClipboard: (text: string) => Promise<void>;
  toggleSection: (section: string) => void;
  fetchStats: () => Promise<void>;
}

// Combined interface for the component
export interface IMiningInstructionsLitecoin 
  extends IMiningInstructionsProps,
    IMiningInstructionsState,
    IMiningInstructionsActions {
  // Component-specific methods
  renderConnectionDetails: () => JSX.Element;
  renderMinerConfigs: () => JSX.Element;
  renderTroubleshooting: () => JSX.Element;
}
