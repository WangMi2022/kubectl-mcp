package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"kubectl-mcp/internal/audit"
	"kubectl-mcp/internal/config"
	"kubectl-mcp/internal/k8s"
	"kubectl-mcp/internal/server"
	"kubectl-mcp/internal/tools"
)

const (
	// 服务器版本
	Version = "1.3.0"

	// 构建时间（可通过 -ldflags 注入）
	BuildTime = "2026-01-30"

	// 优雅关闭超时时间
	ShutdownTimeout = 30 * time.Second
)

// Application 应用程序结构，管理所有组件的生命周期
type Application struct {
	config      *config.ServerConfig
	auditLogger *audit.AuditLogger
	k8sManager  *k8s.K8SClientManager
	httpServer  *server.HTTPServer
	metrics     *audit.MetricsCollector
}

func main() {
	// 创建应用程序实例
	app := &Application{}

	// 初始化应用程序
	if err := app.Initialize(); err != nil {
		fmt.Fprintf(os.Stderr, "初始化失败: %v\n", err)
		os.Exit(1)
	}

	// 启动服务器
	if err := app.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "服务器运行失败: %v\n", err)
		os.Exit(1)
	}
}

// Initialize 初始化应用程序的所有组件
func (app *Application) Initialize() error {
	var err error

	// 1. 加载配置
	fmt.Println("正在加载配置...")
	app.config, err = config.LoadConfig()
	if err != nil {
		return fmt.Errorf("加载配置失败: %w", err)
	}
	fmt.Printf("配置加载成功: 监听地址 %s\n", app.config.GetListenAddress())

	// 2. 初始化审计日志
	fmt.Println("正在初始化审计日志...")
	app.auditLogger, err = app.initAuditLogger()
	if err != nil {
		return fmt.Errorf("初始化审计日志失败: %w", err)
	}
	fmt.Println("审计日志初始化成功")

	// 3. 初始化性能指标收集器
	fmt.Println("正在初始化性能指标收集器...")
	app.metrics = audit.NewMetricsCollector()
	fmt.Println("性能指标收集器初始化成功")

	// 4. 初始化 K8S 客户端管理器
	fmt.Println("正在初始化 Kubernetes 客户端...")
	app.k8sManager, err = k8s.NewK8SClientManager(app.config.KubeconfigPath)
	if err != nil {
		return fmt.Errorf("初始化 K8S 客户端失败: %w", err)
	}
	fmt.Printf("K8S 客户端初始化成功: 当前 context = %s\n", app.k8sManager.GetCurrentContext())

	// 预热 Service 索引器并打印统计信息
	fmt.Println("正在预热 Service 索引器...")
	indexer, err := app.k8sManager.GetServiceIndexer()
	if err != nil {
		fmt.Printf("警告: Service 索引器初始化失败: %v\n", err)
	} else {
		stats := indexer.GetStats()
		fmt.Printf("Service 索引器就绪: %d 个 Service, %d 个 NodePort\n", stats.ServiceCount, stats.NodePortCount)
	}

	// 5. 初始化工具注册表并注册所有工具
	fmt.Println("正在注册工具...")
	toolRegistry, err := app.initToolRegistry()
	if err != nil {
		return fmt.Errorf("初始化工具注册表失败: %w", err)
	}
	fmt.Printf("工具注册成功: 共 %d 个工具\n", toolRegistry.ToolCount())

	// 6. 初始化 HTTP 服务器
	fmt.Println("正在初始化 HTTP 服务器...")
	app.httpServer, err = server.NewHTTPServer(&server.HTTPServerConfig{
		Config:       app.config,
		ToolRegistry: toolRegistry,
		K8SManager:   app.k8sManager,
		AuditLogger:  app.auditLogger,
		Version:      Version,
		Metrics:      app.metrics,
	})
	if err != nil {
		return fmt.Errorf("初始化 HTTP 服务器失败: %w", err)
	}
	fmt.Println("HTTP 服务器初始化成功")

	return nil
}

// initAuditLogger 初始化审计日志器
func (app *Application) initAuditLogger() (*audit.AuditLogger, error) {
	loggerConfig := &audit.LoggerConfig{
		Level:      app.config.LogLevel,
		Format:     app.config.LogFormat,
		MaxSize:    100,
		MaxBackups: 3,
		MaxAge:     28,
		Compress:   true,
	}

	// 根据配置决定输出目标
	if app.config.LogFile != "" {
		loggerConfig.Output = "both"
		loggerConfig.FilePath = app.config.LogFile
	} else {
		loggerConfig.Output = "stdout"
	}

	return audit.NewAuditLogger(loggerConfig)
}

// initToolRegistry 初始化工具注册表并注册所有工具
func (app *Application) initToolRegistry() (*tools.ToolRegistry, error) {
	registry := tools.NewToolRegistry()

	// 注册查询类工具
	if err := tools.RegisterQueryTools(registry); err != nil {
		return nil, fmt.Errorf("注册查询类工具失败: %w", err)
	}

	// 注册创建类工具
	if err := tools.RegisterCreateTools(registry); err != nil {
		return nil, fmt.Errorf("注册创建类工具失败: %w", err)
	}

	// 注册修改类工具
	if err := tools.RegisterUpdateTools(registry); err != nil {
		return nil, fmt.Errorf("注册修改类工具失败: %w", err)
	}

	// 注册删除类工具
	if err := tools.RegisterDeleteTools(registry); err != nil {
		return nil, fmt.Errorf("注册删除类工具失败: %w", err)
	}

	// 注册巡检类工具
	if err := tools.RegisterInspectTools(registry); err != nil {
		return nil, fmt.Errorf("注册巡检类工具失败: %w", err)
	}

	return registry, nil
}

// Run 启动服务器并等待关闭信号
func (app *Application) Run() error {
	// 创建用于接收关闭信号的 channel
	quit := make(chan os.Signal, 1)
	// 监听 SIGINT (Ctrl+C) 和 SIGTERM 信号
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// 创建用于接收服务器错误的 channel
	serverErr := make(chan error, 1)

	// 异步启动 HTTP 服务器
	go func() {
		fmt.Printf("\n========================================\n")
		fmt.Printf("  kubectl-mcp Server v%s\n", Version)
		fmt.Printf("  Build: %s\n", BuildTime)
		fmt.Printf("========================================\n")
		fmt.Printf("监听地址: %s\n", app.config.GetListenAddress())
		fmt.Printf("Kubeconfig: %s\n", app.config.KubeconfigPath)
		fmt.Printf("当前 Context: %s\n", app.k8sManager.GetCurrentContext())
		fmt.Printf("集群地址: %s\n", app.k8sManager.GetClusterServer())
		fmt.Printf("日志级别: %s\n", app.config.LogLevel)
		fmt.Printf("========================================\n")
		fmt.Println("服务器已启动，按 Ctrl+C 停止...")
		fmt.Printf("========================================\n\n")

		if err := app.httpServer.Start(); err != nil {
			serverErr <- err
		}
	}()

	// 等待关闭信号或服务器错误
	select {
	case sig := <-quit:
		fmt.Printf("\n收到信号 %v，正在优雅关闭服务器...\n", sig)
	case err := <-serverErr:
		return fmt.Errorf("服务器异常退出: %w", err)
	}

	// 执行优雅关闭
	return app.Shutdown()
}

// Shutdown 优雅关闭所有组件
func (app *Application) Shutdown() error {
	fmt.Println("正在关闭服务器...")

	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), ShutdownTimeout)
	defer cancel()

	var shutdownErrors []error

	// 1. 关闭 HTTP 服务器（等待正在处理的请求完成）
	if app.httpServer != nil {
		fmt.Println("正在关闭 HTTP 服务器...")
		if err := app.httpServer.Shutdown(ctx); err != nil {
			shutdownErrors = append(shutdownErrors, fmt.Errorf("关闭 HTTP 服务器失败: %w", err))
		} else {
			fmt.Println("HTTP 服务器已关闭")
		}
	}

	// 2. 关闭 K8S 客户端管理器
	if app.k8sManager != nil {
		fmt.Println("正在关闭 K8S 客户端...")
		if err := app.k8sManager.Close(); err != nil {
			shutdownErrors = append(shutdownErrors, fmt.Errorf("关闭 K8S 客户端失败: %w", err))
		} else {
			fmt.Println("K8S 客户端已关闭")
		}
	}

	// 3. 关闭审计日志器
	if app.auditLogger != nil {
		fmt.Println("正在关闭审计日志...")
		if err := app.auditLogger.Close(); err != nil {
			shutdownErrors = append(shutdownErrors, fmt.Errorf("关闭审计日志失败: %w", err))
		} else {
			fmt.Println("审计日志已关闭")
		}
	}

	// 检查是否有错误
	if len(shutdownErrors) > 0 {
		fmt.Println("\n关闭过程中出现以下错误:")
		for _, err := range shutdownErrors {
			fmt.Printf("  - %v\n", err)
		}
		return shutdownErrors[0]
	}

	fmt.Println("\n服务器已完全关闭")
	return nil
}
