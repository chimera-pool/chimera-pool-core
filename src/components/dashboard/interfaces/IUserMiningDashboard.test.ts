/**
 * TDD Tests for User Mining Dashboard Interfaces
 * Tests equipment status detection and dashboard visibility logic
 */

import {
  IEquipmentDevice,
  IUserEquipmentStatus,
  getDashboardVisibility,
  isDeviceActive,
  calculateEquipmentStatus,
  ACTIVE_EQUIPMENT_STATUSES,
} from './IUserMiningDashboard';

describe('IUserMiningDashboard Interfaces', () => {
  describe('isDeviceActive', () => {
    it('should return true for mining status', () => {
      const device: IEquipmentDevice = {
        id: '1',
        name: 'Rig1',
        type: 'gpu',
        status: 'mining',
        hashrate: 100,
        lastSeen: new Date(),
        isActive: true,
      };
      expect(isDeviceActive(device)).toBe(true);
    });

    it('should return true for online status', () => {
      const device: IEquipmentDevice = {
        id: '2',
        name: 'Rig2',
        type: 'asic',
        status: 'online',
        hashrate: 500,
        lastSeen: new Date(),
        isActive: true,
      };
      expect(isDeviceActive(device)).toBe(true);
    });

    it('should return true for idle status', () => {
      const device: IEquipmentDevice = {
        id: '3',
        name: 'Rig3',
        type: 'cpu',
        status: 'idle',
        hashrate: 0,
        lastSeen: new Date(),
        isActive: true,
      };
      expect(isDeviceActive(device)).toBe(true);
    });

    it('should return false for offline status', () => {
      const device: IEquipmentDevice = {
        id: '4',
        name: 'Rig4',
        type: 'gpu',
        status: 'offline',
        hashrate: 0,
        lastSeen: new Date(),
        isActive: false,
      };
      expect(isDeviceActive(device)).toBe(false);
    });

    it('should return false for error status', () => {
      const device: IEquipmentDevice = {
        id: '5',
        name: 'Rig5',
        type: 'asic',
        status: 'error',
        hashrate: 0,
        lastSeen: new Date(),
        isActive: false,
      };
      expect(isDeviceActive(device)).toBe(false);
    });
  });

  describe('calculateEquipmentStatus', () => {
    it('should return correct status for empty device list', () => {
      const result = calculateEquipmentStatus([]);
      
      expect(result.hasEquipment).toBe(false);
      expect(result.hasActiveEquipment).toBe(false);
      expect(result.totalEquipmentCount).toBe(0);
      expect(result.activeEquipmentCount).toBe(0);
    });

    it('should count active devices correctly', () => {
      const devices: IEquipmentDevice[] = [
        { id: '1', name: 'Rig1', type: 'gpu', status: 'mining', hashrate: 100, lastSeen: new Date(), isActive: true },
        { id: '2', name: 'Rig2', type: 'gpu', status: 'offline', hashrate: 0, lastSeen: new Date(), isActive: false },
        { id: '3', name: 'Rig3', type: 'asic', status: 'online', hashrate: 500, lastSeen: new Date(), isActive: true },
      ];
      
      const result = calculateEquipmentStatus(devices);
      
      expect(result.hasEquipment).toBe(true);
      expect(result.hasActiveEquipment).toBe(true);
      expect(result.totalEquipmentCount).toBe(3);
      expect(result.activeEquipmentCount).toBe(2);
    });

    it('should handle all offline devices', () => {
      const devices: IEquipmentDevice[] = [
        { id: '1', name: 'Rig1', type: 'gpu', status: 'offline', hashrate: 0, lastSeen: new Date(), isActive: false },
        { id: '2', name: 'Rig2', type: 'gpu', status: 'error', hashrate: 0, lastSeen: new Date(), isActive: false },
      ];
      
      const result = calculateEquipmentStatus(devices);
      
      expect(result.hasEquipment).toBe(true);
      expect(result.hasActiveEquipment).toBe(false);
      expect(result.totalEquipmentCount).toBe(2);
      expect(result.activeEquipmentCount).toBe(0);
    });

    it('should preserve loading and error states', () => {
      const result = calculateEquipmentStatus([], true, 'Network error', true);
      
      expect(result.isLoading).toBe(true);
      expect(result.error).toBe('Network error');
      expect(result.hasPendingSupport).toBe(true);
    });
  });

  describe('getDashboardVisibility', () => {
    describe('when user is not logged in', () => {
      it('should not show dashboard', () => {
        const equipmentStatus: IUserEquipmentStatus = {
          hasEquipment: true,
          hasActiveEquipment: true,
          totalEquipmentCount: 3,
          activeEquipmentCount: 2,
          hasPendingSupport: false,
          isLoading: false,
          error: null,
        };

        const result = getDashboardVisibility(equipmentStatus, false);

        expect(result.shouldShow).toBe(false);
        expect(result.isExpanded).toBe(false);
        expect(result.canToggle).toBe(false);
      });
    });

    describe('when loading', () => {
      it('should show collapsed with loading message', () => {
        const equipmentStatus: IUserEquipmentStatus = {
          hasEquipment: false,
          hasActiveEquipment: false,
          totalEquipmentCount: 0,
          activeEquipmentCount: 0,
          hasPendingSupport: false,
          isLoading: true,
          error: null,
        };

        const result = getDashboardVisibility(equipmentStatus, true);

        expect(result.shouldShow).toBe(true);
        expect(result.isExpanded).toBe(false);
        expect(result.canToggle).toBe(false);
        expect(result.collapsedMessage).toContain('Loading');
      });
    });

    describe('when user has active equipment', () => {
      it('should show expanded dashboard', () => {
        const equipmentStatus: IUserEquipmentStatus = {
          hasEquipment: true,
          hasActiveEquipment: true,
          totalEquipmentCount: 3,
          activeEquipmentCount: 2,
          hasPendingSupport: false,
          isLoading: false,
          error: null,
        };

        const result = getDashboardVisibility(equipmentStatus, true);

        expect(result.shouldShow).toBe(true);
        expect(result.isExpanded).toBe(true);
        expect(result.canToggle).toBe(true);
      });
    });

    describe('when user has equipment but none active', () => {
      it('should show collapsed dashboard with device count message', () => {
        const equipmentStatus: IUserEquipmentStatus = {
          hasEquipment: true,
          hasActiveEquipment: false,
          totalEquipmentCount: 2,
          activeEquipmentCount: 0,
          hasPendingSupport: false,
          isLoading: false,
          error: null,
        };

        const result = getDashboardVisibility(equipmentStatus, true);

        expect(result.shouldShow).toBe(true);
        expect(result.isExpanded).toBe(false);
        expect(result.canToggle).toBe(true);
        expect(result.collapsedMessage).toContain('2 registered device(s)');
        expect(result.collapsedMessage).toContain('none are currently mining');
      });
    });

    describe('when user has no equipment', () => {
      it('should show collapsed dashboard with setup message', () => {
        const equipmentStatus: IUserEquipmentStatus = {
          hasEquipment: false,
          hasActiveEquipment: false,
          totalEquipmentCount: 0,
          activeEquipmentCount: 0,
          hasPendingSupport: false,
          isLoading: false,
          error: null,
        };

        const result = getDashboardVisibility(equipmentStatus, true);

        expect(result.shouldShow).toBe(true);
        expect(result.isExpanded).toBe(false);
        expect(result.canToggle).toBe(false);
        expect(result.collapsedMessage).toContain('No mining equipment detected');
      });
    });
  });

  describe('ACTIVE_EQUIPMENT_STATUSES', () => {
    it('should include mining, online, and idle', () => {
      expect(ACTIVE_EQUIPMENT_STATUSES).toContain('mining');
      expect(ACTIVE_EQUIPMENT_STATUSES).toContain('online');
      expect(ACTIVE_EQUIPMENT_STATUSES).toContain('idle');
    });

    it('should not include offline or error', () => {
      expect(ACTIVE_EQUIPMENT_STATUSES).not.toContain('offline');
      expect(ACTIVE_EQUIPMENT_STATUSES).not.toContain('error');
    });
  });
});
