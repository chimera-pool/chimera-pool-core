import React, { useState, useEffect } from 'react';

interface AdminNetworkTabProps {
  token: string;
  isActive: boolean;
  showMessage: (type: 'success' | 'error', text: string) => void;
}

interface NetworkConfig {
  id: string;
  name: string;
  display_name: string;
  symbol: string;
  algorithm: string;
  rpc_url: string;
  stratum_port: number;
  pool_fee_percent: number;
  min_payout_threshold: number;
  pool_wallet_address: string;
  is_active: boolean;
  is_default: boolean;
  description?: string;
}

interface NetworkFormState {
  display_name: string;
  rpc_url: string;
  rpc_user: string;
  rpc_password: string;
  pool_wallet_address: string;
  stratum_port: string;
  pool_fee_percent: string;
  min_payout_threshold: string;
  description: string;
}

const styles: { [key: string]: React.CSSProperties } = {
  container: { padding: '20px' },
  header: { marginBottom: '25px' },
  title: { color: '#D4A84B', marginTop: 0, marginBottom: '10px', fontWeight: 600 },
  desc: { color: '#B8B4C8', margin: 0 },
  loading: { padding: '40px', textAlign: 'center', color: '#D4A84B' },
  formInput: { width: '100%', padding: '12px 14px', backgroundColor: 'rgba(13, 8, 17, 0.8)', border: '1px solid rgba(74, 44, 90, 0.5)', borderRadius: '10px', color: '#F0EDF4', boxSizing: 'border-box' as const },
  cancelBtn: { flex: 1, padding: '12px', backgroundColor: 'transparent', border: '1px solid rgba(74, 44, 90, 0.5)', borderRadius: '10px', color: '#B8B4C8', cursor: 'pointer' },
  saveBtn: { flex: 1, padding: '12px', background: 'linear-gradient(135deg, #D4A84B 0%, #B8923A 100%)', border: 'none', borderRadius: '10px', color: '#1A0F1E', fontWeight: 600, cursor: 'pointer' },
};

export function AdminNetworkTab({ token, isActive, showMessage }: AdminNetworkTabProps) {
  const [networks, setNetworks] = useState<NetworkConfig[]>([]);
  const [activeNetwork, setActiveNetwork] = useState<NetworkConfig | null>(null);
  const [networkHistory, setNetworkHistory] = useState<any[]>([]);
  const [networksLoading, setNetworksLoading] = useState(false);
  const [editingNetwork, setEditingNetwork] = useState<NetworkConfig | null>(null);
  const [switchReason, setSwitchReason] = useState('');
  const [networkForm, setNetworkForm] = useState<NetworkFormState>({
    display_name: '',
    rpc_url: '',
    rpc_user: '',
    rpc_password: '',
    pool_wallet_address: '',
    stratum_port: '',
    pool_fee_percent: '',
    min_payout_threshold: '',
    description: ''
  });

  useEffect(() => {
    if (isActive) {
      fetchNetworks();
    }
  }, [isActive]);

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

  const startEditingNetwork = (network: NetworkConfig) => {
    setEditingNetwork(network);
    setNetworkForm({
      display_name: network.display_name,
      rpc_url: network.rpc_url,
      rpc_user: '',
      rpc_password: '',
      pool_wallet_address: network.pool_wallet_address || '',
      stratum_port: network.stratum_port?.toString() || '',
      pool_fee_percent: network.pool_fee_percent?.toString() || '',
      min_payout_threshold: network.min_payout_threshold?.toString() || '',
      description: network.description || ''
    });
  };

  if (!isActive) return null;

  return (
    <div style={styles.container}>
      <div style={styles.header}>
        <h3 style={styles.title}>🌐 Network Configuration</h3>
        <p style={styles.desc}>
          Configure blockchain networks for mining. Switch between networks to mine different cryptocurrencies.
        </p>
      </div>

      {networksLoading ? (
        <div style={styles.loading}>Loading networks...</div>
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
                style={{ ...styles.formInput, marginBottom: '10px' }}
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
                <div key={network.id} style={{ backgroundColor: '#1a1a2e', padding: '15px', borderRadius: '8px', border: `1px solid ${network.is_default ? '#4ade80' : '#2a2a4a'}` }}>
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '10px' }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
                      <span style={{ fontSize: '1.2rem' }}>{network.is_default ? '✅' : '⚪'}</span>
                      <strong style={{ color: '#e0e0e0' }}>{network.display_name}</strong>
                      <span style={{ backgroundColor: '#2a2a4a', color: '#888', padding: '2px 8px', borderRadius: '4px', fontSize: '0.8rem' }}>{network.symbol}</span>
                      <span style={{ backgroundColor: '#1a3a4a', color: '#00d4ff', padding: '2px 8px', borderRadius: '4px', fontSize: '0.8rem' }}>{network.algorithm}</span>
                    </div>
                    <div style={{ display: 'flex', gap: '8px' }}>
                      <button 
                        style={{ padding: '6px 12px', backgroundColor: '#2a2a4a', border: 'none', borderRadius: '4px', color: '#e0e0e0', cursor: 'pointer' }}
                        onClick={() => handleTestConnection(network.id)}
                      >
                        🔌 Test
                      </button>
                      <button 
                        style={{ padding: '6px 12px', backgroundColor: '#1a3a4a', border: 'none', borderRadius: '4px', color: '#00d4ff', cursor: 'pointer' }}
                        onClick={() => startEditingNetwork(network)}
                      >
                        ✏️ Edit
                      </button>
                    </div>
                  </div>
                  <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(150px, 1fr))', gap: '8px', fontSize: '0.85rem', color: '#888' }}>
                    <div>Port: <span style={{ color: '#e0e0e0' }}>{network.stratum_port}</span></div>
                    <div>Fee: <span style={{ color: '#e0e0e0' }}>{network.pool_fee_percent}%</span></div>
                    <div>Min Payout: <span style={{ color: '#e0e0e0' }}>{network.min_payout_threshold}</span></div>
                  </div>
                </div>
              ))}
            </div>
          </div>

          {/* Edit Network Modal */}
          {editingNetwork && (
            <div style={{ position: 'fixed', top: 0, left: 0, right: 0, bottom: 0, backgroundColor: 'rgba(0,0,0,0.8)', display: 'flex', justifyContent: 'center', alignItems: 'center', zIndex: 3000 }}>
              <div style={{ backgroundColor: '#1a1a2e', padding: '30px', borderRadius: '12px', maxWidth: '600px', width: '90%', maxHeight: '80vh', overflow: 'auto', border: '1px solid #2a2a4a' }}>
                <h3 style={{ color: '#00d4ff', margin: '0 0 20px' }}>✏️ Edit {editingNetwork.display_name}</h3>
                
                <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '15px' }}>
                  <div>
                    <label style={{ display: 'block', color: '#888', marginBottom: '5px' }}>Display Name</label>
                    <input style={styles.formInput} value={networkForm.display_name} onChange={e => setNetworkForm({...networkForm, display_name: e.target.value})} />
                  </div>
                  <div>
                    <label style={{ display: 'block', color: '#888', marginBottom: '5px' }}>Stratum Port</label>
                    <input style={styles.formInput} type="number" value={networkForm.stratum_port} onChange={e => setNetworkForm({...networkForm, stratum_port: e.target.value})} />
                  </div>
                  <div style={{ gridColumn: '1 / -1' }}>
                    <label style={{ display: 'block', color: '#888', marginBottom: '5px' }}>RPC URL</label>
                    <input style={styles.formInput} value={networkForm.rpc_url} onChange={e => setNetworkForm({...networkForm, rpc_url: e.target.value})} placeholder="https://rpc.example.com" />
                  </div>
                  <div>
                    <label style={{ display: 'block', color: '#888', marginBottom: '5px' }}>RPC User (optional)</label>
                    <input style={styles.formInput} value={networkForm.rpc_user} onChange={e => setNetworkForm({...networkForm, rpc_user: e.target.value})} />
                  </div>
                  <div>
                    <label style={{ display: 'block', color: '#888', marginBottom: '5px' }}>RPC Password (optional)</label>
                    <input style={styles.formInput} type="password" value={networkForm.rpc_password} onChange={e => setNetworkForm({...networkForm, rpc_password: e.target.value})} placeholder="Leave blank to keep current" />
                  </div>
                  <div style={{ gridColumn: '1 / -1' }}>
                    <label style={{ display: 'block', color: '#888', marginBottom: '5px' }}>Pool Wallet Address</label>
                    <input style={styles.formInput} value={networkForm.pool_wallet_address} onChange={e => setNetworkForm({...networkForm, pool_wallet_address: e.target.value})} />
                  </div>
                  <div>
                    <label style={{ display: 'block', color: '#888', marginBottom: '5px' }}>Pool Fee %</label>
                    <input style={styles.formInput} type="number" step="0.1" value={networkForm.pool_fee_percent} onChange={e => setNetworkForm({...networkForm, pool_fee_percent: e.target.value})} />
                  </div>
                  <div>
                    <label style={{ display: 'block', color: '#888', marginBottom: '5px' }}>Min Payout Threshold</label>
                    <input style={styles.formInput} type="number" step="0.001" value={networkForm.min_payout_threshold} onChange={e => setNetworkForm({...networkForm, min_payout_threshold: e.target.value})} />
                  </div>
                  <div style={{ gridColumn: '1 / -1' }}>
                    <label style={{ display: 'block', color: '#888', marginBottom: '5px' }}>Description</label>
                    <textarea style={{ ...styles.formInput, resize: 'vertical' }} rows={3} value={networkForm.description} onChange={e => setNetworkForm({...networkForm, description: e.target.value})} />
                  </div>
                </div>

                <div style={{ display: 'flex', gap: '10px', marginTop: '20px' }}>
                  <button style={styles.cancelBtn} onClick={() => setEditingNetwork(null)}>Cancel</button>
                  <button style={styles.saveBtn} onClick={() => handleUpdateNetwork(editingNetwork.id)}>💾 Save Changes</button>
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
  );
}

export default AdminNetworkTab;
