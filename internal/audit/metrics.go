package audit

import (
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// MetricsCollector 性能指标收集器
// 收集和暴露 Prometheus 格式的性能指标
type MetricsCollector struct {
	// 请求计数器
	totalRequests   int64 // 总请求数
	successRequests int64 // 成功请求数
	failedRequests  int64 // 失败请求数

	// 并发控制
	currentConcurrent int64 // 当前并发数
	maxConcurrent     int64 // 最大并发数（历史峰值）

	// 延迟统计
	latencySum   int64 // 延迟总和（纳秒）
	latencyCount int64 // 延迟计数

	// 工具调用统计
	toolCalls   map[string]*ToolMetrics
	toolCallsMu sync.RWMutex

	// 缓存统计
	cacheHits   int64 // 缓存命中数
	cacheMisses int64 // 缓存未命中数

	// 错误统计
	errorCounts   map[string]int64
	errorCountsMu sync.RWMutex

	// 启动时间
	startTime time.Time

	// 内存使用阈值告警
	memoryThreshold uint64 // 内存阈值（字节）
	memoryAlertFunc func(current, threshold uint64)
}

// ToolMetrics 工具调用指标
type ToolMetrics struct {
	Calls        int64 // 调用次数
	Successes    int64 // 成功次数
	Failures     int64 // 失败次数
	TotalLatency int64 // 总延迟（纳秒）
}

// MetricsSnapshot 指标快照
type MetricsSnapshot struct {
	Timestamp         time.Time                       `json:"timestamp"`
	Uptime            int64                           `json:"uptime_seconds"`
	TotalRequests     int64                           `json:"total_requests"`
	SuccessRequests   int64                           `json:"success_requests"`
	FailedRequests    int64                           `json:"failed_requests"`
	CurrentConcurrent int64                           `json:"current_concurrent"`
	MaxConcurrent     int64                           `json:"max_concurrent"`
	AvgLatencyMs      float64                         `json:"avg_latency_ms"`
	CacheHitRate      float64                         `json:"cache_hit_rate"`
	CacheHits         int64                           `json:"cache_hits"`
	CacheMisses       int64                           `json:"cache_misses"`
	ToolMetrics       map[string]*ToolMetricsSnapshot `json:"tool_metrics"`
	ErrorCounts       map[string]int64                `json:"error_counts"`
}

// ToolMetricsSnapshot 工具指标快照
type ToolMetricsSnapshot struct {
	Calls        int64   `json:"calls"`
	Successes    int64   `json:"successes"`
	Failures     int64   `json:"failures"`
	AvgLatencyMs float64 `json:"avg_latency_ms"`
	SuccessRate  float64 `json:"success_rate"`
}

// NewMetricsCollector 创建新的指标收集器
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		toolCalls:       make(map[string]*ToolMetrics),
		errorCounts:     make(map[string]int64),
		startTime:       time.Now(),
		memoryThreshold: 1024 * 1024 * 1024, // 默认 1GB
	}
}

// SetMemoryThreshold 设置内存告警阈值
func (m *MetricsCollector) SetMemoryThreshold(threshold uint64, alertFunc func(current, threshold uint64)) {
	m.memoryThreshold = threshold
	m.memoryAlertFunc = alertFunc
}

// RecordRequest 记录请求
func (m *MetricsCollector) RecordRequest(success bool, latency time.Duration) {
	atomic.AddInt64(&m.totalRequests, 1)
	if success {
		atomic.AddInt64(&m.successRequests, 1)
	} else {
		atomic.AddInt64(&m.failedRequests, 1)
	}

	// 记录延迟
	atomic.AddInt64(&m.latencySum, int64(latency))
	atomic.AddInt64(&m.latencyCount, 1)
}

// RecordToolCall 记录工具调用
func (m *MetricsCollector) RecordToolCall(toolName string, success bool, latency time.Duration) {
	m.toolCallsMu.Lock()
	defer m.toolCallsMu.Unlock()

	metrics, exists := m.toolCalls[toolName]
	if !exists {
		metrics = &ToolMetrics{}
		m.toolCalls[toolName] = metrics
	}

	atomic.AddInt64(&metrics.Calls, 1)
	if success {
		atomic.AddInt64(&metrics.Successes, 1)
	} else {
		atomic.AddInt64(&metrics.Failures, 1)
	}
	atomic.AddInt64(&metrics.TotalLatency, int64(latency))
}

// RecordError 记录错误
func (m *MetricsCollector) RecordError(errorType string) {
	m.errorCountsMu.Lock()
	defer m.errorCountsMu.Unlock()
	m.errorCounts[errorType]++
}

// RecordCacheHit 记录缓存命中
func (m *MetricsCollector) RecordCacheHit() {
	atomic.AddInt64(&m.cacheHits, 1)
}

// RecordCacheMiss 记录缓存未命中
func (m *MetricsCollector) RecordCacheMiss() {
	atomic.AddInt64(&m.cacheMisses, 1)
}

// IncrementConcurrent 增加并发计数
func (m *MetricsCollector) IncrementConcurrent() int64 {
	current := atomic.AddInt64(&m.currentConcurrent, 1)
	// 更新最大并发数
	for {
		max := atomic.LoadInt64(&m.maxConcurrent)
		if current <= max {
			break
		}
		if atomic.CompareAndSwapInt64(&m.maxConcurrent, max, current) {
			break
		}
	}
	return current
}

// DecrementConcurrent 减少并发计数
func (m *MetricsCollector) DecrementConcurrent() int64 {
	return atomic.AddInt64(&m.currentConcurrent, -1)
}

// GetCurrentConcurrent 获取当前并发数
func (m *MetricsCollector) GetCurrentConcurrent() int64 {
	return atomic.LoadInt64(&m.currentConcurrent)
}

// GetSnapshot 获取指标快照
func (m *MetricsCollector) GetSnapshot() *MetricsSnapshot {
	snapshot := &MetricsSnapshot{
		Timestamp:         time.Now(),
		Uptime:            int64(time.Since(m.startTime).Seconds()),
		TotalRequests:     atomic.LoadInt64(&m.totalRequests),
		SuccessRequests:   atomic.LoadInt64(&m.successRequests),
		FailedRequests:    atomic.LoadInt64(&m.failedRequests),
		CurrentConcurrent: atomic.LoadInt64(&m.currentConcurrent),
		MaxConcurrent:     atomic.LoadInt64(&m.maxConcurrent),
		CacheHits:         atomic.LoadInt64(&m.cacheHits),
		CacheMisses:       atomic.LoadInt64(&m.cacheMisses),
		ToolMetrics:       make(map[string]*ToolMetricsSnapshot),
		ErrorCounts:       make(map[string]int64),
	}

	// 计算平均延迟
	latencyCount := atomic.LoadInt64(&m.latencyCount)
	if latencyCount > 0 {
		latencySum := atomic.LoadInt64(&m.latencySum)
		snapshot.AvgLatencyMs = float64(latencySum) / float64(latencyCount) / float64(time.Millisecond)
	}

	// 计算缓存命中率
	totalCache := snapshot.CacheHits + snapshot.CacheMisses
	if totalCache > 0 {
		snapshot.CacheHitRate = float64(snapshot.CacheHits) / float64(totalCache)
	}

	// 复制工具指标
	m.toolCallsMu.RLock()
	for name, metrics := range m.toolCalls {
		calls := atomic.LoadInt64(&metrics.Calls)
		successes := atomic.LoadInt64(&metrics.Successes)
		failures := atomic.LoadInt64(&metrics.Failures)
		totalLatency := atomic.LoadInt64(&metrics.TotalLatency)

		toolSnapshot := &ToolMetricsSnapshot{
			Calls:     calls,
			Successes: successes,
			Failures:  failures,
		}

		if calls > 0 {
			toolSnapshot.AvgLatencyMs = float64(totalLatency) / float64(calls) / float64(time.Millisecond)
			toolSnapshot.SuccessRate = float64(successes) / float64(calls)
		}

		snapshot.ToolMetrics[name] = toolSnapshot
	}
	m.toolCallsMu.RUnlock()

	// 复制错误计数
	m.errorCountsMu.RLock()
	for errType, count := range m.errorCounts {
		snapshot.ErrorCounts[errType] = count
	}
	m.errorCountsMu.RUnlock()

	return snapshot
}

// PrometheusHandler 返回 Prometheus 格式的指标 HTTP 处理器
func (m *MetricsCollector) PrometheusHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		snapshot := m.GetSnapshot()

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")

		// 输出 Prometheus 格式的指标
		fmt.Fprintf(w, "# HELP kubectl_mcp_requests_total Total number of requests\n")
		fmt.Fprintf(w, "# TYPE kubectl_mcp_requests_total counter\n")
		fmt.Fprintf(w, "kubectl_mcp_requests_total %d\n", snapshot.TotalRequests)

		fmt.Fprintf(w, "# HELP kubectl_mcp_requests_success_total Total number of successful requests\n")
		fmt.Fprintf(w, "# TYPE kubectl_mcp_requests_success_total counter\n")
		fmt.Fprintf(w, "kubectl_mcp_requests_success_total %d\n", snapshot.SuccessRequests)

		fmt.Fprintf(w, "# HELP kubectl_mcp_requests_failed_total Total number of failed requests\n")
		fmt.Fprintf(w, "# TYPE kubectl_mcp_requests_failed_total counter\n")
		fmt.Fprintf(w, "kubectl_mcp_requests_failed_total %d\n", snapshot.FailedRequests)

		fmt.Fprintf(w, "# HELP kubectl_mcp_concurrent_requests Current number of concurrent requests\n")
		fmt.Fprintf(w, "# TYPE kubectl_mcp_concurrent_requests gauge\n")
		fmt.Fprintf(w, "kubectl_mcp_concurrent_requests %d\n", snapshot.CurrentConcurrent)

		fmt.Fprintf(w, "# HELP kubectl_mcp_max_concurrent_requests Maximum number of concurrent requests\n")
		fmt.Fprintf(w, "# TYPE kubectl_mcp_max_concurrent_requests gauge\n")
		fmt.Fprintf(w, "kubectl_mcp_max_concurrent_requests %d\n", snapshot.MaxConcurrent)

		fmt.Fprintf(w, "# HELP kubectl_mcp_request_latency_ms Average request latency in milliseconds\n")
		fmt.Fprintf(w, "# TYPE kubectl_mcp_request_latency_ms gauge\n")
		fmt.Fprintf(w, "kubectl_mcp_request_latency_ms %.2f\n", snapshot.AvgLatencyMs)

		fmt.Fprintf(w, "# HELP kubectl_mcp_cache_hits_total Total number of cache hits\n")
		fmt.Fprintf(w, "# TYPE kubectl_mcp_cache_hits_total counter\n")
		fmt.Fprintf(w, "kubectl_mcp_cache_hits_total %d\n", snapshot.CacheHits)

		fmt.Fprintf(w, "# HELP kubectl_mcp_cache_misses_total Total number of cache misses\n")
		fmt.Fprintf(w, "# TYPE kubectl_mcp_cache_misses_total counter\n")
		fmt.Fprintf(w, "kubectl_mcp_cache_misses_total %d\n", snapshot.CacheMisses)

		fmt.Fprintf(w, "# HELP kubectl_mcp_cache_hit_rate Cache hit rate\n")
		fmt.Fprintf(w, "# TYPE kubectl_mcp_cache_hit_rate gauge\n")
		fmt.Fprintf(w, "kubectl_mcp_cache_hit_rate %.4f\n", snapshot.CacheHitRate)

		fmt.Fprintf(w, "# HELP kubectl_mcp_uptime_seconds Server uptime in seconds\n")
		fmt.Fprintf(w, "# TYPE kubectl_mcp_uptime_seconds counter\n")
		fmt.Fprintf(w, "kubectl_mcp_uptime_seconds %d\n", snapshot.Uptime)

		// 工具调用指标
		fmt.Fprintf(w, "# HELP kubectl_mcp_tool_calls_total Total number of tool calls\n")
		fmt.Fprintf(w, "# TYPE kubectl_mcp_tool_calls_total counter\n")
		for name, metrics := range snapshot.ToolMetrics {
			fmt.Fprintf(w, "kubectl_mcp_tool_calls_total{tool=\"%s\"} %d\n", name, metrics.Calls)
		}

		fmt.Fprintf(w, "# HELP kubectl_mcp_tool_success_rate Tool success rate\n")
		fmt.Fprintf(w, "# TYPE kubectl_mcp_tool_success_rate gauge\n")
		for name, metrics := range snapshot.ToolMetrics {
			fmt.Fprintf(w, "kubectl_mcp_tool_success_rate{tool=\"%s\"} %.4f\n", name, metrics.SuccessRate)
		}

		fmt.Fprintf(w, "# HELP kubectl_mcp_tool_latency_ms Average tool latency in milliseconds\n")
		fmt.Fprintf(w, "# TYPE kubectl_mcp_tool_latency_ms gauge\n")
		for name, metrics := range snapshot.ToolMetrics {
			fmt.Fprintf(w, "kubectl_mcp_tool_latency_ms{tool=\"%s\"} %.2f\n", name, metrics.AvgLatencyMs)
		}

		// 错误计数
		fmt.Fprintf(w, "# HELP kubectl_mcp_errors_total Total number of errors by type\n")
		fmt.Fprintf(w, "# TYPE kubectl_mcp_errors_total counter\n")
		for errType, count := range snapshot.ErrorCounts {
			fmt.Fprintf(w, "kubectl_mcp_errors_total{type=\"%s\"} %d\n", errType, count)
		}
	}
}

// Reset 重置所有指标
func (m *MetricsCollector) Reset() {
	atomic.StoreInt64(&m.totalRequests, 0)
	atomic.StoreInt64(&m.successRequests, 0)
	atomic.StoreInt64(&m.failedRequests, 0)
	atomic.StoreInt64(&m.currentConcurrent, 0)
	atomic.StoreInt64(&m.maxConcurrent, 0)
	atomic.StoreInt64(&m.latencySum, 0)
	atomic.StoreInt64(&m.latencyCount, 0)
	atomic.StoreInt64(&m.cacheHits, 0)
	atomic.StoreInt64(&m.cacheMisses, 0)

	m.toolCallsMu.Lock()
	m.toolCalls = make(map[string]*ToolMetrics)
	m.toolCallsMu.Unlock()

	m.errorCountsMu.Lock()
	m.errorCounts = make(map[string]int64)
	m.errorCountsMu.Unlock()

	m.startTime = time.Now()
}
