import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import { Leaderboard } from '../Leaderboard';
import { LeaderboardEntry } from '../types';

const mockLeaderboardData: LeaderboardEntry[] = [
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
  {
    rank: 2,
    username: 'HASH_MASTER',
    hashrate: '1.8 TH/s',
    shares: 12100,
    blocks: 2,
    points: 3800,
    badge: 'EXPERT',
    isCurrentUser: false,
  },
  {
    rank: 3,
    username: 'CURRENT_USER',
    hashrate: '1.2 TH/s',
    shares: 8500,
    blocks: 1,
    points: 2500,
    badge: 'ADVANCED',
    isCurrentUser: true,
  },
];

describe('Leaderboard', () => {
  it('should render leaderboard with cyber styling', () => {
    render(<Leaderboard entries={mockLeaderboardData} />);
    
    expect(screen.getByTestId('cyber-leaderboard')).toHaveClass('cyber-leaderboard');
    expect(screen.getByText('LEADERBOARD')).toBeInTheDocument();
  });

  it('should display leaderboard entries in rank order', () => {
    render(<Leaderboard entries={mockLeaderboardData} />);
    
    const entries = screen.getAllByTestId(/leaderboard-entry-/);
    expect(entries).toHaveLength(3);
    
    expect(screen.getByText('CYBER_MINER_01')).toBeInTheDocument();
    expect(screen.getByText('HASH_MASTER')).toBeInTheDocument();
    expect(screen.getByText('CURRENT_USER')).toBeInTheDocument();
  });

  it('should highlight current user entry', () => {
    render(<Leaderboard entries={mockLeaderboardData} />);
    
    const currentUserEntry = screen.getByTestId('leaderboard-entry-3');
    expect(currentUserEntry).toHaveClass('cyber-leaderboard-entry--current-user');
    expect(currentUserEntry).toHaveClass('cyber-glow');
  });

  it('should display rank with cyber formatting', () => {
    render(<Leaderboard entries={mockLeaderboardData} />);
    
    expect(screen.getByText('#1')).toBeInTheDocument();
    expect(screen.getByText('#2')).toBeInTheDocument();
    expect(screen.getByText('#3')).toBeInTheDocument();
  });

  it('should show special styling for top 3 ranks', () => {
    render(<Leaderboard entries={mockLeaderboardData} />);
    
    const firstPlace = screen.getByTestId('leaderboard-entry-1');
    const secondPlace = screen.getByTestId('leaderboard-entry-2');
    const thirdPlace = screen.getByTestId('leaderboard-entry-3');
    
    expect(firstPlace).toHaveClass('cyber-leaderboard-entry--gold');
    expect(secondPlace).toHaveClass('cyber-leaderboard-entry--silver');
    expect(thirdPlace).toHaveClass('cyber-leaderboard-entry--bronze');
  });

  it('should display hashrate with proper formatting', () => {
    render(<Leaderboard entries={mockLeaderboardData} />);
    
    expect(screen.getByText('2.5 TH/s')).toBeInTheDocument();
    expect(screen.getByText('1.8 TH/s')).toBeInTheDocument();
    expect(screen.getByText('1.2 TH/s')).toBeInTheDocument();
  });

  it('should display shares count with cyber formatting', () => {
    render(<Leaderboard entries={mockLeaderboardData} />);
    
    expect(screen.getByText('15,420')).toBeInTheDocument();
    expect(screen.getByText('12,100')).toBeInTheDocument();
    expect(screen.getByText('8,500')).toBeInTheDocument();
  });

  it('should display blocks found with special highlighting', () => {
    render(<Leaderboard entries={mockLeaderboardData} />);
    
    const blockCounts = screen.getAllByTestId(/blocks-count-/);
    blockCounts.forEach(element => {
      expect(element).toHaveClass('cyber-blocks-count');
    });
  });

  it('should display user badges with appropriate styling', () => {
    render(<Leaderboard entries={mockLeaderboardData} />);
    
    const legendBadge = screen.getByText('LEGEND');
    const expertBadge = screen.getByText('EXPERT');
    const advancedBadge = screen.getByText('ADVANCED');
    
    expect(legendBadge).toHaveClass('cyber-badge--legend');
    expect(expertBadge).toHaveClass('cyber-badge--expert');
    expect(advancedBadge).toHaveClass('cyber-badge--advanced');
  });

  it('should filter leaderboard by time period', () => {
    render(<Leaderboard entries={mockLeaderboardData} />);
    
    const weeklyFilter = screen.getByTestId('filter-weekly');
    fireEvent.click(weeklyFilter);
    
    expect(weeklyFilter).toHaveClass('cyber-filter--active');
  });

  it('should handle empty leaderboard', () => {
    render(<Leaderboard entries={[]} />);
    
    expect(screen.getByText('NO_MINERS_YET')).toBeInTheDocument();
    expect(screen.getByText('Be the first to start mining!')).toBeInTheDocument();
  });

  it('should show loading state with cyber animation', () => {
    render(<Leaderboard entries={[]} loading />);
    
    const loadingElement = screen.getByTestId('leaderboard-loading');
    expect(loadingElement).toHaveClass('cyber-loading');
    expect(screen.getByText('LOADING_LEADERBOARD...')).toBeInTheDocument();
  });

  it('should display points with cyber formatting', () => {
    render(<Leaderboard entries={mockLeaderboardData} />);
    
    expect(screen.getByText('5,000 PTS')).toBeInTheDocument();
    expect(screen.getByText('3,800 PTS')).toBeInTheDocument();
    expect(screen.getByText('2,500 PTS')).toBeInTheDocument();
  });
});