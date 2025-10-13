# Jenkins Pipeline Setup Guide

## 🎯 Create JFVM Pipeline with Git Checkout

### **Step 1: Create Pipeline in Jenkins**

1. **Open Jenkins**: http://localhost:8080
2. **Login**: `admin` / `password`
3. **New Item** → Name: `JFVM-Build` → **Pipeline** → **OK**

### **Step 2: Configure Pipeline**

**Copy the complete pipeline script from `jenkins-setup/jfvm-pipeline.groovy`**

The pipeline includes these key features:

#### **🔄 Git Checkout Logic**
```groovy
stage('Checkout Code') {
    steps {
        script {
            // Clean workspace
            sh 'rm -rf .git || true; rm -rf * || true'
            
            // Clone repository
            sh "git clone ${env.REPO_URL} ."
            
            // Checkout specific branch
            sh "git checkout ${params.GIT_BRANCH}"
            
            // Optional: checkout specific commit
            if (params.GIT_COMMIT?.trim()) {
                sh "git checkout ${params.GIT_COMMIT}"
            }
        }
    }
}
```

#### **🏗️ Build Parameters**
- **GIT_BRANCH**: Which branch to build (default: main)
- **GIT_COMMIT**: Specific commit hash (optional)
- **ARCHITECTURES**: primary, linux-only, darwin-only, windows-only, all
- **SKIP_TESTS**: Skip binary testing

#### **🔧 Build Matrix**
- **Primary**: Linux AMD64, Darwin AMD64/ARM64, Windows AMD64
- **All**: Includes Linux 386, FreeBSD, etc.

#### **📤 Artifactory Upload**
Uploads to: `jfvm-binaries/jfvm/v1/{version}/jfvm-{os}-{arch}/`

### **Step 3: Run Build**

1. **Build with Parameters**
2. **Set parameters**:
   - GIT_BRANCH: `main` (or your branch)
   - ARCHITECTURES: `primary`
   - SKIP_TESTS: `false`
3. **Build**

### **Step 4: View Results**

**Build Progress**: http://localhost:8080/job/JFVM-Build/
**Artifacts**: http://localhost:8082/ui/repos/tree/General/jfvm-binaries

## 🎯 **What the Git Checkout Does:**

1. **Clones** your local JFVM repository into Jenkins workspace
2. **Checks out** the specified branch (main, develop, feature/xyz)
3. **Optionally** checks out a specific commit hash
4. **Verifies** essential files (go.mod, main.go) exist
5. **Downloads** Go dependencies
6. **Shows** current Git state (branch, commit, tags)

## 🚀 **Repository Structure Created:**

```
jfvm-binaries/jfvm/v1/dev-{build}/
  ├── jfvm-linux-amd64/
  │   ├── jfvm
  │   └── jfvm.sha256
  ├── jfvm-darwin-amd64/
  │   ├── jfvm
  │   └── jfvm.sha256
  ├── jfvm-darwin-arm64/
  │   ├── jfvm
  │   └── jfvm.sha256
  └── jfvm-windows-amd64/
      ├── jfvm.exe
      └── jfvm.exe.sha256
```

**This follows the exact JFrog CLI repository layout pattern!** 🎉
