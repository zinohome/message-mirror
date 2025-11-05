package main

import (
	"context"
	"testing"
	"time"
)

func TestDeduplicator_IsDuplicate(t *testing.T) {
	config := &DedupConfig{
		Enabled:        true,
		Strategy:       "key_value",
		TTL:            1 * time.Hour,
		MaxEntries:     1000,
		CleanupInterval: 1 * time.Hour,
	}
	
	ctx := context.Background()
	deduplicator := NewDeduplicator(config, ctx)
	defer deduplicator.Stop()
	
	// 创建第一条消息
	msg1 := &Message{
		Key:   []byte("test-key"),
		Value: []byte("test-value"),
	}
	
	// 第一次应该不是重复
	isDup, err := deduplicator.IsDuplicate(msg1)
	if err != nil {
		t.Fatalf("去重检查失败: %v", err)
	}
	if isDup {
		t.Error("第一条消息不应该是重复的")
	}
	
	// 第二条相同内容的消息应该是重复
	isDup, err = deduplicator.IsDuplicate(msg1)
	if err != nil {
		t.Fatalf("去重检查失败: %v", err)
	}
	if !isDup {
		t.Error("相同内容的消息应该是重复的")
	}
	
	// 第三条不同内容的消息应该不是重复
	msg2 := &Message{
		Key:   []byte("test-key"),
		Value: []byte("different-value"),
	}
	
	isDup, err = deduplicator.IsDuplicate(msg2)
	if err != nil {
		t.Fatalf("去重检查失败: %v", err)
	}
	if isDup {
		t.Error("不同内容的消息不应该是重复的")
	}
}

func TestDeduplicator_Strategy_Key(t *testing.T) {
	config := &DedupConfig{
		Enabled:        true,
		Strategy:       "key",
		TTL:            1 * time.Hour,
		MaxEntries:     1000,
		CleanupInterval: 1 * time.Hour,
	}
	
	ctx := context.Background()
	deduplicator := NewDeduplicator(config, ctx)
	defer deduplicator.Stop()
	
	// 相同key，不同value应该被认为是重复（因为只比较key）
	msg1 := &Message{
		Key:   []byte("same-key"),
		Value: []byte("value1"),
	}
	
	msg2 := &Message{
		Key:   []byte("same-key"),
		Value: []byte("value2"),
	}
	
	isDup, err := deduplicator.IsDuplicate(msg1)
	if err != nil {
		t.Fatalf("去重检查失败: %v", err)
	}
	if isDup {
		t.Error("第一条消息不应该是重复的")
	}
	
	isDup, err = deduplicator.IsDuplicate(msg2)
	if err != nil {
		t.Fatalf("去重检查失败: %v", err)
	}
	if !isDup {
		t.Error("相同key的消息应该被认为是重复的（key策略）")
	}
}

func TestDeduplicator_Strategy_Value(t *testing.T) {
	config := &DedupConfig{
		Enabled:        true,
		Strategy:       "value",
		TTL:            1 * time.Hour,
		MaxEntries:     1000,
		CleanupInterval: 1 * time.Hour,
	}
	
	ctx := context.Background()
	deduplicator := NewDeduplicator(config, ctx)
	defer deduplicator.Stop()
	
	// 相同value，不同key应该被认为是重复（因为只比较value）
	msg1 := &Message{
		Key:   []byte("key1"),
		Value: []byte("same-value"),
	}
	
	msg2 := &Message{
		Key:   []byte("key2"),
		Value: []byte("same-value"),
	}
	
	isDup, err := deduplicator.IsDuplicate(msg1)
	if err != nil {
		t.Fatalf("去重检查失败: %v", err)
	}
	if isDup {
		t.Error("第一条消息不应该是重复的")
	}
	
	isDup, err = deduplicator.IsDuplicate(msg2)
	if err != nil {
		t.Fatalf("去重检查失败: %v", err)
	}
	if !isDup {
		t.Error("相同value的消息应该被认为是重复的（value策略）")
	}
}

func TestDeduplicator_Disabled(t *testing.T) {
	config := &DedupConfig{
		Enabled: false,
	}
	
	ctx := context.Background()
	deduplicator := NewDeduplicator(config, ctx)
	defer deduplicator.Stop()
	
	msg := &Message{
		Key:   []byte("test-key"),
		Value: []byte("test-value"),
	}
	
	// 禁用时应该永远不重复
	isDup, err := deduplicator.IsDuplicate(msg)
	if err != nil {
		t.Fatalf("去重检查失败: %v", err)
	}
	if isDup {
		t.Error("禁用去重时不应该返回重复")
	}
	
	// 再次检查应该也不重复
	isDup, err = deduplicator.IsDuplicate(msg)
	if err != nil {
		t.Fatalf("去重检查失败: %v", err)
	}
	if isDup {
		t.Error("禁用去重时不应该返回重复")
	}
}

func TestDeduplicator_GetStats(t *testing.T) {
	config := &DedupConfig{
		Enabled:        true,
		Strategy:       "key_value",
		TTL:            1 * time.Hour,
		MaxEntries:     1000,
		CleanupInterval: 1 * time.Hour,
	}
	
	ctx := context.Background()
	deduplicator := NewDeduplicator(config, ctx)
	defer deduplicator.Stop()
	
	msg := &Message{
		Key:   []byte("test-key"),
		Value: []byte("test-value"),
	}
	
	// 第一次检查
	deduplicator.IsDuplicate(msg)
	
	// 第二次检查（重复）
	deduplicator.IsDuplicate(msg)
	
	totalDedup, entries := deduplicator.GetStats()
	
	if totalDedup != 1 {
		t.Errorf("期望去重1条消息，实际%d", totalDedup)
	}
	
	if entries != 1 {
		t.Errorf("期望1条记录，实际%d", entries)
	}
}

func TestDeduplicator_DefaultConfig(t *testing.T) {
	deduplicator := NewDeduplicator(nil, context.Background())
	defer deduplicator.Stop()
	
	// 默认配置应该是禁用的
	msg := &Message{
		Key:   []byte("test-key"),
		Value: []byte("test-value"),
	}
	
	isDup, err := deduplicator.IsDuplicate(msg)
	if err != nil {
		t.Fatalf("去重检查失败: %v", err)
	}
	if isDup {
		t.Error("默认配置应该禁用去重")
	}
}

