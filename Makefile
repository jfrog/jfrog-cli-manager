# Cleaned Makefile: only builds and installs the main jfvm binary and sets up the shim directory for PATH
.PHONY: build install uninstall clean bootstrap test build-release

JFVM_BIN := jfvm
SHIM_DIR := $(HOME)/.jfvm/shim

# Get version from git tag, fallback to dev
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_DATE := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

build:
	@echo "🔧 Building jfvm CLI..."
	go build -o $(JFVM_BIN) .

build-release:
	@echo "🔧 Building jfvm CLI with version $(VERSION)..."
	go build -ldflags "-X main.Version=$(VERSION) -X main.BuildDate=$(BUILD_DATE) -X main.GitCommit=$(GIT_COMMIT)" -o $(JFVM_BIN) .

install: build-release
	@echo "📂 Creating shim directory: $(SHIM_DIR)"
	mkdir -p $(SHIM_DIR)
	@echo "📥 Installing jfvm binary to $(SHIM_DIR)"
	cp $(JFVM_BIN) $(SHIM_DIR)/
	@echo "✅ Binary installed."

bootstrap: install
	@echo "🔁 Checking shell config for PATH..."
	@grep -q '.jfvm/shim' ~/.bashrc 2>/dev/null || echo 'export PATH="$$HOME/.jfvm/shim:$$PATH"' >> ~/.bashrc
	@grep -q '.jfvm/shim' ~/.zshrc 2>/dev/null || echo 'export PATH="$$HOME/.jfvm/shim:$$PATH"' >> ~/.zshrc
	@grep -q '.jfvm/shim' ~/.profile 2>/dev/null || echo 'export PATH="$$HOME/.jfvm/shim:$$PATH"' >> ~/.profile
	@echo "✅ PATH updated in shell config. Run 'source ~/.bashrc' or 'source ~/.zshrc' to apply."

test: build
	@echo "🧪 Running basic functionality tests..."
	@./$(JFVM_BIN) --help > /dev/null && echo "✅ jfvm help works"
	@./$(JFVM_BIN) list > /dev/null && echo "✅ jfvm list works"
	@./$(JFVM_BIN) history > /dev/null && echo "✅ jfvm history works"
	@echo "✅ All basic tests passed!"

uninstall:
	@echo "🗑️ Removing installed binaries..."
	rm -f $(SHIM_DIR)/$(JFVM_BIN)
	@echo "✅ Uninstalled."

clean:
	@echo "🧹 Cleaning build artifacts..."
	rm -f $(JFVM_BIN)

# E2E Testing
.PHONY: test-e2e test-e2e-local test-e2e-ci test-e2e-ubuntu test-e2e-macos

test-e2e: build
	@echo "Running E2E tests..."
	@chmod +x tests/e2e/run_tests.sh
	@./tests/e2e/run_tests.sh

test-e2e-local: build
	@echo "Running E2E tests locally..."
	@JFVM_PATH=$(PWD)/jfvm go test -v -timeout 10m ./tests/e2e/...

test-e2e-ci: build
	@echo "Running E2E tests for CI..."
	@chmod +x tests/e2e/run_ci_tests.sh
	@./tests/e2e/run_ci_tests.sh

test-e2e-ubuntu: build
	@echo "Running E2E tests on Ubuntu..."
	@JFVM_PATH=$(PWD)/jfvm TEST_FILTER="TestCoreVersionManagement|TestAliasManagement" go test -v -timeout 10m ./tests/e2e/...

test-e2e-macos: build
	@echo "Running E2E tests on macOS..."
	@JFVM_PATH=$(PWD)/jfvm TEST_FILTER="TestCoreVersionManagement|TestAliasManagement" go test -v -timeout 10m ./tests/e2e/...

# Test specific features
test-e2e-latest: build
	@echo "Testing 'latest' functionality..."
	@JFVM_PATH=$(PWD)/jfvm go test -v -timeout 5m ./tests/e2e/ -run TestCoreVersionManagement/Use_Latest_Version

test-e2e-alias: build
	@echo "Testing alias functionality..."
	@JFVM_PATH=$(PWD)/jfvm go test -v -timeout 5m ./tests/e2e/ -run TestAliasManagement

test-e2e-performance: build
	@echo "Testing performance..."
	@JFVM_PATH=$(PWD)/jfvm go test -v -timeout 10m ./tests/e2e/ -run TestPerformance

# Clean test artifacts
clean-tests:
	@echo "Cleaning test artifacts..."
	@rm -rf /tmp/jfvm-e2e-*
	@rm -f test-results-*.json
	@rm -rf coverage-reports/