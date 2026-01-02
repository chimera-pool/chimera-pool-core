import React, { useState, useEffect } from 'react';

interface AdminBugsTabProps {
  token: string;
  isActive: boolean;
  showMessage: (type: 'success' | 'error', text: string) => void;
}

interface BugFilter {
  status: string;
  priority: string;
  category: string;
}

const getBugStatusColor = (status: string) => {
  switch (status) {
    case 'open': return '#f59e0b';
    case 'in_progress': return '#3b82f6';
    case 'resolved': return '#10b981';
    case 'closed': return '#6b7280';
    case 'wont_fix': return '#ef4444';
    default: return '#6b7280';
  }
};

const getBugPriorityColor = (priority: string) => {
  switch (priority) {
    case 'critical': return '#ef4444';
    case 'high': return '#f97316';
    case 'medium': return '#f59e0b';
    case 'low': return '#10b981';
    default: return '#6b7280';
  }
};

const styles: { [key: string]: React.CSSProperties } = {
  container: { background: 'linear-gradient(135deg, #1a1a2e 0%, #0f0f1a 100%)', borderRadius: '12px', padding: '24px', border: '1px solid #2a2a4a' },
  loading: { textAlign: 'center', padding: '60px', color: '#00d4ff' },
};

export function AdminBugsTab({ token, isActive, showMessage }: AdminBugsTabProps) {
  const [adminBugs, setAdminBugs] = useState<any[]>([]);
  const [bugsLoading, setBugsLoading] = useState(false);
  const [bugFilter, setBugFilter] = useState<BugFilter>({ status: '', priority: '', category: '' });
  const [selectedAdminBug, setSelectedAdminBug] = useState<any>(null);
  const [adminBugComment, setAdminBugComment] = useState('');
  const [isInternalComment, setIsInternalComment] = useState(false);

  useEffect(() => {
    if (isActive) {
      fetchAdminBugs();
    }
  }, [isActive, bugFilter]);

  const fetchAdminBugs = async () => {
    setBugsLoading(true);
    try {
      const params = new URLSearchParams();
      if (bugFilter.status) params.set('status', bugFilter.status);
      if (bugFilter.priority) params.set('priority', bugFilter.priority);
      if (bugFilter.category) params.set('category', bugFilter.category);
      
      const response = await fetch(`/api/v1/admin/bugs?${params}`, {
        headers: { 'Authorization': `Bearer ${token}` }
      });
      if (response.ok) {
        const data = await response.json();
        setAdminBugs(data.bugs || []);
      }
    } catch (error) {
      console.error('Failed to fetch bugs:', error);
    } finally {
      setBugsLoading(false);
    }
  };

  const fetchAdminBugDetails = async (bugId: number) => {
    try {
      const response = await fetch(`/api/v1/admin/bugs/${bugId}`, {
        headers: { 'Authorization': `Bearer ${token}` }
      });
      if (response.ok) {
        const data = await response.json();
        setSelectedAdminBug(data);
      }
    } catch (error) {
      console.error('Failed to fetch bug details:', error);
    }
  };

  const handleUpdateBugStatus = async (bugId: number, newStatus: string) => {
    try {
      const response = await fetch(`/api/v1/admin/bugs/${bugId}/status`, {
        method: 'PUT',
        headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
        body: JSON.stringify({ status: newStatus })
      });
      if (response.ok) {
        showMessage('success', `Status updated to ${newStatus}`);
        fetchAdminBugs();
        if (selectedAdminBug) fetchAdminBugDetails(bugId);
      } else {
        const data = await response.json();
        showMessage('error', data.error || 'Failed to update status');
      }
    } catch (error) {
      showMessage('error', 'Failed to update status');
    }
  };

  const handleUpdateBugPriority = async (bugId: number, newPriority: string) => {
    try {
      const response = await fetch(`/api/v1/admin/bugs/${bugId}/priority`, {
        method: 'PUT',
        headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
        body: JSON.stringify({ priority: newPriority })
      });
      if (response.ok) {
        showMessage('success', `Priority updated to ${newPriority}`);
        fetchAdminBugs();
        if (selectedAdminBug) fetchAdminBugDetails(bugId);
      } else {
        const data = await response.json();
        showMessage('error', data.error || 'Failed to update priority');
      }
    } catch (error) {
      showMessage('error', 'Failed to update priority');
    }
  };

  const handleAddAdminBugComment = async () => {
    if (!adminBugComment.trim() || !selectedAdminBug) return;
    try {
      const response = await fetch(`/api/v1/admin/bugs/${selectedAdminBug.bug.id}/comments`, {
        method: 'POST',
        headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
        body: JSON.stringify({ content: adminBugComment, is_internal: isInternalComment })
      });
      if (response.ok) {
        showMessage('success', isInternalComment ? 'Internal note added' : 'Comment added');
        setAdminBugComment('');
        setIsInternalComment(false);
        fetchAdminBugDetails(selectedAdminBug.bug.id);
      } else {
        const data = await response.json();
        showMessage('error', data.error || 'Failed to add comment');
      }
    } catch (error) {
      showMessage('error', 'Failed to add comment');
    }
  };

  const handleDeleteBug = async (bugId: number) => {
    if (!window.confirm('Are you sure you want to delete this bug report?')) return;
    try {
      const response = await fetch(`/api/v1/admin/bugs/${bugId}`, {
        method: 'DELETE',
        headers: { 'Authorization': `Bearer ${token}` }
      });
      if (response.ok) {
        showMessage('success', 'Bug report deleted');
        setSelectedAdminBug(null);
        fetchAdminBugs();
      } else {
        const data = await response.json();
        showMessage('error', data.error || 'Failed to delete bug');
      }
    } catch (error) {
      showMessage('error', 'Failed to delete bug');
    }
  };

  if (!isActive) return null;

  return (
    <div style={styles.container}>
      {selectedAdminBug ? (
        <>
          {/* Bug Detail View */}
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '20px' }}>
            <div>
              <h3 style={{ color: '#00d4ff', margin: 0, display: 'flex', alignItems: 'center', gap: '10px' }}>
                🐛 {selectedAdminBug.bug.report_number}
                <span style={{ padding: '4px 10px', borderRadius: '4px', fontSize: '0.8rem', backgroundColor: getBugStatusColor(selectedAdminBug.bug.status), color: '#fff' }}>
                  {selectedAdminBug.bug.status.replace('_', ' ')}
                </span>
                <span style={{ padding: '4px 10px', borderRadius: '4px', fontSize: '0.8rem', backgroundColor: getBugPriorityColor(selectedAdminBug.bug.priority), color: '#fff' }}>
                  {selectedAdminBug.bug.priority}
                </span>
              </h3>
              <p style={{ color: '#888', margin: '5px 0 0', fontSize: '0.85rem' }}>
                Reported by <strong style={{ color: '#00d4ff' }}>{selectedAdminBug.bug.username}</strong> on {new Date(selectedAdminBug.bug.created_at).toLocaleString()}
              </p>
            </div>
            <button 
              style={{ padding: '8px 16px', backgroundColor: '#2a2a4a', border: 'none', borderRadius: '6px', color: '#e0e0e0', cursor: 'pointer' }}
              onClick={() => setSelectedAdminBug(null)}
            >
              ← Back to List
            </button>
          </div>

          <div style={{ display: 'grid', gridTemplateColumns: '2fr 1fr', gap: '20px' }} className="bug-detail-grid">
            {/* Left Column - Bug Details */}
            <div>
              <div style={{ backgroundColor: '#0a0a15', borderRadius: '8px', padding: '20px', marginBottom: '15px' }}>
                <h4 style={{ color: '#e0e0e0', margin: '0 0 10px' }}>{selectedAdminBug.bug.title}</h4>
                <p style={{ color: '#ccc', whiteSpace: 'pre-wrap', margin: 0 }}>{selectedAdminBug.bug.description}</p>
              </div>

              {selectedAdminBug.bug.steps_to_reproduce && (
                <div style={{ backgroundColor: '#0a0a15', borderRadius: '8px', padding: '15px', marginBottom: '15px' }}>
                  <h5 style={{ color: '#888', margin: '0 0 8px', fontSize: '0.9rem' }}>Steps to Reproduce:</h5>
                  <p style={{ color: '#ccc', whiteSpace: 'pre-wrap', margin: 0, fontSize: '0.9rem' }}>{selectedAdminBug.bug.steps_to_reproduce}</p>
                </div>
              )}

              {(selectedAdminBug.bug.expected_behavior || selectedAdminBug.bug.actual_behavior) && (
                <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '10px', marginBottom: '15px' }}>
                  {selectedAdminBug.bug.expected_behavior && (
                    <div style={{ backgroundColor: '#0a1a0a', borderRadius: '8px', padding: '15px', border: '1px solid #10b981' }}>
                      <h5 style={{ color: '#10b981', margin: '0 0 8px', fontSize: '0.85rem' }}>Expected:</h5>
                      <p style={{ color: '#ccc', margin: 0, fontSize: '0.85rem' }}>{selectedAdminBug.bug.expected_behavior}</p>
                    </div>
                  )}
                  {selectedAdminBug.bug.actual_behavior && (
                    <div style={{ backgroundColor: '#1a0a0a', borderRadius: '8px', padding: '15px', border: '1px solid #ef4444' }}>
                      <h5 style={{ color: '#ef4444', margin: '0 0 8px', fontSize: '0.85rem' }}>Actual:</h5>
                      <p style={{ color: '#ccc', margin: 0, fontSize: '0.85rem' }}>{selectedAdminBug.bug.actual_behavior}</p>
                    </div>
                  )}
                </div>
              )}

              {/* Environment Info */}
              <div style={{ backgroundColor: '#0a0a15', borderRadius: '8px', padding: '15px', marginBottom: '15px' }}>
                <h5 style={{ color: '#888', margin: '0 0 10px', fontSize: '0.85rem' }}>Environment:</h5>
                <div style={{ display: 'grid', gridTemplateColumns: 'auto 1fr', gap: '5px 15px', fontSize: '0.8rem' }}>
                  <span style={{ color: '#666' }}>Category:</span>
                  <span style={{ color: '#ccc' }}>{selectedAdminBug.bug.category}</span>
                  {selectedAdminBug.bug.page_url && (
                    <>
                      <span style={{ color: '#666' }}>Page URL:</span>
                      <span style={{ color: '#00d4ff', wordBreak: 'break-all' }}>{selectedAdminBug.bug.page_url}</span>
                    </>
                  )}
                  {selectedAdminBug.bug.browser_info && (
                    <>
                      <span style={{ color: '#666' }}>Browser:</span>
                      <span style={{ color: '#ccc', fontSize: '0.75rem' }}>{selectedAdminBug.bug.browser_info}</span>
                    </>
                  )}
                  {selectedAdminBug.bug.os_info && (
                    <>
                      <span style={{ color: '#666' }}>OS:</span>
                      <span style={{ color: '#ccc' }}>{selectedAdminBug.bug.os_info}</span>
                    </>
                  )}
                </div>
              </div>

              {/* Comments Section */}
              <div style={{ backgroundColor: '#0a0a15', borderRadius: '8px', padding: '20px' }}>
                <h4 style={{ color: '#e0e0e0', margin: '0 0 15px' }}>💬 Comments ({selectedAdminBug.comments?.length || 0})</h4>
                
                {selectedAdminBug.comments?.map((comment: any) => (
                  <div 
                    key={comment.id} 
                    style={{ 
                      backgroundColor: comment.is_internal ? '#1a1a0a' : (comment.is_status_change ? '#0a1a0a' : '#1a1a2e'), 
                      padding: '12px', 
                      borderRadius: '6px', 
                      marginBottom: '10px', 
                      borderLeft: comment.is_internal ? '3px solid #f59e0b' : (comment.is_status_change ? '3px solid #10b981' : '3px solid #2a2a4a')
                    }}
                  >
                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '5px' }}>
                      <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                        <span style={{ color: '#00d4ff', fontWeight: 'bold', fontSize: '0.9rem' }}>{comment.username}</span>
                        {comment.is_internal && (
                          <span style={{ backgroundColor: '#f59e0b', color: '#000', padding: '1px 6px', borderRadius: '4px', fontSize: '0.7rem', fontWeight: 'bold' }}>INTERNAL</span>
                        )}
                      </div>
                      <span style={{ color: '#666', fontSize: '0.8rem' }}>{new Date(comment.created_at).toLocaleString()}</span>
                    </div>
                    <p style={{ color: '#ccc', margin: 0, fontSize: '0.9rem', whiteSpace: 'pre-wrap' }}>{comment.content}</p>
                  </div>
                ))}

                {/* Add Comment Form */}
                <div style={{ marginTop: '15px', borderTop: '1px solid #2a2a4a', paddingTop: '15px' }}>
                  <textarea 
                    style={{ width: '100%', padding: '12px', backgroundColor: '#1a1a2e', border: '1px solid #2a2a4a', borderRadius: '6px', color: '#e0e0e0', fontSize: '0.9rem', minHeight: '80px', resize: 'vertical', boxSizing: 'border-box' }}
                    placeholder="Add a comment..."
                    value={adminBugComment}
                    onChange={e => setAdminBugComment(e.target.value)}
                  />
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginTop: '10px' }}>
                    <label style={{ display: 'flex', alignItems: 'center', gap: '8px', color: '#f59e0b', cursor: 'pointer' }}>
                      <input 
                        type="checkbox" 
                        checked={isInternalComment} 
                        onChange={e => setIsInternalComment(e.target.checked)}
                        style={{ accentColor: '#f59e0b' }}
                      />
                      🔒 Internal note (not visible to user)
                    </label>
                    <button 
                      style={{ padding: '10px 20px', backgroundColor: '#00d4ff', border: 'none', borderRadius: '6px', color: '#0a0a0f', fontWeight: 'bold', cursor: 'pointer' }}
                      onClick={handleAddAdminBugComment}
                    >
                      Add Comment
                    </button>
                  </div>
                </div>
              </div>
            </div>

            {/* Right Column - Actions */}
            <div>
              <div style={{ backgroundColor: '#0a0a15', borderRadius: '8px', padding: '20px', marginBottom: '15px' }}>
                <h4 style={{ color: '#e0e0e0', margin: '0 0 15px' }}>⚡ Quick Actions</h4>
                
                <div style={{ marginBottom: '15px' }}>
                  <label style={{ display: 'block', color: '#888', marginBottom: '5px', fontSize: '0.85rem' }}>Status</label>
                  <select 
                    style={{ width: '100%', padding: '10px', backgroundColor: '#1a1a2e', border: '1px solid #2a2a4a', borderRadius: '6px', color: '#e0e0e0' }}
                    value={selectedAdminBug.bug.status}
                    onChange={e => handleUpdateBugStatus(selectedAdminBug.bug.id, e.target.value)}
                  >
                    <option value="open">🟡 Open</option>
                    <option value="in_progress">🔵 In Progress</option>
                    <option value="resolved">🟢 Resolved</option>
                    <option value="closed">⚫ Closed</option>
                    <option value="wont_fix">🔴 Won't Fix</option>
                  </select>
                </div>

                <div style={{ marginBottom: '15px' }}>
                  <label style={{ display: 'block', color: '#888', marginBottom: '5px', fontSize: '0.85rem' }}>Priority</label>
                  <select 
                    style={{ width: '100%', padding: '10px', backgroundColor: '#1a1a2e', border: '1px solid #2a2a4a', borderRadius: '6px', color: '#e0e0e0' }}
                    value={selectedAdminBug.bug.priority}
                    onChange={e => handleUpdateBugPriority(selectedAdminBug.bug.id, e.target.value)}
                  >
                    <option value="low">🟢 Low</option>
                    <option value="medium">🟡 Medium</option>
                    <option value="high">🟠 High</option>
                    <option value="critical">🔴 Critical</option>
                  </select>
                </div>

                <button 
                  style={{ width: '100%', padding: '10px', backgroundColor: '#4d1a1a', border: 'none', borderRadius: '6px', color: '#ef4444', cursor: 'pointer', marginTop: '10px' }}
                  onClick={() => handleDeleteBug(selectedAdminBug.bug.id)}
                >
                  🗑️ Delete Bug Report
                </button>
              </div>

              {/* Attachments */}
              {selectedAdminBug.attachments?.length > 0 && (
                <div style={{ backgroundColor: '#0a0a15', borderRadius: '8px', padding: '20px' }}>
                  <h4 style={{ color: '#e0e0e0', margin: '0 0 15px' }}>📎 Attachments ({selectedAdminBug.attachments.length})</h4>
                  {selectedAdminBug.attachments.map((att: any) => (
                    <div key={att.id} style={{ backgroundColor: '#1a1a2e', padding: '10px', borderRadius: '6px', marginBottom: '8px', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                      <div>
                        <span style={{ color: '#00d4ff' }}>{att.is_screenshot ? '📸' : '📄'} {att.original_filename}</span>
                        <span style={{ color: '#666', fontSize: '0.8rem', marginLeft: '10px' }}>({(att.file_size / 1024).toFixed(1)} KB)</span>
                      </div>
                      <button
                        onClick={() => {
                          fetch(`/api/v1/bugs/${selectedAdminBug.bug.id}/attachments/${att.id}`, {
                            headers: { 'Authorization': `Bearer ${token}` }
                          })
                          .then(res => res.blob())
                          .then(blob => {
                            const url = window.URL.createObjectURL(blob);
                            const a = document.createElement('a');
                            a.href = url;
                            a.download = att.original_filename;
                            document.body.appendChild(a);
                            a.click();
                            window.URL.revokeObjectURL(url);
                            a.remove();
                          })
                          .catch(() => showMessage('error', 'Failed to download attachment'));
                        }}
                        style={{ padding: '6px 12px', backgroundColor: '#00d4ff', border: 'none', borderRadius: '4px', color: '#0a0a0f', fontWeight: 'bold', cursor: 'pointer', fontSize: '0.8rem' }}
                      >
                        ⬇️ Download
                      </button>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>
        </>
      ) : (
        <>
          {/* Bug List View */}
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '20px' }}>
            <div>
              <h3 style={{ color: '#00d4ff', margin: 0 }}>🐛 Bug Reports</h3>
              <p style={{ color: '#888', margin: '5px 0 0', fontSize: '0.9rem' }}>
                {adminBugs.length} total reports • {adminBugs.filter(b => b.status === 'open').length} open
              </p>
            </div>
            <button 
              style={{ padding: '8px 16px', backgroundColor: '#00d4ff', border: 'none', borderRadius: '6px', color: '#0a0a0f', fontWeight: 'bold', cursor: 'pointer' }}
              onClick={fetchAdminBugs}
            >
              🔄 Refresh
            </button>
          </div>

          {/* Filters */}
          <div style={{ display: 'flex', gap: '15px', marginBottom: '20px', flexWrap: 'wrap' }} className="bug-filters">
            <select 
              style={{ padding: '8px 12px', backgroundColor: '#1a1a2e', border: '1px solid #2a2a4a', borderRadius: '6px', color: '#e0e0e0' }}
              value={bugFilter.status}
              onChange={e => setBugFilter({...bugFilter, status: e.target.value})}
            >
              <option value="">All Status</option>
              <option value="open">🟡 Open</option>
              <option value="in_progress">🔵 In Progress</option>
              <option value="resolved">🟢 Resolved</option>
              <option value="closed">⚫ Closed</option>
              <option value="wont_fix">🔴 Won't Fix</option>
            </select>
            <select 
              style={{ padding: '8px 12px', backgroundColor: '#1a1a2e', border: '1px solid #2a2a4a', borderRadius: '6px', color: '#e0e0e0' }}
              value={bugFilter.priority}
              onChange={e => setBugFilter({...bugFilter, priority: e.target.value})}
            >
              <option value="">All Priority</option>
              <option value="critical">🔴 Critical</option>
              <option value="high">🟠 High</option>
              <option value="medium">🟡 Medium</option>
              <option value="low">🟢 Low</option>
            </select>
            <select 
              style={{ padding: '8px 12px', backgroundColor: '#1a1a2e', border: '1px solid #2a2a4a', borderRadius: '6px', color: '#e0e0e0' }}
              value={bugFilter.category}
              onChange={e => setBugFilter({...bugFilter, category: e.target.value})}
            >
              <option value="">All Categories</option>
              <option value="ui">UI/Visual</option>
              <option value="performance">Performance</option>
              <option value="crash">Crash/Error</option>
              <option value="security">Security</option>
              <option value="feature_request">Feature Request</option>
              <option value="other">Other</option>
            </select>
          </div>

          {/* Bug List */}
          {bugsLoading ? (
            <div style={styles.loading}>Loading bug reports...</div>
          ) : adminBugs.length === 0 ? (
            <div style={{ textAlign: 'center', padding: '40px', color: '#666' }}>
              <p style={{ fontSize: '1.2rem', margin: '0 0 10px' }}>🎉 No bug reports found</p>
              <p style={{ margin: 0 }}>Either no bugs have been reported or all match your current filters.</p>
            </div>
          ) : (
            <div style={{ display: 'flex', flexDirection: 'column', gap: '10px' }}>
              {adminBugs.map((bug: any) => (
                <div 
                  key={bug.id} 
                  style={{ 
                    backgroundColor: '#0a0a15', 
                    padding: '15px 20px', 
                    borderRadius: '8px', 
                    cursor: 'pointer', 
                    border: '1px solid #2a2a4a',
                    borderLeft: `4px solid ${getBugPriorityColor(bug.priority)}`,
                    transition: 'all 0.2s'
                  }}
                  onClick={() => fetchAdminBugDetails(bug.id)}
                  onMouseEnter={e => { e.currentTarget.style.borderColor = '#00d4ff'; e.currentTarget.style.backgroundColor = '#0f0f1a'; }}
                  onMouseLeave={e => { e.currentTarget.style.borderColor = '#2a2a4a'; e.currentTarget.style.backgroundColor = '#0a0a15'; }}
                >
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: '8px' }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
                      <span style={{ color: '#00d4ff', fontSize: '0.85rem', fontFamily: 'monospace' }}>{bug.report_number}</span>
                      <span style={{ padding: '2px 8px', borderRadius: '4px', fontSize: '0.75rem', backgroundColor: getBugStatusColor(bug.status), color: '#fff' }}>
                        {bug.status.replace('_', ' ')}
                      </span>
                      <span style={{ padding: '2px 8px', borderRadius: '4px', fontSize: '0.75rem', backgroundColor: getBugPriorityColor(bug.priority), color: '#fff' }}>
                        {bug.priority}
                      </span>
                      <span style={{ padding: '2px 8px', borderRadius: '4px', fontSize: '0.75rem', backgroundColor: '#2a2a4a', color: '#888' }}>
                        {bug.category}
                      </span>
                    </div>
                    <span style={{ color: '#666', fontSize: '0.8rem' }}>{new Date(bug.created_at).toLocaleDateString()}</span>
                  </div>
                  <h4 style={{ color: '#e0e0e0', margin: '0 0 8px', fontSize: '1rem' }}>{bug.title}</h4>
                  <div style={{ display: 'flex', gap: '20px', color: '#666', fontSize: '0.8rem' }}>
                    <span>👤 {bug.username}</span>
                    <span>📎 {bug.attachment_count || 0}</span>
                    <span>💬 {bug.comment_count || 0}</span>
                  </div>
                </div>
              ))}
            </div>
          )}
        </>
      )}
    </div>
  );
}

export default AdminBugsTab;
