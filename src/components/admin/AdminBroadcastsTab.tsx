import React, { useState, useEffect } from 'react';

interface Broadcast {
  id: number;
  subject: string;
  message: string;
  broadcastType: string;
  sendEmail: boolean;
  recipientCount: number;
  emailSentCount: number;
  createdAt: string;
  adminName: string;
}

interface AdminBroadcastsTabProps {
  token: string;
  isActive: boolean;
  showMessage: (type: 'success' | 'error', text: string) => void;
}

const AdminBroadcastsTab: React.FC<AdminBroadcastsTabProps> = ({ token, isActive, showMessage }) => {
  const [broadcasts, setBroadcasts] = useState<Broadcast[]>([]);
  const [loading, setLoading] = useState(false);
  const [sending, setSending] = useState(false);
  const [form, setForm] = useState({
    subject: '',
    message: '',
    broadcastType: 'all',
    sendEmail: false
  });

  const fetchBroadcasts = async () => {
    setLoading(true);
    try {
      const res = await fetch('/api/v1/admin/broadcasts', {
        headers: { Authorization: `Bearer ${token}` }
      });
      if (res.ok) {
        const data = await res.json();
        setBroadcasts(data.broadcasts || []);
      }
    } catch (e) {
      console.error('Failed to fetch broadcasts:', e);
    }
    setLoading(false);
  };

  const sendBroadcast = async () => {
    if (!form.subject.trim() || !form.message.trim()) {
      showMessage('error', 'Subject and message are required');
      return;
    }

    setSending(true);
    try {
      const res = await fetch('/api/v1/admin/broadcasts', {
        method: 'POST',
        headers: {
          Authorization: `Bearer ${token}`,
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({
          subject: form.subject,
          message: form.message,
          broadcast_type: form.broadcastType,
          send_email: form.sendEmail
        })
      });

      if (res.ok) {
        const data = await res.json();
        showMessage('success', `Broadcast sent to ${data.recipientCount} users`);
        setForm({ subject: '', message: '', broadcastType: 'all', sendEmail: false });
        fetchBroadcasts();
      } else {
        const data = await res.json();
        showMessage('error', data.error || 'Failed to send broadcast');
      }
    } catch (e) {
      showMessage('error', 'Network error');
    }
    setSending(false);
  };

  useEffect(() => {
    if (isActive) fetchBroadcasts();
  }, [isActive]);

  if (!isActive) return null;

  const formatDate = (dateStr: string) => {
    return new Date(dateStr).toLocaleString();
  };

  const getBroadcastTypeLabel = (type: string) => {
    switch (type) {
      case 'all': return '👥 All Users';
      case 'miners_only': return '⛏️ Active Miners';
      case 'admins_only': return '👑 Admins Only';
      default: return type;
    }
  };

  return (
    <div style={styles.container} data-testid="admin-broadcasts-tab">
      {/* New Broadcast Form */}
      <div style={styles.formSection}>
        <h3 style={styles.sectionTitle}>📢 Send New Broadcast</h3>
        
        <div style={styles.formGroup}>
          <label style={styles.label}>Subject</label>
          <input
            type="text"
            style={styles.input}
            value={form.subject}
            onChange={e => setForm({ ...form, subject: e.target.value })}
            placeholder="Broadcast subject..."
            data-testid="broadcast-subject-input"
          />
        </div>

        <div style={styles.formGroup}>
          <label style={styles.label}>Message</label>
          <textarea
            style={styles.textarea}
            value={form.message}
            onChange={e => setForm({ ...form, message: e.target.value })}
            placeholder="Write your broadcast message..."
            rows={4}
            data-testid="broadcast-message-input"
          />
        </div>

        <div style={styles.formRow}>
          <div style={styles.formGroup}>
            <label style={styles.label}>Recipients</label>
            <select
              style={styles.select}
              value={form.broadcastType}
              onChange={e => setForm({ ...form, broadcastType: e.target.value })}
              data-testid="broadcast-type-select"
            >
              <option value="all">All Users</option>
              <option value="miners_only">Active Miners Only</option>
              <option value="admins_only">Admins Only</option>
            </select>
          </div>

          <div style={styles.checkboxGroup}>
            <label style={styles.checkboxLabel}>
              <input
                type="checkbox"
                checked={form.sendEmail}
                onChange={e => setForm({ ...form, sendEmail: e.target.checked })}
                data-testid="broadcast-email-checkbox"
              />
              <span style={styles.checkboxText}>Also send via email</span>
            </label>
          </div>
        </div>

        <button
          style={styles.sendButton}
          onClick={sendBroadcast}
          disabled={sending}
          data-testid="send-broadcast-btn"
        >
          {sending ? 'Sending...' : '📤 Send Broadcast'}
        </button>
      </div>

      {/* Broadcast History */}
      <div style={styles.historySection}>
        <h3 style={styles.sectionTitle}>📋 Broadcast History</h3>
        
        {loading ? (
          <div style={styles.loading}>Loading...</div>
        ) : broadcasts.length === 0 ? (
          <div style={styles.empty}>No broadcasts sent yet</div>
        ) : (
          <div style={styles.broadcastList}>
            {broadcasts.map(broadcast => (
              <div key={broadcast.id} style={styles.broadcastItem} data-testid={`broadcast-item-${broadcast.id}`}>
                <div style={styles.broadcastHeader}>
                  <span style={styles.broadcastSubject}>{broadcast.subject}</span>
                  <span style={styles.broadcastType}>{getBroadcastTypeLabel(broadcast.broadcastType)}</span>
                </div>
                <div style={styles.broadcastMessage}>{broadcast.message}</div>
                <div style={styles.broadcastMeta}>
                  <span>By {broadcast.adminName}</span>
                  <span>•</span>
                  <span>{formatDate(broadcast.createdAt)}</span>
                  <span>•</span>
                  <span>{broadcast.recipientCount} recipients</span>
                  {broadcast.sendEmail && (
                    <>
                      <span>•</span>
                      <span>📧 {broadcast.emailSentCount} emails</span>
                    </>
                  )}
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
};

const styles: { [key: string]: React.CSSProperties } = {
  container: {
    padding: '20px',
  },
  formSection: {
    background: 'rgba(45, 31, 61, 0.4)',
    borderRadius: '12px',
    padding: '20px',
    marginBottom: '24px',
    border: '1px solid rgba(74, 44, 90, 0.5)',
  },
  historySection: {
    background: 'rgba(45, 31, 61, 0.4)',
    borderRadius: '12px',
    padding: '20px',
    border: '1px solid rgba(74, 44, 90, 0.5)',
  },
  sectionTitle: {
    color: '#D4A84B',
    fontSize: '1.1rem',
    fontWeight: 600,
    margin: '0 0 16px 0',
  },
  formGroup: {
    marginBottom: '16px',
  },
  formRow: {
    display: 'flex',
    gap: '20px',
    alignItems: 'flex-end',
    flexWrap: 'wrap' as const,
  },
  label: {
    display: 'block',
    color: '#B8B4C8',
    fontSize: '0.9rem',
    marginBottom: '6px',
  },
  input: {
    width: '100%',
    padding: '12px',
    background: 'rgba(13, 8, 17, 0.6)',
    border: '1px solid rgba(74, 44, 90, 0.5)',
    borderRadius: '8px',
    color: '#F0EDF4',
    fontSize: '0.95rem',
  },
  textarea: {
    width: '100%',
    padding: '12px',
    background: 'rgba(13, 8, 17, 0.6)',
    border: '1px solid rgba(74, 44, 90, 0.5)',
    borderRadius: '8px',
    color: '#F0EDF4',
    fontSize: '0.95rem',
    resize: 'vertical' as const,
    fontFamily: 'inherit',
  },
  select: {
    padding: '12px',
    background: 'rgba(13, 8, 17, 0.6)',
    border: '1px solid rgba(74, 44, 90, 0.5)',
    borderRadius: '8px',
    color: '#F0EDF4',
    fontSize: '0.95rem',
    minWidth: '180px',
  },
  checkboxGroup: {
    marginBottom: '16px',
  },
  checkboxLabel: {
    display: 'flex',
    alignItems: 'center',
    gap: '8px',
    cursor: 'pointer',
  },
  checkboxText: {
    color: '#B8B4C8',
    fontSize: '0.9rem',
  },
  sendButton: {
    padding: '12px 24px',
    background: 'linear-gradient(135deg, #D4A84B 0%, #B8923A 100%)',
    border: 'none',
    borderRadius: '8px',
    color: '#1A0F1E',
    fontSize: '0.95rem',
    fontWeight: 600,
    cursor: 'pointer',
    transition: 'all 0.2s',
  },
  loading: {
    textAlign: 'center' as const,
    color: '#8B8698',
    padding: '40px',
  },
  empty: {
    textAlign: 'center' as const,
    color: '#8B8698',
    padding: '40px',
  },
  broadcastList: {
    display: 'flex',
    flexDirection: 'column' as const,
    gap: '12px',
  },
  broadcastItem: {
    background: 'rgba(13, 8, 17, 0.4)',
    borderRadius: '8px',
    padding: '16px',
    border: '1px solid rgba(74, 44, 90, 0.3)',
  },
  broadcastHeader: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: '8px',
  },
  broadcastSubject: {
    color: '#F0EDF4',
    fontSize: '1rem',
    fontWeight: 600,
  },
  broadcastType: {
    color: '#D4A84B',
    fontSize: '0.8rem',
    background: 'rgba(212, 168, 75, 0.15)',
    padding: '4px 10px',
    borderRadius: '12px',
  },
  broadcastMessage: {
    color: '#B8B4C8',
    fontSize: '0.9rem',
    lineHeight: 1.5,
    marginBottom: '12px',
  },
  broadcastMeta: {
    display: 'flex',
    gap: '8px',
    color: '#8B8698',
    fontSize: '0.8rem',
    flexWrap: 'wrap' as const,
  },
};

export default AdminBroadcastsTab;
