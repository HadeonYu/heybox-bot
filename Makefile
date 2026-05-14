PROJ_NAME := heybox-bot

GO := go
PWD := $(shell pwd)
SRC_DIR := $(PWD)/src
BIN_DIR := $(PWD)/bin
TARGET := $(BIN_DIR)/$(PROJ_NAME)
SESSION := $(PROJ_NAME)

SOURCES  := $(shell find $(SRC_DIR) -type f -name '*.go')
MODFILES := $(SRC_DIR)/go.mod $(SRC_DIR)/go.sum

default: $(TARGET)

$(TARGET): $(SOURCES) $(MODFILES) | $(BIN_DIR)
	@mkdir -p $(@D)
	@echo "→ building $(PROJ_NAME)"
	@cd $(SRC_DIR) && $(GO) build -o $(TARGET) .
	@echo "→ finish building $(PROJ_NAME)"

$(BIN_DIR):
	@mkdir bin

run: $(TARGET)
	@$(TARGET)

# Run in detached screen session (requires 'screen')
run_detach: $(TARGET)
	@if ! command -v screen >/dev/null 2>&1; then \
		echo "ERROR: 'screen' is not installed. Please install 'screen' and retry."; \
		exit 1; \
	fi
	@if screen -list | grep -q "\.$(SESSION)\s"; then \
		echo "ERROR: screen session '$(SESSION)' already exists. Use 'make attach' or 'make exit' first."; \
		exit 1; \
	fi
	@echo "→ starting $(abspath $(TARGET)) in detached screen session '$(SESSION)'"
	@screen -dmS $(SESSION) $(TARGET)
	@echo "→ started. Use 'make attach' to attach, 'make exit' to stop."

# Attach to the running screen session
attach:
	@if ! command -v screen >/dev/null 2>&1; then \
		echo "ERROR: 'screen' is not installed."; \
		exit 1; \
	fi
	@if ! screen -list | grep -q "\.$(SESSION)\s"; then \
		echo "No screen session named '$(SESSION)' found."; \
		exit 1; \
	fi
	@screen -r $(SESSION)

# Quit the detached screen session
exit:
	@if ! command -v screen >/dev/null 2>&1; then \
		echo "ERROR: 'screen' is not installed."; \
		exit 1; \
	fi
	@if screen -list | grep -q "\.$(SESSION)\s"; then \
		echo "→ stopping screen session '$(SESSION)'"; \
		screen -S $(SESSION) -X quit; \
	else \
		echo "No running screen session '$(SESSION)'."; \
	fi

clean:
	@rm -rf $(BIN_DIR)

.PHONY: default run run_detach attach exit clean
