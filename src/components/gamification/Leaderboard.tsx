import React, { useState } from 'react';
import { LeaderboardEntry } from './types';
import './Leaderboard.css';

export interface LeaderboardProps {
  entries: LeaderboardEntry[];
  loading?: boolean;
}

export const Leaderboard: React.FC<LeaderboardProps> = ({
  entries,
  loading = false,
}) => {
  const [filter, setFilter] = useState<'daily' | 'weekly' | 'monthly' | 'all'>('weekly');

  if (loading) {
    return (
      <div className="cyber-leaderboard" data-testid="cyber-leaderboard">
        <div className="cyber-loading" data-testid="leaderboard-loading">
          <div className="cyber-loading__text">LOADING_LEADERBOARD...</div>
          <div className="cyber-loading__animation">
            <div className="cyber-loading__bar"></div>
            <div className="cyber-loading__bar"></div>
            <div className="cyber-loading__bar"></div>
          </div>
        </div>
      </div>
    );
  }

  if (entries.length === 0) {
    return (
      <div className="cyber-leaderboard" data-testid="cyber-leaderboard">
        <div className="cyber-leaderboard-empty">
          <div className="cyber-leaderboard-empty__title">NO_MINERS_YET</div>
          <div className="cyber-leaderboard-empty__subtitle">Be the first to start mining!</div>
        </div>
      </div>
    );
  }

  const getRankClass = (rank: number) => {
    switch (rank) {
      case 1: return 'cyber-leaderboard-entry--gold';
      case 2: return 'cyber-leaderboard-entry--silver';
      case 3: return 'cyber-leaderboard-entry--bronze';
      default: return '';
    }
  };

  const getBadgeClass = (badge: string) => {
    return `cyber-badge--${badge.toLowerCase()}`;
  };

  const formatNumber = (num: number) => {
    return num.toLocaleString();
  };

  return (
    <div className="cyber-leaderboard" data-testid="cyber-leaderboard">
      <div className="cyber-leaderboard-header">
        <h2 className="cyber-leaderboard-title">LEADERBOARD</h2>
        
        <div className="cyber-leaderboard-filters">
          <button
            className={`cyber-filter ${filter === 'daily' ? 'cyber-filter--active' : ''}`}
            onClick={() => setFilter('daily')}
          >
            DAILY
          </button>
          <button
            className={`cyber-filter ${filter === 'weekly' ? 'cyber-filter--active' : ''}`}
            onClick={() => setFilter('weekly')}
            data-testid="filter-weekly"
          >
            WEEKLY
          </button>
          <button
            className={`cyber-filter ${filter === 'monthly' ? 'cyber-filter--active' : ''}`}
            onClick={() => setFilter('monthly')}
          >
            MONTHLY
          </button>
          <button
            className={`cyber-filter ${filter === 'all' ? 'cyber-filter--active' : ''}`}
            onClick={() => setFilter('all')}
          >
            ALL_TIME
          </button>
        </div>
      </div>

      <div className="cyber-leaderboard-list">
        {entries.map(entry => (
          <div
            key={entry.rank}
            className={`cyber-leaderboard-entry ${getRankClass(entry.rank)} ${
              entry.isCurrentUser ? 'cyber-leaderboard-entry--current-user cyber-glow' : ''
            }`}
            data-testid={`leaderboard-entry-${entry.rank}`}
          >
            <div className="cyber-leaderboard-entry__rank">
              <span className="cyber-rank-number">#{entry.rank}</span>
              {entry.rank <= 3 && (
                <span className="cyber-rank-medal">
                  {entry.rank === 1 ? '🥇' : entry.rank === 2 ? '🥈' : '🥉'}
                </span>
              )}
            </div>

            <div className="cyber-leaderboard-entry__user">
              <div className="cyber-leaderboard-entry__username">
                {entry.username}
                {entry.isCurrentUser && <span className="cyber-user-indicator">(YOU)</span>}
              </div>
              <div className={`cyber-badge ${getBadgeClass(entry.badge)}`}>
                {entry.badge}
              </div>
            </div>

            <div className="cyber-leaderboard-entry__stats">
              <div className="cyber-stat">
                <div className="cyber-stat__label">HASHRATE</div>
                <div className="cyber-stat__value">{entry.hashrate}</div>
              </div>
              
              <div className="cyber-stat">
                <div className="cyber-stat__label">SHARES</div>
                <div className="cyber-stat__value">{formatNumber(entry.shares)}</div>
              </div>
              
              <div className="cyber-stat">
                <div className="cyber-stat__label">BLOCKS</div>
                <div 
                  className="cyber-stat__value cyber-blocks-count"
                  data-testid={`blocks-count-${entry.rank}`}
                >
                  {entry.blocks}
                </div>
              </div>
              
              <div className="cyber-stat">
                <div className="cyber-stat__label">POINTS</div>
                <div className="cyber-stat__value">{formatNumber(entry.points)} PTS</div>
              </div>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
};