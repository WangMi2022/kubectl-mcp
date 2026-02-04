package server

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sync"
	"time"
)

// CacheEntry 缓存条目
type CacheEntry struct {
	Data      interface{} // 缓存数据
	CreatedAt time.Time   // 创建时间
	ExpiresAt time.Time   // 过期时间
	HitCount  int64       // 命中次数
}

// IsExpired 检查是否过期
func (e *CacheEntry) IsExpired() bool {
	return time.Now().After(e.ExpiresAt)
}

// QueryCache 查询结果缓存
// 用于缓存 K8S 查询结果，减少对 API Server 的请求
type QueryCache struct {
	entries    map[string]*CacheEntry
	mu         sync.RWMutex
	defaultTTL time.Duration
	maxSize    int
	enabled    bool

	// 统计
	hits   int64
	misses int64
}

// CacheConfig 缓存配置
type CacheConfig struct {
	Enabled    bool          // 是否启用缓存
	DefaultTTL time.Duration // 默认过期时间
	MaxSize    int           // 最大缓存条目数
}

// NewQueryCache 创建查询缓存
func NewQueryCache(config *CacheConfig) *QueryCache {
	if config == nil {
		config = &CacheConfig{
			Enabled:    true,
			DefaultTTL: 5 * time.Minute,
			MaxSize:    1000,
		}
	}

	cache := &QueryCache{
		entries:    make(map[string]*CacheEntry),
		defaultTTL: config.DefaultTTL,
		maxSize:    config.MaxSize,
		enabled:    config.Enabled,
	}

	// 启动清理协程
	if config.Enabled {
		go cache.cleanupLoop()
	}

	return cache
}

// GenerateKey 生成缓存键
// 基于工具名称、参数和 context 生成唯一键
func (c *QueryCache) GenerateKey(toolName string, args map[string]interface{}, context string) string {
	// 构建键数据
	keyData := map[string]interface{}{
		"tool":    toolName,
		"args":    args,
		"context": context,
	}

	// 序列化为 JSON
	jsonBytes, err := json.Marshal(keyData)
	if err != nil {
		// 如果序列化失败，使用简单的字符串拼接
		return toolName + ":" + context
	}

	// 计算 SHA256 哈希
	hash := sha256.Sum256(jsonBytes)
	return hex.EncodeToString(hash[:])
}

// Get 获取缓存
func (c *QueryCache) Get(key string) (interface{}, bool) {
	if !c.enabled {
		return nil, false
	}

	c.mu.RLock()
	entry, exists := c.entries[key]
	c.mu.RUnlock()

	if !exists {
		c.mu.Lock()
		c.misses++
		c.mu.Unlock()
		return nil, false
	}

	if entry.IsExpired() {
		c.mu.Lock()
		delete(c.entries, key)
		c.misses++
		c.mu.Unlock()
		return nil, false
	}

	c.mu.Lock()
	entry.HitCount++
	c.hits++
	c.mu.Unlock()

	return entry.Data, true
}

// Set 设置缓存
func (c *QueryCache) Set(key string, data interface{}) {
	c.SetWithTTL(key, data, c.defaultTTL)
}

// SetWithTTL 设置缓存（指定 TTL）
func (c *QueryCache) SetWithTTL(key string, data interface{}, ttl time.Duration) {
	if !c.enabled {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// 检查是否需要清理
	if len(c.entries) >= c.maxSize {
		c.evictOldest()
	}

	now := time.Now()
	c.entries[key] = &CacheEntry{
		Data:      data,
		CreatedAt: now,
		ExpiresAt: now.Add(ttl),
		HitCount:  0,
	}
}

// Delete 删除缓存
func (c *QueryCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, key)
}

// Clear 清空缓存
func (c *QueryCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = make(map[string]*CacheEntry)
}

// InvalidateByPrefix 按前缀失效缓存
// 用于在资源变更时失效相关缓存
func (c *QueryCache) InvalidateByPrefix(prefix string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for key := range c.entries {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			delete(c.entries, key)
		}
	}
}

// InvalidateByTool 按工具名称失效缓存
func (c *QueryCache) InvalidateByTool(toolName string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 由于我们使用哈希作为键，需要遍历所有条目
	// 这里简单地清空所有缓存
	// 在生产环境中，可以考虑维护工具名称到键的映射
	c.entries = make(map[string]*CacheEntry)
}

// Size 获取缓存大小
func (c *QueryCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries)
}

// Stats 获取缓存统计
func (c *QueryCache) Stats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return CacheStats{
		Size:    len(c.entries),
		MaxSize: c.maxSize,
		Hits:    c.hits,
		Misses:  c.misses,
		Enabled: c.enabled,
	}
}

// CacheStats 缓存统计
type CacheStats struct {
	Size    int   `json:"size"`
	MaxSize int   `json:"max_size"`
	Hits    int64 `json:"hits"`
	Misses  int64 `json:"misses"`
	Enabled bool  `json:"enabled"`
}

// HitRate 计算命中率
func (s CacheStats) HitRate() float64 {
	total := s.Hits + s.Misses
	if total == 0 {
		return 0
	}
	return float64(s.Hits) / float64(total)
}

// evictOldest 驱逐最旧的条目（必须在持有锁的情况下调用）
func (c *QueryCache) evictOldest() {
	var oldestKey string
	var oldestTime time.Time

	for key, entry := range c.entries {
		if oldestKey == "" || entry.CreatedAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.CreatedAt
		}
	}

	if oldestKey != "" {
		delete(c.entries, oldestKey)
	}
}

// cleanupLoop 定期清理过期条目
func (c *QueryCache) cleanupLoop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.cleanup()
	}
}

// cleanup 清理过期条目
func (c *QueryCache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for key, entry := range c.entries {
		if now.After(entry.ExpiresAt) {
			delete(c.entries, key)
		}
	}
}

// Enable 启用缓存
func (c *QueryCache) Enable() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.enabled = true
}

// Disable 禁用缓存
func (c *QueryCache) Disable() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.enabled = false
}

// IsEnabled 检查缓存是否启用
func (c *QueryCache) IsEnabled() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.enabled
}

// CacheableTools 可缓存的工具列表
// 只有查询类工具的结果才应该被缓存
var CacheableTools = map[string]bool{
	"get_nodes":         true,
	"get_namespaces":    true,
	"get_pods":          true,
	"get_deployments":   true,
	"get_statefulsets":  true,
	"get_daemonsets":    true,
	"get_services":      true,
	"get_configmaps":    true,
	"get_secrets":       true,
	"get_events":        true,
	"describe_resource": true,
	// 注意：get_pod_logs 不应该被缓存，因为日志是实时的
}

// IsCacheable 检查工具是否可缓存
func IsCacheable(toolName string) bool {
	return CacheableTools[toolName]
}

// InvalidatingTools 会导致缓存失效的工具
// 这些工具执行后应该清除相关缓存
var InvalidatingTools = map[string][]string{
	"create_namespace":        {"get_namespaces"},
	"create_pod":              {"get_pods"},
	"create_deployment":       {"get_deployments", "get_pods"},
	"create_service":          {"get_services"},
	"create_configmap":        {"get_configmaps"},
	"create_secret":           {"get_secrets"},
	"delete_namespace":        {"get_namespaces"},
	"delete_pod":              {"get_pods"},
	"delete_deployment":       {"get_deployments", "get_pods"},
	"delete_service":          {"get_services"},
	"delete_configmap":        {"get_configmaps"},
	"delete_secret":           {"get_secrets"},
	"scale_deployment":        {"get_deployments", "get_pods"},
	"scale_statefulset":       {"get_statefulsets", "get_pods"},
	"update_deployment_image": {"get_deployments", "get_pods"},
	"restart_deployment":      {"get_deployments", "get_pods"},
	"apply_yaml":              {}, // 清除所有缓存
	"patch_resource":          {}, // 清除所有缓存
}

// GetInvalidatedTools 获取需要失效的工具缓存
func GetInvalidatedTools(toolName string) []string {
	return InvalidatingTools[toolName]
}
