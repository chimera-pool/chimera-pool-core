import {
  formatHashrate,
  formatNumber,
  formatCurrency,
  formatPercent,
  formatDuration,
  formatRelativeTime,
  truncateString,
  formatBytes,
} from '../formatters';

describe('formatters', () => {
  describe('formatHashrate', () => {
    it('should format H/s correctly', () => {
      expect(formatHashrate(500)).toBe('500.00 H/s');
      expect(formatHashrate(0)).toBe('0.00 H/s');
      expect(formatHashrate(999)).toBe('999.00 H/s');
    });

    it('should format KH/s correctly', () => {
      expect(formatHashrate(1000)).toBe('1.00 KH/s');
      expect(formatHashrate(1500)).toBe('1.50 KH/s');
      expect(formatHashrate(999999)).toBe('1000.00 KH/s');
    });

    it('should format MH/s correctly', () => {
      expect(formatHashrate(1e6)).toBe('1.00 MH/s');
      expect(formatHashrate(1.5e6)).toBe('1.50 MH/s');
      expect(formatHashrate(999e6)).toBe('999.00 MH/s');
    });

    it('should format GH/s correctly', () => {
      expect(formatHashrate(1e9)).toBe('1.00 GH/s');
      expect(formatHashrate(2.5e9)).toBe('2.50 GH/s');
    });

    it('should format TH/s correctly', () => {
      expect(formatHashrate(1e12)).toBe('1.00 TH/s');
      expect(formatHashrate(5.75e12)).toBe('5.75 TH/s');
    });

    it('should format PH/s correctly', () => {
      expect(formatHashrate(1e15)).toBe('1.00 PH/s');
      expect(formatHashrate(10e15)).toBe('10.00 PH/s');
    });
  });

  describe('formatNumber', () => {
    it('should format numbers with thousand separators', () => {
      expect(formatNumber(1000)).toMatch(/1[,.]000/);
      expect(formatNumber(1000000)).toMatch(/1[,.]000[,.]000/);
    });

    it('should handle small numbers', () => {
      expect(formatNumber(0)).toBe('0');
      expect(formatNumber(999)).toBe('999');
    });

    it('should handle negative numbers', () => {
      expect(formatNumber(-1000)).toMatch(/-1[,.]000/);
    });

    it('should handle decimal numbers', () => {
      const result = formatNumber(1234.56);
      expect(result).toContain('1');
      expect(result).toContain('234');
    });
  });

  describe('formatCurrency', () => {
    it('should format with default 8 decimals', () => {
      const result = formatCurrency(100000000);
      expect(result).toContain('1');
    });

    it('should format with custom decimals', () => {
      const result = formatCurrency(100, 2);
      expect(result).toContain('1');
    });

    it('should handle zero', () => {
      const result = formatCurrency(0);
      expect(result).toContain('0');
    });

    it('should handle large amounts', () => {
      const result = formatCurrency(1000000000000);
      expect(result).toContain('10');
    });
  });

  describe('formatPercent', () => {
    it('should format percentage values', () => {
      expect(formatPercent(50)).toBe('50.00%');
      expect(formatPercent(100)).toBe('100.00%');
      expect(formatPercent(0)).toBe('0.00%');
    });

    it('should format decimal values when isDecimal is true', () => {
      expect(formatPercent(0.5, true)).toBe('50.00%');
      expect(formatPercent(1, true)).toBe('100.00%');
      expect(formatPercent(0.25, true)).toBe('25.00%');
    });

    it('should handle fractional percentages', () => {
      expect(formatPercent(33.33)).toBe('33.33%');
      expect(formatPercent(0.3333, true)).toBe('33.33%');
    });
  });

  describe('formatDuration', () => {
    it('should format seconds', () => {
      expect(formatDuration(30)).toBe('30s');
      expect(formatDuration(59)).toBe('59s');
    });

    it('should format minutes', () => {
      expect(formatDuration(60)).toBe('1m');
      expect(formatDuration(120)).toBe('2m');
      expect(formatDuration(3599)).toBe('59m');
    });

    it('should format hours', () => {
      expect(formatDuration(3600)).toBe('1h');
      expect(formatDuration(7200)).toBe('2h');
    });

    it('should format hours and minutes', () => {
      expect(formatDuration(3660)).toBe('1h 1m');
      expect(formatDuration(5400)).toBe('1h 30m');
    });

    it('should format days', () => {
      expect(formatDuration(86400)).toBe('1d');
      expect(formatDuration(172800)).toBe('2d');
    });

    it('should format days and hours', () => {
      expect(formatDuration(90000)).toBe('1d 1h');
      expect(formatDuration(180000)).toBe('2d 2h');
    });

    it('should not show minutes when days are present', () => {
      expect(formatDuration(86460)).toBe('1d');
    });

    it('should handle zero', () => {
      expect(formatDuration(0)).toBe('0s');
    });
  });

  describe('formatRelativeTime', () => {
    beforeEach(() => {
      jest.useFakeTimers();
      jest.setSystemTime(new Date('2024-01-15T12:00:00Z'));
    });

    afterEach(() => {
      jest.useRealTimers();
    });

    it('should return "just now" for recent times', () => {
      const now = new Date();
      expect(formatRelativeTime(now)).toBe('just now');
      
      const thirtySecondsAgo = new Date(now.getTime() - 30000);
      expect(formatRelativeTime(thirtySecondsAgo)).toBe('just now');
    });

    it('should format minutes ago', () => {
      const now = new Date();
      const fiveMinutesAgo = new Date(now.getTime() - 5 * 60 * 1000);
      expect(formatRelativeTime(fiveMinutesAgo)).toBe('5 minutes ago');
    });

    it('should format hours ago', () => {
      const now = new Date();
      const twoHoursAgo = new Date(now.getTime() - 2 * 60 * 60 * 1000);
      expect(formatRelativeTime(twoHoursAgo)).toBe('2 hours ago');
    });

    it('should format days ago', () => {
      const now = new Date();
      const threeDaysAgo = new Date(now.getTime() - 3 * 24 * 60 * 60 * 1000);
      expect(formatRelativeTime(threeDaysAgo)).toBe('3 days ago');
    });

    it('should format as date for older times', () => {
      const now = new Date();
      const twoWeeksAgo = new Date(now.getTime() - 14 * 24 * 60 * 60 * 1000);
      const result = formatRelativeTime(twoWeeksAgo);
      expect(result).not.toContain('ago');
    });

    it('should accept string timestamps', () => {
      const result = formatRelativeTime('2024-01-15T11:55:00Z');
      expect(result).toBe('5 minutes ago');
    });

    it('should accept Date objects', () => {
      const date = new Date('2024-01-15T11:55:00Z');
      expect(formatRelativeTime(date)).toBe('5 minutes ago');
    });
  });

  describe('truncateString', () => {
    it('should truncate long strings', () => {
      const longAddress = '0x1234567890abcdef1234567890abcdef12345678';
      expect(truncateString(longAddress)).toBe('0x123456...345678');
    });

    it('should not truncate short strings', () => {
      const shortString = 'short';
      expect(truncateString(shortString)).toBe('short');
    });

    it('should use custom start and end chars', () => {
      const address = '0x1234567890abcdef1234567890abcdef12345678';
      expect(truncateString(address, 4, 4)).toBe('0x12...5678');
    });

    it('should handle exact boundary length', () => {
      const str = '12345678901234567';
      expect(truncateString(str, 8, 6)).toBe('12345678901234567');
    });

    it('should handle empty string', () => {
      expect(truncateString('')).toBe('');
    });
  });

  describe('formatBytes', () => {
    it('should format bytes', () => {
      expect(formatBytes(500)).toBe('500 B');
      expect(formatBytes(0)).toBe('0 B');
    });

    it('should format KB', () => {
      expect(formatBytes(1000)).toBe('1.00 KB');
      expect(formatBytes(1500)).toBe('1.50 KB');
    });

    it('should format MB', () => {
      expect(formatBytes(1e6)).toBe('1.00 MB');
      expect(formatBytes(1.5e6)).toBe('1.50 MB');
    });

    it('should format GB', () => {
      expect(formatBytes(1e9)).toBe('1.00 GB');
      expect(formatBytes(2.5e9)).toBe('2.50 GB');
    });

    it('should format TB', () => {
      expect(formatBytes(1e12)).toBe('1.00 TB');
      expect(formatBytes(5e12)).toBe('5.00 TB');
    });
  });
});
