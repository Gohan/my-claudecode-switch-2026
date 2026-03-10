#!/usr/bin/env pwsh
# Claude Switch 构建脚本 (PowerShell)
# 用法: .\build.ps1 [windows|linux|darwin] [amd64|arm64] [version]

param(
    [string]$OS = "windows",
    [string]$Arch = "amd64",
    [string]$Version = "dev"
)

$OutputDir = "dist"
$BinaryName = "claude-switch"

# 颜色输出
function Write-Info($message) {
    Write-Host "[INFO] $message" -ForegroundColor Green
}

function Write-Warn($message) {
    Write-Host "[WARN] $message" -ForegroundColor Yellow
}

function Write-Error($message) {
    Write-Host "[ERROR] $message" -ForegroundColor Red
}

# 检查 Go 是否安装
if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    Write-Error "Go not found. Please install Go: https://golang.org/dl/"
    exit 1
}

Write-Info "Using Go: $(go version)"

# 创建输出目录
New-Item -ItemType Directory -Force -Path $OutputDir | Out-Null

# 根据 OS 设置二进制文件名
if ($OS -eq "windows") {
    $BinaryName = "claude-switch.exe"
}

$OutputPath = "$OutputDir/${OS}_${Arch}/$BinaryName"

Write-Info "Building for $OS/$Arch..."

# 构建参数
$BuildTime = (Get-Date -Format "yyyy-MM-ddTHH:mm:ssZ")
$LdFlags = "-s -w -X main.Version=$Version -X main.BuildTime=$BuildTime"

# 设置环境变量并执行编译
$env:CGO_ENABLED = "0"
$env:GOOS = $OS
$env:GOARCH = $Arch

try {
    go build -ldflags "$LdFlags" -o "$OutputPath" .

    if ($LASTEXITCODE -eq 0) {
        Write-Info "Build successful: $OutputPath"
        $fileInfo = Get-Item $OutputPath
        $sizeInMB = [math]::Round($fileInfo.Length / 1MB, 2)
        Write-Info "Size: $sizeInMB MB"
    } else {
        Write-Error "Build failed!"
        exit 1
    }
} catch {
    Write-Error "Build failed: $_"
    exit 1
}

# 构建所有平台的函数
function Build-All {
    Write-Info "Building for all platforms..."

    $platforms = @(
        @{ GOOS = "linux"; GOARCH = "amd64"; Ext = "" },
        @{ GOOS = "linux"; GOARCH = "arm64"; Ext = "" },
        @{ GOOS = "windows"; GOARCH = "amd64"; Ext = ".exe" },
        @{ GOOS = "windows"; GOARCH = "arm64"; Ext = ".exe" },
        @{ GOOS = "darwin"; GOARCH = "amd64"; Ext = "" },
        @{ GOOS = "darwin"; GOARCH = "arm64"; Ext = "" }
    )

    foreach ($platform in $platforms) {
        $env:GOOS = $platform.GOOS
        $env:GOARCH = $platform.GOARCH
        $outputFile = "$OutputDir/$($platform.GOOS)_$($platform.GOARCH)/claude-switch$($platform.Ext)"

        Write-Info "Building $($platform.GOOS)/$($platform.GOARCH)..."
        go build -ldflags "$LdFlags" -o "$outputFile" .

        if ($LASTEXITCODE -eq 0) {
            Write-Info "  -> $outputFile"
        } else {
            Write-Error "Failed to build for $($platform.GOOS)/$($platform.GOARCH)"
        }
    }

    Write-Info "All builds complete!"
}

# 如果传入 "all" 作为 OS 参数，构建所有平台
if ($OS -eq "all") {
    Build-All
}
