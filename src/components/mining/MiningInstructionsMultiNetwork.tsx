import React, { useState, useEffect } from 'react';

// Network configuration interface
interface NetworkConfig {
  id: string;
  name: string;
  symbol: string;
  algorithm: string;
  stratumPort: number;
  poolAddress: string;
  isActive: boolean;
  minerConfigs: MinerConfig[];
  walletPrefix: string;
  explorerUrl?: string;
}

interface MinerConfig {
  name: string;
  command: string;
  example: string;
  supported: boolean;
}

// Predefined network configurations
const NETWORK_CONFIGS: NetworkConfig[] = [
  {
    id: 'litecoin',
    name: 'Litecoin',
    symbol: 'LTC',
    algorithm: 'scrypt',
    stratumPort: 3333,
    poolAddress: 'stratum+tcp://206.162.80.230:3333',
    isActive: true,
    walletPrefix: 'ltc1',
    explorerUrl: 'https://blockchair.com/litecoin',
    minerConfigs: [
      { name: 'CGMiner', command: 'cgminer', example: 'cgminer -o stratum+tcp://206.162.80.230:3333 -u your@email.com -p yourpassword --scrypt', supported: true },
      { name: 'BFGMiner', command: 'bfgminer', example: 'bfgminer --url stratum+tcp://206.162.80.230:3333 --user your@email.com --pass yourpassword --scrypt', supported: true },
      { name: 'lolMiner', command: 'lolminer', example: 'lolminer --pool stratum+tcp://206.162.80.230:3333 --user your@email.com --pass yourpassword --algo scrypt', supported: true },
    ]
  },
  {
    id: 'bitcoin',
    name: 'Bitcoin',
    symbol: 'BTC',
    algorithm: 'sha256',
    stratumPort: 3334,
    poolAddress: 'stratum+tcp://206.162.80.230:3334',
    isActive: false,
    walletPrefix: 'bc1',
    explorerUrl: 'https://blockchair.com/bitcoin',
    minerConfigs: [
      { name: 'CGMiner', command: 'cgminer', example: 'cgminer -o stratum+tcp://206.162.80.230:3334 -u your@email.com -p yourpassword', supported: true },
      { name: 'BFGMiner', command: 'bfgminer', example: 'bfgminer --url stratum+tcp://206.162.80.230:3334 --user your@email.com --pass yourpassword', supported: true },
      { name: 'Antminer', command: 'antminer', example: 'Pool URL: stratum+tcp://206.162.80.230:3334\nWorker: your@email.com\nPassword: yourpassword', supported: true },
    ]
  },
  {
    id: 'ethereum',
    name: 'Ethereum Classic',
    symbol: 'ETC',
    algorithm: 'etchash',
    stratumPort: 3335,
    poolAddress: 'stratum+tcp://206.162.80.230:3335',
    isActive: false,
    walletPrefix: '0x',
    explorerUrl: 'https://blockscout.com/etc/mainnet',
    minerConfigs: [
      { name: 'lolMiner', command: 'lolminer', example: 'lolminer --pool stratum+tcp://206.162.80.230:3335 --user your@email.com --pass yourpassword --algo ETCHASH', supported: true },
      { name: 'T-Rex', command: 'trex', example: 't-rex -a etchash -o stratum+tcp://206.162.80.230:3335 -u your@email.com -p yourpassword', supported: true },
      { name: 'PhoenixMiner', command: 'phoenixminer', example: 'PhoenixMiner.exe -pool stratum+tcp://206.162.80.230:3335 -wal your@email.com -pass yourpassword', supported: true },
    ]
  },
  {
    id: 'kaspa',
    name: 'Kaspa',
    symbol: 'KAS',
    algorithm: 'kHeavyHash',
    stratumPort: 3336,
    poolAddress: 'stratum+tcp://206.162.80.230:3336',
    isActive: false,
    walletPrefix: 'kaspa:',
    explorerUrl: 'https://explorer.kaspa.org',
    minerConfigs: [
      { name: 'lolMiner', command: 'lolminer', example: 'lolminer --pool stratum+tcp://206.162.80.230:3336 --user your@email.com --pass yourpassword --algo KASPA', supported: true },
      { name: 'BzMiner', command: 'bzminer', example: 'bzminer -a kaspa -p stratum+tcp://206.162.80.230:3336 -w your@email.com', supported: true },
    ]
  },
  {
    id: 'ravencoin',
    name: 'Ravencoin',
    symbol: 'RVN',
    algorithm: 'kawpow',
    stratumPort: 3337,
    poolAddress: 'stratum+tcp://206.162.80.230:3337',
    isActive: false,
    walletPrefix: 'R',
    explorerUrl: 'https://ravencoin.network',
    minerConfigs: [
      { name: 'T-Rex', command: 'trex', example: 't-rex -a kawpow -o stratum+tcp://206.162.80.230:3337 -u your@email.com -p yourpassword', supported: true },
      { name: 'NBMiner', command: 'nbminer', example: 'nbminer -a kawpow -o stratum+tcp://206.162.80.230:3337 -u your@email.com', supported: true },
    ]
  },
];

// Styles
const styles: { [key: string]: React.CSSProperties } = {
  container: {
    background: '#111217',
    borderRadius: '12px',
    padding: '24px',
    marginBottom: '24px',
  },
  header: {
    marginBottom: '24px',
  },
  title: {
    fontSize: '1.5rem',
    color: '#F0EDF4',
    marginBottom: '8px',
    fontWeight: 600,
  },
  subtitle: {
    color: '#9A95A8',
    fontSize: '0.95rem',
  },
  networkSelector: {
    display: 'flex',
    gap: '8px',
    flexWrap: 'wrap' as const,
    marginBottom: '24px',
  },
  networkBtn: {
    padding: '12px 20px',
    backgroundColor: 'rgba(31, 20, 40, 0.8)',
    border: '1px solid #4A2C5A',
    borderRadius: '8px',
    color: '#B8B4C8',
    cursor: 'pointer',
    fontSize: '0.9rem',
    fontWeight: 500,
    transition: 'all 0.2s ease',
    display: 'flex',
    alignItems: 'center',
    gap: '8px',
  },
  networkBtnActive: {
    background: 'linear-gradient(135deg, #D4A84B 0%, #B8923A 100%)',
    color: '#1A0F1E',
    borderColor: '#D4A84B',
    boxShadow: '0 0 16px rgba(212, 168, 75, 0.3)',
  },
  networkBtnDisabled: {
    opacity: 0.5,
    cursor: 'not-allowed',
  },
  statusBadge: {
    padding: '2px 8px',
    borderRadius: '12px',
    fontSize: '0.7rem',
    fontWeight: 600,
    textTransform: 'uppercase' as const,
  },
  statusActive: {
    backgroundColor: 'rgba(115, 191, 105, 0.2)',
    color: '#73BF69',
  },
  statusInactive: {
    backgroundColor: 'rgba(154, 149, 168, 0.2)',
    color: '#9A95A8',
  },
  section: {
    background: '#181B1F',
    borderRadius: '8px',
    padding: '20px',
    marginBottom: '16px',
    border: '1px solid rgba(255, 255, 255, 0.08)',
  },
  sectionTitle: {
    fontSize: '1.1rem',
    color: '#F0EDF4',
    marginBottom: '16px',
    fontWeight: 600,
    display: 'flex',
    alignItems: 'center',
    gap: '8px',
  },
  connectionGrid: {
    display: 'grid',
    gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))',
    gap: '16px',
  },
  connectionItem: {
    display: 'flex',
    flexDirection: 'column' as const,
    gap: '4px',
  },
  label: {
    fontSize: '0.8rem',
    color: '#9A95A8',
    fontWeight: 500,
  },
  value: {
    fontSize: '0.95rem',
    color: '#F0EDF4',
    fontWeight: 500,
  },
  codeBlock: {
    background: '#0D0E11',
    padding: '12px 16px',
    borderRadius: '6px',
    fontFamily: 'monospace',
    fontSize: '0.85rem',
    color: '#D4A84B',
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    gap: '12px',
    overflowX: 'auto' as const,
  },
  copyBtn: {
    padding: '6px 12px',
    background: 'linear-gradient(135deg, #D4A84B 0%, #B8923A 100%)',
    border: 'none',
    borderRadius: '4px',
    color: '#1A0F1E',
    cursor: 'pointer',
    fontSize: '0.75rem',
    fontWeight: 600,
    flexShrink: 0,
  },
  minerCard: {
    background: '#1F2228',
    borderRadius: '8px',
    padding: '16px',
    marginBottom: '12px',
    border: '1px solid rgba(255, 255, 255, 0.05)',
  },
  minerName: {
    fontSize: '1rem',
    color: '#F0EDF4',
    fontWeight: 600,
    marginBottom: '8px',
  },
  alertBox: {
    background: 'rgba(212, 168, 75, 0.1)',
    border: '1px solid rgba(212, 168, 75, 0.3)',
    borderRadius: '8px',
    padding: '16px',
    marginTop: '16px',
  },
  alertTitle: {
    color: '#D4A84B',
    fontSize: '0.95rem',
    fontWeight: 600,
    marginBottom: '8px',
  },
  alertText: {
    color: '#B8B4C8',
    fontSize: '0.85rem',
    lineHeight: 1.5,
  },
  comingSoon: {
    textAlign: 'center' as const,
    padding: '40px',
    color: '#9A95A8',
  },
};

export interface MiningInstructionsMultiNetworkProps {
  className?: string;
  userEmail?: string;
  onCopySuccess?: (text: string) => void;
}

// Helper to replace placeholder email with actual user email
const replaceEmail = (text: string, userEmail?: string): string => {
  const email = userEmail || 'your@email.com';
  return text.replace(/your@email\.com/g, email);
};

export const MiningInstructionsMultiNetwork: React.FC<MiningInstructionsMultiNetworkProps> = ({
  className = '',
  userEmail,
  onCopySuccess
}) => {
  const [selectedNetwork, setSelectedNetwork] = useState<string>('litecoin');
  const [copied, setCopied] = useState<string | null>(null);
  const [expandedMiners, setExpandedMiners] = useState<Set<string>>(new Set(['CGMiner']));

  const network = NETWORK_CONFIGS.find(n => n.id === selectedNetwork) || NETWORK_CONFIGS[0];

  const copyToClipboard = async (text: string) => {
    try {
      await navigator.clipboard.writeText(text);
      setCopied(text);
      onCopySuccess?.(text);
      setTimeout(() => setCopied(null), 2000);
    } catch (err) {
      console.error('Failed to copy:', err);
    }
  };

  const toggleMiner = (minerName: string) => {
    setExpandedMiners(prev => {
      const newSet = new Set(prev);
      if (newSet.has(minerName)) {
        newSet.delete(minerName);
      } else {
        newSet.add(minerName);
      }
      return newSet;
    });
  };

  return (
    <div 
      className={`mining-instructions-multi ${className}`}
      data-testid="mining-instructions-multi-network"
      role="region"
      aria-label="Mining instructions for multiple networks"
      style={styles.container}
    >
      {/* Header */}
      <div style={styles.header}>
        <h2 style={styles.title} data-testid="instructions-title">Connect Your Miner</h2>
        <p style={styles.subtitle}>Select a network below to view connection instructions</p>
      </div>

      {/* Network Selector */}
      <div style={styles.networkSelector} data-testid="network-selector" role="tablist" aria-label="Select mining network">
        {NETWORK_CONFIGS.map((net) => (
          <button
            key={net.id}
            data-testid={`network-btn-${net.id}`}
            role="tab"
            aria-selected={selectedNetwork === net.id}
            aria-controls={`panel-${net.id}`}
            style={{
              ...styles.networkBtn,
              ...(selectedNetwork === net.id ? styles.networkBtnActive : {}),
              ...(!net.isActive && selectedNetwork !== net.id ? styles.networkBtnDisabled : {}),
            }}
            onClick={() => setSelectedNetwork(net.id)}
          >
            <span style={{ fontWeight: 600 }}>{net.symbol}</span>
            <span>{net.name}</span>
            <span style={{
              ...styles.statusBadge,
              ...(net.isActive ? styles.statusActive : styles.statusInactive),
            }}>
              {net.isActive ? 'Active' : 'Coming Soon'}
            </span>
          </button>
        ))}
      </div>

      {/* Network Content */}
      <div id={`panel-${network.id}`} role="tabpanel" aria-labelledby={`network-btn-${network.id}`}>
        {network.isActive ? (
          <>
            {/* Connection Details */}
            <div style={styles.section} data-testid="connection-details-section">
              <h3 style={styles.sectionTitle}>
                <span style={{ width: '10px', height: '10px', borderRadius: '50%', backgroundColor: '#73BF69' }} />
                Connection Details
              </h3>
              <div style={styles.connectionGrid}>
                <div style={styles.connectionItem}>
                  <span style={styles.label}>Pool Address</span>
                  <div style={styles.codeBlock}>
                    <code data-testid="pool-address">{network.poolAddress}</code>
                    <button 
                      style={styles.copyBtn}
                      onClick={() => copyToClipboard(network.poolAddress)}
                      aria-label="Copy pool address"
                      data-testid="copy-pool-address-btn"
                    >
                      {copied === network.poolAddress ? '✓ Copied' : 'Copy'}
                    </button>
                  </div>
                </div>
                <div style={styles.connectionItem}>
                  <span style={styles.label}>Algorithm</span>
                  <span style={styles.value}>{network.algorithm.toUpperCase()}</span>
                </div>
                <div style={styles.connectionItem}>
                  <span style={styles.label}>Port</span>
                  <span style={styles.value}>{network.stratumPort}</span>
                </div>
                <div style={styles.connectionItem}>
                  <span style={styles.label}>Currency</span>
                  <span style={styles.value}>{network.symbol}</span>
                </div>
              </div>
            </div>

            {/* Step-by-Step Guide */}
            <div style={styles.section} data-testid="step-by-step-guide">
              <h3 style={styles.sectionTitle}>
                <span style={{ width: '10px', height: '10px', borderRadius: '50%', backgroundColor: '#7B5EA7' }} />
                Step-by-Step Setup Guide
              </h3>
              <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
                <div style={{ display: 'flex', gap: '16px', alignItems: 'flex-start' }}>
                  <div style={{ width: '32px', height: '32px', borderRadius: '50%', background: 'linear-gradient(135deg, #D4A84B 0%, #B8923A 100%)', display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#1A0F1E', fontWeight: 700, flexShrink: 0 }}>1</div>
                  <div>
                    <h4 style={{ color: '#F0EDF4', margin: '0 0 4px 0', fontSize: '0.95rem' }}>Create an Account</h4>
                    <p style={{ color: '#9A95A8', margin: 0, fontSize: '0.85rem' }}>Register on Chimera Pool using your email address. This email will be your miner username.</p>
                  </div>
                </div>
                <div style={{ display: 'flex', gap: '16px', alignItems: 'flex-start' }}>
                  <div style={{ width: '32px', height: '32px', borderRadius: '50%', background: 'linear-gradient(135deg, #D4A84B 0%, #B8923A 100%)', display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#1A0F1E', fontWeight: 700, flexShrink: 0 }}>2</div>
                  <div>
                    <h4 style={{ color: '#F0EDF4', margin: '0 0 4px 0', fontSize: '0.95rem' }}>Add Your {network.symbol} Wallet Address</h4>
                    <p style={{ color: '#9A95A8', margin: 0, fontSize: '0.85rem' }}>Go to your Profile and add your {network.symbol} wallet address (starts with <code style={{ color: '#D4A84B' }}>{network.walletPrefix}</code>). This is where your mining rewards will be sent.</p>
                  </div>
                </div>
                <div style={{ display: 'flex', gap: '16px', alignItems: 'flex-start' }}>
                  <div style={{ width: '32px', height: '32px', borderRadius: '50%', background: 'linear-gradient(135deg, #D4A84B 0%, #B8923A 100%)', display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#1A0F1E', fontWeight: 700, flexShrink: 0 }}>3</div>
                  <div>
                    <h4 style={{ color: '#F0EDF4', margin: '0 0 4px 0', fontSize: '0.95rem' }}>Download Mining Software</h4>
                    <p style={{ color: '#9A95A8', margin: 0, fontSize: '0.85rem' }}>Download one of the supported mining programs below ({network.minerConfigs.map(m => m.name).join(', ')}). Extract the files to a folder on your computer.</p>
                  </div>
                </div>
                <div style={{ display: 'flex', gap: '16px', alignItems: 'flex-start' }}>
                  <div style={{ width: '32px', height: '32px', borderRadius: '50%', background: 'linear-gradient(135deg, #D4A84B 0%, #B8923A 100%)', display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#1A0F1E', fontWeight: 700, flexShrink: 0 }}>4</div>
                  <div>
                    <h4 style={{ color: '#F0EDF4', margin: '0 0 4px 0', fontSize: '0.95rem' }}>Configure Your Miner</h4>
                    <p style={{ color: '#9A95A8', margin: 0, fontSize: '0.85rem' }}>Copy the command from the "Mining Software Commands" section below. Replace <code style={{ color: '#D4A84B' }}>yourpassword</code> with any password you choose (or just use "x").</p>
                  </div>
                </div>
                <div style={{ display: 'flex', gap: '16px', alignItems: 'flex-start' }}>
                  <div style={{ width: '32px', height: '32px', borderRadius: '50%', background: 'linear-gradient(135deg, #D4A84B 0%, #B8923A 100%)', display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#1A0F1E', fontWeight: 700, flexShrink: 0 }}>5</div>
                  <div>
                    <h4 style={{ color: '#F0EDF4', margin: '0 0 4px 0', fontSize: '0.95rem' }}>Start Mining!</h4>
                    <p style={{ color: '#9A95A8', margin: 0, fontSize: '0.85rem' }}>Run the command in your terminal or create a batch file (.bat). Your miner should connect and you'll see your hashrate appear on the dashboard within a few minutes.</p>
                  </div>
                </div>
              </div>
            </div>

            {/* Miner Configurations */}
            <div style={styles.section} data-testid="miner-configs-section">
              <h3 style={styles.sectionTitle}>
                <span style={{ width: '10px', height: '10px', borderRadius: '50%', backgroundColor: '#D4A84B' }} />
                Mining Software Commands
              </h3>
              <p style={{ color: '#9A95A8', fontSize: '0.85rem', marginBottom: '16px' }}>
                {userEmail 
                  ? <>Your email (<code style={{ color: '#D4A84B' }}>{userEmail}</code>) is automatically filled in below. Just copy and run!</>
                  : <>Replace <code style={{ color: '#D4A84B' }}>your@email.com</code> with your registered email address.</>
                }
              </p>
              {network.minerConfigs.map((miner, idx) => {
                const minerCommand = replaceEmail(miner.example, userEmail);
                return (
                  <div key={idx} style={styles.minerCard} data-testid={`miner-card-${miner.name.toLowerCase()}`}>
                    <div 
                      style={{ ...styles.minerName, cursor: 'pointer', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}
                      onClick={() => toggleMiner(miner.name)}
                      role="button"
                      aria-expanded={expandedMiners.has(miner.name)}
                      tabIndex={0}
                      onKeyPress={(e) => e.key === 'Enter' && toggleMiner(miner.name)}
                    >
                      <span>{miner.name}</span>
                      <span style={{ color: '#9A95A8', fontSize: '0.9rem' }}>
                        {expandedMiners.has(miner.name) ? '▼' : '▶'}
                      </span>
                    </div>
                    {expandedMiners.has(miner.name) && (
                      <div style={styles.codeBlock}>
                        <code style={{ whiteSpace: 'pre-wrap', wordBreak: 'break-all' }}>{minerCommand}</code>
                        <button 
                          style={styles.copyBtn}
                          onClick={() => copyToClipboard(minerCommand)}
                          aria-label={`Copy ${miner.name} command`}
                          data-testid={`copy-${miner.name.toLowerCase()}-btn`}
                        >
                          {copied === minerCommand ? '✓ Copied' : 'Copy'}
                        </button>
                      </div>
                    )}
                  </div>
                );
              })}
            </div>

            {/* Troubleshooting */}
            <div style={styles.section} data-testid="troubleshooting-section">
              <h3 style={styles.sectionTitle}>
                <span style={{ width: '10px', height: '10px', borderRadius: '50%', backgroundColor: '#FF6B6B' }} />
                Troubleshooting Tips
              </h3>
              <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(280px, 1fr))', gap: '12px' }}>
                <div style={{ background: 'rgba(255, 107, 107, 0.1)', borderRadius: '8px', padding: '12px', border: '1px solid rgba(255, 107, 107, 0.2)' }}>
                  <h4 style={{ color: '#FF6B6B', margin: '0 0 4px 0', fontSize: '0.9rem' }}>Connection Refused</h4>
                  <p style={{ color: '#9A95A8', margin: 0, fontSize: '0.8rem' }}>Verify the pool address and port. Check your firewall isn't blocking port {network.stratumPort}.</p>
                </div>
                <div style={{ background: 'rgba(255, 107, 107, 0.1)', borderRadius: '8px', padding: '12px', border: '1px solid rgba(255, 107, 107, 0.2)' }}>
                  <h4 style={{ color: '#FF6B6B', margin: '0 0 4px 0', fontSize: '0.9rem' }}>Authentication Failed</h4>
                  <p style={{ color: '#9A95A8', margin: 0, fontSize: '0.8rem' }}>Make sure you're using your registered email as the username. Password can be anything.</p>
                </div>
                <div style={{ background: 'rgba(212, 168, 75, 0.1)', borderRadius: '8px', padding: '12px', border: '1px solid rgba(212, 168, 75, 0.2)' }}>
                  <h4 style={{ color: '#D4A84B', margin: '0 0 4px 0', fontSize: '0.9rem' }}>Shares Rejected</h4>
                  <p style={{ color: '#9A95A8', margin: 0, fontSize: '0.8rem' }}>Ensure you're using the correct algorithm ({network.algorithm.toUpperCase()}) for {network.name}.</p>
                </div>
                <div style={{ background: 'rgba(212, 168, 75, 0.1)', borderRadius: '8px', padding: '12px', border: '1px solid rgba(212, 168, 75, 0.2)' }}>
                  <h4 style={{ color: '#D4A84B', margin: '0 0 4px 0', fontSize: '0.9rem' }}>Low Hashrate</h4>
                  <p style={{ color: '#9A95A8', margin: 0, fontSize: '0.8rem' }}>Check your miner settings and ensure proper cooling. GPU/ASIC may need driver updates.</p>
                </div>
              </div>
            </div>

            {/* Wallet Reminder */}
            <div style={styles.alertBox} data-testid="wallet-reminder">
              <div style={styles.alertTitle}>⚠️ Important: Wallet Address Required</div>
              <p style={styles.alertText}>
                Make sure to add your {network.symbol} wallet address (starting with <code>{network.walletPrefix}</code>) 
                to your profile to receive payouts. Without a wallet address, you cannot receive mining rewards.
              </p>
            </div>
          </>
        ) : (
          /* Coming Soon State */
          <div style={styles.comingSoon} data-testid="coming-soon-message">
            <h3 style={{ color: '#F0EDF4', marginBottom: '16px' }}>{network.name} Mining Coming Soon</h3>
            <p style={{ marginBottom: '8px' }}>
              We're working on adding support for {network.name} ({network.symbol}) mining.
            </p>
            <p>
              Algorithm: <strong>{network.algorithm.toUpperCase()}</strong> | Port: <strong>{network.stratumPort}</strong>
            </p>
            <p style={{ marginTop: '16px', color: '#D4A84B' }}>
              Check back soon for updates!
            </p>
          </div>
        )}
      </div>
    </div>
  );
};

export default MiningInstructionsMultiNetwork;
