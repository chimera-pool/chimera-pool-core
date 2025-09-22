import React, { useState, useEffect } from 'react';
import { CyberButton } from '../cyber/CyberButton';
import { CyberStatusCard } from '../cyber/CyberStatusCard';
import { AchievementSystem } from '../gamification/AchievementSystem';
import { Leaderboard } from '../gamification/Leaderboard';
import { AIHelpAssistant } from '../ai/AIHelpAssistant';
import { useWebSocket } from '../../hooks/useWebSocket';
import { Achievement, LeaderboardEntry } from '../gamification/types';
import './CyberMiningDashboard.css';

export interface PoolStats {
  hashrate: string;
  miners: number;
  blocks: number;
  difficulty: string;
  algorithm: string;
  uptime: string;
}

export interface MinerStats {
  hashrate: string;
  shares: number;
  balance: string;
  status: 'online' | 'offline' | 'error';
}

export interface CyberMiningDashboardProps {
  className?: string;
}

export const CyberMiningDashboard: React.FC<CyberMiningDashboardProps> = ({
  className = '',
}) => {
  const [poolStats, setPoolStats] = useState<PoolStats>({
    hashrate: '0 H/s',
    miners: 0,
    blocks: 0,
    difficulty: '0',
    algorithm: 'BLAKE2S',
    uptime: '0h 0m',
  });

  const [minerStats, setMinerStats] = useState<MinerStats>({
    hashrate: '0 H/s',
    shares: 0,
    balance: '0.00000000',
    status: 'offline',
  });

  const [achievements, setAchievements] = useState<Achievement[]>([]);
  const [leaderboard, setLeaderboard] = useState<LeaderboardEntry[]>([]);
  const [notifications, setNotifications] = useState<string[]>([]);

  const { connectionState, sendMessage } = useWebSocket('ws://localhost:8080', {
    onMessage: (message) => {
      switch (message.type) {
        case 'POOL_STATS_UPDATE':
          setPoolStats(message.payload);
          break;
        case 'MINER_STATS_UPDATE':
          setMinerStats(message.payload);
          break;
        case 'ACHIEVEMENT_UNLOCKED':
          setAchievements(prev => prev.map(a => 
            a.id === message.payload.id 
              ? { ...a, unlocked: true, unlockedAt: new Date() }
              : a
          ));
          setNotifications(prev => [...prev, `Achievement unlocked: ${message.payload.title}`]);
          break;
        case 'LEADERBOARD_UPDATE':
          setLeaderboard(message.payload);
          break;
      }
    },
    onConnect: () => {
      sendMessage({ type: 'SUBSCRIBE_POOL_STATS' });
      sendMessage({ type: 'SUBSCRIBE_MINER_STATS' });
      sendMessage({ type: 'SUBSCRIBE_ACHIEVEMENTS' });
      sendMessage({ type: 'SUBSCRIBE_LEADERBOARD' });
    },
  });

  useEffect(() => {
    // Load initial data
    const loadInitialData = async () => {
      try {
        const [achievementsRes, leaderboardRes] = await Promise.all([
          fetch('/api/achievements'),
          fetch('/api/leaderboard'),
        ]);

        if (achievementsRes.ok) {
          const achievementsData = await achievementsRes.json();
          setAchievements(achievementsData);
        }

        if (leaderboardRes.ok) {
          const leaderboardData = await leaderboardRes.json();
          setLeaderboard(leaderboardData);
        }
      } catch (error) {
        console.error('Failed to load initial data:', error);
      }
    };

    loadInitialData();
  }, []);

  const handleStartMining = () => {
    sendMessage({ type: 'START_MINING' });
  };

  const handleStopMining = () => {
    sendMessage({ type: 'STOP_MINING' });
  };

  const dashboardClasses = [
    'cyber-mining-dashboard',
    className,
  ].filter(Boolean).join(' ');

  return (
    <div className={dashboardClasses} data-testid="cyber-mining-dashboard">
      {/* Cyber-themed header */}
      <div className="cyber-dashboard-header">
        <div className="cyber-title-container">
          <h1 className="cyber-dashboard-title">
            <span className="cyber-bracket">[</span>
            CHIMERA_MINING_POOL
            <span className="cyber-bracket">]</span>
          </h1>
          <div className="cyber-status-line">
            <span className="cyber-timestamp">{new Date().toISOString()}</span>
            <div className="cyber-connection-indicator">
              <div className={`cyber-pulse ${connectionState === 'connected' ? 'cyber-pulse--active' : ''}`}></div>
              {connectionState.toUpperCase()}
            </div>
          </div>
        </div>
      </div>

      {/* Pool Status Grid */}
      <div className="cyber-status-grid">
        <CyberStatusCard
          label="POOL_HASHRATE"
          value={poolStats.hashrate}
          status="healthy"
          icon="⚡"
        />
        <CyberStatusCard
          label="ACTIVE_MINERS"
          value={poolStats.miners.toString()}
          status="healthy"
          icon="👥"
        />
        <CyberStatusCard
          label="BLOCKS_FOUND"
          value={poolStats.blocks.toString()}
          status="healthy"
          icon="🎯"
        />
        <CyberStatusCard
          label="ALGORITHM"
          value={poolStats.algorithm}
          status="healthy"
          icon="🔧"
        />
      </div>

      {/* Miner Controls */}
      <div className="cyber-miner-controls">
        <div className="cyber-panel">
          <div className="cyber-panel-header">
            <h3 className="cyber-panel-title">
              <span className="cyber-icon">⛏️</span>
              MINER_CONTROL
            </h3>
          </div>
          <div className="cyber-panel-content">
            <div className="cyber-miner-stats">
              <div className="cyber-stat">
                <div className="cyber-stat-label">YOUR_HASHRATE</div>
                <div className="cyber-stat-value">{minerStats.hashrate}</div>
              </div>
              <div className="cyber-stat">
                <div className="cyber-stat-label">SHARES</div>
                <div className="cyber-stat-value">{minerStats.shares}</div>
              </div>
              <div className="cyber-stat">
                <div className="cyber-stat-label">BALANCE</div>
                <div className="cyber-stat-value">{minerStats.balance} KAS</div>
              </div>
            </div>
            
            <div className="cyber-control-buttons">
              <CyberButton
                variant="primary"
                onClick={handleStartMining}
                disabled={minerStats.status === 'online'}
                icon="▶️"
              >
                START_MINING
              </CyberButton>
              <CyberButton
                variant="secondary"
                onClick={handleStopMining}
                disabled={minerStats.status === 'offline'}
                icon="⏹️"
              >
                STOP_MINING
              </CyberButton>
            </div>
          </div>
        </div>
      </div>

      {/* Main Content Grid */}
      <div className="cyber-main-content">
        <div className="cyber-content-left">
          <AchievementSystem achievements={achievements} />
        </div>
        
        <div className="cyber-content-right">
          <Leaderboard entries={leaderboard} />
        </div>
      </div>

      {/* Notifications */}
      {notifications.length > 0 && (
        <div className="cyber-notifications">
          {notifications.map((notification, index) => (
            <div
              key={index}
              className="cyber-notification"
              data-testid={`notification-${index}`}
            >
              <div className="cyber-notification-content">
                {notification}
              </div>
              <button
                className="cyber-notification-close"
                onClick={() => setNotifications(prev => prev.filter((_, i) => i !== index))}
              >
                ✕
              </button>
            </div>
          ))}
        </div>
      )}

      {/* AI Help Assistant */}
      <AIHelpAssistant />

      {/* Cyber-themed background effects */}
      <div className="cyber-background-effects">
        <div className="cyber-grid-overlay"></div>
        <div className="cyber-scan-lines"></div>
      </div>
    </div>
  );
};