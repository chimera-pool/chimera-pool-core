import React, { useState } from 'react';

// ============================================================================
// MY BUGS MODAL COMPONENT
// Extracted from App.tsx for modular architecture
// Displays user's bug reports list and details
// ============================================================================

interface Bug {
  id: number;
  report_number: string;
  title: string;
  description: string;
  steps_to_reproduce?: string;
  status: string;
  priority: string;
  category: string;
  attachment_count?: number;
  comment_count?: number;
  created_at: string;
}

interface BugComment {
  id: number;
  username: string;
  content: string;
  is_status_change: boolean;
  created_at: string;
}

interface BugDetails {
  bug: Bug;
  comments: BugComment[];
}

interface MyBugsModalProps {
  isOpen: boolean;
  onClose: () => void;
  bugs: Bug[];
  token: string;
  showMessage: (type: 'success' | 'error', text: string) => void;
  onOpenNewReport: () => void;
}

const styles = {
  overlay: {
    position: 'fixed' as const,
    inset: 0,
    backgroundColor: 'rgba(13, 8, 17, 0.9)',
    backdropFilter: 'blur(8px)',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    zIndex: 1000,
    padding: '15px',
    boxSizing: 'border-box' as const,
  },
  modal: {
    background: 'linear-gradient(180deg, #2D1F3D 0%, #1A0F1E 100%)',
    borderRadius: '20px',
    padding: '28px',
    maxWidth: '750px',
    width: '100%',
    border: '1px solid #4A2C5A',
    maxHeight: 'calc(100vh - 30px)',
    overflowY: 'auto' as const,
    boxSizing: 'border-box' as const,
    boxShadow: '0 24px 48px rgba(0, 0, 0, 0.5)',
  },
  header: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: '20px',
  },
  title: {
    color: '#D4A84B',
    margin: 0,
    fontSize: '1.4rem',
    fontWeight: 600,
  },
  backBtn: {
    padding: '10px 18px',
    backgroundColor: 'rgba(74, 44, 90, 0.5)',
    border: '1px solid #4A2C5A',
    borderRadius: '10px',
    color: '#B8B4C8',
    cursor: 'pointer',
    fontWeight: 500,
  },
  newReportBtn: {
    padding: '8px 16px',
    backgroundColor: '#00d4ff',
    border: 'none',
    borderRadius: '6px',
    color: '#0a0a0f',
    fontWeight: 'bold',
    cursor: 'pointer',
  },
  bugCard: {
    backgroundColor: '#0a0a15',
    padding: '15px',
    borderRadius: '8px',
    cursor: 'pointer',
    border: '1px solid #2a2a4a',
    transition: 'border-color 0.2s',
  },
  closeBtn: {
    marginTop: '20px',
    width: '100%',
    padding: '12px',
    backgroundColor: '#2a2a4a',
    border: 'none',
    borderRadius: '6px',
    color: '#e0e0e0',
    cursor: 'pointer',
  },
  commentInput: {
    width: '100%',
    padding: '12px',
    backgroundColor: '#0a0a15',
    border: '1px solid #2a2a4a',
    borderRadius: '6px',
    color: '#e0e0e0',
    fontSize: '0.9rem',
    minHeight: '60px',
    resize: 'vertical' as const,
    boxSizing: 'border-box' as const,
  },
  addCommentBtn: {
    marginTop: '10px',
    padding: '10px 20px',
    backgroundColor: '#00d4ff',
    border: 'none',
    borderRadius: '6px',
    color: '#0a0a0f',
    fontWeight: 'bold',
    cursor: 'pointer',
  },
};

const getStatusColor = (status: string): string => {
  switch (status) {
    case 'open': return '#f59e0b';
    case 'in_progress': return '#3b82f6';
    case 'resolved': return '#10b981';
    case 'closed': return '#6b7280';
    case 'wont_fix': return '#ef4444';
    default: return '#6b7280';
  }
};

const getPriorityColor = (priority: string): string => {
  switch (priority) {
    case 'critical': return '#ef4444';
    case 'high': return '#f97316';
    case 'medium': return '#f59e0b';
    case 'low': return '#10b981';
    default: return '#6b7280';
  }
};

export function MyBugsModal({
  isOpen,
  onClose,
  bugs,
  token,
  showMessage,
  onOpenNewReport,
}: MyBugsModalProps) {
  const [selectedBug, setSelectedBug] = useState<BugDetails | null>(null);
  const [bugComment, setBugComment] = useState('');

  if (!isOpen) return null;

  const handleViewBugDetails = async (bugId: number) => {
    try {
      const response = await fetch(`/api/v1/bugs/${bugId}`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (response.ok) {
        const data = await response.json();
        setSelectedBug(data);
      }
    } catch {
      showMessage('error', 'Failed to fetch bug details');
    }
  };

  const handleAddComment = async () => {
    if (!bugComment.trim() || !selectedBug) return;

    try {
      const response = await fetch(`/api/v1/bugs/${selectedBug.bug.id}/comments`, {
        method: 'POST',
        headers: {
          Authorization: `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ content: bugComment }),
      });

      if (response.ok) {
        showMessage('success', 'Comment added');
        setBugComment('');
        handleViewBugDetails(selectedBug.bug.id);
      } else {
        const data = await response.json();
        showMessage('error', data.error || 'Failed to add comment');
      }
    } catch {
      showMessage('error', 'Network error');
    }
  };

  const handleClose = () => {
    setSelectedBug(null);
    onClose();
  };

  return (
    <div
      style={styles.overlay}
      onClick={handleClose}
      data-testid="my-bugs-modal-overlay"
    >
      <div
        style={styles.modal}
        onClick={(e) => e.stopPropagation()}
        data-testid="my-bugs-modal-container"
      >
        {selectedBug ? (
          <>
            {/* Bug Details View */}
            <div style={styles.header}>
              <h2 style={styles.title}>🐛 {selectedBug.bug.report_number}</h2>
              <button
                style={styles.backBtn}
                onClick={() => setSelectedBug(null)}
                data-testid="my-bugs-back-btn"
              >
                ← Back to List
              </button>
            </div>

            <div style={{ marginBottom: '20px' }}>
              <h3 style={{ color: '#e0e0e0', margin: '0 0 10px 0' }}>
                {selectedBug.bug.title}
              </h3>
              <div style={{ display: 'flex', gap: '10px', marginBottom: '15px' }}>
                <span
                  style={{
                    padding: '4px 8px',
                    borderRadius: '4px',
                    fontSize: '0.8rem',
                    backgroundColor: getStatusColor(selectedBug.bug.status),
                    color: '#fff',
                  }}
                >
                  {selectedBug.bug.status.replace('_', ' ')}
                </span>
                <span
                  style={{
                    padding: '4px 8px',
                    borderRadius: '4px',
                    fontSize: '0.8rem',
                    backgroundColor: getPriorityColor(selectedBug.bug.priority),
                    color: '#fff',
                  }}
                >
                  {selectedBug.bug.priority}
                </span>
                <span
                  style={{
                    padding: '4px 8px',
                    borderRadius: '4px',
                    fontSize: '0.8rem',
                    backgroundColor: '#2a2a4a',
                    color: '#888',
                  }}
                >
                  {selectedBug.bug.category}
                </span>
              </div>
              <p style={{ color: '#ccc', whiteSpace: 'pre-wrap', margin: '0 0 15px 0' }}>
                {selectedBug.bug.description}
              </p>

              {selectedBug.bug.steps_to_reproduce && (
                <div style={{ marginBottom: '15px' }}>
                  <h4 style={{ color: '#888', margin: '0 0 5px 0', fontSize: '0.9rem' }}>
                    Steps to Reproduce:
                  </h4>
                  <p style={{ color: '#ccc', whiteSpace: 'pre-wrap', margin: 0, fontSize: '0.9rem' }}>
                    {selectedBug.bug.steps_to_reproduce}
                  </p>
                </div>
              )}
            </div>

            {/* Comments Section */}
            <div style={{ borderTop: '1px solid #2a2a4a', paddingTop: '20px' }}>
              <h4 style={{ color: '#e0e0e0', margin: '0 0 15px 0' }}>
                💬 Comments ({selectedBug.comments?.length || 0})
              </h4>

              {selectedBug.comments?.map((comment) => (
                <div
                  key={comment.id}
                  style={{
                    backgroundColor: comment.is_status_change ? '#1a2a1a' : '#0a0a15',
                    padding: '12px',
                    borderRadius: '6px',
                    marginBottom: '10px',
                    borderLeft: comment.is_status_change ? '3px solid #10b981' : '3px solid #2a2a4a',
                  }}
                >
                  <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '5px' }}>
                    <span style={{ color: '#00d4ff', fontWeight: 'bold', fontSize: '0.9rem' }}>
                      {comment.username}
                    </span>
                    <span style={{ color: '#666', fontSize: '0.8rem' }}>
                      {new Date(comment.created_at).toLocaleString()}
                    </span>
                  </div>
                  <p style={{ color: '#ccc', margin: 0, fontSize: '0.9rem', whiteSpace: 'pre-wrap' }}>
                    {comment.content}
                  </p>
                </div>
              ))}

              {/* Add Comment */}
              <div style={{ marginTop: '15px' }}>
                <textarea
                  style={styles.commentInput}
                  placeholder="Add a comment..."
                  value={bugComment}
                  onChange={(e) => setBugComment(e.target.value)}
                  data-testid="my-bugs-comment-input"
                  aria-label="Add comment"
                />
                <button
                  style={styles.addCommentBtn}
                  onClick={handleAddComment}
                  data-testid="my-bugs-add-comment-btn"
                >
                  Add Comment
                </button>
              </div>
            </div>
          </>
        ) : (
          <>
            {/* Bug List View */}
            <div style={styles.header}>
              <h2 style={{ color: '#00d4ff', margin: 0 }}>🐛 My Bug Reports</h2>
              <button
                style={styles.newReportBtn}
                onClick={() => {
                  onClose();
                  onOpenNewReport();
                }}
                data-testid="my-bugs-new-report-btn"
              >
                + New Report
              </button>
            </div>

            {bugs.length === 0 ? (
              <p style={{ color: '#888', textAlign: 'center', padding: '40px' }}>
                No bug reports yet. Click "New Report" to submit one.
              </p>
            ) : (
              <div style={{ display: 'flex', flexDirection: 'column', gap: '10px' }}>
                {bugs.map((bug) => (
                  <div
                    key={bug.id}
                    style={styles.bugCard}
                    onClick={() => handleViewBugDetails(bug.id)}
                    onMouseEnter={(e) => (e.currentTarget.style.borderColor = '#00d4ff')}
                    onMouseLeave={(e) => (e.currentTarget.style.borderColor = '#2a2a4a')}
                    data-testid={`bug-card-${bug.id}`}
                  >
                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: '8px' }}>
                      <span style={{ color: '#00d4ff', fontSize: '0.85rem' }}>
                        {bug.report_number}
                      </span>
                      <div style={{ display: 'flex', gap: '5px' }}>
                        <span
                          style={{
                            padding: '2px 6px',
                            borderRadius: '4px',
                            fontSize: '0.75rem',
                            backgroundColor: getStatusColor(bug.status),
                            color: '#fff',
                          }}
                        >
                          {bug.status.replace('_', ' ')}
                        </span>
                        <span
                          style={{
                            padding: '2px 6px',
                            borderRadius: '4px',
                            fontSize: '0.75rem',
                            backgroundColor: getPriorityColor(bug.priority),
                            color: '#fff',
                          }}
                        >
                          {bug.priority}
                        </span>
                      </div>
                    </div>
                    <h4 style={{ color: '#e0e0e0', margin: '0 0 5px 0', fontSize: '1rem' }}>
                      {bug.title}
                    </h4>
                    <div style={{ display: 'flex', gap: '15px', color: '#666', fontSize: '0.8rem' }}>
                      <span>📎 {bug.attachment_count || 0}</span>
                      <span>💬 {bug.comment_count || 0}</span>
                      <span>{new Date(bug.created_at).toLocaleDateString()}</span>
                    </div>
                  </div>
                ))}
              </div>
            )}

            <button
              style={styles.closeBtn}
              onClick={handleClose}
              data-testid="my-bugs-close-btn"
            >
              Close
            </button>
          </>
        )}
      </div>
    </div>
  );
}

export default MyBugsModal;
