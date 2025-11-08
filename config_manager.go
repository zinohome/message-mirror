package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"sync"

	"gopkg.in/yaml.v3"
)

// ConfigManager 配置管理器，支持热重载
type ConfigManager struct {
	configPath string
	config     *Config
	mu         sync.RWMutex
	reloadChan chan *Config
	listeners  []ConfigReloadListener
}

// ConfigReloadListener 配置重载监听器
type ConfigReloadListener interface {
	OnConfigReload(oldConfig, newConfig *Config) error
}

// NewConfigManager 创建配置管理器
func NewConfigManager(configPath string) (*ConfigManager, error) {
	config, err := LoadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("加载配置失败: %w", err)
	}

	return &ConfigManager{
		configPath: configPath,
		config:     config,
		reloadChan: make(chan *Config, 1),
		listeners:  make([]ConfigReloadListener, 0),
	}, nil
}

// GetConfig 获取当前配置（只读）
func (cm *ConfigManager) GetConfig() *Config {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.config
}

// UpdateConfig 更新配置（支持热重载）
func (cm *ConfigManager) UpdateConfig(newConfig *Config) error {
	// 验证配置
	if err := validateConfig(newConfig); err != nil {
		return fmt.Errorf("配置验证失败: %w", err)
	}

	cm.mu.Lock()
	oldConfig := cm.config
	cm.config = newConfig
	cm.mu.Unlock()

	// 通知监听器
	for _, listener := range cm.listeners {
		if err := listener.OnConfigReload(oldConfig, newConfig); err != nil {
			// 如果某个监听器失败，回滚配置
			cm.mu.Lock()
			cm.config = oldConfig
			cm.mu.Unlock()
			return fmt.Errorf("配置重载失败: %w", err)
		}
	}

	// 保存到文件
	if err := cm.SaveConfigToFile(newConfig); err != nil {
		log.Printf("保存配置到文件失败: %v", err)
		// 不返回错误，因为配置已经在内存中更新
	}

	return nil
}

// UpdateConfigFromJSON 从JSON更新配置
func (cm *ConfigManager) UpdateConfigFromJSON(jsonData []byte) error {
	var newConfig Config
	if err := json.Unmarshal(jsonData, &newConfig); err != nil {
		return fmt.Errorf("解析JSON配置失败: %w", err)
	}

	return cm.UpdateConfig(&newConfig)
}

// SaveConfigToFile 保存配置到文件
func (cm *ConfigManager) SaveConfigToFile(config *Config) error {
	// 将配置转换为YAML
	yamlData, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}

	// 写入文件
	if err := ioutil.WriteFile(cm.configPath, yamlData, 0644); err != nil {
		return fmt.Errorf("写入配置文件失败: %w", err)
	}

	return nil
}

// ReloadFromFile 从文件重新加载配置
func (cm *ConfigManager) ReloadFromFile() error {
	newConfig, err := LoadConfig(cm.configPath)
	if err != nil {
		return fmt.Errorf("重新加载配置失败: %w", err)
	}

	return cm.UpdateConfig(newConfig)
}

// RegisterListener 注册配置重载监听器
func (cm *ConfigManager) RegisterListener(listener ConfigReloadListener) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.listeners = append(cm.listeners, listener)
}

// WatchFile 监控配置文件变化（可选功能）
func (cm *ConfigManager) WatchFile() error {
	// 这里可以使用fsnotify实现文件监控
	// 为了简化，暂时不实现
	return nil
}

// GetConfigJSON 获取配置的JSON表示
func (cm *ConfigManager) GetConfigJSON() ([]byte, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return json.Marshal(cm.config)
}

// GetConfigYAML 获取配置的YAML表示
func (cm *ConfigManager) GetConfigYAML() ([]byte, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return yaml.Marshal(cm.config)
}

