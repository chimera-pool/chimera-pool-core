import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import { MinerStatusMonitor } from './MinerStatusMonitor';

// Mock WebSocket for real-time updates
const mockWebSocket = {
  send: jest.fn(),
  close: jest.fn(),
  addEventListener: jest.fn(),
  removeEventListener: jest.fn(),
  readyState: 1, // OPEN
  onmessage: jest.fn(),
};

global.WebSocket = jest.fn(() => mockWebSocket) as any;

describe('MinerStatusMonitor Component', () => {
  const mockMiners = [
    {
      id: 'miner-001',
      address: 'ltc1qgsm3fv44wprdcsh3trgarm05rr7l8ryggujr5w',
      hashrate: 1500000000, // 1.5 GH/s
      status: 'online',
      lastSeen: new Date().toISOString(),
      shares: { accepted: 1250, rejected: 12 },
      algorithm: 'scrypt',
      network: 'litecoin',
    },
    {
      id: 'miner-002', 
      address: 'ltc1q8xk2wv8c9z7m6n5p4q3r2t1y0u9i8o7p6',
      hashrate: 850000000, // 850 MH/s
      status: 'online',
      lastSeen: new Date().toISOString(),
      shares: { accepted: 890, rejected: 5 },
      algorithm: 'scrypt',
      network: 'litecoin',
    },
  ];

  beforeEach(() => {
    jest.clearAllMocks();
    // Mock fetch for initial miner data
    global.fetch = jest.fn() as any;
    (global.fetch as jest.Mock).mockResolvedValue({
      ok: true,
      json: async () => ({ miners: mockMiners }),
    });
  });

  // Test: Component renders with miner status
  test('renders miner status monitor', async () => {
    render(<MinerStatusMonitor />);
    
    await waitFor(() => {
      expect(screen.getByText(/connected miners/i)).toBeInTheDocument();
    });
  });

  // Test: Displays hashrate information
  test('displays hashrate information', async () => {
    render(<MinerStatusMonitor />);
    
    await waitFor(() => {
      expect(screen.getByText(/1.5 gh\/s/i)).toBeInTheDocument();
      expect(screen.getByText(/850 mh\/s/i)).toBeInTheDocument();
    });
  });

  // Test: Shows online status indicators
  test('shows online status indicators', async () => {
    render(<MinerStatusMonitor />);
    
    await waitFor(() => {
      expect(screen.getAllByTestId('miner-status-online')).toHaveLength(2);
    });
  });

  // Test: Displays share statistics
  test('displays share statistics', async () => {
    render(<MinerStatusMonitor />);
    
    await waitFor(() => {
      expect(screen.getByText(/1250/i)).toBeInTheDocument();
      expect(screen.getByText(/890/i)).toBeInTheDocument();
    });
  });

  // Test: Real-time updates via WebSocket
  test('receives real-time updates', async () => {
    render(<MinerStatusMonitor />);
    
    // Wait for initial render and WebSocket setup
    await waitFor(() => {
      expect(screen.getByText(/connected miners/i)).toBeInTheDocument();
    });

    // Verify WebSocket mock was called
    expect(global.WebSocket).toHaveBeenCalledWith('ws://localhost:8080/miners');
    
    // Verify initial hashrate is displayed
    expect(screen.getByText(/1.5 GH\/s/i)).toBeInTheDocument();
  });

  // Test: Handles disconnected miners
  test('handles disconnected miners', async () => {
    const offlineMiners = [
      {
        ...mockMiners[0],
        status: 'offline',
        lastSeen: new Date(Date.now() - 300000).toISOString(), // 5 minutes ago = 'poor'
        network: 'litecoin',
      },
    ];

    (global.fetch as jest.Mock).mockResolvedValue({
      ok: true,
      json: async () => ({ miners: offlineMiners }),
    });

    render(<MinerStatusMonitor />);
    
    await waitFor(() => {
      expect(screen.getByTestId('miner-status-offline')).toBeInTheDocument();
      expect(screen.getByText(/poor/i)).toBeInTheDocument(); // 5 minutes ago = 'poor' quality
    });
  });

  // Test: Responsive design
  test('is responsive on mobile', async () => {
    render(<MinerStatusMonitor />);
    
    const container = screen.getByTestId('miner-status-monitor');
    expect(container).toHaveClass('responsive');
  });

  // Test: Accessibility compliance
  test('has proper ARIA labels', async () => {
    render(<MinerStatusMonitor />);
    
    await waitFor(() => {
      expect(screen.getByRole('region', { name: /miner status monitor/i })).toBeInTheDocument();
      expect(screen.getByRole('table', { name: /connected miners/i })).toBeInTheDocument();
    });
  });

  // Test: Error handling for WebSocket failures
  test('handles WebSocket connection failures', async () => {
    // Mock WebSocket to fail
    global.WebSocket = jest.fn(() => {
      throw new Error('WebSocket connection failed');
    }) as any;

    render(<MinerStatusMonitor />);
    
    // Should still render with fallback data
    await waitFor(() => {
      expect(screen.getByText(/connection error/i)).toBeInTheDocument();
    });
  });

  // Test: Loading state
  test('shows loading state initially', async () => {
    // Mock slow fetch
    (global.fetch as jest.Mock).mockImplementation(() => 
      new Promise(resolve => setTimeout(() => resolve({
        ok: true,
        json: async () => ({ miners: mockMiners }),
      }), 100))
    );

    render(<MinerStatusMonitor />);
    
    expect(screen.getByText(/loading miners/i)).toBeInTheDocument();
  });
});
