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

interface Badge {
  icon: string;
  color: string;
  name: string;
  type?: string;
  isPrimary?: boolean;
}

interface RoleBadge {
  icon: string;
  color: string;
  name: string;
  type: string;
}

interface MessageReaction {
  emoji: string;
  name: string;
  count: number;
  hasReacted: boolean;
}

interface ReactionType {
  id: number;
  emoji: string;
  name: string;
  category: string;
}

interface ChatMessage {
  id: number;
  content: string;
  isEdited: boolean;
  createdAt: string;
  user: {
    id: number;
    username: string;
    role?: string;
    roleBadge?: RoleBadge | null;
    badgeIcon: string;
    badgeColor: string;
    badgeName?: string;
    badges?: Badge[];
  };
  replyToId?: number;
  reactions?: MessageReaction[];
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
  const [leaderboardPage, setLeaderboardPage] = useState(1);
  const [leaderboardPageSize, setLeaderboardPageSize] = useState(20);
  const [leaderboardPagination, setLeaderboardPagination] = useState({ totalUsers: 0, totalPages: 0 });
  const [myRank, setMyRank] = useState<number | null>(null);
  const [myPage, setMyPage] = useState<number | null>(null);
  
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

  // Message action state (edit, delete, reply)
  const [editingMessage, setEditingMessage] = useState<ChatMessage | null>(null);
  const [editContent, setEditContent] = useState('');
  const [replyingTo, setReplyingTo] = useState<ChatMessage | null>(null);
  const [messageMenuOpen, setMessageMenuOpen] = useState<number | null>(null);
  
  // Reactions state
  const [reactionTypes, setReactionTypes] = useState<ReactionType[]>([]);
  const [showReactionPicker, setShowReactionPicker] = useState<number | null>(null);

  const fetchChannels = async () => {
    try {
      const res = await fetch('/api/v1/community/channels', { headers: { Authorization: `Bearer ${token}` } });
      if (res.ok) {
        const data = await res.json();
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
    fetchReactionTypes();
    const interval = setInterval(fetchOnlineUsers, 30000);
    return () => clearInterval(interval);
  }, []);

  useEffect(() => {
    if (selectedChannel) fetchMessages();
  }, [selectedChannel]);

  useEffect(() => {
    if (activeView === 'leaderboard') fetchLeaderboard(leaderboardPage, leaderboardPageSize);
  }, [activeView, leaderboardType, leaderboardPage, leaderboardPageSize]);

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

  const fetchReactionTypes = async () => {
    try {
      const res = await fetch('/api/v1/community/reaction-types', { headers: { Authorization: `Bearer ${token}` } });
      if (res.ok) {
        const data = await res.json();
        setReactionTypes(data.reactionTypes || []);
      }
    } catch (e) { console.error('fetchReactionTypes error:', e); }
  };

  const toggleReaction = async (messageId: number, emoji: string) => {
    try {
      const res = await fetch(`/api/v1/community/messages/${messageId}/reactions`, {
        method: 'POST',
        headers: { 
          Authorization: `Bearer ${token}`,
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({ emoji })
      });
      if (res.ok) {
        // Refresh messages to get updated reactions
        fetchMessages();
      } else {
        const data = await res.json();
        showMessage('error', data.error || 'Failed to toggle reaction');
      }
    } catch (e) { 
      console.error('toggleReaction error:', e);
      showMessage('error', 'Failed to toggle reaction');
    }
    setShowReactionPicker(null);
  };

  const fetchLeaderboard = async (page = leaderboardPage, pageSize = leaderboardPageSize) => {
    try {
      const res = await fetch(`/api/v1/community/leaderboard?type=${leaderboardType}&page=${page}&pageSize=${pageSize}`, { headers: { Authorization: `Bearer ${token}` } });
      if (res.ok) {
        const data = await res.json();
        setLeaderboard(data.leaderboard || []);
        setLeaderboardPagination(data.pagination || { totalUsers: 0, totalPages: 0 });
        setMyRank(data.myRank);
        setMyPage(data.myPage);
      }
    } catch (e) { console.error(e); }
  };

  const sendMessage = async () => {
    if (!newMessage.trim() || !selectedChannel) return;
    try {
      const payload: any = { content: newMessage };
      if (replyingTo) {
        payload.reply_to_id = replyingTo.id;
      }
      const res = await fetch(`/api/v1/community/channels/${selectedChannel.id}/messages`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
        body: JSON.stringify(payload)
      });
      if (res.ok) {
        setNewMessage('');
        setReplyingTo(null);
        fetchMessages();
      }
    } catch (e) { console.error(e); }
  };

  const handleEditMessage = async () => {
    if (!editingMessage || !editContent.trim()) return;
    try {
      const res = await fetch(`/api/v1/community/messages/${editingMessage.id}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
        body: JSON.stringify({ content: editContent })
      });
      if (res.ok) {
        setEditingMessage(null);
        setEditContent('');
        fetchMessages();
        showMessage('success', 'Message updated');
      } else {
        const data = await res.json();
        showMessage('error', data.error || 'Failed to edit message');
      }
    } catch (e) { 
      showMessage('error', 'Network error');
    }
  };

  const handleDeleteMessage = async (messageId: number) => {
    if (!window.confirm('Delete this message?')) return;
    try {
      const res = await fetch(`/api/v1/community/messages/${messageId}`, {
        method: 'DELETE',
        headers: { Authorization: `Bearer ${token}` }
      });
      if (res.ok) {
        fetchMessages();
        showMessage('success', 'Message deleted');
      } else {
        const data = await res.json();
        showMessage('error', data.error || 'Failed to delete message');
      }
    } catch (e) {
      showMessage('error', 'Network error');
    }
  };

  const startEditMessage = (msg: ChatMessage) => {
    setEditingMessage(msg);
    setEditContent(msg.content);
    setMessageMenuOpen(null);
  };

  const startReplyMessage = (msg: ChatMessage) => {
    setReplyingTo(msg);
    setMessageMenuOpen(null);
  };

  const cancelEdit = () => {
    setEditingMessage(null);
    setEditContent('');
  };

  const cancelReply = () => {
    setReplyingTo(null);
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

  // Format hashrate with appropriate units
  const formatHashrate = (hashrate: number): string => {
    if (hashrate === 0) return '0 H/s';
    const units = ['H/s', 'KH/s', 'MH/s', 'GH/s', 'TH/s', 'PH/s', 'EH/s'];
    let unitIndex = 0;
    let value = hashrate;
    while (value >= 1000 && unitIndex < units.length - 1) {
      value /= 1000;
      unitIndex++;
    }
    return `${value.toFixed(2)} ${units[unitIndex]}`;
  };

  // Format large numbers with K, M, B suffixes
  const formatNumber = (num: number): string => {
    if (num === 0) return '0';
    if (num >= 1000000000) return `${(num / 1000000000).toFixed(1)}B`;
    if (num >= 1000000) return `${(num / 1000000).toFixed(1)}M`;
    if (num >= 1000) return `${(num / 1000).toFixed(1)}K`;
    return num.toString();
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
                messages.map(msg => {
                  const isOwner = msg.user.id === user?.id;
                  const canDelete = isOwner || isModerator;
                  const canEdit = isOwner;
                  
                  // Find parent message if this is a reply
                  const parentMsg = msg.replyToId ? messages.find(m => m.id === msg.replyToId) : null;

                  return (
                    <div key={msg.id} style={styles.message}>
                      {/* Reply indicator */}
                      {parentMsg && (
                        <div style={styles.replyIndicator}>
                          <span style={{ color: '#666' }}>↳ Replying to </span>
                          <span style={{ color: '#00d4ff' }}>{parentMsg.user.username}</span>
                          <span style={{ color: '#666', marginLeft: '8px', fontSize: '0.85em' }}>
                            "{parentMsg.content.substring(0, 50)}{parentMsg.content.length > 50 ? '...' : ''}"
                          </span>
                        </div>
                      )}
                      
                      <div style={styles.messageHeader}>
                        {/* Role badge (admin/mod) first */}
                        {msg.user.roleBadge && (
                          <span 
                            style={{ ...styles.messageBadge, color: msg.user.roleBadge.color, marginRight: '2px' }} 
                            title={msg.user.roleBadge.name}
                          >
                            {msg.user.roleBadge.icon}
                          </span>
                        )}
                        {/* Achievement badge */}
                        <span style={{ ...styles.messageBadge, color: msg.user.badgeColor }} title={msg.user.badgeName || 'Newcomer'}>
                          {msg.user.badgeIcon}
                        </span>
                        {/* Additional badges (show up to 2 more) */}
                        {msg.user.badges?.slice(0, 2).filter((b: any) => !b.isPrimary).map((badge: any, idx: number) => (
                          <span key={idx} style={{ fontSize: '0.9rem', color: badge.color, marginRight: '2px' }} title={badge.name}>
                            {badge.icon}
                          </span>
                        ))}
                        <span style={{
                          ...styles.messageUsername,
                          color: msg.user.role === 'super_admin' ? '#fbbf24' : 
                                 msg.user.role === 'admin' ? '#ef4444' : 
                                 msg.user.role === 'moderator' ? '#f97316' : '#00d4ff'
                        }}>
                          {msg.user.username}
                        </span>
                        <span style={styles.messageTime}>
                          {formatTime(msg.createdAt)}
                          {msg.isEdited && <span style={{ color: '#888', marginLeft: '5px' }}>(edited)</span>}
                        </span>
                        
                        {/* Message action buttons */}
                        <div style={styles.messageActions}>
                          <button 
                            style={styles.actionBtn} 
                            onClick={() => startReplyMessage(msg)}
                            title="Reply"
                          >
                            ↩️
                          </button>
                          {canEdit && (
                            <button 
                              style={styles.actionBtn} 
                              onClick={() => startEditMessage(msg)}
                              title="Edit"
                            >
                              ✏️
                            </button>
                          )}
                          {canDelete && (
                            <button 
                              style={styles.actionBtn} 
                              onClick={() => handleDeleteMessage(msg.id)}
                              title="Delete"
                            >
                              🗑️
                            </button>
                          )}
                        </div>
                      </div>
                      
                      {/* Message content or edit form */}
                      {editingMessage?.id === msg.id ? (
                        <div style={styles.editForm}>
                          <input
                            style={styles.editInput}
                            type="text"
                            value={editContent}
                            onChange={e => setEditContent(e.target.value)}
                            onKeyPress={e => e.key === 'Enter' && handleEditMessage()}
                            autoFocus
                          />
                          <button style={styles.editSaveBtn} onClick={handleEditMessage}>Save</button>
                          <button style={styles.editCancelBtn} onClick={cancelEdit}>Cancel</button>
                        </div>
                      ) : (
                        <div style={styles.messageContent}>{msg.content}</div>
                      )}
                      
                      {/* Reactions display and picker */}
                      <div style={styles.reactionsContainer}>
                        {/* Existing reactions */}
                        {msg.reactions && msg.reactions.length > 0 && (
                          <div style={styles.reactionsRow}>
                            {msg.reactions.map((reaction, idx) => (
                              <button
                                key={idx}
                                style={{
                                  ...styles.reactionBadge,
                                  ...(reaction.hasReacted ? styles.reactionBadgeActive : {})
                                }}
                                onClick={() => toggleReaction(msg.id, reaction.emoji)}
                                title={reaction.name}
                              >
                                <span>{reaction.emoji}</span>
                                <span style={styles.reactionCount}>{reaction.count}</span>
                              </button>
                            ))}
                          </div>
                        )}
                        
                        {/* Add reaction button */}
                        <div style={{ position: 'relative', display: 'inline-block' }}>
                          <button
                            style={styles.addReactionBtn}
                            onClick={() => setShowReactionPicker(showReactionPicker === msg.id ? null : msg.id)}
                            title="Add reaction"
                          >
                            😀+
                          </button>
                          
                          {/* Reaction picker dropdown */}
                          {showReactionPicker === msg.id && (
                            <div style={styles.reactionPicker}>
                              {reactionTypes.map(rt => (
                                <button
                                  key={rt.id}
                                  style={styles.reactionPickerItem}
                                  onClick={() => toggleReaction(msg.id, rt.emoji)}
                                  title={rt.name}
                                >
                                  {rt.emoji}
                                </button>
                              ))}
                            </div>
                          )}
                        </div>
                      </div>
                    </div>
                  );
                })
              )}
            </div>

            {/* Reply preview banner */}
            {replyingTo && (
              <div style={styles.replyBanner}>
                <span>↩️ Replying to <strong>{replyingTo.user.username}</strong></span>
                <button style={styles.cancelReplyBtn} onClick={cancelReply}>✕</button>
              </div>
            )}

            <div style={styles.inputContainer}>
              <input
                style={styles.messageInput}
                type="text"
                placeholder={replyingTo ? `Reply to ${replyingTo.user.username}...` : (selectedChannel ? `Message #${selectedChannel.name}` : 'Select a channel...')}
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
                <option value="engagement">Engagement</option>
                <option value="forum">Forum Activity</option>
              </select>
            </div>
            <div style={styles.leaderboardList}>
              {leaderboard.map((entry: any) => (
                <div key={entry.userId} style={styles.leaderboardEntry}>
                  {/* Rank */}
                  <div style={styles.leaderboardRankBadge}>
                    <span style={{
                      ...styles.rankNumber,
                      color: entry.rank === 1 ? '#fbbf24' : entry.rank === 2 ? '#c0c0c0' : entry.rank === 3 ? '#cd7f32' : '#888'
                    }}>
                      #{entry.rank}
                    </span>
                  </div>
                  
                  {/* All Badges */}
                  <div style={styles.badgeStack}>
                    {/* Role badge first if exists */}
                    {entry.roleBadge && (
                      <span style={{ ...styles.roleBadge, color: entry.roleBadge.color }} title={entry.roleBadge.name}>
                        {entry.roleBadge.icon}
                      </span>
                    )}
                    {/* Primary achievement badge */}
                    <span style={{ color: entry.primaryBadge?.color || '#4ade80' }} title={entry.primaryBadge?.name || 'Newcomer'}>
                      {entry.primaryBadge?.icon || '🌱'}
                    </span>
                    {/* Additional badges (show up to 3 more) */}
                    {entry.badges?.slice(0, 3).filter((b: any) => !b.isPrimary).map((badge: any, idx: number) => (
                      <span key={idx} style={{ color: badge.color, fontSize: '0.9rem' }} title={badge.name}>
                        {badge.icon}
                      </span>
                    ))}
                    {entry.badges?.length > 4 && (
                      <span style={styles.moreBadges} title={`+${entry.badges.length - 4} more badges`}>+{entry.badges.length - 4}</span>
                    )}
                  </div>

                  {/* Username and Role */}
                  <div style={styles.userInfo}>
                    <span style={{
                      ...styles.leaderboardName,
                      color: entry.role === 'super_admin' ? '#fbbf24' : entry.role === 'admin' ? '#ef4444' : entry.role === 'moderator' ? '#f97316' : '#e0e0e0'
                    }}>
                      {entry.username}
                    </span>
                    {entry.role !== 'user' && (
                      <span style={styles.roleTag}>{entry.role.replace('_', ' ')}</span>
                    )}
                  </div>

                  {/* Stats */}
                  <div style={styles.statsGrid}>
                    <div style={styles.statItem}>
                      <span style={styles.statLabel}>Hashrate</span>
                      <span style={styles.statValue}>{formatHashrate(entry.stats?.currentHashrate || 0)}</span>
                    </div>
                    <div style={styles.statItem}>
                      <span style={styles.statLabel}>Miners</span>
                      <span style={styles.statValue}>{entry.stats?.activeMiners || 0}</span>
                    </div>
                    <div style={styles.statItem}>
                      <span style={styles.statLabel}>Blocks</span>
                      <span style={styles.statValue}>{entry.stats?.blocksFound || 0}</span>
                    </div>
                    <div style={styles.statItem}>
                      <span style={styles.statLabel}>Shares</span>
                      <span style={styles.statValue}>{formatNumber(entry.stats?.totalShares || 0)}</span>
                    </div>
                    <div style={styles.statItem}>
                      <span style={styles.statLabel}>Posts</span>
                      <span style={styles.statValue}>{entry.stats?.forumPosts || 0}</span>
                    </div>
                    <div style={styles.statItem}>
                      <span style={styles.statLabel}>Clout</span>
                      <span style={{ ...styles.statValue, color: '#00d4ff' }}>{formatNumber(entry.stats?.engagementScore || 0)}</span>
                    </div>
                  </div>
                </div>
              ))}
            </div>
            
            {/* Pagination Controls */}
            <div style={styles.paginationContainer}>
              <div style={styles.paginationInfo}>
                {myRank && (
                  <button 
                    style={styles.findMeButton}
                    onClick={() => { if (myPage) { setLeaderboardPage(myPage); } }}
                    title={`Your rank: #${myRank}`}
                  >
                    📍 Find Me (#{myRank})
                  </button>
                )}
                <span style={styles.paginationText}>
                  Showing {((leaderboardPage - 1) * leaderboardPageSize) + 1}-{Math.min(leaderboardPage * leaderboardPageSize, leaderboardPagination.totalUsers)} of {leaderboardPagination.totalUsers} miners
                </span>
              </div>
              <div style={styles.paginationControls}>
                <select 
                  style={styles.pageSizeSelect}
                  value={leaderboardPageSize}
                  onChange={e => { setLeaderboardPageSize(Number(e.target.value)); setLeaderboardPage(1); }}
                >
                  <option value={20}>20 per page</option>
                  <option value={50}>50 per page</option>
                  <option value={100}>100 per page</option>
                </select>
                <button 
                  style={{ ...styles.pageButton, opacity: leaderboardPage === 1 ? 0.5 : 1 }}
                  onClick={() => setLeaderboardPage(1)}
                  disabled={leaderboardPage === 1}
                >⏮ First</button>
                <button 
                  style={{ ...styles.pageButton, opacity: leaderboardPage === 1 ? 0.5 : 1 }}
                  onClick={() => setLeaderboardPage(p => Math.max(1, p - 1))}
                  disabled={leaderboardPage === 1}
                >◀ Prev</button>
                <span style={styles.pageIndicator}>
                  Page {leaderboardPage} of {leaderboardPagination.totalPages}
                </span>
                <button 
                  style={{ ...styles.pageButton, opacity: leaderboardPage >= leaderboardPagination.totalPages ? 0.5 : 1 }}
                  onClick={() => setLeaderboardPage(p => Math.min(leaderboardPagination.totalPages, p + 1))}
                  disabled={leaderboardPage >= leaderboardPagination.totalPages}
                >Next ▶</button>
                <button 
                  style={{ ...styles.pageButton, opacity: leaderboardPage >= leaderboardPagination.totalPages ? 0.5 : 1 }}
                  onClick={() => setLeaderboardPage(leaderboardPagination.totalPages)}
                  disabled={leaderboardPage >= leaderboardPagination.totalPages}
                >Last ⏭</button>
              </div>
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
  pageContainer: { display: 'flex', height: 'calc(100vh - 100px)', background: 'linear-gradient(180deg, #0D0811 0%, #1A0F1E 100%)' },
  leftSidebar: { width: '260px', background: 'linear-gradient(180deg, rgba(45, 31, 61, 0.4) 0%, rgba(26, 15, 30, 0.6) 100%)', borderRight: '1px solid rgba(74, 44, 90, 0.4)', display: 'flex', flexDirection: 'column', overflowY: 'auto' },
  sidebarHeader: { padding: '16px', borderBottom: '1px solid rgba(74, 44, 90, 0.4)', color: '#D4A84B', fontWeight: 600, fontSize: '1.05rem', display: 'flex', justifyContent: 'space-between', alignItems: 'center' },
  addBtn: { background: 'none', border: 'none', cursor: 'pointer', fontSize: '1rem', padding: '4px 8px', borderRadius: '6px', color: '#B8B4C8', transition: 'all 0.2s' },
  emptyState: { padding: '20px', textAlign: 'center', color: '#B8B4C8' },
  category: { marginBottom: '5px' },
  categoryHeader: { display: 'flex', alignItems: 'center', gap: '8px', padding: '10px 15px', color: '#B8B4C8', fontSize: '0.8rem', cursor: 'pointer', textTransform: 'uppercase', letterSpacing: '0.03em' },
  categoryName: { fontWeight: 600 },
  channel: { display: 'flex', alignItems: 'center', gap: '8px', padding: '10px 15px 10px 25px', color: '#B8B4C8', cursor: 'pointer', borderRadius: '8px', margin: '2px 8px', transition: 'all 0.2s' },
  channelActive: { background: 'rgba(212, 168, 75, 0.15)', color: '#F0EDF4', borderLeft: '2px solid #D4A84B' },
  channelHash: { color: '#7B5EA7' },
  channelBadge: { marginLeft: 'auto', fontSize: '0.8rem' },
  onlineSection: { marginTop: 'auto', borderTop: '1px solid rgba(74, 44, 90, 0.4)', padding: '12px' },
  onlineHeader: { padding: '8px', color: '#B8B4C8', fontSize: '0.75rem', textTransform: 'uppercase', letterSpacing: '0.03em' },
  onlineUser: { display: 'flex', alignItems: 'center', gap: '8px', padding: '6px 8px', fontSize: '0.9rem' },
  onlineIndicator: { width: '8px', height: '8px', borderRadius: '50%', backgroundColor: '#4ade80', boxShadow: '0 0 6px rgba(74, 222, 128, 0.5)' },
  onlineUsername: { color: '#F0EDF4' },
  mainContent: { flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' },
  secondaryNav: { display: 'flex', gap: '8px', padding: '12px 20px', borderBottom: '1px solid rgba(74, 44, 90, 0.4)', background: 'rgba(45, 31, 61, 0.3)' },
  secondaryTab: { padding: '10px 20px', backgroundColor: 'transparent', border: 'none', color: '#B8B4C8', fontSize: '0.95rem', cursor: 'pointer', borderRadius: '8px', transition: 'all 0.2s', fontWeight: 500 },
  secondaryTabActive: { background: 'rgba(212, 168, 75, 0.2)', color: '#D4A84B' },
  chatContainer: { flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' },
  chatHeader: { padding: '16px 20px', borderBottom: '1px solid rgba(74, 44, 90, 0.4)', background: 'rgba(45, 31, 61, 0.3)' },
  chatChannelName: { color: '#F0EDF4', fontWeight: 600, fontSize: '1.1rem' },
  chatChannelDesc: { color: '#B8B4C8', fontSize: '0.9rem', marginLeft: '15px' },
  messagesContainer: { flex: 1, overflowY: 'auto', padding: '20px' },
  noMessages: { textAlign: 'center', color: '#B8B4C8', padding: '40px' },
  message: { marginBottom: '20px' },
  messageHeader: { display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '4px' },
  messageBadge: { fontSize: '1.1rem' },
  messageUsername: { color: '#D4A84B', fontWeight: 600 },
  messageTime: { color: '#7A7490', fontSize: '0.8rem' },
  messageContent: { color: '#F0EDF4', paddingLeft: '28px' },
  inputContainer: { display: 'flex', gap: '10px', padding: '16px 20px', borderTop: '1px solid rgba(74, 44, 90, 0.4)', background: 'rgba(45, 31, 61, 0.3)' },
  messageInput: { flex: 1, padding: '12px 16px', backgroundColor: 'rgba(13, 8, 17, 0.8)', border: '1px solid rgba(74, 44, 90, 0.5)', borderRadius: '10px', color: '#F0EDF4', fontSize: '1rem', transition: 'border-color 0.2s' },
  sendBtn: { padding: '12px 24px', background: 'linear-gradient(135deg, #D4A84B 0%, #B8923A 100%)', border: 'none', borderRadius: '10px', color: '#1A0F1E', fontWeight: 600, cursor: 'pointer', boxShadow: '0 2px 8px rgba(212, 168, 75, 0.3)' },
  leaderboardContainer: { padding: '20px', overflowY: 'auto' },
  leaderboardHeader: { display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '20px', color: '#D4A84B' },
  leaderboardSelect: { padding: '10px 16px', backgroundColor: 'rgba(13, 8, 17, 0.8)', border: '1px solid rgba(74, 44, 90, 0.5)', borderRadius: '8px', color: '#F0EDF4' },
  leaderboardList: { display: 'flex', flexDirection: 'column', gap: '12px' },
  leaderboardEntry: { display: 'flex', alignItems: 'center', gap: '12px', padding: '18px 22px', background: 'linear-gradient(180deg, rgba(45, 31, 61, 0.5) 0%, rgba(26, 15, 30, 0.7) 100%)', borderRadius: '14px', border: '1px solid rgba(74, 44, 90, 0.4)', flexWrap: 'wrap', transition: 'all 0.2s' },
  leaderboardRankBadge: { minWidth: '45px', textAlign: 'center' },
  rankNumber: { fontWeight: 700, fontSize: '1.1rem' },
  badgeStack: { display: 'flex', alignItems: 'center', gap: '4px', fontSize: '1.2rem', minWidth: '80px' },
  roleBadge: { fontSize: '1.3rem' },
  moreBadges: { fontSize: '0.7rem', color: '#B8B4C8', backgroundColor: 'rgba(74, 44, 90, 0.5)', padding: '2px 6px', borderRadius: '10px' },
  userInfo: { display: 'flex', flexDirection: 'column', gap: '2px', minWidth: '120px', flex: '1' },
  leaderboardName: { fontWeight: 600, fontSize: '1rem' },
  roleTag: { fontSize: '0.7rem', color: '#7B5EA7', textTransform: 'uppercase', letterSpacing: '0.5px' },
  statsGrid: { display: 'grid', gridTemplateColumns: 'repeat(6, 1fr)', gap: '12px', marginLeft: 'auto' },
  statItem: { display: 'flex', flexDirection: 'column', alignItems: 'center', minWidth: '60px' },
  statLabel: { fontSize: '0.65rem', color: '#B8B4C8', textTransform: 'uppercase', letterSpacing: '0.5px' },
  statValue: { fontSize: '0.9rem', fontWeight: 600, color: '#F0EDF4' },
  leaderboardValue: { color: '#D4A84B', fontWeight: 600 },
  forumsContainer: { padding: '20px' },
  loading: { display: 'flex', justifyContent: 'center', alignItems: 'center', height: 'calc(100vh - 100px)', color: '#D4A84B', fontSize: '1.2rem' },
  modalOverlay: { position: 'fixed', top: 0, left: 0, right: 0, bottom: 0, backgroundColor: 'rgba(13, 8, 17, 0.92)', backdropFilter: 'blur(8px)', display: 'flex', justifyContent: 'center', alignItems: 'center', zIndex: 1000 },
  modal: { background: 'linear-gradient(180deg, #2D1F3D 0%, #1A0F1E 100%)', padding: '28px', borderRadius: '16px', border: '1px solid rgba(212, 168, 75, 0.3)', width: '100%', maxWidth: '450px', boxShadow: '0 24px 48px rgba(0, 0, 0, 0.5)' },
  modalTitle: { color: '#D4A84B', margin: '0 0 20px', fontWeight: 600 },
  formGroup: { marginBottom: '16px' },
  label: { display: 'block', color: '#B8B4C8', marginBottom: '6px', fontSize: '0.9rem', fontWeight: 500 },
  input: { width: '100%', padding: '12px 14px', backgroundColor: 'rgba(13, 8, 17, 0.8)', border: '1px solid rgba(74, 44, 90, 0.5)', borderRadius: '10px', color: '#F0EDF4', fontSize: '1rem', boxSizing: 'border-box', transition: 'border-color 0.2s' },
  select: { width: '100%', padding: '12px 14px', backgroundColor: 'rgba(13, 8, 17, 0.8)', border: '1px solid rgba(74, 44, 90, 0.5)', borderRadius: '10px', color: '#F0EDF4', fontSize: '1rem' },
  modalActions: { display: 'flex', gap: '12px', marginTop: '24px' },
  cancelBtn: { flex: 1, padding: '12px', backgroundColor: 'transparent', border: '1px solid rgba(74, 44, 90, 0.5)', borderRadius: '10px', color: '#B8B4C8', cursor: 'pointer', transition: 'all 0.2s' },
  submitBtn: { flex: 1, padding: '12px', background: 'linear-gradient(135deg, #D4A84B 0%, #B8923A 100%)', border: 'none', borderRadius: '10px', color: '#1A0F1E', fontWeight: 600, cursor: 'pointer', boxShadow: '0 2px 8px rgba(212, 168, 75, 0.3)' },
  
  // Message action styles
  messageActions: { marginLeft: 'auto', display: 'flex', gap: '4px', opacity: 0.6 },
  actionBtn: { background: 'none', border: 'none', cursor: 'pointer', fontSize: '0.9rem', padding: '2px 6px', borderRadius: '4px', transition: 'all 0.2s' },
  replyIndicator: { paddingLeft: '28px', marginBottom: '4px', fontSize: '0.85rem', borderLeft: '2px solid rgba(74, 44, 90, 0.5)', paddingTop: '2px', paddingBottom: '2px' },
  replyBanner: { display: 'flex', justifyContent: 'space-between', alignItems: 'center', padding: '10px 20px', background: 'rgba(212, 168, 75, 0.1)', borderTop: '1px solid rgba(212, 168, 75, 0.3)', color: '#F0EDF4', fontSize: '0.9rem' },
  cancelReplyBtn: { background: 'none', border: 'none', color: '#B8B4C8', cursor: 'pointer', fontSize: '1rem', padding: '4px 8px' },
  editForm: { display: 'flex', gap: '8px', paddingLeft: '28px', alignItems: 'center' },
  editInput: { flex: 1, padding: '10px 14px', backgroundColor: 'rgba(13, 8, 17, 0.8)', border: '1px solid rgba(212, 168, 75, 0.4)', borderRadius: '8px', color: '#F0EDF4', fontSize: '0.95rem' },
  editSaveBtn: { padding: '10px 18px', background: 'linear-gradient(135deg, #4ADE80 0%, #22C55E 100%)', border: 'none', borderRadius: '8px', color: '#1A0F1E', fontWeight: 600, cursor: 'pointer', fontSize: '0.85rem' },
  editCancelBtn: { padding: '10px 18px', backgroundColor: 'transparent', border: '1px solid rgba(74, 44, 90, 0.5)', borderRadius: '8px', color: '#B8B4C8', cursor: 'pointer', fontSize: '0.85rem' },
  
  // Reactions styles
  reactionsContainer: { display: 'flex', alignItems: 'center', gap: '6px', paddingLeft: '28px', marginTop: '6px', flexWrap: 'wrap' as const },
  reactionsRow: { display: 'flex', gap: '4px', flexWrap: 'wrap' as const },
  reactionBadge: { display: 'flex', alignItems: 'center', gap: '4px', padding: '4px 8px', background: 'rgba(74, 44, 90, 0.3)', border: '1px solid rgba(74, 44, 90, 0.5)', borderRadius: '12px', cursor: 'pointer', fontSize: '0.85rem', transition: 'all 0.2s', color: '#F0EDF4' },
  reactionBadgeActive: { background: 'rgba(212, 168, 75, 0.2)', borderColor: 'rgba(212, 168, 75, 0.5)' },
  reactionCount: { fontSize: '0.75rem', color: '#B8B4C8' },
  addReactionBtn: { background: 'rgba(74, 44, 90, 0.2)', border: '1px solid rgba(74, 44, 90, 0.4)', borderRadius: '12px', padding: '4px 8px', cursor: 'pointer', fontSize: '0.8rem', color: '#B8B4C8', transition: 'all 0.2s' },
  reactionPicker: { position: 'absolute' as const, bottom: '100%', left: 0, marginBottom: '8px', display: 'flex', flexWrap: 'wrap' as const, gap: '4px', padding: '10px', background: 'linear-gradient(180deg, #2D1F3D 0%, #1A0F1E 100%)', border: '1px solid rgba(74, 44, 90, 0.5)', borderRadius: '12px', boxShadow: '0 8px 24px rgba(0, 0, 0, 0.4)', maxWidth: '200px', zIndex: 100 },
  reactionPickerItem: { background: 'none', border: 'none', padding: '6px', cursor: 'pointer', fontSize: '1.2rem', borderRadius: '6px', transition: 'all 0.2s' },
  
  paginationContainer: { display: 'flex', justifyContent: 'space-between', alignItems: 'center', padding: '16px 20px', background: 'rgba(45, 31, 61, 0.3)', borderTop: '1px solid rgba(74, 44, 90, 0.4)', marginTop: '10px', borderRadius: '0 0 14px 14px', flexWrap: 'wrap' as const, gap: '10px' },
  paginationInfo: { display: 'flex', alignItems: 'center', gap: '15px' },
  paginationText: { color: '#B8B4C8', fontSize: '0.9rem' },
  paginationControls: { display: 'flex', alignItems: 'center', gap: '8px' },
  pageSizeSelect: { padding: '8px 12px', backgroundColor: 'rgba(13, 8, 17, 0.8)', border: '1px solid rgba(74, 44, 90, 0.5)', borderRadius: '8px', color: '#F0EDF4', fontSize: '0.85rem', cursor: 'pointer' },
  pageButton: { padding: '8px 14px', background: 'rgba(74, 44, 90, 0.4)', border: 'none', borderRadius: '8px', color: '#F0EDF4', cursor: 'pointer', fontSize: '0.85rem', transition: 'all 0.2s' },
  pageIndicator: { padding: '8px 14px', color: '#D4A84B', fontSize: '0.9rem', fontWeight: 600 },
  findMeButton: { padding: '10px 18px', background: 'linear-gradient(135deg, #D4A84B 0%, #B8923A 100%)', border: 'none', borderRadius: '8px', color: '#1A0F1E', cursor: 'pointer', fontSize: '0.85rem', fontWeight: 600, transition: 'all 0.2s', boxShadow: '0 2px 8px rgba(212, 168, 75, 0.3)' },
};

export default CommunityPage;
