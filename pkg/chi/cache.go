package chi

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"net/http"
	"sync"
	"time"
	
	"github.com/gin-gonic/gin"
)

// CacheProvider 缓存提供者接口
type CacheProvider interface {
	Get(ctx context.Context, key string) (interface{}, error)
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
	Clear(ctx context.Context) error
	Keys(ctx context.Context, pattern string) ([]string, error)
}

// MemoryCache 内存缓存实现
type MemoryCache struct {
	mu    sync.RWMutex
	items map[string]*cacheItem
}

type cacheItem struct {
	value     interface{}
	expiredAt time.Time
}

// NewMemoryCache 创建内存缓存
func NewMemoryCache() *MemoryCache {
	cache := &MemoryCache{
		items: make(map[string]*cacheItem),
	}
	
	// 启动清理goroutine
	go cache.cleanup()
	
	return cache
}

// Get 获取缓存值
func (mc *MemoryCache) Get(ctx context.Context, key string) (interface{}, error) {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	
	item, exists := mc.items[key]
	if !exists {
		return nil, fmt.Errorf("cache miss")
	}
	
	if time.Now().After(item.expiredAt) {
		delete(mc.items, key)
		return nil, fmt.Errorf("cache expired")
	}
	
	return item.value, nil
}

// Set 设置缓存值
func (mc *MemoryCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	expiredAt := time.Now().Add(ttl)
	if ttl <= 0 {
		expiredAt = time.Now().Add(24 * time.Hour) // 默认24小时
	}
	
	mc.items[key] = &cacheItem{
		value:     value,
		expiredAt: expiredAt,
	}
	
	return nil
}

// Delete 删除缓存值
func (mc *MemoryCache) Delete(ctx context.Context, key string) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	delete(mc.items, key)
	return nil
}

// Exists 检查键是否存在
func (mc *MemoryCache) Exists(ctx context.Context, key string) (bool, error) {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	
	item, exists := mc.items[key]
	if !exists {
		return false, nil
	}
	
	if time.Now().After(item.expiredAt) {
		delete(mc.items, key)
		return false, nil
	}
	
	return true, nil
}

// Clear 清空缓存
func (mc *MemoryCache) Clear(ctx context.Context) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	mc.items = make(map[string]*cacheItem)
	return nil
}

// Keys 获取匹配的键
func (mc *MemoryCache) Keys(ctx context.Context, pattern string) ([]string, error) {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	
	keys := make([]string, 0)
	for key := range mc.items {
		// 简单的通配符匹配（可以扩展）
		if pattern == "*" || key == pattern {
			keys = append(keys, key)
		}
	}
	
	return keys, nil
}

// cleanup 清理过期项
func (mc *MemoryCache) cleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	
	for range ticker.C {
		mc.mu.Lock()
		now := time.Now()
		for key, item := range mc.items {
			if now.After(item.expiredAt) {
				delete(mc.items, key)
			}
		}
		mc.mu.Unlock()
	}
}

// CacheManager 缓存管理器
type CacheManager struct {
	providers map[string]CacheProvider
	default_  string
	mu        sync.RWMutex
}

// NewCacheManager 创建缓存管理器
func NewCacheManager() *CacheManager {
	cm := &CacheManager{
		providers: make(map[string]CacheProvider),
		default_:  "memory",
	}
	
	// 注册默认的内存缓存
	cm.RegisterProvider("memory", NewMemoryCache())
	
	return cm
}

// RegisterProvider 注册缓存提供者
func (cm *CacheManager) RegisterProvider(name string, provider CacheProvider) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	cm.providers[name] = provider
}

// GetProvider 获取缓存提供者
func (cm *CacheManager) GetProvider(name string) CacheProvider {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	
	if name == "" {
		name = cm.default_
	}
	
	return cm.providers[name]
}

// SetDefault 设置默认缓存提供者
func (cm *CacheManager) SetDefault(name string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	cm.default_ = name
}

// Get 从默认缓存获取值
func (cm *CacheManager) Get(ctx context.Context, key string) (interface{}, error) {
	provider := cm.GetProvider("")
	if provider == nil {
		return nil, fmt.Errorf("no default cache provider")
	}
	return provider.Get(ctx, key)
}

// Set 向默认缓存设置值
func (cm *CacheManager) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	provider := cm.GetProvider("")
	if provider == nil {
		return fmt.Errorf("no default cache provider")
	}
	return provider.Set(ctx, key, value, ttl)
}

// ResponseCache 响应缓存中间件
func ResponseCacheMiddleware(cache CacheManager, ttl time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 只缓存GET请求
		if c.Request.Method != "GET" {
			c.Next()
			return
		}
		
		// 生成缓存键
		key := generateCacheKey(c)
		
		// 尝试从缓存获取
		ctx := c.Request.Context()
		if cachedResponse, err := cache.Get(ctx, key); err == nil {
			if response, ok := cachedResponse.(CachedResponse); ok {
				// 设置缓存头
				c.Header("X-Cache", "HIT")
				c.Header("X-Cache-Key", key)
				
				// 返回缓存的响应
				for k, v := range response.Headers {
					c.Header(k, v)
				}
				c.Data(response.StatusCode, response.ContentType, response.Body)
				c.Abort()
				return
			}
		}
		
		// 缓存未命中，继续处理
		c.Header("X-Cache", "MISS")
		c.Header("X-Cache-Key", key)
		
		// 使用响应写入器捕获响应
		writer := &responseWriter{
			ResponseWriter: c.Writer,
			body:          make([]byte, 0),
		}
		c.Writer = writer
		
		c.Next()
		
		// 缓存响应（只缓存200状态码）
		if writer.status == 200 && len(writer.body) > 0 {
			cachedResponse := CachedResponse{
				StatusCode:  writer.status,
				Headers:     writer.headers,
				Body:        writer.body,
				ContentType: writer.Header().Get("Content-Type"),
				CachedAt:    time.Now(),
			}
			
			cache.Set(ctx, key, cachedResponse, ttl)
		}
	}
}

// CachedResponse 缓存的响应
type CachedResponse struct {
	StatusCode  int               `json:"status_code"`
	Headers     map[string]string `json:"headers"`
	Body        []byte            `json:"body"`
	ContentType string            `json:"content_type"`
	CachedAt    time.Time         `json:"cached_at"`
}

// responseWriter 响应写入器
type responseWriter struct {
	gin.ResponseWriter
	body    []byte
	status  int
	headers map[string]string
}

// Write 写入响应体
func (rw *responseWriter) Write(data []byte) (int, error) {
	rw.body = append(rw.body, data...)
	return rw.ResponseWriter.Write(data)
}

// WriteHeader 写入响应头
func (rw *responseWriter) WriteHeader(statusCode int) {
	rw.status = statusCode
	rw.ResponseWriter.WriteHeader(statusCode)
}

// Header 获取响应头
func (rw *responseWriter) Header() http.Header {
	if rw.headers == nil {
		rw.headers = make(map[string]string)
	}
	return rw.ResponseWriter.Header()
}

// generateCacheKey 生成缓存键
func generateCacheKey(c *gin.Context) string {
	key := fmt.Sprintf("%s:%s:%s", c.Request.Method, c.Request.URL.Path, c.Request.URL.RawQuery)
	
	// 考虑用户信息
	if userID := c.GetString("user_id"); userID != "" {
		key = fmt.Sprintf("%s:user:%s", key, userID)
	}
	
	// MD5哈希
	hash := md5.Sum([]byte(key))
	return hex.EncodeToString(hash[:])
}

// CacheStats 缓存统计
type CacheStats struct {
	Hits       int64   `json:"hits"`
	Misses     int64   `json:"misses"`
	HitRate    float64 `json:"hit_rate"`
	KeyCount   int64   `json:"key_count"`
	MemorySize int64   `json:"memory_size_bytes"`
}

// StatsCacheProvider 带统计功能的缓存提供者包装器
type StatsCacheProvider struct {
	provider CacheProvider
	stats    CacheStats
	mu       sync.RWMutex
}

// NewStatsCacheProvider 创建带统计的缓存提供者
func NewStatsCacheProvider(provider CacheProvider) *StatsCacheProvider {
	return &StatsCacheProvider{
		provider: provider,
	}
}

// Get 获取值并记录统计
func (scp *StatsCacheProvider) Get(ctx context.Context, key string) (interface{}, error) {
	value, err := scp.provider.Get(ctx, key)
	
	scp.mu.Lock()
	if err == nil {
		scp.stats.Hits++
	} else {
		scp.stats.Misses++
	}
	
	total := scp.stats.Hits + scp.stats.Misses
	if total > 0 {
		scp.stats.HitRate = float64(scp.stats.Hits) / float64(total) * 100
	}
	scp.mu.Unlock()
	
	return value, err
}

// Set 设置值
func (scp *StatsCacheProvider) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	return scp.provider.Set(ctx, key, value, ttl)
}

// Delete 删除值
func (scp *StatsCacheProvider) Delete(ctx context.Context, key string) error {
	return scp.provider.Delete(ctx, key)
}

// Exists 检查存在
func (scp *StatsCacheProvider) Exists(ctx context.Context, key string) (bool, error) {
	return scp.provider.Exists(ctx, key)
}

// Clear 清空缓存
func (scp *StatsCacheProvider) Clear(ctx context.Context) error {
	return scp.provider.Clear(ctx)
}

// Keys 获取键列表
func (scp *StatsCacheProvider) Keys(ctx context.Context, pattern string) ([]string, error) {
	return scp.provider.Keys(ctx, pattern)
}

// GetStats 获取统计信息
func (scp *StatsCacheProvider) GetStats() CacheStats {
	scp.mu.RLock()
	defer scp.mu.RUnlock()
	return scp.stats
}

// ResetStats 重置统计信息
func (scp *StatsCacheProvider) ResetStats() {
	scp.mu.Lock()
	defer scp.mu.Unlock()
	scp.stats = CacheStats{}
}