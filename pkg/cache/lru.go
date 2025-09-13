package cache

import (
	"container/list"
	"sync"
)

// LRUNode LRU节点
type LRUNode struct {
	Key   string
	Value interface{}
}

// LRUCache LRU缓存实现
type LRUCache struct {
	capacity int
	cache    map[string]*list.Element
	list     *list.List
	mutex    sync.RWMutex
}

// NewLRUCache 创建LRU缓存
func NewLRUCache(capacity int) *LRUCache {
	return &LRUCache{
		capacity: capacity,
		cache:    make(map[string]*list.Element),
		list:     list.New(),
	}
}

// Get 获取值并更新位置
func (lru *LRUCache) Get(key string) interface{} {
	lru.mutex.Lock()
	defer lru.mutex.Unlock()

	if elem, exists := lru.cache[key]; exists {
		// 移动到链表头部
		lru.list.MoveToFront(elem)
		return elem.Value.(*LRUNode).Value
	}

	return nil
}

// Set 设置值
func (lru *LRUCache) Set(key string, value interface{}) {
	lru.mutex.Lock()
	defer lru.mutex.Unlock()

	// 如果容量为0，不存储任何内容
	if lru.capacity <= 0 {
		return
	}

	if elem, exists := lru.cache[key]; exists {
		// 更新现有值并移动到头部
		lru.list.MoveToFront(elem)
		elem.Value.(*LRUNode).Value = value
		return
	}

	// 检查容量限制
	if lru.list.Len() >= lru.capacity {
		lru.removeOldest()
	}

	// 添加新元素到头部
	node := &LRUNode{Key: key, Value: value}
	elem := lru.list.PushFront(node)
	lru.cache[key] = elem
}

// Remove 删除指定键
func (lru *LRUCache) Remove(key string) bool {
	lru.mutex.Lock()
	defer lru.mutex.Unlock()

	if elem, exists := lru.cache[key]; exists {
		lru.removeElement(elem)
		return true
	}

	return false
}

// RemoveOldest 删除最久未使用的元素
func (lru *LRUCache) RemoveOldest() string {
	lru.mutex.Lock()
	defer lru.mutex.Unlock()

	return lru.removeOldest()
}

// Clear 清空缓存
func (lru *LRUCache) Clear() {
	lru.mutex.Lock()
	defer lru.mutex.Unlock()

	lru.cache = make(map[string]*list.Element)
	lru.list.Init()
}

// Len 返回缓存大小
func (lru *LRUCache) Len() int {
	lru.mutex.RLock()
	defer lru.mutex.RUnlock()

	return lru.list.Len()
}

// Keys 返回所有键（从最新到最旧）
func (lru *LRUCache) Keys() []string {
	lru.mutex.RLock()
	defer lru.mutex.RUnlock()

	keys := make([]string, 0, lru.list.Len())
	for elem := lru.list.Front(); elem != nil; elem = elem.Next() {
		keys = append(keys, elem.Value.(*LRUNode).Key)
	}

	return keys
}

// 私有方法

// removeOldest 删除最久未使用的元素
func (lru *LRUCache) removeOldest() string {
	elem := lru.list.Back()
	if elem != nil {
		key := elem.Value.(*LRUNode).Key
		lru.removeElement(elem)
		return key
	}
	return ""
}

// removeElement 删除指定元素
func (lru *LRUCache) removeElement(elem *list.Element) {
	lru.list.Remove(elem)
	node := elem.Value.(*LRUNode)
	delete(lru.cache, node.Key)
}
