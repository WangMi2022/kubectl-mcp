# 编译指南

## 快速开始

### Windows 环境

使用 PowerShell 脚本：

```powershell
# 编译所有平台
.\build.ps1

# 只编译 Windows 版本
.\build.ps1 -Platform windows

# 只编译 Linux 版本
.\build.ps1 -Platform linux

# 指定版本号
.\build.ps1 -Version "1.3.0"
```

### Linux/macOS 环境

使用 Bash 脚本：

```bash
# 添加执行权限
chmod +x build.sh

# 编译所有平台
./build.sh all

# 只编译 Windows 版本
./build.sh windows

# 只编译 Linux 版本
./build.sh linux
```

## 手动编译

### Windows 版本

```powershell
$env:GOOS="windows"
$env:GOARCH="amd64"
go build -o kubectl-mcp.exe -ldflags "-X main.Version=1.3.0 -X main.BuildTime=$(Get-Date -Format 'yyyy-MM-dd')" ./cmd/server
```

### Linux 版本

```powershell
# 在 Windows 上交叉编译 Linux 版本
$env:GOOS="linux"
$env:GOARCH="amd64"
go build -o kubectl-mcp-linux-amd64 -ldflags "-X main.Version=1.3.0 -X main.BuildTime=$(Get-Date -Format 'yyyy-MM-dd')" ./cmd/server
```

或在 Linux/macOS 上：

```bash
GOOS=linux GOARCH=amd64 go build -o kubectl-mcp-linux-amd64 -ldflags "-X main.Version=1.3.0 -X main.BuildTime=$(date +%Y-%m-%d)" ./cmd/server
```

### macOS 版本

```bash
GOOS=darwin GOARCH=amd64 go build -o kubectl-mcp-darwin-amd64 -ldflags "-X main.Version=1.3.0 -X main.BuildTime=$(date +%Y-%m-%d)" ./cmd/server
```

## 编译产物

编译成功后会生成以下文件：

- `kubectl-mcp.exe` - Windows 版本 (~68 MB)
- `kubectl-mcp-linux-amd64` - Linux 版本 (~68 MB)

## Docker 编译

使用 Dockerfile 构建镜像：

```bash
docker build -t kubectl-mcp:1.3.0 .
```

## 验证编译结果

### Windows

```powershell
.\kubectl-mcp.exe
```

### Linux

```bash
chmod +x kubectl-mcp-linux-amd64
./kubectl-mcp-linux-amd64
```

## 支持的平台

| 操作系统 | 架构 | 文件名 |
|---------|------|--------|
| Windows | amd64 | kubectl-mcp.exe |
| Linux | amd64 | kubectl-mcp-linux-amd64 |
| macOS | amd64 | kubectl-mcp-darwin-amd64 |
| macOS | arm64 | kubectl-mcp-darwin-arm64 |

## 注意事项

1. 需要 Go 1.21 或更高版本
2. 编译产物已在 `.gitignore` 中配置，不会提交到版本控制
3. 交叉编译不需要安装目标平台的工具链
4. 编译时会自动注入版本号和构建日期
