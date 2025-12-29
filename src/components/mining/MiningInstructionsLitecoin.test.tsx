import React from 'react';
import { render, screen, fireEvent, waitFor, within } from '@testing-library/react';
import { MiningInstructionsLitecoin } from './MiningInstructionsLitecoin';

// Mock the pool stats API
global.fetch = jest.fn();

describe('MiningInstructionsLitecoin Component', () => {
  const mockStats = {
    network: 'litecoin',
    currency: 'LTC',
    total_miners: 2,
    total_hashrate: 20000000000000,
    blocks_found: 0,
    minimum_payout: 0.01,
    payment_interval: '24h'
  };

  beforeEach(() => {
    (fetch as jest.Mock).mockClear();
    (fetch as jest.Mock).mockResolvedValueOnce({
      ok: true,
      json: async () => mockStats
    });
  });

  // Test: Component renders with Litecoin-specific instructions
  test('renders Litecoin mining instructions', async () => {
    render(<MiningInstructionsLitecoin />);
    
    expect(screen.getByText(/Connect Your Litecoin Miner/i)).toBeInTheDocument();
  });

  // Test: Displays correct stratum URL for Litecoin
  test('displays correct stratum URL for Litecoin', async () => {
    render(<MiningInstructionsLitecoin />);
    
    const connectionDetails = screen.getByTestId('connection-details');
    const stratumUrl = within(connectionDetails).getByText(/stratum\+tcp:\/\/206\.162\.80\.230:3333/i);
    expect(stratumUrl).toBeInTheDocument();
  });

  // Test: Shows Scrypt algorithm for Litecoin
  test('displays Scrypt algorithm for Litecoin', async () => {
    render(<MiningInstructionsLitecoin />);
    
    const connectionDetails = screen.getByTestId('connection-details');
    expect(within(connectionDetails).getByText(/scrypt/i)).toBeInTheDocument();
    expect(within(connectionDetails).getByText(/algorithm/i)).toBeInTheDocument();
  });

  // Test: Includes supported Litecoin mining software
  test('lists supported Litecoin mining software', async () => {
    render(<MiningInstructionsLitecoin />);
    
    const minerConfigs = screen.getByTestId('miner-configs');
    expect(within(minerConfigs).getAllByText(/cgminer/i)).toHaveLength(2);
    expect(within(minerConfigs).getAllByText(/bfgminer/i)).toHaveLength(2);
    expect(within(minerConfigs).getAllByText(/lolminer/i)).toHaveLength(2);
  });

  // Test: Shows example configuration for CGMiner
  test('displays CGMiner example configuration', async () => {
    render(<MiningInstructionsLitecoin />);
    
    const minerConfigs = screen.getByTestId('miner-configs');
    expect(within(minerConfigs).getAllByText(/cgminer/i)).toHaveLength(2);
    expect(within(minerConfigs).getByText(/-o stratum\+tcp:\/\//i)).toBeInTheDocument();
    expect(within(minerConfigs).getByText(/-u your@email\.com/i)).toBeInTheDocument();
  });

  // Test: Shows example configuration for BFGMiner
  test('displays BFGMiner example configuration', async () => {
    render(<MiningInstructionsLitecoin />);
    
    const minerConfigs = screen.getByTestId('miner-configs');
    expect(within(minerConfigs).getAllByText(/bfgminer/i)).toHaveLength(2);
    expect(within(minerConfigs).getByText(/--url stratum\+tcp:\/\//i)).toBeInTheDocument();
  });

  // Test: Includes troubleshooting section
  test('includes troubleshooting information', async () => {
    render(<MiningInstructionsLitecoin />);
    
    expect(screen.getByText(/troubleshooting/i)).toBeInTheDocument();
    expect(screen.getByText(/connection refused/i)).toBeInTheDocument();
    expect(screen.getByText(/authentication failed/i)).toBeInTheDocument();
  });

  // Test: Shows wallet address requirement
  test('shows wallet address reminder', async () => {
    render(<MiningInstructionsLitecoin />);
    
    const walletReminder = screen.getByTestId('wallet-reminder');
    expect(walletReminder).toBeInTheDocument();
    expect(within(walletReminder).getByText(/LTC/i)).toBeInTheDocument();
  });

  // Test: Displays network status
  test('displays Litecoin network status', async () => {
    render(<MiningInstructionsLitecoin />);
    
    const networkStatus = screen.getByTestId('network-status');
    expect(networkStatus).toBeInTheDocument();
    expect(networkStatus.querySelector('.status-active')).toBeInTheDocument();
  });

  // Test: Copy to clipboard functionality
  test('copies stratum URL to clipboard', async () => {
    const mockWriteText = jest.fn();
    Object.assign(navigator, {
      clipboard: {
        writeText: mockWriteText
      }
    });

    render(<MiningInstructionsLitecoin />);
    
    const copyButtons = screen.getAllByText(/copy/i);
    fireEvent.click(copyButtons[0]);

    expect(mockWriteText).toHaveBeenCalledWith('stratum+tcp://206.162.80.230:3333');
  });

  // Test: Responsive design
  test('is responsive on mobile', async () => {
    render(<MiningInstructionsLitecoin />);
    
    const container = screen.getByTestId('mining-instructions-container');
    expect(container).toHaveClass('responsive');
  });

  // Test: Accessibility compliance
  test('has proper ARIA labels', async () => {
    render(<MiningInstructionsLitecoin />);
    
    expect(screen.getByRole('region', { name: /mining instructions/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /copy pool address/i })).toBeInTheDocument();
  });
});
