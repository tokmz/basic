package config

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRemoteConfig(t *testing.T) {
	provider := NewMockRemoteProvider(true)
	remoteConfig := NewRemoteConfig(provider, true)

	assert.NotNil(t, remoteConfig)
	assert.Equal(t, provider, remoteConfig.provider)
	assert.True(t, remoteConfig.debug)
	assert.NotNil(t, remoteConfig.ctx)
	assert.NotNil(t, remoteConfig.cancel)
}

func TestRemoteConfig_Connect(t *testing.T) {
	provider := NewMockRemoteProvider(true)
	remoteConfig := NewRemoteConfig(provider, true)
	defer remoteConfig.Close()

	// 测试连接
	err := remoteConfig.Connect()
	assert.NoError(t, err)
	assert.True(t, provider.IsConnected())
}

func TestRemoteConfig_GetSet(t *testing.T) {
	provider := NewMockRemoteProvider(true)
	remoteConfig := NewRemoteConfig(provider, true)
	defer remoteConfig.Close()

	// 连接
	err := remoteConfig.Connect()
	require.NoError(t, err)

	// 测试设置配置
	testKey := "test.key"
	testValue := []byte("test.value")

	err = remoteConfig.Set(testKey, testValue)
	assert.NoError(t, err)

	// 测试获取配置
	value, err := remoteConfig.Get(testKey)
	assert.NoError(t, err)
	assert.Equal(t, testValue, value)

	// 测试获取不存在的键
	_, err = remoteConfig.Get("nonexistent.key")
	assert.Error(t, err)
}

func TestRemoteConfig_Watch(t *testing.T) {
	provider := NewMockRemoteProvider(true)
	remoteConfig := NewRemoteConfig(provider, true)
	defer remoteConfig.Close()

	// 连接
	err := remoteConfig.Connect()
	require.NoError(t, err)

	// 设置初始值
	testKey := "watch.key"
	initialValue := []byte("initial.value")
	err = remoteConfig.Set(testKey, initialValue)
	require.NoError(t, err)

	// 设置监听
	changed := make(chan []byte, 2)
	err = remoteConfig.Watch(testKey, func(value []byte) {
		changed <- value
	})
	require.NoError(t, err)

	// 等待初始值回调
	select {
	case value := <-changed:
		assert.Equal(t, initialValue, value)
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for initial value")
	}

	// 更新值
	updatedValue := []byte("updated.value")
	err = remoteConfig.Set(testKey, updatedValue)
	require.NoError(t, err)

	// 等待更新值回调
	select {
	case value := <-changed:
		assert.Equal(t, updatedValue, value)
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for updated value")
	}
}

func TestRemoteConfig_AutoConnect(t *testing.T) {
	provider := NewMockRemoteProvider(true)
	remoteConfig := NewRemoteConfig(provider, true)
	defer remoteConfig.Close()

	// 在未连接状态下直接操作，应该自动连接
	testKey := "auto.connect.key"
	testValue := []byte("auto.connect.value")

	// 设置配置（应该自动连接）
	err := remoteConfig.Set(testKey, testValue)
	assert.NoError(t, err)
	assert.True(t, provider.IsConnected())

	// 获取配置（应该自动连接）
	provider.Close() // 断开连接
	value, err := remoteConfig.Get(testKey)
	assert.NoError(t, err)
	assert.Equal(t, testValue, value)
	assert.True(t, provider.IsConnected())
}

func TestRemoteConfig_Close(t *testing.T) {
	provider := NewMockRemoteProvider(true)
	remoteConfig := NewRemoteConfig(provider, true)

	// 连接
	err := remoteConfig.Connect()
	require.NoError(t, err)
	assert.True(t, provider.IsConnected())

	// 关闭
	err = remoteConfig.Close()
	assert.NoError(t, err)
	assert.False(t, provider.IsConnected())
}

func TestMockRemoteProvider(t *testing.T) {
	provider := NewMockRemoteProvider(true)

	// 初始状态
	assert.False(t, provider.IsConnected())

	// 测试连接
	ctx := context.Background()
	err := provider.Connect(ctx)
	assert.NoError(t, err)
	assert.True(t, provider.IsConnected())

	// 测试在未连接状态下的操作
	provider.Close()
	_, err = provider.Get(ctx, "test.key")
	assert.Error(t, err)

	err = provider.Set(ctx, "test.key", []byte("test.value"))
	assert.Error(t, err)

	err = provider.Watch(ctx, "test.key", func([]byte) {})
	assert.Error(t, err)

	// 重新连接
	err = provider.Connect(ctx)
	require.NoError(t, err)

	// 测试设置和获取
	testKey := "test.key"
	testValue := []byte("test.value")

	err = provider.Set(ctx, testKey, testValue)
	assert.NoError(t, err)

	value, err := provider.Get(ctx, testKey)
	assert.NoError(t, err)
	assert.Equal(t, testValue, value)

	// 测试获取不存在的键
	_, err = provider.Get(ctx, "nonexistent.key")
	assert.Error(t, err)

	// 测试监听
	watchKey := "watch.key"
	watchValue := []byte("watch.value")

	// 先设置值
	err = provider.Set(ctx, watchKey, watchValue)
	require.NoError(t, err)

	// 设置监听
	changed := make(chan []byte, 2)
	err = provider.Watch(ctx, watchKey, func(value []byte) {
		changed <- value
	})
	require.NoError(t, err)

	// 应该立即收到当前值
	select {
	case value := <-changed:
		assert.Equal(t, watchValue, value)
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for watch callback")
	}

	// 更新值
	updatedValue := []byte("updated.watch.value")
	err = provider.Set(ctx, watchKey, updatedValue)
	require.NoError(t, err)

	// 应该收到更新的值
	select {
	case value := <-changed:
		assert.Equal(t, updatedValue, value)
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for updated watch callback")
	}
}

func TestCreateRemoteProvider(t *testing.T) {
	tests := []struct {
		name    string
		opts    *RemoteConfigOptions
		wantErr bool
	}{
		{
			name:    "nil options",
			opts:    nil,
			wantErr: true,
		},
		{
			name: "mock provider",
			opts: &RemoteConfigOptions{
				Provider: "mock",
				Endpoint: "localhost:2379",
				Debug:    true,
			},
			wantErr: false,
		},
		{
			name: "etcd provider (not implemented)",
			opts: &RemoteConfigOptions{
				Provider: "etcd",
				Endpoint: "localhost:2379",
				Debug:    true,
			},
			wantErr: true,
		},
		{
			name: "consul provider (not implemented)",
			opts: &RemoteConfigOptions{
				Provider: "consul",
				Endpoint: "localhost:8500",
				Debug:    true,
			},
			wantErr: true,
		},
		{
			name: "unsupported provider",
			opts: &RemoteConfigOptions{
				Provider: "unsupported",
				Endpoint: "localhost:1234",
				Debug:    true,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := CreateRemoteProvider(tt.opts)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, provider)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, provider)
			}
		})
	}
}

func TestMockRemoteProvider_MultipleWatchers(t *testing.T) {
	provider := NewMockRemoteProvider(true)
	ctx := context.Background()

	// 连接
	err := provider.Connect(ctx)
	require.NoError(t, err)

	testKey := "multi.watch.key"
	testValue := []byte("multi.watch.value")

	// 设置多个监听者
	changed1 := make(chan []byte, 2)
	changed2 := make(chan []byte, 2)

	err = provider.Watch(ctx, testKey, func(value []byte) {
		changed1 <- value
	})
	require.NoError(t, err)

	err = provider.Watch(ctx, testKey, func(value []byte) {
		changed2 <- value
	})
	require.NoError(t, err)

	// 设置值
	err = provider.Set(ctx, testKey, testValue)
	require.NoError(t, err)

	// 两个监听者都应该收到通知
	select {
	case value := <-changed1:
		assert.Equal(t, testValue, value)
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for watcher 1")
	}

	select {
	case value := <-changed2:
		assert.Equal(t, testValue, value)
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for watcher 2")
	}

	// 更新值
	updatedValue := []byte("updated.multi.watch.value")
	err = provider.Set(ctx, testKey, updatedValue)
	require.NoError(t, err)

	// 两个监听者都应该收到更新通知
	select {
	case value := <-changed1:
		assert.Equal(t, updatedValue, value)
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for updated watcher 1")
	}

	select {
	case value := <-changed2:
		assert.Equal(t, updatedValue, value)
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for updated watcher 2")
	}
}

func TestRemoteConfig_ContextCancellation(t *testing.T) {
	provider := NewMockRemoteProvider(true)
	remoteConfig := NewRemoteConfig(provider, true)

	// 连接
	err := remoteConfig.Connect()
	require.NoError(t, err)

	// 设置监听
	testKey := "context.cancel.key"
	changed := make(chan []byte, 1)
	err = remoteConfig.Watch(testKey, func(value []byte) {
		changed <- value
	})
	require.NoError(t, err)

	// 关闭远程配置（取消上下文）
	err = remoteConfig.Close()
	assert.NoError(t, err)

	// 验证提供者也被关闭
	assert.False(t, provider.IsConnected())
}

func BenchmarkMockRemoteProvider_Get(b *testing.B) {
	provider := NewMockRemoteProvider(false)
	ctx := context.Background()

	// 连接并设置测试数据
	err := provider.Connect(ctx)
	require.NoError(b, err)

	testKey := "bench.key"
	testValue := []byte("bench.value")
	err = provider.Set(ctx, testKey, testValue)
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = provider.Get(ctx, testKey)
	}
}

func BenchmarkMockRemoteProvider_Set(b *testing.B) {
	provider := NewMockRemoteProvider(false)
	ctx := context.Background()

	// 连接
	err := provider.Connect(ctx)
	require.NoError(b, err)

	testValue := []byte("bench.value")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		testKey := "bench.key." + string(rune(i))
		_ = provider.Set(ctx, testKey, testValue)
	}
}

func BenchmarkRemoteConfig_GetSet(b *testing.B) {
	provider := NewMockRemoteProvider(false)
	remoteConfig := NewRemoteConfig(provider, false)
	defer remoteConfig.Close()

	// 连接
	err := remoteConfig.Connect()
	require.NoError(b, err)

	testValue := []byte("bench.value")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		testKey := "bench.key." + string(rune(i))
		_ = remoteConfig.Set(testKey, testValue)
		_, _ = remoteConfig.Get(testKey)
	}
}
