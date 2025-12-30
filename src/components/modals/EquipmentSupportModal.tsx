import React from 'react';

// ============================================================================
// EQUIPMENT SUPPORT MODAL COMPONENT
// Extracted from App.tsx for modular architecture
// Handles equipment support request form
// ============================================================================

interface EquipmentSupportForm {
  issue_type: string;
  equipment_type: string;
  description: string;
  error_message: string;
}

interface EquipmentSupportModalProps {
  isOpen: boolean;
  onClose: () => void;
  form: EquipmentSupportForm;
  setForm: (form: EquipmentSupportForm) => void;
  onSubmit: () => void;
}

const styles = {
  overlay: {
    position: 'fixed' as const,
    top: 0,
    left: 0,
    right: 0,
    bottom: 0,
    backgroundColor: 'rgba(0,0,0,0.85)',
    display: 'flex',
    justifyContent: 'center',
    alignItems: 'center',
    zIndex: 1000,
    padding: '15px',
    boxSizing: 'border-box' as const,
  },
  modal: {
    backgroundColor: '#1a1a2e',
    padding: '20px',
    borderRadius: '12px',
    border: '2px solid #f59e0b',
    maxWidth: '480px',
    width: '100%',
    maxHeight: 'calc(100vh - 30px)',
    overflowY: 'auto' as const,
    boxSizing: 'border-box' as const,
  },
  title: {
    color: '#f59e0b',
    marginTop: 0,
  },
  description: {
    color: '#888',
    marginBottom: '20px',
  },
  label: {
    display: 'block',
    color: '#888',
    marginBottom: '4px',
    fontSize: '0.85rem',
  },
  input: {
    width: '100%',
    padding: '10px',
    backgroundColor: '#0a0a15',
    border: '1px solid #2a2a4a',
    borderRadius: '6px',
    color: '#e0e0e0',
    fontSize: '0.95rem',
    marginBottom: '12px',
    boxSizing: 'border-box' as const,
  },
  select: {
    width: '100%',
    padding: '10px',
    backgroundColor: '#0a0a15',
    border: '1px solid #2a2a4a',
    borderRadius: '6px',
    color: '#e0e0e0',
    fontSize: '0.95rem',
    marginBottom: '12px',
    cursor: 'pointer',
    boxSizing: 'border-box' as const,
  },
  textarea: {
    width: '100%',
    padding: '10px',
    backgroundColor: '#0a0a15',
    border: '1px solid #2a2a4a',
    borderRadius: '6px',
    color: '#e0e0e0',
    fontSize: '0.95rem',
    marginBottom: '12px',
    minHeight: '80px',
    resize: 'vertical' as const,
    boxSizing: 'border-box' as const,
  },
  tipBox: {
    backgroundColor: '#0a1a15',
    border: '1px solid #4ade80',
    borderRadius: '8px',
    padding: '15px',
    marginBottom: '20px',
  },
  tipText: {
    color: '#4ade80',
    margin: 0,
    fontSize: '0.9rem',
  },
  buttonRow: {
    display: 'flex',
    gap: '10px',
    justifyContent: 'flex-end',
  },
  cancelBtn: {
    padding: '10px 18px',
    backgroundColor: 'transparent',
    border: '1px solid #888',
    borderRadius: '6px',
    color: '#888',
    cursor: 'pointer',
    fontSize: '0.9rem',
  },
  submitBtn: {
    padding: '10px 18px',
    backgroundColor: '#f59e0b',
    border: 'none',
    borderRadius: '6px',
    color: '#0a0a0f',
    fontWeight: 'bold' as const,
    cursor: 'pointer',
    fontSize: '0.9rem',
  },
};

export function EquipmentSupportModal({
  isOpen,
  onClose,
  form,
  setForm,
  onSubmit,
}: EquipmentSupportModalProps) {
  if (!isOpen) return null;

  const isValid = form.equipment_type && form.description;

  return (
    <div
      style={styles.overlay}
      onClick={onClose}
      data-testid="equipment-support-modal-overlay"
    >
      <div
        style={styles.modal}
        onClick={(e) => e.stopPropagation()}
        data-testid="equipment-support-modal-container"
      >
        <h2 style={styles.title}>🆘 Equipment Support Request</h2>
        <p style={styles.description}>
          Having trouble getting your equipment online? Our team is here to help!
        </p>

        <div>
          <label style={styles.label}>Issue Type *</label>
          <select
            style={styles.select}
            value={form.issue_type}
            onChange={(e) => setForm({ ...form, issue_type: e.target.value })}
            data-testid="equipment-support-issue-type-select"
            aria-label="Issue type"
          >
            <option value="connection">Cannot connect to pool</option>
            <option value="configuration">Configuration help needed</option>
            <option value="hardware">Hardware compatibility question</option>
            <option value="performance">Low hashrate / performance issues</option>
            <option value="errors">Error messages</option>
            <option value="other">Other</option>
          </select>
        </div>

        <div>
          <label style={styles.label}>Equipment Type *</label>
          <select
            style={styles.select}
            value={form.equipment_type}
            onChange={(e) => setForm({ ...form, equipment_type: e.target.value })}
            data-testid="equipment-support-equipment-type-select"
            aria-label="Equipment type"
          >
            <option value="">Select your equipment...</option>
            <option value="blockdag_x100">BlockDAG X100 ASIC</option>
            <option value="blockdag_x30">BlockDAG X30 ASIC</option>
            <option value="gpu_nvidia">NVIDIA GPU</option>
            <option value="gpu_amd">AMD GPU</option>
            <option value="cpu">CPU</option>
            <option value="other_asic">Other ASIC</option>
            <option value="other">Other</option>
          </select>
        </div>

        <div>
          <label style={styles.label}>Describe Your Issue *</label>
          <textarea
            style={styles.textarea}
            placeholder="Please describe what's happening and what you've tried so far..."
            value={form.description}
            onChange={(e) => setForm({ ...form, description: e.target.value })}
            data-testid="equipment-support-description-input"
            aria-label="Issue description"
          />
        </div>

        <div>
          <label style={styles.label}>Error Message (if any)</label>
          <input
            style={styles.input}
            type="text"
            placeholder="Copy any error messages here"
            value={form.error_message}
            onChange={(e) => setForm({ ...form, error_message: e.target.value })}
            data-testid="equipment-support-error-input"
            aria-label="Error message"
          />
        </div>

        <div style={styles.tipBox}>
          <p style={styles.tipText}>
            💡 <strong>Quick Tips:</strong> Make sure your miner is configured with the
            correct pool address (stratum+tcp://206.162.80.230:3333) and your wallet
            address as the username.
          </p>
        </div>

        <div style={styles.buttonRow}>
          <button
            style={styles.cancelBtn}
            onClick={onClose}
            data-testid="equipment-support-cancel-btn"
          >
            Cancel
          </button>
          <button
            style={{ ...styles.submitBtn, opacity: !isValid ? 0.5 : 1 }}
            disabled={!isValid}
            onClick={onSubmit}
            data-testid="equipment-support-submit-btn"
          >
            Submit Support Request
          </button>
        </div>
      </div>
    </div>
  );
}

export default EquipmentSupportModal;
