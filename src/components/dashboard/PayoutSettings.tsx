import React, { useState, useEffect } from 'react';
import { colors, gradients } from '../../styles/shared';

// ============================================================================
// PAYOUT SETTINGS COMPONENT
// Allows users to select payout mode and configure payout preferences
// ============================================================================

interface PayoutMode {
  mode: string;
  name: string;
  description: string;
  fee_percent: number;
  is_enabled: boolean;
  risk_level: string;
  best_for: string;
}

interface PayoutSettings {
  user_id: number;
  payout_mode: string;
  min_payout_amount: number;
  payout_address: string;
  auto_payout_enable: boolean;
  fee_percent: number;
}

interface PayoutSettingsProps {
  token: string;
}

const styles: { [key: string]: React.CSSProperties } = {
  section: {
    background: 'linear-gradient(180deg, rgba(45, 31, 61, 0.6) 0%, rgba(26, 15, 30, 0.8) 100%)',
    borderRadius: '16px',
    padding: '28px',
    border: '1px solid rgba(74, 44, 90, 0.5)',
    marginBottom: '30px',
    boxShadow: '0 4px 24px rgba(0, 0, 0, 0.2)',
  },
  sectionTitle: {
    fontSize: '1.4rem',
    color: colors.primary,
    margin: '0 0 16px',
    display: 'flex',
    alignItems: 'center',
    gap: '10px',
    fontWeight: 700,
    textShadow: '0 2px 4px rgba(0, 0, 0, 0.3)',
  },
  description: {
    color: colors.textSecondary,
    fontSize: '0.9rem',
    marginBottom: '20px',
  },
  modesGrid: {
    display: 'grid',
    gridTemplateColumns: 'repeat(auto-fit, minmax(280px, 1fr))',
    gap: '15px',
    marginBottom: '25px',
  },
  modeCard: {
    background: 'linear-gradient(180deg, rgba(31, 20, 40, 0.8) 0%, rgba(26, 15, 30, 0.9) 100%)',
    padding: '22px',
    borderRadius: '14px',
    border: '2px solid rgba(74, 44, 90, 0.4)',
    cursor: 'pointer',
    transition: 'all 0.25s ease',
  },
  modeCardSelected: {
    borderColor: '#D4A84B',
    boxShadow: '0 0 20px rgba(212, 168, 75, 0.3)',
    background: 'linear-gradient(180deg, rgba(45, 31, 61, 0.9) 0%, rgba(26, 15, 30, 0.95) 100%)',
  },
  modeCardHover: {
    borderColor: 'rgba(212, 168, 75, 0.5)',
    transform: 'translateY(-2px)',
    boxShadow: '0 4px 16px rgba(0, 0, 0, 0.3)',
  },
  modeCardDisabled: {
    opacity: 0.5,
    cursor: 'not-allowed',
  },
  modeHeader: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: '10px',
  },
  modeName: {
    color: colors.primary,
    fontSize: '1.1rem',
    fontWeight: 'bold',
    textTransform: 'uppercase' as const,
  },
  modeFee: {
    backgroundColor: colors.bgCard,
    color: colors.success,
    padding: '4px 10px',
    borderRadius: '20px',
    fontSize: '0.85rem',
    fontWeight: 'bold',
  },
  modeDescription: {
    color: colors.textSecondary,
    fontSize: '0.85rem',
    marginBottom: '12px',
    lineHeight: '1.4',
  },
  modeBestFor: {
    color: colors.textPrimary,
    fontSize: '0.8rem',
    fontStyle: 'italic',
  },
  riskBadge: {
    display: 'inline-block',
    padding: '3px 8px',
    borderRadius: '4px',
    fontSize: '0.75rem',
    fontWeight: 'bold',
    marginLeft: '8px',
  },
  riskLow: {
    backgroundColor: `${colors.success}30`,
    color: colors.success,
  },
  riskMedium: {
    backgroundColor: `${colors.warning}30`,
    color: colors.warning,
  },
  riskHigh: {
    backgroundColor: `${colors.error}30`,
    color: colors.error,
  },
  settingsForm: {
    background: 'linear-gradient(180deg, rgba(13, 8, 17, 0.7) 0%, rgba(26, 15, 30, 0.85) 100%)',
    padding: '24px',
    borderRadius: '14px',
    marginTop: '24px',
    border: '1px solid rgba(74, 44, 90, 0.4)',
  },
  formTitle: {
    color: colors.primary,
    fontSize: '1rem',
    marginBottom: '15px',
  },
  formGroup: {
    marginBottom: '15px',
  },
  label: {
    display: 'block',
    color: colors.textSecondary,
    fontSize: '0.85rem',
    marginBottom: '6px',
  },
  input: {
    width: '100%',
    padding: '10px 12px',
    backgroundColor: colors.bgCard,
    border: `1px solid ${colors.border}`,
    borderRadius: '6px',
    color: colors.textPrimary,
    fontSize: '0.95rem',
  },
  checkbox: {
    display: 'flex',
    alignItems: 'center',
    gap: '10px',
    cursor: 'pointer',
  },
  checkboxInput: {
    width: '18px',
    height: '18px',
    accentColor: colors.primary,
  },
  button: {
    background: 'linear-gradient(135deg, #D4A84B 0%, #B8923A 100%)',
    color: '#1A0F1E',
    padding: '14px 28px',
    border: 'none',
    borderRadius: '10px',
    fontSize: '1rem',
    fontWeight: 700,
    cursor: 'pointer',
    transition: 'all 0.25s ease',
    marginTop: '12px',
    boxShadow: '0 2px 12px rgba(212, 168, 75, 0.3)',
  },
  buttonDisabled: {
    opacity: 0.6,
    cursor: 'not-allowed',
  },
  successMessage: {
    backgroundColor: `${colors.success}20`,
    color: colors.success,
    padding: '10px 15px',
    borderRadius: '6px',
    marginTop: '15px',
    fontSize: '0.9rem',
  },
  errorMessage: {
    backgroundColor: `${colors.error}20`,
    color: colors.error,
    padding: '10px 15px',
    borderRadius: '6px',
    marginTop: '15px',
    fontSize: '0.9rem',
  },
  currentMode: {
    display: 'flex',
    alignItems: 'center',
    gap: '10px',
    padding: '12px 16px',
    backgroundColor: `${colors.primary}20`,
    borderRadius: '8px',
    marginBottom: '20px',
  },
  currentModeLabel: {
    color: colors.textSecondary,
    fontSize: '0.9rem',
  },
  currentModeValue: {
    color: colors.primary,
    fontSize: '1rem',
    fontWeight: 'bold',
    textTransform: 'uppercase' as const,
  },
};

export function PayoutSettings({ token }: PayoutSettingsProps) {
  const [modes, setModes] = useState<PayoutMode[]>([]);
  const [settings, setSettings] = useState<PayoutSettings | null>(null);
  const [selectedMode, setSelectedMode] = useState<string>('pplns');
  const [minPayout, setMinPayout] = useState<string>('0.01');
  const [autoPayout, setAutoPayout] = useState<boolean>(true);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [message, setMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null);

  useEffect(() => {
    fetchData();
  }, [token]);

  const fetchData = async () => {
    try {
      const [modesRes, settingsRes] = await Promise.all([
        fetch('/api/v1/payout-modes'),
        fetch('/api/v1/user/payout-settings', {
          headers: { 'Authorization': `Bearer ${token}` }
        })
      ]);

      if (modesRes.ok) {
        const modesData = await modesRes.json();
        setModes(modesData.modes || []);
      }

      if (settingsRes.ok) {
        const settingsData = await settingsRes.json();
        setSettings(settingsData);
        setSelectedMode(settingsData.payout_mode || 'pplns');
        setMinPayout((settingsData.min_payout_amount / 100000000).toString());
        setAutoPayout(settingsData.auto_payout_enable ?? true);
      }
    } catch (error) {
      console.error('Failed to fetch payout data:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleSave = async () => {
    setSaving(true);
    setMessage(null);

    try {
      const response = await fetch('/api/v1/user/payout-settings', {
        method: 'PUT',
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          payout_mode: selectedMode,
          min_payout_amount: Math.floor(parseFloat(minPayout) * 100000000),
          auto_payout_enable: autoPayout,
        }),
      });

      if (response.ok) {
        setMessage({ type: 'success', text: 'Payout settings updated successfully!' });
        fetchData(); // Refresh data
      } else {
        const error = await response.json();
        setMessage({ type: 'error', text: error.error || 'Failed to update settings' });
      }
    } catch (error) {
      setMessage({ type: 'error', text: 'Network error. Please try again.' });
    } finally {
      setSaving(false);
    }
  };

  const getRiskStyle = (risk: string): React.CSSProperties => {
    switch (risk) {
      case 'low': return { ...styles.riskBadge, ...styles.riskLow };
      case 'medium': return { ...styles.riskBadge, ...styles.riskMedium };
      case 'high': return { ...styles.riskBadge, ...styles.riskHigh };
      default: return styles.riskBadge;
    }
  };

  const getModeDisplayName = (mode: string): string => {
    const names: { [key: string]: string } = {
      'pplns': 'PPLNS',
      'pps': 'PPS',
      'pps_plus': 'PPS+',
      'fpps': 'FPPS',
      'score': 'SCORE',
      'solo': 'SOLO',
      'slice': 'SLICE',
    };
    return names[mode] || mode.toUpperCase();
  };

  if (loading) {
    return (
      <section style={styles.section}>
        <h2 style={styles.sectionTitle}>💰 Payout Settings</h2>
        <div style={{ textAlign: 'center', padding: '20px', color: colors.textSecondary }}>
          Loading payout options...
        </div>
      </section>
    );
  }

  return (
    <section style={styles.section}>
      <h2 style={styles.sectionTitle}>💰 Payout Settings</h2>
      <p style={styles.description}>
        Choose your preferred payout method. Different modes offer different risk/reward balances.
      </p>

      {/* Current Mode Display */}
      {settings && (
        <div style={styles.currentMode}>
          <span style={styles.currentModeLabel}>Current Mode:</span>
          <span style={styles.currentModeValue}>{getModeDisplayName(settings.payout_mode)}</span>
          <span style={{ color: colors.success, fontSize: '0.9rem' }}>
            ({settings.fee_percent}% fee)
          </span>
        </div>
      )}

      {/* Payout Mode Selection */}
      <div style={styles.modesGrid}>
        {modes.filter(m => m.is_enabled).map((mode) => (
          <div
            key={mode.mode}
            style={{
              ...styles.modeCard,
              ...(selectedMode === mode.mode ? styles.modeCardSelected : {}),
            }}
            onClick={() => setSelectedMode(mode.mode)}
          >
            <div style={styles.modeHeader}>
              <span style={styles.modeName}>{getModeDisplayName(mode.mode)}</span>
              <span style={styles.modeFee}>{mode.fee_percent}% fee</span>
            </div>
            <p style={styles.modeDescription}>{mode.description}</p>
            <div>
              <span style={getRiskStyle(mode.risk_level)}>
                {mode.risk_level.toUpperCase()} RISK
              </span>
            </div>
            <p style={styles.modeBestFor}>Best for: {mode.best_for}</p>
          </div>
        ))}
      </div>

      {/* Additional Settings */}
      <div style={styles.settingsForm}>
        <h3 style={styles.formTitle}>⚙️ Additional Settings</h3>
        
        <div style={styles.formGroup}>
          <label style={styles.label}>Minimum Payout Amount (LTC)</label>
          <input
            type="number"
            style={styles.input}
            value={minPayout}
            onChange={(e) => setMinPayout(e.target.value)}
            min="0.001"
            step="0.001"
            placeholder="0.01"
          />
        </div>

        <div style={styles.formGroup}>
          <label style={styles.checkbox}>
            <input
              type="checkbox"
              style={styles.checkboxInput}
              checked={autoPayout}
              onChange={(e) => setAutoPayout(e.target.checked)}
            />
            <span style={{ color: colors.textPrimary }}>
              Enable automatic payouts when threshold is reached
            </span>
          </label>
        </div>

        <button
          style={{
            ...styles.button,
            ...(saving ? styles.buttonDisabled : {}),
          }}
          onClick={handleSave}
          disabled={saving}
        >
          {saving ? 'Saving...' : 'Save Payout Settings'}
        </button>

        {message && (
          <div style={message.type === 'success' ? styles.successMessage : styles.errorMessage}>
            {message.text}
          </div>
        )}
      </div>
    </section>
  );
}

export default PayoutSettings;
