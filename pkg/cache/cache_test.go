package cache

import (
	"testing"
	"time"
)

func TestCacheItem_IsExpired(t *testing.T) {
	tests := []struct {
		name     string
		item     *CacheItem
		expected bool
	}{
		{
			name: "永不过期的缓存项",
			item: &CacheItem{
				Key:       "test",
				Value:     "value",
				TTL:       0,
				CreatedAt: time.Now(),
			},
			expected: false,
		},
		{
			name: "未过期的缓存项",
			item: &CacheItem{
				Key:       "test",
				Value:     "value",
				TTL:       time.Hour,
				CreatedAt: time.Now(),
			},
			expected: false,
		},
		{
			name: "已过期的缓存项",
			item: &CacheItem{
				Key:       "test",
				Value:     "value",
				TTL:       time.Millisecond,
				CreatedAt: time.Now().Add(-time.Second),
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.item.IsExpired(); got != tt.expected {
				t.Errorf("CacheItem.IsExpired() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestCacheItem_Touch(t *testing.T) {
	item := &CacheItem{
		Key:         "test",
		Value:       "value",
		AccessCount: 0,
		AccessAt:    time.Time{},
	}

	originalTime := item.AccessAt
	originalCount := item.AccessCount

	item.Touch()

	if item.AccessCount != originalCount+1 {
		t.Errorf("Touch() didn't increment AccessCount: got %d, want %d", item.AccessCount, originalCount+1)
	}

	if !item.AccessAt.After(originalTime) {
		t.Error("Touch() didn't update AccessAt time")
	}
}

func TestCacheLevel_String(t *testing.T) {
	tests := []struct {
		level    CacheLevel
		expected string
	}{
		{LevelMemory, "memory"},
		{LevelRedis, "redis"},
		{CacheLevel(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.level.String(); got != tt.expected {
				t.Errorf("CacheLevel.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// BenchmarkCacheItem_IsExpired 性能测试
func BenchmarkCacheItem_IsExpired(b *testing.B) {
	item := &CacheItem{
		Key:       "test",
		Value:     "value",
		TTL:       time.Hour,
		CreatedAt: time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = item.IsExpired()
	}
}

func BenchmarkCacheItem_Touch(b *testing.B) {
	item := &CacheItem{
		Key:   "test",
		Value: "value",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		item.Touch()
	}
}
