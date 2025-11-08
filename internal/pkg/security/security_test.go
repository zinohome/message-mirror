package security

import (
	"testing"
	"time"
)

func TestNewTLSConfig(t *testing.T) {
	// 测试禁用TLS
	config := &TLSConfig{
		Enabled: false,
	}
	
	tlsConfig, err := NewTLSConfig(config)
	if err != nil {
		t.Fatalf("创建TLS配置失败: %v", err)
	}
	
	if tlsConfig != nil {
		t.Error("禁用TLS时应该返回nil")
	}
	
	// 测试启用TLS（跳过验证）
	config = &TLSConfig{
		Enabled:            true,
		InsecureSkipVerify: true,
		MinVersion:         "1.2",
	}
	
	tlsConfig, err = NewTLSConfig(config)
	if err != nil {
		t.Fatalf("创建TLS配置失败: %v", err)
	}
	
	if tlsConfig == nil {
		t.Error("TLS配置不应该为nil")
	}
	
	if !tlsConfig.InsecureSkipVerify {
		t.Error("应该跳过证书验证")
	}
	
	if tlsConfig.MinVersion != 0x0303 { // TLS 1.2
		t.Error("TLS最小版本应该是1.2")
	}
}

func TestParseTLSVersion(t *testing.T) {
	tests := []struct {
		version string
		want    uint16
		wantErr bool
	}{
		{"1.0", 0x0301, false},
		{"1.1", 0x0302, false},
		{"1.2", 0x0303, false},
		{"1.3", 0x0304, false},
		{"2.0", 0, true},
	}
	
	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			version, err := parseTLSVersion(tt.version)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseTLSVersion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && version != tt.want {
				t.Errorf("parseTLSVersion() = %x, want %x", version, tt.want)
			}
		})
	}
}

func TestConfigEncryption(t *testing.T) {
	// 测试配置加密
	encryption := NewConfigEncryption(true, []byte("test-key-123456"))
	
	original := "sensitive-password"
	encrypted, err := encryption.Encrypt(original)
	if err != nil {
		t.Fatalf("加密失败: %v", err)
	}
	
	if encrypted == original {
		t.Error("加密后的值应该不同于原始值")
	}
	
	// 测试禁用加密
	encryptionDisabled := NewConfigEncryption(false, []byte("test-key"))
	encrypted2, err := encryptionDisabled.Encrypt(original)
	if err != nil {
		t.Fatalf("加密失败: %v", err)
	}
	
	if encrypted2 != original {
		t.Error("禁用加密时应该返回原始值")
	}
}

func TestSecurityAudit(t *testing.T) {
	// 测试安全审计
	audit := NewSecurityAudit(true)
	
	audit.LogAccess("read", "config.yaml", true)
	audit.LogAccess("write", "config.yaml", false)
	audit.LogConfigChange("password", "old", "new")
	
	// 等待异步写入
	time.Sleep(100 * time.Millisecond)
	
	// 测试禁用审计
	auditDisabled := NewSecurityAudit(false)
	auditDisabled.LogAccess("read", "config.yaml", true)
	// 禁用时不应该记录日志
}

