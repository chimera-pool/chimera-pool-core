// ============================================================================
// MFA/TOTP UTILITIES
// Multi-factor authentication helpers for frontend integration
// Following Interface Segregation Principle
// ============================================================================

/** MFA setup response from backend */
export interface MFASetupResponse {
  secret: string;
  qrCodeUrl: string;
  backupCodes: string[];
}

/** MFA verification request */
export interface MFAVerifyRequest {
  code: string;
  rememberDevice?: boolean;
}

/** MFA status */
export interface MFAStatus {
  enabled: boolean;
  lastVerified?: string;
  backupCodesRemaining?: number;
}

/** API endpoints for MFA */
const MFA_ENDPOINTS = {
  setup: '/api/v1/user/mfa/setup',
  enable: '/api/v1/user/mfa/enable',
  disable: '/api/v1/user/mfa/disable',
  verify: '/api/v1/user/mfa/verify',
  status: '/api/v1/user/mfa/status',
  backupCodes: '/api/v1/user/mfa/backup-codes',
} as const;

/** Initialize MFA setup - get QR code and secret */
export async function initMFASetup(token: string): Promise<MFASetupResponse> {
  const response = await fetch(MFA_ENDPOINTS.setup, {
    method: 'POST',
    headers: {
      'Authorization': `Bearer ${token}`,
      'Content-Type': 'application/json',
    },
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({ message: 'Failed to setup MFA' }));
    throw new Error(error.message || 'Failed to setup MFA');
  }

  return response.json();
}

/** Enable MFA after verifying setup code */
export async function enableMFA(token: string, code: string): Promise<{ success: boolean; backupCodes: string[] }> {
  const response = await fetch(MFA_ENDPOINTS.enable, {
    method: 'POST',
    headers: {
      'Authorization': `Bearer ${token}`,
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ code }),
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({ message: 'Failed to enable MFA' }));
    throw new Error(error.message || 'Invalid verification code');
  }

  return response.json();
}

/** Disable MFA */
export async function disableMFA(token: string, code: string, password: string): Promise<{ success: boolean }> {
  const response = await fetch(MFA_ENDPOINTS.disable, {
    method: 'POST',
    headers: {
      'Authorization': `Bearer ${token}`,
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ code, password }),
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({ message: 'Failed to disable MFA' }));
    throw new Error(error.message || 'Failed to disable MFA');
  }

  return response.json();
}

/** Verify MFA code during login */
export async function verifyMFA(
  tempToken: string, 
  code: string, 
  rememberDevice: boolean = false
): Promise<{ token: string; user: any }> {
  const response = await fetch(MFA_ENDPOINTS.verify, {
    method: 'POST',
    headers: {
      'Authorization': `Bearer ${tempToken}`,
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ code, rememberDevice }),
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({ message: 'Invalid code' }));
    throw new Error(error.message || 'Invalid verification code');
  }

  return response.json();
}

/** Get MFA status */
export async function getMFAStatus(token: string): Promise<MFAStatus> {
  const response = await fetch(MFA_ENDPOINTS.status, {
    headers: {
      'Authorization': `Bearer ${token}`,
    },
  });

  if (!response.ok) {
    throw new Error('Failed to get MFA status');
  }

  return response.json();
}

/** Generate new backup codes */
export async function regenerateBackupCodes(token: string, code: string): Promise<{ backupCodes: string[] }> {
  const response = await fetch(MFA_ENDPOINTS.backupCodes, {
    method: 'POST',
    headers: {
      'Authorization': `Bearer ${token}`,
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ code }),
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({ message: 'Failed to regenerate codes' }));
    throw new Error(error.message || 'Failed to regenerate backup codes');
  }

  return response.json();
}

/** Validate TOTP code format (6 digits) */
export function isValidTOTPCode(code: string): boolean {
  return /^\d{6}$/.test(code);
}

/** Validate backup code format (8 alphanumeric chars) */
export function isValidBackupCode(code: string): boolean {
  return /^[A-Z0-9]{8}$/i.test(code.replace(/-/g, ''));
}

/** Format backup code for display (XXXX-XXXX) */
export function formatBackupCode(code: string): string {
  const clean = code.replace(/-/g, '').toUpperCase();
  return `${clean.slice(0, 4)}-${clean.slice(4, 8)}`;
}

/** Copy backup codes to clipboard */
export async function copyBackupCodesToClipboard(codes: string[]): Promise<boolean> {
  try {
    const formatted = codes.map(formatBackupCode).join('\n');
    await navigator.clipboard.writeText(formatted);
    return true;
  } catch {
    return false;
  }
}

/** Download backup codes as text file */
export function downloadBackupCodes(codes: string[], filename: string = 'chimera-pool-backup-codes.txt'): void {
  const content = [
    'Chimera Pool - MFA Backup Codes',
    '================================',
    '',
    'Keep these codes in a safe place.',
    'Each code can only be used once.',
    '',
    ...codes.map(formatBackupCode),
    '',
    `Generated: ${new Date().toISOString()}`,
  ].join('\n');

  const blob = new Blob([content], { type: 'text/plain' });
  const url = URL.createObjectURL(blob);
  const link = document.createElement('a');
  link.href = url;
  link.download = filename;
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
  URL.revokeObjectURL(url);
}

/** Device fingerprint for "remember this device" feature */
export async function getDeviceFingerprint(): Promise<string> {
  const components = [
    navigator.userAgent,
    navigator.language,
    screen.width,
    screen.height,
    screen.colorDepth,
    new Date().getTimezoneOffset(),
  ];
  
  const data = components.join('|');
  const encoder = new TextEncoder();
  const hashBuffer = await crypto.subtle.digest('SHA-256', encoder.encode(data));
  const hashArray = Array.from(new Uint8Array(hashBuffer));
  return hashArray.map(b => b.toString(16).padStart(2, '0')).join('');
}

/** Check if device is remembered (via localStorage) */
export function isDeviceRemembered(): boolean {
  return localStorage.getItem('mfa_device_remembered') === 'true';
}

/** Set device as remembered */
export function setDeviceRemembered(remember: boolean): void {
  if (remember) {
    localStorage.setItem('mfa_device_remembered', 'true');
  } else {
    localStorage.removeItem('mfa_device_remembered');
  }
}

export default {
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
};
