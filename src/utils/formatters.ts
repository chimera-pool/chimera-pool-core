// ============================================================================
// CHIMERA POOL - UTILITY FORMATTERS
// World-class formatting functions with proper typing and documentation
// ============================================================================

/**
 * Formats a hashrate value into human-readable string with appropriate unit
 * @param hashrate - Raw hashrate in H/s
 * @returns Formatted string with unit (H/s, KH/s, MH/s, GH/s, TH/s, PH/s)
 */
export function formatHashrate(hashrate: number): string {
  if (hashrate >= 1e15) return (hashrate / 1e15).toFixed(2) + ' PH/s';
  if (hashrate >= 1e12) return (hashrate / 1e12).toFixed(2) + ' TH/s';
  if (hashrate >= 1e9) return (hashrate / 1e9).toFixed(2) + ' GH/s';
  if (hashrate >= 1e6) return (hashrate / 1e6).toFixed(2) + ' MH/s';
  if (hashrate >= 1e3) return (hashrate / 1e3).toFixed(2) + ' KH/s';
  return hashrate.toFixed(2) + ' H/s';
}

/**
 * Formats a number with thousand separators
 * @param num - Number to format
 * @returns Formatted string with commas
 */
export function formatNumber(num: number): string {
  return num.toLocaleString();
}

/**
 * Formats a currency amount (assumes 8 decimal places like satoshis)
 * @param amount - Amount in smallest unit (e.g., satoshis)
 * @param decimals - Number of decimal places (default 8)
 * @returns Formatted string
 */
export function formatCurrency(amount: number, decimals: number = 8): string {
  const value = amount / Math.pow(10, decimals);
  return value.toLocaleString(undefined, {
    minimumFractionDigits: 2,
    maximumFractionDigits: decimals,
  });
}

/**
 * Formats a percentage value
 * @param value - Decimal value (0-1) or percentage (0-100)
 * @param isDecimal - If true, value is treated as decimal (default false)
 * @returns Formatted percentage string
 */
export function formatPercent(value: number, isDecimal: boolean = false): string {
  const pct = isDecimal ? value * 100 : value;
  return pct.toFixed(2) + '%';
}

/**
 * Formats a duration in seconds to human-readable string
 * @param seconds - Duration in seconds
 * @returns Formatted string (e.g., "2d 5h 30m")
 */
export function formatDuration(seconds: number): string {
  if (seconds < 60) return `${Math.floor(seconds)}s`;
  
  const days = Math.floor(seconds / 86400);
  const hours = Math.floor((seconds % 86400) / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  
  const parts: string[] = [];
  if (days > 0) parts.push(`${days}d`);
  if (hours > 0) parts.push(`${hours}h`);
  if (minutes > 0 && days === 0) parts.push(`${minutes}m`);
  
  return parts.join(' ') || '0m';
}

/**
 * Formats a timestamp to relative time string
 * @param timestamp - ISO timestamp or Date object
 * @returns Relative time string (e.g., "5 minutes ago")
 */
export function formatRelativeTime(timestamp: string | Date): string {
  const date = typeof timestamp === 'string' ? new Date(timestamp) : timestamp;
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffSec = Math.floor(diffMs / 1000);
  
  if (diffSec < 60) return 'just now';
  if (diffSec < 3600) return `${Math.floor(diffSec / 60)} minutes ago`;
  if (diffSec < 86400) return `${Math.floor(diffSec / 3600)} hours ago`;
  if (diffSec < 604800) return `${Math.floor(diffSec / 86400)} days ago`;
  
  return date.toLocaleDateString();
}

/**
 * Truncates a string (e.g., wallet address) with ellipsis
 * @param str - String to truncate
 * @param startChars - Characters to show at start
 * @param endChars - Characters to show at end
 * @returns Truncated string
 */
export function truncateString(str: string, startChars: number = 8, endChars: number = 6): string {
  if (str.length <= startChars + endChars + 3) return str;
  return `${str.slice(0, startChars)}...${str.slice(-endChars)}`;
}

/**
 * Formats bytes to human-readable size
 * @param bytes - Number of bytes
 * @returns Formatted string (e.g., "1.5 GB")
 */
export function formatBytes(bytes: number): string {
  if (bytes >= 1e12) return (bytes / 1e12).toFixed(2) + ' TB';
  if (bytes >= 1e9) return (bytes / 1e9).toFixed(2) + ' GB';
  if (bytes >= 1e6) return (bytes / 1e6).toFixed(2) + ' MB';
  if (bytes >= 1e3) return (bytes / 1e3).toFixed(2) + ' KB';
  return bytes + ' B';
}
