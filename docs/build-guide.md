# 构建指南

本文档详细说明如何编译、打包和分发 Claude Switch。

## 1. 环境准备

### 1.1 必需依赖

| 工具 | 版本 | 说明 |
|------|------|------|
| Go | 1.21+ | 主要编程语言 |
| Make | 任意 | 构建自动化 |
| Git | 任意 | 版本控制 |

### 1.2 可选依赖（打包用）

| 工具 | 用途 |
|------|------|
| fyne | Fyne 官方打包工具 |
| upx | 二进制压缩 |
| osslsigncode | Windows 签名 |

### 1.3 安装 Fyne 工具

```bash
go install fyne.io/fyne/v2/cmd/fyne@latest
```

## 2. 开发构建

### 2.1 快速开始

```bash
# 克隆项目
git clone https://github.com/yourusername/claude-switch.git
cd claude-switch

# 下载依赖
go mod download

# 构建 TUI 版本
make tui

# 构建 GUI 版本
make gui
```

### 2.2 开发模式运行

```bash
# TUI 开发
make dev-tui

# GUI 开发
make dev-gui
```

### 2.3 运行测试

```bash
# 全部测试
go test ./...

# 带覆盖率
go test -cover ./...

# 特定包测试
go test ./internal/core/...
```

## 3. 平台特定构建

### 3.1 Windows

#### 本地构建（Windows 环境）

```bash
# 开发版本（带控制台窗口，方便调试）
go build -o bin/claude-switch-gui.exe ./cmd/claude-switch-gui

# 发布版本（GUI 模式，无控制台窗口）
go build -ldflags "-H=windowsgui" -o bin/claude-switch-gui.exe ./cmd/claude-switch-gui
```

#### 交叉编译（Linux/macOS 构建 Windows）

```bash
# 安装 mingw（macOS）
brew install mingw-w64

# 交叉编译
GOOS=windows GOARCH=amd64 CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc \
    go build -ldflags "-H=windowsgui" -o bin/claude-switch-gui.exe ./cmd/claude-switch-gui

# 使用 Makefile
make release-windows
```

#### DPI 感知（高分辨率屏幕）

创建 `build/windows/manifest.xml`：

```xml
<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<assembly xmlns="urn:schemas-microsoft-com:asm.v1" manifestVersion="1.0">
    <application xmlns="urn:schemas-microsoft-com:asm.v3">
        <windowsSettings>
            <dpiAware xmlns="http://schemas.microsoft.com/SMI/2005/WindowsSettings">true/pm</dpiAware>
            <dpiAwareness xmlns="http://schemas.microsoft.com/SMI/2016/WindowsSettings">permonitorv2,permonitor,system</dpiAwareness>
        </windowsSettings>
    </application>
</assembly>
```

使用 `fyne package` 自动嵌入 manifest。

### 3.2 macOS

#### 本地构建

```bash
# Intel Mac
GOOS=darwin GOARCH=amd64 go build -o bin/claude-switch-gui ./cmd/claude-switch-gui

# Apple Silicon
GOOS=darwin GOARCH=arm64 go build -o bin/claude-switch-gui ./cmd/claude-switch-gui

# Universal Binary（Intel + Apple Silicon）
GOOS=darwin GOARCH=amd64 go build -o bin/claude-switch-gui-amd64 ./cmd/claude-switch-gui
GOOS=darwin GOARCH=arm64 go build -o bin/claude-switch-gui-arm64 ./cmd/claude-switch-gui
lipo -create -output bin/claude-switch-gui bin/claude-switch-gui-amd64 bin/claude-switch-gui-arm64
rm bin/claude-switch-gui-amd64 bin/claude-switch-gui-arm64
```

#### 打包为 App Bundle

```bash
# 使用 fyne package
fyne package -os darwin \
    -name "Claude Switch" \
    -appID com.claude-switch.gui \
    -icon assets/icon.png \
    -sourceDir cmd/claude-switch-gui

# 输出: Claude Switch.app
```

#### 签名（可选）

```bash
# 使用开发者证书签名
codesign --deep --force --verify --verbose \
    --sign "Developer ID Application: Your Name" \
    "Claude Switch.app"

# 公证（可选，需要 Apple Developer 账号）
xcrun altool --notarize-app \
    --primary-bundle-id com.claude-switch.gui \
    --username "your@email.com" \
    --password "@keychain:AC_PASSWORD" \
    --file "Claude Switch.app"
```

### 3.3 Linux

```bash
# 构建
make release-linux

# 或使用 fyne（自动处理依赖）
fyne package -os linux -name "Claude Switch" -icon assets/icon.png

# 创建 deb 包（Debian/Ubuntu）
# 使用 dpkg-deb 或 nfpm
```

#### 创建 deb 包

```bash
# 安装 nfpm
go install github.com/goreleaser/nfpm/v2/cmd/nfpm@latest

# nfpm.yaml
nfpm pkg -t dist/claude-switch_1.0.0_amd64.deb -p deb
```

## 4. Makefile 完整配置

```makefile
# 变量
APP_NAME = claude-switch
GUI_NAME = claude-switch-gui
VERSION = 1.0.0
BUILD_TIME = $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_COMMIT = $(shell git rev-parse --short HEAD)

# 链接标志
LDFLAGS = -ldflags "-X main.Version=$(VERSION) \
    -X main.BuildTime=$(BUILD_TIME) \
    -X main.GitCommit=$(GIT_COMMIT) \
    -s -w"

# 默认目标
.PHONY: all
all: tui gui

# ========== 开发构建 ==========
.PHONY: tui
tui:
	go build $(LDFLAGS) -o bin/$(APP_NAME) ./cmd/$(APP_NAME)

.PHONY: gui
gui:
	go build $(LDFLAGS) -o bin/$(GUI_NAME) ./cmd/$(GUI_NAME)

.PHONY: dev-tui
dev-tui:
	go run ./cmd/$(APP_NAME)

.PHONY: dev-gui
dev-gui:
	go run ./cmd/$(GUI_NAME)

# ========== 测试 ==========
.PHONY: test
test:
	go test -v ./...

.PHONY: test-coverage
test-coverage:
	go test -cover -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# ========== 代码质量 ==========
.PHONY: lint
lint:
	golangci-lint run

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: vet
vet:
	go vet ./...

# ========== 清理 ==========
.PHONY: clean
clean:
	rm -rf bin/ dist/ coverage.out coverage.html

# ========== Windows ==========
.PHONY: release-windows
release-windows:
	mkdir -p dist/windows
	# TUI
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) \
		-o dist/windows/$(APP_NAME).exe ./cmd/$(APP_NAME)
	# GUI
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -H=windowsgui \
		-o dist/windows/$(GUI_NAME).exe ./cmd/$(GUI_NAME)

.PHONY: package-windows
package-windows: release-windows
	# 使用 fyne package
	fyne package -os windows -name "Claude Switch" \
		-icon assets/icon.png -appID com.claude-switch.gui \
		-sourceDir cmd/claude-switch-gui -output dist/windows/

# ========== macOS ==========
.PHONY: release-macos
release-macos:
	mkdir -p dist/macos
	# AMD64
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) \
		-o dist/macos/$(GUI_NAME)-amd64 ./cmd/$(GUI_NAME)
	# ARM64
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) \
		-o dist/macos/$(GUI_NAME)-arm64 ./cmd/$(GUI_NAME)
	# Universal Binary
	lipo -create -output dist/macos/$(GUI_NAME) \
		dist/macos/$(GUI_NAME)-amd64 dist/macos/$(GUI_NAME)-arm64
	rm dist/macos/$(GUI_NAME)-amd64 dist/macos/$(GUI_NAME)-arm64

.PHONY: package-macos
package-macos:
	fyne package -os darwin -name "Claude Switch" \
		-icon assets/icon.png -appID com.claude-switch.gui \
		-sourceDir cmd/claude-switch-gui

# ========== Linux ==========
.PHONY: release-linux
release-linux:
	mkdir -p dist/linux
	go build $(LDFLAGS) -o dist/linux/$(GUI_NAME) ./cmd/$(GUI_NAME)

# ========== 全平台 ==========
.PHONY: release-all
release-all: clean release-windows release-macos release-linux

# ========== 压缩 ==========
.PHONY: compress
compress:
	# 使用 upx 压缩（可选）
	upx dist/windows/*.exe || true
	upx dist/linux/$(GUI_NAME) || true

# ========== 检查 ==========
.PHONY: check
check: fmt vet lint test
```

## 5. 资源文件

### 5.1 图标要求

| 平台 | 格式 | 推荐尺寸 |
|------|------|----------|
| Windows | .ico | 256x256, 128x128, 64x64, 32x32, 16x16 |
| macOS | .icns | 1024x1024, 512x512, 256x256, 128x128, 64x64, 32x32, 16x16 |
| Linux | .png | 256x256, 128x128, 64x64, 48x48, 32x32, 16x16 |

### 5.2 生成图标

```bash
# macOS: png 转 icns
mkdir -p icon.iconset
sips -z 16 16     icon.png --out icon.iconset/icon_16x16.png
sips -z 32 32     icon.png --out icon.iconset/icon_16x16@2x.png
sips -z 32 32     icon.png --out icon.iconset/icon_32x32.png
sips -z 64 64     icon.png --out icon.iconset/icon_32x32@2x.png
sips -z 128 128   icon.png --out icon.iconset/icon_128x128.png
sips -z 256 256   icon.png --out icon.iconset/icon_128x128@2x.png
sips -z 256 256   icon.png --out icon.iconset/icon_256x256.png
sips -z 512 512   icon.png --out icon.iconset/icon_256x256@2x.png
sips -z 512 512   icon.png --out icon.iconset/icon_512x512.png
sips -z 1024 1024 icon.png --out icon.iconset/icon_512x512@2x.png
iconutil -c icns icon.iconset -o assets/icon.icns
rm -rf icon.iconset

# Windows: png 转 ico
# 使用 ImageMagick 或在线工具
convert icon.png -define icon:auto-resize=256,128,64,48,32,16 assets/icon.ico
```

### 5.3 嵌入资源

```bash
# 生成资源 Go 文件
fyne bundle -name resourceIcon -package main -o cmd/claude-switch-gui/icon.go assets/icon.png
```

## 6. 发布检查清单

### 6.1 功能测试

- [ ] TUI 版本所有功能正常
- [ ] GUI 版本所有功能正常
- [ ] Profile CRUD 正常
- [ ] Run 功能正常
- [ ] 错误提示正常

### 6.2 平台测试

- [ ] Windows 10/11 测试通过
- [ ] macOS Intel 测试通过
- [ ] macOS Apple Silicon 测试通过
- [ ] Linux (Ubuntu/Fedora) 测试通过

### 6.3 发布文件

- [ ] claude-switch (TUI, 全平台)
- [ ] claude-switch-gui.exe (Windows)
- [ ] Claude Switch.app (macOS)
- [ ] claude-switch-gui (Linux)
- [ ] README.md
- [ ] CHANGELOG.md

## 7. 常见问题

### Q: Windows 下 GUI 版本仍有控制台窗口？

A: 确保使用 `-ldflags "-H=windowsgui"`

### Q: macOS 提示 "无法打开，因为无法验证开发者"？

A: 右键点击 → 打开 → 仍要打开。或签名后分发。

### Q: Linux 下缺少 libGL.so？

A: 安装显卡驱动或 mesa：`sudo apt install mesa-utils`

### Q: 二进制文件太大？

A: 使用 `-ldflags "-s -w"` 去掉符号表，或使用 upx 压缩。
