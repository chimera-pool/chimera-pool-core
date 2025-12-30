// ============================================================================
// UTILS INDEX
// Central export for all utility functions
// Following Interface Segregation Principle - import only what you need
// ============================================================================

// Formatters
export * from './formatters';

// Logger utility (use instead of console.log)
export { logger, debug, info, warn, error } from './logger';

// Accessibility utilities
export {
  createFocusTrap,
  getFocusableElements,
  announceToScreenReader,
  createSkipLink,
  generateAriaId,
  KeyboardKeys,
  isActivationKey,
  handleKeyboardActivation,
  createAriaInputProps,
  prefersReducedMotion,
  prefersHighContrast,
  getContrastRatio,
  meetsContrastAA,
  meetsContrastAAA,
} from './accessibility';
export type { FocusTrapOptions, AriaButtonProps, AriaInputProps } from './accessibility';

// Animation utilities
export {
  timings,
  easings,
  transition,
  transitions,
  keyframes,
  animation,
  injectKeyframes,
  staggerDelay,
  animationStyles,
  getAnimationStyles,
} from './animations';

// MFA utilities
export {
  initMFASetup,
  enableMFA,
  disableMFA,
  verifyMFA,
  getMFAStatus,
  regenerateBackupCodes,
  isValidTOTPCode,
  isValidBackupCode,
  formatBackupCode,
  copyBackupCodesToClipboard,
  downloadBackupCodes,
  getDeviceFingerprint,
  isDeviceRemembered,
  setDeviceRemembered,
} from './mfa';

// Audit log utilities
export {
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
} from './auditLog';

// Notification utilities
export {
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
} from './notifications';

// Performance utilities
export {
  performanceThresholds,
  getWebVitals,
  debounce,
  throttle,
  rafThrottle,
  batchUpdates,
  getVisibleItems,
  RequestPool,
  LRUCache,
  getPerformanceRating,
  formatBytes,
  formatDuration,
} from './performance';

// Types
export type { MFASetupResponse, MFAVerifyRequest, MFAStatus } from './mfa';
export type { AuditLogEntry, AuditAction, AuditLogFilter, AuditLogResponse, AuditLogStats } from './auditLog';
export type { NotificationChannel, AlertSeverity, NotificationPreferences, AlertType, AlertConfig } from './notifications';
export type { PerformanceMetrics, RequestTiming, PerformanceRating } from './performance';
export type { AnimationState } from './animations';
