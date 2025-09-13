package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sync/singleflight"
)

// DistributedCache 分布式缓存
type DistributedCache struct {
	config       DistributedConfig
	localCache   Cache
	remoteCache  Cache
	consistency  ConsistencyManager
	synchronizer *Synchronizer
	eventBus     *EventBus
	nodeManager  *NodeManager
	singleflight singleflight.Group
	closed       int32
	mu           sync.RWMutex
}

// ConsistencyManager 一致性管理器
type ConsistencyManager interface {
	EnsureConsistency(ctx context.Context, key string, operation string) error
	CheckConsistency(ctx context.Context, keys []string) ([]string, error)
	Invalidate(ctx context.Context, key string) error
}

// Synchronizer 同步器
type Synchronizer struct {
	config      DistributedConfig
	eventBus    *EventBus
	nodeManager *NodeManager
	ticker      *time.Ticker
	stopCh      chan struct{}
	mu          sync.RWMutex
}

// EventBus 事件总线
type EventBus struct {
	subscribers map[string][]EventHandler
	mu          sync.RWMutex
}

// EventHandler 事件处理器
type EventHandler func(event *CacheEvent) error

// CacheEvent 缓存事件
type CacheEvent struct {
	Type      string        `json:"type"`
	Key       string        `json:"key"`
	Value     interface{}   `json:"value,omitempty"`
	TTL       time.Duration `json:"ttl,omitempty"`
	NodeID    string        `json:"node_id"`
	Timestamp time.Time     `json:"timestamp"`
	Checksum  string        `json:"checksum,omitempty"`
}

// NodeManager 节点管理器
type NodeManager struct {
	nodeID    string
	nodes     map[string]*Node
	heartbeat *time.Ticker
	mu        sync.RWMutex
}

// Node 节点信息
type Node struct {
	ID       string    `json:"id"`
	Address  string    `json:"address"`
	Status   string    `json:"status"`
	LastSeen time.Time `json:"last_seen"`
	Version  string    `json:"version"`
}

// EventualConsistency 最终一致性实现
type EventualConsistency struct {
	config      DistributedConfig
	nodeManager *NodeManager
	eventBus    *EventBus
}

// StrongConsistency 强一致性实现
type StrongConsistency struct {
	config      DistributedConfig
	nodeManager *NodeManager
	eventBus    *EventBus
	locks       sync.Map // 分布式锁
}

// NewDistributedCache 创建分布式缓存
func NewDistributedCache(config Config) (*DistributedCache, error) {
	if !config.Distributed.EnableSync {
		return nil, fmt.Errorf("distributed sync is not enabled")
	}

	dc := &DistributedCache{
		config: config.Distributed,
	}

	// 创建本地缓存
	dc.localCache = NewMemoryCache(config.Memory)

	// 创建远程缓存
	if config.MultiLevel.EnableL2 {
		remoteCache, err := NewRedisCache(config.Redis)
		if err != nil {
			return nil, err
		}
		dc.remoteCache = remoteCache
	}

	// 创建节点管理器
	dc.nodeManager = NewNodeManager(config.Distributed.NodeID)

	// 创建事件总线
	dc.eventBus = NewEventBus()

	// 创建一致性管理器
	switch config.Distributed.ConsistencyLevel {
	case ConsistencyEventual:
		dc.consistency = NewEventualConsistency(config.Distributed, dc.nodeManager, dc.eventBus)
	case ConsistencyStrong:
		dc.consistency = NewStrongConsistency(config.Distributed, dc.nodeManager, dc.eventBus)
	default:
		dc.consistency = NewEventualConsistency(config.Distributed, dc.nodeManager, dc.eventBus)
	}

	// 创建同步器
	dc.synchronizer = NewSynchronizer(config.Distributed, dc.eventBus, dc.nodeManager)

	return dc, nil
}

// Get 获取缓存值
func (dc *DistributedCache) Get(ctx context.Context, key string) (interface{}, error) {
	if atomic.LoadInt32(&dc.closed) == 1 {
		return nil, ErrCacheClosed
	}

	// 使用读锁保护缓存状态
	dc.mu.RLock()
	defer dc.mu.RUnlock()

	// 使用singleflight防止缓存击穿
	result, err, _ := dc.singleflight.Do(key, func() (interface{}, error) {
		return dc.doGet(ctx, key)
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}

// Set 设置缓存值
func (dc *DistributedCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	if atomic.LoadInt32(&dc.closed) == 1 {
		return ErrCacheClosed
	}

	// 使用写锁保护写操作
	dc.mu.Lock()
	defer dc.mu.Unlock()

	// 确保一致性
	if err := dc.consistency.EnsureConsistency(ctx, key, "set"); err != nil {
		return err
	}

	// 设置到本地缓存
	if err := dc.localCache.Set(ctx, key, value, ttl); err != nil {
		return err
	}

	// 设置到远程缓存
	if dc.remoteCache != nil {
		if err := dc.remoteCache.Set(ctx, key, value, ttl); err != nil {
			// 远程失败不影响本地，但需要记录
			return err
		}
	}

	// 发布缓存事件
	event := &CacheEvent{
		Type:      "set",
		Key:       key,
		Value:     value,
		TTL:       ttl,
		NodeID:    dc.config.NodeID,
		Timestamp: time.Now(),
		Checksum:  dc.calculateChecksum(key, value),
	}

	dc.eventBus.Publish("cache.set", event)

	return nil
}

// Delete 删除缓存键
func (dc *DistributedCache) Delete(ctx context.Context, key string) error {
	if atomic.LoadInt32(&dc.closed) == 1 {
		return ErrCacheClosed
	}

	// 确保一致性
	if err := dc.consistency.EnsureConsistency(ctx, key, "delete"); err != nil {
		return err
	}

	// 从本地缓存删除
	dc.localCache.Delete(ctx, key)

	// 从远程缓存删除
	if dc.remoteCache != nil {
		dc.remoteCache.Delete(ctx, key)
	}

	// 发布缓存事件
	event := &CacheEvent{
		Type:      "delete",
		Key:       key,
		NodeID:    dc.config.NodeID,
		Timestamp: time.Now(),
	}

	dc.eventBus.Publish("cache.delete", event)

	return nil
}

// Exists 检查键是否存在
func (dc *DistributedCache) Exists(ctx context.Context, key string) (bool, error) {
	if atomic.LoadInt32(&dc.closed) == 1 {
		return false, ErrCacheClosed
	}

	// 先检查本地缓存
	if exists, err := dc.localCache.Exists(ctx, key); err == nil && exists {
		return true, nil
	}

	// 再检查远程缓存
	if dc.remoteCache != nil {
		return dc.remoteCache.Exists(ctx, key)
	}

	return false, nil
}

// Clear 清空所有缓存
func (dc *DistributedCache) Clear(ctx context.Context) error {
	if atomic.LoadInt32(&dc.closed) == 1 {
		return ErrCacheClosed
	}

	// 清空本地缓存
	if err := dc.localCache.Clear(ctx); err != nil {
		return err
	}

	// 清空远程缓存
	if dc.remoteCache != nil {
		if err := dc.remoteCache.Clear(ctx); err != nil {
			return err
		}
	}

	// 发布清空事件
	event := &CacheEvent{
		Type:      "clear",
		NodeID:    dc.config.NodeID,
		Timestamp: time.Now(),
	}

	dc.eventBus.Publish("cache.clear", event)

	return nil
}

// Keys 获取匹配模式的所有键
func (dc *DistributedCache) Keys(ctx context.Context, pattern string) ([]string, error) {
	if atomic.LoadInt32(&dc.closed) == 1 {
		return nil, ErrCacheClosed
	}

	keyMap := make(map[string]bool)

	// 从本地缓存获取
	if localKeys, err := dc.localCache.Keys(ctx, pattern); err == nil {
		for _, key := range localKeys {
			keyMap[key] = true
		}
	}

	// 从远程缓存获取
	if dc.remoteCache != nil {
		if remoteKeys, err := dc.remoteCache.Keys(ctx, pattern); err == nil {
			for _, key := range remoteKeys {
				keyMap[key] = true
			}
		}
	}

	result := make([]string, 0, len(keyMap))
	for key := range keyMap {
		result = append(result, key)
	}

	return result, nil
}

// Stats 获取缓存统计信息
func (dc *DistributedCache) Stats() CacheStats {
	var totalStats CacheStats

	// 本地缓存统计
	if dc.localCache != nil {
		localStats := dc.localCache.Stats()
		totalStats.Hits += localStats.Hits
		totalStats.Misses += localStats.Misses
		totalStats.Sets += localStats.Sets
		totalStats.Deletes += localStats.Deletes
		totalStats.Errors += localStats.Errors
		totalStats.TotalRequests += localStats.TotalRequests
		totalStats.Size += localStats.Size
	}

	// 远程缓存统计
	if dc.remoteCache != nil {
		remoteStats := dc.remoteCache.Stats()
		totalStats.Hits += remoteStats.Hits
		totalStats.Misses += remoteStats.Misses
		totalStats.Sets += remoteStats.Sets
		totalStats.Deletes += remoteStats.Deletes
		totalStats.Errors += remoteStats.Errors
		totalStats.TotalRequests += remoteStats.TotalRequests
	}

	// 计算命中率
	if totalStats.TotalRequests > 0 {
		totalStats.HitRate = float64(totalStats.Hits) / float64(totalStats.TotalRequests)
	}

	return totalStats
}

// Close 关闭缓存
func (dc *DistributedCache) Close() error {
	if !atomic.CompareAndSwapInt32(&dc.closed, 0, 1) {
		return nil
	}

	var errs []error

	// 停止同步器
	if dc.synchronizer != nil {
		if err := dc.synchronizer.Stop(); err != nil {
			errs = append(errs, err)
		}
	}

	// 关闭本地缓存
	if dc.localCache != nil {
		if err := dc.localCache.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	// 关闭远程缓存
	if dc.remoteCache != nil {
		if err := dc.remoteCache.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	// 停止节点管理器
	if dc.nodeManager != nil {
		dc.nodeManager.Stop()
	}

	if len(errs) > 0 {
		return errs[0]
	}

	return nil
}

// Start 启动分布式缓存
func (dc *DistributedCache) Start(ctx context.Context) error {
	// 启动节点管理器
	if err := dc.nodeManager.Start(ctx); err != nil {
		return err
	}

	// 启动同步器
	if err := dc.synchronizer.Start(ctx); err != nil {
		return err
	}

	// 订阅缓存事件
	dc.eventBus.Subscribe("cache.set", dc.handleSetEvent)
	dc.eventBus.Subscribe("cache.delete", dc.handleDeleteEvent)
	dc.eventBus.Subscribe("cache.clear", dc.handleClearEvent)

	return nil
}

// 私有方法

// doGet 执行获取操作
func (dc *DistributedCache) doGet(ctx context.Context, key string) (interface{}, error) {
	// 先从本地缓存获取
	if value, err := dc.localCache.Get(ctx, key); err == nil {
		return value, nil
	}

	// 从远程缓存获取
	if dc.remoteCache != nil {
		if value, err := dc.remoteCache.Get(ctx, key); err == nil {
			// 异步同步到本地缓存
			go func() {
				dc.localCache.Set(context.Background(), key, value, time.Hour)
			}()
			return value, nil
		}
	}

	return nil, ErrKeyNotFound
}

// calculateChecksum 计算校验和
func (dc *DistributedCache) calculateChecksum(key string, value interface{}) string {
	data, _ := json.Marshal(map[string]interface{}{
		"key":   key,
		"value": value,
	})

	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// handleSetEvent 处理设置事件
func (dc *DistributedCache) handleSetEvent(event *CacheEvent) error {
	if event.NodeID == dc.config.NodeID {
		return nil // 忽略自己的事件
	}

	// 同步到本地缓存
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return dc.localCache.Set(ctx, event.Key, event.Value, event.TTL)
}

// handleDeleteEvent 处理删除事件
func (dc *DistributedCache) handleDeleteEvent(event *CacheEvent) error {
	if event.NodeID == dc.config.NodeID {
		return nil // 忽略自己的事件
	}

	// 从本地缓存删除
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return dc.localCache.Delete(ctx, event.Key)
}

// handleClearEvent 处理清空事件
func (dc *DistributedCache) handleClearEvent(event *CacheEvent) error {
	if event.NodeID == dc.config.NodeID {
		return nil // 忽略自己的事件
	}

	// 清空本地缓存
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return dc.localCache.Clear(ctx)
}

// 工厂函数

// NewEventBus 创建事件总线
func NewEventBus() *EventBus {
	return &EventBus{
		subscribers: make(map[string][]EventHandler),
	}
}

// Subscribe 订阅事件
func (eb *EventBus) Subscribe(event string, handler EventHandler) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	eb.subscribers[event] = append(eb.subscribers[event], handler)
}

// Publish 发布事件
func (eb *EventBus) Publish(event string, data *CacheEvent) {
	eb.mu.RLock()
	handlers := eb.subscribers[event]
	eb.mu.RUnlock()

	for _, handler := range handlers {
		go handler(data)
	}
}

// NewNodeManager 创建节点管理器
func NewNodeManager(nodeID string) *NodeManager {
	return &NodeManager{
		nodeID: nodeID,
		nodes:  make(map[string]*Node),
	}
}

// Start 启动节点管理器
func (nm *NodeManager) Start(ctx context.Context) error {
	nm.heartbeat = time.NewTicker(30 * time.Second)
	go nm.heartbeatLoop(ctx)
	return nil
}

// Stop 停止节点管理器
func (nm *NodeManager) Stop() {
	if nm.heartbeat != nil {
		nm.heartbeat.Stop()
	}
}

// heartbeatLoop 心跳循环
func (nm *NodeManager) heartbeatLoop(ctx context.Context) {
	for {
		select {
		case <-nm.heartbeat.C:
			nm.sendHeartbeat()
		case <-ctx.Done():
			return
		}
	}
}

// sendHeartbeat 发送心跳
func (nm *NodeManager) sendHeartbeat() {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	// 实现心跳逻辑
	currentTime := time.Now()

	// 更新当前节点状态
	if nm.nodes == nil {
		nm.nodes = make(map[string]*Node)
	}

	// 更新或添加当前节点
	nm.nodes[nm.nodeID] = &Node{
		ID:       nm.nodeID,
		Status:   "active",
		LastSeen: currentTime,
		Version:  "1.0.0",
	}

	// 检查并清理过期节点
	for nodeID, node := range nm.nodes {
		if nodeID != nm.nodeID && time.Since(node.LastSeen) > 90*time.Second {
			node.Status = "inactive"
		}
	}
}

// NewSynchronizer 创建同步器
func NewSynchronizer(config DistributedConfig, eventBus *EventBus, nodeManager *NodeManager) *Synchronizer {
	return &Synchronizer{
		config:      config,
		eventBus:    eventBus,
		nodeManager: nodeManager,
		stopCh:      make(chan struct{}),
	}
}

// Start 启动同步器
func (s *Synchronizer) Start(ctx context.Context) error {
	if s.config.SyncInterval > 0 {
		s.ticker = time.NewTicker(s.config.SyncInterval)
		go s.syncLoop(ctx)
	}
	return nil
}

// Stop 停止同步器
func (s *Synchronizer) Stop() error {
	if s.ticker != nil {
		s.ticker.Stop()
	}
	close(s.stopCh)
	return nil
}

// syncLoop 同步循环
func (s *Synchronizer) syncLoop(ctx context.Context) {
	for {
		select {
		case <-s.ticker.C:
			s.sync(ctx)
		case <-s.stopCh:
			return
		case <-ctx.Done():
			return
		}
	}
}

// sync 执行同步
func (s *Synchronizer) sync(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 获取当前活跃节点
	if s.nodeManager == nil {
		return
	}

	// 实现同步逻辑
	// 1. 检查节点状态
	// 2. 同步缓存数据
	// 3. 处理节点失效

	// 发布同步事件
	event := &CacheEvent{
		Type:      "sync",
		NodeID:    s.config.NodeID,
		Timestamp: time.Now(),
	}

	if s.eventBus != nil {
		s.eventBus.Publish("cache.sync", event)
	}
}

// 一致性实现

// NewEventualConsistency 创建最终一致性管理器
func NewEventualConsistency(config DistributedConfig, nodeManager *NodeManager, eventBus *EventBus) *EventualConsistency {
	return &EventualConsistency{
		config:      config,
		nodeManager: nodeManager,
		eventBus:    eventBus,
	}
}

// EnsureConsistency 确保一致性
func (ec *EventualConsistency) EnsureConsistency(ctx context.Context, key string, operation string) error {
	// 最终一致性不需要同步等待，但需要记录操作以便后续同步

	if key == "" {
		return fmt.Errorf("key cannot be empty")
	}

	if operation == "" {
		return fmt.Errorf("operation cannot be empty")
	}

	// 发布操作事件以便其他节点最终同步
	if ec.eventBus != nil {
		event := &CacheEvent{
			Type:      "consistency_check",
			Key:       key,
			NodeID:    ec.config.NodeID,
			Timestamp: time.Now(),
		}

		ec.eventBus.Publish("cache.consistency", event)
	}

	return nil
}

// CheckConsistency 检查一致性
func (ec *EventualConsistency) CheckConsistency(ctx context.Context, keys []string) ([]string, error) {
	if len(keys) == 0 {
		return []string{}, nil
	}

	inconsistentKeys := make([]string, 0)

	// 对于最终一致性，我们检查是否有键需要重新同步
	// 实际实现中，这里会检查各节点的键版本或时间戳

	// 检查节点管理器状态
	if ec.nodeManager == nil {
		return keys, fmt.Errorf("node manager not available")
	}

	// 模拟一致性检查逻辑
	for _, key := range keys {
		// 在实际场景中，这里会：
		// 1. 查询各节点的键状态
		// 2. 比较版本号或时间戳
		// 3. 识别不一致的键

		// 为了演示，我们假设某些键可能不一致
		if len(key) > 10 { // 示例条件：长键名可能不一致
			inconsistentKeys = append(inconsistentKeys, key)
		}
	}

	return inconsistentKeys, nil
}

// Invalidate 失效处理
func (ec *EventualConsistency) Invalidate(ctx context.Context, key string) error {
	// 发布失效事件
	event := &CacheEvent{
		Type:      "invalidate",
		Key:       key,
		NodeID:    ec.config.NodeID,
		Timestamp: time.Now(),
	}

	ec.eventBus.Publish("cache.invalidate", event)
	return nil
}

// NewStrongConsistency 创建强一致性管理器
func NewStrongConsistency(config DistributedConfig, nodeManager *NodeManager, eventBus *EventBus) *StrongConsistency {
	return &StrongConsistency{
		config:      config,
		nodeManager: nodeManager,
		eventBus:    eventBus,
	}
}

// EnsureConsistency 确保强一致性
func (sc *StrongConsistency) EnsureConsistency(ctx context.Context, key string, operation string) error {
	if key == "" {
		return fmt.Errorf("key cannot be empty")
	}

	if operation == "" {
		return fmt.Errorf("operation cannot be empty")
	}

	// 获取分布式锁
	lockKey := fmt.Sprintf("lock:%s", key)
	lockTime := time.Now()

	// 简单的锁实现(实际项目中需要更完善的分布式锁)
	if _, loaded := sc.locks.LoadOrStore(lockKey, lockTime); loaded {
		return fmt.Errorf("key is locked by another operation")
	}

	// 操作完成后释放锁
	defer sc.locks.Delete(lockKey)

	// 强一致性需要确保操作在所有节点上同步完成
	// 在实际实现中，这里会：
	// 1. 向所有活跃节点发送操作请求
	// 2. 等待所有节点确认
	// 3. 只有当所有节点都成功时才返回成功

	// 发布强一致性事件
	if sc.eventBus != nil {
		event := &CacheEvent{
			Type:      "strong_consistency_operation",
			Key:       key,
			NodeID:    sc.config.NodeID,
			Timestamp: lockTime,
		}

		sc.eventBus.Publish("cache.strong_consistency", event)
	}

	return nil
}

// CheckConsistency 检查强一致性
func (sc *StrongConsistency) CheckConsistency(ctx context.Context, keys []string) ([]string, error) {
	if len(keys) == 0 {
		return []string{}, nil
	}

	inconsistentKeys := make([]string, 0)

	// 检查节点管理器状态
	if sc.nodeManager == nil {
		return keys, fmt.Errorf("node manager not available")
	}

	// 强一致性要求严格的同步检查
	for _, key := range keys {
		// 检查键是否被锁定
		lockKey := fmt.Sprintf("lock:%s", key)
		if _, locked := sc.locks.Load(lockKey); locked {
			// 被锁定的键可能不一致
			inconsistentKeys = append(inconsistentKeys, key)
			continue
		}

		// 在实际场景中，这里会：
		// 1. 向所有活跃节点查询键的状态
		// 2. 比较值和版本号
		// 3. 确保所有节点的数据完全一致
		// 4. 如有不一致，触发同步机制

		// 模拟检查：检查键的长度作为示例条件
		if len(key) > 8 && len(key)%2 == 0 {
			inconsistentKeys = append(inconsistentKeys, key)
		}
	}

	// 如果发现不一致，记录事件
	if len(inconsistentKeys) > 0 && sc.eventBus != nil {
		event := &CacheEvent{
			Type:      "consistency_violation",
			NodeID:    sc.config.NodeID,
			Timestamp: time.Now(),
		}
		sc.eventBus.Publish("cache.consistency_violation", event)
	}

	return inconsistentKeys, nil
}

// Invalidate 强一致性失效处理
func (sc *StrongConsistency) Invalidate(ctx context.Context, key string) error {
	// 同步失效到所有节点
	event := &CacheEvent{
		Type:      "invalidate",
		Key:       key,
		NodeID:    sc.config.NodeID,
		Timestamp: time.Now(),
	}

	sc.eventBus.Publish("cache.invalidate", event)
	return nil
}
