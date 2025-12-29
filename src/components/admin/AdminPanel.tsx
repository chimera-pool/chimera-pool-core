import React, { useState, useEffect, useCallback, memo } from 'react';
import { useAutoRefresh, REFRESH_INTERVALS } from '../../hooks/useAutoRefresh';
import {
  ResponsiveContainer,
  AreaChart,
  Area,
  LineChart,
  Line,
  BarChart,
  Bar,
  PieChart,
  Pie,
  Cell,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend
} from 'recharts';
import { formatHashrate } from '../../utils/formatters';
import AdminStatsTab from './tabs/AdminStatsTab';
import { AdminGrafanaSection } from './tabs/AdminGrafanaSection';
import AdminBugsTab from './tabs/AdminBugsTab';
import AdminMinersTab from './tabs/AdminMinersTab';
import AdminNetworkTab from './tabs/AdminNetworkTab';
import AdminRolesTab from './tabs/AdminRolesTab';
import AdminAlgorithmTab from './tabs/AdminAlgorithmTab';
import AdminUsersTab from './tabs/AdminUsersTab';

type TimeRange = '1h' | '6h' | '24h' | '7d' | '30d' | '3m' | '6m' | '1y' | 'all';

const graphStyles: { [key: string]: React.CSSProperties } = {
  section: { background: 'linear-gradient(135deg, #1a1a2e 0%, #0f0f1a 100%)', borderRadius: '12px', padding: '24px', border: '1px solid #2a2a4a', marginBottom: '20px' },
  header: { display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '20px', flexWrap: 'wrap', gap: '15px' },
  title: { fontSize: '1.3rem', color: '#00d4ff', margin: 0 },
  timeSelector: { display: 'flex', gap: '8px', flexWrap: 'wrap' },
  timeBtn: { padding: '6px 12px', backgroundColor: '#0a0a15', border: '1px solid #2a2a4a', borderRadius: '6px', color: '#888', cursor: 'pointer', fontSize: '0.85rem' },
  timeBtnActive: { backgroundColor: '#00d4ff', color: '#0a0a0f', borderColor: '#00d4ff' },
  loading: { textAlign: 'center', padding: '60px', color: '#00d4ff' },
  chartsGrid: { display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(400px, 1fr))', gap: '20px' },
  chartCard: { backgroundColor: '#0a0a15', borderRadius: '10px', padding: '20px', border: '1px solid #2a2a4a' },
  chartTitle: { color: '#e0e0e0', fontSize: '1rem', marginTop: 0, marginBottom: '15px' },
};

interface AdminUser {
  id: number;
  username: string;
  email: string;
  payout_address: string;
  pool_fee_percent: number;
  is_active: boolean;
  is_admin: boolean;
  role: string;
  created_at: string;
  total_earnings: number;
  pending_payout: number;
  total_hashrate: number;
  active_miners: number;
  wallet_count: number;
  primary_wallet: string;
  total_allocated: number;
}
interface AdminPanelProps {
  token: string;
  onClose: () => void;
  showMessage: (type: 'success' | 'error', text: string) => void;
}

function AdminPanel({ token, onClose, showMessage }: AdminPanelProps) {
  const [activeTab, setActiveTab] = useState<'users' | 'stats' | 'algorithm' | 'network' | 'roles' | 'bugs' | 'miners'>('users');
  const [users, setUsers] = useState<AdminUser[]>([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState('');
  const [page, setPage] = useState(1);
  const [totalCount, setTotalCount] = useState(0);
  const [selectedUser, setSelectedUser] = useState<any>(null);
  const [editingUser, setEditingUser] = useState<AdminUser | null>(null);
  const [editForm, setEditForm] = useState({ pool_fee_percent: '', payout_address: '', is_active: true, is_admin: false });
  const pageSize = 10;

  // User sorting state
  const [sortField, setSortField] = useState<'id' | 'username' | 'email' | 'wallet_count' | 'total_hashrate' | 'total_earnings' | 'pool_fee_percent' | 'is_active'>('id');
  const [sortDirection, setSortDirection] = useState<'asc' | 'desc'>('asc');

  // Bug reports state
  const [adminBugs, setAdminBugs] = useState<any[]>([]);
  const [bugsLoading, setBugsLoading] = useState(false);
  const [bugFilter, setBugFilter] = useState({ status: '', priority: '', category: '' });
  const [selectedAdminBug, setSelectedAdminBug] = useState<any>(null);
  const [adminBugComment, setAdminBugComment] = useState('');
  const [isInternalComment, setIsInternalComment] = useState(false);

  // Role management state
  const [moderators, setModerators] = useState<any[]>([]);
  const [admins, setAdmins] = useState<any[]>([]);
  const [rolesLoading, setRolesLoading] = useState(false);
  const [roleChangeUser, setRoleChangeUser] = useState<any>(null);
  const [newRole, setNewRole] = useState('');

  // Pool statistics state removed - now managed by isolated AdminStatsTab component

  // Algorithm settings state
  const [algorithmData, setAlgorithmData] = useState<any>(null);
  const [algorithmForm, setAlgorithmForm] = useState({
    algorithm: '',
    algorithm_variant: '',
    difficulty_target: '',
    block_time: '',
    stratum_port: '',
    algorithm_params: ''
  });

  // Network configuration state
  const [networks, setNetworks] = useState<any[]>([]);

  // Miner monitoring state
  const [allMiners, setAllMiners] = useState<any[]>([]);
  const [minersLoading, setMinersLoading] = useState(false);
  const [minerSearch, setMinerSearch] = useState('');
  const [minerPage, setMinerPage] = useState(1);
  const [minerTotal, setMinerTotal] = useState(0);
  const [activeMinersOnly, setActiveMinersOnly] = useState(false);
  const [selectedMiner, setSelectedMiner] = useState<any>(null);
  const [selectedUserMiners, setSelectedUserMiners] = useState<any>(null);
  const [activeNetwork, setActiveNetwork] = useState<any>(null);
  const [networksLoading, setNetworksLoading] = useState(false);
  const [selectedNetwork, setSelectedNetwork] = useState<any>(null);
  const [editingNetwork, setEditingNetwork] = useState<any>(null);
  const [networkForm, setNetworkForm] = useState({
    name: '', symbol: '', display_name: '', algorithm: 'scrypt',
    rpc_url: '', rpc_user: '', rpc_password: '',
    pool_wallet_address: '', stratum_port: '3333',
    block_time_target: '150', pool_fee_percent: '1.0',
    min_payout_threshold: '0.01', network_type: 'mainnet', description: ''
  });
  const [switchReason, setSwitchReason] = useState('');
  const [networkHistory, setNetworkHistory] = useState<any[]>([]);
  const [savingAlgorithm, setSavingAlgorithm] = useState(false);

  // Community/Channel management state
  const [channels, setChannels] = useState<any[]>([]);
  const [categories, setCategories] = useState<any[]>([]);
  const [channelsLoading, setChannelsLoading] = useState(false);
  const [showCreateChannel, setShowCreateChannel] = useState(false);
  const [showCreateCategory, setShowCreateCategory] = useState(false);
  const [editingChannel, setEditingChannel] = useState<any>(null);
  const [channelForm, setChannelForm] = useState({ name: '', description: '', category_id: '', type: 'text', is_read_only: false, admin_only_post: false });
  const [categoryForm, setCategoryForm] = useState({ name: '', description: '' });

  // Stats tab auto-refresh removed - now managed by isolated AdminStatsTab component

  useEffect(() => { fetchUsers(); }, [page, search, sortField, sortDirection]);
  useEffect(() => { if (activeTab === 'algorithm') fetchAlgorithmSettings(); }, [activeTab]);
  useEffect(() => { if (activeTab === 'roles') fetchRoles(); }, [activeTab]);
  useEffect(() => { if (activeTab === 'bugs') fetchAdminBugs(); }, [activeTab, bugFilter]);
  useEffect(() => { if (activeTab === 'network') fetchNetworks(); }, [activeTab]);
  useEffect(() => { if (activeTab === 'miners') fetchAllMiners(); }, [activeTab, minerPage, minerSearch, activeMinersOnly]);

  // Miner monitoring functions
  const fetchAllMiners = async () => {
    setMinersLoading(true);
    try {
      const params = new URLSearchParams({
        page: minerPage.toString(),
        limit: '20',
        ...(minerSearch && { search: minerSearch }),
        ...(activeMinersOnly && { active: 'true' })
      });
      const response = await fetch(`/api/v1/admin/monitoring/miners?${params}`, {
        headers: { 'Authorization': `Bearer ${token}` }
      });
      if (response.ok) {
        const data = await response.json();
        setAllMiners(data.miners || []);
        setMinerTotal(data.total || 0);
      }
    } catch (error) {
      console.error('Failed to fetch miners:', error);
    } finally {
      setMinersLoading(false);
    }
  };

  const fetchMinerDetail = async (minerId: number) => {
    try {
      const response = await fetch(`/api/v1/admin/monitoring/miners/${minerId}`, {
        headers: { 'Authorization': `Bearer ${token}` }
      });
      if (response.ok) {
        const data = await response.json();
        setSelectedMiner(data);
      }
    } catch (error) {
      console.error('Failed to fetch miner details:', error);
    }
  };

  const fetchUserMiners = async (userId: number) => {
    try {
      const response = await fetch(`/api/v1/admin/monitoring/users/${userId}/miners`, {
        headers: { 'Authorization': `Bearer ${token}` }
      });
      if (response.ok) {
        const data = await response.json();
        setSelectedUserMiners(data);
      }
    } catch (error) {
      console.error('Failed to fetch user miners:', error);
    }
  };

  // Bug management functions
  const fetchAdminBugs = async () => {
    setBugsLoading(true);
    try {
      const params = new URLSearchParams();
      if (bugFilter.status) params.append('status', bugFilter.status);
      if (bugFilter.priority) params.append('priority', bugFilter.priority);
      if (bugFilter.category) params.append('category', bugFilter.category);
      
      const response = await fetch(`/api/v1/admin/bugs?${params.toString()}`, {
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
      showMessage('error', 'Failed to fetch bug details');
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
      showMessage('error', 'Network error');
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
      showMessage('error', 'Network error');
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
      showMessage('error', 'Network error');
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
      showMessage('error', 'Network error');
    }
  };

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

  const fetchRoles = async () => {
    setRolesLoading(true);
    try {
      const headers = { 'Authorization': `Bearer ${token}` };
      const [modsRes, adminsRes] = await Promise.all([
        fetch('/api/v1/admin/moderators', { headers }),
        fetch('/api/v1/admin/admins', { headers })
      ]);
      if (modsRes.ok) {
        const data = await modsRes.json();
        setModerators(data.moderators || []);
      }
      if (adminsRes.ok) {
        const data = await adminsRes.json();
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
        showMessage('success', `User role changed to ${newRole}`);
        setRoleChangeUser(null);
        setNewRole('');
        fetchRoles();
        fetchUsers();
      } else {
        const data = await response.json();
        showMessage('error', data.error || 'Failed to change role');
      }
    } catch (error) {
      showMessage('error', 'Network error');
    }
  };

  const fetchChannelsAndCategories = async () => {
    setChannelsLoading(true);
    try {
      const headers = { 'Authorization': `Bearer ${token}` };
      const [channelsRes, categoriesRes] = await Promise.all([
        fetch('/api/v1/community/channels', { headers }),
        fetch('/api/v1/community/channel-categories', { headers })
      ]);
      if (channelsRes.ok) {
        const data = await channelsRes.json();
        setChannels(data.channels || []);
      }
      if (categoriesRes.ok) {
        const data = await categoriesRes.json();
        setCategories(data.categories || []);
      }
    } catch (error) {
      console.error('Failed to fetch channels:', error);
    } finally {
      setChannelsLoading(false);
    }
  };

  const handleCreateChannel = async () => {
    if (!channelForm.name || !channelForm.category_id) {
      showMessage('error', 'Channel name and category are required');
      return;
    }
    try {
      const response = await fetch('/api/v1/admin/community/channels', {
        method: 'POST',
        headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
        body: JSON.stringify(channelForm)
      });
      if (response.ok) {
        showMessage('success', 'Channel created successfully');
        setShowCreateChannel(false);
        setChannelForm({ name: '', description: '', category_id: '', type: 'text', is_read_only: false, admin_only_post: false });
        fetchChannelsAndCategories();
      } else {
        const data = await response.json();
        showMessage('error', data.error || 'Failed to create channel');
      }
    } catch (error) {
      showMessage('error', 'Network error');
    }
  };

  const handleUpdateChannel = async () => {
    if (!editingChannel || !channelForm.name) {
      showMessage('error', 'Channel name is required');
      return;
    }
    try {
      const response = await fetch(`/api/v1/admin/community/channels/${editingChannel.id}`, {
        method: 'PUT',
        headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
        body: JSON.stringify(channelForm)
      });
      if (response.ok) {
        showMessage('success', 'Channel updated successfully');
        setEditingChannel(null);
        setChannelForm({ name: '', description: '', category_id: '', type: 'text', is_read_only: false, admin_only_post: false });
        fetchChannelsAndCategories();
      } else {
        const data = await response.json();
        showMessage('error', data.error || 'Failed to update channel');
      }
    } catch (error) {
      showMessage('error', 'Network error');
    }
  };

  const handleDeleteChannel = async (channelId: string) => {
    if (!window.confirm('Are you sure you want to delete this channel?')) return;
    try {
      const response = await fetch(`/api/v1/admin/community/channels/${channelId}`, {
        method: 'DELETE',
        headers: { 'Authorization': `Bearer ${token}` }
      });
      if (response.ok) {
        showMessage('success', 'Channel deleted');
        fetchChannelsAndCategories();
      } else {
        const data = await response.json();
        showMessage('error', data.error || 'Failed to delete channel');
      }
    } catch (error) {
      showMessage('error', 'Network error');
    }
  };

  const handleCreateCategory = async () => {
    if (!categoryForm.name) {
      showMessage('error', 'Category name is required');
      return;
    }
    try {
      const response = await fetch('/api/v1/admin/community/channel-categories', {
        method: 'POST',
        headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
        body: JSON.stringify(categoryForm)
      });
      if (response.ok) {
        showMessage('success', 'Category created successfully');
        setShowCreateCategory(false);
        setCategoryForm({ name: '', description: '' });
        fetchChannelsAndCategories();
      } else {
        const data = await response.json();
        showMessage('error', data.error || 'Failed to create category');
      }
    } catch (error) {
      showMessage('error', 'Network error');
    }
  };

  const handleDeleteCategory = async (categoryId: string) => {
    if (!window.confirm('Are you sure you want to delete this category? All channels in this category will be deleted.')) return;
    try {
      const response = await fetch(`/api/v1/admin/community/channel-categories/${categoryId}`, {
        method: 'DELETE',
        headers: { 'Authorization': `Bearer ${token}` }
      });
      if (response.ok) {
        showMessage('success', 'Category deleted');
        fetchChannelsAndCategories();
      } else {
        const data = await response.json();
        showMessage('error', data.error || 'Failed to delete category');
      }
    } catch (error) {
      showMessage('error', 'Network error');
    }
  };

  const openEditChannel = (channel: any) => {
    setEditingChannel(channel);
    setChannelForm({
      name: channel.name,
      description: channel.description || '',
      category_id: channel.category_id,
      type: channel.type || 'text',
      is_read_only: channel.is_read_only || false,
      admin_only_post: channel.admin_only_post || false
    });
  };

  // fetchPoolStatsInternal removed - now managed by isolated AdminStatsTab component

  const fetchAlgorithmSettings = async () => {
    try {
      const response = await fetch('/api/v1/admin/algorithm', {
        headers: { 'Authorization': `Bearer ${token}` }
      });
      if (response.ok) {
        const data = await response.json();
        setAlgorithmData(data);
        setAlgorithmForm({
          algorithm: data.algorithm?.algorithm?.value || '',
          algorithm_variant: data.algorithm?.algorithm_variant?.value || '',
          difficulty_target: data.algorithm?.difficulty_target?.value || '',
          block_time: data.algorithm?.block_time?.value || '',
          stratum_port: data.algorithm?.stratum_port?.value || '',
          algorithm_params: data.algorithm?.algorithm_params?.value || '{}'
        });
      }
    } catch (error) {
      console.error('Failed to fetch algorithm settings:', error);
    }
  };

  const handleSaveAlgorithm = async () => {
    setSavingAlgorithm(true);
    try {
      const response = await fetch('/api/v1/admin/algorithm', {
        method: 'PUT',
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json'
        },
        body: JSON.stringify(algorithmForm)
      });
      if (response.ok) {
        const data = await response.json();
        showMessage('success', data.message + (data.note ? ` Note: ${data.note}` : ''));
        fetchAlgorithmSettings();
      } else {
        const data = await response.json();
        showMessage('error', data.error || 'Failed to update algorithm settings');
      }
    } catch (error) {
      showMessage('error', 'Network error');
    } finally {
      setSavingAlgorithm(false);
    }
  };

  // User sorting handler
  const handleSort = (field: typeof sortField) => {
    if (sortField === field) {
      setSortDirection(sortDirection === 'asc' ? 'desc' : 'asc');
    } else {
      setSortField(field);
      setSortDirection('asc');
    }
  };

  // Server-side sorting - users are already sorted by the API

  // Sortable header component
  const SortableHeader = ({ field, label }: { field: typeof sortField, label: string }) => (
    <th 
      style={{ ...adminStyles.th, cursor: 'pointer', userSelect: 'none', transition: 'background 0.2s' }}
      onClick={() => handleSort(field)}
    >
      {label} {sortField === field && (sortDirection === 'asc' ? '↑' : '↓')}
    </th>
  );

  // Network configuration functions
  const fetchNetworks = async () => {
    setNetworksLoading(true);
    try {
      const [networksRes, activeRes, historyRes] = await Promise.all([
        fetch('/api/v1/admin/networks', { headers: { 'Authorization': `Bearer ${token}` } }),
        fetch('/api/v1/network/active', { headers: { 'Authorization': `Bearer ${token}` } }),
        fetch('/api/v1/admin/networks/history', { headers: { 'Authorization': `Bearer ${token}` } })
      ]);
      if (networksRes.ok) {
        const data = await networksRes.json();
        setNetworks(data.networks || []);
      }
      if (activeRes.ok) {
        const data = await activeRes.json();
        setActiveNetwork(data.network);
      }
      if (historyRes.ok) {
        const data = await historyRes.json();
        setNetworkHistory(data.history || []);
      }
    } catch (error) {
      console.error('Failed to fetch networks:', error);
    } finally {
      setNetworksLoading(false);
    }
  };

  const handleSwitchNetwork = async (networkName: string) => {
    if (!window.confirm(`Switch mining to ${networkName}? This will affect all connected miners.`)) return;
    try {
      const response = await fetch('/api/v1/admin/networks/switch', {
        method: 'POST',
        headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
        body: JSON.stringify({ network_name: networkName, reason: switchReason || 'Manual switch from admin panel' })
      });
      if (response.ok) {
        showMessage('success', `Switched to ${networkName} successfully!`);
        setSwitchReason('');
        fetchNetworks();
      } else {
        const data = await response.json();
        showMessage('error', data.error || 'Failed to switch network');
      }
    } catch (error) {
      showMessage('error', 'Network error');
    }
  };

  const handleUpdateNetwork = async (networkId: string) => {
    try {
      const response = await fetch(`/api/v1/admin/networks/${networkId}`, {
        method: 'PUT',
        headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
        body: JSON.stringify({
          display_name: networkForm.display_name || undefined,
          rpc_url: networkForm.rpc_url || undefined,
          rpc_user: networkForm.rpc_user || undefined,
          rpc_password: networkForm.rpc_password || undefined,
          pool_wallet_address: networkForm.pool_wallet_address || undefined,
          stratum_port: networkForm.stratum_port ? parseInt(networkForm.stratum_port) : undefined,
          pool_fee_percent: networkForm.pool_fee_percent ? parseFloat(networkForm.pool_fee_percent) : undefined,
          min_payout_threshold: networkForm.min_payout_threshold ? parseFloat(networkForm.min_payout_threshold) : undefined,
          description: networkForm.description || undefined
        })
      });
      if (response.ok) {
        showMessage('success', 'Network updated successfully');
        setEditingNetwork(null);
        fetchNetworks();
      } else {
        const data = await response.json();
        showMessage('error', data.error || 'Failed to update network');
      }
    } catch (error) {
      showMessage('error', 'Network error');
    }
  };

  const handleTestConnection = async (networkId: string) => {
    try {
      const response = await fetch(`/api/v1/admin/networks/${networkId}/test`, {
        method: 'POST',
        headers: { 'Authorization': `Bearer ${token}` }
      });
      const data = await response.json();
      if (data.success) {
        showMessage('success', 'Connection test passed!');
      } else {
        showMessage('error', `Connection test failed: ${data.error}`);
      }
    } catch (error) {
      showMessage('error', 'Network error');
    }
  };

  const fetchUsers = async () => {
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
  };

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

  const totalPages = Math.ceil(totalCount / pageSize);

  return (
    <div style={adminStyles.overlay} className="admin-overlay" onClick={onClose}>
      <div style={adminStyles.panel} className="admin-panel" onClick={e => e.stopPropagation()}>
        <div style={adminStyles.header}>
          <h2 style={adminStyles.title}>🛡️ Admin Panel</h2>
          <button style={adminStyles.closeBtn} onClick={onClose}>×</button>
        </div>

        {/* Tabs */}
        <div style={adminStyles.tabs} className="admin-tabs">
          <button 
            style={{...adminStyles.tab, ...(activeTab === 'users' ? adminStyles.tabActive : {})}} 
            className="admin-tab"
            onClick={() => setActiveTab('users')}
          >
            👥 Users
          </button>
          <button 
            style={{...adminStyles.tab, ...(activeTab === 'stats' ? adminStyles.tabActive : {})}} 
            className="admin-tab"
            onClick={() => setActiveTab('stats')}
          >
            📊 Stats
          </button>
          <button 
            style={{...adminStyles.tab, ...(activeTab === 'algorithm' ? adminStyles.tabActive : {})}} 
            className="admin-tab"
            onClick={() => setActiveTab('algorithm')}
          >
            ⚙️ Algo
          </button>
          <button 
            style={{...adminStyles.tab, ...(activeTab === 'network' ? adminStyles.tabActive : {})}} 
            className="admin-tab"
            onClick={() => setActiveTab('network')}
          >
            🌐 Network
          </button>
          <button 
            style={{...adminStyles.tab, ...(activeTab === 'roles' ? adminStyles.tabActive : {})}} 
            className="admin-tab"
            onClick={() => setActiveTab('roles')}
          >
            👑 Roles
          </button>
          <button 
            style={{...adminStyles.tab, ...(activeTab === 'bugs' ? adminStyles.tabActive : {}), position: 'relative'}} 
            className="admin-tab"
            onClick={() => setActiveTab('bugs')}
          >
            🐛 Bugs
            {adminBugs.filter(b => b.status === 'open').length > 0 && (
              <span style={{ position: 'absolute', top: '-5px', right: '-5px', backgroundColor: '#ef4444', color: '#fff', fontSize: '0.7rem', padding: '2px 6px', borderRadius: '10px', minWidth: '18px', textAlign: 'center' }}>
                {adminBugs.filter(b => b.status === 'open').length}
              </span>
            )}
          </button>
          <button 
            style={{...adminStyles.tab, ...(activeTab === 'miners' ? adminStyles.tabActive : {})}} 
            className="admin-tab"
            onClick={() => setActiveTab('miners')}
          >
            ⛏️ Miners
          </button>
        </div>

        {/* User Management Tab */}
        <AdminUsersTab token={token} isActive={activeTab === 'users'} showMessage={showMessage} onClose={onClose} />

        {/* Pool Statistics Tab */}
        {activeTab === 'stats' && (
          <div style={adminStyles.algorithmContainer}>
            <div style={adminStyles.algoHeader}>
              <h3 style={adminStyles.algoTitle}>📊 Pool Statistics</h3>
              <p style={adminStyles.algoDesc}>
                View pool performance metrics and charts.
              </p>
            </div>
            <AdminStatsTab token={token} isActive={activeTab === 'stats'} />
          </div>
        )}

        {/* Algorithm Settings Tab */}
        <AdminAlgorithmTab token={token} isActive={activeTab === 'algorithm'} showMessage={showMessage} />

        {/* Legacy Algorithm code removed - now using AdminAlgorithmTab component */}
        {/* Network Configuration Tab */}
        <AdminNetworkTab token={token} isActive={activeTab === 'network'} showMessage={showMessage} />
      </div>
    </div>
  );
}

const adminStyles: { [key: string]: React.CSSProperties } = {
  overlay: { position: 'fixed', top: 0, left: 0, right: 0, bottom: 0, backgroundColor: 'rgba(13, 8, 17, 0.95)', backdropFilter: 'blur(8px)', display: 'flex', justifyContent: 'center', alignItems: 'flex-start', padding: '20px', zIndex: 2000, overflowY: 'auto' },
  panel: { background: 'linear-gradient(180deg, #2D1F3D 0%, #1A0F1E 100%)', borderRadius: '16px', width: '100%', maxWidth: '1200px', maxHeight: '90vh', overflow: 'auto', position: 'relative', border: '1px solid rgba(74, 44, 90, 0.4)', boxShadow: '0 24px 48px rgba(0, 0, 0, 0.5)' },
  header: { display: 'flex', justifyContent: 'space-between', alignItems: 'center', padding: '20px 24px', borderBottom: '1px solid rgba(74, 44, 90, 0.4)' },
  title: { color: '#D4A84B', margin: 0, fontSize: '1.5rem', fontWeight: 600 },
  closeBtn: { background: 'none', border: 'none', color: '#B8B4C8', fontSize: '28px', cursor: 'pointer', transition: 'color 0.2s' },
  tabs: { display: 'flex', borderBottom: '1px solid rgba(74, 44, 90, 0.4)', padding: '0 20px', flexWrap: 'wrap' as const },
  tab: { padding: '14px 22px', backgroundColor: 'transparent', border: 'none', color: '#B8B4C8', fontSize: '0.95rem', cursor: 'pointer', borderBottom: '3px solid transparent', marginBottom: '-1px', fontWeight: 500, transition: 'all 0.2s' },
  tabActive: { color: '#D4A84B', borderBottomColor: '#D4A84B' },
  searchBar: { padding: '16px 20px' },
  searchInput: { width: '100%', padding: '12px 16px', backgroundColor: 'rgba(13, 8, 17, 0.8)', border: '1px solid rgba(74, 44, 90, 0.5)', borderRadius: '10px', color: '#F0EDF4', fontSize: '1rem', boxSizing: 'border-box' as const, transition: 'border-color 0.2s' },
  loading: { padding: '40px', textAlign: 'center', color: '#D4A84B' },
  tableContainer: { overflowX: 'auto', padding: '0 20px' },
  table: { width: '100%', borderCollapse: 'collapse' },
  th: { padding: '14px', textAlign: 'left', borderBottom: '2px solid rgba(74, 44, 90, 0.5)', color: '#D4A84B', fontSize: '0.8rem', textTransform: 'uppercase', letterSpacing: '0.03em', fontWeight: 600 },
  tr: { borderBottom: '1px solid rgba(74, 44, 90, 0.3)', transition: 'background 0.2s' },
  td: { padding: '14px', color: '#F0EDF4' },
  adminBadge: { color: '#D4A84B' },
  activeBadge: { background: 'rgba(74, 222, 128, 0.15)', color: '#4ade80', padding: '4px 10px', borderRadius: '6px', fontSize: '0.8rem', border: '1px solid rgba(74, 222, 128, 0.3)' },
  inactiveBadge: { background: 'rgba(248, 113, 113, 0.15)', color: '#f87171', padding: '4px 10px', borderRadius: '6px', fontSize: '0.8rem', border: '1px solid rgba(248, 113, 113, 0.3)' },
  actionBtn: { background: 'none', border: 'none', cursor: 'pointer', fontSize: '1.1rem', padding: '4px 8px', transition: 'transform 0.2s' },
  pagination: { display: 'flex', justifyContent: 'center', alignItems: 'center', gap: '20px', padding: '20px' },
  pageBtn: { padding: '10px 18px', background: 'rgba(74, 44, 90, 0.4)', border: 'none', borderRadius: '8px', color: '#F0EDF4', cursor: 'pointer', transition: 'all 0.2s' },
  pageInfo: { color: '#B8B4C8' },
  editModal: { position: 'absolute', top: '50%', left: '50%', transform: 'translate(-50%, -50%)', background: 'linear-gradient(180deg, #2D1F3D 0%, #1A0F1E 100%)', padding: '28px', borderRadius: '16px', border: '1px solid rgba(212, 168, 75, 0.3)', minWidth: '400px', zIndex: 10, boxShadow: '0 24px 48px rgba(0, 0, 0, 0.5)' },
  editTitle: { color: '#D4A84B', marginTop: 0, fontWeight: 600 },
  formGroup: { marginBottom: '16px' },
  label: { display: 'block', color: '#B8B4C8', marginBottom: '6px', fontSize: '0.9rem', fontWeight: 500 },
  formInput: { width: '100%', padding: '12px 14px', backgroundColor: 'rgba(13, 8, 17, 0.8)', border: '1px solid rgba(74, 44, 90, 0.5)', borderRadius: '10px', color: '#F0EDF4', boxSizing: 'border-box' as const, transition: 'border-color 0.2s' },
  checkboxLabel: { display: 'inline-flex', alignItems: 'center', gap: '8px', color: '#F0EDF4', marginRight: '20px' },
  editActions: { display: 'flex', gap: '12px', marginTop: '24px' },
  cancelBtn: { flex: 1, padding: '12px', backgroundColor: 'transparent', border: '1px solid rgba(74, 44, 90, 0.5)', borderRadius: '10px', color: '#B8B4C8', cursor: 'pointer', transition: 'all 0.2s' },
  saveBtn: { flex: 1, padding: '12px', background: 'linear-gradient(135deg, #D4A84B 0%, #B8923A 100%)', border: 'none', borderRadius: '10px', color: '#1A0F1E', fontWeight: 600, cursor: 'pointer', boxShadow: '0 2px 8px rgba(212, 168, 75, 0.3)' },
  detailModal: { position: 'absolute', top: '10px', right: '10px', bottom: '10px', width: '400px', background: 'linear-gradient(180deg, rgba(45, 31, 61, 0.95) 0%, rgba(26, 15, 30, 0.98) 100%)', borderRadius: '12px', border: '1px solid rgba(74, 44, 90, 0.4)', padding: '20px', overflowY: 'auto' },
  closeDetailBtn: { position: 'absolute', top: '10px', right: '10px', background: 'none', border: 'none', color: '#B8B4C8', fontSize: '24px', cursor: 'pointer' },
  detailTitle: { color: '#D4A84B', marginTop: 0, fontWeight: 600 },
  detailCard: { background: 'rgba(13, 8, 17, 0.6)', padding: '16px', borderRadius: '10px', marginBottom: '15px', border: '1px solid rgba(74, 44, 90, 0.3)' },
  subTitle: { color: '#D4A84B', marginTop: '20px', marginBottom: '10px', fontWeight: 600 },
  minerRow: { display: 'flex', justifyContent: 'space-between', padding: '10px', borderBottom: '1px solid rgba(74, 44, 90, 0.3)', background: 'rgba(45, 31, 61, 0.3)', marginBottom: '5px', borderRadius: '6px' },
  algorithmContainer: { padding: '20px' },
  algoHeader: { marginBottom: '25px' },
  algoTitle: { color: '#D4A84B', marginTop: 0, marginBottom: '10px', fontWeight: 600 },
  algoDesc: { color: '#B8B4C8', margin: 0 },
  algoGrid: { display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(280px, 1fr))', gap: '20px', marginBottom: '25px' },
  algoCard: { background: 'rgba(13, 8, 17, 0.6)', padding: '20px', borderRadius: '12px', border: '1px solid rgba(74, 44, 90, 0.4)' },
  algoLabel: { display: 'block', color: '#D4A84B', marginBottom: '10px', fontSize: '0.85rem', textTransform: 'uppercase', letterSpacing: '0.03em', fontWeight: 500 },
  algoSelect: { width: '100%', padding: '12px 14px', backgroundColor: 'rgba(13, 8, 17, 0.8)', border: '1px solid rgba(74, 44, 90, 0.5)', borderRadius: '10px', color: '#F0EDF4', fontSize: '1rem' },
  algoInput: { width: '100%', padding: '12px 14px', backgroundColor: 'rgba(13, 8, 17, 0.8)', border: '1px solid rgba(74, 44, 90, 0.5)', borderRadius: '10px', color: '#F0EDF4', fontSize: '1rem', boxSizing: 'border-box' as const },
  algoTextarea: { width: '100%', padding: '12px 14px', backgroundColor: 'rgba(13, 8, 17, 0.8)', border: '1px solid rgba(74, 44, 90, 0.5)', borderRadius: '10px', color: '#F0EDF4', fontSize: '0.9rem', fontFamily: 'monospace', resize: 'vertical' as const, boxSizing: 'border-box' as const },
  algoHint: { color: '#7A7490', fontSize: '0.85rem', margin: '8px 0 0' },
  algoActions: { textAlign: 'center', marginBottom: '20px' },
  algoSaveBtn: { padding: '14px 40px', background: 'linear-gradient(135deg, #D4A84B 0%, #B8923A 100%)', border: 'none', borderRadius: '10px', color: '#1A0F1E', fontSize: '1.05rem', fontWeight: 600, cursor: 'pointer', boxShadow: '0 2px 12px rgba(212, 168, 75, 0.3)' },
  algoWarning: { background: 'rgba(212, 168, 75, 0.1)', border: '1px solid rgba(212, 168, 75, 0.3)', borderRadius: '10px', padding: '16px', color: '#D4A84B', fontSize: '0.9rem', lineHeight: '1.5' },
};

export default AdminPanel;
