import React, { useState, useEffect } from 'react';
import { IMiner, IMinerStatusMonitorProps, ConnectionQuality } from './interfaces';

// Utility function to format hashrate
const formatHashrate = (hashrate: number): string => {
  if (hashrate >= 1000000000) return `${(hashrate / 1000000000).toFixed(1)} GH/s`;
  if (hashrate >= 1000000) return `${(hashrate / 1000000).toFixed(0)} MH/s`;
  return `${(hashrate / 1000).toFixed(0)} KH/s`;
};

// Utility function to get connection quality
const getConnectionQuality = (lastSeen: string): 'excellent' | 'good' | 'poor' | 'offline' => {
  const now = Date.now();
  const lastSeenTime = new Date(lastSeen).getTime();
  const diffMinutes = (now - lastSeenTime) / (1000 * 60);
  
  if (diffMinutes < 1) return 'excellent';
  if (diffMinutes < 5) return 'good';
  if (diffMinutes < 30) return 'poor';
  return 'offline';
};

// Utility function to get connection quality color
const getConnectionQualityColor = (quality: string): string => {
  switch (quality) {
    case 'excellent': return 'bg-green-500';
    case 'good': return 'bg-yellow-500';
    case 'poor': return 'bg-orange-500';
    case 'offline': return 'bg-red-500';
    default: return 'bg-gray-500';
  }
};

// Main component
export const MinerStatusMonitor: React.FC<IMinerStatusMonitorProps> = ({
  className = '',
  websocketUrl = 'ws://localhost:8080/miners',
  apiUrl = '/api/v1/miners',
  network = 'litecoin'
}) => {
  const [miners, setMiners] = useState<IMiner[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [ws, setWs] = useState<WebSocket | null>(null);

  // Fetch initial miner data
  useEffect(() => {
    const fetchMiners = async () => {
      try {
        const response = await fetch(apiUrl);
        if (!response.ok) throw new Error('Failed to fetch miners');
        const data = await response.json();
        setMiners(data.miners || []);
      } catch (err) {
        setError('Connection error');
      } finally {
        setLoading(false);
      }
    };
    fetchMiners();
  }, [apiUrl]);

  // WebSocket connection for real-time updates
  useEffect(() => {
    try {
      const websocket = new WebSocket(websocketUrl);
      websocket.onmessage = (event) => {
        const message = JSON.parse(event.data);
        if (message.type === 'miner_update') {
          setMiners(prev => prev.map(miner =>
            miner.id === message.miner.id ? { ...miner, ...message.miner } : miner
          ));
        }
      };
      setWs(websocket);
    } catch (err) {
      setError('WebSocket connection failed');
    }
    return () => {
      if (ws) ws.close();
    };
  }, [websocketUrl]);

  // Loading state
  if (loading) {
    return (
      <div data-testid="miner-status-monitor" className={`miner-status-monitor responsive ${className}`}>
        <div className="loading-state">Loading miners...</div>
      </div>
    );
  }

  // Error state
  if (error) {
    return (
      <div data-testid="miner-status-monitor" className={`miner-status-monitor responsive ${className}`}>
        <div className="error-state">Connection error</div>
      </div>
    );
  }

  // Main render
  return (
    <div
      data-testid="miner-status-monitor"
      className={`miner-status-monitor responsive ${className}`}
      role="region"
      aria-label="Miner Status Monitor"
    >
      <h2>Connected Miners - {network.toUpperCase()}</h2>
      <table role="table" aria-label="Connected miners">
        <thead>
          <tr>
            <th>Status</th>
            <th>Hashrate</th>
            <th>Shares</th>
            <th>Algorithm</th>
            <th>Network</th>
          </tr>
        </thead>
        <tbody>
          {miners.map((miner) => (
            <tr key={miner.id}>
              <td>
                <div className="flex items-center gap-2">
                  <div
                    data-testid={`miner-status-${miner.status}`}
                    className={`status-indicator w-3 h-3 rounded-full ${getConnectionQualityColor(getConnectionQuality(miner.lastSeen))}`}
                    title={`Connection: ${getConnectionQuality(miner.lastSeen)} - Last seen: ${new Date(miner.lastSeen).toLocaleString()}`}
                  />
                  <span className={`text-sm font-medium ${
                    getConnectionQuality(miner.lastSeen) === 'excellent' ? 'text-green-700' :
                    getConnectionQuality(miner.lastSeen) === 'good' ? 'text-yellow-700' :
                    getConnectionQuality(miner.lastSeen) === 'poor' ? 'text-orange-700' :
                    'text-red-700'
                  }`}>
                    {getConnectionQuality(miner.lastSeen)}
                  </span>
                </div>
              </td>
              <td data-testid={`miner-hashrate-${miner.id}`}>
                {formatHashrate(miner.hashrate)}
              </td>
              <td data-testid={`miner-shares-${miner.id}`}>
                {miner.shares.accepted}/{miner.shares.rejected}
              </td>
              <td data-testid={`miner-algorithm-${miner.id}`}>
                {miner.algorithm}
              </td>
              <td data-testid={`miner-network-${miner.id}`}>
                <span className="px-2 py-1 text-xs font-medium bg-gray-100 text-gray-800 rounded">
                  {miner.network.toUpperCase()}
                </span>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
};