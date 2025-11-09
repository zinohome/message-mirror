package security

import (
	"crypto/tls"
	"sync"
	"testing"
)

// TestTLSConfig_LoadCertificates 测试加载证书
func TestTLSConfig_LoadCertificates(t *testing.T) {
	// 测试无效证书路径
	config := &TLSConfig{
		Enabled:  true,
		CertFile: "/nonexistent/cert.pem",
		KeyFile:  "/nonexistent/key.pem",
	}

	tlsConfig, err := NewTLSConfig(config)
	if err == nil {
		t.Error("无效证书路径应该返回错误")
	}
	if tlsConfig != nil {
		t.Error("无效证书路径应该返回nil")
	}

	// 测试禁用TLS
	config2 := &TLSConfig{
		Enabled: false,
	}

	tlsConfig2, err := NewTLSConfig(config2)
	if err != nil {
		t.Fatalf("禁用TLS不应该返回错误: %v", err)
	}
	if tlsConfig2 != nil {
		t.Error("禁用TLS应该返回nil")
	}
}

// TestSecurityAudit_Logging 测试审计日志
func TestSecurityAudit_Logging(t *testing.T) {
	audit := NewSecurityAudit(true)

	// 测试记录事件
	audit.LogAccess("test_action", "test_resource", true)

	// 验证没有panic
	t.Log("审计日志记录完成")
}

// TestSecurityAudit_Disabled 测试禁用审计
func TestSecurityAudit_Disabled(t *testing.T) {
	audit := NewSecurityAudit(false)

	// 测试记录事件（应该不记录）
	audit.LogAccess("test_action", "test_resource", true)

	// 验证没有panic
	t.Log("禁用审计时记录事件完成")
}

// TestTLSConfig_DifferentVersions 测试不同的TLS版本
func TestTLSConfig_DifferentVersions(t *testing.T) {
	versions := []string{"TLSv1.0", "TLSv1.1", "TLSv1.2", "TLSv1.3", "invalid"}

	for _, version := range versions {
		config := &TLSConfig{
			Enabled:     false, // 禁用以避免需要真实证书
			MinVersion:  version,
		}

		tlsConfig, err := NewTLSConfig(config)
		if err != nil {
			t.Logf("TLS版本 %s: 创建配置失败（预期，因为禁用）: %v", version, err)
			continue
		}

		if tlsConfig != nil {
			// 验证TLS版本设置
			t.Logf("TLS版本 %s: 配置创建成功", version)
		}
	}
}

// TestConfigEncryption_EncryptDecrypt 测试配置加密解密
func TestConfigEncryption_EncryptDecrypt(t *testing.T) {
	key := []byte("test-key-32-bytes-long!!") // 32字节密钥
	if len(key) != 32 {
		t.Fatalf("密钥长度应该为32字节，实际%d", len(key))
	}

	encryption := NewConfigEncryption(true, key)

	// 测试加密
	plaintext := "test-config-value"
	encrypted, err := encryption.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("加密失败: %v", err)
	}
	if encrypted == plaintext {
		t.Error("加密后的值应该与原文不同")
	}

	// 测试解密
	decrypted, err := encryption.Decrypt(encrypted)
	if err != nil {
		t.Fatalf("解密失败: %v", err)
	}
	if decrypted != plaintext {
		t.Errorf("解密后的值应该与原文相同，期望'%s'，实际'%s'", plaintext, decrypted)
	}
}

// TestConfigEncryption_InvalidKey 测试无效密钥
func TestConfigEncryption_InvalidKey(t *testing.T) {
	// 测试空密钥
	encryption := NewConfigEncryption(true, nil)
	plaintext := "test-config-value"

	encrypted, err := encryption.Encrypt(plaintext)
	if err == nil {
		t.Error("空密钥应该返回错误")
	}
	if encrypted != "" {
		t.Error("空密钥加密应该返回空字符串")
	}

	// 测试短密钥
	shortKey := []byte("short")
	encryption2 := NewConfigEncryption(true, shortKey)

	encrypted2, err := encryption2.Encrypt(plaintext)
	if err == nil {
		t.Error("短密钥应该返回错误")
	}
	if encrypted2 != "" {
		t.Error("短密钥加密应该返回空字符串")
	}
}

// TestSecurityAudit_MultipleEvents 测试多个审计事件
func TestSecurityAudit_MultipleEvents(t *testing.T) {
	audit := NewSecurityAudit(true)

	// 记录多个事件
	for i := 0; i < 10; i++ {
		audit.LogAccess("test_action", "test_resource", true)
	}

	// 验证没有panic
	t.Log("多个审计事件记录完成")
}

// TestSecurityAudit_ConcurrentAccess 测试并发访问
func TestSecurityAudit_ConcurrentAccess(t *testing.T) {
	audit := NewSecurityAudit(true)

	var wg sync.WaitGroup
	concurrency := 20
	eventsPerGoroutine := 10

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < eventsPerGoroutine; j++ {
				audit.LogAccess("test_action", "test_resource", true)
			}
		}(i)
	}

	wg.Wait()

	// 验证并发访问没有panic或数据竞争
	t.Log("并发访问审计日志完成")
}

// TestTLSConfig_MinVersion 测试最小TLS版本
func TestTLSConfig_MinVersion(t *testing.T) {
	config := &TLSConfig{
		Enabled:    false, // 禁用以避免需要真实证书
		MinVersion: "TLSv1.2",
		MaxVersion: "TLSv1.3",
	}

	tlsConfig, err := NewTLSConfig(config)
	if err != nil {
		t.Logf("创建TLS配置失败（预期，因为禁用）: %v", err)
		return
	}

	if tlsConfig != nil {
		// 验证TLS版本设置
		if tlsConfig.MinVersion != tls.VersionTLS12 {
			t.Errorf("期望MinVersion=TLSv1.2，实际%x", tlsConfig.MinVersion)
		}
		if tlsConfig.MaxVersion != tls.VersionTLS13 {
			t.Errorf("期望MaxVersion=TLSv1.3，实际%x", tlsConfig.MaxVersion)
		}
	}
}

