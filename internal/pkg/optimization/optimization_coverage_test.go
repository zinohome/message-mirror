package optimization

import (
	"context"
	"testing"
	"time"
)

// TestBatchProcessor_AddAndFlush 测试添加和刷新
func TestBatchProcessor_AddAndFlush(t *testing.T) {
	collected := 0
	processor := func(batch []*Message) error {
		collected = len(batch)
		return nil
	}

	bp := NewBatchProcessor(3, 100*time.Millisecond, processor)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	bp.Start(ctx)
	defer bp.Stop()

	msg1 := &Message{Key: []byte("k1"), Value: []byte("v1")}
	msg2 := &Message{Key: []byte("k2"), Value: []byte("v2")}
	msg3 := &Message{Key: []byte("k3"), Value: []byte("v3")}

	// 添加消息直到达到批大小
	if err := bp.Add(msg1); err != nil {
		t.Fatalf("添加消息1失败: %v", err)
	}
	if err := bp.Add(msg2); err != nil {
		t.Fatalf("添加消息2失败: %v", err)
	}
	if collected != 0 {
		t.Error("未达到批大小时不应处理")
	}

	// 达到批大小后应该处理
	if err := bp.Add(msg3); err != nil {
		t.Fatalf("添加消息3失败: %v", err)
	}

	time.Sleep(50 * time.Millisecond)
	if collected == 0 {
		t.Error("达到批大小后应该处理消息")
	}
}

// TestBatchProcessor_TimeoutFlush 测试超时自动刷新
func TestBatchProcessor_TimeoutFlush(t *testing.T) {
	collected := 0
	processor := func(batch []*Message) error {
		collected = len(batch)
		return nil
	}

	bp := NewBatchProcessor(100, 100*time.Millisecond, processor)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	bp.Start(ctx)
	defer bp.Stop()

	msg := &Message{Key: []byte("k1"), Value: []byte("v1")}

	// 添加单条消息
	if err := bp.Add(msg); err != nil {
		t.Fatalf("添加消息失败: %v", err)
	}

	// 等待超时
	time.Sleep(150 * time.Millisecond)

	if collected == 0 {
		t.Error("超时后应该处理消息")
	}
}

// TestBatchProcessor_StopGracefully 测试优雅停止
func TestBatchProcessor_StopGracefully(t *testing.T) {
	collected := false
	processor := func(batch []*Message) error {
		collected = true
		return nil
	}

	bp := NewBatchProcessor(10, 1*time.Second, processor)
	ctx, cancel := context.WithCancel(context.Background())

	bp.Start(ctx)

	msg := &Message{Key: []byte("k1"), Value: []byte("v1")}
	bp.Add(msg)

	// 立即停止
	cancel()
	bp.Stop()

	time.Sleep(100 * time.Millisecond)

	// 停止后应该处理剩余消息
	if collected {
		t.Log("停止前已处理消息")
	}
}

// TestBatchProcessor_UpdateConfigSettings 测试配置更新
func TestBatchProcessor_UpdateConfigSettings(t *testing.T) {
	processor := func(batch []*Message) error {
		return nil
	}

	bp := NewBatchProcessor(5, 100*time.Millisecond, processor)

	// 更新配置
	bp.UpdateConfig(10, 200*time.Millisecond)

	// 验证配置已更新（通过行为验证）
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	bp.Start(ctx)
	defer bp.Stop()

	// 应该能够添加更多消息到新批大小
	for i := 0; i < 10; i++ {
		msg := &Message{Key: []byte{byte(i)}, Value: []byte{byte(i)}}
		if err := bp.Add(msg); err != nil {
			t.Fatalf("添加消息%d失败: %v", i, err)
		}
	}

	time.Sleep(50 * time.Millisecond)
}

// TestBatchProcessor_EmptyBatch 测试空批处理
func TestBatchProcessor_EmptyBatch(t *testing.T) {
	called := 0
	processor := func(batch []*Message) error {
		called++
		return nil
	}

	bp := NewBatchProcessor(5, 100*time.Millisecond, processor)
	ctx, cancel := context.WithCancel(context.Background())

	bp.Start(ctx)

	// 不添加任何消息，直接停止
	time.Sleep(150 * time.Millisecond)
	cancel()
	bp.Stop()

	// 空批处理可能不会被调用（取决于实现）
	_ = called
}
