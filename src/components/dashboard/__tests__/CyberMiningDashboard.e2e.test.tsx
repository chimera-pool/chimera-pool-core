import React from 'react';
import { render, screen } from '@testing-library/react';
import { CyberMiningDashboard } from '../CyberMiningDashboard';

// Mock fetch for API calls
global.fetch = jest.fn();

// Mock WebSocket with proper methods
const mockWebSocket = {
  send: jest.fn(),
  close: jest.fn(),
  addEventListener: jest.fn(),
  removeEventListener: jest.fn(),
  readyState: 1,
};

global.WebSocket = jest.fn().mockImplementation(() => mockWebSocket) as any;

describe('CyberMiningDashboard E2E', () => {
  beforeEach(() => {
    (fetch as jest.Mock).mockReset();
    (fetch as jest.Mock).mockResolvedValue({
      ok: true,
      json: () => Promise.resolve([]),
    });
    mockWebSocket.send.mockClear();
    mockWebSocket.close.mockClear();
    mockWebSocket.addEventListener.mockClear();
    mockWebSocket.removeEventListener.mockClear();
  });

  it('should render complete dashboard with all components', () => {
    render(<CyberMiningDashboard />);

    // Check main dashboard elements
    expect(screen.getByTestId('cyber-mining-dashboard')).toBeInTheDocument();
    expect(screen.getByText('CHIMERA_MINING_POOL')).toBeInTheDocument();

    // Check status cards
    expect(screen.getByText('POOL_HASHRATE')).toBeInTheDocument();
    expect(screen.getByText('ACTIVE_MINERS')).toBeInTheDocument();
    expect(screen.getByText('BLOCKS_FOUND')).toBeInTheDocument();
    expect(screen.getByText('ALGORITHM')).toBeInTheDocument();

    // Check miner controls
    expect(screen.getByText('MINER_CONTROL')).toBeInTheDocument();
    expect(screen.getByText('START_MINING')).toBeInTheDocument();
    expect(screen.getByText('STOP_MINING')).toBeInTheDocument();

    // Check achievement system
    expect(screen.getByTestId('achievement-system')).toBeInTheDocument();

    // Check leaderboard
    expect(screen.getByTestId('cyber-leaderboard')).toBeInTheDocument();

    // Check AI assistant
    expect(screen.getByTestId('ai-help-assistant')).toBeInTheDocument();
  });

  it('should be responsive and maintain cyber theme', () => {
    render(<CyberMiningDashboard />);

    const dashboard = screen.getByTestId('cyber-mining-dashboard');
    
    // Check cyber theme classes
    expect(dashboard).toHaveClass('cyber-mining-dashboard');
    
    // Check cyber-themed elements
    expect(screen.getByText('CHIMERA_MINING_POOL')).toBeInTheDocument();
  });

  it('should render miner stats section', () => {
    render(<CyberMiningDashboard />);
    
    expect(screen.getByText('YOUR_HASHRATE')).toBeInTheDocument();
    expect(screen.getByText('SHARES')).toBeInTheDocument();
    expect(screen.getByText('BALANCE')).toBeInTheDocument();
  });

  it('should have AI assistant toggle', () => {
    render(<CyberMiningDashboard />);

    const toggleButton = screen.getByTestId('ai-assistant-toggle');
    expect(toggleButton).toBeInTheDocument();
  });

  it('should display empty states initially', () => {
    render(<CyberMiningDashboard />);
    
    expect(screen.getByText('NO_ACHIEVEMENTS_YET')).toBeInTheDocument();
    expect(screen.getByText('NO_MINERS_YET')).toBeInTheDocument();
  });

  it('should create WebSocket connection', () => {
    render(<CyberMiningDashboard />);
    
    expect(global.WebSocket).toHaveBeenCalled();
  });
});