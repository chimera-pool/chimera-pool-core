import React, { useState } from 'react';

interface BugReportModalProps {
  isOpen: boolean;
  onClose: () => void;
  token: string;
  showMessage: (type: 'success' | 'error', text: string) => void;
}

interface BugReportForm {
  title: string;
  description: string;
  steps_to_reproduce: string;
  expected_behavior: string;
  actual_behavior: string;
  category: string;
  screenshot: string;
}

const styles = {
  overlay: {
    position: 'fixed' as const,
    top: 0,
    left: 0,
    right: 0,
    bottom: 0,
    backgroundColor: 'rgba(13, 8, 17, 0.95)',
    backdropFilter: 'blur(8px)',
    display: 'flex',
    justifyContent: 'center',
    alignItems: 'center',
    zIndex: 2000,
    padding: '20px',
  },
  modal: {
    background: 'linear-gradient(180deg, #2D1F3D 0%, #1A0F1E 100%)',
    borderRadius: '16px',
    padding: '28px',
    width: '100%',
    maxWidth: '600px',
    maxHeight: '90vh',
    overflow: 'auto',
    border: '1px solid rgba(74, 44, 90, 0.4)',
    boxShadow: '0 24px 48px rgba(0, 0, 0, 0.5)',
  },
  title: {
    color: '#D4A84B',
    marginBottom: '20px',
    display: 'flex',
    alignItems: 'center',
    gap: '10px',
    fontSize: '1.4rem',
    fontWeight: 600,
  },
  label: {
    display: 'block',
    color: '#B8B4C8',
    marginBottom: '8px',
    fontSize: '0.9rem',
    fontWeight: 500,
  },
  input: {
    width: '100%',
    padding: '14px',
    backgroundColor: 'rgba(26, 15, 30, 0.8)',
    border: '1px solid #4A2C5A',
    borderRadius: '10px',
    color: '#F0EDF4',
    fontSize: '1rem',
    boxSizing: 'border-box' as const,
  },
  textarea: {
    width: '100%',
    padding: '14px',
    backgroundColor: 'rgba(26, 15, 30, 0.8)',
    border: '1px solid #4A2C5A',
    borderRadius: '10px',
    color: '#F0EDF4',
    fontSize: '1rem',
    minHeight: '100px',
    resize: 'vertical' as const,
    boxSizing: 'border-box' as const,
  },
  select: {
    width: '100%',
    padding: '14px',
    backgroundColor: 'rgba(26, 15, 30, 0.8)',
    border: '1px solid #4A2C5A',
    borderRadius: '10px',
    color: '#F0EDF4',
    fontSize: '1rem',
    cursor: 'pointer',
  },
  formGroup: {
    marginBottom: '15px',
  },
  buttonRow: {
    display: 'flex',
    gap: '12px',
    marginTop: '24px',
  },
  cancelBtn: {
    flex: 1,
    padding: '14px',
    backgroundColor: 'transparent',
    border: '1px solid #4A2C5A',
    borderRadius: '10px',
    color: '#B8B4C8',
    cursor: 'pointer',
    fontSize: '1rem',
  },
  submitBtn: {
    flex: 1,
    padding: '14px',
    background: 'linear-gradient(135deg, #D4A84B 0%, #B8923A 100%)',
    border: 'none',
    borderRadius: '10px',
    color: '#1A0F1E',
    fontWeight: 600,
    cursor: 'pointer',
    fontSize: '1rem',
  },
  uploadLabel: {
    flex: 1,
    padding: '14px',
    backgroundColor: 'rgba(26, 15, 30, 0.8)',
    border: '2px dashed #7B5EA7',
    borderRadius: '10px',
    color: '#B8B4C8',
    cursor: 'pointer',
    textAlign: 'center' as const,
  },
  removeBtn: {
    padding: '14px',
    backgroundColor: 'rgba(139, 69, 69, 0.3)',
    border: '1px solid #8B4545',
    borderRadius: '10px',
    color: '#f87171',
    cursor: 'pointer',
  },
};

export function BugReportModal({ isOpen, onClose, token, showMessage }: BugReportModalProps) {
  const [form, setForm] = useState<BugReportForm>({
    title: '',
    description: '',
    steps_to_reproduce: '',
    expected_behavior: '',
    actual_behavior: '',
    category: 'other',
    screenshot: '',
  });
  const [loading, setLoading] = useState(false);

  const resetForm = () => {
    setForm({
      title: '',
      description: '',
      steps_to_reproduce: '',
      expected_behavior: '',
      actual_behavior: '',
      category: 'other',
      screenshot: '',
    });
  };

  const handleSubmit = async () => {
    if (!form.title || !form.description) {
      showMessage('error', 'Title and description are required');
      return;
    }

    setLoading(true);
    try {
      const response = await fetch('/api/v1/bugs', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`,
        },
        body: JSON.stringify({
          title: form.title,
          description: form.description,
          steps_to_reproduce: form.steps_to_reproduce,
          expected_behavior: form.expected_behavior,
          actual_behavior: form.actual_behavior,
          category: form.category,
          screenshot: form.screenshot || undefined,
        }),
      });

      if (response.ok) {
        showMessage('success', 'Bug report submitted successfully! Thank you for helping improve the pool.');
        resetForm();
        onClose();
      } else {
        const data = await response.json();
        showMessage('error', data.error || 'Failed to submit bug report');
      }
    } catch (error) {
      showMessage('error', 'Network error. Please try again.');
    } finally {
      setLoading(false);
    }
  };

  const handleFileChange = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) {
      if (file.size > 10 * 1024 * 1024) {
        showMessage('error', 'Screenshot must be under 10MB');
        return;
      }
      const reader = new FileReader();
      reader.onload = () => {
        const base64 = (reader.result as string).split(',')[1];
        setForm({ ...form, screenshot: base64 });
      };
      reader.readAsDataURL(file);
    }
  };

  if (!isOpen) return null;

  return (
    <div style={styles.overlay} onClick={onClose} data-testid="bug-report-modal-overlay">
      <div style={styles.modal} onClick={e => e.stopPropagation()} data-testid="bug-report-modal-container">
        <h2 style={styles.title}>🐛 Report a Bug</h2>

        <div style={styles.formGroup}>
          <label style={styles.label}>Title *</label>
          <input
            style={styles.input}
            placeholder="Brief description of the issue"
            value={form.title}
            onChange={e => setForm({ ...form, title: e.target.value })}
            data-testid="bug-report-title-input"
            aria-label="Bug title"
          />
        </div>

        <div style={styles.formGroup}>
          <label style={styles.label}>Category</label>
          <select
            style={styles.select}
            value={form.category}
            onChange={e => setForm({ ...form, category: e.target.value })}
            data-testid="bug-report-category-select"
            aria-label="Bug category"
          >
            <option value="ui">🎨 UI/Visual Issue</option>
            <option value="performance">⚡ Performance</option>
            <option value="crash">💥 Crash/Error</option>
            <option value="security">🔒 Security Concern</option>
            <option value="feature_request">✨ Feature Request</option>
            <option value="other">📝 Other</option>
          </select>
        </div>

        <div style={styles.formGroup}>
          <label style={styles.label}>Description *</label>
          <textarea
            style={styles.textarea}
            placeholder="Describe the issue in detail..."
            value={form.description}
            onChange={e => setForm({ ...form, description: e.target.value })}
            data-testid="bug-report-description-input"
            aria-label="Bug description"
          />
        </div>

        <div style={styles.formGroup}>
          <label style={styles.label}>Steps to Reproduce</label>
          <textarea
            style={{ ...styles.textarea, minHeight: '80px' }}
            placeholder="1. Go to...&#10;2. Click on...&#10;3. See error"
            value={form.steps_to_reproduce}
            onChange={e => setForm({ ...form, steps_to_reproduce: e.target.value })}
          />
        </div>

        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '15px', marginBottom: '15px' }}>
          <div>
            <label style={styles.label}>Expected Behavior</label>
            <textarea
              style={{ ...styles.textarea, minHeight: '60px', fontSize: '0.9rem' }}
              placeholder="What should happen?"
              value={form.expected_behavior}
              onChange={e => setForm({ ...form, expected_behavior: e.target.value })}
            />
          </div>
          <div>
            <label style={styles.label}>Actual Behavior</label>
            <textarea
              style={{ ...styles.textarea, minHeight: '60px', fontSize: '0.9rem' }}
              placeholder="What actually happens?"
              value={form.actual_behavior}
              onChange={e => setForm({ ...form, actual_behavior: e.target.value })}
            />
          </div>
        </div>

        <div style={styles.formGroup}>
          <label style={styles.label}>📸 Screenshot (optional)</label>
          <div style={{ display: 'flex', gap: '10px', alignItems: 'center' }}>
            <label style={styles.uploadLabel}>
              <input
                type="file"
                accept="image/*"
                style={{ display: 'none' }}
                onChange={handleFileChange}
              />
              {form.screenshot ? '✅ Screenshot attached' : '📎 Click to attach screenshot'}
            </label>
            {form.screenshot && (
              <button
                style={styles.removeBtn}
                onClick={() => setForm({ ...form, screenshot: '' })}
              >
                ✕
              </button>
            )}
          </div>
        </div>

        <div style={styles.buttonRow}>
          <button style={styles.cancelBtn} onClick={onClose} data-testid="bug-report-cancel-btn">
            Cancel
          </button>
          <button
            style={{ ...styles.submitBtn, opacity: loading ? 0.7 : 1 }}
            onClick={handleSubmit}
            disabled={loading}
            data-testid="bug-report-submit-btn"
          >
            {loading ? 'Submitting...' : '🐛 Submit Report'}
          </button>
        </div>
      </div>
    </div>
  );
}

export default BugReportModal;
