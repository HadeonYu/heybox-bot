PROJ_NAME := heybox-bot
VERSION ?= v1.0.1

GO ?= go
ROOT_DIR := $(CURDIR)
SRC_DIR := $(ROOT_DIR)/src
BIN_DIR := $(ROOT_DIR)/bin
PACKAGE_DIR := package
PACKAGE_STAGING_DIR := $(PACKAGE_DIR)/staging
SESSION := $(PROJ_NAME)

# 本设备编译目标后缀
ifeq ($(OS),Windows_NT)
	NATIVE_EXT := .exe
else
	NATIVE_EXT :=
endif

# 创建目录封装
ifeq ($(OS),Windows_NT)
	MKDIR_P = if not exist "$(subst /,\,$@)" mkdir "$(subst /,\,$@)"
else
	MKDIR_P = mkdir -p "$@"
endif

TARGET_MAIN := $(PROJ_NAME)
TARGET_MAIN_DIR := $(SRC_DIR)/cmd/$(TARGET_MAIN)
GO_LDFLAGS := -ldflags "-X main.Version=$(VERSION)"

TARGET_NATIVE := $(BIN_DIR)/$(TARGET_MAIN)$(NATIVE_EXT)

# 交叉编译
LINUX_AMD64_DIR    := $(BIN_DIR)/linux-amd64
TARGET_MAIN_LINUX_AMD64 := $(LINUX_AMD64_DIR)/$(TARGET_MAIN)

WINDOWS_AMD64_DIR    := $(BIN_DIR)/windows-amd64
TARGET_MAIN_WINDOWS_AMD64 := $(WINDOWS_AMD64_DIR)/$(TARGET_MAIN).exe

MACOS_ARM64_DIR      := $(BIN_DIR)/macos-arm64
TARGET_MAIN_MACOS_ARM64   := $(MACOS_ARM64_DIR)/$(TARGET_MAIN)

PACKAGE_LINUX_AMD64   := $(PACKAGE_DIR)/$(PROJ_NAME)-linux-amd64.tar.gz
PACKAGE_WINDOWS_AMD64 := $(PACKAGE_DIR)/$(PROJ_NAME)-windows-amd64.zip
PACKAGE_MACOS_ARM64   := $(PACKAGE_DIR)/$(PROJ_NAME)-macos-arm64.tar.gz

default: build

# 编译宿主机平台可执行文件
build: | $(BIN_DIR)
	@echo "[INFO] building native $(PROJ_NAME)"
	@cd $(TARGET_MAIN_DIR) && $(GO) build $(GO_LDFLAGS) -o $(TARGET_NATIVE) .
	@echo "[INFO] finish building $(PROJ_NAME): $(TARGET_NATIVE)"

# 编译linux amd64可执行文件
linux-amd64: | $(LINUX_AMD64_DIR)
	@echo "[INFO] building $(PROJ_NAME) for linux/amd64"
	@cd $(TARGET_MAIN_DIR) && GOOS=linux GOARCH=amd64 $(GO) build $(GO_LDFLAGS) -o $(TARGET_MAIN_LINUX_AMD64) .
	@echo "[INFO] finish: $(TARGET_MAIN_LINUX_AMD64)"

# 编译windows amd64 可执行文件
windows-amd64: | $(WINDOWS_AMD64_DIR)
	@echo "[INFO] building $(PROJ_NAME) for windows/amd64"
	@cd $(TARGET_MAIN_DIR) && GOOS=windows GOARCH=amd64 $(GO) build $(GO_LDFLAGS) -o $(TARGET_MAIN_WINDOWS_AMD64) .
	@echo "[INFO] finish: $(TARGET_MAIN_WINDOWS_AMD64)"

# 编译 macOS Apple Silicon 可执行文件
macos-arm64: | $(MACOS_ARM64_DIR)
	@echo "[INFO] building $(PROJ_NAME) for darwin/arm64"
	@cd $(TARGET_MAIN_DIR) && GOOS=darwin GOARCH=arm64 $(GO) build $(GO_LDFLAGS) -o $(TARGET_MAIN_MACOS_ARM64) .
	@echo "[INFO] finish: $(TARGET_MAIN_MACOS_ARM64)"

# 打包 linux amd64 release
package-linux-amd64: linux-amd64 | $(PACKAGE_DIR) $(PACKAGE_STAGING_DIR)
	@echo "[INFO] packaging $(PROJ_NAME) for linux/amd64"
	@rm -rf $(PACKAGE_STAGING_DIR)/$(PROJ_NAME)-linux-amd64
	@mkdir -p $(PACKAGE_STAGING_DIR)/$(PROJ_NAME)-linux-amd64
	@install -m 755 $(TARGET_MAIN_LINUX_AMD64) $(PACKAGE_STAGING_DIR)/$(PROJ_NAME)-linux-amd64/$(PROJ_NAME)
	@install -m 644 $(ROOT_DIR)/system_prompt.md $(PACKAGE_STAGING_DIR)/$(PROJ_NAME)-linux-amd64/system_prompt.md
	@install -m 644 $(ROOT_DIR)/config-example.yaml $(PACKAGE_STAGING_DIR)/$(PROJ_NAME)-linux-amd64/config.yaml
	@tar -C $(PACKAGE_STAGING_DIR) -czf $(PACKAGE_LINUX_AMD64) $(PROJ_NAME)-linux-amd64
	@echo "[INFO] packaged: $(PACKAGE_LINUX_AMD64)"

# 打包 windows amd64 release
package-windows-amd64: windows-amd64 | $(PACKAGE_DIR) $(PACKAGE_STAGING_DIR)
	@if ! command -v zip >/dev/null 2>&1; then \
		echo "ERROR: 'zip' is not installed. Please install 'zip' and retry."; \
		exit 1; \
	fi
	@echo "[INFO] packaging $(PROJ_NAME) for windows/amd64"
	@rm -rf $(PACKAGE_STAGING_DIR)/$(PROJ_NAME)-windows-amd64
	@mkdir -p $(PACKAGE_STAGING_DIR)/$(PROJ_NAME)-windows-amd64
	@install -m 755 $(TARGET_MAIN_WINDOWS_AMD64) $(PACKAGE_STAGING_DIR)/$(PROJ_NAME)-windows-amd64/$(PROJ_NAME).exe
	@install -m 644 $(ROOT_DIR)/system_prompt.md $(PACKAGE_STAGING_DIR)/$(PROJ_NAME)-windows-amd64/system_prompt.md
	@install -m 644 $(ROOT_DIR)/config-example.yaml $(PACKAGE_STAGING_DIR)/$(PROJ_NAME)-windows-amd64/config.yaml
	@cd $(PACKAGE_STAGING_DIR) && zip -qr $(abspath $(PACKAGE_WINDOWS_AMD64)) $(PROJ_NAME)-windows-amd64
	@echo "[INFO] packaged: $(PACKAGE_WINDOWS_AMD64)"

# 打包 macOS Apple Silicon release
package-macos-arm64: macos-arm64 | $(PACKAGE_DIR) $(PACKAGE_STAGING_DIR)
	@echo "[INFO] packaging $(PROJ_NAME) for darwin/arm64"
	@rm -rf $(PACKAGE_STAGING_DIR)/$(PROJ_NAME)-macos-arm64
	@mkdir -p $(PACKAGE_STAGING_DIR)/$(PROJ_NAME)-macos-arm64
	@install -m 755 $(TARGET_MAIN_MACOS_ARM64) $(PACKAGE_STAGING_DIR)/$(PROJ_NAME)-macos-arm64/$(PROJ_NAME)
	@install -m 644 $(ROOT_DIR)/system_prompt.md $(PACKAGE_STAGING_DIR)/$(PROJ_NAME)-macos-arm64/system_prompt.md
	@install -m 644 $(ROOT_DIR)/config-example.yaml $(PACKAGE_STAGING_DIR)/$(PROJ_NAME)-macos-arm64/config.yaml
	@tar -C $(PACKAGE_STAGING_DIR) -czf $(PACKAGE_MACOS_ARM64) $(PROJ_NAME)-macos-arm64
	@echo "[INFO] packaged: $(PACKAGE_MACOS_ARM64)"

# 编译、打包 release
ifeq ($(OS),Windows_NT)
release:
	@echo "暂不支持在windows系统上打包release"
	@exit 1
else
release: package-linux-amd64 package-windows-amd64 package-macos-arm64
endif

$(BIN_DIR) $(LINUX_AMD64_DIR) $(WINDOWS_AMD64_DIR) $(MACOS_ARM64_DIR) $(PACKAGE_DIR) $(PACKAGE_STAGING_DIR):
	$(MKDIR_P)

run: build
	@$(TARGET_NATIVE)

ifeq ($(OS),Windows_NT)
run_detach attach exit:
	@echo "ERROR: '$@' is only supported on Linux/macOS."
	@exit 1
else

# 用screen后台运行，不支持windows
run_detach: build
	@if ! command -v screen >/dev/null 2>&1; then \
		echo "ERROR: 'screen' is not installed. Please install 'screen' and retry."; \
		exit 1; \
	fi
	@if screen -list | grep -q "[.]$(SESSION)[[:space:]]"; then \
		echo "ERROR: screen session '$(SESSION)' already exists. Use 'make attach' or 'make exit' first."; \
		exit 1; \
	fi
	@echo "[INFO] starting $(abspath $(TARGET_NATIVE)) in detached screen session '$(SESSION)'"
	@screen -dmS $(SESSION) $(TARGET_NATIVE)
	@echo "[INFO] started. Use 'make attach' to attach, 'make exit' to stop."

# 进入screen的终端
attach:
	@if ! command -v screen >/dev/null 2>&1; then \
		echo "ERROR: 'screen' is not installed."; \
		exit 1; \
	fi
	@if ! screen -list | grep -q "[.]$(SESSION)[[:space:]]"; then \
		echo "No screen session named '$(SESSION)' found."; \
		exit 1; \
	fi
	@screen -r $(SESSION)

# 退出后台运行
exit:
	@if ! command -v screen >/dev/null 2>&1; then \
		echo "ERROR: 'screen' is not installed."; \
		exit 1; \
	fi
	@if screen -list | grep -q "[.]$(SESSION)[[:space:]]"; then \
		echo "[INFO] stopping screen session '$(SESSION)'"; \
		screen -S $(SESSION) -X quit; \
	else \
		echo "No running screen session '$(SESSION)'."; \
	fi

endif # ifeq ($(OS),Windows_NT)

clean:
	@rm -rf $(BIN_DIR)
	@rm -rf $(PACKAGE_DIR)

.PHONY: default build package release \
	linux-amd64 windows-amd64 macos-arm64 \
	package-linux-amd64 package-windows-amd64 package-macos-arm64 \
	run run_detach attach exit clean
