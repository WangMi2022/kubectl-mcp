package audit

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// UserInfo 定义用户信息
type UserInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Role string `json:"role,omitempty"`
}

// OperationLog 定义操作日志的结构
type OperationLog struct {
	Timestamp    time.Time              `json:"timestamp"`
	User         *UserInfo              `json:"user,omitempty"`
	Tool         string                 `json:"tool"`
	Arguments    map[string]interface{} `json:"arguments,omitempty"`
	Context      string                 `json:"context"`
	Namespace    string                 `json:"namespace,omitempty"`
	Success      bool                   `json:"success"`
	Error        string                 `json:"error,omitempty"`
	Duration     time.Duration          `json:"duration"`
	ResourceType string                 `json:"resourceType,omitempty"`
	ResourceName string                 `json:"resourceName,omitempty"`
}

// Metrics 定义性能指标
type Metrics struct {
	Timestamp       time.Time     `json:"timestamp"`
	Operation       string        `json:"operation"`
	Duration        time.Duration `json:"duration"`
	Success         bool          `json:"success"`
	ConcurrentCount int           `json:"concurrentCount,omitempty"`
}

// LoggerConfig 定义日志配置
type LoggerConfig struct {
	Level      string // 日志级别: debug, info, warn, error
	Format     string // 日志格式: json, text
	Output     string // 输出目标: stdout, file, both
	FilePath   string // 日志文件路径
	MaxSize    int    // 日志文件最大大小(MB)
	MaxBackups int    // 保留的旧日志文件数量
	MaxAge     int    // 保留旧日志文件的最大天数
	Compress   bool   // 是否压缩旧日志文件
}

// AuditLogger 定义审计日志系统
type AuditLogger struct {
	logger *zap.Logger
	config *LoggerConfig
	file   *os.File // 保存文件句柄以便关闭
	mu     sync.RWMutex
}

// NewAuditLogger 创建新的审计日志器
func NewAuditLogger(config *LoggerConfig) (*AuditLogger, error) {
	if config == nil {
		config = &LoggerConfig{
			Level:      "info",
			Format:     "json",
			Output:     "stdout",
			MaxSize:    100,
			MaxBackups: 3,
			MaxAge:     28,
			Compress:   true,
		}
	}

	// 解析日志级别
	var level zapcore.Level
	switch config.Level {
	case "debug":
		level = zapcore.DebugLevel
	case "info":
		level = zapcore.InfoLevel
	case "warn":
		level = zapcore.WarnLevel
	case "error":
		level = zapcore.ErrorLevel
	default:
		level = zapcore.InfoLevel
	}

	// 配置编码器
	var encoderConfig zapcore.EncoderConfig
	if config.Format == "json" {
		encoderConfig = zap.NewProductionEncoderConfig()
	} else {
		encoderConfig = zap.NewDevelopmentEncoderConfig()
	}
	encoderConfig.TimeKey = "timestamp"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	// 创建编码器
	var encoder zapcore.Encoder
	if config.Format == "json" {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}

	// 配置输出
	var writeSyncer zapcore.WriteSyncer
	var file *os.File
	switch config.Output {
	case "file":
		if config.FilePath == "" {
			return nil, fmt.Errorf("日志文件路径不能为空")
		}
		var err error
		file, err = os.OpenFile(config.FilePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return nil, fmt.Errorf("打开日志文件失败: %w", err)
		}
		writeSyncer = zapcore.AddSync(file)
	case "both":
		if config.FilePath == "" {
			return nil, fmt.Errorf("日志文件路径不能为空")
		}
		var err error
		file, err = os.OpenFile(config.FilePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return nil, fmt.Errorf("打开日志文件失败: %w", err)
		}
		writeSyncer = zapcore.NewMultiWriteSyncer(
			zapcore.AddSync(os.Stdout),
			zapcore.AddSync(file),
		)
	default: // stdout
		writeSyncer = zapcore.AddSync(os.Stdout)
	}

	// 创建 core
	core := zapcore.NewCore(encoder, writeSyncer, level)

	// 创建 logger
	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))

	return &AuditLogger{
		logger: logger,
		config: config,
		file:   file,
	}, nil
}

// LogOperation 记录操作日志
func (l *AuditLogger) LogOperation(log *OperationLog) error {
	if log == nil {
		return fmt.Errorf("操作日志不能为空")
	}

	l.mu.RLock()
	defer l.mu.RUnlock()

	// 确保时间戳存在
	if log.Timestamp.IsZero() {
		log.Timestamp = time.Now()
	}

	// 构建日志字段
	fields := []zap.Field{
		zap.Time("timestamp", log.Timestamp),
		zap.String("tool", log.Tool),
		zap.String("context", log.Context),
		zap.Bool("success", log.Success),
		zap.Duration("duration", log.Duration),
	}

	// 添加可选字段
	if log.User != nil {
		fields = append(fields,
			zap.String("user_id", log.User.ID),
			zap.String("user_name", log.User.Name),
		)
		if log.User.Role != "" {
			fields = append(fields, zap.String("user_role", log.User.Role))
		}
	}

	if log.Namespace != "" {
		fields = append(fields, zap.String("namespace", log.Namespace))
	}

	if log.ResourceType != "" {
		fields = append(fields, zap.String("resource_type", log.ResourceType))
	}

	if log.ResourceName != "" {
		fields = append(fields, zap.String("resource_name", log.ResourceName))
	}

	if len(log.Arguments) > 0 {
		argsJSON, _ := json.Marshal(log.Arguments)
		fields = append(fields, zap.String("arguments", string(argsJSON)))
	}

	// 根据成功/失败记录不同级别的日志
	if log.Success {
		l.logger.Info("操作成功", fields...)
	} else {
		if log.Error != "" {
			fields = append(fields, zap.String("error", log.Error))
		}
		l.logger.Error("操作失败", fields...)
	}

	return nil
}

// LogError 记录错误日志
func (l *AuditLogger) LogError(err error, context string) error {
	if err == nil {
		return fmt.Errorf("错误对象不能为空")
	}

	l.mu.RLock()
	defer l.mu.RUnlock()

	l.logger.Error("系统错误",
		zap.Time("timestamp", time.Now()),
		zap.String("context", context),
		zap.Error(err),
	)

	return nil
}

// LogMetrics 记录性能指标
func (l *AuditLogger) LogMetrics(metrics *Metrics) error {
	if metrics == nil {
		return fmt.Errorf("性能指标不能为空")
	}

	l.mu.RLock()
	defer l.mu.RUnlock()

	// 确保时间戳存在
	if metrics.Timestamp.IsZero() {
		metrics.Timestamp = time.Now()
	}

	fields := []zap.Field{
		zap.Time("timestamp", metrics.Timestamp),
		zap.String("operation", metrics.Operation),
		zap.Duration("duration", metrics.Duration),
		zap.Bool("success", metrics.Success),
	}

	if metrics.ConcurrentCount > 0 {
		fields = append(fields, zap.Int("concurrent_count", metrics.ConcurrentCount))
	}

	l.logger.Info("性能指标", fields...)

	return nil
}

// Close 关闭日志器
func (l *AuditLogger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	var errs []error

	// 先同步日志
	if l.logger != nil {
		if err := l.logger.Sync(); err != nil {
			// 忽略 stdout/stderr 的 sync 错误（在某些系统上会返回错误）
			if l.config.Output != "stdout" {
				errs = append(errs, err)
			}
		}
	}

	// 关闭文件句柄
	if l.file != nil {
		if err := l.file.Close(); err != nil {
			errs = append(errs, err)
		}
		l.file = nil
	}

	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}

// SetLevel 动态设置日志级别
func (l *AuditLogger) SetLevel(level string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	switch level {
	case "debug", "info", "warn", "error":
		l.config.Level = level
		// 注意：zap 不支持动态修改级别，需要重新创建 logger
		// 这里只更新配置，实际应用中可能需要重新创建 logger
		return nil
	default:
		return fmt.Errorf("无效的日志级别: %s", level)
	}
}

// GetConfig 获取当前配置
func (l *AuditLogger) GetConfig() *LoggerConfig {
	l.mu.RLock()
	defer l.mu.RUnlock()

	// 返回配置的副本
	configCopy := *l.config
	return &configCopy
}
