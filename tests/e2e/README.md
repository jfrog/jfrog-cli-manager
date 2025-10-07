# jfcm End-to-End (E2E) Test Suite

This directory contains comprehensive end-to-end tests for the jfcm CLI tool. These tests verify that all features work correctly across different platforms and scenarios.

## 🧪 Test Coverage

### Core Features
- ✅ **Version Management**: Install, use, list, remove, clear
- ✅ **Latest Version**: Automatic fetching and installation
- ✅ **Alias Management**: Set, get, use, remove aliases
- ✅ **Project Integration**: `.jfrog-version` file support
- ✅ **Local Binary Linking**: Link custom binaries
- ✅ **Version Comparison**: Compare outputs between versions
- ✅ **Benchmarking**: Performance testing across versions
- ✅ **History Tracking**: Usage history and analytics
- ✅ **Error Handling**: Invalid inputs and edge cases
- ✅ **Security**: Binary permissions and file security

### Platform Support
- ✅ **Ubuntu**: Linux AMD64 testing
- ✅ **macOS**: Apple Silicon and Intel testing
- 🔄 **Windows**: Planned support

## 🚀 Running Tests

### Local Development

```bash
# Run all E2E tests
make test-e2e

# Run tests locally (faster)
make test-e2e-local

# Run specific test categories
make test-e2e-latest    # Test 'latest' functionality
make test-e2e-alias     # Test alias management
make test-e2e-performance # Test performance

# Run with custom timeout
TEST_TIMEOUT_OVERRIDE=5m make test-e2e-local
```

### CI/CD Pipeline

The tests run automatically on:
- **Push to main/develop**: Full test suite
- **Pull Requests**: Full test suite + PR comments
- **Manual trigger**: Selective platform testing

### GitHub Actions

```bash
# Trigger workflow manually
gh workflow run e2e-tests.yml

# Run on specific platform
gh workflow run e2e-tests.yml -f platform=ubuntu
gh workflow run e2e-tests.yml -f platform=macos
```

## 📋 Test Structure

### Test Files
- `test_suite.go`: Main test suite with all test cases
- `run_tests.sh`: Local test runner script
- `run_ci_tests.sh`: CI-optimized test runner
- `test_config.yaml`: Test configuration and scenarios

### Test Categories

#### 1. Core Version Management (`TestCoreVersionManagement`)
- Install specific versions
- List installed versions
- Use specific versions
- Use latest version (with download)
- Remove versions
- Clear all versions

#### 2. Alias Management (`TestAliasManagement`)
- Set valid aliases
- Get alias values
- Use aliases
- Block reserved keywords (e.g., "latest")
- Remove aliases

#### 3. Project Integration (`TestProjectSpecificVersion`)
- Use `.jfrog-version` files
- Handle missing project files
- Validate project file content

#### 4. Local Binary Linking (`TestLinkLocalBinary`)
- Link custom binaries
- Use linked binaries
- Handle invalid binary paths

#### 5. Advanced Features
- **Version Comparison** (`TestCompareVersions`)
- **Benchmarking** (`TestBenchmarkVersions`)
- **History Tracking** (`TestHistoryTracking`)

#### 6. Error Handling (`TestErrorHandling`)
- Invalid version numbers
- Non-existent versions
- Invalid commands
- Missing arguments

#### 7. Performance (`TestPerformance`)
- Install performance
- List performance
- Concurrent operations

#### 8. Security (`TestSecurity`)
- Binary permissions
- File security

## 🔧 Test Configuration

### Environment Variables
- `jfcm_PATH`: Path to jfcm binary
- `TEST_TIMEOUT_OVERRIDE`: Custom test timeout
- `TEST_FILTER`: Run specific test patterns
- `jfcm_DEBUG`: Enable debug output

### Test Timeouts
- **Short**: 30s (quick operations)
- **Medium**: 120s (install/use operations)
- **Long**: 300s (download operations)
- **Very Long**: 600s (full workflows)

## 📊 Test Results

### Local Results
Tests output detailed information including:
- Test execution time
- Success/failure status
- Error messages
- Performance metrics

### CI Results
GitHub Actions provides:
- ✅/❌ Status badges
- Detailed logs
- Test artifacts
- PR comments with results
- Coverage reports

## 🐛 Debugging Tests

### Common Issues

1. **Network Timeouts**
   ```bash
   # Increase timeout for slow connections
   TEST_TIMEOUT_OVERRIDE=10m make test-e2e-local
   ```

2. **Permission Issues**
   ```bash
   # Ensure scripts are executable
   chmod +x tests/e2e/*.sh
   ```

3. **Binary Not Found**
   ```bash
   # Set correct jfcm_PATH
   jfcm_PATH=/path/to/jfcm make test-e2e-local
   ```

### Debug Mode
```bash
# Enable debug output
jfcm_DEBUG=1 make test-e2e-local
```

### Running Individual Tests
```bash
# Run specific test
go test -v ./tests/e2e/ -run TestCoreVersionManagement

# Run specific subtest
go test -v ./tests/e2e/ -run TestCoreVersionManagement/Install_Version
```

## 📈 Adding New Tests

### Test Structure
```go
func TestNewFeature(t *testing.T) {
    ts := SetupTestSuite(t)
    defer ts.CleanupTestSuite(t)

    t.Run("Test Case Name", func(t *testing.T) {
        output, err := ts.RunCommand(t, "command", "args")
        ts.AssertSuccess(t, output, err)
        ts.AssertContains(t, output, "expected")
    })
}
```

### Best Practices
1. **Isolation**: Each test should be independent
2. **Cleanup**: Always clean up after tests
3. **Assertions**: Use specific assertions for clarity
4. **Timeouts**: Set appropriate timeouts for operations
5. **Documentation**: Add clear test descriptions

## 🔄 Continuous Integration

### Workflow Triggers
- **Automatic**: Push to main/develop branches
- **Manual**: Workflow dispatch with platform selection
- **Scheduled**: Daily performance tests (planned)

### Artifacts
- Test results (JSON format)
- Coverage reports
- Performance metrics
- Security scan results

### Notifications
- PR comments with test results
- Slack/Discord notifications (configurable)
- Email alerts for failures

## 🎯 Quality Metrics

### Test Coverage Goals
- **Line Coverage**: >90%
- **Function Coverage**: >95%
- **Branch Coverage**: >85%

### Performance Targets
- **Install Time**: <30s for stable versions
- **List Time**: <5s
- **Use Time**: <10s

### Reliability Targets
- **Test Flakiness**: <1%
- **False Positives**: <0.1%
- **False Negatives**: <0.1%

## 🤝 Contributing

### Adding Tests
1. Create test function in `test_suite.go`
2. Add test configuration in `test_config.yaml`
3. Update this README
4. Run tests locally
5. Submit PR

### Test Review Checklist
- [ ] Tests are isolated and independent
- [ ] Proper cleanup is implemented
- [ ] Timeouts are appropriate
- [ ] Error cases are covered
- [ ] Documentation is updated
- [ ] CI passes on all platforms

## 📚 Resources

- [Go Testing Package](https://golang.org/pkg/testing/)
- [GitHub Actions Documentation](https://docs.github.com/en/actions)
- [jfcm Main Documentation](../README.md) 