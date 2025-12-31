import React, { useState, useEffect } from 'react';

interface ReferralInfo {
  code: string;
  description: string;
  referrer_discount: number;
  referee_discount: number;
  times_used: number;
  max_uses: number | null;
  total_referrals: number;
  my_discount: number;
  effective_fee: number;
}

interface Referral {
  username: string;
  status: 'pending' | 'confirmed' | 'expired' | 'cancelled';
  created_at: string;
  confirmed_at: string | null;
  total_shares: number;
  total_hashrate: number;
  clout_bonus: number;
}

interface ReferralPanelProps {
  token: string;
  showMessage: (type: 'success' | 'error', text: string) => void;
}

function ReferralPanel({ token, showMessage }: ReferralPanelProps) {
  const [referralInfo, setReferralInfo] = useState<ReferralInfo | null>(null);
  const [referrals, setReferrals] = useState<Referral[]>([]);
  const [loading, setLoading] = useState(true);
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    fetchReferralData();
  }, []);

  const fetchReferralData = async () => {
    try {
      const [infoRes, listRes] = await Promise.all([
        fetch('/api/v1/user/referral', { headers: { Authorization: `Bearer ${token}` } }),
        fetch('/api/v1/user/referrals', { headers: { Authorization: `Bearer ${token}` } })
      ]);

      if (infoRes.ok) {
        const data = await infoRes.json();
        setReferralInfo(data);
      }

      if (listRes.ok) {
        const data = await listRes.json();
        setReferrals(data.referrals || []);
      }
    } catch (e) {
      console.error('Failed to fetch referral data:', e);
    }
    setLoading(false);
  };

  const copyReferralLink = () => {
    if (!referralInfo?.code) return;
    const link = `${window.location.origin}/register?ref=${referralInfo.code}`;
    navigator.clipboard.writeText(link);
    setCopied(true);
    showMessage('success', 'Referral link copied to clipboard!');
    setTimeout(() => setCopied(false), 2000);
  };

  const copyReferralCode = () => {
    if (!referralInfo?.code) return;
    navigator.clipboard.writeText(referralInfo.code);
    setCopied(true);
    showMessage('success', 'Referral code copied!');
    setTimeout(() => setCopied(false), 2000);
  };

  const formatHashrate = (hashrate: number): string => {
    if (hashrate === 0) return '0 H/s';
    const units = ['H/s', 'KH/s', 'MH/s', 'GH/s', 'TH/s', 'PH/s'];
    let unitIndex = 0;
    let value = hashrate;
    while (value >= 1000 && unitIndex < units.length - 1) {
      value /= 1000;
      unitIndex++;
    }
    return `${value.toFixed(2)} ${units[unitIndex]}`;
  };

  const formatDate = (dateStr: string) => {
    return new Date(dateStr).toLocaleDateString();
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'confirmed': return '#4ade80';
      case 'pending': return '#fbbf24';
      case 'expired': return '#6b7280';
      case 'cancelled': return '#ef4444';
      default: return '#B8B4C8';
    }
  };

  if (loading) {
    return <div style={styles.loading}>Loading referral data...</div>;
  }

  return (
    <div style={styles.container} data-testid="referral-panel">
      {/* Referral Code Section */}
      <div style={styles.codeSection}>
        <h3 style={styles.sectionTitle}>🔗 Your Referral Code</h3>
        
        {referralInfo ? (
          <>
            <div style={styles.codeDisplay}>
              <span style={styles.code} data-testid="referral-code">{referralInfo.code}</span>
              <button 
                style={styles.copyBtn} 
                onClick={copyReferralCode}
                data-testid="copy-code-btn"
              >
                {copied ? '✓ Copied' : '📋 Copy'}
              </button>
            </div>
            
            <button 
              style={styles.copyLinkBtn} 
              onClick={copyReferralLink}
              data-testid="copy-link-btn"
            >
              📤 Copy Referral Link
            </button>

            {/* Stats Grid */}
            <div style={styles.statsGrid}>
              <div style={styles.statCard}>
                <span style={styles.statValue}>{referralInfo.total_referrals}</span>
                <span style={styles.statLabel}>Total Referrals</span>
              </div>
              <div style={styles.statCard}>
                <span style={styles.statValue}>{referralInfo.referrer_discount}%</span>
                <span style={styles.statLabel}>Your Discount</span>
              </div>
              <div style={styles.statCard}>
                <span style={styles.statValue}>{referralInfo.referee_discount}%</span>
                <span style={styles.statLabel}>Friend Gets</span>
              </div>
              <div style={styles.statCard}>
                <span style={styles.statValue}>{referralInfo.effective_fee}%</span>
                <span style={styles.statLabel}>Your Pool Fee</span>
              </div>
            </div>

            <p style={styles.description}>{referralInfo.description}</p>
          </>
        ) : (
          <p style={styles.noData}>No referral code available</p>
        )}
      </div>

      {/* Referrals List */}
      <div style={styles.listSection}>
        <h3 style={styles.sectionTitle}>👥 Your Referrals ({referrals.length})</h3>
        
        {referrals.length === 0 ? (
          <div style={styles.emptyState}>
            <p>No referrals yet. Share your code to earn rewards!</p>
            <p style={styles.emptyHint}>Each confirmed referral reduces your pool fee.</p>
          </div>
        ) : (
          <div style={styles.referralList} data-testid="referral-list">
            {referrals.map((referral, idx) => (
              <div key={idx} style={styles.referralItem} data-testid="referral-item">
                <div style={styles.referralHeader}>
                  <span style={styles.referralUsername}>{referral.username}</span>
                  <span style={{
                    ...styles.referralStatus,
                    color: getStatusColor(referral.status)
                  }}>
                    {referral.status.toUpperCase()}
                  </span>
                </div>
                <div style={styles.referralStats}>
                  <span>📅 {formatDate(referral.created_at)}</span>
                  <span>⚡ {formatHashrate(referral.total_hashrate)}</span>
                  <span>📊 {referral.total_shares.toLocaleString()} shares</span>
                  {referral.clout_bonus > 0 && (
                    <span style={styles.cloutBonus}>+{referral.clout_bonus} clout</span>
                  )}
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}

const styles: { [key: string]: React.CSSProperties } = {
  container: {
    padding: '24px',
    background: 'linear-gradient(180deg, rgba(45, 31, 61, 0.4) 0%, rgba(26, 15, 30, 0.6) 100%)',
    borderRadius: '16px',
    border: '1px solid rgba(74, 44, 90, 0.4)',
  },
  loading: {
    padding: '40px',
    textAlign: 'center',
    color: '#B8B4C8',
  },
  codeSection: {
    marginBottom: '32px',
  },
  sectionTitle: {
    color: '#D4A84B',
    fontSize: '1.1rem',
    fontWeight: 600,
    marginBottom: '16px',
    margin: '0 0 16px 0',
  },
  codeDisplay: {
    display: 'flex',
    alignItems: 'center',
    gap: '12px',
    marginBottom: '16px',
  },
  code: {
    fontSize: '1.5rem',
    fontWeight: 700,
    color: '#F0EDF4',
    fontFamily: 'monospace',
    padding: '12px 20px',
    background: 'rgba(13, 8, 17, 0.8)',
    borderRadius: '10px',
    border: '1px solid rgba(212, 168, 75, 0.3)',
    letterSpacing: '2px',
  },
  copyBtn: {
    padding: '12px 20px',
    background: 'rgba(74, 44, 90, 0.4)',
    border: '1px solid rgba(74, 44, 90, 0.5)',
    borderRadius: '10px',
    color: '#F0EDF4',
    cursor: 'pointer',
    fontSize: '0.95rem',
    transition: 'all 0.2s',
  },
  copyLinkBtn: {
    width: '100%',
    padding: '14px',
    background: 'linear-gradient(135deg, #D4A84B 0%, #B8923A 100%)',
    border: 'none',
    borderRadius: '10px',
    color: '#1A0F1E',
    fontWeight: 600,
    cursor: 'pointer',
    fontSize: '1rem',
    marginBottom: '20px',
    boxShadow: '0 2px 8px rgba(212, 168, 75, 0.3)',
  },
  statsGrid: {
    display: 'grid',
    gridTemplateColumns: 'repeat(4, 1fr)',
    gap: '12px',
    marginBottom: '16px',
  },
  statCard: {
    display: 'flex',
    flexDirection: 'column',
    alignItems: 'center',
    padding: '16px',
    background: 'rgba(13, 8, 17, 0.6)',
    borderRadius: '12px',
    border: '1px solid rgba(74, 44, 90, 0.3)',
  },
  statValue: {
    fontSize: '1.4rem',
    fontWeight: 700,
    color: '#D4A84B',
  },
  statLabel: {
    fontSize: '0.75rem',
    color: '#B8B4C8',
    textTransform: 'uppercase',
    marginTop: '4px',
    letterSpacing: '0.5px',
  },
  description: {
    color: '#B8B4C8',
    fontSize: '0.9rem',
    margin: 0,
    lineHeight: 1.5,
  },
  noData: {
    color: '#7A7490',
    textAlign: 'center',
    padding: '20px',
  },
  listSection: {},
  emptyState: {
    textAlign: 'center',
    padding: '32px',
    color: '#B8B4C8',
  },
  emptyHint: {
    color: '#7A7490',
    fontSize: '0.85rem',
    marginTop: '8px',
  },
  referralList: {
    display: 'flex',
    flexDirection: 'column',
    gap: '12px',
  },
  referralItem: {
    padding: '16px',
    background: 'rgba(13, 8, 17, 0.5)',
    borderRadius: '12px',
    border: '1px solid rgba(74, 44, 90, 0.3)',
  },
  referralHeader: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: '8px',
  },
  referralUsername: {
    color: '#F0EDF4',
    fontWeight: 600,
    fontSize: '1rem',
  },
  referralStatus: {
    fontSize: '0.75rem',
    fontWeight: 600,
    padding: '4px 10px',
    borderRadius: '12px',
    background: 'rgba(0, 0, 0, 0.3)',
  },
  referralStats: {
    display: 'flex',
    gap: '16px',
    fontSize: '0.85rem',
    color: '#B8B4C8',
    flexWrap: 'wrap' as const,
  },
  cloutBonus: {
    color: '#4ade80',
    fontWeight: 600,
  },
};

export default ReferralPanel;
