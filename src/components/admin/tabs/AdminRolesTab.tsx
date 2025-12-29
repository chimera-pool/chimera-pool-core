import React, { useState, useEffect } from 'react';

interface AdminRolesTabProps {
  token: string;
  isActive: boolean;
  showMessage: (type: 'success' | 'error', text: string) => void;
  onNavigateToUsers?: () => void;
}

interface UserRole {
  id: number;
  username: string;
  email: string;
  role: string;
  created_at: string;
}

const styles: { [key: string]: React.CSSProperties } = {
  container: { padding: '20px' },
  header: { marginBottom: '25px' },
  title: { color: '#D4A84B', marginTop: 0, marginBottom: '10px', fontWeight: 600 },
  desc: { color: '#B8B4C8', margin: 0 },
  loading: { padding: '40px', textAlign: 'center', color: '#D4A84B' },
  actionBtn: { background: 'none', border: 'none', cursor: 'pointer', fontSize: '1.1rem', padding: '4px 8px' },
  formGroup: { marginBottom: '16px' },
  label: { display: 'block', color: '#B8B4C8', marginBottom: '6px', fontSize: '0.9rem', fontWeight: 500 },
  select: { width: '100%', padding: '12px 14px', backgroundColor: 'rgba(13, 8, 17, 0.8)', border: '1px solid rgba(74, 44, 90, 0.5)', borderRadius: '10px', color: '#F0EDF4' },
  cancelBtn: { flex: 1, padding: '12px', backgroundColor: 'transparent', border: '1px solid rgba(74, 44, 90, 0.5)', borderRadius: '10px', color: '#B8B4C8', cursor: 'pointer' },
  saveBtn: { flex: 1, padding: '12px', background: 'linear-gradient(135deg, #D4A84B 0%, #B8923A 100%)', border: 'none', borderRadius: '10px', color: '#1A0F1E', fontWeight: 600, cursor: 'pointer' },
  editModal: { position: 'fixed' as const, top: '50%', left: '50%', transform: 'translate(-50%, -50%)', background: 'linear-gradient(180deg, #2D1F3D 0%, #1A0F1E 100%)', padding: '28px', borderRadius: '16px', border: '1px solid rgba(212, 168, 75, 0.3)', minWidth: '400px', zIndex: 3000, boxShadow: '0 24px 48px rgba(0, 0, 0, 0.5)' },
  editTitle: { color: '#D4A84B', marginTop: 0, fontWeight: 600 },
  editActions: { display: 'flex', gap: '12px', marginTop: '24px' },
};

export function AdminRolesTab({ token, isActive, showMessage, onNavigateToUsers }: AdminRolesTabProps) {
  const [moderators, setModerators] = useState<UserRole[]>([]);
  const [admins, setAdmins] = useState<UserRole[]>([]);
  const [rolesLoading, setRolesLoading] = useState(false);
  const [roleChangeUser, setRoleChangeUser] = useState<UserRole | null>(null);
  const [newRole, setNewRole] = useState('');

  useEffect(() => {
    if (isActive) {
      fetchRoles();
    }
  }, [isActive]);

  const fetchRoles = async () => {
    setRolesLoading(true);
    try {
      const response = await fetch('/api/v1/admin/roles', {
        headers: { 'Authorization': `Bearer ${token}` }
      });
      if (response.ok) {
        const data = await response.json();
        setModerators(data.moderators || []);
        setAdmins(data.admins || []);
      }
    } catch (error) {
      console.error('Failed to fetch roles:', error);
    } finally {
      setRolesLoading(false);
    }
  };

  const handleChangeRole = async () => {
    if (!roleChangeUser || !newRole) return;
    try {
      const response = await fetch(`/api/v1/admin/users/${roleChangeUser.id}/role`, {
        method: 'PUT',
        headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
        body: JSON.stringify({ role: newRole })
      });
      if (response.ok) {
        showMessage('success', `Changed ${roleChangeUser.username}'s role to ${newRole}`);
        setRoleChangeUser(null);
        setNewRole('');
        fetchRoles();
      } else {
        const data = await response.json();
        showMessage('error', data.error || 'Failed to change role');
      }
    } catch (error) {
      showMessage('error', 'Network error');
    }
  };

  if (!isActive) return null;

  return (
    <div style={styles.container}>
      <div style={styles.header}>
        <h3 style={styles.title}>👑 Role Management</h3>
        <p style={styles.desc}>
          Manage user roles and permissions. Promote users to moderators or admins.
        </p>
      </div>

      {rolesLoading ? (
        <div style={styles.loading}>Loading roles...</div>
      ) : (
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(400px, 1fr))', gap: '20px' }}>
          {/* Admins Section */}
          <div style={{ backgroundColor: '#0a0a15', borderRadius: '8px', border: '1px solid #9b59b6', overflow: 'hidden' }}>
            <div style={{ padding: '15px 20px', backgroundColor: '#1a1a2e', borderBottom: '1px solid #9b59b6' }}>
              <h4 style={{ margin: 0, color: '#9b59b6', display: 'flex', alignItems: 'center', gap: '8px' }}>
                👑 Administrators ({admins.length})
              </h4>
              <p style={{ margin: '5px 0 0', color: '#888', fontSize: '0.85rem' }}>
                Full access to all pool settings and user management
              </p>
            </div>
            <div style={{ padding: '15px 20px', maxHeight: '400px', overflowY: 'auto' }}>
              {admins.length === 0 ? (
                <p style={{ color: '#666', fontStyle: 'italic', margin: 0 }}>No administrators assigned</p>
              ) : (
                <table style={{ width: '100%', borderCollapse: 'collapse' }}>
                  <thead>
                    <tr style={{ borderBottom: '1px solid #2a2a4a' }}>
                      <th style={{ padding: '8px', textAlign: 'left', color: '#D4A84B', fontSize: '0.8rem' }}>ID</th>
                      <th style={{ padding: '8px', textAlign: 'left', color: '#9b59b6', fontSize: '0.8rem' }}>User</th>
                      <th style={{ padding: '8px', textAlign: 'left', color: '#888', fontSize: '0.8rem' }}>Role</th>
                      <th style={{ padding: '8px', textAlign: 'left', color: '#888', fontSize: '0.8rem' }}>Joined</th>
                      <th style={{ padding: '8px', textAlign: 'center', color: '#888', fontSize: '0.8rem' }}>Actions</th>
                    </tr>
                  </thead>
                  <tbody>
                    {admins.map((admin) => (
                      <tr key={admin.id} style={{ borderBottom: '1px solid #1a1a2e' }}>
                        <td style={{ padding: '10px 8px', color: '#D4A84B', fontFamily: 'monospace', fontWeight: 600 }}>#{admin.id}</td>
                        <td style={{ padding: '10px 8px' }}>
                          <div style={{ color: '#9b59b6', fontWeight: 'bold' }}>{admin.username}</div>
                          <div style={{ color: '#666', fontSize: '0.75rem' }}>{admin.email}</div>
                        </td>
                        <td style={{ padding: '10px 8px' }}>
                          <span style={{ backgroundColor: admin.role === 'super_admin' ? '#4d1a4d' : '#2a1a4a', color: admin.role === 'super_admin' ? '#d946ef' : '#9b59b6', padding: '3px 8px', borderRadius: '4px', fontSize: '0.75rem' }}>
                            {admin.role === 'super_admin' ? '⭐ Super Admin' : '👑 Admin'}
                          </span>
                        </td>
                        <td style={{ padding: '10px 8px', color: '#888', fontSize: '0.8rem' }}>
                          {admin.created_at ? new Date(admin.created_at).toLocaleDateString() : 'N/A'}
                        </td>
                        <td style={{ padding: '10px 8px', textAlign: 'center' }}>
                          <button 
                            style={{ ...styles.actionBtn, color: '#fbbf24', padding: '4px 8px' }}
                            onClick={() => { setRoleChangeUser(admin); setNewRole('user'); }}
                            title="Change Role"
                          >
                            ✏️
                          </button>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              )}
            </div>
          </div>

          {/* Moderators Section */}
          <div style={{ backgroundColor: '#0a0a15', borderRadius: '8px', border: '1px solid #00d4ff', overflow: 'hidden' }}>
            <div style={{ padding: '15px 20px', backgroundColor: '#1a1a2e', borderBottom: '1px solid #00d4ff' }}>
              <h4 style={{ margin: 0, color: '#00d4ff', display: 'flex', alignItems: 'center', gap: '8px' }}>
                🛡️ Moderators ({moderators.length})
              </h4>
              <p style={{ margin: '5px 0 0', color: '#888', fontSize: '0.85rem' }}>
                Can manage community channels, mute users, and view reports
              </p>
            </div>
            <div style={{ padding: '15px 20px', maxHeight: '400px', overflowY: 'auto' }}>
              {moderators.length === 0 ? (
                <p style={{ color: '#666', fontStyle: 'italic', margin: 0 }}>No moderators assigned</p>
              ) : (
                <table style={{ width: '100%', borderCollapse: 'collapse' }}>
                  <thead>
                    <tr style={{ borderBottom: '1px solid #2a2a4a' }}>
                      <th style={{ padding: '8px', textAlign: 'left', color: '#D4A84B', fontSize: '0.8rem' }}>ID</th>
                      <th style={{ padding: '8px', textAlign: 'left', color: '#00d4ff', fontSize: '0.8rem' }}>User</th>
                      <th style={{ padding: '8px', textAlign: 'left', color: '#888', fontSize: '0.8rem' }}>Joined</th>
                      <th style={{ padding: '8px', textAlign: 'center', color: '#888', fontSize: '0.8rem' }}>Actions</th>
                    </tr>
                  </thead>
                  <tbody>
                    {moderators.map((mod) => (
                      <tr key={mod.id} style={{ borderBottom: '1px solid #1a1a2e' }}>
                        <td style={{ padding: '10px 8px', color: '#D4A84B', fontFamily: 'monospace', fontWeight: 600 }}>#{mod.id}</td>
                        <td style={{ padding: '10px 8px' }}>
                          <div style={{ color: '#00d4ff', fontWeight: 'bold' }}>{mod.username}</div>
                          <div style={{ color: '#666', fontSize: '0.75rem' }}>{mod.email}</div>
                        </td>
                        <td style={{ padding: '10px 8px', color: '#888', fontSize: '0.8rem' }}>
                          {mod.created_at ? new Date(mod.created_at).toLocaleDateString() : 'N/A'}
                        </td>
                        <td style={{ padding: '10px 8px', textAlign: 'center' }}>
                          <button 
                            style={{ ...styles.actionBtn, color: '#fbbf24', padding: '4px 8px' }}
                            onClick={() => { setRoleChangeUser(mod); setNewRole('user'); }}
                            title="Change Role"
                          >
                            ✏️
                          </button>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              )}
            </div>
          </div>
        </div>
      )}

      {/* Promote User Section */}
      <div style={{ marginTop: '25px', backgroundColor: '#0a0a15', borderRadius: '8px', border: '1px solid #4ade80', padding: '20px' }}>
        <h4 style={{ margin: '0 0 15px', color: '#4ade80' }}>➕ Promote a User</h4>
        <p style={{ color: '#888', fontSize: '0.9rem', marginBottom: '15px' }}>
          Search for a user from the User Management tab, then return here to promote them.
        </p>
        <div style={{ display: 'flex', gap: '10px', flexWrap: 'wrap' }}>
          <button 
            style={{ padding: '12px 24px', background: 'linear-gradient(135deg, #00d4ff 0%, #0099cc 100%)', border: 'none', borderRadius: '8px', color: '#0a0a0f', fontWeight: 600, cursor: 'pointer' }}
            onClick={onNavigateToUsers}
          >
            👥 Go to User Management
          </button>
        </div>
      </div>

      {/* Role Hierarchy Info */}
      <div style={{ marginTop: '25px', backgroundColor: '#1a1a2e', borderRadius: '8px', padding: '20px' }}>
        <h4 style={{ margin: '0 0 15px', color: '#fbbf24' }}>📋 Role Hierarchy</h4>
        <div style={{ display: 'grid', gap: '12px' }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
            <span style={{ backgroundColor: '#4d1a4d', color: '#d946ef', padding: '4px 10px', borderRadius: '4px', fontSize: '0.85rem', minWidth: '120px', textAlign: 'center' }}>⭐ Super Admin</span>
            <span style={{ color: '#888', fontSize: '0.9rem' }}>Full access, can promote/demote admins</span>
          </div>
          <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
            <span style={{ backgroundColor: '#2a1a4a', color: '#9b59b6', padding: '4px 10px', borderRadius: '4px', fontSize: '0.85rem', minWidth: '120px', textAlign: 'center' }}>👑 Admin</span>
            <span style={{ color: '#888', fontSize: '0.9rem' }}>Pool settings, ban users, promote moderators</span>
          </div>
          <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
            <span style={{ backgroundColor: '#1a3a4a', color: '#00d4ff', padding: '4px 10px', borderRadius: '4px', fontSize: '0.85rem', minWidth: '120px', textAlign: 'center' }}>🛡️ Moderator</span>
            <span style={{ color: '#888', fontSize: '0.9rem' }}>Manage channels, mute users, view reports</span>
          </div>
          <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
            <span style={{ backgroundColor: '#2a2a4a', color: '#888', padding: '4px 10px', borderRadius: '4px', fontSize: '0.85rem', minWidth: '120px', textAlign: 'center' }}>👤 User</span>
            <span style={{ color: '#888', fontSize: '0.9rem' }}>Standard mining pool access</span>
          </div>
        </div>
      </div>

      {/* Role Change Modal */}
      {roleChangeUser && (
        <>
          <div style={{ position: 'fixed', top: 0, left: 0, right: 0, bottom: 0, backgroundColor: 'rgba(0,0,0,0.6)', zIndex: 2999 }} onClick={() => setRoleChangeUser(null)} />
          <div style={styles.editModal}>
            <h3 style={styles.editTitle}>Change Role: {roleChangeUser.username}</h3>
            <p style={{ color: '#888', marginBottom: '15px' }}>
              Current role: <strong style={{ color: '#00d4ff' }}>{roleChangeUser.role || 'user'}</strong>
            </p>
            <div style={styles.formGroup}>
              <label style={styles.label}>New Role</label>
              <select 
                style={styles.select} 
                value={newRole} 
                onChange={e => setNewRole(e.target.value)}
              >
                <option value="">Select a role</option>
                <option value="user">👤 User</option>
                <option value="moderator">🛡️ Moderator</option>
                <option value="admin">👑 Admin</option>
                <option value="super_admin">⭐ Super Admin</option>
              </select>
            </div>
            <div style={styles.editActions}>
              <button style={styles.cancelBtn} onClick={() => { setRoleChangeUser(null); setNewRole(''); }}>Cancel</button>
              <button style={styles.saveBtn} onClick={handleChangeRole} disabled={!newRole}>Change Role</button>
            </div>
          </div>
        </>
      )}
    </div>
  );
}

export default AdminRolesTab;
