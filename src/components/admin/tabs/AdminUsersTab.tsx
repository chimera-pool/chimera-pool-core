import React, { useState, useEffect, useCallback } from 'react';
import { formatHashrate } from '../../../utils/formatters';

interface AdminUsersTabProps {
  token: string;
  isActive: boolean;
  showMessage: (type: 'success' | 'error', text: string) => void;
  onClose: () => void;
}

interface AdminUser {
  id: number;
  username: string;
  email: string;
  payout_address: string;
  pool_fee_percent: number | null;
  total_earnings: number;
  pending_payout: number;
  blocks_found: number;
  is_active: boolean;
  is_admin: boolean;
  total_hashrate: number;
  wallet_count: number;
  primary_wallet: string;
  total_allocated: number;
  role?: string;
}

interface UserDetailData {
  user: AdminUser;
  wallets?: any[];
  wallet_summary?: {
    total_allocated: number;
    has_multiple_wallets: boolean;
    remaining_percent: number;
  };
  shares_stats: {
    total_shares: number;
    valid_shares: number;
    invalid_shares: number;
    last_24_hours: number;
  };
  miners?: any[];
}

interface EditForm {
  pool_fee_percent: string;
  payout_address: string;
  is_active: boolean;
  is_admin: boolean;
}

const styles: { [key: string]: React.CSSProperties } = {
  searchBar: { padding: '16px 20px' },
  searchInput: { width: '100%', padding: '12px 16px', backgroundColor: 'rgba(13, 8, 17, 0.8)', border: '1px solid rgba(74, 44, 90, 0.5)', borderRadius: '10px', color: '#F0EDF4', fontSize: '1rem', boxSizing: 'border-box' as const },
  loading: { padding: '40px', textAlign: 'center', color: '#D4A84B' },
  tableContainer: { overflowX: 'auto', padding: '0 20px' },
  table: { width: '100%', borderCollapse: 'collapse' },
  th: { padding: '14px', textAlign: 'left', borderBottom: '2px solid rgba(74, 44, 90, 0.5)', color: '#D4A84B', fontSize: '0.8rem', textTransform: 'uppercase', letterSpacing: '0.03em', fontWeight: 600, cursor: 'pointer' },
  tr: { borderBottom: '1px solid rgba(74, 44, 90, 0.3)' },
  td: { padding: '14px', color: '#F0EDF4' },
  adminBadge: { color: '#D4A84B' },
  activeBadge: { background: 'rgba(74, 222, 128, 0.15)', color: '#4ade80', padding: '4px 10px', borderRadius: '6px', fontSize: '0.8rem', border: '1px solid rgba(74, 222, 128, 0.3)' },
  inactiveBadge: { background: 'rgba(248, 113, 113, 0.15)', color: '#f87171', padding: '4px 10px', borderRadius: '6px', fontSize: '0.8rem', border: '1px solid rgba(248, 113, 113, 0.3)' },
  actionBtn: { background: 'none', border: 'none', cursor: 'pointer', fontSize: '1.1rem', padding: '4px 8px' },
  pagination: { display: 'flex', justifyContent: 'center', alignItems: 'center', gap: '20px', padding: '20px' },
  pageBtn: { padding: '10px 18px', background: 'rgba(74, 44, 90, 0.4)', border: 'none', borderRadius: '8px', color: '#F0EDF4', cursor: 'pointer' },
  pageInfo: { color: '#B8B4C8' },
  editModal: { position: 'fixed' as const, top: '50%', left: '50%', transform: 'translate(-50%, -50%)', background: 'linear-gradient(180deg, #2D1F3D 0%, #1A0F1E 100%)', padding: '28px', borderRadius: '16px', border: '1px solid rgba(212, 168, 75, 0.3)', minWidth: '400px', zIndex: 3000, boxShadow: '0 24px 48px rgba(0, 0, 0, 0.5)' },
  editTitle: { color: '#D4A84B', marginTop: 0, fontWeight: 600 },
  formGroup: { marginBottom: '16px' },
  label: { display: 'block', color: '#B8B4C8', marginBottom: '6px', fontSize: '0.9rem', fontWeight: 500 },
  formInput: { width: '100%', padding: '12px 14px', backgroundColor: 'rgba(13, 8, 17, 0.8)', border: '1px solid rgba(74, 44, 90, 0.5)', borderRadius: '10px', color: '#F0EDF4', boxSizing: 'border-box' as const },
  checkboxLabel: { display: 'inline-flex', alignItems: 'center', gap: '8px', color: '#F0EDF4', marginRight: '20px' },
  editActions: { display: 'flex', gap: '12px', marginTop: '24px' },
  cancelBtn: { flex: 1, padding: '12px', backgroundColor: 'transparent', border: '1px solid rgba(74, 44, 90, 0.5)', borderRadius: '10px', color: '#B8B4C8', cursor: 'pointer' },
  saveBtn: { flex: 1, padding: '12px', background: 'linear-gradient(135deg, #D4A84B 0%, #B8923A 100%)', border: 'none', borderRadius: '10px', color: '#1A0F1E', fontWeight: 600, cursor: 'pointer' },
  detailModal: { position: 'fixed' as const, top: '10%', right: '20px', width: '400px', maxHeight: '80vh', overflow: 'auto', background: 'linear-gradient(180deg, #2D1F3D 0%, #1A0F1E 100%)', padding: '24px', borderRadius: '16px', border: '1px solid rgba(74, 44, 90, 0.4)', zIndex: 2500, boxShadow: '0 16px 32px rgba(0, 0, 0, 0.4)' },
  closeDetailBtn: { position: 'absolute' as const, top: '15px', right: '15px', background: 'none', border: 'none', color: '#B8B4C8', fontSize: '24px', cursor: 'pointer' },
  detailTitle: { color: '#D4A84B', marginTop: 0 },
  detailCard: { backgroundColor: 'rgba(13, 8, 17, 0.6)', padding: '15px', borderRadius: '10px', marginBottom: '15px', border: '1px solid rgba(74, 44, 90, 0.3)' },
  subTitle: { color: '#B8B4C8', marginBottom: '10px', fontSize: '0.95rem' },
  minerRow: { display: 'flex', justifyContent: 'space-between', alignItems: 'center', padding: '10px 15px', backgroundColor: 'rgba(13, 8, 17, 0.4)', borderRadius: '8px', marginBottom: '8px' },
  select: { width: '100%', padding: '12px 14px', backgroundColor: 'rgba(13, 8, 17, 0.8)', border: '1px solid rgba(74, 44, 90, 0.5)', borderRadius: '10px', color: '#F0EDF4' },
};

export function AdminUsersTab({ token, isActive, showMessage, onClose }: AdminUsersTabProps) {
  const [users, setUsers] = useState<AdminUser[]>([]);
  const [loading, setLoading] = useState(false);
  const [page, setPage] = useState(1);
  const [pageSize] = useState(10);
  const [totalCount, setTotalCount] = useState(0);
  const [search, setSearch] = useState('');
  const [sortField, setSortField] = useState('id');
  const [sortDirection, setSortDirection] = useState<'asc' | 'desc'>('asc');
  const [editingUser, setEditingUser] = useState<AdminUser | null>(null);
  const [editForm, setEditForm] = useState<EditForm>({ pool_fee_percent: '', payout_address: '', is_active: true, is_admin: false });
  const [selectedUser, setSelectedUser] = useState<UserDetailData | null>(null);
  const [roleChangeUser, setRoleChangeUser] = useState<AdminUser | null>(null);
  const [newRole, setNewRole] = useState('');

  const fetchUsers = useCallback(async () => {
    setLoading(true);
    try {
      const params = new URLSearchParams({ 
        page: String(page), 
        page_size: String(pageSize),
        sort_field: sortField,
        sort_direction: sortDirection
      });
      if (search) params.append('search', search);
      const response = await fetch(`/api/v1/admin/users?${params}`, { headers: { 'Authorization': `Bearer ${token}` } });
      if (response.ok) {
        const data = await response.json();
        setUsers(data.users || []);
        setTotalCount(data.total_count || 0);
      } else if (response.status === 403) {
        showMessage('error', 'Admin access required');
        onClose();
      }
    } catch (error) {
      console.error('Failed to fetch users:', error);
    } finally {
      setLoading(false);
    }
  }, [token, page, pageSize, sortField, sortDirection, search, showMessage, onClose]);

  useEffect(() => {
    if (isActive) {
      fetchUsers();
    }
  }, [isActive, fetchUsers]);

  const fetchUserDetail = async (userId: number) => {
    try {
      const response = await fetch(`/api/v1/admin/users/${userId}`, { headers: { 'Authorization': `Bearer ${token}` } });
      if (response.ok) setSelectedUser(await response.json());
    } catch (error) {
      console.error('Failed to fetch user details:', error);
    }
  };

  const handleEditUser = (user: AdminUser) => {
    setEditingUser(user);
    setEditForm({ pool_fee_percent: user.pool_fee_percent?.toString() || '', payout_address: user.payout_address || '', is_active: user.is_active, is_admin: user.is_admin });
  };

  const handleSaveUser = async () => {
    if (!editingUser) return;
    const updates: any = {};
    if (editForm.pool_fee_percent !== '') {
      const fee = parseFloat(editForm.pool_fee_percent);
      if (isNaN(fee) || fee < 0 || fee > 100) { showMessage('error', 'Pool fee must be between 0 and 100'); return; }
      updates.pool_fee_percent = fee;
    }
    if (editForm.payout_address) updates.payout_address = editForm.payout_address;
    updates.is_active = editForm.is_active;
    updates.is_admin = editForm.is_admin;

    try {
      const response = await fetch(`/api/v1/admin/users/${editingUser.id}`, {
        method: 'PUT',
        headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
        body: JSON.stringify(updates)
      });
      if (response.ok) { showMessage('success', 'User updated successfully'); setEditingUser(null); fetchUsers(); }
      else { const data = await response.json(); showMessage('error', data.error || 'Failed to update user'); }
    } catch (error) { showMessage('error', 'Network error'); }
  };

  const handleDeleteUser = async (userId: number) => {
    if (!window.confirm('Are you sure you want to deactivate this user?')) return;
    try {
      const response = await fetch(`/api/v1/admin/users/${userId}`, { method: 'DELETE', headers: { 'Authorization': `Bearer ${token}` } });
      if (response.ok) { showMessage('success', 'User deactivated'); fetchUsers(); }
      else { const data = await response.json(); showMessage('error', data.error || 'Failed to deactivate user'); }
    } catch (error) { showMessage('error', 'Network error'); }
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
        fetchUsers();
      } else {
        const data = await response.json();
        showMessage('error', data.error || 'Failed to change role');
      }
    } catch (error) {
      showMessage('error', 'Network error');
    }
  };

  const handleSort = (field: string) => {
    if (sortField === field) {
      setSortDirection(sortDirection === 'asc' ? 'desc' : 'asc');
    } else {
      setSortField(field);
      setSortDirection('asc');
    }
  };

  const totalPages = Math.ceil(totalCount / pageSize);

  const SortableHeader = ({ field, label }: { field: string; label: string }) => (
    <th 
      style={styles.th} 
      onClick={() => handleSort(field)}
    >
      {label} {sortField === field && (sortDirection === 'asc' ? '↑' : '↓')}
    </th>
  );

  if (!isActive) return null;

  return (
    <>
      <div style={styles.searchBar}>
        <input style={styles.searchInput} type="text" placeholder="Search users by username or email..." value={search} onChange={e => { setSearch(e.target.value); setPage(1); }} />
      </div>

      {loading ? (
        <div style={styles.loading}>Loading users...</div>
      ) : (
        <>
          <div style={styles.tableContainer}>
            <table style={styles.table}>
              <thead>
                <tr>
                  <SortableHeader field="id" label="#" />
                  <SortableHeader field="username" label="Username" />
                  <SortableHeader field="email" label="Email" />
                  <SortableHeader field="wallet_count" label="Wallets" />
                  <SortableHeader field="total_hashrate" label="Hashrate" />
                  <SortableHeader field="total_earnings" label="Earnings" />
                  <SortableHeader field="pool_fee_percent" label="Fee %" />
                  <SortableHeader field="is_active" label="Status" />
                  <th style={{...styles.th, cursor: 'default'}}>Actions</th>
                </tr>
              </thead>
              <tbody>
                {users.map(user => (
                  <tr key={user.id} style={styles.tr}>
                    <td style={{...styles.td, color: '#D4A84B', fontWeight: 600, fontFamily: 'monospace'}}>{user.id}</td>
                    <td style={styles.td}><span style={user.is_admin ? styles.adminBadge : {}}>{user.username} {user.is_admin && '👑'}</span></td>
                    <td style={styles.td}>{user.email}</td>
                    <td style={styles.td}>
                      <div style={{ display: 'flex', flexDirection: 'column', gap: '2px' }}>
                        <span style={{ color: user.wallet_count > 1 ? '#00d4ff' : '#888', fontWeight: user.wallet_count > 1 ? 'bold' : 'normal' }}>
                          {user.wallet_count || 0} wallet{user.wallet_count !== 1 ? 's' : ''}
                        </span>
                        {user.primary_wallet && (
                          <span style={{ fontSize: '0.75rem', color: '#666', maxWidth: '120px', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }} title={user.primary_wallet}>
                            {user.primary_wallet.substring(0, 12)}...
                          </span>
                        )}
                        {user.wallet_count > 1 && (
                          <span style={{ fontSize: '0.7rem', backgroundColor: '#1a3a4a', color: '#00d4ff', padding: '1px 4px', borderRadius: '3px', display: 'inline-block', width: 'fit-content' }}>
                            Split: {user.total_allocated?.toFixed(0) || 0}%
                          </span>
                        )}
                      </div>
                    </td>
                    <td style={styles.td}>{formatHashrate(user.total_hashrate)}</td>
                    <td style={styles.td}>{user.total_earnings.toFixed(4)}</td>
                    <td style={styles.td}>{user.pool_fee_percent || 'Default'}</td>
                    <td style={styles.td}><span style={user.is_active ? styles.activeBadge : styles.inactiveBadge}>{user.is_active ? 'Active' : 'Inactive'}</span></td>
                    <td style={styles.td}>
                      <button style={styles.actionBtn} onClick={() => fetchUserDetail(user.id)} title="View Details">👁️</button>
                      <button style={styles.actionBtn} onClick={() => handleEditUser(user)} title="Edit User">✏️</button>
                      <button style={{...styles.actionBtn, backgroundColor: '#2a1a4a', borderRadius: '4px'}} onClick={() => setRoleChangeUser(user)} title="Change Role">👑</button>
                      <button style={{...styles.actionBtn, opacity: 0.7}} onClick={() => handleDeleteUser(user.id)} title="Delete User">🗑️</button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          <div style={styles.pagination}>
            <button style={styles.pageBtn} disabled={page <= 1} onClick={() => setPage(p => p - 1)}>← Prev</button>
            <span style={styles.pageInfo}>Page {page} of {totalPages} ({totalCount} users)</span>
            <button style={styles.pageBtn} disabled={page >= totalPages} onClick={() => setPage(p => p + 1)}>Next →</button>
          </div>
        </>
      )}

      {/* Edit User Modal */}
      {editingUser && (
        <>
          <div style={{ position: 'fixed', top: 0, left: 0, right: 0, bottom: 0, backgroundColor: 'rgba(0,0,0,0.6)', zIndex: 2999 }} onClick={() => setEditingUser(null)} />
          <div style={styles.editModal}>
            <h3 style={styles.editTitle}>Edit User: {editingUser.username}</h3>
            <div style={styles.formGroup}>
              <label style={styles.label}>Pool Fee % (leave empty for default)</label>
              <input style={styles.formInput} type="number" min="0" max="100" step="0.1" placeholder="e.g., 1.5" value={editForm.pool_fee_percent} onChange={e => setEditForm({...editForm, pool_fee_percent: e.target.value})} />
            </div>
            <div style={styles.formGroup}>
              <label style={styles.label}>Payout Address</label>
              <input style={styles.formInput} type="text" placeholder="0x..." value={editForm.payout_address} onChange={e => setEditForm({...editForm, payout_address: e.target.value})} />
            </div>
            <div style={styles.formGroup}>
              <label style={styles.checkboxLabel}><input type="checkbox" checked={editForm.is_active} onChange={e => setEditForm({...editForm, is_active: e.target.checked})} /> Active</label>
              <label style={styles.checkboxLabel}><input type="checkbox" checked={editForm.is_admin} onChange={e => setEditForm({...editForm, is_admin: e.target.checked})} /> Admin</label>
            </div>
            <div style={styles.editActions}>
              <button style={styles.cancelBtn} onClick={() => setEditingUser(null)}>Cancel</button>
              <button style={styles.saveBtn} onClick={handleSaveUser}>Save Changes</button>
            </div>
          </div>
        </>
      )}

      {/* User Detail Modal */}
      {selectedUser && (
        <div style={styles.detailModal}>
          <button style={styles.closeDetailBtn} onClick={() => setSelectedUser(null)}>×</button>
          <h3 style={styles.detailTitle}>User Details: {selectedUser.user.username}</h3>
          <div style={styles.detailCard}>
            <p><strong>Email:</strong> {selectedUser.user.email}</p>
            <p><strong>Payout Address:</strong> {selectedUser.user.payout_address || 'Not set'}</p>
            <p><strong>Pool Fee:</strong> {selectedUser.user.pool_fee_percent || 'Default'}%</p>
            <p><strong>Total Earnings:</strong> {selectedUser.user.total_earnings}</p>
            <p><strong>Pending Payout:</strong> {selectedUser.user.pending_payout}</p>
            <p><strong>Blocks Found:</strong> {selectedUser.user.blocks_found}</p>
          </div>

          {/* Wallet Configuration Section */}
          <h4 style={styles.subTitle}>💰 Wallet Configuration ({selectedUser.wallets?.length || 0} wallets)</h4>
          {selectedUser.wallet_summary && (
            <div style={{ ...styles.detailCard, backgroundColor: '#0a1520', borderColor: '#00d4ff' }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '10px' }}>
                <span>Total Allocated:</span>
                <span style={{ color: selectedUser.wallet_summary.total_allocated >= 100 ? '#4ade80' : '#fbbf24', fontWeight: 'bold' }}>
                  {selectedUser.wallet_summary.total_allocated?.toFixed(1)}%
                </span>
              </div>
              {selectedUser.wallet_summary.has_multiple_wallets && (
                <div style={{ backgroundColor: '#1a3a4a', padding: '8px 12px', borderRadius: '6px', marginBottom: '10px' }}>
                  <span style={{ color: '#00d4ff', fontSize: '0.9rem' }}>⚡ Multi-wallet split payments enabled</span>
                </div>
              )}
              {selectedUser.wallet_summary.remaining_percent > 0 && (
                <div style={{ backgroundColor: '#4d3a1a', padding: '8px 12px', borderRadius: '6px' }}>
                  <span style={{ color: '#fbbf24', fontSize: '0.9rem' }}>⚠️ {selectedUser.wallet_summary.remaining_percent?.toFixed(1)}% unallocated</span>
                </div>
              )}
            </div>
          )}
          {selectedUser.wallets && selectedUser.wallets.length > 0 ? (
            <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
              {selectedUser.wallets.map((wallet: any) => (
                <div key={wallet.id} style={{ 
                  ...styles.detailCard, 
                  display: 'flex', 
                  justifyContent: 'space-between', 
                  alignItems: 'center',
                  borderColor: wallet.is_primary ? '#9b59b6' : '#2a2a4a',
                  backgroundColor: wallet.is_active ? '#0a0a15' : '#1a1a1a',
                  opacity: wallet.is_active ? 1 : 0.6
                }}>
                  <div style={{ flex: 1 }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '4px' }}>
                      {wallet.is_primary && <span style={{ backgroundColor: '#4d1a4d', color: '#d946ef', padding: '2px 6px', borderRadius: '4px', fontSize: '0.7rem' }}>PRIMARY</span>}
                      {!wallet.is_active && <span style={{ backgroundColor: '#4d1a1a', color: '#ef4444', padding: '2px 6px', borderRadius: '4px', fontSize: '0.7rem' }}>INACTIVE</span>}
                      {wallet.label && <span style={{ color: '#888', fontSize: '0.85rem' }}>{wallet.label}</span>}
                    </div>
                    <p style={{ margin: 0, fontFamily: 'monospace', fontSize: '0.85rem', color: '#00d4ff', wordBreak: 'break-all' }}>
                      {wallet.address}
                    </p>
                  </div>
                  <div style={{ 
                    backgroundColor: '#1a1a2e', 
                    padding: '8px 16px', 
                    borderRadius: '8px', 
                    marginLeft: '15px',
                    textAlign: 'center',
                    minWidth: '80px'
                  }}>
                    <div style={{ fontSize: '1.2rem', fontWeight: 'bold', color: '#4ade80' }}>{wallet.percentage}%</div>
                    <div style={{ fontSize: '0.7rem', color: '#888' }}>allocation</div>
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <div style={{ ...styles.detailCard, textAlign: 'center', color: '#666' }}>
              <p>No wallets configured. User is using legacy payout address.</p>
            </div>
          )}

          <h4 style={styles.subTitle}>Share Statistics</h4>
          <div style={styles.detailCard}>
            <p><strong>Total Shares:</strong> {selectedUser.shares_stats?.total_shares || 0}</p>
            <p><strong>Valid:</strong> {selectedUser.shares_stats?.valid_shares || 0}</p>
            <p><strong>Invalid:</strong> {selectedUser.shares_stats?.invalid_shares || 0}</p>
            <p><strong>Last 24h:</strong> {selectedUser.shares_stats?.last_24_hours || 0}</p>
          </div>
          
          <h4 style={styles.subTitle}>Miners ({selectedUser.miners?.length || 0})</h4>
          {selectedUser.miners?.map((m: any) => (
            <div key={m.id} style={styles.minerRow}>
              <span>{m.name}</span>
              <span>{formatHashrate(m.hashrate)}</span>
              <span style={m.is_active ? styles.activeBadge : styles.inactiveBadge}>{m.is_active ? 'Online' : 'Offline'}</span>
            </div>
          ))}
        </div>
      )}

      {/* Role Change Modal */}
      {roleChangeUser && (
        <>
          <div style={{ position: 'fixed', top: 0, left: 0, right: 0, bottom: 0, backgroundColor: 'rgba(0,0,0,0.6)', zIndex: 2999 }} onClick={() => setRoleChangeUser(null)} />
          <div style={styles.editModal}>
            <h3 style={styles.editTitle}>👑 Change Role: {roleChangeUser.username}</h3>
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
    </>
  );
}

export default AdminUsersTab;
