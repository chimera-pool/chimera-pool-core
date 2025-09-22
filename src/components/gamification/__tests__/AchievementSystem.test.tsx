import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { AchievementSystem } from '../AchievementSystem';
import { Achievement, AchievementType } from '../types';

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
  {
    id: 'hash-warrior',
    title: 'HASH_WARRIOR',
    description: 'Maintain 1 TH/s for 24 hours',
    type: AchievementType.PERFORMANCE,
    icon: '⚔️',
    points: 500,
    unlocked: false,
    progress: 0.7,
    maxProgress: 1,
  },
  {
    id: 'block-finder',
    title: 'BLOCK_FINDER',
    description: 'Find your first block',
    type: AchievementType.RARE,
    icon: '💎',
    points: 1000,
    unlocked: false,
    progress: 0,
    maxProgress: 1,
  },
];

describe('AchievementSystem', () => {
  it('should render achievement system with cyber styling', () => {
    render(<AchievementSystem achievements={mockAchievements} />);
    
    expect(screen.getByTestId('achievement-system')).toHaveClass('cyber-achievement-system');
    expect(screen.getByText('ACHIEVEMENTS')).toBeInTheDocument();
  });

  it('should display unlocked achievements with cyber glow effect', () => {
    render(<AchievementSystem achievements={mockAchievements} />);
    
    const unlockedAchievement = screen.getByTestId('achievement-first-share');
    expect(unlockedAchievement).toHaveClass('cyber-achievement--unlocked');
    expect(unlockedAchievement).toHaveClass('cyber-glow');
  });

  it('should display locked achievements with muted styling', () => {
    render(<AchievementSystem achievements={mockAchievements} />);
    
    const lockedAchievement = screen.getByTestId('achievement-hash-warrior');
    expect(lockedAchievement).toHaveClass('cyber-achievement--locked');
    expect(lockedAchievement).not.toHaveClass('cyber-glow');
  });

  it('should show progress bars for achievements in progress', () => {
    render(<AchievementSystem achievements={mockAchievements} />);
    
    const progressBar = screen.getByTestId('progress-hash-warrior');
    expect(progressBar).toBeInTheDocument();
    expect(progressBar).toHaveStyle('width: 70%');
  });

  it('should display achievement points with cyber formatting', () => {
    render(<AchievementSystem achievements={mockAchievements} />);
    
    expect(screen.getByText('100 PTS')).toBeInTheDocument();
    expect(screen.getByText('500 PTS')).toBeInTheDocument();
    expect(screen.getByText('1000 PTS')).toBeInTheDocument();
  });

  it('should show total points earned', () => {
    render(<AchievementSystem achievements={mockAchievements} />);
    
    expect(screen.getByText('TOTAL_POINTS: 100')).toBeInTheDocument();
  });

  it('should filter achievements by type', () => {
    render(<AchievementSystem achievements={mockAchievements} />);
    
    const milestoneFilter = screen.getByTestId('filter-milestone');
    fireEvent.click(milestoneFilter);
    
    expect(screen.getByTestId('achievement-first-share')).toBeInTheDocument();
    expect(screen.queryByTestId('achievement-hash-warrior')).not.toBeInTheDocument();
  });

  it('should show achievement unlock animation', async () => {
    const { rerender } = render(<AchievementSystem achievements={mockAchievements} />);
    
    const updatedAchievements = mockAchievements.map(a => 
      a.id === 'hash-warrior' 
        ? { ...a, unlocked: true, unlockedAt: new Date() }
        : a
    );
    
    rerender(<AchievementSystem achievements={updatedAchievements} />);
    
    await waitFor(() => {
      const notification = screen.getByTestId('achievement-unlock-notification');
      expect(notification).toBeInTheDocument();
      expect(notification).toHaveClass('cyber-achievement-unlock');
    });
  });

  it('should display rare achievements with special styling', () => {
    render(<AchievementSystem achievements={mockAchievements} />);
    
    const rareAchievement = screen.getByTestId('achievement-block-finder');
    expect(rareAchievement).toHaveClass('cyber-achievement--rare');
  });

  it('should show achievement details on hover', async () => {
    render(<AchievementSystem achievements={mockAchievements} />);
    
    const achievement = screen.getByTestId('achievement-first-share');
    fireEvent.mouseEnter(achievement);
    
    await waitFor(() => {
      const tooltip = screen.getByTestId('achievement-tooltip');
      expect(tooltip).toBeInTheDocument();
      expect(tooltip.style.display).toBe('block');
    });
  });

  it('should handle empty achievements list', () => {
    render(<AchievementSystem achievements={[]} />);
    
    expect(screen.getByText('NO_ACHIEVEMENTS_YET')).toBeInTheDocument();
    expect(screen.getByText('Start mining to unlock achievements!')).toBeInTheDocument();
  });
});