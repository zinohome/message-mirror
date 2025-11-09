package core

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"testing"

	"gopkg.in/yaml.v3"
)

// TestNewConfigManager 测试创建配置管理器
func TestNewConfigManager(t *testing.T) {
	// 创建临时配置文件
	tmpFile, err := createTempConfigFile(t)
	if err != nil {
		t.Fatalf("创建临时配置文件失败: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// 测试正常创建
	cm, err := NewConfigManager(tmpFile.Name())
	if err != nil {
		t.Fatalf("创建ConfigManager失败: %v", err)
	}
	if cm == nil {
		t.Fatal("ConfigManager不应该为nil")
	}
	if cm.configPath != tmpFile.Name() {
		t.Errorf("期望configPath='%s'，实际'%s'", tmpFile.Name(), cm.configPath)
	}
	if cm.config == nil {
		t.Fatal("config不应该为nil")
	}

	// 测试不存在的文件
	_, err = NewConfigManager("/nonexistent/config.yaml")
	if err == nil {
		t.Error("应该返回错误，因为文件不存在")
	}
}

// TestConfigManager_GetConfig 测试获取配置（并发安全）
func TestConfigManager_GetConfig(t *testing.T) {
	tmpFile, err := createTempConfigFile(t)
	if err != nil {
		t.Fatalf("创建临时配置文件失败: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	cm, err := NewConfigManager(tmpFile.Name())
	if err != nil {
		t.Fatalf("创建ConfigManager失败: %v", err)
	}

	// 测试获取配置
	config := cm.GetConfig()
	if config == nil {
		t.Fatal("配置不应该为nil")
	}
	if config.Source.Type != "kafka" {
		t.Errorf("期望source.type='kafka'，实际'%s'", config.Source.Type)
	}

	// 测试并发访问
	var wg sync.WaitGroup
	concurrency := 10
	wg.Add(concurrency)
	for i := 0; i < concurrency; i++ {
		go func() {
			defer wg.Done()
			cfg := cm.GetConfig()
			if cfg == nil {
				t.Error("并发访问时配置不应该为nil")
			}
		}()
	}
	wg.Wait()
}

// TestConfigManager_UpdateConfig 测试更新配置
func TestConfigManager_UpdateConfig(t *testing.T) {
	tmpFile, err := createTempConfigFile(t)
	if err != nil {
		t.Fatalf("创建临时配置文件失败: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	cm, err := NewConfigManager(tmpFile.Name())
	if err != nil {
		t.Fatalf("创建ConfigManager失败: %v", err)
	}

	// 测试正常更新
	newConfig := cm.GetConfig()
	newConfig.Mirror.WorkerCount = 8
	err = cm.UpdateConfig(newConfig)
	if err != nil {
		t.Fatalf("更新配置失败: %v", err)
	}
	updatedConfig := cm.GetConfig()
	if updatedConfig.Mirror.WorkerCount != 8 {
		t.Errorf("期望worker_count=8，实际%d", updatedConfig.Mirror.WorkerCount)
	}

	// 测试配置验证失败
	invalidConfig := &Config{
		Source: SourceConfig{
			Type: "", // 无效：缺少类型
		},
		Target: TargetConfig{
			Brokers: []string{"localhost:9092"},
		},
		Mirror: MirrorConfig{
			WorkerCount: 4,
		},
	}
	err = cm.UpdateConfig(invalidConfig)
	if err == nil {
		t.Error("应该返回错误，因为配置验证失败")
	}

	// 测试监听器失败回滚
	// 先清除之前的监听器，注册一个会失败的监听器
	cm2, err := NewConfigManager(tmpFile.Name())
	if err != nil {
		t.Fatalf("创建新的ConfigManager失败: %v", err)
	}
	
	mockListener := &mockConfigReloadListener{
		shouldFail: true,
	}
	cm2.RegisterListener(mockListener)
	
	originalWorkerCount := cm2.GetConfig().Mirror.WorkerCount
	
	// 创建一个新的配置对象，而不是修改现有配置
	failingConfig := &Config{
		Source: SourceConfig{
			Type: "kafka",
		},
		Target: TargetConfig{
			Brokers: []string{"localhost:9092"},
		},
		Mirror: MirrorConfig{
			WorkerCount: 16,
		},
	}
	
	err = cm2.UpdateConfig(failingConfig)
	if err == nil {
		t.Error("应该返回错误，因为监听器失败")
	}
	
	// 验证配置已回滚
	rolledBackConfig := cm2.GetConfig()
	if rolledBackConfig.Mirror.WorkerCount != originalWorkerCount {
		t.Errorf("配置应该回滚到原始值%d，实际%d", originalWorkerCount, rolledBackConfig.Mirror.WorkerCount)
	}
}

// TestConfigManager_UpdateConfigFromJSON 测试从JSON更新
func TestConfigManager_UpdateConfigFromJSON(t *testing.T) {
	tmpFile, err := createTempConfigFile(t)
	if err != nil {
		t.Fatalf("创建临时配置文件失败: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	cm, err := NewConfigManager(tmpFile.Name())
	if err != nil {
		t.Fatalf("创建ConfigManager失败: %v", err)
	}

	// 测试有效JSON - 需要完整的配置结构
	// 注意：Config结构体只有mapstructure标签，没有json标签
	// json.Unmarshal会尝试匹配字段名（忽略大小写），但最好使用PascalCase
	// 或者先获取当前配置，修改后序列化为JSON再反序列化
	currentConfig := cm.GetConfig()
	currentConfig.Mirror.WorkerCount = 6
	validJSON, err := json.Marshal(currentConfig)
	if err != nil {
		t.Fatalf("序列化配置失败: %v", err)
	}
	
	err = cm.UpdateConfigFromJSON(validJSON)
	if err != nil {
		t.Fatalf("从JSON更新配置失败: %v", err)
	}
	config := cm.GetConfig()
	if config.Mirror.WorkerCount != 6 {
		t.Errorf("期望worker_count=6，实际%d", config.Mirror.WorkerCount)
	}

	// 测试无效JSON
	invalidJSON := `{invalid json}`
	err = cm.UpdateConfigFromJSON([]byte(invalidJSON))
	if err == nil {
		t.Error("应该返回错误，因为JSON无效")
	}
}

// TestConfigManager_SaveConfigToFile 测试保存到文件
func TestConfigManager_SaveConfigToFile(t *testing.T) {
	tmpFile, err := createTempConfigFile(t)
	if err != nil {
		t.Fatalf("创建临时配置文件失败: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	cm, err := NewConfigManager(tmpFile.Name())
	if err != nil {
		t.Fatalf("创建ConfigManager失败: %v", err)
	}

	// 修改配置（不通过UpdateConfig，直接保存以测试SaveConfigToFile）
	// 因为UpdateConfig也会调用SaveConfigToFile，所以我们需要直接测试SaveConfigToFile
	newConfig := cm.GetConfig()
	newConfig.Mirror.WorkerCount = 10
	
	// 直接保存到文件（不通过UpdateConfig）
	err = cm.SaveConfigToFile(newConfig)
	if err != nil {
		t.Fatalf("保存配置到文件失败: %v", err)
	}

	// 验证文件已保存（通过直接读取YAML文件并解析）
	// 注意：LoadConfig使用viper，可能会应用默认值，所以我们需要直接读取YAML
	yamlData, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("读取保存的YAML文件失败: %v", err)
	}
	
	// 使用yaml.Unmarshal直接解析
	var savedConfig Config
	err = yaml.Unmarshal(yamlData, &savedConfig)
	if err != nil {
		t.Fatalf("解析保存的YAML文件失败: %v", err)
	}
	
	if savedConfig.Mirror.WorkerCount != 10 {
		t.Errorf("期望worker_count=10，实际%d", savedConfig.Mirror.WorkerCount)
	}
	
	// 也测试通过LoadConfig重新加载（虽然可能应用默认值）
	reloadedCM, err := NewConfigManager(tmpFile.Name())
	if err != nil {
		t.Fatalf("重新加载配置失败: %v", err)
	}
	reloadedConfig := reloadedCM.GetConfig()
	// LoadConfig可能会应用默认值，所以这里只验证配置能正常加载
	if reloadedConfig == nil {
		t.Error("重新加载的配置不应该为nil")
	}
}

// TestConfigManager_ReloadFromFile 测试从文件重载
func TestConfigManager_ReloadFromFile(t *testing.T) {
	tmpFile, err := createTempConfigFile(t)
	if err != nil {
		t.Fatalf("创建临时配置文件失败: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	cm, err := NewConfigManager(tmpFile.Name())
	if err != nil {
		t.Fatalf("创建ConfigManager失败: %v", err)
	}

	// 修改文件内容
	newConfigContent := `
source:
  type: "kafka"
  brokers:
    - "localhost:9092"
  topic: "test-topic"

target:
  brokers:
    - "localhost:9093"
  topic: "target-topic"

mirror:
  worker_count: 12
`
	err = os.WriteFile(tmpFile.Name(), []byte(newConfigContent), 0644)
	if err != nil {
		t.Fatalf("写入配置文件失败: %v", err)
	}

	// 重载配置
	err = cm.ReloadFromFile()
	if err != nil {
		t.Fatalf("重载配置失败: %v", err)
	}

	// 验证配置已更新
	config := cm.GetConfig()
	if config.Mirror.WorkerCount != 12 {
		t.Errorf("期望worker_count=12，实际%d", config.Mirror.WorkerCount)
	}
}

// TestConfigManager_RegisterListener 测试注册监听器
func TestConfigManager_RegisterListener(t *testing.T) {
	tmpFile, err := createTempConfigFile(t)
	if err != nil {
		t.Fatalf("创建临时配置文件失败: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	cm, err := NewConfigManager(tmpFile.Name())
	if err != nil {
		t.Fatalf("创建ConfigManager失败: %v", err)
	}

	// 注册监听器
	mockListener := &mockConfigReloadListener{}
	cm.RegisterListener(mockListener)

	// 更新配置，应该触发监听器
	newConfig := cm.GetConfig()
	newConfig.Mirror.WorkerCount = 14
	err = cm.UpdateConfig(newConfig)
	if err != nil {
		t.Fatalf("更新配置失败: %v", err)
	}

	// 验证监听器被调用
	if !mockListener.called {
		t.Error("监听器应该被调用")
	}
	if mockListener.oldConfig == nil || mockListener.newConfig == nil {
		t.Error("监听器应该接收到配置")
	}
}

// TestConfigManager_GetConfigJSON 测试获取JSON配置
func TestConfigManager_GetConfigJSON(t *testing.T) {
	tmpFile, err := createTempConfigFile(t)
	if err != nil {
		t.Fatalf("创建临时配置文件失败: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	cm, err := NewConfigManager(tmpFile.Name())
	if err != nil {
		t.Fatalf("创建ConfigManager失败: %v", err)
	}

	// 获取JSON配置
	jsonData, err := cm.GetConfigJSON()
	if err != nil {
		t.Fatalf("获取JSON配置失败: %v", err)
	}
	if len(jsonData) == 0 {
		t.Error("JSON数据不应该为空")
	}

	// 验证JSON格式
	var config Config
	err = json.Unmarshal(jsonData, &config)
	if err != nil {
		t.Fatalf("解析JSON配置失败: %v", err)
	}
	if config.Source.Type != "kafka" {
		t.Errorf("期望source.type='kafka'，实际'%s'", config.Source.Type)
	}
}

// TestConfigManager_GetConfigYAML 测试获取YAML配置
func TestConfigManager_GetConfigYAML(t *testing.T) {
	tmpFile, err := createTempConfigFile(t)
	if err != nil {
		t.Fatalf("创建临时配置文件失败: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	cm, err := NewConfigManager(tmpFile.Name())
	if err != nil {
		t.Fatalf("创建ConfigManager失败: %v", err)
	}

	// 获取YAML配置
	yamlData, err := cm.GetConfigYAML()
	if err != nil {
		t.Fatalf("获取YAML配置失败: %v", err)
	}
	if len(yamlData) == 0 {
		t.Error("YAML数据不应该为空")
	}

	// 验证YAML格式（通过重新加载）
	tmpFile2, err := os.CreateTemp("", "test-config-*.yaml")
	if err != nil {
		t.Fatalf("创建临时文件失败: %v", err)
	}
	defer os.Remove(tmpFile2.Name())
	
	err = os.WriteFile(tmpFile2.Name(), yamlData, 0644)
	if err != nil {
		t.Fatalf("写入YAML文件失败: %v", err)
	}

	reloadedConfig, err := LoadConfig(tmpFile2.Name())
	if err != nil {
		t.Fatalf("重新加载YAML配置失败: %v", err)
	}
	if reloadedConfig.Source.Type != "kafka" {
		t.Errorf("期望source.type='kafka'，实际'%s'", reloadedConfig.Source.Type)
	}
}

// TestConfigManager_ConcurrentAccess 测试并发访问安全性
func TestConfigManager_ConcurrentAccess(t *testing.T) {
	tmpFile, err := createTempConfigFile(t)
	if err != nil {
		t.Fatalf("创建临时配置文件失败: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	cm, err := NewConfigManager(tmpFile.Name())
	if err != nil {
		t.Fatalf("创建ConfigManager失败: %v", err)
	}

	var wg sync.WaitGroup
	concurrency := 20
	wg.Add(concurrency * 3) // 3种操作：GetConfig, UpdateConfig, GetConfigJSON

	// 并发读取
	for i := 0; i < concurrency; i++ {
		go func() {
			defer wg.Done()
			config := cm.GetConfig()
			if config == nil {
				t.Error("并发读取时配置不应该为nil")
			}
		}()
	}

	// 并发更新
	for i := 0; i < concurrency; i++ {
		go func(id int) {
			defer wg.Done()
			newConfig := cm.GetConfig()
			newConfig.Mirror.WorkerCount = 4 + id
			_ = cm.UpdateConfig(newConfig) // 忽略错误，因为可能有竞争
		}(i)
	}

	// 并发获取JSON
	for i := 0; i < concurrency; i++ {
		go func() {
			defer wg.Done()
			_, err := cm.GetConfigJSON()
			if err != nil {
				t.Errorf("并发获取JSON配置失败: %v", err)
			}
		}()
	}

	wg.Wait()
}

// TestConfigManager_WatchFile 测试监控文件（当前未实现）
func TestConfigManager_WatchFile(t *testing.T) {
	tmpFile, err := createTempConfigFile(t)
	if err != nil {
		t.Fatalf("创建临时配置文件失败: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	cm, err := NewConfigManager(tmpFile.Name())
	if err != nil {
		t.Fatalf("创建ConfigManager失败: %v", err)
	}

	// 当前实现返回nil
	err = cm.WatchFile()
	if err != nil {
		t.Errorf("WatchFile应该返回nil（未实现），实际返回: %v", err)
	}
}

// 辅助函数：创建临时配置文件
func createTempConfigFile(t *testing.T) (*os.File, error) {
	configContent := `
source:
  type: "kafka"
  brokers:
    - "localhost:9092"
  topic: "test-topic"

target:
  brokers:
    - "localhost:9093"
  topic: "target-topic"

mirror:
  worker_count: 4
`
	tmpFile, err := os.CreateTemp("", "test-config-*.yaml")
	if err != nil {
		return nil, err
	}
	
	if _, err := tmpFile.WriteString(configContent); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return nil, err
	}
	tmpFile.Close()
	
	return tmpFile, nil
}

// mockConfigReloadListener 模拟配置重载监听器
type mockConfigReloadListener struct {
	called     bool
	oldConfig  *Config
	newConfig  *Config
	shouldFail bool
}

func (m *mockConfigReloadListener) OnConfigReload(oldConfig, newConfig *Config) error {
	m.called = true
	m.oldConfig = oldConfig
	m.newConfig = newConfig
	if m.shouldFail {
		return fmt.Errorf("模拟监听器失败")
	}
	return nil
}

