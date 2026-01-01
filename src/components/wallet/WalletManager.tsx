import React, { useState, useEffect } from 'react';
import { colors, gradients } from '../../styles/shared';

// ============================================================================
// WALLET MANAGER COMPONENT
// Manages user's payout wallets with percentage allocation
// ============================================================================

export interface UserWallet {
  id: number;
  address: string;
  label: string;
  percentage: number;
  is_primary: boolean;
  is_active: boolean;
  created_at: string;
}

export interface WalletSummary {
  total_wallets: number;
  active_wallets: number;
  total_percentage: number;
  remaining_percentage: number;
  has_primary_wallet: boolean;
}

export interface WalletManagerProps {
  token: string;
  showMessage: (type: 'success' | 'error', text: string) => void;
}

const styles: { [key: string]: React.CSSProperties } = {
  section: {
    background: 'linear-gradient(180deg, rgba(45, 31, 61, 0.5) 0%, rgba(26, 15, 30, 0.7) 100%)',
    borderRadius: '16px',
    padding: '24px',
    border: '1px solid rgba(74, 44, 90, 0.5)',
    marginBottom: '30px',
    boxShadow: '0 4px 20px rgba(0, 0, 0, 0.2)',
  },
  sectionTitle: {
    fontSize: '1.25rem',
    color: '#F0EDF4',
    margin: '0 0 16px',
    fontWeight: 700,
  },
  loading: {
    textAlign: 'center',
    padding: '40px',
    color: colors.primary,
  },
  header: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: '20px',
  },
  formContainer: {
    backgroundColor: colors.bgInput,
    padding: '20px',
    borderRadius: '8px',
    marginBottom: '20px',
    border: `1px solid ${colors.border}`,
  },
  formTitle: {
    color: colors.primary,
    margin: '0 0 15px',
    fontSize: '1.1rem',
  },
  form: {
    display: 'flex',
    flexDirection: 'column' as const,
    gap: '15px',
  },
  formRow: {
    display: 'flex',
    gap: '20px',
    flexWrap: 'wrap' as const,
  },
  formGroup: {
    flex: 1,
    minWidth: '200px',
  },
  formActions: {
    display: 'flex',
    gap: '10px',
    justifyContent: 'flex-end',
    marginTop: '10px',
  },
  label: {
    display: 'block',
    color: colors.textSecondary,
    fontSize: '0.85rem',
    textTransform: 'uppercase' as const,
    marginBottom: '6px',
  },
  input: {
    width: '100%',
    padding: '10px 14px',
    backgroundColor: colors.bgCard,
    border: `1px solid ${colors.border}`,
    borderRadius: '6px',
    color: colors.textPrimary,
    fontSize: '0.95rem',
    boxSizing: 'border-box' as const,
  },
  percentageInput: {
    display: 'flex',
    alignItems: 'center',
    gap: '8px',
  },
  percentSign: {
    color: colors.textSecondary,
    fontSize: '1rem',
  },
  checkboxLabel: {
    display: 'flex',
    alignItems: 'center',
    gap: '8px',
    color: colors.textPrimary,
    cursor: 'pointer',
    marginTop: '20px',
  },
  addBtn: {
    padding: '10px 20px',
    backgroundColor: colors.primary,
    border: 'none',
    borderRadius: '6px',
    color: colors.bgDark,
    fontWeight: 'bold',
    cursor: 'pointer',
    fontSize: '0.9rem',
  },
  saveBtn: {
    padding: '10px 20px',
    backgroundColor: colors.primary,
    border: 'none',
    borderRadius: '6px',
    color: colors.bgDark,
    fontWeight: 'bold',
    cursor: 'pointer',
  },
  cancelBtn: {
    padding: '10px 20px',
    backgroundColor: colors.border,
    border: 'none',
    borderRadius: '6px',
    color: colors.textPrimary,
    cursor: 'pointer',
  },
  hint: {
    color: '#666',
    fontSize: '0.8rem',
    marginTop: '4px',
  },
  summaryBar: {
    display: 'flex',
    gap: '20px',
    alignItems: 'center',
    background: 'linear-gradient(180deg, rgba(13, 8, 17, 0.7) 0%, rgba(26, 15, 30, 0.85) 100%)',
    padding: '18px 24px',
    borderRadius: '12px',
    marginBottom: '20px',
    flexWrap: 'wrap' as const,
    border: '1px solid rgba(74, 44, 90, 0.4)',
  },
  summaryItem: {
    display: 'flex',
    flexDirection: 'column' as const,
    gap: '4px',
  },
  summaryLabel: {
    color: '#B8B4C8',
    fontSize: '0.75rem',
    textTransform: 'uppercase' as const,
    letterSpacing: '0.03em',
  },
  summaryValue: {
    color: '#D4A84B',
    fontSize: '1.4rem',
    fontWeight: 700,
  },
  progressBar: {
    flex: 1,
    minWidth: '200px',
    height: '10px',
    backgroundColor: 'rgba(26, 15, 30, 0.8)',
    borderRadius: '5px',
    overflow: 'hidden',
  },
  progressFill: {
    height: '100%',
    background: 'linear-gradient(90deg, #4ADE80 0%, #22C55E 100%)',
    borderRadius: '5px',
    transition: 'width 0.5s ease',
  },
  walletsList: {
    display: 'flex',
    flexDirection: 'column' as const,
    gap: '12px',
  },
  walletCard: {
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'space-between',
    background: 'linear-gradient(180deg, rgba(13, 8, 17, 0.7) 0%, rgba(26, 15, 30, 0.85) 100%)',
    padding: '18px 22px',
    borderRadius: '12px',
    border: '1px solid rgba(74, 44, 90, 0.4)',
    transition: 'all 0.2s ease',
  },
  walletMain: {
    display: 'flex',
    alignItems: 'center',
    gap: '30px',
    flex: 1,
  },
  walletInfo: {
    display: 'flex',
    flexDirection: 'column' as const,
    gap: '6px',
  },
  walletHeader: {
    display: 'flex',
    alignItems: 'center',
    gap: '10px',
  },
  walletLabel: {
    color: colors.textPrimary,
    fontWeight: 'bold',
    fontSize: '1rem',
  },
  walletPercentage: {
    display: 'flex',
    flexDirection: 'column' as const,
    alignItems: 'flex-end',
    gap: '2px',
  },
  percentageValue: {
    color: colors.primary,
    fontSize: '1.5rem',
    fontWeight: 'bold',
  },
  percentageLabel: {
    color: '#666',
    fontSize: '0.75rem',
  },
  walletActions: {
    display: 'flex',
    gap: '8px',
    alignItems: 'center',
  },
  editBtn: {
    background: 'none',
    border: 'none',
    cursor: 'pointer',
    fontSize: '1.1rem',
    padding: '6px',
  },
  deleteBtn: {
    background: 'none',
    border: 'none',
    cursor: 'pointer',
    fontSize: '1.1rem',
    padding: '6px',
  },
  toggleSwitch: {
    position: 'relative' as const,
    width: '50px',
    height: '26px',
    backgroundColor: '#4a1a1a',
    borderRadius: '13px',
    cursor: 'pointer',
    transition: 'background-color 0.3s',
  },
  toggleSwitchActive: {
    backgroundColor: '#1a4a2a',
  },
  toggleKnob: {
    position: 'absolute' as const,
    top: '3px',
    left: '3px',
    width: '20px',
    height: '20px',
    backgroundColor: '#fff',
    borderRadius: '50%',
    transition: 'transform 0.3s',
  },
  toggleKnobActive: {
    transform: 'translateX(24px)',
  },
  allocationSlider: {
    display: 'flex',
    alignItems: 'center',
    gap: '10px',
    marginTop: '10px',
    padding: '10px',
    backgroundColor: 'rgba(0,0,0,0.2)',
    borderRadius: '8px',
  },
  slider: {
    flex: 1,
    height: '6px',
    appearance: 'none' as const,
    backgroundColor: 'rgba(74, 44, 90, 0.5)',
    borderRadius: '3px',
    cursor: 'pointer',
  },
  sliderValue: {
    minWidth: '60px',
    textAlign: 'right' as const,
    color: colors.primary,
    fontWeight: 'bold',
  },
  editForm: {
    display: 'flex',
    gap: '10px',
    alignItems: 'center',
    flexWrap: 'wrap' as const,
    width: '100%',
  },
  addressCode: {
    fontFamily: 'monospace',
    color: colors.textSecondary,
    fontSize: '0.85rem',
  },
  primaryBadge: {
    backgroundColor: '#4a3a1a',
    color: colors.warning,
    padding: '3px 8px',
    borderRadius: '4px',
    fontSize: '0.75rem',
  },
  inactiveBadge: {
    backgroundColor: '#4a1a1a',
    color: colors.error,
    padding: '3px 8px',
    borderRadius: '4px',
    fontSize: '0.75rem',
  },
  emptyState: {
    textAlign: 'center' as const,
    padding: '48px',
    color: '#B8B4C8',
    background: 'linear-gradient(180deg, rgba(45, 31, 61, 0.4) 0%, rgba(26, 15, 30, 0.6) 100%)',
    borderRadius: '12px',
    border: '2px dashed rgba(74, 44, 90, 0.5)',
  },
  previewBox: {
    backgroundColor: colors.bgInput,
    padding: '20px',
    borderRadius: '8px',
    marginTop: '20px',
    border: `1px dashed ${colors.border}`,
  },
  previewTitle: {
    color: colors.textSecondary,
    margin: '0 0 15px',
    fontSize: '0.95rem',
  },
  previewList: {
    display: 'flex',
    flexDirection: 'column' as const,
    gap: '8px',
  },
  previewItem: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    padding: '8px 12px',
    backgroundColor: colors.bgCard,
    borderRadius: '4px',
  },
  previewAmount: {
    color: colors.primary,
    fontWeight: 'bold',
    fontFamily: 'monospace',
  },
};

export function WalletManager({ token, showMessage }: WalletManagerProps) {
  const [wallets, setWallets] = useState<UserWallet[]>([]);
  const [summary, setSummary] = useState<WalletSummary | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [showAddForm, setShowAddForm] = useState(false);
  const [editingWallet, setEditingWallet] = useState<UserWallet | null>(null);
  const [newWallet, setNewWallet] = useState({ 
    address: '', 
    label: '', 
    percentage: 100, 
    is_primary: false 
  });

  useEffect(() => {
    fetchWallets();
  }, [token]);

  const fetchWallets = async () => {
    try {
      const res = await fetch('/api/v1/user/wallets', {
        headers: { 'Authorization': `Bearer ${token}` }
      });
      if (res.ok) {
        const data = await res.json();
        setWallets(data.wallets || []);
        setSummary(data.summary || null);
      }
    } catch (error) {
      console.error('Failed to fetch wallets:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleAddWallet = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newWallet.address.trim()) return;
    
    setSaving(true);
    try {
      const res = await fetch('/api/v1/user/wallets', {
        method: 'POST',
        headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
        body: JSON.stringify(newWallet)
      });
      
      if (res.ok) {
        showMessage('success', 'Wallet added successfully!');
        setNewWallet({ 
          address: '', 
          label: '', 
          percentage: summary?.remaining_percentage || 100, 
          is_primary: false 
        });
        setShowAddForm(false);
        fetchWallets();
      } else {
        const data = await res.json();
        showMessage('error', data.error || 'Failed to add wallet');
      }
    } catch (error) {
      showMessage('error', 'Network error. Please try again.');
    } finally {
      setSaving(false);
    }
  };

  const handleUpdateWallet = async (wallet: UserWallet) => {
    setSaving(true);
    try {
      const res = await fetch(`/api/v1/user/wallets/${wallet.id}`, {
        method: 'PUT',
        headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
        body: JSON.stringify(wallet)
      });
      
      if (res.ok) {
        showMessage('success', 'Wallet updated successfully!');
        setEditingWallet(null);
        fetchWallets();
      } else {
        const data = await res.json();
        showMessage('error', data.error || 'Failed to update wallet');
      }
    } catch (error) {
      showMessage('error', 'Network error. Please try again.');
    } finally {
      setSaving(false);
    }
  };

  const handleDeleteWallet = async (walletId: number) => {
    if (!window.confirm('Are you sure you want to delete this wallet?')) return;
    
    try {
      const res = await fetch(`/api/v1/user/wallets/${walletId}`, {
        method: 'DELETE',
        headers: { 'Authorization': `Bearer ${token}` }
      });
      
      if (res.ok) {
        showMessage('success', 'Wallet deleted successfully!');
        fetchWallets();
      } else {
        const data = await res.json();
        showMessage('error', data.error || 'Failed to delete wallet');
      }
    } catch (error) {
      showMessage('error', 'Network error. Please try again.');
    }
  };

  // Toggle wallet active/inactive with auto-rebalancing
  const handleToggleActive = async (walletId: number, newActiveState: boolean) => {
    setSaving(true);
    try {
      const res = await fetch(`/api/v1/user/wallets/${walletId}/toggle`, {
        method: 'PUT',
        headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
        body: JSON.stringify({ is_active: newActiveState })
      });
      
      if (res.ok) {
        const data = await res.json();
        showMessage('success', newActiveState ? 'Wallet activated - allocations auto-balanced' : 'Wallet deactivated - allocations redistributed');
        fetchWallets(); // Refresh to get new allocations
      } else {
        const data = await res.json();
        showMessage('error', data.error || 'Failed to toggle wallet');
      }
    } catch (error) {
      showMessage('error', 'Network error. Please try again.');
    } finally {
      setSaving(false);
    }
  };

  // Update allocation with auto-rebalancing
  const handleAllocationChange = async (walletId: number, newPercentage: number) => {
    if (newPercentage < 0 || newPercentage > 100) return;
    
    setSaving(true);
    try {
      const res = await fetch(`/api/v1/user/wallets/${walletId}/allocation`, {
        method: 'PUT',
        headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
        body: JSON.stringify({ percentage: newPercentage })
      });
      
      if (res.ok) {
        showMessage('success', 'Allocation updated - other wallets auto-balanced');
        fetchWallets(); // Refresh to get new allocations
      } else {
        const data = await res.json();
        showMessage('error', data.error || 'Failed to update allocation');
      }
    } catch (error) {
      showMessage('error', 'Network error. Please try again.');
    } finally {
      setSaving(false);
    }
  };

  if (loading) {
    return (
      <section style={styles.section}>
        <h2 style={styles.sectionTitle}>💰 Wallet Settings</h2>
        <div style={styles.loading}>Loading wallet settings...</div>
      </section>
    );
  }

  return (
    <section style={styles.section}>
      <div style={styles.header}>
        <h2 style={{ ...styles.sectionTitle, margin: 0 }}>💰 Multi-Wallet Payout Settings</h2>
        <button 
          style={styles.addBtn} 
          onClick={() => { 
            setShowAddForm(true); 
            setNewWallet({ ...newWallet, percentage: summary?.remaining_percentage || 100 }); 
          }}
          disabled={summary?.remaining_percentage === 0}
        >
          + Add Wallet
        </button>
      </div>

      {/* Summary Bar */}
      {summary && (
        <div style={styles.summaryBar}>
          <div style={styles.summaryItem}>
            <span style={styles.summaryLabel}>Active Wallets</span>
            <span style={styles.summaryValue}>{summary.active_wallets}</span>
          </div>
          <div style={styles.summaryItem}>
            <span style={styles.summaryLabel}>Allocated</span>
            <span style={styles.summaryValue}>{summary.total_percentage.toFixed(1)}%</span>
          </div>
          <div style={styles.summaryItem}>
            <span style={styles.summaryLabel}>Remaining</span>
            <span style={{
              ...styles.summaryValue, 
              color: summary.remaining_percentage > 0 ? colors.warning : colors.success
            }}>
              {summary.remaining_percentage.toFixed(1)}%
            </span>
          </div>
          <div style={styles.progressBar}>
            <div style={{ ...styles.progressFill, width: `${summary.total_percentage}%` }}></div>
          </div>
        </div>
      )}

      {/* Add Wallet Form */}
      {showAddForm && (
        <div style={styles.formContainer}>
          <h3 style={styles.formTitle}>Add New Wallet</h3>
          <form onSubmit={handleAddWallet} style={styles.form}>
            <div style={styles.formRow}>
              <div style={styles.formGroup}>
                <label style={styles.label}>Wallet Address *</label>
                <input
                  style={styles.input}
                  type="text"
                  value={newWallet.address}
                  onChange={(e) => setNewWallet({ ...newWallet, address: e.target.value })}
                  placeholder="0x..."
                  required
                />
              </div>
              <div style={styles.formGroup}>
                <label style={styles.label}>Label</label>
                <input
                  style={styles.input}
                  type="text"
                  value={newWallet.label}
                  onChange={(e) => setNewWallet({ ...newWallet, label: e.target.value })}
                  placeholder="e.g., Main, Hardware, Exchange"
                />
              </div>
            </div>
            <div style={styles.formRow}>
              <div style={styles.formGroup}>
                <label style={styles.label}>Payout Percentage *</label>
                <div style={styles.percentageInput}>
                  <input
                    style={{ ...styles.input, width: '100px' }}
                    type="number"
                    min="0.01"
                    max={summary?.remaining_percentage || 100}
                    step="0.01"
                    value={newWallet.percentage}
                    onChange={(e) => setNewWallet({ ...newWallet, percentage: parseFloat(e.target.value) || 0 })}
                    required
                  />
                  <span style={styles.percentSign}>%</span>
                </div>
                <p style={styles.hint}>Available: {summary?.remaining_percentage.toFixed(2)}%</p>
              </div>
              <div style={styles.formGroup}>
                <label style={styles.checkboxLabel}>
                  <input
                    type="checkbox"
                    checked={newWallet.is_primary}
                    onChange={(e) => setNewWallet({ ...newWallet, is_primary: e.target.checked })}
                  />
                  Set as Primary Wallet
                </label>
              </div>
            </div>
            <div style={styles.formActions}>
              <button type="button" style={styles.cancelBtn} onClick={() => setShowAddForm(false)}>
                Cancel
              </button>
              <button type="submit" style={styles.saveBtn} disabled={saving}>
                {saving ? 'Adding...' : 'Add Wallet'}
              </button>
            </div>
          </form>
        </div>
      )}

      {/* Wallets List */}
      {wallets.length === 0 ? (
        <div style={styles.emptyState}>
          <p>No wallets configured yet.</p>
          <p style={{ color: colors.textSecondary, fontSize: '0.9rem' }}>
            Add a wallet to receive mining payouts.
          </p>
        </div>
      ) : (
        <div style={styles.walletsList}>
          {wallets.map((wallet) => (
            <div 
              key={wallet.id} 
              style={{ ...styles.walletCard, opacity: wallet.is_active ? 1 : 0.6 }}
            >
              {editingWallet?.id === wallet.id ? (
                <div style={styles.editForm}>
                  <input
                    style={styles.input}
                    type="text"
                    value={editingWallet.address}
                    onChange={(e) => setEditingWallet({ ...editingWallet, address: e.target.value })}
                    placeholder="Wallet address"
                  />
                  <input
                    style={{ ...styles.input, width: '150px' }}
                    type="text"
                    value={editingWallet.label}
                    onChange={(e) => setEditingWallet({ ...editingWallet, label: e.target.value })}
                    placeholder="Label"
                  />
                  <div style={styles.percentageInput}>
                    <input
                      style={{ ...styles.input, width: '80px' }}
                      type="number"
                      min="0.01"
                      max="100"
                      step="0.01"
                      value={editingWallet.percentage}
                      onChange={(e) => setEditingWallet({ 
                        ...editingWallet, 
                        percentage: parseFloat(e.target.value) || 0 
                      })}
                    />
                    <span style={styles.percentSign}>%</span>
                  </div>
                  <button 
                    style={styles.saveBtn} 
                    onClick={() => handleUpdateWallet(editingWallet)} 
                    disabled={saving}
                  >
                    Save
                  </button>
                  <button style={styles.cancelBtn} onClick={() => setEditingWallet(null)}>
                    Cancel
                  </button>
                </div>
              ) : (
                <div style={{ width: '100%' }}>
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                    <div style={styles.walletMain}>
                      <div style={styles.walletInfo}>
                        <div style={styles.walletHeader}>
                          <span style={styles.walletLabel}>{wallet.label || 'Wallet'}</span>
                          {wallet.is_primary && (
                            <span style={styles.primaryBadge}>⭐ Primary</span>
                          )}
                          {!wallet.is_active && (
                            <span style={styles.inactiveBadge}>Inactive</span>
                          )}
                        </div>
                        <code style={styles.addressCode}>
                          {wallet.address.slice(0, 12)}...{wallet.address.slice(-10)}
                        </code>
                      </div>
                      <div style={styles.walletPercentage}>
                        <span style={styles.percentageValue}>{wallet.percentage.toFixed(1)}%</span>
                        <span style={styles.percentageLabel}>of payouts</span>
                      </div>
                    </div>
                    <div style={styles.walletActions}>
                      {/* Active/Inactive Toggle */}
                      <div 
                        data-testid={`wallet-toggle-${wallet.id}`}
                        style={{
                          ...styles.toggleSwitch,
                          ...(wallet.is_active ? styles.toggleSwitchActive : {})
                        }}
                        onClick={() => !saving && handleToggleActive(wallet.id, !wallet.is_active)}
                        title={wallet.is_active ? 'Click to deactivate' : 'Click to activate'}
                      >
                        <div style={{
                          ...styles.toggleKnob,
                          ...(wallet.is_active ? styles.toggleKnobActive : {})
                        }} />
                      </div>
                      <button 
                        style={styles.editBtn} 
                        onClick={() => setEditingWallet({ ...wallet })}
                        data-testid={`wallet-edit-${wallet.id}`}
                      >
                        ✏️
                      </button>
                      <button 
                        style={styles.deleteBtn} 
                        onClick={() => handleDeleteWallet(wallet.id)}
                        data-testid={`wallet-delete-${wallet.id}`}
                      >
                        🗑️
                      </button>
                    </div>
                  </div>
                  
                  {/* Allocation Slider - only show for active wallets with multiple wallets */}
                  {wallet.is_active && wallets.filter(w => w.is_active).length > 1 && (
                    <div style={styles.allocationSlider}>
                      <span style={{ color: '#888', fontSize: '0.8rem' }}>Allocation:</span>
                      <input
                        type="range"
                        min="1"
                        max="99"
                        value={wallet.percentage}
                        onChange={(e) => {
                          const newVal = parseFloat(e.target.value);
                          // Optimistic UI update
                          setWallets(prev => prev.map(w => 
                            w.id === wallet.id ? { ...w, percentage: newVal } : w
                          ));
                        }}
                        onMouseUp={(e) => handleAllocationChange(wallet.id, parseFloat((e.target as HTMLInputElement).value))}
                        onTouchEnd={(e) => handleAllocationChange(wallet.id, parseFloat((e.target as HTMLInputElement).value))}
                        style={styles.slider}
                        disabled={saving}
                        data-testid={`wallet-slider-${wallet.id}`}
                      />
                      <span style={styles.sliderValue}>{wallet.percentage.toFixed(1)}%</span>
                    </div>
                  )}
                </div>
              )}
            </div>
          ))}
        </div>
      )}

      {/* Payout Split Preview */}
      {wallets.length > 1 && summary && summary.total_percentage === 100 && (
        <div style={styles.previewBox}>
          <h4 style={styles.previewTitle}>📊 Payout Split Preview (Example: 10 BDAG)</h4>
          <div style={styles.previewList}>
            {wallets.filter(w => w.is_active).map((wallet) => (
              <div key={wallet.id} style={styles.previewItem}>
                <span>{wallet.label || 'Wallet'}</span>
                <span style={styles.previewAmount}>
                  {(10 * wallet.percentage / 100).toFixed(4)} BDAG
                </span>
              </div>
            ))}
          </div>
        </div>
      )}
    </section>
  );
}

export default WalletManager;
