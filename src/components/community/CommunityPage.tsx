import React, { useState, useEffect } from 'react';

// Interfaces
interface Channel {
  id: number;
  name: string;
  description: string;
  type: string;
  isReadOnly: boolean;
  adminOnlyPost: boolean;
}

interface ChannelCategory {
  id: number;
  name: string;
  description?: string;
  channels: Channel[];
}

interface ChatMessage {
  id: number;
  content: string;
  isEdited: boolean;
  createdAt: string;
  user: {
    id: number;
    username: string;
    badgeIcon: string;
    badgeColor: string;
  };
  replyToId?: number;
}

interface ForumCategory {
  id: number;
  name: string;
  description: string;
  icon: string;
  postCount: number;
}

interface ForumPost {
  id: number;
  title: string;
  preview: string;
  tags: string[];
  viewCount: number;
  replyCount: number;
  upvotes: number;
  isPinned: boolean;
  isLocked: boolean;
  createdAt: string;
  author: {
    id: number;
    username: string;
    badgeIcon: string;
  };
}

interface OnlineUser {
  id: number;
  username: string;
  status: string;
  badgeIcon: string;
}

interface CommunityPageProps {
  token: string;
  user: any;
  showMessage: (type: 'success' | 'error', text: string) => void;
}

function CommunityPage({ token, user, showMessage }: CommunityPageProps) {
  const [activeView, setActiveView] = useState<'chat' | 'forums' | 'leaderboard'>('chat');
  const [categories, setCategories] = useState<ChannelCategory[]>([]);
  const [selectedChannel, setSelectedChannel] = useState<Channel | null>(null);
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [newMessage, setNewMessage] = useState('');
  const [onlineUsers, setOnlineUsers] = useState<OnlineUser[]>([]);
  const [collapsedCategories, setCollapsedCategories] = useState<Set<number>>(new Set());
  const [loading, setLoading] = useState(true);
  const [leaderboard, setLeaderboard] = useState<any[]>([]);
  const [leaderboardType, setLeaderboardType] = useState('hashrate');
  
  // Admin channel management state
  const [showCreateChannel, setShowCreateChannel] = useState(false);
  const [showCreateCategory, setShowCreateCategory] = useState(false);
  const [showEditChannel, setShowEditChannel] = useState(false);
  const [showEditCategory, setShowEditCategory] = useState(false);
  const [editingChannel, setEditingChannel] = useState<Channel | null>(null);
  const [editingCategory, setEditingCategory] = useState<ChannelCategory | null>(null);
  const [channelForm, setChannelForm] = useState({ name: '', description: '', category_id: '', type: 'text', is_read_only: false, admin_only_post: false });
  const [categoryForm, setCategoryForm] = useState({ name: '', description: '' });

  const isAdmin = user?.is_admin || user?.role === 'admin' || user?.role === 'super_admin';
  const isModerator = isAdmin || user?.role === 'moderator';

  const fetchChannels = async () => {
    console.log('fetchChannels called');
    try {
      const res = await fetch('/api/v1/community/channels', { headers: { Authorization: `Bearer ${token}` } });
      console.log('fetchChannels response:', res.status);
      if (res.ok) {
        const data = await res.json();
        console.log('fetchChannels data:', data);
        setCategories(data.categories || []);
        if (data.categories?.length > 0 && data.categories[0].channels?.length > 0) {
          setSelectedChannel(data.categories[0].channels[0]);
        }
      } else {
        console.error('fetchChannels failed:', res.status, await res.text());
      }
    } catch (e) { console.error('fetchChannels error:', e); }
    setLoading(false);
  };

  useEffect(() => {
    fetchChannels();
    fetchOnlineUsers();
    const interval = setInterval(fetchOnlineUsers, 30000);
    return () => clearInterval(interval);
  }, []);

  useEffect(() => {
    if (selectedChannel) fetchMessages();
  }, [selectedChannel]);

  useEffect(() => {
    if (activeView === 'leaderboard') fetchLeaderboard();
  }, [activeView, leaderboardType]);

  const fetchMessages = async () => {
    if (!selectedChannel) return;
    try {
      const res = await fetch(`/api/v1/community/channels/${selectedChannel.id}/messages`, { headers: { Authorization: `Bearer ${token}` } });
      if (res.ok) {
        const data = await res.json();
        setMessages(data.messages || []);
      }
    } catch (e) { console.error(e); }
  };

  const fetchOnlineUsers = async () => {
    try {
      const res = await fetch('/api/v1/community/online-users', { headers: { Authorization: `Bearer ${token}` } });
      if (res.ok) {
        const data = await res.json();
        setOnlineUsers(data.users || []);
      }
    } catch (e) { console.error(e); }
  };

  const fetchLeaderboard = async () => {
    try {
      const res = await fetch(`/api/v1/community/leaderboard?type=${leaderboardType}`, { headers: { Authorization: `Bearer ${token}` } });
      if (res.ok) {
        const data = await res.json();
        setLeaderboard(data.leaderboard || []);
      }
    } catch (e) { console.error(e); }
  };

  const sendMessage = async () => {
    if (!newMessage.trim() || !selectedChannel) return;
    try {
      const res = await fetch(`/api/v1/community/channels/${selectedChannel.id}/messages`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
        body: JSON.stringify({ content: newMessage })
      });
      if (res.ok) {
        setNewMessage('');
        fetchMessages();
      }
    } catch (e) { console.error(e); }
  };

  const toggleCategory = (catId: number) => {
    const newCollapsed = new Set(collapsedCategories);
    if (newCollapsed.has(catId)) newCollapsed.delete(catId);
    else newCollapsed.add(catId);
    setCollapsedCategories(newCollapsed);
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
        fetchChannels();
      } else {
        const data = await response.json();
        showMessage('error', data.error || 'Failed to create channel');
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
        fetchChannels();
      } else {
        const data = await response.json();
        showMessage('error', data.error || 'Failed to create category');
      }
    } catch (error) {
      showMessage('error', 'Network error');
    }
  };

  const handleDeleteCategory = async (categoryId: string) => {
    if (!window.confirm('Delete this category? All channels in it must be deleted first.')) return;
    try {
      const response = await fetch(`/api/v1/admin/community/channel-categories/${categoryId}`, {
        method: 'DELETE',
        headers: { 'Authorization': `Bearer ${token}` }
      });
      if (response.ok) {
        showMessage('success', 'Category deleted');
        fetchChannels();
      } else {
        const data = await response.json();
        showMessage('error', data.error || 'Failed to delete category');
      }
    } catch (error) {
      showMessage('error', 'Network error');
    }
  };

  const handleDeleteChannel = async (channelId: string) => {
    if (!window.confirm('Delete this channel?')) return;
    try {
      const response = await fetch(`/api/v1/admin/community/channels/${channelId}`, {
        method: 'DELETE',
        headers: { 'Authorization': `Bearer ${token}` }
      });
      if (response.ok) {
        showMessage('success', 'Channel deleted');
        if (String(selectedChannel?.id) === channelId) setSelectedChannel(null);
        fetchChannels();
      } else {
        const data = await response.json();
        showMessage('error', data.error || 'Failed to delete channel');
      }
    } catch (error) {
      showMessage('error', 'Network error');
    }
  };

  const handleEditCategory = async () => {
    if (!editingCategory || !categoryForm.name) return;
    try {
      const response = await fetch(`/api/v1/admin/community/channel-categories/${editingCategory.id}`, {
        method: 'PUT',
        headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
        body: JSON.stringify({ name: categoryForm.name, description: categoryForm.description })
      });
      if (response.ok) {
        showMessage('success', 'Category updated');
        setShowEditCategory(false);
        setEditingCategory(null);
        setCategoryForm({ name: '', description: '' });
        fetchChannels();
      } else {
        const data = await response.json();
        showMessage('error', data.error || 'Failed to update category');
      }
    } catch (error) {
      showMessage('error', 'Network error');
    }
  };

  const handleEditChannel = async () => {
    if (!editingChannel || !channelForm.name) return;
    try {
      const response = await fetch(`/api/v1/admin/community/channels/${editingChannel.id}`, {
        method: 'PUT',
        headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
        body: JSON.stringify(channelForm)
      });
      if (response.ok) {
        showMessage('success', 'Channel updated');
        setShowEditChannel(false);
        setEditingChannel(null);
        setChannelForm({ name: '', description: '', category_id: '', type: 'text', is_read_only: false, admin_only_post: false });
        fetchChannels();
      } else {
        const data = await response.json();
        showMessage('error', data.error || 'Failed to update channel');
      }
    } catch (error) {
      showMessage('error', 'Network error');
    }
  };

  const formatTime = (dateStr: string) => {
    const date = new Date(dateStr);
    return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
  };

  if (loading) return <div style={styles.loading}>Loading community...</div>;

  return (
    <div style={styles.pageContainer}>
      {/* Left Sidebar - Channels */}
      <div style={styles.leftSidebar}>
        <div style={styles.sidebarHeader}>
          <span>💬 Channels</span>
          {isModerator && (
            <div style={{ display: 'flex', gap: '5px' }}>
              <button style={styles.addBtn} onClick={() => setShowCreateCategory(true)} title="New Category">📁+</button>
              <button style={styles.addBtn} onClick={() => { setShowCreateChannel(true); setChannelForm({ ...channelForm, category_id: categories[0]?.id?.toString() || '' }); }} title="New Channel">💬+</button>
            </div>
          )}
        </div>
        
        {categories.length === 0 ? (
          <div style={styles.emptyState}>
            <p>No channels yet.</p>
            {isModerator && <p style={{ color: '#00d4ff', fontSize: '0.85rem' }}>Click 📁+ to create a category first.</p>}
          </div>
        ) : (
          categories.map(cat => (
            <div key={cat.id} style={styles.category}>
              <div style={{ ...styles.categoryHeader, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                <div onClick={() => toggleCategory(cat.id)} style={{ display: 'flex', alignItems: 'center', flex: 1, cursor: 'pointer' }}>
                  <span>{collapsedCategories.has(cat.id) ? '▶' : '▼'}</span>
                  <span style={styles.categoryName}>{cat.name}</span>
                </div>
                {isModerator && (
                  <div style={{ display: 'flex', gap: '4px' }}>
                    <button onClick={(e) => { e.stopPropagation(); setEditingCategory(cat); setCategoryForm({ name: cat.name, description: cat.description || '' }); setShowEditCategory(true); }} style={{ background: 'none', border: 'none', color: '#888', cursor: 'pointer', fontSize: '12px', padding: '2px' }} title="Edit Category">✏️</button>
                    <button onClick={(e) => { e.stopPropagation(); handleDeleteCategory(String(cat.id)); }} style={{ background: 'none', border: 'none', color: '#888', cursor: 'pointer', fontSize: '12px', padding: '2px' }} title="Delete Category">🗑️</button>
                  </div>
                )}
              </div>
              {!collapsedCategories.has(cat.id) && cat.channels?.map(ch => (
                <div
                  key={ch.id}
                  style={{ ...styles.channel, ...(selectedChannel?.id === ch.id ? styles.channelActive : {}), display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}
                >
                  <div onClick={() => setSelectedChannel(ch)} style={{ display: 'flex', alignItems: 'center', flex: 1, cursor: 'pointer' }}>
                    <span style={styles.channelHash}>#</span>
                    <span>{ch.name}</span>
                    {ch.type === 'announcement' && <span style={styles.channelBadge}>📢</span>}
                    {ch.type === 'regional' && <span style={styles.channelBadge}>🌍</span>}
                  </div>
                  {isModerator && (
                    <div style={{ display: 'flex', gap: '4px' }}>
                      <button onClick={(e) => { e.stopPropagation(); setEditingChannel(ch); setChannelForm({ name: ch.name, description: ch.description || '', category_id: String(cat.id), type: ch.type || 'text', is_read_only: ch.isReadOnly || false, admin_only_post: ch.adminOnlyPost || false }); setShowEditChannel(true); }} style={{ background: 'none', border: 'none', color: '#888', cursor: 'pointer', fontSize: '10px', padding: '2px' }} title="Edit Channel">✏️</button>
                      <button onClick={(e) => { e.stopPropagation(); handleDeleteChannel(String(ch.id)); }} style={{ background: 'none', border: 'none', color: '#888', cursor: 'pointer', fontSize: '10px', padding: '2px' }} title="Delete Channel">🗑️</button>
                    </div>
                  )}
                </div>
              ))}
            </div>
          ))
        )}

        {/* Online Users */}
        <div style={styles.onlineSection}>
          <div style={styles.onlineHeader}>Online — {onlineUsers.length}</div>
          {onlineUsers.slice(0, 15).map(u => (
            <div key={u.id} style={styles.onlineUser}>
              <span style={styles.onlineIndicator}></span>
              <span>{u.badgeIcon}</span>
              <span style={styles.onlineUsername}>{u.username}</span>
            </div>
          ))}
        </div>
      </div>

      {/* Main Content Area */}
      <div style={styles.mainContent}>
        {/* Secondary Navigation */}
        <div style={styles.secondaryNav}>
          {[
            { key: 'chat', label: '💬 Chat' },
            { key: 'forums', label: '📋 Forums' },
            { key: 'leaderboard', label: '🏆 Leaderboard' },
          ].map(tab => (
            <button
              key={tab.key}
              onClick={() => setActiveView(tab.key as any)}
              style={{ ...styles.secondaryTab, ...(activeView === tab.key ? styles.secondaryTabActive : {}) }}
            >
              {tab.label}
            </button>
          ))}
        </div>

        {/* Chat View */}
        {activeView === 'chat' && (
          <div style={styles.chatContainer}>
            <div style={styles.chatHeader}>
              <span style={styles.chatChannelName}># {selectedChannel?.name || 'Select a channel'}</span>
              <span style={styles.chatChannelDesc}>{selectedChannel?.description}</span>
            </div>

            <div style={styles.messagesContainer}>
              {messages.length === 0 ? (
                <div style={styles.noMessages}>
                  <p>No messages yet. Be the first to say something!</p>
                </div>
              ) : (
                messages.map(msg => (
                  <div key={msg.id} style={styles.message}>
                    <div style={styles.messageHeader}>
                      <span style={{ ...styles.messageBadge, color: msg.user.badgeColor }}>{msg.user.badgeIcon}</span>
                      <span style={styles.messageUsername}>{msg.user.username}</span>
                      <span style={styles.messageTime}>{formatTime(msg.createdAt)}</span>
                    </div>
                    <div style={styles.messageContent}>{msg.content}</div>
                  </div>
                ))
              )}
            </div>

            <div style={styles.inputContainer}>
              <input
                style={styles.messageInput}
                type="text"
                placeholder={selectedChannel ? `Message #${selectedChannel.name}` : 'Select a channel...'}
                value={newMessage}
                onChange={e => setNewMessage(e.target.value)}
                onKeyPress={e => e.key === 'Enter' && sendMessage()}
                disabled={!selectedChannel || selectedChannel.isReadOnly}
              />
              <button style={styles.sendBtn} onClick={sendMessage} disabled={!selectedChannel}>
                Send
              </button>
            </div>
          </div>
        )}

        {/* Leaderboard View */}
        {activeView === 'leaderboard' && (
          <div style={styles.leaderboardContainer}>
            <div style={styles.leaderboardHeader}>
              <h3>🏆 Mining Leaderboard</h3>
              <select 
                style={styles.leaderboardSelect}
                value={leaderboardType}
                onChange={e => setLeaderboardType(e.target.value)}
              >
                <option value="hashrate">Hashrate</option>
                <option value="shares">Shares</option>
                <option value="blocks">Blocks Found</option>
              </select>
            </div>
            <div style={styles.leaderboardList}>
              {leaderboard.map((entry, idx) => (
                <div key={entry.id} style={styles.leaderboardEntry}>
                  <span style={styles.leaderboardRank}>#{idx + 1}</span>
                  <span style={styles.leaderboardBadge}>{entry.badgeIcon}</span>
                  <span style={styles.leaderboardName}>{entry.username}</span>
                  <span style={styles.leaderboardValue}>{entry.value}</span>
                </div>
              ))}
            </div>
          </div>
        )}

        {/* Forums View */}
        {activeView === 'forums' && (
          <div style={styles.forumsContainer}>
            <h3 style={{ color: '#00d4ff', margin: '0 0 20px' }}>📋 Forums</h3>
            <p style={{ color: '#888' }}>Forums coming soon...</p>
          </div>
        )}
      </div>

      {/* Create Channel Modal */}
      {showCreateChannel && (
        <div style={styles.modalOverlay} onClick={() => setShowCreateChannel(false)}>
          <div style={styles.modal} onClick={e => e.stopPropagation()}>
            <h3 style={styles.modalTitle}>Create New Channel</h3>
            <div style={styles.formGroup}>
              <label style={styles.label}>Channel Name *</label>
              <input style={styles.input} type="text" placeholder="e.g., general-chat" value={channelForm.name} onChange={e => setChannelForm({...channelForm, name: e.target.value})} />
            </div>
            <div style={styles.formGroup}>
              <label style={styles.label}>Description</label>
              <input style={styles.input} type="text" placeholder="What's this channel for?" value={channelForm.description} onChange={e => setChannelForm({...channelForm, description: e.target.value})} />
            </div>
            <div style={styles.formGroup}>
              <label style={styles.label}>Category *</label>
              <select style={styles.select} value={channelForm.category_id} onChange={e => setChannelForm({...channelForm, category_id: e.target.value})}>
                <option value="">Select a category</option>
                {categories.map(cat => <option key={cat.id} value={cat.id}>{cat.name}</option>)}
              </select>
            </div>
            <div style={styles.formGroup}>
              <label style={styles.label}>Channel Type</label>
              <select style={styles.select} value={channelForm.type} onChange={e => setChannelForm({...channelForm, type: e.target.value})}>
                <option value="text">💬 Text</option>
                <option value="announcement">📢 Announcement</option>
                <option value="regional">🌍 Regional</option>
              </select>
            </div>
            <div style={styles.modalActions}>
              <button style={styles.cancelBtn} onClick={() => setShowCreateChannel(false)}>Cancel</button>
              <button style={styles.submitBtn} onClick={handleCreateChannel}>Create</button>
            </div>
          </div>
        </div>
      )}

      {/* Create Category Modal */}
      {showCreateCategory && (
        <div style={styles.modalOverlay} onClick={() => setShowCreateCategory(false)}>
          <div style={styles.modal} onClick={e => e.stopPropagation()}>
            <h3 style={styles.modalTitle}>Create New Category</h3>
            <div style={styles.formGroup}>
              <label style={styles.label}>Category Name *</label>
              <input style={styles.input} type="text" placeholder="e.g., General, Mining Talk" value={categoryForm.name} onChange={e => setCategoryForm({...categoryForm, name: e.target.value})} />
            </div>
            <div style={styles.formGroup}>
              <label style={styles.label}>Description</label>
              <input style={styles.input} type="text" placeholder="What topics belong here?" value={categoryForm.description} onChange={e => setCategoryForm({...categoryForm, description: e.target.value})} />
            </div>
            <div style={styles.modalActions}>
              <button style={styles.cancelBtn} onClick={() => setShowCreateCategory(false)}>Cancel</button>
              <button style={styles.submitBtn} onClick={handleCreateCategory}>Create</button>
            </div>
          </div>
        </div>
      )}

      {/* Edit Category Modal */}
      {showEditCategory && editingCategory && (
        <div style={styles.modalOverlay} onClick={() => { setShowEditCategory(false); setEditingCategory(null); }}>
          <div style={styles.modal} onClick={e => e.stopPropagation()}>
            <h3 style={styles.modalTitle}>Edit Category</h3>
            <div style={styles.formGroup}>
              <label style={styles.label}>Category Name *</label>
              <input style={styles.input} type="text" value={categoryForm.name} onChange={e => setCategoryForm({...categoryForm, name: e.target.value})} />
            </div>
            <div style={styles.formGroup}>
              <label style={styles.label}>Description</label>
              <input style={styles.input} type="text" value={categoryForm.description} onChange={e => setCategoryForm({...categoryForm, description: e.target.value})} />
            </div>
            <div style={styles.modalActions}>
              <button style={styles.cancelBtn} onClick={() => { setShowEditCategory(false); setEditingCategory(null); }}>Cancel</button>
              <button style={styles.submitBtn} onClick={handleEditCategory}>Save</button>
            </div>
          </div>
        </div>
      )}

      {/* Edit Channel Modal */}
      {showEditChannel && editingChannel && (
        <div style={styles.modalOverlay} onClick={() => { setShowEditChannel(false); setEditingChannel(null); }}>
          <div style={styles.modal} onClick={e => e.stopPropagation()}>
            <h3 style={styles.modalTitle}>Edit Channel</h3>
            <div style={styles.formGroup}>
              <label style={styles.label}>Channel Name *</label>
              <input style={styles.input} type="text" value={channelForm.name} onChange={e => setChannelForm({...channelForm, name: e.target.value})} />
            </div>
            <div style={styles.formGroup}>
              <label style={styles.label}>Description</label>
              <input style={styles.input} type="text" value={channelForm.description} onChange={e => setChannelForm({...channelForm, description: e.target.value})} />
            </div>
            <div style={styles.formGroup}>
              <label style={styles.label}>Channel Type</label>
              <select style={styles.select} value={channelForm.type} onChange={e => setChannelForm({...channelForm, type: e.target.value})}>
                <option value="text">💬 Text</option>
                <option value="announcement">📢 Announcement</option>
                <option value="regional">🌍 Regional</option>
              </select>
            </div>
            <div style={styles.modalActions}>
              <button style={styles.cancelBtn} onClick={() => { setShowEditChannel(false); setEditingChannel(null); }}>Cancel</button>
              <button style={styles.submitBtn} onClick={handleEditChannel}>Save</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

const styles: { [key: string]: React.CSSProperties } = {
  pageContainer: { display: 'flex', height: 'calc(100vh - 100px)', backgroundColor: '#0a0a0f' },
  leftSidebar: { width: '260px', backgroundColor: '#1a1a2e', borderRight: '1px solid #2a2a4a', display: 'flex', flexDirection: 'column', overflowY: 'auto' },
  sidebarHeader: { padding: '15px', borderBottom: '1px solid #2a2a4a', color: '#00d4ff', fontWeight: 'bold', fontSize: '1.1rem', display: 'flex', justifyContent: 'space-between', alignItems: 'center' },
  addBtn: { background: 'none', border: 'none', cursor: 'pointer', fontSize: '1rem', padding: '4px 8px', borderRadius: '4px', color: '#888' },
  emptyState: { padding: '20px', textAlign: 'center', color: '#666' },
  category: { marginBottom: '5px' },
  categoryHeader: { display: 'flex', alignItems: 'center', gap: '8px', padding: '8px 15px', color: '#888', fontSize: '0.85rem', cursor: 'pointer', textTransform: 'uppercase' },
  categoryName: { fontWeight: 'bold' },
  channel: { display: 'flex', alignItems: 'center', gap: '8px', padding: '8px 15px 8px 25px', color: '#888', cursor: 'pointer', borderRadius: '4px', margin: '2px 8px' },
  channelActive: { backgroundColor: '#2a2a4a', color: '#e0e0e0' },
  channelHash: { color: '#666' },
  channelBadge: { marginLeft: 'auto', fontSize: '0.8rem' },
  onlineSection: { marginTop: 'auto', borderTop: '1px solid #2a2a4a', padding: '10px' },
  onlineHeader: { padding: '8px', color: '#888', fontSize: '0.8rem', textTransform: 'uppercase' },
  onlineUser: { display: 'flex', alignItems: 'center', gap: '8px', padding: '4px 8px', fontSize: '0.9rem' },
  onlineIndicator: { width: '8px', height: '8px', borderRadius: '50%', backgroundColor: '#4ade80' },
  onlineUsername: { color: '#e0e0e0' },
  mainContent: { flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' },
  secondaryNav: { display: 'flex', gap: '5px', padding: '10px 20px', borderBottom: '1px solid #2a2a4a', backgroundColor: '#1a1a2e' },
  secondaryTab: { padding: '10px 20px', backgroundColor: 'transparent', border: 'none', color: '#888', fontSize: '0.95rem', cursor: 'pointer', borderRadius: '6px' },
  secondaryTabActive: { backgroundColor: '#2a2a4a', color: '#00d4ff' },
  chatContainer: { flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' },
  chatHeader: { padding: '15px 20px', borderBottom: '1px solid #2a2a4a', backgroundColor: '#1a1a2e' },
  chatChannelName: { color: '#e0e0e0', fontWeight: 'bold', fontSize: '1.1rem' },
  chatChannelDesc: { color: '#888', fontSize: '0.9rem', marginLeft: '15px' },
  messagesContainer: { flex: 1, overflowY: 'auto', padding: '20px' },
  noMessages: { textAlign: 'center', color: '#666', padding: '40px' },
  message: { marginBottom: '20px' },
  messageHeader: { display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '4px' },
  messageBadge: { fontSize: '1.1rem' },
  messageUsername: { color: '#00d4ff', fontWeight: 'bold' },
  messageTime: { color: '#666', fontSize: '0.8rem' },
  messageContent: { color: '#e0e0e0', paddingLeft: '28px' },
  inputContainer: { display: 'flex', gap: '10px', padding: '15px 20px', borderTop: '1px solid #2a2a4a', backgroundColor: '#1a1a2e' },
  messageInput: { flex: 1, padding: '12px 16px', backgroundColor: '#0a0a15', border: '1px solid #2a2a4a', borderRadius: '8px', color: '#e0e0e0', fontSize: '1rem' },
  sendBtn: { padding: '12px 24px', backgroundColor: '#00d4ff', border: 'none', borderRadius: '8px', color: '#0a0a0f', fontWeight: 'bold', cursor: 'pointer' },
  leaderboardContainer: { padding: '20px', overflowY: 'auto' },
  leaderboardHeader: { display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '20px', color: '#00d4ff' },
  leaderboardSelect: { padding: '8px 16px', backgroundColor: '#1a1a2e', border: '1px solid #2a2a4a', borderRadius: '6px', color: '#e0e0e0' },
  leaderboardList: { display: 'flex', flexDirection: 'column', gap: '10px' },
  leaderboardEntry: { display: 'flex', alignItems: 'center', gap: '15px', padding: '15px', backgroundColor: '#1a1a2e', borderRadius: '8px', border: '1px solid #2a2a4a' },
  leaderboardRank: { color: '#f59e0b', fontWeight: 'bold', width: '40px' },
  leaderboardBadge: { fontSize: '1.2rem' },
  leaderboardName: { flex: 1, color: '#e0e0e0' },
  leaderboardValue: { color: '#00d4ff', fontWeight: 'bold' },
  forumsContainer: { padding: '20px' },
  loading: { display: 'flex', justifyContent: 'center', alignItems: 'center', height: 'calc(100vh - 100px)', color: '#00d4ff', fontSize: '1.2rem' },
  modalOverlay: { position: 'fixed', top: 0, left: 0, right: 0, bottom: 0, backgroundColor: 'rgba(0,0,0,0.8)', display: 'flex', justifyContent: 'center', alignItems: 'center', zIndex: 1000 },
  modal: { backgroundColor: '#1a1a2e', padding: '30px', borderRadius: '12px', border: '1px solid #2a2a4a', width: '100%', maxWidth: '450px' },
  modalTitle: { color: '#00d4ff', margin: '0 0 20px' },
  formGroup: { marginBottom: '15px' },
  label: { display: 'block', color: '#888', marginBottom: '5px', fontSize: '0.9rem' },
  input: { width: '100%', padding: '10px 12px', backgroundColor: '#0a0a15', border: '1px solid #2a2a4a', borderRadius: '6px', color: '#e0e0e0', fontSize: '1rem', boxSizing: 'border-box' },
  select: { width: '100%', padding: '10px 12px', backgroundColor: '#0a0a15', border: '1px solid #2a2a4a', borderRadius: '6px', color: '#e0e0e0', fontSize: '1rem' },
  modalActions: { display: 'flex', gap: '10px', marginTop: '20px' },
  cancelBtn: { flex: 1, padding: '10px', backgroundColor: '#2a2a4a', border: 'none', borderRadius: '6px', color: '#e0e0e0', cursor: 'pointer' },
  submitBtn: { flex: 1, padding: '10px', backgroundColor: '#00d4ff', border: 'none', borderRadius: '6px', color: '#0a0a0f', fontWeight: 'bold', cursor: 'pointer' },
};

export default CommunityPage;
