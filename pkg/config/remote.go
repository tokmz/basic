package config

import (
	"context"
	"fmt"
	"time"
)

// RemoteConfigProvider 远程配置提供者接口
type RemoteConfigProvider interface {
	// Connect 连接到远程配置中心
	Connect(ctx context.Context) error
	// Get 获取配置值
	Get(ctx context.Context, key string) ([]byte, error)
	// Set 设置配置值
	Set(ctx context.Context, key string, value []byte) error
	// Watch 监听配置变化
	Watch(ctx context.Context, key string, callback func([]byte)) error
	// Close 关闭连接
	Close() error
	// IsConnected 检查连接状态
	IsConnected() bool
}

// RemoteConfig 远程配置管理器
type RemoteConfig struct {
	provider RemoteConfigProvider
	ctx      context.Context
	cancel   context.CancelFunc
	debug    bool
}

// NewRemoteConfig 创建远程配置管理器
func NewRemoteConfig(provider RemoteConfigProvider, debug bool) *RemoteConfig {
	ctx, cancel := context.WithCancel(context.Background())
	return &RemoteConfig{
		provider: provider,
		ctx:      ctx,
		cancel:   cancel,
		debug:    debug,
	}
}

// Connect 连接到远程配置中心
func (rc *RemoteConfig) Connect() error {
	if rc.debug {
		fmt.Printf("[REMOTE] Connecting to remote config provider...\n")
	}

	ctx, cancel := context.WithTimeout(rc.ctx, 10*time.Second)
	defer cancel()

	err := rc.provider.Connect(ctx)
	if err != nil {
		return fmt.Errorf("failed to connect to remote config provider: %w", err)
	}

	if rc.debug {
		fmt.Printf("[REMOTE] Successfully connected to remote config provider\n")
	}

	return nil
}

// Get 获取远程配置
func (rc *RemoteConfig) Get(key string) ([]byte, error) {
	if !rc.provider.IsConnected() {
		if err := rc.Connect(); err != nil {
			return nil, err
		}
	}

	ctx, cancel := context.WithTimeout(rc.ctx, 5*time.Second)
	defer cancel()

	value, err := rc.provider.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get remote config for key %s: %w", key, err)
	}

	if rc.debug {
		fmt.Printf("[REMOTE] Retrieved config for key: %s\n", key)
	}

	return value, nil
}

// Set 设置远程配置
func (rc *RemoteConfig) Set(key string, value []byte) error {
	if !rc.provider.IsConnected() {
		if err := rc.Connect(); err != nil {
			return err
		}
	}

	ctx, cancel := context.WithTimeout(rc.ctx, 5*time.Second)
	defer cancel()

	err := rc.provider.Set(ctx, key, value)
	if err != nil {
		return fmt.Errorf("failed to set remote config for key %s: %w", key, err)
	}

	if rc.debug {
		fmt.Printf("[REMOTE] Set config for key: %s\n", key)
	}

	return nil
}

// Watch 监听远程配置变化
func (rc *RemoteConfig) Watch(key string, callback func([]byte)) error {
	if !rc.provider.IsConnected() {
		if err := rc.Connect(); err != nil {
			return err
		}
	}

	if rc.debug {
		fmt.Printf("[REMOTE] Starting to watch key: %s\n", key)
	}

	err := rc.provider.Watch(rc.ctx, key, func(value []byte) {
		if rc.debug {
			fmt.Printf("[REMOTE] Config changed for key: %s\n", key)
		}
		callback(value)
	})

	if err != nil {
		return fmt.Errorf("failed to watch remote config for key %s: %w", key, err)
	}

	return nil
}

// Close 关闭远程配置连接
func (rc *RemoteConfig) Close() error {
	if rc.cancel != nil {
		rc.cancel()
	}

	if rc.provider != nil {
		return rc.provider.Close()
	}

	return nil
}

// MockRemoteProvider 模拟远程配置提供者（用于测试）
type MockRemoteProvider struct {
	data      map[string][]byte
	watchers  map[string][]func([]byte)
	connected bool
	debug     bool
}

// NewMockRemoteProvider 创建模拟远程配置提供者
func NewMockRemoteProvider(debug bool) *MockRemoteProvider {
	return &MockRemoteProvider{
		data:     make(map[string][]byte),
		watchers: make(map[string][]func([]byte)),
		debug:    debug,
	}
}

// Connect 连接（模拟）
func (m *MockRemoteProvider) Connect(ctx context.Context) error {
	m.connected = true
	if m.debug {
		fmt.Printf("[MOCK] Connected to mock remote provider\n")
	}
	return nil
}

// Get 获取配置值（模拟）
func (m *MockRemoteProvider) Get(ctx context.Context, key string) ([]byte, error) {
	if !m.connected {
		return nil, fmt.Errorf("not connected")
	}

	value, exists := m.data[key]
	if !exists {
		return nil, fmt.Errorf("key not found: %s", key)
	}

	return value, nil
}

// Set 设置配置值（模拟）
func (m *MockRemoteProvider) Set(ctx context.Context, key string, value []byte) error {
	if !m.connected {
		return fmt.Errorf("not connected")
	}

	m.data[key] = value

	// 通知所有监听者
	if watchers, exists := m.watchers[key]; exists {
		for _, watcher := range watchers {
			go watcher(value)
		}
	}

	return nil
}

// Watch 监听配置变化（模拟）
func (m *MockRemoteProvider) Watch(ctx context.Context, key string, callback func([]byte)) error {
	if !m.connected {
		return fmt.Errorf("not connected")
	}

	if m.watchers[key] == nil {
		m.watchers[key] = make([]func([]byte), 0)
	}

	m.watchers[key] = append(m.watchers[key], callback)

	// 如果键已存在，立即触发回调
	if value, exists := m.data[key]; exists {
		go callback(value)
	}

	return nil
}

// Close 关闭连接（模拟）
func (m *MockRemoteProvider) Close() error {
	m.connected = false
	if m.debug {
		fmt.Printf("[MOCK] Disconnected from mock remote provider\n")
	}
	return nil
}

// IsConnected 检查连接状态（模拟）
func (m *MockRemoteProvider) IsConnected() bool {
	return m.connected
}

// RemoteConfigOptions 远程配置选项
type RemoteConfigOptions struct {
	Provider string            // 提供者类型 (etcd, consul, mock)
	Endpoint string            // 连接端点
	Timeout  time.Duration     // 连接超时
	Auth     map[string]string // 认证信息
	Debug    bool              // 调试模式
}

// CreateRemoteProvider 创建远程配置提供者
func CreateRemoteProvider(opts *RemoteConfigOptions) (RemoteConfigProvider, error) {
	if opts == nil {
		return nil, fmt.Errorf("remote config options cannot be nil")
	}

	switch opts.Provider {
	case "mock":
		return NewMockRemoteProvider(opts.Debug), nil
	case "etcd":
		// TODO: 实现etcd提供者
		return nil, fmt.Errorf("etcd provider not implemented yet")
	case "consul":
		// TODO: 实现consul提供者
		return nil, fmt.Errorf("consul provider not implemented yet")
	default:
		return nil, fmt.Errorf("unsupported remote provider: %s", opts.Provider)
	}
}
