import React, { useState, useEffect } from 'react';
import { colors, gradients } from '../../styles/shared';

// ============================================================================
// NOTIFICATION SETTINGS COMPONENT
// Allows users to configure their notification preferences
// ============================================================================

interface NotificationSettingsData {
  user_id: number;
  email: string;
  discord_webhook: string;
  phone_number: string;
  worker_offline_enabled: boolean;
  worker_offline_delay: number;
  hashrate_drop_enabled: boolean;
  hashrate_drop_percent: number;
  block_found_enabled: boolean;
  payout_enabled: boolean;
  email_enabled: boolean;
  discord_enabled: boolean;
  sms_enabled: boolean;
  max_alerts_per_hour: number;
  quiet_hours_start: number | null;
  quiet_hours_end: number | null;
}

interface NotificationSettingsProps {
  token: string;
}

const styles: { [key: string]: React.CSSProperties } = {
  container: {
    background: gradients.card,
    borderRadius: '12px',
    padding: '24px',
    border: `1px solid ${colors.border}`,
    marginBottom: '30px',
  },
  title: {
    fontSize: '1.3rem',
    color: colors.primary,
    margin: '0 0 8px',
    display: 'flex',
    alignItems: 'center',
    gap: '10px',
  },
  description: {
    color: colors.textSecondary,
    fontSize: '0.9rem',
    marginBottom: '24px',
  },
  section: {
    marginBottom: '24px',
    paddingBottom: '24px',
    borderBottom: `1px solid ${colors.border}`,
  },
  sectionTitle: {
    fontSize: '1.1rem',
    color: colors.textPrimary,
    marginBottom: '16px',
    fontWeight: 'bold',
  },
  grid: {
    display: 'grid',
    gridTemplateColumns: 'repeat(auto-fit, minmax(300px, 1fr))',
    gap: '16px',
  },
  toggleRow: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    padding: '12px 16px',
    backgroundColor: colors.bgInput,
    borderRadius: '8px',
    marginBottom: '12px',
  },
  toggleLabel: {
    display: 'flex',
    flexDirection: 'column' as const,
    gap: '4px',
  },
  toggleName: {
    color: colors.textPrimary,
    fontSize: '0.95rem',
    fontWeight: '500',
  },
  toggleDescription: {
    color: colors.textSecondary,
    fontSize: '0.8rem',
  },
  toggle: {
    position: 'relative' as const,
    width: '50px',
    height: '26px',
    backgroundColor: colors.border,
    borderRadius: '13px',
    cursor: 'pointer',
    transition: 'background-color 0.2s',
  },
  toggleActive: {
    backgroundColor: colors.primary,
  },
  toggleKnob: {
    position: 'absolute' as const,
    top: '3px',
    left: '3px',
    width: '20px',
    height: '20px',
    backgroundColor: colors.textPrimary,
    borderRadius: '50%',
    transition: 'transform 0.2s',
  },
  toggleKnobActive: {
    transform: 'translateX(24px)',
  },
  inputGroup: {
    marginBottom: '16px',
  },
  label: {
    display: 'block',
    color: colors.textPrimary,
    fontSize: '0.9rem',
    marginBottom: '8px',
  },
  input: {
    width: '100%',
    padding: '12px 16px',
    backgroundColor: colors.bgInput,
    border: `1px solid ${colors.border}`,
    borderRadius: '8px',
    color: colors.textPrimary,
    fontSize: '0.95rem',
    outline: 'none',
    transition: 'border-color 0.2s',
  },
  inputFocus: {
    borderColor: colors.primary,
  },
  select: {
    width: '100%',
    padding: '12px 16px',
    backgroundColor: colors.bgInput,
    border: `1px solid ${colors.border}`,
    borderRadius: '8px',
    color: colors.textPrimary,
    fontSize: '0.95rem',
    outline: 'none',
    cursor: 'pointer',
  },
  buttonRow: {
    display: 'flex',
    gap: '12px',
    justifyContent: 'flex-end',
    marginTop: '24px',
  },
  button: {
    padding: '12px 24px',
    borderRadius: '8px',
    fontSize: '0.95rem',
    fontWeight: 'bold',
    cursor: 'pointer',
    transition: 'all 0.2s',
    border: 'none',
  },
  buttonPrimary: {
    backgroundColor: colors.primary,
    color: colors.bgCard,
  },
  buttonSecondary: {
    backgroundColor: 'transparent',
    color: colors.primary,
    border: `1px solid ${colors.primary}`,
  },
  testButton: {
    padding: '8px 16px',
    backgroundColor: colors.bgCard,
    color: colors.primary,
    border: `1px solid ${colors.primary}`,
    borderRadius: '6px',
    fontSize: '0.85rem',
    cursor: 'pointer',
  },
  successMessage: {
    backgroundColor: `${colors.success}20`,
    color: colors.success,
    padding: '12px 16px',
    borderRadius: '8px',
    marginBottom: '16px',
  },
  errorMessage: {
    backgroundColor: `${colors.error}20`,
    color: colors.error,
    padding: '12px 16px',
    borderRadius: '8px',
    marginBottom: '16px',
  },
  channelCard: {
    backgroundColor: colors.bgInput,
    borderRadius: '10px',
    padding: '16px',
    border: `1px solid ${colors.border}`,
  },
  channelHeader: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: '12px',
  },
  channelIcon: {
    fontSize: '1.5rem',
    marginRight: '10px',
  },
};

const NotificationSettings: React.FC<NotificationSettingsProps> = ({ token }) => {
  const [settings, setSettings] = useState<NotificationSettingsData | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [message, setMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null);
  const [testingChannel, setTestingChannel] = useState<string | null>(null);

  useEffect(() => {
    fetchSettings();
  }, [token]);

  const fetchSettings = async () => {
    try {
      const response = await fetch('/api/v1/notifications/settings', {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (response.ok) {
        const data = await response.json();
        setSettings(data);
      } else {
        // Use defaults if no settings exist
        setSettings({
          user_id: 0,
          email: '',
          discord_webhook: '',
          phone_number: '',
          worker_offline_enabled: true,
          worker_offline_delay: 5,
          hashrate_drop_enabled: true,
          hashrate_drop_percent: 50,
          block_found_enabled: true,
          payout_enabled: true,
          email_enabled: true,
          discord_enabled: false,
          sms_enabled: false,
          max_alerts_per_hour: 10,
          quiet_hours_start: null,
          quiet_hours_end: null,
        });
      }
    } catch (error) {
      console.error('Failed to fetch notification settings:', error);
    } finally {
      setLoading(false);
    }
  };

  const saveSettings = async () => {
    if (!settings) return;
    setSaving(true);
    setMessage(null);

    try {
      const response = await fetch('/api/v1/notifications/settings', {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify(settings),
      });

      if (response.ok) {
        setMessage({ type: 'success', text: 'Notification settings saved successfully!' });
      } else {
        setMessage({ type: 'error', text: 'Failed to save settings. Please try again.' });
      }
    } catch (error) {
      setMessage({ type: 'error', text: 'Network error. Please try again.' });
    } finally {
      setSaving(false);
    }
  };

  const testNotification = async (channel: string) => {
    setTestingChannel(channel);
    try {
      const response = await fetch(`/api/v1/notifications/test?channel=${channel}`, {
        method: 'POST',
        headers: { Authorization: `Bearer ${token}` },
      });

      if (response.ok) {
        setMessage({ type: 'success', text: `Test ${channel} notification sent!` });
      } else {
        setMessage({ type: 'error', text: `Failed to send test ${channel} notification.` });
      }
    } catch (error) {
      setMessage({ type: 'error', text: 'Network error. Please try again.' });
    } finally {
      setTestingChannel(null);
    }
  };

  const updateSetting = <K extends keyof NotificationSettingsData>(
    key: K,
    value: NotificationSettingsData[K]
  ) => {
    if (settings) {
      setSettings({ ...settings, [key]: value });
    }
  };

  const Toggle: React.FC<{
    enabled: boolean;
    onChange: (enabled: boolean) => void;
  }> = ({ enabled, onChange }) => (
    <div
      style={{ ...styles.toggle, ...(enabled ? styles.toggleActive : {}) }}
      onClick={() => onChange(!enabled)}
    >
      <div style={{ ...styles.toggleKnob, ...(enabled ? styles.toggleKnobActive : {}) }} />
    </div>
  );

  if (loading) {
    return (
      <div style={styles.container}>
        <div style={{ textAlign: 'center', color: colors.textSecondary, padding: '40px' }}>
          Loading notification settings...
        </div>
      </div>
    );
  }

  if (!settings) return null;

  return (
    <div style={styles.container}>
      <h2 style={styles.title}>🔔 Notification Settings</h2>
      <p style={styles.description}>
        Configure how and when you receive notifications about your mining activity.
      </p>

      {message && (
        <div style={message.type === 'success' ? styles.successMessage : styles.errorMessage}>
          {message.text}
        </div>
      )}

      {/* Notification Channels */}
      <div style={styles.section}>
        <h3 style={styles.sectionTitle}>📬 Notification Channels</h3>
        <div style={styles.grid}>
          {/* Email Channel */}
          <div style={styles.channelCard}>
            <div style={styles.channelHeader}>
              <div style={{ display: 'flex', alignItems: 'center' }}>
                <span style={styles.channelIcon}>📧</span>
                <span style={styles.toggleName}>Email Notifications</span>
              </div>
              <Toggle
                enabled={settings.email_enabled}
                onChange={(v) => updateSetting('email_enabled', v)}
              />
            </div>
            <div style={styles.inputGroup}>
              <input
                type="email"
                style={styles.input}
                placeholder="your@email.com"
                value={settings.email}
                onChange={(e) => updateSetting('email', e.target.value)}
                disabled={!settings.email_enabled}
              />
            </div>
            {settings.email_enabled && settings.email && (
              <button
                style={styles.testButton}
                onClick={() => testNotification('email')}
                disabled={testingChannel === 'email'}
              >
                {testingChannel === 'email' ? 'Sending...' : 'Send Test'}
              </button>
            )}
          </div>

          {/* Discord Channel */}
          <div style={styles.channelCard}>
            <div style={styles.channelHeader}>
              <div style={{ display: 'flex', alignItems: 'center' }}>
                <span style={styles.channelIcon}>💬</span>
                <span style={styles.toggleName}>Discord Notifications</span>
              </div>
              <Toggle
                enabled={settings.discord_enabled}
                onChange={(v) => updateSetting('discord_enabled', v)}
              />
            </div>
            <div style={styles.inputGroup}>
              <input
                type="url"
                style={styles.input}
                placeholder="https://discord.com/api/webhooks/..."
                value={settings.discord_webhook}
                onChange={(e) => updateSetting('discord_webhook', e.target.value)}
                disabled={!settings.discord_enabled}
              />
            </div>
            {settings.discord_enabled && settings.discord_webhook && (
              <button
                style={styles.testButton}
                onClick={() => testNotification('discord')}
                disabled={testingChannel === 'discord'}
              >
                {testingChannel === 'discord' ? 'Sending...' : 'Send Test'}
              </button>
            )}
          </div>
        </div>
      </div>

      {/* Alert Types */}
      <div style={styles.section}>
        <h3 style={styles.sectionTitle}>⚡ Alert Types</h3>

        <div style={styles.toggleRow}>
          <div style={styles.toggleLabel}>
            <span style={styles.toggleName}>Worker Offline Alerts</span>
            <span style={styles.toggleDescription}>
              Get notified when a worker stops submitting shares
            </span>
          </div>
          <Toggle
            enabled={settings.worker_offline_enabled}
            onChange={(v) => updateSetting('worker_offline_enabled', v)}
          />
        </div>

        {settings.worker_offline_enabled && (
          <div style={{ ...styles.inputGroup, marginLeft: '20px', marginBottom: '20px' }}>
            <label style={styles.label}>Alert after offline for (minutes)</label>
            <select
              style={styles.select}
              value={settings.worker_offline_delay}
              onChange={(e) => updateSetting('worker_offline_delay', parseInt(e.target.value))}
            >
              <option value={3}>3 minutes</option>
              <option value={5}>5 minutes</option>
              <option value={10}>10 minutes</option>
              <option value={15}>15 minutes</option>
              <option value={30}>30 minutes</option>
            </select>
          </div>
        )}

        <div style={styles.toggleRow}>
          <div style={styles.toggleLabel}>
            <span style={styles.toggleName}>Hashrate Drop Alerts</span>
            <span style={styles.toggleDescription}>
              Get notified when your hashrate drops significantly
            </span>
          </div>
          <Toggle
            enabled={settings.hashrate_drop_enabled}
            onChange={(v) => updateSetting('hashrate_drop_enabled', v)}
          />
        </div>

        {settings.hashrate_drop_enabled && (
          <div style={{ ...styles.inputGroup, marginLeft: '20px', marginBottom: '20px' }}>
            <label style={styles.label}>Alert when hashrate drops by (%)</label>
            <select
              style={styles.select}
              value={settings.hashrate_drop_percent}
              onChange={(e) => updateSetting('hashrate_drop_percent', parseInt(e.target.value))}
            >
              <option value={30}>30%</option>
              <option value={50}>50%</option>
              <option value={70}>70%</option>
              <option value={90}>90%</option>
            </select>
          </div>
        )}

        <div style={styles.toggleRow}>
          <div style={styles.toggleLabel}>
            <span style={styles.toggleName}>Block Found Alerts</span>
            <span style={styles.toggleDescription}>
              Get notified when the pool finds a block
            </span>
          </div>
          <Toggle
            enabled={settings.block_found_enabled}
            onChange={(v) => updateSetting('block_found_enabled', v)}
          />
        </div>

        <div style={styles.toggleRow}>
          <div style={styles.toggleLabel}>
            <span style={styles.toggleName}>Payout Alerts</span>
            <span style={styles.toggleDescription}>
              Get notified when payouts are sent or fail
            </span>
          </div>
          <Toggle
            enabled={settings.payout_enabled}
            onChange={(v) => updateSetting('payout_enabled', v)}
          />
        </div>
      </div>

      {/* Rate Limiting */}
      <div style={{ ...styles.section, borderBottom: 'none' }}>
        <h3 style={styles.sectionTitle}>🛡️ Rate Limiting</h3>
        <div style={styles.inputGroup}>
          <label style={styles.label}>Maximum alerts per hour</label>
          <select
            style={styles.select}
            value={settings.max_alerts_per_hour}
            onChange={(e) => updateSetting('max_alerts_per_hour', parseInt(e.target.value))}
          >
            <option value={5}>5 alerts</option>
            <option value={10}>10 alerts</option>
            <option value={20}>20 alerts</option>
            <option value={50}>50 alerts</option>
            <option value={100}>Unlimited</option>
          </select>
        </div>
      </div>

      {/* Save Button */}
      <div style={styles.buttonRow}>
        <button
          style={{ ...styles.button, ...styles.buttonSecondary }}
          onClick={fetchSettings}
          disabled={saving}
        >
          Reset
        </button>
        <button
          style={{ ...styles.button, ...styles.buttonPrimary }}
          onClick={saveSettings}
          disabled={saving}
        >
          {saving ? 'Saving...' : 'Save Settings'}
        </button>
      </div>
    </div>
  );
};

export default NotificationSettings;
