import React, { useState } from 'react';
import { IMiningInstructionsProps } from './interfaces';

export const MiningInstructionsLitecoin: React.FC<IMiningInstructionsProps> = ({
  className = '',
  showAdvanced = false,
  onCopySuccess
}) => {
  const [copied, setCopied] = useState<string | null>(null);
  const [stats] = useState({
    total_miners: 2,
    total_hashrate: 20000000000,
    blocks_found: 0
  });

  const stratumUrl = 'stratum+tcp://206.162.80.230:3333';
  const algorithm = 'scrypt';
  const network = 'litecoin';
  const currency = 'LTC';

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

  const renderConnectionDetails = () => (
    <div className="connection-details" data-testid="connection-details">
      <h3>Connection Details</h3>
      <div className="connection-grid">
        <div className="connection-item">
          <label htmlFor="pool-address">Pool Address:</label>
          <code id="pool-address" className="pool-address" aria-label="Pool address">{stratumUrl}</code>
          <button 
            onClick={() => copyToClipboard(stratumUrl)}
            className="copy-button"
            aria-label="Copy pool address"
          >
            {copied === stratumUrl ? 'Copied!' : 'Copy'}
          </button>
        </div>
        <div className="connection-item">
          <label>Algorithm:</label>
          <span className="algorithm">{algorithm.toUpperCase()}</span>
        </div>
        <div className="connection-item">
          <label>Network:</label>
          <span className="network">{network}</span>
        </div>
        <div className="connection-item">
          <label>Currency:</label>
          <span className="currency">{currency}</span>
        </div>
      </div>
    </div>
  );

  const renderMinerConfigs = () => {
    const miners = [
      {
        name: 'CGMiner',
        command: 'cgminer',
        example: `cgminer -o ${stratumUrl} -u your@email.com -p yourpassword --scrypt`,
        supported: true
      },
      {
        name: 'BFGMiner', 
        command: 'bfgminer',
        example: `bfgminer --url ${stratumUrl} --user your@email.com --pass yourpassword --scrypt`,
        supported: true
      },
      {
        name: 'lolMiner',
        command: 'lolminer',
        example: `lolminer --pool ${stratumUrl} --user your@email.com --pass yourpassword --algo scrypt`,
        supported: true
      }
    ];

    return (
      <div className="miner-configs" data-testid="miner-configs">
        <h3>Supported Mining Software</h3>
        <div className="miners-grid">
          {miners.map((miner, index) => (
            <div key={index} className="miner-card">
              <h4>{miner.name}</h4>
              <div className="config-example">
                <code>{miner.example}</code>
                <button 
                  onClick={() => copyToClipboard(miner.example)}
                  className="copy-button"
                  aria-label={`Copy ${miner.name} configuration`}
                >
                  {copied === miner.example ? 'Copied!' : 'Copy'}
                </button>
              </div>
            </div>
          ))}
        </div>
      </div>
    );
  };

  const renderTroubleshooting = () => {
    const tips = [
      {
        issue: 'Connection Refused',
        solution: 'Verify the pool address and port. Check your firewall settings.',
        priority: 'high'
      },
      {
        issue: 'Authentication Failed',
        solution: 'Ensure you\'re using your registered email as username and correct password.',
        priority: 'high'
      },
      {
        issue: 'Shares Rejected',
        solution: 'Check your hashrate settings and ensure you\'re mining the correct algorithm.',
        priority: 'medium'
      },
      {
        issue: 'Low Hashrate',
        solution: 'Optimize your miner settings and ensure proper cooling.',
        priority: 'low'
      }
    ];

    return (
      <div className="troubleshooting" data-testid="troubleshooting">
        <h3>Troubleshooting</h3>
        <div className="tips-grid">
          {tips.map((tip, index) => (
            <div key={index} className={`tip-card priority-${tip.priority}`}>
              <h4>{tip.issue}</h4>
              <p>{tip.solution}</p>
            </div>
          ))}
        </div>
      </div>
    );
  };

  const renderNetworkStatus = () => (
    <div className="network-status" data-testid="network-status">
      <h3>Litecoin Network Status</h3>
      <div className="status-grid">
          <div className="status-item">
            <label>Status:</label>
            <span className="status-active">Active</span>
          </div>
          <div className="status-item">
            <label>Active Miners:</label>
            <span>{stats?.total_miners || 0}</span>
          </div>
          <div className="status-item">
            <label>Pool Hashrate:</label>
            <span>{stats?.total_hashrate ? (stats.total_hashrate / 1000000000).toFixed(2) + ' GH/s' : '0 GH/s'}</span>
          </div>
          <div className="status-item">
            <label>Blocks Found:</label>
            <span>{stats?.blocks_found || 0}</span>
          </div>
        </div>
    </div>
  );

  const renderStepByStepGuide = () => (
    <div className="step-by-step-guide" data-testid="step-by-step-guide">
      <h3>Step-by-Step Guide</h3>
      <div className="steps">
        <div className="step">
          <h4>Step 1: Register Account</h4>
          <p>Create an account on our platform using your email address.</p>
        </div>
        <div className="step">
          <h4>Step 2: Add Wallet Address</h4>
          <p>Navigate to your profile and add your LTC wallet address for payouts.</p>
        </div>
        <div className="step">
          <h4>Step 3: Configure Miner</h4>
          <p>Use one of the supported mining software configurations above.</p>
        </div>
        <div className="step">
          <h4>Step 4: Start Mining</h4>
          <p>Launch your miner and watch your hashrate appear in the dashboard.</p>
        </div>
      </div>
    </div>
  );

  return (
    <div 
      className={`mining-instructions-litecoin responsive ${className}`}
      data-testid="mining-instructions-container"
      role="region"
      aria-label="Mining instructions"
    >
      <h2>Connect Your Litecoin Miner</h2>
      <p>Configure your miner to connect to our Litecoin pool and start earning LTC rewards.</p>
      
      {renderConnectionDetails()}
      {renderStepByStepGuide()}
      {renderMinerConfigs()}
      {renderNetworkStatus()}
      {renderTroubleshooting()}
      
      <div className="wallet-reminder" data-testid="wallet-reminder">
        <h3>Important: Wallet Address Required</h3>
        <p>Make sure to add your LTC wallet address to your profile to receive payouts. Without a wallet address, you cannot receive mining rewards.</p>
      </div>
    </div>
  );
};
