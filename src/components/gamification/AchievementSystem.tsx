import React, { useState, useEffect } from 'react';
import { Achievement, AchievementType } from './types';
import './AchievementSystem.css';

export interface AchievementSystemProps {
  achievements: Achievement[];
}

export const AchievementSystem: React.FC<AchievementSystemProps> = ({
  achievements,
}) => {
  const [filter, setFilter] = useState<AchievementType | 'all'>('all');
  const [newlyUnlocked, setNewlyUnlocked] = useState<Achievement | null>(null);

  const totalPoints = achievements
    .filter(a => a.unlocked)
    .reduce((sum, a) => sum + a.points, 0);

  const filteredAchievements = achievements.filter(achievement => 
    filter === 'all' || achievement.type === filter
  );

  // Check for newly unlocked achievements
  useEffect(() => {
    const recentlyUnlocked = achievements.find(a => 
      a.unlocked && 
      a.unlockedAt && 
      Date.now() - a.unlockedAt.getTime() < 5000 // Within last 5 seconds
    );
    
    if (recentlyUnlocked) {
      setNewlyUnlocked(recentlyUnlocked);
      setTimeout(() => setNewlyUnlocked(null), 3000);
    }
  }, [achievements]);

  if (achievements.length === 0) {
    return (
      <div className="cyber-achievement-system" data-testid="achievement-system">
        <div className="cyber-achievement-empty">
          <div className="cyber-achievement-empty__title">NO_ACHIEVEMENTS_YET</div>
          <div className="cyber-achievement-empty__subtitle">Start mining to unlock achievements!</div>
        </div>
      </div>
    );
  }

  return (
    <div className="cyber-achievement-system" data-testid="achievement-system">
      <div className="cyber-achievement-header">
        <h2 className="cyber-achievement-title">ACHIEVEMENTS</h2>
        <div className="cyber-achievement-points">TOTAL_POINTS: {totalPoints}</div>
      </div>

      <div className="cyber-achievement-filters">
        <button
          className={`cyber-filter ${filter === 'all' ? 'cyber-filter--active' : ''}`}
          onClick={() => setFilter('all')}
        >
          ALL
        </button>
        <button
          className={`cyber-filter ${filter === AchievementType.MILESTONE ? 'cyber-filter--active' : ''}`}
          onClick={() => setFilter(AchievementType.MILESTONE)}
          data-testid="filter-milestone"
        >
          MILESTONES
        </button>
        <button
          className={`cyber-filter ${filter === AchievementType.PERFORMANCE ? 'cyber-filter--active' : ''}`}
          onClick={() => setFilter(AchievementType.PERFORMANCE)}
        >
          PERFORMANCE
        </button>
        <button
          className={`cyber-filter ${filter === AchievementType.RARE ? 'cyber-filter--active' : ''}`}
          onClick={() => setFilter(AchievementType.RARE)}
        >
          RARE
        </button>
      </div>

      <div className="cyber-achievement-grid">
        {filteredAchievements.map(achievement => (
          <div
            key={achievement.id}
            className={`cyber-achievement ${
              achievement.unlocked ? 'cyber-achievement--unlocked cyber-glow' : 'cyber-achievement--locked'
            } ${achievement.type === AchievementType.RARE ? 'cyber-achievement--rare' : ''}`}
            data-testid={`achievement-${achievement.id}`}
            onMouseEnter={() => {
              const tooltip = document.querySelector('[data-testid="achievement-tooltip"]') as HTMLElement;
              if (tooltip) {
                tooltip.style.display = 'block';
              }
            }}
            onMouseLeave={() => {
              const tooltip = document.querySelector('[data-testid="achievement-tooltip"]') as HTMLElement;
              if (tooltip) {
                tooltip.style.display = 'none';
              }
            }}
          >
            <div className="cyber-achievement__icon">{achievement.icon}</div>
            <div className="cyber-achievement__content">
              <div className="cyber-achievement__title">{achievement.title}</div>
              <div className="cyber-achievement__description">{achievement.description}</div>
              <div className="cyber-achievement__points">{achievement.points} PTS</div>
              
              {!achievement.unlocked && achievement.progress > 0 && (
                <div className="cyber-achievement__progress">
                  <div className="cyber-progress-bar">
                    <div 
                      className="cyber-progress-fill"
                      style={{ width: `${(achievement.progress / achievement.maxProgress) * 100}%` }}
                      data-testid={`progress-${achievement.id}`}
                    />
                  </div>
                  <div className="cyber-progress-text">
                    {Math.round((achievement.progress / achievement.maxProgress) * 100)}%
                  </div>
                </div>
              )}
            </div>
          </div>
        ))}
      </div>

      {/* Achievement unlock notification */}
      {newlyUnlocked && (
        <div 
          className="cyber-achievement-unlock"
          data-testid="achievement-unlock-notification"
        >
          <div className="cyber-achievement-unlock__content">
            <div className="cyber-achievement-unlock__icon">{newlyUnlocked.icon}</div>
            <div className="cyber-achievement-unlock__text">
              <div className="cyber-achievement-unlock__title">ACHIEVEMENT_UNLOCKED!</div>
              <div className="cyber-achievement-unlock__name">{newlyUnlocked.title}</div>
            </div>
          </div>
        </div>
      )}

      {/* Tooltip */}
      <div className="cyber-achievement-tooltip" data-testid="achievement-tooltip" style={{ display: 'none' }}>
        <div className="cyber-tooltip-content">
          {achievements.find(a => a.id === 'first-share')?.description || 'Achievement description'}
        </div>
      </div>
    </div>
  );
};