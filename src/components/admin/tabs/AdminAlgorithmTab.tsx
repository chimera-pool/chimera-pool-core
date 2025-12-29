import React, { useState, useEffect } from 'react';

interface AdminAlgorithmTabProps {
  token: string;
  isActive: boolean;
  showMessage: (type: 'success' | 'error', text: string) => void;
}

interface AlgorithmForm {
  algorithm: string;
  algorithm_variant: string;
  difficulty_target: string;
  block_time: string;
  stratum_port: string;
  algorithm_params: string;
}

const styles: { [key: string]: React.CSSProperties } = {
  container: { padding: '20px' },
  header: { marginBottom: '25px' },
  title: { color: '#D4A84B', marginTop: 0, marginBottom: '10px', fontWeight: 600 },
  desc: { color: '#B8B4C8', margin: 0, lineHeight: 1.6 },
  grid: { display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(280px, 1fr))', gap: '20px', marginBottom: '25px' },
  card: { backgroundColor: 'rgba(13, 8, 17, 0.6)', padding: '20px', borderRadius: '12px', border: '1px solid rgba(74, 44, 90, 0.3)' },
  label: { display: 'block', color: '#D4A84B', marginBottom: '10px', fontSize: '0.9rem', fontWeight: 500 },
  input: { width: '100%', padding: '12px 14px', backgroundColor: 'rgba(13, 8, 17, 0.8)', border: '1px solid rgba(74, 44, 90, 0.5)', borderRadius: '10px', color: '#F0EDF4', boxSizing: 'border-box' as const },
  select: { width: '100%', padding: '12px 14px', backgroundColor: 'rgba(13, 8, 17, 0.8)', border: '1px solid rgba(74, 44, 90, 0.5)', borderRadius: '10px', color: '#F0EDF4' },
  textarea: { width: '100%', padding: '12px 14px', backgroundColor: 'rgba(13, 8, 17, 0.8)', border: '1px solid rgba(74, 44, 90, 0.5)', borderRadius: '10px', color: '#F0EDF4', fontFamily: 'monospace', resize: 'vertical' as const, boxSizing: 'border-box' as const },
  hint: { color: '#888', fontSize: '0.8rem', marginTop: '8px', margin: 0 },
  saveBtn: { padding: '14px 28px', background: 'linear-gradient(135deg, #D4A84B 0%, #B8923A 100%)', border: 'none', borderRadius: '10px', color: '#1A0F1E', fontWeight: 600, cursor: 'pointer', fontSize: '1rem' },
  warning: { backgroundColor: 'rgba(251, 191, 36, 0.1)', border: '1px solid rgba(251, 191, 36, 0.3)', borderRadius: '8px', padding: '15px', marginTop: '20px', color: '#fbbf24' },
};

export function AdminAlgorithmTab({ token, isActive, showMessage }: AdminAlgorithmTabProps) {
  const [algorithmData, setAlgorithmData] = useState<any>(null);
  const [algorithmForm, setAlgorithmForm] = useState<AlgorithmForm>({
    algorithm: '',
    algorithm_variant: '',
    difficulty_target: '',
    block_time: '',
    stratum_port: '',
    algorithm_params: ''
  });
  const [savingAlgorithm, setSavingAlgorithm] = useState(false);
  const [customAlgoName, setCustomAlgoName] = useState('scrpy-variant');
  const [customAlgoCode, setCustomAlgoCode] = useState('');

  useEffect(() => {
    if (isActive) {
      fetchAlgorithmSettings();
    }
  }, [isActive]);

  const fetchAlgorithmSettings = async () => {
    try {
      const response = await fetch('/api/v1/admin/algorithm', {
        headers: { 'Authorization': `Bearer ${token}` }
      });
      if (response.ok) {
        const data = await response.json();
        setAlgorithmData(data);
        setAlgorithmForm({
          algorithm: data.algorithm || '',
          algorithm_variant: data.algorithm_variant || '',
          difficulty_target: data.difficulty_target?.toString() || '',
          block_time: data.block_time?.toString() || '',
          stratum_port: data.stratum_port?.toString() || '',
          algorithm_params: data.algorithm_params || ''
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
        headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
        body: JSON.stringify(algorithmForm)
      });
      if (response.ok) {
        showMessage('success', 'Algorithm settings saved successfully');
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

  if (!isActive) return null;

  return (
    <div style={styles.container}>
      <div style={styles.header}>
        <h3 style={styles.title}>⚙️ Mining Algorithm Configuration</h3>
        <p style={styles.desc}>
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

      <div style={styles.grid}>
        <div style={styles.card}>
          <label style={styles.label}>Algorithm Type</label>
          <select 
            style={styles.select}
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
          <p style={styles.hint}>Select "Custom" to define a new algorithm from BlockDAG specifications</p>
        </div>

        <div style={styles.card}>
          <label style={styles.label}>Algorithm Variant / Version</label>
          <input 
            style={styles.input}
            type="text"
            value={algorithmForm.algorithm_variant}
            onChange={e => setAlgorithmForm({...algorithmForm, algorithm_variant: e.target.value})}
            placeholder="e.g., scrpy-v1.0, blockdag-mainnet"
          />
          <p style={styles.hint}>Version identifier for the algorithm variant</p>
        </div>

        <div style={styles.card}>
          <label style={styles.label}>Base Difficulty</label>
          <input 
            style={styles.input}
            type="text"
            value={algorithmForm.difficulty_target}
            onChange={e => setAlgorithmForm({...algorithmForm, difficulty_target: e.target.value})}
            placeholder="e.g., 1.0"
          />
          <p style={styles.hint}>Starting difficulty for share validation</p>
        </div>

        <div style={styles.card}>
          <label style={styles.label}>Target Block Time (seconds)</label>
          <input 
            style={styles.input}
            type="text"
            value={algorithmForm.block_time}
            onChange={e => setAlgorithmForm({...algorithmForm, block_time: e.target.value})}
            placeholder="e.g., 10"
          />
          <p style={styles.hint}>Expected time between blocks</p>
        </div>

        <div style={styles.card}>
          <label style={styles.label}>Stratum Port</label>
          <input 
            style={styles.input}
            type="text"
            value={algorithmForm.stratum_port}
            onChange={e => setAlgorithmForm({...algorithmForm, stratum_port: e.target.value})}
            placeholder="e.g., 3333"
          />
          <p style={styles.hint}>Port for miner connections</p>
        </div>

        <div style={{...styles.card, gridColumn: '1 / -1'}}>
          <label style={styles.label}>Algorithm Parameters (JSON)</label>
          <textarea 
            style={styles.textarea}
            value={algorithmForm.algorithm_params}
            onChange={e => setAlgorithmForm({...algorithmForm, algorithm_params: e.target.value})}
            placeholder='{"N": 1024, "r": 1, "p": 1, "keyLen": 32}'
            rows={4}
          />
          <p style={styles.hint}>Scrypt parameters: N (CPU/memory cost), r (block size), p (parallelization), keyLen (output length)</p>
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
            style={{ ...styles.input, backgroundColor: '#1a1a2e', border: '1px solid #9b59b6' }}
            type="text"
            placeholder="e.g., blockdag-scrpy-v2, bdag-mainnet-algo"
            value={customAlgoName}
            onChange={e => setCustomAlgoName(e.target.value)}
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
            value={customAlgoCode}
            onChange={e => setCustomAlgoCode(e.target.value)}
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

      <div style={{ marginTop: '25px' }}>
        <button 
          style={styles.saveBtn}
          onClick={handleSaveAlgorithm}
          disabled={savingAlgorithm}
        >
          {savingAlgorithm ? 'Saving...' : '💾 Save Algorithm Settings'}
        </button>
      </div>

      <div style={styles.warning}>
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
  );
}

export default AdminAlgorithmTab;
