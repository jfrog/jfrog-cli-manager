.PHONY: build install uninstall clean bootstrap test

JFVM_BIN := jfvm
SHIM_BIN := jf
SHIM_DIR := $(HOME)/.jfvm/shim

build:
	@echo "🔧 Building jfvm CLI..."
	go build -o $(JFVM_BIN) .
	@echo "🔧 Building jf shim..."
	cd shim && go build -o $(SHIM_BIN) .

install: build
	@echo "📂 Creating shim directory: $(SHIM_DIR)"
	mkdir -p $(SHIM_DIR)
	@echo "📥 Installing binaries to $(SHIM_DIR)"
	cp $(JFVM_BIN) $(SHIM_DIR)/
	cp shim/$(SHIM_BIN) $(SHIM_DIR)/
	@echo "✅ Binaries installed."

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
	rm -f $(SHIM_DIR)/$(JFVM_BIN) $(SHIM_DIR)/$(SHIM_BIN)
	@echo "✅ Uninstalled."

clean:
	@echo "🧹 Cleaning build artifacts..."
	rm -f $(JFVM_BIN)
	cd shim && rm -f $(SHIM_BIN)

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