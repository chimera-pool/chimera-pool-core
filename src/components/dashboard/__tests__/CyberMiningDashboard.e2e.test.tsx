import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { CyberMiningDashboard } from '../CyberMiningDashboard';
import { Achievement, AchievementType, LeaderboardEntry } from '../../gamification/types';

// Mock fetch for API calls
global.fetch = jest.fn();

// Mock WebSocket with proper methods
const mockWebSocket = {
  send: jest.fn(),
  close: jest.fn(),
  addEventListener: jest.fn(),
  removeEventListener: jest.fn(),
  readyState: WebSocket.OPEN,
};

global.WebSocket = jest.fn(() => mockWebSocket) as any;

describe('CyberMiningDashboard E2E', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    (fetch as jest.Mock).mockResolvedValue({
      ok: true,
      json: () => Promise.resolve([]),
    });
  });

  it('should render complete dashboard with all components', async () => {
    const mockAchievements: Achievement[] = [
      {
        id: 'first-share',
        title: 'FIRST_SHARE',
        description: 'Submit your first valid share',
        type: AchievementType.MILESTONE,
        icon: '🎯',
        points: 100,
        unlocked: true,
        unlockedAt: new Date('2023-01-01'),
        progress: 1,
        maxProgress: 1,
      },
    ];

    const mockLeaderboard: LeaderboardEntry[] = [
      {
        rank: 1,
        username: 'CYBER_MINER_01',
        hashrate: '2.5 TH/s',
        shares: 15420,
        blocks: 3,
        points: 5000,
        badge: 'LEGEND',
        isCurrentUser: false,
      },
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

    // Wait for data to load
    await waitFor(() => {
      expect(fetch).toHaveBeenCalledWith('/api/achievements');
      expect(fetch).toHaveBeenCalledWith('/api/leaderboard');
    });

    // Check achievement system
    expect(screen.getByTestId('achievement-system')).toBeInTheDocument();

    // Check leaderboard
    expect(screen.getByTestId('cyber-leaderboard')).toBeInTheDocument();

    // Check AI assistant
    expect(screen.getByTestId('ai-help-assistant')).toBeInTheDocument();
  });

  it('should handle complete mining workflow', async () => {
    render(<CyberMiningDashboard />);

    // Simulate WebSocket connection
    const connectHandler = mockWebSocket.addEventListener.mock.calls.find(
      call => call[0] === 'open'
    )?.[1];

    if (connectHandler) {
      connectHandler();
    }

    // Check connection status
    expect(screen.getByText('CONNECTED')).toBeInTheDocument();

    // Start mining
    const startButton = screen.getByText('START_MINING');
    fireEvent.click(startButton);

    expect(mockWebSocket.send).toHaveBeenCalledWith(
      JSON.stringify({ type: 'START_MINING' })
    );

    // Simulate pool stats update
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

    // Check updated stats
    expect(screen.getByText('1.5 TH/s')).toBeInTheDocument();
    expect(screen.getByText('42')).toBeInTheDocument();
    expect(screen.getByText('10')).toBeInTheDocument();

    // Simulate miner stats update
    const minerStatsMessage = {
      data: JSON.stringify({
        type: 'MINER_STATS_UPDATE',
        payload: {
          hashrate: '500 MH/s',
          shares: 150,
          balance: '0.00123456',
          status: 'online',
        }
      })
    };

    if (messageHandler) {
      messageHandler(minerStatsMessage);
    }

    // Check miner stats
    expect(screen.getByText('500 MH/s')).toBeInTheDocument();
    expect(screen.getByText('150')).toBeInTheDocument();
    expect(screen.getByText('0.00123456 KAS')).toBeInTheDocument();

    // Stop mining
    const stopButton = screen.getByText('STOP_MINING');
    fireEvent.click(stopButton);

    expect(mockWebSocket.send).toHaveBeenCalledWith(
      JSON.stringify({ type: 'STOP_MINING' })
    );
  });

  it('should handle achievement unlocking with notifications', async () => {
    render(<CyberMiningDashboard />);

    // Simulate WebSocket connection
    const connectHandler = mockWebSocket.addEventListener.mock.calls.find(
      call => call[0] === 'open'
    )?.[1];

    if (connectHandler) {
      connectHandler();
    }

    // Simulate achievement unlock
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

    // Check notification appears
    expect(screen.getByText('Achievement unlocked: First Share')).toBeInTheDocument();

    // Dismiss notification
    const notification = screen.getByTestId('notification-0');
    const closeButton = notification.querySelector('.cyber-notification-close');

    if (closeButton) {
      fireEvent.click(closeButton);
    }

    // Check notification is removed
    expect(screen.queryByText('Achievement unlocked: First Share')).not.toBeInTheDocument();
  });

  it('should handle AI assistant interaction', async () => {
    render(<CyberMiningDashboard />);

    // Open AI assistant
    const toggleButton = screen.getByTestId('ai-assistant-toggle');
    fireEvent.click(toggleButton);

    // Check AI interface is visible
    expect(screen.getByTestId('ai-chat-interface')).toBeInTheDocument();
    expect(screen.getByText('Hello! I\'m your mining assistant. How can I help you today?')).toBeInTheDocument();

    // Check quick help buttons
    expect(screen.getByText('SETUP_HELP')).toBeInTheDocument();
    expect(screen.getByText('PERFORMANCE_TIPS')).toBeInTheDocument();
    expect(screen.getByText('TROUBLESHOOTING')).toBeInTheDocument();
    expect(screen.getByText('PAYOUT_INFO')).toBeInTheDocument();
  });

  it('should be responsive and maintain cyber theme', () => {
    render(<CyberMiningDashboard />);

    const dashboard = screen.getByTestId('cyber-mining-dashboard');
    
    // Check cyber theme classes
    expect(dashboard).toHaveClass('cyber-mining-dashboard');
    
    // Check cyber-themed elements
    expect(screen.getByText('CHIMERA_MINING_POOL')).toBeInTheDocument();
    
    // Check brackets in title
    const brackets = screen.getAllByText(/[\[\]]/);
    expect(brackets.length).toBeGreaterThan(0);
    
    // Check connection indicator
    expect(screen.getByText('CONNECTING')).toBeInTheDocument();
  });

  it('should handle errors gracefully', async () => {
    // Mock API error
    (fetch as jest.Mock).mockRejectedValue(new Error('API Error'));
    
    const consoleSpy = jest.spyOn(console, 'error').mockImplementation();
    
    render(<CyberMiningDashboard />);
    
    await waitFor(() => {
      expect(consoleSpy).toHaveBeenCalledWith('Failed to load initial data:', expect.any(Error));
    });
    
    // Dashboard should still render
    expect(screen.getByTestId('cyber-mining-dashboard')).toBeInTheDocument();
    
    consoleSpy.mockRestore();
  });
});