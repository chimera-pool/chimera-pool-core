import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { CyberMiningDashboard } from '../CyberMiningDashboard';
import { it } from 'node:test';
import { it } from 'node:test';
import { it } from 'node:test';
import { it } from 'node:test';
import { it } from 'node:test';
import { it } from 'node:test';
import { it } from 'node:test';
import { it } from 'node:test';
import { it } from 'node:test';
import { it } from 'node:test';
import { it } from 'node:test';
import { it } from 'node:test';
import { it } from 'node:test';
import { it } from 'node:test';
import { it } from 'node:test';
import { it } from 'node:test';
import { it } from 'node:test';
import { beforeEach } from 'node:test';
import { describe } from 'node:test';

// Mock fetch for API calls
global.fetch = jest.fn();

// Mock WebSocket
const mockWebSocket = {
  send: jest.fn(),
  close: jest.fn(),
  addEventListener: jest.fn(),
  removeEventListener: jest.fn(),
  readyState: WebSocket.OPEN,
};

global.WebSocket = jest.fn(() => mockWebSocket) as any;

describe('CyberMiningDashboard', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    (fetch as jest.Mock).mockResolvedValue({
      ok: true,
      json: () => Promise.resolve([]),
    });
  });

  it('should render dashboard with cyber styling', () => {
    render(<CyberMiningDashboard />);
    
    expect(screen.getByTestId('cyber-mining-dashboard')).toHaveClass('cyber-mining-dashboard');
    expect(screen.getByText('CHIMERA_MINING_POOL')).toBeInTheDocument();
  });

  it('should display pool status cards', () => {
    render(<CyberMiningDashboard />);
    
    expect(screen.getByText('POOL_HASHRATE')).toBeInTheDocument();
    expect(screen.getByText('ACTIVE_MINERS')).toBeInTheDocument();
    expect(screen.getByText('BLOCKS_FOUND')).toBeInTheDocument();
    expect(screen.getByText('ALGORITHM')).toBeInTheDocument();
  });

  it('should display miner control panel', () => {
    render(<CyberMiningDashboard />);
    
    expect(screen.getByText('MINER_CONTROL')).toBeInTheDocument();
    expect(screen.getByText('YOUR_HASHRATE')).toBeInTheDocument();
    expect(screen.getByText('SHARES')).toBeInTheDocument();
    expect(screen.getByText('BALANCE')).toBeInTheDocument();
  });

  it('should have start and stop mining buttons', () => {
    render(<CyberMiningDashboard />);
    
    const startButton = screen.getByText('START_MINING');
    const stopButton = screen.getByText('STOP_MINING');
    
    expect(startButton).toBeInTheDocument();
    expect(stopButton).toBeInTheDocument();
    expect(stopButton).toBeDisabled(); // Should be disabled when offline
  });

  it('should handle start mining button click', () => {
    render(<CyberMiningDashboard />);
    
    // Simulate WebSocket connection
    const connectHandler = mockWebSocket.addEventListener.mock.calls.find(
      call => call[0] === 'open'
    )?.[1];
    
    if (connectHandler) {
      connectHandler();
    }
    
    const startButton = screen.getByText('START_MINING');
    fireEvent.click(startButton);
    
    expect(mockWebSocket.send).toHaveBeenCalledWith(
      JSON.stringify({ type: 'START_MINING' })
    );
  });

  it('should handle stop mining button click', () => {
    render(<CyberMiningDashboard />);
    
    // Simulate WebSocket connection
    const connectHandler = mockWebSocket.addEventListener.mock.calls.find(
      call => call[0] === 'open'
    )?.[1];
    
    if (connectHandler) {
      connectHandler();
    }
    
    const stopButton = screen.getByText('STOP_MINING');
    fireEvent.click(stopButton);
    
    expect(mockWebSocket.send).toHaveBeenCalledWith(
      JSON.stringify({ type: 'STOP_MINING' })
    );
  });

  it('should display connection status', () => {
    render(<CyberMiningDashboard />);
    
    expect(screen.getByText('CONNECTING')).toBeInTheDocument();
  });

  it('should update connection status when connected', () => {
    render(<CyberMiningDashboard />);
    
    // Simulate WebSocket connection
    const connectHandler = mockWebSocket.addEventListener.mock.calls.find(
      call => call[0] === 'open'
    )?.[1];
    
    if (connectHandler) {
      connectHandler();
    }
    
    expect(screen.getByText('CONNECTED')).toBeInTheDocument();
  });

  it('should handle WebSocket messages', () => {
    render(<CyberMiningDashboard />);
    
    const messageHandler = mockWebSocket.addEventListener.mock.calls.find(
      call => call[0] === 'message'
    )?.[1];
    
    const poolStatsMessage = {
      data: JSON.stringify({
        type: 'POOL_STATS_UPDATE',
        payload: {
          hashrate: '1.5 TH/s',
          miners: 42,
          blocks: 10,
          difficulty: '1000000',
          algorithm: 'BLAKE2S',
          uptime: '2h 30m',
        }
      })
    };
    
    if (messageHandler) {
      messageHandler(poolStatsMessage);
    }
    
    expect(screen.getByText('1.5 TH/s')).toBeInTheDocument();
    expect(screen.getByText('42')).toBeInTheDocument();
  });

  it('should display achievement unlocked notification', () => {
    render(<CyberMiningDashboard />);
    
    const messageHandler = mockWebSocket.addEventListener.mock.calls.find(
      call => call[0] === 'message'
    )?.[1];
    
    const achievementMessage = {
      data: JSON.stringify({
        type: 'ACHIEVEMENT_UNLOCKED',
        payload: {
          id: 'first-share',
          title: 'First Share',
        }
      })
    };
    
    if (messageHandler) {
      messageHandler(achievementMessage);
    }
    
    expect(screen.getByText('Achievement unlocked: First Share')).toBeInTheDocument();
  });

  it('should allow dismissing notifications', () => {
    render(<CyberMiningDashboard />);
    
    const messageHandler = mockWebSocket.addEventListener.mock.calls.find(
      call => call[0] === 'message'
    )?.[1];
    
    const achievementMessage = {
      data: JSON.stringify({
        type: 'ACHIEVEMENT_UNLOCKED',
        payload: {
          id: 'first-share',
          title: 'First Share',
        }
      })
    };
    
    if (messageHandler) {
      messageHandler(achievementMessage);
    }
    
    const notification = screen.getByTestId('notification-0');
    const closeButton = notification.querySelector('.cyber-notification-close');
    
    expect(closeButton).toBeInTheDocument();
    
    if (closeButton) {
      fireEvent.click(closeButton);
    }
    
    expect(screen.queryByText('Achievement unlocked: First Share')).not.toBeInTheDocument();
  });

  it('should load initial data on mount', async () => {
    const mockAchievements = [
      { id: 'test', title: 'Test Achievement', unlocked: false }
    ];
    const mockLeaderboard = [
      { rank: 1, username: 'TestUser', hashrate: '1 TH/s' }
    ];

    (fetch as jest.Mock)
      .mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve(mockAchievements),
      })
      .mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve(mockLeaderboard),
      });

    render(<CyberMiningDashboard />);

    await waitFor(() => {
      expect(fetch).toHaveBeenCalledWith('/api/achievements');
      expect(fetch).toHaveBeenCalledWith('/api/leaderboard');
    });
  });

  it('should handle API errors gracefully', async () => {
    (fetch as jest.Mock).mockRejectedValue(new Error('API Error'));
    
    const consoleSpy = jest.spyOn(console, 'error').mockImplementation();
    
    render(<CyberMiningDashboard />);
    
    await waitFor(() => {
      expect(consoleSpy).toHaveBeenCalledWith('Failed to load initial data:', expect.any(Error));
    });
    
    consoleSpy.mockRestore();
  });

  it('should display current timestamp', () => {
    const mockDate = new Date('2023-01-01T12:00:00Z');
    jest.spyOn(global, 'Date').mockImplementation(() => mockDate as any);
    
    render(<CyberMiningDashboard />);
    
    expect(screen.getByText('2023-01-01T12:00:00.000Z')).toBeInTheDocument();
    
    jest.restoreAllMocks();
  });

  it('should include AI help assistant', () => {
    render(<CyberMiningDashboard />);
    
    expect(screen.getByTestId('ai-help-assistant')).toBeInTheDocument();
  });

  it('should include achievement system', () => {
    render(<CyberMiningDashboard />);
    
    expect(screen.getByTestId('achievement-system')).toBeInTheDocument();
  });

  it('should include leaderboard', () => {
    render(<CyberMiningDashboard />);
    
    expect(screen.getByTestId('cyber-leaderboard')).toBeInTheDocument();
  });
});