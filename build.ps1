#!/usr/bin/env pwsh
<#
.SYNOPSIS
    kubectl-mcp 编译脚本
.DESCRIPTION
    支持编译 Windows 和 Linux 版本
.PARAMETER Platform
    目标平台: windows, linux, all (默认: all)
.PARAMETER Version
    版本号 (默认: 从 main.go 读取)
.EXAMPLE
    .\build.ps1 -Platform windows
    .\build.ps1 -Platform linux
    .\build.ps1 -Platform all
#>

param(
    [Parameter()]
    [ValidateSet("windows", "linux", "all")]
    [string]$Platform = "all",
    
    [Parameter()]
    [string]$Version = "1.3.0"
)

# 设置错误处理
$ErrorActionPreference = "Stop"

# 获取构建日期
$BuildTime = Get-Date -Format "yyyy-MM-dd"

# 构建标志
$LdFlags = "-X main.Version=$Version -X main.BuildTime=$BuildTime"

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  kubectl-mcp 编译脚本" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "版本号: $Version" -ForegroundColor Green
Write-Host "构建日期: $BuildTime" -ForegroundColor Green
Write-Host "目标平台: $Platform" -ForegroundColor Green
Write-Host "========================================`n" -ForegroundColor Cyan

# 编译 Windows 版本
function Build-Windows {
    Write-Host "正在编译 Windows 版本..." -ForegroundColor Yellow
    $env:GOOS = "windows"
    $env:GOARCH = "amd64"
    
    go build -o kubectl-mcp.exe -ldflags $LdFlags ./cmd/server
    
    if ($LASTEXITCODE -eq 0 -and (Test-Path kubectl-mcp.exe)) {
        $size = [math]::Round((Get-Item kubectl-mcp.exe).Length / 1MB, 2)
        Write-Host "✅ Windows 版本编译成功: kubectl-mcp.exe ($size MB)" -ForegroundColor Green
    } else {
        Write-Host "❌ Windows 版本编译失败" -ForegroundColor Red
        exit 1
    }
}

# 编译 Linux 版本
function Build-Linux {
    Write-Host "正在编译 Linux 版本..." -ForegroundColor Yellow
    $env:GOOS = "linux"
    $env:GOARCH = "amd64"
    
    go build -o kubectl-mcp-linux-amd64 -ldflags $LdFlags ./cmd/server
    
    if ($LASTEXITCODE -eq 0 -and (Test-Path kubectl-mcp-linux-amd64)) {
        $size = [math]::Round((Get-Item kubectl-mcp-linux-amd64).Length / 1MB, 2)
        Write-Host "✅ Linux 版本编译成功: kubectl-mcp-linux-amd64 ($size MB)" -ForegroundColor Green
    } else {
        Write-Host "❌ Linux 版本编译失败" -ForegroundColor Red
        exit 1
    }
}

# 根据参数执行编译
try {
    switch ($Platform) {
        "windows" {
            Build-Windows
        }
        "linux" {
            Build-Linux
        }
        "all" {
            Build-Windows
            Write-Host ""
            Build-Linux
        }
    }
    
    Write-Host "`n========================================" -ForegroundColor Cyan
    Write-Host "  编译完成！" -ForegroundColor Green
    Write-Host "========================================" -ForegroundColor Cyan
    
} catch {
    Write-Host "`n❌ 编译过程中发生错误: $_" -ForegroundColor Red
    exit 1
}
