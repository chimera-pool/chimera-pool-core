import React, { useState, useEffect, useRef } from 'react';

interface Notification {
  id: number;
  type: string;
  title: string;
  message: string;
  link: string;
  isRead: boolean;
  createdAt: string;
  metadata?: {
    message_id?: number;
    channel_id?: number;
    author?: string;
  };
}

interface NotificationBellProps {
  token: string;
}

const NotificationBell: React.FC<NotificationBellProps> = ({ token }) => {
  const [notifications, setNotifications] = useState<Notification[]>([]);
  const [unreadCount, setUnreadCount] = useState(0);
  const [isOpen, setIsOpen] = useState(false);
  const [loading, setLoading] = useState(false);
  const dropdownRef = useRef<HTMLDivElement>(null);

  const fetchNotifications = async () => {
    if (!token) return;
    setLoading(true);
    try {
      const res = await fetch('/api/v1/community/notifications', {
        headers: { Authorization: `Bearer ${token}` }
      });
      if (res.ok) {
        const data = await res.json();
        setNotifications(data.notifications || []);
        setUnreadCount(data.unreadCount || 0);
      }
    } catch (e) {
      console.error('Failed to fetch notifications:', e);
    }
    setLoading(false);
  };

  const markAsRead = async (ids?: number[]) => {
    try {
      await fetch('/api/v1/community/notifications/read', {
        method: 'PUT',
        headers: {
          Authorization: `Bearer ${token}`,
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({ ids: ids || [] })
      });
      fetchNotifications();
    } catch (e) {
      console.error('Failed to mark notifications as read:', e);
    }
  };

  const handleNotificationClick = (notif: Notification) => {
    if (!notif.isRead) {
      markAsRead([notif.id]);
    }
    if (notif.link) {
      window.location.href = notif.link;
    }
    setIsOpen(false);
  };

  useEffect(() => {
    fetchNotifications();
    const interval = setInterval(fetchNotifications, 30000);
    return () => clearInterval(interval);
  }, [token]);

  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
        setIsOpen(false);
      }
    };
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  const getNotificationIcon = (type: string) => {
    switch (type) {
      case 'reply': return '💬';
      case 'mention': return '@';
      case 'reaction': return '👍';
      case 'broadcast': return '📢';
      case 'payout': return '💰';
      case 'worker_alert': return '⚠️';
      case 'system': return '🔔';
      default: return '🔔';
    }
  };

  const formatTime = (dateStr: string) => {
    const date = new Date(dateStr);
    const now = new Date();
    const diff = now.getTime() - date.getTime();
    const minutes = Math.floor(diff / 60000);
    const hours = Math.floor(diff / 3600000);
    const days = Math.floor(diff / 86400000);

    if (minutes < 1) return 'Just now';
    if (minutes < 60) return `${minutes}m ago`;
    if (hours < 24) return `${hours}h ago`;
    if (days < 7) return `${days}d ago`;
    return date.toLocaleDateString();
  };

  return (
    <div style={styles.container} ref={dropdownRef} data-testid="notification-bell">
      <button
        style={styles.bellButton}
        onClick={() => setIsOpen(!isOpen)}
        data-testid="notification-bell-btn"
        aria-label="Notifications"
      >
        🔔
        {unreadCount > 0 && (
          <span style={styles.badge} data-testid="notification-badge">
            {unreadCount > 99 ? '99+' : unreadCount}
          </span>
        )}
      </button>

      {isOpen && (
        <div style={styles.dropdown} data-testid="notification-dropdown">
          <div style={styles.header}>
            <span style={styles.headerTitle}>Notifications</span>
            {unreadCount > 0 && (
              <button
                style={styles.markAllBtn}
                onClick={() => markAsRead()}
                data-testid="mark-all-read-btn"
              >
                Mark all read
              </button>
            )}
          </div>

          <div style={styles.notificationList}>
            {loading && notifications.length === 0 ? (
              <div style={styles.emptyState}>Loading...</div>
            ) : notifications.length === 0 ? (
              <div style={styles.emptyState}>No notifications yet</div>
            ) : (
              notifications.map(notif => (
                <div
                  key={notif.id}
                  style={{
                    ...styles.notificationItem,
                    ...(notif.isRead ? {} : styles.unreadItem)
                  }}
                  onClick={() => handleNotificationClick(notif)}
                  data-testid={`notification-item-${notif.id}`}
                >
                  <div style={styles.notifIcon}>
                    {getNotificationIcon(notif.type)}
                  </div>
                  <div style={styles.notifContent}>
                    <div style={styles.notifTitle}>{notif.title}</div>
                    {notif.message && (
                      <div style={styles.notifMessage}>{notif.message}</div>
                    )}
                    <div style={styles.notifTime}>{formatTime(notif.createdAt)}</div>
                  </div>
                  {!notif.isRead && <div style={styles.unreadDot} />}
                </div>
              ))
            )}
          </div>
        </div>
      )}
    </div>
  );
};

const styles: { [key: string]: React.CSSProperties } = {
  container: {
    position: 'relative',
    display: 'inline-block',
  },
  bellButton: {
    background: 'transparent',
    border: 'none',
    fontSize: '1.4rem',
    cursor: 'pointer',
    padding: '8px',
    position: 'relative',
    transition: 'transform 0.2s',
  },
  badge: {
    position: 'absolute',
    top: '2px',
    right: '2px',
    background: 'linear-gradient(135deg, #EF4444 0%, #DC2626 100%)',
    color: 'white',
    fontSize: '0.65rem',
    fontWeight: 700,
    padding: '2px 5px',
    borderRadius: '10px',
    minWidth: '16px',
    textAlign: 'center',
  },
  dropdown: {
    position: 'absolute',
    top: '100%',
    right: 0,
    width: '360px',
    maxHeight: '480px',
    background: 'linear-gradient(180deg, #2D1F3D 0%, #1A0F1E 100%)',
    border: '1px solid rgba(74, 44, 90, 0.5)',
    borderRadius: '12px',
    boxShadow: '0 8px 32px rgba(0, 0, 0, 0.4)',
    zIndex: 1000,
    overflow: 'hidden',
  },
  header: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    padding: '14px 16px',
    borderBottom: '1px solid rgba(74, 44, 90, 0.4)',
  },
  headerTitle: {
    color: '#F0EDF4',
    fontWeight: 600,
    fontSize: '1rem',
  },
  markAllBtn: {
    background: 'transparent',
    border: 'none',
    color: '#D4A84B',
    fontSize: '0.85rem',
    cursor: 'pointer',
    padding: '4px 8px',
    borderRadius: '4px',
    transition: 'background 0.2s',
  },
  notificationList: {
    maxHeight: '400px',
    overflowY: 'auto',
  },
  notificationItem: {
    display: 'flex',
    alignItems: 'flex-start',
    gap: '12px',
    padding: '12px 16px',
    cursor: 'pointer',
    transition: 'background 0.2s',
    borderBottom: '1px solid rgba(74, 44, 90, 0.2)',
  },
  unreadItem: {
    background: 'rgba(212, 168, 75, 0.08)',
  },
  notifIcon: {
    fontSize: '1.2rem',
    width: '32px',
    height: '32px',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    background: 'rgba(74, 44, 90, 0.3)',
    borderRadius: '8px',
    flexShrink: 0,
  },
  notifContent: {
    flex: 1,
    minWidth: 0,
  },
  notifTitle: {
    color: '#F0EDF4',
    fontSize: '0.9rem',
    fontWeight: 500,
    marginBottom: '2px',
  },
  notifMessage: {
    color: '#B8B4C8',
    fontSize: '0.8rem',
    overflow: 'hidden',
    textOverflow: 'ellipsis',
    whiteSpace: 'nowrap',
  },
  notifTime: {
    color: '#8B8698',
    fontSize: '0.75rem',
    marginTop: '4px',
  },
  unreadDot: {
    width: '8px',
    height: '8px',
    background: '#D4A84B',
    borderRadius: '50%',
    flexShrink: 0,
    marginTop: '6px',
  },
  emptyState: {
    padding: '40px 20px',
    textAlign: 'center',
    color: '#8B8698',
    fontSize: '0.9rem',
  },
};

export default NotificationBell;
