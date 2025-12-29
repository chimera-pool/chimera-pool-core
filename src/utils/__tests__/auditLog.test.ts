import {
  getActionDisplayName,
  getActionCategory,
  getActionSeverity,
  formatRelativeTime,
  exportAuditLogsCSV,
  groupEntriesByDate,
  getAuditLogStats,
  AuditLogEntry,
} from '../auditLog';

const mockEntry: AuditLogEntry = {
  id: '1',
  userId: 'user1',
  username: 'testuser',
  action: 'login',
  resource: 'auth',
  ipAddress: '192.168.1.1',
  userAgent: 'Mozilla/5.0',
  timestamp: new Date().toISOString(),
  success: true,
};

describe('Audit Log Utilities', () => {
  describe('getActionDisplayName', () => {
    it('should return display name for login', () => {
      expect(getActionDisplayName('login')).toBe('Login');
    });

    it('should return display name for password_change', () => {
      expect(getActionDisplayName('password_change')).toBe('Password Changed');
    });

    it('should return display name for mfa_enable', () => {
      expect(getActionDisplayName('mfa_enable')).toBe('MFA Enabled');
    });

    it('should return action as fallback for unknown action', () => {
      expect(getActionDisplayName('unknown' as any)).toBe('unknown');
    });
  });

  describe('getActionCategory', () => {
    it('should categorize login as Authentication', () => {
      expect(getActionCategory('login')).toBe('Authentication');
    });

    it('should categorize password_change as Security', () => {
      expect(getActionCategory('password_change')).toBe('Security');
    });

    it('should categorize profile_update as Account', () => {
      expect(getActionCategory('profile_update')).toBe('Account');
    });

    it('should categorize user_create as User Management', () => {
      expect(getActionCategory('user_create')).toBe('User Management');
    });

    it('should return Other for unknown actions', () => {
      expect(getActionCategory('unknown' as any)).toBe('Other');
    });
  });

  describe('getActionSeverity', () => {
    it('should return warning for failed actions', () => {
      expect(getActionSeverity('login', false)).toBe('warning');
    });

    it('should return critical for user_delete', () => {
      expect(getActionSeverity('user_delete', true)).toBe('critical');
    });

    it('should return critical for role_change', () => {
      expect(getActionSeverity('role_change', true)).toBe('critical');
    });

    it('should return warning for login_failed', () => {
      expect(getActionSeverity('login_failed', true)).toBe('warning');
    });

    it('should return info for regular actions', () => {
      expect(getActionSeverity('login', true)).toBe('info');
    });
  });

  describe('formatRelativeTime', () => {
    it('should return "Just now" for recent timestamps', () => {
      const now = new Date().toISOString();
      expect(formatRelativeTime(now)).toBe('Just now');
    });

    it('should return minutes ago', () => {
      const fiveMinutesAgo = new Date(Date.now() - 5 * 60 * 1000).toISOString();
      expect(formatRelativeTime(fiveMinutesAgo)).toBe('5m ago');
    });

    it('should return hours ago', () => {
      const twoHoursAgo = new Date(Date.now() - 2 * 60 * 60 * 1000).toISOString();
      expect(formatRelativeTime(twoHoursAgo)).toBe('2h ago');
    });

    it('should return days ago', () => {
      const threeDaysAgo = new Date(Date.now() - 3 * 24 * 60 * 60 * 1000).toISOString();
      expect(formatRelativeTime(threeDaysAgo)).toBe('3d ago');
    });
  });

  describe('exportAuditLogsCSV', () => {
    it('should export entries to CSV format', () => {
      const entries: AuditLogEntry[] = [mockEntry];
      const csv = exportAuditLogsCSV(entries);
      
      expect(csv).toContain('Timestamp');
      expect(csv).toContain('User');
      expect(csv).toContain('Action');
      expect(csv).toContain('testuser');
      expect(csv).toContain('Login');
    });

    it('should handle empty entries', () => {
      const csv = exportAuditLogsCSV([]);
      expect(csv).toContain('Timestamp');
      expect(csv.split('\n').length).toBe(1); // Only header
    });
  });

  describe('groupEntriesByDate', () => {
    it('should group entries by date', () => {
      const today = new Date().toISOString();
      const yesterday = new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString();
      
      const entries: AuditLogEntry[] = [
        { ...mockEntry, id: '1', timestamp: today },
        { ...mockEntry, id: '2', timestamp: today },
        { ...mockEntry, id: '3', timestamp: yesterday },
      ];

      const groups = groupEntriesByDate(entries);
      
      expect(groups.size).toBe(2);
    });
  });

  describe('getAuditLogStats', () => {
    it('should calculate stats correctly', () => {
      const entries: AuditLogEntry[] = [
        { ...mockEntry, id: '1', action: 'login', success: true },
        { ...mockEntry, id: '2', action: 'login', success: true },
        { ...mockEntry, id: '3', action: 'logout', success: true },
        { ...mockEntry, id: '4', action: 'login_failed', success: false, userId: 'user2' },
      ];

      const stats = getAuditLogStats(entries);
      
      expect(stats.total).toBe(4);
      expect(stats.byAction['login']).toBe(2);
      expect(stats.byAction['logout']).toBe(1);
      expect(stats.byCategory['Authentication']).toBe(4);
      expect(stats.successRate).toBe(75);
      expect(stats.uniqueUsers).toBe(2);
    });

    it('should handle empty entries', () => {
      const stats = getAuditLogStats([]);
      
      expect(stats.total).toBe(0);
      expect(stats.successRate).toBe(100);
      expect(stats.uniqueUsers).toBe(0);
    });
  });
});
