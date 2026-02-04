package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// ServerConfig 定义服务器的完整配置结构
type ServerConfig struct {
	// HTTP 服务器配置
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`

	// Kubeconfig 配置
	KubeconfigPath string `mapstructure:"kubeconfigPath"`
	DefaultContext string `mapstructure:"defaultContext"`

	// 日志配置
	LogLevel  string `mapstructure:"logLevel"`
	LogFormat string `mapstructure:"logFormat"`
	LogFile   string `mapstructure:"logFile"`

	// 性能配置
	MaxConcurrentRequests int           `mapstructure:"maxConcurrentRequests"`
	RequestTimeout        time.Duration `mapstructure:"requestTimeout"`

	// 安全配置
	APIToken       string   `mapstructure:"apiToken"`
	AllowedOrigins []string `mapstructure:"allowedOrigins"`

	// 缓存配置
	EnableCache bool          `mapstructure:"enableCache"`
	CacheTTL    time.Duration `mapstructure:"cacheTTL"`
}

// LoadConfig 加载配置，优先级：命令行参数 > 环境变量 > 配置文件 > 默认值
func LoadConfig() (*ServerConfig, error) {
	// 设置默认值
	setDefaults()

	// 绑定命令行参数
	bindFlags()

	// 检查是否通过环境变量指定了配置文件
	configFile := os.Getenv("KUBECTL_MCP_CONFIG")
	if configFile != "" {
		// 使用指定的配置文件
		viper.SetConfigFile(configFile)
		if err := viper.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("读取配置文件失败: %w", err)
		}
	} else {
		// 设置配置文件搜索路径
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
		viper.AddConfigPath(".")
		viper.AddConfigPath("./config")
		viper.AddConfigPath("/etc/kubectl-mcp")

		// 读取配置文件（如果存在）
		if err := viper.ReadInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
				return nil, fmt.Errorf("读取配置文件失败: %w", err)
			}
			// 配置文件不存在不是错误，使用默认值和环境变量
		}
	}

	// 绑定环境变量
	viper.SetEnvPrefix("KUBECTL_MCP")
	viper.AutomaticEnv()

	// 显式绑定环境变量到配置键
	viper.BindEnv("host", "KUBECTL_MCP_HOST")
	viper.BindEnv("port", "KUBECTL_MCP_PORT")
	viper.BindEnv("kubeconfigPath", "KUBECTL_MCP_KUBECONFIGPATH")
	viper.BindEnv("defaultContext", "KUBECTL_MCP_DEFAULTCONTEXT")
	viper.BindEnv("logLevel", "KUBECTL_MCP_LOGLEVEL")
	viper.BindEnv("logFormat", "KUBECTL_MCP_LOGFORMAT")
	viper.BindEnv("logFile", "KUBECTL_MCP_LOGFILE")
	viper.BindEnv("maxConcurrentRequests", "KUBECTL_MCP_MAXCONCURRENTREQUESTS")
	viper.BindEnv("requestTimeout", "KUBECTL_MCP_REQUESTTIMEOUT")
	viper.BindEnv("apiToken", "KUBECTL_MCP_APITOKEN")
	viper.BindEnv("enableCache", "KUBECTL_MCP_ENABLECACHE")
	viper.BindEnv("cacheTTL", "KUBECTL_MCP_CACHETTL")

	// 解析配置到结构体
	var config ServerConfig
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("解析配置失败: %w", err)
	}

	// 验证配置
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("配置验证失败: %w", err)
	}

	return &config, nil
}

// setDefaults 设置默认配置值
func setDefaults() {
	// HTTP 服务器默认值
	viper.SetDefault("host", "0.0.0.0")
	viper.SetDefault("port", 8080)

	// Kubeconfig 默认值
	homeDir, err := os.UserHomeDir()
	if err == nil {
		viper.SetDefault("kubeconfigPath", filepath.Join(homeDir, ".kube", "config"))
	}
	viper.SetDefault("defaultContext", "")

	// 日志默认值
	viper.SetDefault("logLevel", "info")
	viper.SetDefault("logFormat", "json")
	viper.SetDefault("logFile", "")

	// 性能默认值
	viper.SetDefault("maxConcurrentRequests", 100)
	viper.SetDefault("requestTimeout", 30*time.Second)

	// 安全默认值
	viper.SetDefault("apiToken", "")
	viper.SetDefault("allowedOrigins", []string{"*"})

	// 缓存默认值
	viper.SetDefault("enableCache", true)
	viper.SetDefault("cacheTTL", 5*time.Minute)
}

var flagsInitialized bool

// bindFlags 绑定命令行参数
func bindFlags() {
	// 避免重复定义标志
	if flagsInitialized {
		return
	}
	flagsInitialized = true

	// HTTP 服务器参数
	pflag.String("host", "", "服务器监听地址")
	pflag.Int("port", 0, "服务器监听端口")

	// Kubeconfig 参数
	pflag.String("kubeconfig", "", "kubeconfig 文件路径")
	pflag.String("context", "", "默认使用的 Kubernetes context")

	// 日志参数
	pflag.String("log-level", "", "日志级别 (debug, info, warn, error)")
	pflag.String("log-format", "", "日志格式 (json, text)")
	pflag.String("log-file", "", "日志文件路径")

	// 性能参数
	pflag.Int("max-concurrent-requests", 0, "最大并发请求数")
	pflag.Duration("request-timeout", 0, "请求超时时间")

	// 安全参数
	pflag.String("api-token", "", "API 认证 Token")

	// 缓存参数
	pflag.Bool("enable-cache", true, "是否启用缓存")
	pflag.Duration("cache-ttl", 0, "缓存过期时间")

	pflag.Parse()

	// 绑定到 viper
	viper.BindPFlag("host", pflag.Lookup("host"))
	viper.BindPFlag("port", pflag.Lookup("port"))
	viper.BindPFlag("kubeconfigPath", pflag.Lookup("kubeconfig"))
	viper.BindPFlag("defaultContext", pflag.Lookup("context"))
	viper.BindPFlag("logLevel", pflag.Lookup("log-level"))
	viper.BindPFlag("logFormat", pflag.Lookup("log-format"))
	viper.BindPFlag("logFile", pflag.Lookup("log-file"))
	viper.BindPFlag("maxConcurrentRequests", pflag.Lookup("max-concurrent-requests"))
	viper.BindPFlag("requestTimeout", pflag.Lookup("request-timeout"))
	viper.BindPFlag("apiToken", pflag.Lookup("api-token"))
	viper.BindPFlag("enableCache", pflag.Lookup("enable-cache"))
	viper.BindPFlag("cacheTTL", pflag.Lookup("cache-ttl"))
}

// Validate 验证配置的有效性
func (c *ServerConfig) Validate() error {
	// 验证端口范围
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("端口号必须在 1-65535 范围内，当前值: %d", c.Port)
	}

	// 验证 kubeconfig 路径
	if c.KubeconfigPath == "" {
		return fmt.Errorf("kubeconfig 路径不能为空")
	}

	// 检查 kubeconfig 文件是否存在
	if _, err := os.Stat(c.KubeconfigPath); os.IsNotExist(err) {
		return fmt.Errorf("kubeconfig 文件不存在: %s", c.KubeconfigPath)
	}

	// 验证日志级别
	validLogLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLogLevels[c.LogLevel] {
		return fmt.Errorf("无效的日志级别: %s，有效值: debug, info, warn, error", c.LogLevel)
	}

	// 验证日志格式
	validLogFormats := map[string]bool{
		"json": true,
		"text": true,
	}
	if !validLogFormats[c.LogFormat] {
		return fmt.Errorf("无效的日志格式: %s，有效值: json, text", c.LogFormat)
	}

	// 验证性能配置
	if c.MaxConcurrentRequests < 1 {
		return fmt.Errorf("最大并发请求数必须大于 0，当前值: %d", c.MaxConcurrentRequests)
	}

	if c.RequestTimeout < time.Second {
		return fmt.Errorf("请求超时时间必须至少为 1 秒，当前值: %v", c.RequestTimeout)
	}

	// 验证缓存配置
	if c.EnableCache && c.CacheTTL < time.Second {
		return fmt.Errorf("缓存过期时间必须至少为 1 秒，当前值: %v", c.CacheTTL)
	}

	return nil
}

// GetKubeconfigPath 获取 kubeconfig 文件路径
func (c *ServerConfig) GetKubeconfigPath() string {
	return c.KubeconfigPath
}

// GetListenAddress 获取服务器监听地址
func (c *ServerConfig) GetListenAddress() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// IsDebugMode 判断是否为调试模式
func (c *ServerConfig) IsDebugMode() bool {
	return c.LogLevel == "debug"
}
