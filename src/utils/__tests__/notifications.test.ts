import {
  defaultAlertConfigs,
  isValidPhoneNumber,
  formatPhoneNumber,
  getAlertTypeDisplayName,
  getAlertTypeDescription,
  getSeverityColor,
  isInQuietHours,
  getPushPermission,
} from '../notifications';

describe('Notification Utilities', () => {
  describe('defaultAlertConfigs', () => {
    it('should have worker_offline config', () => {
      expect(defaultAlertConfigs.worker_offline).toBeDefined();
      expect(defaultAlertConfigs.worker_offline.severity).toBe('warning');
    });

    it('should have security_alert as critical', () => {
      expect(defaultAlertConfigs.security_alert.severity).toBe('critical');
    });

    it('should include SMS for hashrate_drop', () => {
      expect(defaultAlertConfigs.hashrate_drop.channels).toContain('sms');
    });
  });

  describe('isValidPhoneNumber', () => {
    it('should validate E.164 format', () => {
      expect(isValidPhoneNumber('+14155551234')).toBe(true);
      expect(isValidPhoneNumber('+442071234567')).toBe(true);
    });

    it('should handle spaces and dashes', () => {
      expect(isValidPhoneNumber('+1 415-555-1234')).toBe(true);
      expect(isValidPhoneNumber('+1 (415) 555-1234')).toBe(true);
    });

    it('should reject invalid formats', () => {
      expect(isValidPhoneNumber('4155551234')).toBe(false);
      expect(isValidPhoneNumber('+0123456789')).toBe(false);
      expect(isValidPhoneNumber('invalid')).toBe(false);
    });
  });

  describe('formatPhoneNumber', () => {
    it('should format 10-digit US number', () => {
      expect(formatPhoneNumber('4155551234')).toBe('(415) 555-1234');
    });

    it('should format 11-digit US number with country code', () => {
      expect(formatPhoneNumber('14155551234')).toBe('+1 (415) 555-1234');
    });

    it('should return original if cannot format', () => {
      expect(formatPhoneNumber('+442071234567')).toBe('+442071234567');
    });
  });

  describe('getAlertTypeDisplayName', () => {
    it('should return display name for worker_offline', () => {
      expect(getAlertTypeDisplayName('worker_offline')).toBe('Worker Offline');
    });

    it('should return display name for payout_sent', () => {
      expect(getAlertTypeDisplayName('payout_sent')).toBe('Payout Sent');
    });

    it('should return type as fallback', () => {
      expect(getAlertTypeDisplayName('unknown' as any)).toBe('unknown');
    });
  });

  describe('getAlertTypeDescription', () => {
    it('should return description for worker_offline', () => {
      const desc = getAlertTypeDescription('worker_offline');
      expect(desc).toContain('offline');
    });

    it('should return empty string for unknown types', () => {
      expect(getAlertTypeDescription('unknown' as any)).toBe('');
    });
  });

  describe('getSeverityColor', () => {
    it('should return blue for info', () => {
      expect(getSeverityColor('info')).toBe('#60A5FA');
    });

    it('should return yellow for warning', () => {
      expect(getSeverityColor('warning')).toBe('#FBBF24');
    });

    it('should return red for error', () => {
      expect(getSeverityColor('error')).toBe('#EF4444');
    });

    it('should return dark red for critical', () => {
      expect(getSeverityColor('critical')).toBe('#DC2626');
    });
  });

  describe('isInQuietHours', () => {
    it('should return false when quiet hours disabled', () => {
      expect(isInQuietHours({ enabled: false, start: '22:00', end: '08:00', timezone: 'UTC' })).toBe(false);
    });

    it('should return false when no quiet hours config', () => {
      expect(isInQuietHours(undefined)).toBe(false);
    });

    it('should check time range correctly', () => {
      const now = new Date();
      const currentHour = now.getHours();
      
      // Create quiet hours that include current time
      const quietHours = {
        enabled: true,
        start: `${String(currentHour).padStart(2, '0')}:00`,
        end: `${String((currentHour + 1) % 24).padStart(2, '0')}:00`,
        timezone: 'UTC',
      };
      
      expect(isInQuietHours(quietHours)).toBe(true);
    });

    it('should check time range that does not include current time', () => {
      const now = new Date();
      const futureHour = (now.getHours() + 5) % 24;
      
      const quietHours = {
        enabled: true,
        start: `${String(futureHour).padStart(2, '0')}:00`,
        end: `${String((futureHour + 1) % 24).padStart(2, '0')}:00`,
        timezone: 'UTC',
      };
      
      expect(isInQuietHours(quietHours)).toBe(false);
    });
  });

  describe('getPushPermission', () => {
    it('should return permission status', () => {
      const permission = getPushPermission();
      expect(['granted', 'denied', 'default', 'unsupported']).toContain(permission);
    });
  });
});
