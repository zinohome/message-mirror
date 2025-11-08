package main

// getWebUIHTML 返回Web UI的HTML内容
func getWebUIHTML() string {
	return `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Message Mirror - 配置管理</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
            background: #f5f5f5;
            color: #333;
            line-height: 1.6;
        }
        .container {
            max-width: 1200px;
            margin: 0 auto;
            padding: 20px;
        }
        header {
            background: #fff;
            padding: 20px;
            border-radius: 8px;
            margin-bottom: 20px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        h1 {
            color: #1890ff;
            margin-bottom: 10px;
        }
        .tabs {
            display: flex;
            gap: 10px;
            margin-bottom: 20px;
            background: #fff;
            padding: 10px;
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        .tab {
            padding: 10px 20px;
            cursor: pointer;
            border: none;
            background: #f0f0f0;
            border-radius: 4px;
            transition: all 0.3s;
        }
        .tab.active {
            background: #1890ff;
            color: #fff;
        }
        .tab-content {
            display: none;
            background: #fff;
            padding: 20px;
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        .tab-content.active {
            display: block;
        }
        .form-group {
            margin-bottom: 20px;
        }
        label {
            display: block;
            margin-bottom: 5px;
            font-weight: 500;
            color: #555;
        }
        input, select, textarea {
            width: 100%;
            padding: 8px 12px;
            border: 1px solid #d9d9d9;
            border-radius: 4px;
            font-size: 14px;
        }
        input:focus, select:focus, textarea:focus {
            outline: none;
            border-color: #1890ff;
        }
        textarea {
            min-height: 100px;
            font-family: monospace;
            resize: vertical;
        }
        .btn {
            padding: 10px 20px;
            border: none;
            border-radius: 4px;
            cursor: pointer;
            font-size: 14px;
            transition: all 0.3s;
        }
        .btn-primary {
            background: #1890ff;
            color: #fff;
        }
        .btn-primary:hover {
            background: #40a9ff;
        }
        .btn-success {
            background: #52c41a;
            color: #fff;
        }
        .btn-success:hover {
            background: #73d13d;
        }
        .btn-danger {
            background: #ff4d4f;
            color: #fff;
        }
        .btn-danger:hover {
            background: #ff7875;
        }
        .btn-group {
            display: flex;
            gap: 10px;
            margin-top: 20px;
        }
        .alert {
            padding: 12px;
            border-radius: 4px;
            margin-bottom: 20px;
        }
        .alert-success {
            background: #f6ffed;
            border: 1px solid #b7eb8f;
            color: #52c41a;
        }
        .alert-error {
            background: #fff2f0;
            border: 1px solid #ffccc7;
            color: #ff4d4f;
        }
        .stats-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 20px;
            margin-bottom: 20px;
        }
        .stat-card {
            background: #fff;
            padding: 20px;
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        .stat-value {
            font-size: 24px;
            font-weight: bold;
            color: #1890ff;
        }
        .stat-label {
            color: #999;
            font-size: 14px;
            margin-top: 5px;
        }
        .json-editor {
            font-family: 'Courier New', monospace;
            font-size: 12px;
        }
        .section {
            margin-bottom: 30px;
        }
        .section-title {
            font-size: 18px;
            font-weight: 600;
            margin-bottom: 15px;
            color: #333;
            border-bottom: 2px solid #1890ff;
            padding-bottom: 5px;
        }
        .row {
            display: grid;
            grid-template-columns: 1fr 1fr;
            gap: 15px;
        }
        @media (max-width: 768px) {
            .row {
                grid-template-columns: 1fr;
            }
        }
    </style>
</head>
<body>
    <div class="container">
        <header>
            <h1>Message Mirror 配置管理</h1>
            <p>实时配置管理和监控</p>
        </header>

        <div class="tabs">
            <button class="tab active" onclick="switchTab('config')">配置管理</button>
            <button class="tab" onclick="switchTab('stats')">统计信息</button>
        </div>

        <div id="alert-container"></div>

        <div id="config-tab" class="tab-content active">
            <div class="section">
                <div class="section-title">数据源配置</div>
                <div class="form-group">
                    <label>数据源类型</label>
                    <select id="source-type">
                        <option value="kafka">Kafka</option>
                        <option value="rabbitmq">RabbitMQ</option>
                        <option value="file">文件监控</option>
                    </select>
                </div>
                <div id="source-config"></div>
            </div>

            <div class="section">
                <div class="section-title">目标配置</div>
                <div class="form-group">
                    <label>Brokers (逗号分隔)</label>
                    <input type="text" id="target-brokers" placeholder="localhost:9092,localhost:9093">
                </div>
                <div class="form-group">
                    <label>Topic</label>
                    <input type="text" id="target-topic" placeholder="target-topic">
                </div>
            </div>

            <div class="section">
                <div class="section-title">镜像配置</div>
                <div class="row">
                    <div class="form-group">
                        <label>Worker数量</label>
                        <input type="number" id="worker-count" min="1" value="4">
                    </div>
                    <div class="form-group">
                        <label>批处理大小</label>
                        <input type="number" id="batch-size" min="1" value="100">
                    </div>
                </div>
                <div class="row">
                    <div class="form-group">
                        <label>消费速率限制 (消息/秒, 0=不限制)</label>
                        <input type="number" id="consumer-rate-limit" min="0" value="0" step="0.1">
                    </div>
                    <div class="form-group">
                        <label>生产速率限制 (消息/秒, 0=不限制)</label>
                        <input type="number" id="producer-rate-limit" min="0" value="0" step="0.1">
                    </div>
                </div>
                <div class="form-group">
                    <label>
                        <input type="checkbox" id="batch-enabled"> 启用批处理
                    </label>
                </div>
            </div>

            <div class="section">
                <div class="section-title">高级配置 (JSON)</div>
                <div class="form-group">
                    <textarea id="config-json" class="json-editor" placeholder="完整配置JSON..."></textarea>
                </div>
            </div>

            <div class="btn-group">
                <button class="btn btn-primary" onclick="loadConfig()">加载配置</button>
                <button class="btn btn-success" onclick="saveConfig()">保存配置</button>
                <button class="btn btn-primary" onclick="reloadConfig()">重载配置</button>
            </div>
        </div>

        <div id="stats-tab" class="tab-content">
            <div class="stats-grid" id="stats-grid"></div>
            <button class="btn btn-primary" onclick="loadStats()">刷新统计</button>
        </div>
    </div>

    <script>
        let currentConfig = null;

        function switchTab(tab) {
            document.querySelectorAll('.tab').forEach(t => t.classList.remove('active'));
            document.querySelectorAll('.tab-content').forEach(c => c.classList.remove('active'));
            
            event.target.classList.add('active');
            document.getElementById(tab + '-tab').classList.add('active');
        }

        function showAlert(message, type) {
            const container = document.getElementById('alert-container');
            container.innerHTML = '<div class="alert alert-' + type + '">' + message + '</div>';
            setTimeout(() => {
                container.innerHTML = '';
            }, 5000);
        }

        async function loadConfig() {
            try {
                const response = await fetch('/api/config');
                if (!response.ok) throw new Error('加载配置失败');
                const config = await response.json();
                currentConfig = config;
                populateForm(config);
                showAlert('配置加载成功', 'success');
            } catch (error) {
                showAlert('加载配置失败: ' + error.message, 'error');
            }
        }

        function populateForm(config) {
            if (config.source) {
                document.getElementById('source-type').value = config.source.type || 'kafka';
            }
            if (config.target) {
                document.getElementById('target-brokers').value = (config.target.brokers || []).join(',');
                document.getElementById('target-topic').value = config.target.topic || '';
            }
            if (config.mirror) {
                document.getElementById('worker-count').value = config.mirror.worker_count || 4;
                document.getElementById('batch-size').value = config.mirror.batch_size || 100;
                document.getElementById('consumer-rate-limit').value = config.mirror.consumer_rate_limit || 0;
                document.getElementById('producer-rate-limit').value = config.mirror.producer_rate_limit || 0;
                document.getElementById('batch-enabled').checked = config.mirror.batch_enabled || false;
            }
            document.getElementById('config-json').value = JSON.stringify(config, null, 2);
        }

        async function saveConfig() {
            try {
                const jsonText = document.getElementById('config-json').value;
                let config;
                try {
                    config = JSON.parse(jsonText);
                } catch (e) {
                    // 如果JSON无效，从表单构建配置
                    config = buildConfigFromForm();
                }

                const response = await fetch('/api/config', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(config)
                });

                if (!response.ok) {
                    const error = await response.json();
                    throw new Error(error.error || '保存配置失败');
                }

                showAlert('配置保存成功，已热重载', 'success');
                currentConfig = config;
            } catch (error) {
                showAlert('保存配置失败: ' + error.message, 'error');
            }
        }

        function buildConfigFromForm() {
            const config = currentConfig || {};
            config.source = config.source || {};
            config.target = config.target || {};
            config.mirror = config.mirror || {};

            config.source.type = document.getElementById('source-type').value;
            config.target.brokers = document.getElementById('target-brokers').value.split(',').map(s => s.trim()).filter(s => s);
            config.target.topic = document.getElementById('target-topic').value;
            config.mirror.worker_count = parseInt(document.getElementById('worker-count').value) || 4;
            config.mirror.batch_size = parseInt(document.getElementById('batch-size').value) || 100;
            config.mirror.consumer_rate_limit = parseFloat(document.getElementById('consumer-rate-limit').value) || 0;
            config.mirror.producer_rate_limit = parseFloat(document.getElementById('producer-rate-limit').value) || 0;
            config.mirror.batch_enabled = document.getElementById('batch-enabled').checked;

            return config;
        }

        async function reloadConfig() {
            try {
                const response = await fetch('/api/config/reload', { method: 'POST' });
                if (!response.ok) throw new Error('重载配置失败');
                showAlert('配置已从文件重载', 'success');
                await loadConfig();
            } catch (error) {
                showAlert('重载配置失败: ' + error.message, 'error');
            }
        }

        async function loadStats() {
            try {
                const response = await fetch('/api/stats');
                if (!response.ok) throw new Error('加载统计失败');
                const stats = await response.json();
                displayStats(stats);
            } catch (error) {
                showAlert('加载统计失败: ' + error.message, 'error');
            }
        }

        function displayStats(stats) {
            const grid = document.getElementById('stats-grid');
            const consumed = stats.messages_consumed || 0;
            const produced = stats.messages_produced || 0;
            const bytesConsumed = (stats.bytes_consumed / 1024 / 1024).toFixed(2);
            const bytesProduced = (stats.bytes_produced / 1024 / 1024).toFixed(2);
            const errors = stats.errors || 0;
            const uptimeHours = Math.floor(stats.uptime_seconds / 3600);
            const uptimeMinutes = Math.floor((stats.uptime_seconds % 3600) / 60);
            grid.innerHTML = 
                '<div class="stat-card">' +
                    '<div class="stat-value">' + consumed + '</div>' +
                    '<div class="stat-label">消费消息数</div>' +
                '</div>' +
                '<div class="stat-card">' +
                    '<div class="stat-value">' + produced + '</div>' +
                    '<div class="stat-label">生产消息数</div>' +
                '</div>' +
                '<div class="stat-card">' +
                    '<div class="stat-value">' + bytesConsumed + ' MB</div>' +
                    '<div class="stat-label">消费字节数</div>' +
                '</div>' +
                '<div class="stat-card">' +
                    '<div class="stat-value">' + bytesProduced + ' MB</div>' +
                    '<div class="stat-label">生产字节数</div>' +
                '</div>' +
                '<div class="stat-card">' +
                    '<div class="stat-value">' + errors + '</div>' +
                    '<div class="stat-label">错误数</div>' +
                '</div>' +
                '<div class="stat-card">' +
                    '<div class="stat-value">' + uptimeHours + 'h ' + uptimeMinutes + 'm</div>' +
                    '<div class="stat-label">运行时间</div>' +
                '</div>';
        }

        // 页面加载时自动加载配置和统计
        window.onload = function() {
            loadConfig();
            loadStats();
            setInterval(loadStats, 5000); // 每5秒刷新统计
        };
    </script>
</body>
</html>`
}

