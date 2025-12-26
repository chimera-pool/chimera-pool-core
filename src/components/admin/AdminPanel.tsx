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
        {activeTab === 'users' && (
          <>
            <div style={adminStyles.searchBar}>
              <input style={adminStyles.searchInput} type="text" placeholder="Search users by username or email..." value={search} onChange={e => { setSearch(e.target.value); setPage(1); }} />
            </div>

            {loading ? (
          <div style={adminStyles.loading}>Loading users...</div>
        ) : (
          <>
            <div style={adminStyles.tableContainer} className="admin-table-container">
              <table style={adminStyles.table}>
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
                    <th style={adminStyles.th}>Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {users.map(user => (
                    <tr key={user.id} style={adminStyles.tr}>
                      <td style={{...adminStyles.td, color: '#D4A84B', fontWeight: 600, fontFamily: 'monospace'}}>{user.id}</td>
                      <td style={adminStyles.td}><span style={user.is_admin ? adminStyles.adminBadge : {}}>{user.username} {user.is_admin && '👑'}</span></td>
                      <td style={adminStyles.td}>{user.email}</td>
                      <td style={adminStyles.td}>
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
                      <td style={adminStyles.td}>{formatHashrate(user.total_hashrate)}</td>
                      <td style={adminStyles.td}>{user.total_earnings.toFixed(4)}</td>
                      <td style={adminStyles.td}>{user.pool_fee_percent || 'Default'}</td>
                      <td style={adminStyles.td}><span style={user.is_active ? adminStyles.activeBadge : adminStyles.inactiveBadge}>{user.is_active ? 'Active' : 'Inactive'}</span></td>
                      <td style={adminStyles.td}>
                        <button style={adminStyles.actionBtn} onClick={() => fetchUserDetail(user.id)} title="View Details">👁️</button>
                        <button style={adminStyles.actionBtn} onClick={() => handleEditUser(user)} title="Edit User">✏️</button>
                        <button style={{...adminStyles.actionBtn, backgroundColor: '#2a1a4a', borderColor: '#8b5cf6'}} onClick={() => setRoleChangeUser(user)} title="Change Role">👑</button>
                        <button style={{...adminStyles.actionBtn, opacity: 0.7}} onClick={() => handleDeleteUser(user.id)} title="Delete User">🗑️</button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>

            <div style={adminStyles.pagination}>
              <button style={adminStyles.pageBtn} disabled={page <= 1} onClick={() => setPage(p => p - 1)}>← Prev</button>
              <span style={adminStyles.pageInfo}>Page {page} of {totalPages} ({totalCount} users)</span>
              <button style={adminStyles.pageBtn} disabled={page >= totalPages} onClick={() => setPage(p => p + 1)}>Next →</button>
            </div>
          </>
        )}

        {editingUser && (
          <div style={adminStyles.editModal}>
            <h3 style={adminStyles.editTitle}>Edit User: {editingUser.username}</h3>
            <div style={adminStyles.formGroup}>
              <label style={adminStyles.label}>Pool Fee % (leave empty for default)</label>
              <input style={adminStyles.formInput} type="number" min="0" max="100" step="0.1" placeholder="e.g., 1.5" value={editForm.pool_fee_percent} onChange={e => setEditForm({...editForm, pool_fee_percent: e.target.value})} />
            </div>
            <div style={adminStyles.formGroup}>
              <label style={adminStyles.label}>Payout Address</label>
              <input style={adminStyles.formInput} type="text" placeholder="0x..." value={editForm.payout_address} onChange={e => setEditForm({...editForm, payout_address: e.target.value})} />
            </div>
            <div style={adminStyles.formGroup}>
              <label style={adminStyles.checkboxLabel}><input type="checkbox" checked={editForm.is_active} onChange={e => setEditForm({...editForm, is_active: e.target.checked})} /> Active</label>
              <label style={adminStyles.checkboxLabel}><input type="checkbox" checked={editForm.is_admin} onChange={e => setEditForm({...editForm, is_admin: e.target.checked})} /> Admin</label>
            </div>
            <div style={adminStyles.editActions}>
              <button style={adminStyles.cancelBtn} onClick={() => setEditingUser(null)}>Cancel</button>
              <button style={adminStyles.saveBtn} onClick={handleSaveUser}>Save Changes</button>
            </div>
          </div>
        )}

        {selectedUser && (
          <div style={adminStyles.detailModal}>
            <button style={adminStyles.closeDetailBtn} onClick={() => setSelectedUser(null)}>×</button>
            <h3 style={adminStyles.detailTitle}>User Details: {selectedUser.user.username}</h3>
            <div style={adminStyles.detailCard}>
              <p><strong>Email:</strong> {selectedUser.user.email}</p>
              <p><strong>Payout Address:</strong> {selectedUser.user.payout_address || 'Not set'}</p>
              <p><strong>Pool Fee:</strong> {selectedUser.user.pool_fee_percent || 'Default'}%</p>
              <p><strong>Total Earnings:</strong> {selectedUser.user.total_earnings}</p>
              <p><strong>Pending Payout:</strong> {selectedUser.user.pending_payout}</p>
              <p><strong>Blocks Found:</strong> {selectedUser.user.blocks_found}</p>
            </div>

            {/* Wallet Configuration Section */}
            <h4 style={adminStyles.subTitle}>💰 Wallet Configuration ({selectedUser.wallets?.length || 0} wallets)</h4>
            {selectedUser.wallet_summary && (
              <div style={{ ...adminStyles.detailCard, backgroundColor: '#0a1520', borderColor: '#00d4ff' }}>
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
            {selectedUser.wallets?.length > 0 ? (
              <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
                {selectedUser.wallets.map((wallet: any) => (
                  <div key={wallet.id} style={{ 
                    ...adminStyles.detailCard, 
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
              <div style={{ ...adminStyles.detailCard, textAlign: 'center', color: '#666' }}>
                <p>No wallets configured. User is using legacy payout address.</p>
              </div>
            )}

            <h4 style={adminStyles.subTitle}>Share Statistics</h4>
            <div style={adminStyles.detailCard}>
              <p><strong>Total Shares:</strong> {selectedUser.shares_stats.total_shares}</p>
              <p><strong>Valid:</strong> {selectedUser.shares_stats.valid_shares}</p>
              <p><strong>Invalid:</strong> {selectedUser.shares_stats.invalid_shares}</p>
              <p><strong>Last 24h:</strong> {selectedUser.shares_stats.last_24_hours}</p>
            </div>
            <h4 style={adminStyles.subTitle}>Miners ({selectedUser.miners?.length || 0})</h4>
            {selectedUser.miners?.map((m: any) => (
              <div key={m.id} style={adminStyles.minerRow}>
                <span>{m.name}</span>
                <span>{formatHashrate(m.hashrate)}</span>
                <span style={m.is_active ? adminStyles.activeBadge : adminStyles.inactiveBadge}>{m.is_active ? 'Online' : 'Offline'}</span>
              </div>
            ))}
          </div>
        )}

        {/* Role Change Modal (accessible from Users tab) */}
        {roleChangeUser && (
          <div style={adminStyles.editModal}>
            <h3 style={adminStyles.editTitle}>👑 Change Role: {roleChangeUser.username}</h3>
            <p style={{ color: '#888', marginBottom: '15px' }}>
              Current role: <strong style={{ color: '#00d4ff' }}>{roleChangeUser.role || 'user'}</strong>
            </p>
            <div style={adminStyles.formGroup}>
              <label style={adminStyles.label}>New Role</label>
              <select 
                style={adminStyles.algoSelect} 
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
            <div style={adminStyles.editActions}>
              <button style={adminStyles.cancelBtn} onClick={() => { setRoleChangeUser(null); setNewRole(''); }}>Cancel</button>
              <button style={adminStyles.saveBtn} onClick={handleChangeRole} disabled={!newRole}>Change Role</button>
            </div>
          </div>
        )}
          </>
        )}

        {/* Pool Statistics Tab - Isolated Component (prevents parent re-renders) */}
        {activeTab === 'stats' && <AdminGrafanaSection token={token} />}
        <AdminStatsTab token={token} isActive={activeTab === 'stats'} />

        {/* Algorithm Settings Tab */}
        {activeTab === 'algorithm' && (
          <div style={adminStyles.algorithmContainer}>
            <div style={adminStyles.algoHeader}>
              <h3 style={adminStyles.algoTitle}>⚙️ Mining Algorithm Configuration</h3>
              <p style={adminStyles.algoDesc}>
                Configure the mining algorithm for the pool. BlockDAG uses a custom Scrpy-variant algorithm.
                When BlockDAG releases their official algorithm specification, paste it below.
              </p>
            </div>

            {/* Custom Algorithm Notice */}
            <div style={{ backgroundColor: '#1a2a3a', padding: '20px', borderRadius: '12px', marginBottom: '25px', border: '2px solid #00d4ff' }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: '12px', marginBottom: '12px' }}>
                <span style={{ fontSize: '1.5rem' }}>🔷</span>
                <h4 style={{ color: '#00d4ff', margin: 0 }}>BlockDAG Custom Algorithm</h4>
              </div>
              <p style={{ color: '#b0b0b0', margin: 0, lineHeight: '1.6' }}>
                This pool supports BlockDAG's proprietary Scrpy-variant algorithm. When BlockDAG releases 
                the official algorithm specification or updates, use the <strong>"Custom Algorithm Code"</strong> section 
                below to paste the algorithm definition. The pool will automatically apply the new algorithm.
              </p>
            </div>

            <div style={adminStyles.algoGrid}>
              <div style={adminStyles.algoCard}>
                <label style={adminStyles.algoLabel}>Algorithm Type</label>
                <select 
                  style={adminStyles.algoSelect}
                  value={algorithmForm.algorithm}
                  onChange={e => setAlgorithmForm({...algorithmForm, algorithm: e.target.value})}
                >
                  <option value="scrpy-variant">Scrpy-Variant (BlockDAG Custom)</option>
                  <option value="scrypt">Scrypt</option>
                  <option value="sha256">SHA-256</option>
                  <option value="blake3">Blake3</option>
                  <option value="ethash">Ethash</option>
                  <option value="kawpow">KawPow</option>
                  <option value="custom">Custom (Define Below)</option>
                  {algorithmData?.supported_algorithms?.map((algo: any) => (
                    <option key={algo.id} value={algo.id}>{algo.name}</option>
                  ))}
                </select>
                <p style={adminStyles.algoHint}>
                  Select "Custom" to define a new algorithm from BlockDAG specifications
                </p>
              </div>

              <div style={adminStyles.algoCard}>
                <label style={adminStyles.algoLabel}>Algorithm Variant / Version</label>
                <input 
                  style={adminStyles.algoInput}
                  type="text"
                  value={algorithmForm.algorithm_variant}
                  onChange={e => setAlgorithmForm({...algorithmForm, algorithm_variant: e.target.value})}
                  placeholder="e.g., scrpy-v1.0, blockdag-mainnet"
                />
                <p style={adminStyles.algoHint}>Version identifier for the algorithm variant</p>
              </div>

              <div style={adminStyles.algoCard}>
                <label style={adminStyles.algoLabel}>Base Difficulty</label>
                <input 
                  style={adminStyles.algoInput}
                  type="text"
                  value={algorithmForm.difficulty_target}
                  onChange={e => setAlgorithmForm({...algorithmForm, difficulty_target: e.target.value})}
                  placeholder="e.g., 1.0"
                />
                <p style={adminStyles.algoHint}>Starting difficulty for share validation</p>
              </div>

              <div style={adminStyles.algoCard}>
                <label style={adminStyles.algoLabel}>Target Block Time (seconds)</label>
                <input 
                  style={adminStyles.algoInput}
                  type="text"
                  value={algorithmForm.block_time}
                  onChange={e => setAlgorithmForm({...algorithmForm, block_time: e.target.value})}
                  placeholder="e.g., 10"
                />
                <p style={adminStyles.algoHint}>Expected time between blocks</p>
              </div>

              <div style={adminStyles.algoCard}>
                <label style={adminStyles.algoLabel}>Stratum Port</label>
                <input 
                  style={adminStyles.algoInput}
                  type="text"
                  value={algorithmForm.stratum_port}
                  onChange={e => setAlgorithmForm({...algorithmForm, stratum_port: e.target.value})}
                  placeholder="e.g., 3333"
                />
                <p style={adminStyles.algoHint}>Port for miner connections</p>
              </div>

              <div style={{...adminStyles.algoCard, gridColumn: '1 / -1'}}>
                <label style={adminStyles.algoLabel}>Algorithm Parameters (JSON)</label>
                <textarea 
                  style={adminStyles.algoTextarea}
                  value={algorithmForm.algorithm_params}
                  onChange={e => setAlgorithmForm({...algorithmForm, algorithm_params: e.target.value})}
                  placeholder='{"N": 1024, "r": 1, "p": 1, "keyLen": 32}'
                  rows={4}
                />
                <p style={adminStyles.algoHint}>Scrypt parameters: N (CPU/memory cost), r (block size), p (parallelization), keyLen (output length)</p>
              </div>
            </div>

            {/* Custom Algorithm Code Section */}
            <div style={{ backgroundColor: '#0a1015', padding: '25px', borderRadius: '12px', marginTop: '25px', border: '2px dashed #9b59b6' }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: '12px', marginBottom: '15px' }}>
                <span style={{ fontSize: '1.5rem' }}>📝</span>
                <h4 style={{ color: '#9b59b6', margin: 0 }}>Custom Algorithm Code (BlockDAG Official)</h4>
              </div>
              <p style={{ color: '#888', marginBottom: '15px', lineHeight: '1.6' }}>
                When BlockDAG releases their official algorithm specification, paste the complete algorithm definition below.
                This supports Go code, JSON configuration, or algorithm pseudocode that will be compiled into the mining validator.
              </p>
              
              <div style={{ marginBottom: '15px' }}>
                <label style={{ display: 'block', color: '#00d4ff', marginBottom: '8px', fontSize: '0.9rem', textTransform: 'uppercase' }}>
                  Algorithm Name / Identifier
                </label>
                <input 
                  style={{ ...adminStyles.algoInput, backgroundColor: '#1a1a2e', border: '1px solid #9b59b6' }}
                  type="text"
                  placeholder="e.g., blockdag-scrpy-v2, bdag-mainnet-algo"
                  defaultValue="scrpy-variant"
                />
              </div>

              <div style={{ marginBottom: '15px' }}>
                <label style={{ display: 'block', color: '#00d4ff', marginBottom: '8px', fontSize: '0.9rem', textTransform: 'uppercase' }}>
                  Custom Algorithm Code / Specification
                </label>
                <textarea 
                  style={{ 
                    width: '100%', 
                    minHeight: '300px', 
                    backgroundColor: '#0a0a15', 
                    border: '1px solid #9b59b6', 
                    borderRadius: '8px', 
                    color: '#00ff88', 
                    fontFamily: 'monospace', 
                    fontSize: '0.9rem', 
                    padding: '15px',
                    boxSizing: 'border-box',
                    lineHeight: '1.6',
                    resize: 'vertical'
                  }}
                  placeholder={`// Paste BlockDAG's official algorithm specification here
// Example format:

{
  "algorithm": "scrpy-variant",
  "version": "1.0.0",
  "parameters": {
    "N": 1024,
    "r": 1,
    "p": 1,
    "keyLen": 32,
    "salt": "BlockDAG",
    "hashFunction": "sha256"
  },
  "validation": {
    "targetBits": 24,
    "difficultyAdjustment": "DAA",
    "blockTimeTarget": 10
  },
  "customCode": "// Go implementation or pseudocode"
}`}
                />
              </div>

              <div style={{ display: 'flex', gap: '10px', flexWrap: 'wrap' }}>
                <button 
                  style={{ 
                    padding: '12px 24px', 
                    backgroundColor: '#9b59b6', 
                    border: 'none', 
                    borderRadius: '8px', 
                    color: '#fff', 
                    fontWeight: 'bold', 
                    cursor: 'pointer',
                    display: 'flex',
                    alignItems: 'center',
                    gap: '8px'
                  }}
                >
                  ✅ Validate Algorithm
                </button>
                <button 
                  style={{ 
                    padding: '12px 24px', 
                    backgroundColor: '#1a4d4d', 
                    border: '1px solid #4ade80', 
                    borderRadius: '8px', 
                    color: '#4ade80', 
                    fontWeight: 'bold', 
                    cursor: 'pointer',
                    display: 'flex',
                    alignItems: 'center',
                    gap: '8px'
                  }}
                >
                  🧪 Test with Sample Block
                </button>
                <button 
                  style={{ 
                    padding: '12px 24px', 
                    backgroundColor: 'transparent', 
                    border: '1px solid #888', 
                    borderRadius: '8px', 
                    color: '#888', 
                    cursor: 'pointer' 
                  }}
                >
                  📋 Load from Clipboard
                </button>
              </div>

              <div style={{ marginTop: '15px', padding: '12px', backgroundColor: '#1a2a1a', borderRadius: '6px', border: '1px solid #4ade80' }}>
                <p style={{ margin: 0, color: '#4ade80', fontSize: '0.9rem' }}>
                  💡 <strong>Tip:</strong> After pasting the algorithm, click "Validate Algorithm" to check for syntax errors, 
                  then "Test with Sample Block" to verify it produces valid hashes before saving.
                </p>
              </div>
            </div>

            <div style={adminStyles.algoActions}>
              <button 
                style={adminStyles.algoSaveBtn}
                onClick={handleSaveAlgorithm}
                disabled={savingAlgorithm}
              >
                {savingAlgorithm ? 'Saving...' : '💾 Save Algorithm Settings'}
              </button>
            </div>

            <div style={adminStyles.algoWarning}>
              <strong>⚠️ Important:</strong> After changing algorithm settings, you may need to restart the stratum server 
              for changes to take effect. Notify miners before making algorithm changes as they may need to update their mining software.
            </div>

            {/* Hardware Difficulty Tiers Info */}
            <div style={{ backgroundColor: '#0a0a15', padding: '20px', borderRadius: '12px', marginTop: '20px', border: '1px solid #2a2a4a' }}>
              <h4 style={{ color: '#00d4ff', margin: '0 0 15px' }}>🖥️ Hardware-Aware Difficulty Tiers</h4>
              <p style={{ color: '#888', marginBottom: '15px', fontSize: '0.9rem' }}>
                The pool automatically detects miner hardware and applies appropriate difficulty levels:
              </p>
              <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(150px, 1fr))', gap: '10px' }}>
                <div style={{ backgroundColor: '#1a1a2e', padding: '12px', borderRadius: '8px', textAlign: 'center' }}>
                  <span style={{ display: 'block', fontSize: '1.2rem', marginBottom: '5px' }}>💻</span>
                  <span style={{ color: '#888', fontSize: '0.8rem' }}>CPU</span>
                  <span style={{ display: 'block', color: '#00d4ff', fontWeight: 'bold' }}>Base × 1</span>
                </div>
                <div style={{ backgroundColor: '#1a1a2e', padding: '12px', borderRadius: '8px', textAlign: 'center' }}>
                  <span style={{ display: 'block', fontSize: '1.2rem', marginBottom: '5px' }}>🎮</span>
                  <span style={{ color: '#888', fontSize: '0.8rem' }}>GPU</span>
                  <span style={{ display: 'block', color: '#00d4ff', fontWeight: 'bold' }}>Base × 16</span>
                </div>
                <div style={{ backgroundColor: '#1a1a2e', padding: '12px', borderRadius: '8px', textAlign: 'center' }}>
                  <span style={{ display: 'block', fontSize: '1.2rem', marginBottom: '5px' }}>🔧</span>
                  <span style={{ color: '#888', fontSize: '0.8rem' }}>FPGA</span>
                  <span style={{ display: 'block', color: '#00d4ff', fontWeight: 'bold' }}>Base × 64</span>
                </div>
                <div style={{ backgroundColor: '#1a1a2e', padding: '12px', borderRadius: '8px', textAlign: 'center' }}>
                  <span style={{ display: 'block', fontSize: '1.2rem', marginBottom: '5px' }}>⚡</span>
                  <span style={{ color: '#888', fontSize: '0.8rem' }}>ASIC</span>
                  <span style={{ display: 'block', color: '#00d4ff', fontWeight: 'bold' }}>Base × 256</span>
                </div>
                <div style={{ backgroundColor: '#1a2a3a', padding: '12px', borderRadius: '8px', textAlign: 'center', border: '1px solid #9b59b6' }}>
                  <span style={{ display: 'block', fontSize: '1.2rem', marginBottom: '5px' }}>🔷</span>
                  <span style={{ color: '#9b59b6', fontSize: '0.8rem' }}>X30/X100</span>
                  <span style={{ display: 'block', color: '#9b59b6', fontWeight: 'bold' }}>Base × 1024</span>
                </div>
              </div>
            </div>
          </div>
        )}

        {/* Network Configuration Tab */}
        {activeTab === 'network' && (
          <div style={adminStyles.algorithmContainer}>
            <div style={adminStyles.algoHeader}>
              <h3 style={adminStyles.algoTitle}>🌐 Network Configuration</h3>
              <p style={adminStyles.algoDesc}>
                Configure blockchain networks for mining. Switch between networks to mine different cryptocurrencies.
              </p>
            </div>

            {networksLoading ? (
              <div style={adminStyles.loading}>Loading networks...</div>
            ) : (
              <>
                {/* Active Network Card */}
                {activeNetwork && (
                  <div style={{ backgroundColor: '#0a2a1a', padding: '20px', borderRadius: '12px', marginBottom: '25px', border: '2px solid #4ade80' }}>
                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '15px' }}>
                      <h4 style={{ color: '#4ade80', margin: 0, display: 'flex', alignItems: 'center', gap: '10px' }}>
                        <span style={{ fontSize: '1.5rem' }}>✅</span> Active Network: {activeNetwork.display_name}
                      </h4>
                      <span style={{ backgroundColor: '#4ade80', color: '#0a2a1a', padding: '4px 12px', borderRadius: '20px', fontSize: '0.85rem', fontWeight: 'bold' }}>
                        {activeNetwork.symbol}
                      </span>
                    </div>
                    <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))', gap: '15px' }}>
                      <div><span style={{ color: '#888' }}>Algorithm:</span> <span style={{ color: '#e0e0e0' }}>{activeNetwork.algorithm}</span></div>
                      <div><span style={{ color: '#888' }}>Stratum Port:</span> <span style={{ color: '#e0e0e0' }}>{activeNetwork.stratum_port}</span></div>
                      <div><span style={{ color: '#888' }}>Pool Fee:</span> <span style={{ color: '#e0e0e0' }}>{activeNetwork.pool_fee_percent}%</span></div>
                      <div><span style={{ color: '#888' }}>Min Payout:</span> <span style={{ color: '#e0e0e0' }}>{activeNetwork.min_payout_threshold} {activeNetwork.symbol}</span></div>
                      <div style={{ gridColumn: '1 / -1' }}><span style={{ color: '#888' }}>RPC URL:</span> <span style={{ color: '#00d4ff', fontFamily: 'monospace', fontSize: '0.9rem' }}>{activeNetwork.rpc_url}</span></div>
                      <div style={{ gridColumn: '1 / -1' }}><span style={{ color: '#888' }}>Wallet:</span> <span style={{ color: '#f59e0b', fontFamily: 'monospace', fontSize: '0.85rem' }}>{activeNetwork.pool_wallet_address || 'Not configured'}</span></div>
                    </div>
                  </div>
                )}

                {/* Switch Network Section */}
                <div style={{ backgroundColor: '#1a1a2e', padding: '20px', borderRadius: '12px', marginBottom: '25px', border: '1px solid #2a2a4a' }}>
                  <h4 style={{ color: '#9b59b6', margin: '0 0 15px' }}>🔄 Switch Mining Network</h4>
                  <div style={{ marginBottom: '15px' }}>
                    <input 
                      style={{ ...adminStyles.algoInput, marginBottom: '10px' }}
                      type="text"
                      placeholder="Reason for switch (optional)"
                      value={switchReason}
                      onChange={e => setSwitchReason(e.target.value)}
                    />
                  </div>
                  <div style={{ display: 'flex', flexWrap: 'wrap', gap: '10px' }}>
                    {networks.map(network => (
                      <button
                        key={network.id}
                        style={{
                          padding: '12px 20px',
                          backgroundColor: network.is_default ? '#4ade80' : (network.is_active ? '#1a3a3a' : '#2a2a4a'),
                          border: network.is_default ? 'none' : '1px solid #4a4a6a',
                          borderRadius: '8px',
                          color: network.is_default ? '#0a2a1a' : '#e0e0e0',
                          cursor: network.is_default ? 'default' : 'pointer',
                          fontWeight: network.is_default ? 'bold' : 'normal',
                          display: 'flex',
                          alignItems: 'center',
                          gap: '8px'
                        }}
                        onClick={() => !network.is_default && handleSwitchNetwork(network.name)}
                        disabled={network.is_default}
                      >
                        <span>{network.is_default ? '✅' : '🔘'}</span>
                        {network.display_name} ({network.symbol})
                        <span style={{ fontSize: '0.8rem', opacity: 0.7 }}>{network.algorithm}</span>
                      </button>
                    ))}
                  </div>
                </div>

                {/* All Networks List */}
                <div style={{ backgroundColor: '#0a0a15', padding: '20px', borderRadius: '12px', border: '1px solid #2a2a4a' }}>
                  <h4 style={{ color: '#00d4ff', margin: '0 0 20px' }}>📋 Configured Networks</h4>
                  <div style={{ display: 'flex', flexDirection: 'column', gap: '15px' }}>
                    {networks.map(network => (
                      <div key={network.id} style={{ backgroundColor: '#1a1a2e', padding: '20px', borderRadius: '10px', border: `1px solid ${network.is_default ? '#4ade80' : '#2a2a4a'}` }}>
                        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: '15px' }}>
                          <div>
                            <h5 style={{ color: '#e0e0e0', margin: '0 0 5px', display: 'flex', alignItems: 'center', gap: '10px' }}>
                              {network.display_name}
                              <span style={{ backgroundColor: '#2a2a4a', padding: '2px 8px', borderRadius: '4px', fontSize: '0.8rem', color: '#00d4ff' }}>{network.symbol}</span>
                              {network.is_default && <span style={{ backgroundColor: '#4ade80', color: '#0a2a1a', padding: '2px 8px', borderRadius: '4px', fontSize: '0.75rem' }}>ACTIVE</span>}
                            </h5>
                            <p style={{ color: '#888', margin: 0, fontSize: '0.9rem' }}>{network.description || 'No description'}</p>
                          </div>
                          <div style={{ display: 'flex', gap: '8px' }}>
                            <button
                              style={{ padding: '6px 12px', backgroundColor: '#2a2a4a', border: 'none', borderRadius: '4px', color: '#00d4ff', cursor: 'pointer', fontSize: '0.85rem' }}
                              onClick={() => handleTestConnection(network.id)}
                            >
                              🔗 Test
                            </button>
                            <button
                              style={{ padding: '6px 12px', backgroundColor: '#2a2a4a', border: 'none', borderRadius: '4px', color: '#f59e0b', cursor: 'pointer', fontSize: '0.85rem' }}
                              onClick={() => {
                                setEditingNetwork(network);
                                setNetworkForm({
                                  name: network.name,
                                  symbol: network.symbol,
                                  display_name: network.display_name,
                                  algorithm: network.algorithm,
                                  rpc_url: network.rpc_url,
                                  rpc_user: network.rpc_user || '',
                                  rpc_password: '',
                                  pool_wallet_address: network.pool_wallet_address,
                                  stratum_port: String(network.stratum_port),
                                  block_time_target: String(network.block_time_target),
                                  pool_fee_percent: String(network.pool_fee_percent),
                                  min_payout_threshold: String(network.min_payout_threshold),
                                  network_type: network.network_type,
                                  description: network.description || ''
                                });
                              }}
                            >
                              ✏️ Edit
                            </button>
                          </div>
                        </div>
                        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(180px, 1fr))', gap: '10px', fontSize: '0.9rem' }}>
                          <div><span style={{ color: '#666' }}>Algorithm:</span> <span style={{ color: '#9b59b6' }}>{network.algorithm}</span></div>
                          <div><span style={{ color: '#666' }}>Port:</span> <span style={{ color: '#e0e0e0' }}>{network.stratum_port}</span></div>
                          <div><span style={{ color: '#666' }}>Fee:</span> <span style={{ color: '#e0e0e0' }}>{network.pool_fee_percent}%</span></div>
                          <div><span style={{ color: '#666' }}>Type:</span> <span style={{ color: '#e0e0e0' }}>{network.network_type}</span></div>
                        </div>
                      </div>
                    ))}
                  </div>
                </div>

                {/* Edit Network Modal */}
                {editingNetwork && (
                  <div style={{ position: 'fixed', top: 0, left: 0, right: 0, bottom: 0, backgroundColor: 'rgba(0,0,0,0.8)', display: 'flex', justifyContent: 'center', alignItems: 'center', zIndex: 3000 }}>
                    <div style={{ backgroundColor: '#1a1a2e', padding: '30px', borderRadius: '12px', width: '90%', maxWidth: '600px', maxHeight: '80vh', overflow: 'auto' }}>
                      <h3 style={{ color: '#f59e0b', margin: '0 0 20px' }}>✏️ Edit Network: {editingNetwork.display_name}</h3>
                      
                      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '15px' }}>
                        <div>
                          <label style={{ display: 'block', color: '#888', marginBottom: '5px' }}>Display Name</label>
                          <input style={adminStyles.formInput} value={networkForm.display_name} onChange={e => setNetworkForm({...networkForm, display_name: e.target.value})} />
                        </div>
                        <div>
                          <label style={{ display: 'block', color: '#888', marginBottom: '5px' }}>Stratum Port</label>
                          <input style={adminStyles.formInput} type="number" value={networkForm.stratum_port} onChange={e => setNetworkForm({...networkForm, stratum_port: e.target.value})} />
                        </div>
                        <div style={{ gridColumn: '1 / -1' }}>
                          <label style={{ display: 'block', color: '#888', marginBottom: '5px' }}>RPC URL</label>
                          <input style={adminStyles.formInput} value={networkForm.rpc_url} onChange={e => setNetworkForm({...networkForm, rpc_url: e.target.value})} placeholder="https://rpc.example.com" />
                        </div>
                        <div>
                          <label style={{ display: 'block', color: '#888', marginBottom: '5px' }}>RPC User (optional)</label>
                          <input style={adminStyles.formInput} value={networkForm.rpc_user} onChange={e => setNetworkForm({...networkForm, rpc_user: e.target.value})} />
                        </div>
                        <div>
                          <label style={{ display: 'block', color: '#888', marginBottom: '5px' }}>RPC Password (optional)</label>
                          <input style={adminStyles.formInput} type="password" value={networkForm.rpc_password} onChange={e => setNetworkForm({...networkForm, rpc_password: e.target.value})} placeholder="Leave blank to keep current" />
                        </div>
                        <div style={{ gridColumn: '1 / -1' }}>
                          <label style={{ display: 'block', color: '#888', marginBottom: '5px' }}>Pool Wallet Address</label>
                          <input style={adminStyles.formInput} value={networkForm.pool_wallet_address} onChange={e => setNetworkForm({...networkForm, pool_wallet_address: e.target.value})} />
                        </div>
                        <div>
                          <label style={{ display: 'block', color: '#888', marginBottom: '5px' }}>Pool Fee %</label>
                          <input style={adminStyles.formInput} type="number" step="0.1" value={networkForm.pool_fee_percent} onChange={e => setNetworkForm({...networkForm, pool_fee_percent: e.target.value})} />
                        </div>
                        <div>
                          <label style={{ display: 'block', color: '#888', marginBottom: '5px' }}>Min Payout Threshold</label>
                          <input style={adminStyles.formInput} type="number" step="0.001" value={networkForm.min_payout_threshold} onChange={e => setNetworkForm({...networkForm, min_payout_threshold: e.target.value})} />
                        </div>
                        <div style={{ gridColumn: '1 / -1' }}>
                          <label style={{ display: 'block', color: '#888', marginBottom: '5px' }}>Description</label>
                          <textarea style={{ ...adminStyles.formInput, resize: 'vertical' }} rows={3} value={networkForm.description} onChange={e => setNetworkForm({...networkForm, description: e.target.value})} />
                        </div>
                      </div>

                      <div style={{ display: 'flex', gap: '10px', marginTop: '20px' }}>
                        <button style={adminStyles.cancelBtn} onClick={() => setEditingNetwork(null)}>Cancel</button>
                        <button style={adminStyles.saveBtn} onClick={() => handleUpdateNetwork(editingNetwork.id)}>💾 Save Changes</button>
                      </div>
                    </div>
                  </div>
                )}

                {/* Switch History */}
                {networkHistory.length > 0 && (
                  <div style={{ backgroundColor: '#0a0a15', padding: '20px', borderRadius: '12px', marginTop: '25px', border: '1px solid #2a2a4a' }}>
                    <h4 style={{ color: '#888', margin: '0 0 15px' }}>📜 Network Switch History</h4>
                    <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
                      {networkHistory.slice(0, 5).map((h: any) => (
                        <div key={h.id} style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', padding: '10px', backgroundColor: '#1a1a2e', borderRadius: '6px' }}>
                          <span style={{ color: '#e0e0e0' }}>
                            <span style={{ color: h.status === 'completed' ? '#4ade80' : '#f59e0b' }}>●</span>{' '}
                            {h.switch_reason}
                          </span>
                          <span style={{ color: '#666', fontSize: '0.85rem' }}>{new Date(h.started_at).toLocaleString()}</span>
                        </div>
                      ))}
                    </div>
                  </div>
                )}

                {/* Litecoin Quick Setup */}
                <div style={{ backgroundColor: '#1a2a3a', padding: '20px', borderRadius: '12px', marginTop: '25px', border: '1px solid #00d4ff' }}>
                  <h4 style={{ color: '#00d4ff', margin: '0 0 15px' }}>💡 Litecoin Quick Setup Guide</h4>
                  <p style={{ color: '#888', marginBottom: '15px' }}>
                    To test your X100 ASIC on Litecoin's network:
                  </p>
                  <ol style={{ color: '#e0e0e0', paddingLeft: '20px', lineHeight: '1.8' }}>
                    <li>Edit the <strong>Litecoin</strong> network configuration above</li>
                    <li>Set <strong>RPC URL</strong> to your Litecoin node (e.g., <code style={{ backgroundColor: '#2a2a4a', padding: '2px 6px', borderRadius: '4px' }}>http://localhost:9332</code>)</li>
                    <li>Set <strong>Pool Wallet Address</strong> to your Litecoin wallet (starts with <code style={{ backgroundColor: '#2a2a4a', padding: '2px 6px', borderRadius: '4px' }}>L</code> or <code style={{ backgroundColor: '#2a2a4a', padding: '2px 6px', borderRadius: '4px' }}>ltc1</code>)</li>
                    <li>Click <strong>Test Connection</strong> to verify RPC connectivity</li>
                    <li>Click the <strong>Litecoin (LTC)</strong> button above to switch the pool</li>
                  </ol>
                </div>
              </>
            )}
          </div>
        )}

        {/* Role Management Tab */}
        {activeTab === 'roles' && (
          <div style={adminStyles.algorithmContainer}>
            <div style={adminStyles.algoHeader}>
              <h3 style={adminStyles.algoTitle}>👑 Role Management</h3>
              <p style={adminStyles.algoDesc}>
                Manage user roles and permissions. Promote users to moderators or admins.
              </p>
            </div>

            {rolesLoading ? (
              <div style={adminStyles.loading}>Loading roles...</div>
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
                  <div style={{ padding: '15px 20px', maxHeight: '300px', overflowY: 'auto' }}>
                    {admins.length === 0 ? (
                      <p style={{ color: '#666', fontStyle: 'italic', margin: 0 }}>No administrators assigned</p>
                    ) : (
                      admins.map((admin: any) => (
                        <div key={admin.id} style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', padding: '10px 12px', backgroundColor: '#1a1a2e', borderRadius: '6px', marginBottom: '8px' }}>
                          <div>
                            <span style={{ color: '#9b59b6', fontWeight: 'bold' }}>{admin.username}</span>
                            <p style={{ margin: '2px 0 0', color: '#888', fontSize: '0.8rem' }}>{admin.email}</p>
                            <span style={{ backgroundColor: admin.role === 'super_admin' ? '#4d1a4d' : '#2a2a4a', color: admin.role === 'super_admin' ? '#d946ef' : '#9b59b6', padding: '2px 6px', borderRadius: '4px', fontSize: '0.7rem' }}>
                              {admin.role === 'super_admin' ? '⭐ Super Admin' : 'Admin'}
                            </span>
                          </div>
                          <button 
                            style={{ ...adminStyles.actionBtn, color: '#fbbf24' }}
                            onClick={() => { setRoleChangeUser(admin); setNewRole('user'); }}
                            title="Change Role"
                          >
                            ✏️
                          </button>
                        </div>
                      ))
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
                  <div style={{ padding: '15px 20px', maxHeight: '300px', overflowY: 'auto' }}>
                    {moderators.length === 0 ? (
                      <p style={{ color: '#666', fontStyle: 'italic', margin: 0 }}>No moderators assigned</p>
                    ) : (
                      moderators.map((mod: any) => (
                        <div key={mod.id} style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', padding: '10px 12px', backgroundColor: '#1a1a2e', borderRadius: '6px', marginBottom: '8px' }}>
                          <div>
                            <span style={{ color: '#00d4ff', fontWeight: 'bold' }}>{mod.username}</span>
                            <p style={{ margin: '2px 0 0', color: '#888', fontSize: '0.8rem' }}>{mod.email}</p>
                          </div>
                          <button 
                            style={{ ...adminStyles.actionBtn, color: '#fbbf24' }}
                            onClick={() => { setRoleChangeUser(mod); setNewRole('user'); }}
                            title="Change Role"
                          >
                            ✏️
                          </button>
                        </div>
                      ))
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
                  style={{ ...adminStyles.algoSaveBtn, backgroundColor: '#00d4ff' }}
                  onClick={() => setActiveTab('users')}
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
              <div style={adminStyles.editModal}>
                <h3 style={adminStyles.editTitle}>Change Role: {roleChangeUser.username}</h3>
                <p style={{ color: '#888', marginBottom: '15px' }}>
                  Current role: <strong style={{ color: '#00d4ff' }}>{roleChangeUser.role || 'user'}</strong>
                </p>
                <div style={adminStyles.formGroup}>
                  <label style={adminStyles.label}>New Role</label>
                  <select 
                    style={adminStyles.algoSelect} 
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
                <div style={adminStyles.editActions}>
                  <button style={adminStyles.cancelBtn} onClick={() => { setRoleChangeUser(null); setNewRole(''); }}>Cancel</button>
                  <button style={adminStyles.saveBtn} onClick={handleChangeRole} disabled={!newRole}>Change Role</button>
                </div>
              </div>
            )}
          </div>
        )}

        {/* Bug Reports Tab */}
        {activeTab === 'bugs' && (
          <div style={adminStyles.algorithmContainer}>
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
                          <div key={att.id} style={{ backgroundColor: '#1a1a2e', padding: '10px', borderRadius: '6px', marginBottom: '8px' }}>
                            <span style={{ color: '#00d4ff' }}>{att.is_screenshot ? '📸' : '📄'} {att.original_filename}</span>
                            <span style={{ color: '#666', fontSize: '0.8rem', marginLeft: '10px' }}>({(att.file_size / 1024).toFixed(1)} KB)</span>
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
                  <div style={adminStyles.loading}>Loading bug reports...</div>
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
        )}

        {/* Miner Monitoring Tab */}
        {activeTab === 'miners' && (
          <div style={adminStyles.algorithmContainer}>
            {selectedMiner ? (
              // Miner Detail View
              <div>
                <button 
                  style={{ marginBottom: '20px', padding: '8px 16px', backgroundColor: '#2a2a4a', border: 'none', borderRadius: '6px', color: '#e0e0e0', cursor: 'pointer' }}
                  onClick={() => setSelectedMiner(null)}
                >
                  ← Back to Miners List
                </button>

                <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(300px, 1fr))', gap: '20px' }}>
                  {/* Miner Info Card */}
                  <div style={{ ...adminStyles.algoCard, borderColor: selectedMiner.is_active ? '#4ade80' : '#f87171' }}>
                    <h3 style={{ color: '#00d4ff', marginTop: 0 }}>⛏️ {selectedMiner.name}</h3>
                    <p><strong>User:</strong> {selectedMiner.username} (ID: {selectedMiner.user_id})</p>
                    <p><strong>IP Address:</strong> {selectedMiner.address || 'Unknown'}</p>
                    <p><strong>Status:</strong> <span style={selectedMiner.is_active ? adminStyles.activeBadge : adminStyles.inactiveBadge}>{selectedMiner.is_active ? '🟢 Active' : '🔴 Offline'}</span></p>
                    <p><strong>Connection:</strong> {selectedMiner.connection_duration}</p>
                    <p><strong>Uptime (24h):</strong> {selectedMiner.uptime_percent?.toFixed(1)}%</p>
                    <p><strong>Last Seen:</strong> {new Date(selectedMiner.last_seen).toLocaleString()}</p>
                  </div>

                  {/* Hashrate Card */}
                  <div style={adminStyles.algoCard}>
                    <h3 style={{ color: '#9b59b6', marginTop: 0 }}>⚡ Performance</h3>
                    <p><strong>Reported Hashrate:</strong> {formatHashrate(selectedMiner.hashrate)}</p>
                    <p><strong>Effective Hashrate:</strong> {formatHashrate(selectedMiner.performance?.effective_hashrate || 0)}</p>
                    <p><strong>Efficiency:</strong> {selectedMiner.performance?.efficiency_percent?.toFixed(1) || 0}%</p>
                    <p><strong>Shares/Min:</strong> {selectedMiner.performance?.shares_per_minute?.toFixed(2) || 0}</p>
                    <p><strong>Avg Share Time:</strong> {selectedMiner.performance?.avg_share_time_seconds?.toFixed(1) || 0}s</p>
                    <p><strong>Est. Daily Shares:</strong> {selectedMiner.performance?.estimated_daily_shares || 0}</p>
                  </div>

                  {/* Share Stats Card */}
                  <div style={adminStyles.algoCard}>
                    <h3 style={{ color: '#4ade80', marginTop: 0 }}>📊 Share Statistics</h3>
                    <p><strong>Total Shares:</strong> {selectedMiner.share_stats?.total_shares || 0}</p>
                    <p><strong>Valid:</strong> <span style={{ color: '#4ade80' }}>{selectedMiner.share_stats?.valid_shares || 0}</span></p>
                    <p><strong>Invalid:</strong> <span style={{ color: '#f87171' }}>{selectedMiner.share_stats?.invalid_shares || 0}</span></p>
                    <p><strong>Acceptance Rate:</strong> {selectedMiner.share_stats?.acceptance_rate?.toFixed(2) || 0}%</p>
                    <p><strong>Last Hour:</strong> {selectedMiner.share_stats?.last_hour || 0}</p>
                    <p><strong>Last 24h:</strong> {selectedMiner.share_stats?.last_24_hours || 0}</p>
                    <p><strong>Avg Difficulty:</strong> {selectedMiner.share_stats?.avg_difficulty?.toFixed(4) || 0}</p>
                  </div>

                  {/* Troubleshooting Card */}
                  <div style={{ ...adminStyles.algoCard, borderColor: '#f59e0b' }}>
                    <h3 style={{ color: '#f59e0b', marginTop: 0 }}>🔧 Troubleshooting</h3>
                    {selectedMiner.share_stats?.acceptance_rate < 95 && (
                      <div style={{ backgroundColor: '#4d2a1a', padding: '10px', borderRadius: '6px', marginBottom: '10px' }}>
                        <strong style={{ color: '#f87171' }}>⚠️ Low Acceptance Rate</strong>
                        <p style={{ margin: '5px 0 0', color: '#fbbf24', fontSize: '0.9rem' }}>
                          {(100 - (selectedMiner.share_stats?.acceptance_rate || 0)).toFixed(1)}% of shares are invalid. Check miner configuration.
                        </p>
                      </div>
                    )}
                    {!selectedMiner.is_active && (
                      <div style={{ backgroundColor: '#4d1a1a', padding: '10px', borderRadius: '6px', marginBottom: '10px' }}>
                        <strong style={{ color: '#f87171' }}>🔴 Miner Offline</strong>
                        <p style={{ margin: '5px 0 0', color: '#fbbf24', fontSize: '0.9rem' }}>
                          Last seen: {new Date(selectedMiner.last_seen).toLocaleString()}
                        </p>
                      </div>
                    )}
                    {selectedMiner.performance?.efficiency_percent < 80 && selectedMiner.performance?.efficiency_percent > 0 && (
                      <div style={{ backgroundColor: '#4d3a1a', padding: '10px', borderRadius: '6px', marginBottom: '10px' }}>
                        <strong style={{ color: '#fbbf24' }}>⚡ Low Efficiency</strong>
                        <p style={{ margin: '5px 0 0', color: '#fbbf24', fontSize: '0.9rem' }}>
                          Effective hashrate is only {selectedMiner.performance?.efficiency_percent?.toFixed(1)}% of reported. Possible network issues.
                        </p>
                      </div>
                    )}
                    {selectedMiner.share_stats?.acceptance_rate >= 95 && selectedMiner.is_active && (
                      <div style={{ backgroundColor: '#1a4d1a', padding: '10px', borderRadius: '6px' }}>
                        <strong style={{ color: '#4ade80' }}>✅ Miner Healthy</strong>
                        <p style={{ margin: '5px 0 0', color: '#4ade80', fontSize: '0.9rem' }}>
                          No issues detected. Miner is operating normally.
                        </p>
                      </div>
                    )}
                  </div>
                </div>

                {/* Visual Charts Section */}
                <h3 style={{ color: '#00d4ff', marginTop: '30px' }}>📊 Visual Analytics</h3>
                <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(350px, 1fr))', gap: '20px', marginBottom: '30px' }}>
                  
                  {/* Share Distribution Pie Chart */}
                  <div style={graphStyles.chartCard}>
                    <h4 style={graphStyles.chartTitle}>Share Distribution</h4>
                    <ResponsiveContainer width="100%" height={250}>
                      <PieChart>
                        <Pie
                          data={[
                            { name: 'Valid', value: selectedMiner.share_stats?.valid_shares || 0, color: '#4ade80' },
                            { name: 'Invalid', value: selectedMiner.share_stats?.invalid_shares || 0, color: '#f87171' }
                          ]}
                          cx="50%"
                          cy="50%"
                          innerRadius={60}
                          outerRadius={90}
                          paddingAngle={5}
                          dataKey="value"
                          label={({ name, percent }) => `${name}: ${(percent * 100).toFixed(1)}%`}
                        >
                          <Cell fill="#4ade80" />
                          <Cell fill="#f87171" />
                        </Pie>
                        <Tooltip 
                          contentStyle={{ backgroundColor: '#1a1a2e', border: '1px solid #2a2a4a', borderRadius: '8px' }}
                          formatter={(value: number) => [value, 'Shares']}
                        />
                        <Legend />
                      </PieChart>
                    </ResponsiveContainer>
                  </div>

                  {/* Share Timeline Bar Chart */}
                  <div style={graphStyles.chartCard}>
                    <h4 style={graphStyles.chartTitle}>Recent Share Activity</h4>
                    <ResponsiveContainer width="100%" height={250}>
                      <BarChart data={
                        (selectedMiner.recent_shares || []).slice(0, 10).reverse().map((share: any, idx: number) => ({
                          name: `#${idx + 1}`,
                          difficulty: share.difficulty,
                          valid: share.is_valid ? share.difficulty : 0,
                          invalid: !share.is_valid ? share.difficulty : 0
                        }))
                      }>
                        <CartesianGrid strokeDasharray="3 3" stroke="#2a2a4a" />
                        <XAxis dataKey="name" stroke="#888" fontSize={12} />
                        <YAxis stroke="#888" fontSize={12} />
                        <Tooltip 
                          contentStyle={{ backgroundColor: '#1a1a2e', border: '1px solid #2a2a4a', borderRadius: '8px' }}
                          formatter={(value: number) => [value.toFixed(4), 'Difficulty']}
                        />
                        <Bar dataKey="valid" stackId="a" fill="#4ade80" name="Valid" />
                        <Bar dataKey="invalid" stackId="a" fill="#f87171" name="Invalid" />
                      </BarChart>
                    </ResponsiveContainer>
                  </div>

                  {/* Performance Gauge */}
                  <div style={graphStyles.chartCard}>
                    <h4 style={graphStyles.chartTitle}>Efficiency Breakdown</h4>
                    <ResponsiveContainer width="100%" height={250}>
                      <BarChart
                        layout="vertical"
                        data={[
                          { name: 'Acceptance', value: selectedMiner.share_stats?.acceptance_rate || 0, fill: '#4ade80' },
                          { name: 'Efficiency', value: selectedMiner.performance?.efficiency_percent || 0, fill: '#00d4ff' },
                          { name: 'Uptime (24h)', value: selectedMiner.uptime_percent || 0, fill: '#9b59b6' }
                        ]}
                        margin={{ left: 20, right: 30 }}
                      >
                        <CartesianGrid strokeDasharray="3 3" stroke="#2a2a4a" />
                        <XAxis type="number" domain={[0, 100]} stroke="#888" fontSize={12} tickFormatter={(v) => `${v}%`} />
                        <YAxis type="category" dataKey="name" stroke="#888" fontSize={12} width={80} />
                        <Tooltip 
                          contentStyle={{ backgroundColor: '#1a1a2e', border: '1px solid #2a2a4a', borderRadius: '8px' }}
                          formatter={(value: number) => [`${value.toFixed(1)}%`, 'Value']}
                        />
                        <Bar dataKey="value" radius={[0, 4, 4, 0]}>
                          {[
                            { name: 'Acceptance', fill: '#4ade80' },
                            { name: 'Efficiency', fill: '#00d4ff' },
                            { name: 'Uptime', fill: '#9b59b6' }
                          ].map((entry, index) => (
                            <Cell key={`cell-${index}`} fill={entry.fill} />
                          ))}
                        </Bar>
                      </BarChart>
                    </ResponsiveContainer>
                  </div>

                  {/* Hashrate Comparison */}
                  <div style={graphStyles.chartCard}>
                    <h4 style={graphStyles.chartTitle}>Hashrate Analysis</h4>
                    <ResponsiveContainer width="100%" height={250}>
                      <BarChart
                        data={[
                          { name: 'Reported', value: selectedMiner.hashrate || 0 },
                          { name: 'Effective', value: selectedMiner.performance?.effective_hashrate || 0 }
                        ]}
                      >
                        <CartesianGrid strokeDasharray="3 3" stroke="#2a2a4a" />
                        <XAxis dataKey="name" stroke="#888" fontSize={12} />
                        <YAxis stroke="#888" fontSize={12} tickFormatter={(v) => formatHashrate(v)} />
                        <Tooltip 
                          contentStyle={{ backgroundColor: '#1a1a2e', border: '1px solid #2a2a4a', borderRadius: '8px' }}
                          formatter={(value: number) => [formatHashrate(value), 'Hashrate']}
                        />
                        <Bar dataKey="value" fill="#9b59b6" radius={[4, 4, 0, 0]} />
                      </BarChart>
                    </ResponsiveContainer>
                  </div>
                </div>

                {/* Recent Shares Table */}
                <h3 style={{ color: '#00d4ff', marginTop: '30px' }}>📋 Recent Shares</h3>
                <div style={adminStyles.tableContainer}>
                  <table style={adminStyles.table}>
                    <thead>
                      <tr>
                        <th style={adminStyles.th}>Time</th>
                        <th style={adminStyles.th}>Difficulty</th>
                        <th style={adminStyles.th}>Status</th>
                        <th style={adminStyles.th}>Nonce</th>
                      </tr>
                    </thead>
                    <tbody>
                      {(selectedMiner.recent_shares || []).map((share: any) => (
                        <tr key={share.id} style={adminStyles.tr}>
                          <td style={adminStyles.td}>{share.time_since}</td>
                          <td style={adminStyles.td}>{share.difficulty?.toFixed(4)}</td>
                          <td style={adminStyles.td}>
                            <span style={share.is_valid ? adminStyles.activeBadge : adminStyles.inactiveBadge}>
                              {share.is_valid ? '✓ Valid' : '✗ Invalid'}
                            </span>
                          </td>
                          <td style={adminStyles.td}><code style={{ fontSize: '0.8rem' }}>{share.nonce?.substring(0, 16)}...</code></td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </div>
            ) : selectedUserMiners ? (
              // User Miners Summary View
              <div>
                <button 
                  style={{ marginBottom: '20px', padding: '8px 16px', backgroundColor: '#2a2a4a', border: 'none', borderRadius: '6px', color: '#e0e0e0', cursor: 'pointer' }}
                  onClick={() => setSelectedUserMiners(null)}
                >
                  ← Back to Miners List
                </button>

                <div style={{ ...adminStyles.algoCard, marginBottom: '20px' }}>
                  <h3 style={{ color: '#00d4ff', marginTop: 0 }}>👤 {selectedUserMiners.username}'s Mining Overview</h3>
                  <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(150px, 1fr))', gap: '15px', marginTop: '15px' }}>
                    <div style={{ textAlign: 'center' }}>
                      <div style={{ fontSize: '2rem', color: '#9b59b6' }}>{selectedUserMiners.total_miners}</div>
                      <div style={{ color: '#888' }}>Total Miners</div>
                    </div>
                    <div style={{ textAlign: 'center' }}>
                      <div style={{ fontSize: '2rem', color: '#4ade80' }}>{selectedUserMiners.active_miners}</div>
                      <div style={{ color: '#888' }}>Active</div>
                    </div>
                    <div style={{ textAlign: 'center' }}>
                      <div style={{ fontSize: '2rem', color: '#f87171' }}>{selectedUserMiners.inactive_miners}</div>
                      <div style={{ color: '#888' }}>Offline</div>
                    </div>
                    <div style={{ textAlign: 'center' }}>
                      <div style={{ fontSize: '2rem', color: '#00d4ff' }}>{formatHashrate(selectedUserMiners.total_hashrate)}</div>
                      <div style={{ color: '#888' }}>Total Hashrate</div>
                    </div>
                    <div style={{ textAlign: 'center' }}>
                      <div style={{ fontSize: '2rem', color: '#fbbf24' }}>{selectedUserMiners.total_shares_24h}</div>
                      <div style={{ color: '#888' }}>Shares (24h)</div>
                    </div>
                  </div>
                </div>

                <h3 style={{ color: '#00d4ff' }}>⛏️ Miners</h3>
                <div style={{ display: 'flex', flexDirection: 'column', gap: '10px' }}>
                  {(selectedUserMiners.miners || []).map((miner: any) => (
                    <div 
                      key={miner.id} 
                      style={{ ...adminStyles.algoCard, cursor: 'pointer', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}
                      onClick={() => fetchMinerDetail(miner.id)}
                    >
                      <div>
                        <strong style={{ color: '#e0e0e0' }}>{miner.name}</strong>
                        <span style={{ marginLeft: '10px', ...miner.is_active ? adminStyles.activeBadge : adminStyles.inactiveBadge }}>
                          {miner.is_active ? 'Online' : 'Offline'}
                        </span>
                      </div>
                      <div style={{ display: 'flex', gap: '20px', color: '#888' }}>
                        <span>⚡ {formatHashrate(miner.hashrate)}</span>
                        <span>📊 {miner.shares_24h} shares</span>
                        <span>✓ {miner.valid_percent?.toFixed(1)}%</span>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            ) : (
              // All Miners List View
              <>
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '20px', flexWrap: 'wrap', gap: '15px' }}>
                  <div>
                    <h3 style={{ color: '#9b59b6', margin: 0 }}>⛏️ Miner Monitoring</h3>
                    <p style={{ color: '#888', margin: '5px 0 0', fontSize: '0.9rem' }}>
                      {minerTotal} miners • View detailed performance and troubleshooting info
                    </p>
                  </div>
                  <div style={{ display: 'flex', gap: '10px', alignItems: 'center' }}>
                    <label style={{ color: '#888', display: 'flex', alignItems: 'center', gap: '5px' }}>
                      <input 
                        type="checkbox" 
                        checked={activeMinersOnly} 
                        onChange={e => { setActiveMinersOnly(e.target.checked); setMinerPage(1); }}
                      />
                      Active only
                    </label>
                    <button 
                      style={{ padding: '8px 16px', backgroundColor: '#00d4ff', border: 'none', borderRadius: '6px', color: '#0a0a0f', fontWeight: 'bold', cursor: 'pointer' }}
                      onClick={fetchAllMiners}
                    >
                      🔄 Refresh
                    </button>
                  </div>
                </div>

                <div style={adminStyles.searchBar}>
                  <input 
                    style={adminStyles.searchInput} 
                    type="text" 
                    placeholder="Search miners by name..." 
                    value={minerSearch} 
                    onChange={e => { setMinerSearch(e.target.value); setMinerPage(1); }} 
                  />
                </div>

                {minersLoading ? (
                  <div style={adminStyles.loading}>Loading miners...</div>
                ) : allMiners.length === 0 ? (
                  <div style={{ textAlign: 'center', padding: '40px', color: '#666' }}>
                    <p style={{ fontSize: '1.2rem', margin: '0 0 10px' }}>No miners found</p>
                    <p style={{ margin: 0 }}>No miners are currently registered in the pool.</p>
                  </div>
                ) : (
                  <>
                    <div style={adminStyles.tableContainer}>
                      <table style={adminStyles.table}>
                        <thead>
                          <tr>
                            <th style={adminStyles.th}>Miner Name</th>
                            <th style={adminStyles.th}>IP Address</th>
                            <th style={adminStyles.th}>Hashrate</th>
                            <th style={adminStyles.th}>Shares (24h)</th>
                            <th style={adminStyles.th}>Valid %</th>
                            <th style={adminStyles.th}>Status</th>
                            <th style={adminStyles.th}>Last Seen</th>
                            <th style={adminStyles.th}>Actions</th>
                          </tr>
                        </thead>
                        <tbody>
                          {allMiners.map((miner: any) => (
                            <tr key={miner.id} style={adminStyles.tr}>
                              <td style={adminStyles.td}><strong>{miner.name}</strong></td>
                              <td style={adminStyles.td}>{miner.address || 'Unknown'}</td>
                              <td style={adminStyles.td}>{formatHashrate(miner.hashrate)}</td>
                              <td style={adminStyles.td}>{miner.shares_24h}</td>
                              <td style={adminStyles.td}>
                                <span style={{ color: miner.valid_percent >= 95 ? '#4ade80' : miner.valid_percent >= 80 ? '#fbbf24' : '#f87171' }}>
                                  {miner.valid_percent?.toFixed(1)}%
                                </span>
                              </td>
                              <td style={adminStyles.td}>
                                <span style={miner.is_active ? adminStyles.activeBadge : adminStyles.inactiveBadge}>
                                  {miner.is_active ? '🟢 Online' : '🔴 Offline'}
                                </span>
                              </td>
                              <td style={adminStyles.td}>{new Date(miner.last_seen).toLocaleString()}</td>
                              <td style={adminStyles.td}>
                                <button 
                                  style={{ ...adminStyles.actionBtn, backgroundColor: '#1a3a4a', borderRadius: '4px' }} 
                                  onClick={() => fetchMinerDetail(miner.id)}
                                  title="View Details"
                                >
                                  🔍
                                </button>
                              </td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </div>

                    <div style={adminStyles.pagination}>
                      <button style={adminStyles.pageBtn} disabled={minerPage <= 1} onClick={() => setMinerPage(p => p - 1)}>← Prev</button>
                      <span style={adminStyles.pageInfo}>Page {minerPage} of {Math.ceil(minerTotal / 20)} ({minerTotal} miners)</span>
                      <button style={adminStyles.pageBtn} disabled={minerPage >= Math.ceil(minerTotal / 20)} onClick={() => setMinerPage(p => p + 1)}>Next →</button>
                    </div>
                  </>
                )}
              </>
            )}
          </div>
        )}
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
