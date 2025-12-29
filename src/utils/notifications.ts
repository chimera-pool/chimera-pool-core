// ============================================================================
// NOTIFICATION UTILITIES
// SMS, Email, and Push notification helpers for alerts
// Following Interface Segregation Principle
// ============================================================================

/** Notification channel types */
export type NotificationChannel = 'email' | 'sms' | 'push' | 'webhook';

/** Alert severity levels */
export type AlertSeverity = 'info' | 'warning' | 'error' | 'critical';

/** Notification preferences */
export interface NotificationPreferences {
  email: boolean;
  sms: boolean;
  push: boolean;
  webhook: boolean;
  phoneNumber?: string;
  webhookUrl?: string;
  alertTypes: AlertType[];
  quietHours?: {
    enabled: boolean;
    start: string; // HH:MM format
    end: string;
    timezone: string;
  };
}

/** Alert types users can subscribe to */
export type AlertType =
  | 'worker_offline'
  | 'worker_online'
  | 'hashrate_drop'
  | 'hashrate_recovery'
  | 'payout_sent'
  | 'payout_failed'
  | 'block_found'
  | 'security_alert'
  | 'pool_maintenance'
  | 'new_feature';

/** Alert configuration */
export interface AlertConfig {
  type: AlertType;
  channels: NotificationChannel[];
  severity: AlertSeverity;
  throttleMinutes: number;
}

/** Default alert configurations */
export const defaultAlertConfigs: Record<AlertType, AlertConfig> = {
  worker_offline: {
    type: 'worker_offline',
    channels: ['email', 'push'],
    severity: 'warning',
    throttleMinutes: 15,
  },
  worker_online: {
    type: 'worker_online',
    channels: ['push'],
    severity: 'info',
    throttleMinutes: 5,
  },
  hashrate_drop: {
    type: 'hashrate_drop',
    channels: ['email', 'sms', 'push'],
    severity: 'warning',
    throttleMinutes: 30,
  },
  hashrate_recovery: {
    type: 'hashrate_recovery',
    channels: ['push'],
    severity: 'info',
    throttleMinutes: 5,
  },
  payout_sent: {
    type: 'payout_sent',
    channels: ['email', 'push'],
    severity: 'info',
    throttleMinutes: 0,
  },
  payout_failed: {
    type: 'payout_failed',
    channels: ['email', 'sms', 'push'],
    severity: 'error',
    throttleMinutes: 60,
  },
  block_found: {
    type: 'block_found',
    channels: ['email', 'push'],
    severity: 'info',
    throttleMinutes: 0,
  },
  security_alert: {
    type: 'security_alert',
    channels: ['email', 'sms', 'push'],
    severity: 'critical',
    throttleMinutes: 0,
  },
  pool_maintenance: {
    type: 'pool_maintenance',
    channels: ['email', 'push'],
    severity: 'warning',
    throttleMinutes: 60,
  },
  new_feature: {
    type: 'new_feature',
    channels: ['email'],
    severity: 'info',
    throttleMinutes: 1440, // Once per day
  },
};

/** API endpoints */
const NOTIFICATION_ENDPOINTS = {
  preferences: '/api/v1/user/notifications/preferences',
  test: '/api/v1/user/notifications/test',
  history: '/api/v1/user/notifications/history',
} as const;

/** Get notification preferences */
export async function getNotificationPreferences(token: string): Promise<NotificationPreferences> {
  const response = await fetch(NOTIFICATION_ENDPOINTS.preferences, {
    headers: { 'Authorization': `Bearer ${token}` },
  });

  if (!response.ok) {
    throw new Error('Failed to get notification preferences');
  }

  return response.json();
}

/** Update notification preferences */
export async function updateNotificationPreferences(
  token: string,
  preferences: Partial<NotificationPreferences>
): Promise<NotificationPreferences> {
  const response = await fetch(NOTIFICATION_ENDPOINTS.preferences, {
    method: 'PUT',
    headers: {
      'Authorization': `Bearer ${token}`,
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(preferences),
  });

  if (!response.ok) {
    throw new Error('Failed to update notification preferences');
  }

  return response.json();
}

/** Send test notification */
export async function sendTestNotification(
  token: string,
  channel: NotificationChannel
): Promise<{ success: boolean; message: string }> {
  const response = await fetch(NOTIFICATION_ENDPOINTS.test, {
    method: 'POST',
    headers: {
      'Authorization': `Bearer ${token}`,
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ channel }),
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({ message: 'Failed to send test notification' }));
    throw new Error(error.message);
  }

  return response.json();
}

/** Validate phone number format */
export function isValidPhoneNumber(phone: string): boolean {
  // E.164 format: +[country code][number]
  const e164Regex = /^\+[1-9]\d{1,14}$/;
  return e164Regex.test(phone.replace(/[\s-()]/g, ''));
}

/** Format phone number for display */
export function formatPhoneNumber(phone: string): string {
  const cleaned = phone.replace(/\D/g, '');
  
  if (cleaned.length === 10) {
    return `(${cleaned.slice(0, 3)}) ${cleaned.slice(3, 6)}-${cleaned.slice(6)}`;
  }
  if (cleaned.length === 11 && cleaned[0] === '1') {
    return `+1 (${cleaned.slice(1, 4)}) ${cleaned.slice(4, 7)}-${cleaned.slice(7)}`;
  }
  
  return phone;
}

/** Get alert type display name */
export function getAlertTypeDisplayName(type: AlertType): string {
  const displayNames: Record<AlertType, string> = {
    worker_offline: 'Worker Offline',
    worker_online: 'Worker Online',
    hashrate_drop: 'Hashrate Drop',
    hashrate_recovery: 'Hashrate Recovery',
    payout_sent: 'Payout Sent',
    payout_failed: 'Payout Failed',
    block_found: 'Block Found',
    security_alert: 'Security Alert',
    pool_maintenance: 'Pool Maintenance',
    new_feature: 'New Feature',
  };

  return displayNames[type] || type;
}

/** Get alert type description */
export function getAlertTypeDescription(type: AlertType): string {
  const descriptions: Record<AlertType, string> = {
    worker_offline: 'Get notified when a worker goes offline',
    worker_online: 'Get notified when a worker comes back online',
    hashrate_drop: 'Get notified when hashrate drops significantly',
    hashrate_recovery: 'Get notified when hashrate recovers',
    payout_sent: 'Get notified when a payout is sent',
    payout_failed: 'Get notified when a payout fails',
    block_found: 'Get notified when the pool finds a block',
    security_alert: 'Get notified of security-related events',
    pool_maintenance: 'Get notified of scheduled maintenance',
    new_feature: 'Get notified of new features and updates',
  };

  return descriptions[type] || '';
}

/** Get severity color */
export function getSeverityColor(severity: AlertSeverity): string {
  const colors: Record<AlertSeverity, string> = {
    info: '#60A5FA',
    warning: '#FBBF24',
    error: '#EF4444',
    critical: '#DC2626',
  };

  return colors[severity];
}

/** Check if currently in quiet hours */
export function isInQuietHours(quietHours?: NotificationPreferences['quietHours']): boolean {
  if (!quietHours?.enabled) return false;

  const now = new Date();
  const [startHour, startMin] = quietHours.start.split(':').map(Number);
  const [endHour, endMin] = quietHours.end.split(':').map(Number);
  
  const currentMinutes = now.getHours() * 60 + now.getMinutes();
  const startMinutes = startHour * 60 + startMin;
  const endMinutes = endHour * 60 + endMin;

  if (startMinutes <= endMinutes) {
    return currentMinutes >= startMinutes && currentMinutes < endMinutes;
  } else {
    // Spans midnight
    return currentMinutes >= startMinutes || currentMinutes < endMinutes;
  }
}

/** Request push notification permission */
export async function requestPushPermission(): Promise<NotificationPermission> {
  if (!('Notification' in window)) {
    throw new Error('Push notifications not supported');
  }

  return Notification.requestPermission();
}

/** Check push notification permission */
export function getPushPermission(): NotificationPermission | 'unsupported' {
  if (!('Notification' in window)) {
    return 'unsupported';
  }
  return Notification.permission;
}

export default {
  defaultAlertConfigs,
  getNotificationPreferences,
  updateNotificationPreferences,
  sendTestNotification,
  isValidPhoneNumber,
  formatPhoneNumber,
  getAlertTypeDisplayName,
  getAlertTypeDescription,
  getSeverityColor,
  isInQuietHours,
  requestPushPermission,
  getPushPermission,
};
