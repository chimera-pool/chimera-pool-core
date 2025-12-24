/**
 * Admin Panel Hooks - ISP Compliant State Management
 * 
 * Each hook manages state for a specific tab, preventing
 * cross-tab re-renders and keeping the AdminPanel performant.
 */

export { useAdminStatsTab } from './useAdminStatsTab';
export type { 
  TimeRange,
  HashrateDataPoint,
  SharesDataPoint,
  MinersDataPoint,
  PayoutsDataPoint,
  DistributionDataPoint,
  AdminStatsTabState,
} from './useAdminStatsTab';
