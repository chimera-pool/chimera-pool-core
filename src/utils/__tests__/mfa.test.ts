import {
  isValidTOTPCode,
  isValidBackupCode,
  formatBackupCode,
  isDeviceRemembered,
  setDeviceRemembered,
} from '../mfa';

describe('MFA Utilities', () => {
  describe('isValidTOTPCode', () => {
    it('should return true for valid 6-digit code', () => {
      expect(isValidTOTPCode('123456')).toBe(true);
      expect(isValidTOTPCode('000000')).toBe(true);
      expect(isValidTOTPCode('999999')).toBe(true);
    });

    it('should return false for invalid codes', () => {
      expect(isValidTOTPCode('12345')).toBe(false);
      expect(isValidTOTPCode('1234567')).toBe(false);
      expect(isValidTOTPCode('abcdef')).toBe(false);
      expect(isValidTOTPCode('12345a')).toBe(false);
      expect(isValidTOTPCode('')).toBe(false);
    });
  });

  describe('isValidBackupCode', () => {
    it('should return true for valid 8-char alphanumeric code', () => {
      expect(isValidBackupCode('ABCD1234')).toBe(true);
      expect(isValidBackupCode('abcd1234')).toBe(true);
      expect(isValidBackupCode('12345678')).toBe(true);
    });

    it('should return true for code with dash', () => {
      expect(isValidBackupCode('ABCD-1234')).toBe(true);
    });

    it('should return false for invalid codes', () => {
      expect(isValidBackupCode('ABCD123')).toBe(false);
      expect(isValidBackupCode('ABCD12345')).toBe(false);
      expect(isValidBackupCode('ABCD-123')).toBe(false);
      expect(isValidBackupCode('')).toBe(false);
    });
  });

  describe('formatBackupCode', () => {
    it('should format code with dash', () => {
      expect(formatBackupCode('ABCD1234')).toBe('ABCD-1234');
    });

    it('should uppercase and format code', () => {
      expect(formatBackupCode('abcd1234')).toBe('ABCD-1234');
    });

    it('should handle code with existing dash', () => {
      expect(formatBackupCode('ABCD-1234')).toBe('ABCD-1234');
    });
  });

  describe('Device Remember Functions', () => {
    beforeEach(() => {
      localStorage.clear();
    });

    it('should return false when device is not remembered', () => {
      expect(isDeviceRemembered()).toBe(false);
    });

    it('should set device as remembered', () => {
      setDeviceRemembered(true);
      expect(isDeviceRemembered()).toBe(true);
    });

    it('should clear remembered status', () => {
      setDeviceRemembered(true);
      setDeviceRemembered(false);
      expect(isDeviceRemembered()).toBe(false);
    });
  });
});
