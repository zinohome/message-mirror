import React, { useState, useEffect, useRef } from 'react';
import axios from 'axios';
import './App.css';

function App() {
  const [activeTab, setActiveTab] = useState('overview');
  const [config, setConfig] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [stats, setStats] = useState(null);
  const [configEditing, setConfigEditing] = useState(false);
  const [newConfig, setNewConfig] = useState('');
  const wsRef = useRef(null);

  // 获取配置
  const fetchConfig = async () => {
    try {
      const response = await axios.get('/api/config');
      setConfig(response.data);
      setNewConfig(JSON.stringify(response.data, null, 2));
      setError(null);
    } catch (err) {
      setError('Failed to fetch config: ' + err.message);
    } finally {
      setLoading(false);
    }
  };

  // 初始化WebSocket连接获取实时统计
  useEffect(() => {
    fetchConfig();

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    wsRef.current = new WebSocket(`${protocol}//${window.location.host}/ws/stats`);

    wsRef.current.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        setStats(data);
      } catch (e) {
        console.error('Failed to parse WebSocket message:', e);
      }
    };

    wsRef.current.onerror = (error) => {
      console.error('WebSocket error:', error);
      setError('WebSocket connection failed');
    };

    return () => {
      if (wsRef.current) {
        wsRef.current.close();
      }
    };
  }, []);

  // 保存配置
  const handleSaveConfig = async () => {
    try {
      const parsedConfig = JSON.parse(newConfig);
      await axios.post('/api/config', parsedConfig);
      setConfig(parsedConfig);
      setConfigEditing(false);
      setError(null);
      alert('Configuration saved successfully');
    } catch (err) {
      setError('Failed to save config: ' + err.message);
    }
  };

  // 重载配置
  const handleReloadConfig = async () => {
    try {
      await axios.post('/api/config/reload');
      setError(null);
      alert('Configuration reloaded successfully');
    } catch (err) {
      setError('Failed to reload config: ' + err.message);
    }
  };

  if (loading) {
    return <div className="loading">Loading...</div>;
  }

  return (
    <div className="app">
      <header className="header">
        <div className="header-content">
          <h1>Message Mirror</h1>
          <p>Message Mirroring & Configuration Management</p>
        </div>
      </header>

      {error && <div className="alert alert-error">{error}</div>}

      <div className="tabs">
        <button
          className={`tab ${activeTab === 'overview' ? 'active' : ''}`}
          onClick={() => setActiveTab('overview')}
        >
          Overview
        </button>
        <button
          className={`tab ${activeTab === 'config' ? 'active' : ''}`}
          onClick={() => setActiveTab('config')}
        >
          Configuration
        </button>
        <button
          className={`tab ${activeTab === 'monitoring' ? 'active' : ''}`}
          onClick={() => setActiveTab('monitoring')}
        >
          Monitoring
        </button>
      </div>

      <div className="container">
        {/* Overview Tab */}
        {activeTab === 'overview' && (
          <div className="tab-content">
            <h2>System Overview</h2>
            {stats ? (
              <div className="stats-grid">
                <div className="stat-card">
                  <div className="stat-label">Messages Consumed</div>
                  <div className="stat-value">{stats.messages_consumed?.toLocaleString() || 0}</div>
                </div>
                <div className="stat-card">
                  <div className="stat-label">Messages Produced</div>
                  <div className="stat-value">{stats.messages_produced?.toLocaleString() || 0}</div>
                </div>
                <div className="stat-card">
                  <div className="stat-label">Bytes Consumed</div>
                  <div className="stat-value">{formatBytes(stats.bytes_consumed || 0)}</div>
                </div>
                <div className="stat-card">
                  <div className="stat-label">Errors</div>
                  <div className="stat-value error">{stats.errors || 0}</div>
                </div>
              </div>
            ) : (
              <p>Waiting for statistics...</p>
            )}
          </div>
        )}

        {/* Configuration Tab */}
        {activeTab === 'config' && (
          <div className="tab-content">
            <h2>Configuration</h2>
            {configEditing ? (
              <div>
                <textarea
                  value={newConfig}
                  onChange={(e) => setNewConfig(e.target.value)}
                  className="json-editor"
                  rows="20"
                />
                <div className="button-group">
                  <button className="btn btn-primary" onClick={handleSaveConfig}>
                    Save
                  </button>
                  <button
                    className="btn btn-secondary"
                    onClick={() => {
                      setConfigEditing(false);
                      setNewConfig(JSON.stringify(config, null, 2));
                    }}
                  >
                    Cancel
                  </button>
                </div>
              </div>
            ) : (
              <div>
                <pre className="json-display">{JSON.stringify(config, null, 2)}</pre>
                <div className="button-group">
                  <button className="btn btn-primary" onClick={() => setConfigEditing(true)}>
                    Edit
                  </button>
                  <button className="btn btn-success" onClick={handleReloadConfig}>
                    Reload Config
                  </button>
                </div>
              </div>
            )}
          </div>
        )}

        {/* Monitoring Tab */}
        {activeTab === 'monitoring' && (
          <div className="tab-content">
            <h2>Real-time Monitoring</h2>
            {stats ? (
              <div className="monitoring-section">
                <h3>Current Statistics</h3>
                <table className="stats-table">
                  <tbody>
                    <tr>
                      <td>Messages Consumed</td>
                      <td>{stats.messages_consumed?.toLocaleString() || 0}</td>
                    </tr>
                    <tr>
                      <td>Messages Produced</td>
                      <td>{stats.messages_produced?.toLocaleString() || 0}</td>
                    </tr>
                    <tr>
                      <td>Bytes Consumed</td>
                      <td>{formatBytes(stats.bytes_consumed || 0)}</td>
                    </tr>
                    <tr>
                      <td>Bytes Produced</td>
                      <td>{formatBytes(stats.bytes_produced || 0)}</td>
                    </tr>
                    <tr>
                      <td>Total Errors</td>
                      <td className="error">{stats.errors || 0}</td>
                    </tr>
                    <tr>
                      <td>Uptime</td>
                      <td>{stats.uptime || 'N/A'}</td>
                    </tr>
                  </tbody>
                </table>
              </div>
            ) : (
              <p>Connecting to real-time data...</p>
            )}
          </div>
        )}
      </div>
    </div>
  );
}

function formatBytes(bytes) {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return Math.round((bytes / Math.pow(k, i)) * 100) / 100 + ' ' + sizes[i];
}

export default App;
