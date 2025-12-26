/**
 * User Mining Dashboard Interfaces
 * Following Interface Segregation Principle (ISP)
 * 
 * Separates concerns:
 * - IUserEquipmentStatus: Equipment connectivity state
 * - IUserMiningStats: Mining statistics
 * - IUserDashboardVisibility: Dashboard display logic
 */

/**
 * Equipment status for a single mining device
 */
export interface IEquipmentDevice {
  id: string | number;
  name: string;
  type: 'asic' | 'gpu' | 'cpu' | 'unknown';
  status: 'mining' | 'online' | 'idle' | 'offline' | 'error';
  hashrate: number;
  lastSeen: Date | string;
  isActive: boolean;
}

/**
 * Aggregated user equipment status
 * Used to determine dashboard visibility
 */
export interface IUserEquipmentStatus {
  /** Whether user has any registered equipment */
  hasEquipment: boolean;
  /** Whether any equipment is actively mining (status: mining, online, idle) */
  hasActiveEquipment: boolean;
  /** Total count of registered equipment */
  totalEquipmentCount: number;
  /** Count of equipment currently active/online */
  activeEquipmentCount: number;
  /** Whether user has a pending support request */
  hasPendingSupport: boolean;
  /** Loading state */
  isLoading: boolean;
  /** Error state */
  error: string | null;
}

/**
 * User mining statistics (separate from equipment status)
 */
export interface IUserMiningStats {
  totalHashrate: number;
  totalShares: number;
  validShares: number;
  invalidShares: number;
  successRate: number;
  pendingPayout: number;
  totalEarnings: number;
}

/**
 * Dashboard visibility state
 * Determines how the "Your Mining Dashboard" section displays
 */
export interface IUserDashboardVisibility {
  /** Whether dashboard section should be shown at all */
  shouldShow: boolean;
  /** Whether dashboard should be expanded or collapsed */
  isExpanded: boolean;
  /** Whether user can manually toggle expansion */
  canToggle: boolean;
  /** Message to show when collapsed */
  collapsedMessage: string;
}

/**
 * Props for the collapsible user mining dashboard
 */
export interface ICollapsibleDashboardProps {
  token: string;
  equipmentStatus: IUserEquipmentStatus;
  onToggleExpand?: () => void;
  forceExpanded?: boolean;
}

/**
 * Determines dashboard visibility based on equipment status
 */
export function getDashboardVisibility(
  equipmentStatus: IUserEquipmentStatus,
  isLoggedIn: boolean
): IUserDashboardVisibility {
  // Not logged in = don't show
  if (!isLoggedIn) {
    return {
      shouldShow: false,
      isExpanded: false,
      canToggle: false,
      collapsedMessage: '',
    };
  }

  // Still loading = show loading state
  if (equipmentStatus.isLoading) {
    return {
      shouldShow: true,
      isExpanded: false,
      canToggle: false,
      collapsedMessage: 'Loading your mining equipment...',
    };
  }

  // Has active equipment = show expanded
  if (equipmentStatus.hasActiveEquipment) {
    return {
      shouldShow: true,
      isExpanded: true,
      canToggle: true,
      collapsedMessage: '',
    };
  }

  // Has equipment but none active = show collapsed with message
  if (equipmentStatus.hasEquipment) {
    return {
      shouldShow: true,
      isExpanded: false,
      canToggle: true,
      collapsedMessage: `You have ${equipmentStatus.totalEquipmentCount} registered device(s), but none are currently mining. Connect your equipment to see stats.`,
    };
  }

  // No equipment at all = show collapsed with setup message
  return {
    shouldShow: true,
    isExpanded: false,
    canToggle: false,
    collapsedMessage: 'No mining equipment detected. Connect your miner to start seeing your personal dashboard.',
  };
}

/**
 * Equipment status values that count as "active"
 */
export const ACTIVE_EQUIPMENT_STATUSES = ['mining', 'online', 'idle'] as const;

/**
 * Check if a device is considered active
 */
export function isDeviceActive(device: IEquipmentDevice): boolean {
  return ACTIVE_EQUIPMENT_STATUSES.includes(device.status as typeof ACTIVE_EQUIPMENT_STATUSES[number]);
}

/**
 * Calculate equipment status from device list
 */
export function calculateEquipmentStatus(
  devices: IEquipmentDevice[],
  isLoading: boolean = false,
  error: string | null = null,
  hasPendingSupport: boolean = false
): IUserEquipmentStatus {
  const activeDevices = devices.filter(isDeviceActive);

  return {
    hasEquipment: devices.length > 0,
    hasActiveEquipment: activeDevices.length > 0,
    totalEquipmentCount: devices.length,
    activeEquipmentCount: activeDevices.length,
    hasPendingSupport,
    isLoading,
    error,
  };
}
