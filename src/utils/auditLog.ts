// ============================================================================
// AUDIT LOG UTILITIES
// Track and display user actions for security and compliance
// Following Interface Segregation Principle
// ============================================================================

/** Audit log entry from backend */
export interface AuditLogEntry {
  id: string;
  userId: string;
  username: string;
  action: AuditAction;
  resource: string;
  resourceId?: string;
  details?: Record<string, any>;
  ipAddress: string;
  userAgent: string;
  timestamp: string;
  success: boolean;
  errorMessage?: string;
}

/** Audit action types */
export type AuditAction =
  | 'login'
  | 'logout'
  | 'login_failed'
  | 'password_change'
  | 'password_reset'
  | 'mfa_enable'
  | 'mfa_disable'
  | 'mfa_verify'
  | 'profile_update'
  | 'wallet_update'
  | 'user_create'
  | 'user_update'
  | 'user_delete'
  | 'role_change'
  | 'settings_change'
  | 'payout_request'
  | 'worker_create'
  | 'worker_delete'
  | 'api_key_create'
  | 'api_key_revoke'
  | 'admin_action';

/** Audit log filter options */
export interface AuditLogFilter {
  userId?: string;
  action?: AuditAction | AuditAction[];
  resource?: string;
  startDate?: string;
  endDate?: string;
  success?: boolean;
  page?: number;
  limit?: number;
}

/** Paginated audit log response */
export interface AuditLogResponse {
  entries: AuditLogEntry[];
  total: number;
  page: number;
  limit: number;
  hasMore: boolean;
}

/** API endpoint */
const AUDIT_LOG_ENDPOINT = '/api/v1/admin/audit-logs';

/** Fetch audit logs with filters */
export async function fetchAuditLogs(
  token: string,
  filters: AuditLogFilter = {}
): Promise<AuditLogResponse> {
  const params = new URLSearchParams();
  
  if (filters.userId) params.set('userId', filters.userId);
  if (filters.action) {
    const actions = Array.isArray(filters.action) ? filters.action : [filters.action];
    actions.forEach(a => params.append('action', a));
  }
  if (filters.resource) params.set('resource', filters.resource);
  if (filters.startDate) params.set('startDate', filters.startDate);
  if (filters.endDate) params.set('endDate', filters.endDate);
  if (filters.success !== undefined) params.set('success', String(filters.success));
  if (filters.page) params.set('page', String(filters.page));
  if (filters.limit) params.set('limit', String(filters.limit));

  const url = `${AUDIT_LOG_ENDPOINT}?${params.toString()}`;
  
  const response = await fetch(url, {
    headers: {
      'Authorization': `Bearer ${token}`,
    },
  });

  if (!response.ok) {
    throw new Error('Failed to fetch audit logs');
  }

  return response.json();
}

/** Get action display name */
export function getActionDisplayName(action: AuditAction): string {
  const displayNames: Record<AuditAction, string> = {
    login: 'Login',
    logout: 'Logout',
    login_failed: 'Login Failed',
    password_change: 'Password Changed',
    password_reset: 'Password Reset',
    mfa_enable: 'MFA Enabled',
    mfa_disable: 'MFA Disabled',
    mfa_verify: 'MFA Verified',
    profile_update: 'Profile Updated',
    wallet_update: 'Wallet Updated',
    user_create: 'User Created',
    user_update: 'User Updated',
    user_delete: 'User Deleted',
    role_change: 'Role Changed',
    settings_change: 'Settings Changed',
    payout_request: 'Payout Requested',
    worker_create: 'Worker Created',
    worker_delete: 'Worker Deleted',
    api_key_create: 'API Key Created',
    api_key_revoke: 'API Key Revoked',
    admin_action: 'Admin Action',
  };

  return displayNames[action] || action;
}

/** Get action category for grouping */
export function getActionCategory(action: AuditAction): string {
  const categories: Record<AuditAction, string> = {
    login: 'Authentication',
    logout: 'Authentication',
    login_failed: 'Authentication',
    password_change: 'Security',
    password_reset: 'Security',
    mfa_enable: 'Security',
    mfa_disable: 'Security',
    mfa_verify: 'Security',
    profile_update: 'Account',
    wallet_update: 'Account',
    user_create: 'User Management',
    user_update: 'User Management',
    user_delete: 'User Management',
    role_change: 'User Management',
    settings_change: 'Settings',
    payout_request: 'Payouts',
    worker_create: 'Mining',
    worker_delete: 'Mining',
    api_key_create: 'API',
    api_key_revoke: 'API',
    admin_action: 'Administration',
  };

  return categories[action] || 'Other';
}

/** Get action severity level */
export type ActionSeverity = 'info' | 'warning' | 'critical';

export function getActionSeverity(action: AuditAction, success: boolean): ActionSeverity {
  if (!success) return 'warning';
  
  const criticalActions: AuditAction[] = [
    'user_delete',
    'role_change',
    'mfa_disable',
    'api_key_revoke',
    'admin_action',
  ];

  const warningActions: AuditAction[] = [
    'login_failed',
    'password_reset',
    'settings_change',
  ];

  if (criticalActions.includes(action)) return 'critical';
  if (warningActions.includes(action)) return 'warning';
  return 'info';
}

/** Format audit log timestamp */
export function formatAuditTimestamp(timestamp: string): string {
  const date = new Date(timestamp);
  return date.toLocaleString();
}

/** Format relative time (e.g., "2 hours ago") */
export function formatRelativeTime(timestamp: string): string {
  const date = new Date(timestamp);
  const now = new Date();
  const diff = now.getTime() - date.getTime();
  
  const seconds = Math.floor(diff / 1000);
  const minutes = Math.floor(seconds / 60);
  const hours = Math.floor(minutes / 60);
  const days = Math.floor(hours / 24);

  if (days > 0) return `${days}d ago`;
  if (hours > 0) return `${hours}h ago`;
  if (minutes > 0) return `${minutes}m ago`;
  return 'Just now';
}

/** Export audit logs as CSV */
export function exportAuditLogsCSV(entries: AuditLogEntry[]): string {
  const headers = ['Timestamp', 'User', 'Action', 'Resource', 'IP Address', 'Success', 'Details'];
  
  const rows = entries.map(entry => [
    entry.timestamp,
    entry.username,
    getActionDisplayName(entry.action),
    entry.resource + (entry.resourceId ? `:${entry.resourceId}` : ''),
    entry.ipAddress,
    entry.success ? 'Yes' : 'No',
    entry.details ? JSON.stringify(entry.details) : '',
  ]);

  const csvContent = [
    headers.join(','),
    ...rows.map(row => row.map(cell => `"${String(cell).replace(/"/g, '""')}"`).join(',')),
  ].join('\n');

  return csvContent;
}

/** Download audit logs as CSV file */
export function downloadAuditLogsCSV(entries: AuditLogEntry[], filename: string = 'audit-logs.csv'): void {
  const csv = exportAuditLogsCSV(entries);
  const blob = new Blob([csv], { type: 'text/csv;charset=utf-8;' });
  const url = URL.createObjectURL(blob);
  const link = document.createElement('a');
  link.href = url;
  link.download = filename;
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
  URL.revokeObjectURL(url);
}

/** Group entries by date */
export function groupEntriesByDate(entries: AuditLogEntry[]): Map<string, AuditLogEntry[]> {
  const groups = new Map<string, AuditLogEntry[]>();
  
  entries.forEach(entry => {
    const date = new Date(entry.timestamp).toLocaleDateString();
    const existing = groups.get(date) || [];
    groups.set(date, [...existing, entry]);
  });

  return groups;
}

/** Get summary statistics for audit logs */
export interface AuditLogStats {
  total: number;
  byAction: Record<string, number>;
  byCategory: Record<string, number>;
  successRate: number;
  uniqueUsers: number;
}

export function getAuditLogStats(entries: AuditLogEntry[]): AuditLogStats {
  const byAction: Record<string, number> = {};
  const byCategory: Record<string, number> = {};
  const userIds = new Set<string>();
  let successCount = 0;

  entries.forEach(entry => {
    // Count by action
    byAction[entry.action] = (byAction[entry.action] || 0) + 1;
    
    // Count by category
    const category = getActionCategory(entry.action);
    byCategory[category] = (byCategory[category] || 0) + 1;
    
    // Track unique users
    userIds.add(entry.userId);
    
    // Count successes
    if (entry.success) successCount++;
  });

  return {
    total: entries.length,
    byAction,
    byCategory,
    successRate: entries.length > 0 ? (successCount / entries.length) * 100 : 100,
    uniqueUsers: userIds.size,
  };
}

export default {
  fetchAuditLogs,
  getActionDisplayName,
  getActionCategory,
  getActionSeverity,
  formatAuditTimestamp,
  formatRelativeTime,
  exportAuditLogsCSV,
  downloadAuditLogsCSV,
  groupEntriesByDate,
  getAuditLogStats,
};
