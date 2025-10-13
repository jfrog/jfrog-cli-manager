# JFVM Jenkins Local Development Environment

Complete Jenkins setup for JFVM development with Artifactory integration.

## Quick Start

```bash
# Install and start everything
cd jenkins-setup
./install.sh

# Access services
# Jenkins:     http://localhost:8080 (admin/admin123)
# Artifactory: http://localhost:8082 (admin/password)
```

## What You Get

### Services
- **Jenkins**: Fully configured with required plugins and shared library
- **Artifactory**: Repository for storing and managing JFVM artifacts
- **Docker Network**: Secure communication between services

### Repositories Created
- `jfvm-binaries` - Binary artifacts organized by version/platform
- `jfvm-docker` - Docker test images
- `jfvm-npm` - NPM packages (future use)

### Repository Structure (JFrog CLI Style)
```
jfvm-binaries/
  jfvm/v1/{version}/
    jfvm-linux-amd64/jfvm
    jfvm-linux-arm64/jfvm
    jfvm-darwin-amd64/jfvm
    jfvm-darwin-arm64/jfvm
    jfvm-windows-amd64/jfvm.exe
    jfvm-freebsd-amd64/jfvm
    jfvm-linux-s390x/jfvm
    jfvm-linux-ppc64le/jfvm
```

## Pipeline Features

### Build Matrix
- **Primary Platforms**: Linux, macOS, Windows (amd64, arm64)
- **Extended Platforms**: FreeBSD, s390x, ppc64le
- **Parallel Building**: All architectures built simultaneously

### Testing Framework
- **Cross-Platform**: Docker-based testing for different OS/arch
- **Test Cases**: Version check, help command, basic functionality
- **JUnit Reports**: Integrated with Jenkins test reporting

### Shared Library Functions
- `buildJfvmPipeline()` - Main pipeline orchestrator
- `buildBinaries()` - Parallel binary building
- `publishArtifacts()` - Artifactory publishing with retry logic
- `testBinaries()` - Cross-platform testing
- `createRepositories()` - Automated repository setup

## Usage

### Create Pipeline Job
1. Open Jenkins at http://localhost:8080
2. New Item → Pipeline
3. Pipeline Script from SCM:
   - Repository URL: `file:///path/to/jfrog-cli-vm`
   - Script Path: `Jenkinsfile.local`

### Build Parameters
- **BUILD_TYPE**: dev, release, pr
- **VERSION**: Custom version (auto-detected if empty)
- **ARCHITECTURES**: primary, all, linux-only, darwin-only, windows-only
- **SKIP_TESTS**: Skip cross-platform testing
- **CREATE_REPOSITORIES**: Auto-create Artifactory repositories

### Viewing Results
- **Artifacts**: Jenkins job artifacts + Artifactory browser
- **Test Results**: Jenkins test reports
- **Build Info**: Artifactory build info integration

## Management

### Control Scripts
```bash
./start.sh      # Start environment
./stop.sh       # Stop environment (preserve data)
./cleanup.sh    # Complete cleanup (delete everything)
```

### Repository Management
```bash
./create-repositories.sh [artifactory_url] [user] [password]
```

### Troubleshooting
- **Logs**: `docker-compose logs jenkins` or `docker-compose logs artifactory`
- **Network**: Check `docker network ls` for `jenkins-setup_jfvm-network`
- **Connectivity**: Test with `curl http://localhost:8080` and `curl http://localhost:8082`

## Networking Details

### Container Communication
- **Network**: Custom bridge `jfvm-network` (172.20.0.0/16)
- **Artifactory**: Static IP 172.20.0.10
- **Jenkins**: Static IP 172.20.0.20
- **Service Discovery**: Container names + static IPs for reliability

### Port Mapping
- **Jenkins**: 8080 → 8080 (Web UI), 50000 → 50000 (Agent)
- **Artifactory**: 8082 → 8082 (Web UI), 8081 → 8081 (Router)

## Security

### Default Credentials
- **Jenkins**: admin/admin123
- **Artifactory**: admin/password

### Recommendations
- Change default passwords in production
- Use Jenkins credential management for sensitive data
- Configure proper user access controls

## Advanced Configuration

### Custom Artifactory URL
If using external Artifactory:
```bash
export ARTIFACTORY_URL=http://your-artifactory:8082
./start.sh
```

### Jenkins Configuration as Code
Edit `jenkins.yaml` to modify:
- User accounts and permissions
- Tool configurations
- Plugin settings
- Global configurations

### Shared Library Customization
Modify files in `shared-library/` to:
- Add new build steps
- Customize test procedures
- Integrate with other tools
- Extend Artifactory operations
