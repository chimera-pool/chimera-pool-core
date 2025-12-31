// ============================================================================
// COMMUNITY TYPES
// Type definitions for community components
// ============================================================================

export interface Channel {
  id: number;
  name: string;
  description: string;
  type: string;
  isReadOnly: boolean;
  adminOnlyPost: boolean;
}

export interface ChannelCategory {
  id: number;
  name: string;
  description?: string;
  channels: Channel[];
}

export interface Badge {
  icon: string;
  color: string;
  name: string;
  type?: string;
  isPrimary?: boolean;
}

export interface RoleBadge {
  icon: string;
  color: string;
  name: string;
  type: string;
}

export interface MessageReaction {
  emoji: string;
  name: string;
  count: number;
  hasReacted: boolean;
}

export interface ReactionType {
  id: number;
  emoji: string;
  name: string;
  category: string;
}

export interface ChatMessage {
  id: number;
  content: string;
  isEdited: boolean;
  createdAt: string;
  user: {
    id: number;
    username: string;
    role?: string;
    roleBadge?: RoleBadge | null;
    badgeIcon: string;
    badgeColor: string;
    badgeName?: string;
    badges?: Badge[];
  };
  replyToId?: number;
  reactions?: MessageReaction[];
}

export interface ForumCategory {
  id: number;
  name: string;
  description: string;
  icon: string;
  postCount: number;
}

export interface ForumPost {
  id: number;
  title: string;
  preview: string;
  tags: string[];
  viewCount: number;
  replyCount: number;
  upvotes: number;
  isPinned: boolean;
  isLocked: boolean;
  createdAt: string;
  author: {
    id: number;
    username: string;
    badgeIcon: string;
  };
}

export interface OnlineUser {
  id: number;
  username: string;
  status: string;
  badgeIcon: string;
}

export interface LeaderboardEntry {
  userId: number;
  username: string;
  rank: number;
  role: string;
  roleBadge?: RoleBadge;
  primaryBadge?: Badge;
  badges?: Badge[];
  stats?: {
    currentHashrate: number;
    activeMiners: number;
    blocksFound: number;
    totalShares: number;
    forumPosts: number;
    engagementScore: number;
  };
}

export interface LeaderboardPagination {
  totalUsers: number;
  totalPages: number;
}

export interface CommunityPageProps {
  token: string;
  user: any;
  showMessage: (type: 'success' | 'error', text: string) => void;
}

export type CommunityView = 'chat' | 'forums' | 'leaderboard';
export type LeaderboardType = 'hashrate' | 'shares' | 'blocks' | 'engagement' | 'forum';

// Referral System Types
export interface ReferralInfo {
  code: string;
  description: string;
  referrer_discount: number;
  referee_discount: number;
  times_used: number;
  max_uses: number | null;
  total_referrals: number;
  my_discount: number;
  effective_fee: number;
}

export interface Referral {
  username: string;
  status: 'pending' | 'confirmed' | 'expired' | 'cancelled';
  created_at: string;
  confirmed_at: string | null;
  total_shares: number;
  total_hashrate: number;
  clout_bonus: number;
}

export interface ChannelForm {
  name: string;
  description: string;
  category_id: string;
  type: string;
  is_read_only: boolean;
  admin_only_post: boolean;
}

export interface CategoryForm {
  name: string;
  description: string;
}
