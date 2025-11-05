package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
)

// TLSConfig TLS配置
type TLSConfig struct {
	Enabled            bool   `mapstructure:"enabled"`
	CertFile           string `mapstructure:"cert_file"`
	KeyFile            string `mapstructure:"key_file"`
	CAFile             string `mapstructure:"ca_file"`
	InsecureSkipVerify bool   `mapstructure:"insecure_skip_verify"`
	MinVersion         string `mapstructure:"min_version"`
	MaxVersion         string `mapstructure:"max_version"`
}

// NewTLSConfig 创建TLS配置
func NewTLSConfig(config *TLSConfig) (*tls.Config, error) {
	if !config.Enabled {
		return nil, nil
	}
	
	tlsConfig := &tls.Config{
		InsecureSkipVerify: config.InsecureSkipVerify,
	}
	
	// 加载客户端证书（如果提供）
	if config.CertFile != "" && config.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(config.CertFile, config.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("加载TLS证书失败: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}
	
	// 加载CA证书（如果提供）
	if config.CAFile != "" {
		caCert, err := os.ReadFile(config.CAFile)
		if err != nil {
			return nil, fmt.Errorf("读取CA证书失败: %w", err)
		}
		
		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("解析CA证书失败")
		}
		tlsConfig.RootCAs = caCertPool
	}
	
	// 设置TLS版本
	if config.MinVersion != "" {
		version, err := parseTLSVersion(config.MinVersion)
		if err != nil {
			return nil, fmt.Errorf("无效的TLS最小版本: %w", err)
		}
		tlsConfig.MinVersion = version
	}
	
	if config.MaxVersion != "" {
		version, err := parseTLSVersion(config.MaxVersion)
		if err != nil {
			return nil, fmt.Errorf("无效的TLS最大版本: %w", err)
		}
		tlsConfig.MaxVersion = version
	}
	
	return tlsConfig, nil
}

// parseTLSVersion 解析TLS版本字符串
func parseTLSVersion(version string) (uint16, error) {
	switch version {
	case "1.0":
		return tls.VersionTLS10, nil
	case "1.1":
		return tls.VersionTLS11, nil
	case "1.2":
		return tls.VersionTLS12, nil
	case "1.3":
		return tls.VersionTLS13, nil
	default:
		return 0, fmt.Errorf("不支持的TLS版本: %s", version)
	}
}

// ConfigEncryption 配置加密（用于敏感信息）
type ConfigEncryption struct {
	enabled bool
	key     []byte
}

// NewConfigEncryption 创建配置加密器
func NewConfigEncryption(enabled bool, key []byte) *ConfigEncryption {
	return &ConfigEncryption{
		enabled: enabled,
		key:     key,
	}
}

// Encrypt 加密敏感配置值（简单实现，生产环境应使用更安全的方案）
func (ce *ConfigEncryption) Encrypt(value string) (string, error) {
	if !ce.enabled {
		return value, nil
	}
	
	// 这里实现简单的XOR加密（仅示例，生产环境应使用AES等）
	encrypted := make([]byte, len(value))
	for i := 0; i < len(value); i++ {
		encrypted[i] = value[i] ^ ce.key[i%len(ce.key)]
	}
	
	// Base64编码（实际应该使用更安全的编码）
	return fmt.Sprintf("encrypted:%x", encrypted), nil
}

// Decrypt 解密配置值
func (ce *ConfigEncryption) Decrypt(encrypted string) (string, error) {
	if !ce.enabled {
		return encrypted, nil
	}
	
	// 这里应该实现对应的解密逻辑
	// 仅示例，生产环境需要完整实现
	return encrypted, nil
}

// SecurityAudit 安全审计日志
type SecurityAudit struct {
	enabled bool
	logger  *Logger
}

// NewSecurityAudit 创建安全审计
func NewSecurityAudit(enabled bool, logger *Logger) *SecurityAudit {
	return &SecurityAudit{
		enabled: enabled,
		logger:  logger,
	}
}

// LogAccess 记录访问日志
func (sa *SecurityAudit) LogAccess(action string, resource string, success bool) {
	if !sa.enabled {
		return
	}
	
	status := "success"
	if !success {
		status = "failed"
	}
	
	sa.logger.Printf("[安全审计] action=%s resource=%s status=%s", action, resource, status)
}

// LogConfigChange 记录配置变更
func (sa *SecurityAudit) LogConfigChange(key string, oldValue string, newValue string) {
	if !sa.enabled {
		return
	}
	
	sa.logger.Printf("[安全审计] config_change key=%s old_value=%s new_value=%s", key, oldValue, newValue)
}

