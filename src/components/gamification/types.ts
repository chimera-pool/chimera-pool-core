export enum AchievementType {
  MILESTONE = 'milestone',
  PERFORMANCE = 'performance',
  RARE = 'rare',
  SOCIAL = 'social',
}

export interface Achievement {
  id: string;
  title: string;
  description: string;
  type: AchievementType;
  icon: string;
  points: number;
  unlocked: boolean;
  unlockedAt?: Date;
  progress: number;
  maxProgress: number;
}

export interface LeaderboardEntry {
  rank: number;
  username: string;
  hashrate: string;
  shares: number;
  blocks: number;
  points: number;
  badge: string;
  isCurrentUser: boolean;
}