# Project Rename: git-audit-cli → codegate

Successfully renamed the project from `git-audit-cli` to `codegate`.

## What Changed

### 1. Go Module & Imports

- **go.mod**: Module name changed to `codegate`
- **All \*.go files**: Import paths updated from `git-audit-cli/*` to `codegate/*`
- All 14 Go source files updated

### 2. Binary & Build

- **Binary name**: `git-audit` → `codegate`
- **Makefile**: Updated binary name
- **.gitignore**: Updated to exclude `codegate` binary

### 3. Configuration

- **Config file**: `.git-audit.yaml` → `.codegate.yaml`
- **Example config**: Renamed to `.codegate.example.yaml`
- **Environment prefix**: `GIT_AUDIT_*` → `CODEGATE_*`
- **Default paths**: Updated in `cmd/root.go` and `cmd/init.go`

### 4. Command Line

- **Command name**: `git-audit` → `codegate`
- **Usage**: All commands now use `codegate` prefix
  ```bash
  codegate version
  codegate init
  codegate review
  ```

### 5. Branding & Documentation

- **Project name**: "Git Audit CLI" → "CodeGate"
- **Description**: Updated to emphasize code quality gate concept
- **All documentation**: README, CONTRIBUTING, CLAUDE.md, CHANGELOG updated

### 6. Internal References

- **Debug log**: `git-audit-json-debug.log` → `codegate-debug.log`
- **Temp files**: `git-audit-*.md` → `codegate-*.md`
- **User-Agent**: `git-audit-cli/1.0` → `codegate/1.0`

### 7. GitHub Workflows

- **CI workflow**: Updated all binary references
- **Release workflow**: Updated to build and release `codegate-*` binaries
- **Artifact names**: `git-audit-linux-amd64` → `codegate-linux-amd64`, etc.

## Files Modified

- **Go files**: 14 files (all import paths updated)
- **Documentation**: 5 files (README, CONTRIBUTING, CLAUDE.md, CHANGELOG, OPEN_SOURCE_CHANGES)
- **Configuration**: 3 files (Makefile, .gitignore, .codegate.example.yaml)
- **Commands**: 4 files (root.go, init.go, version.go, review.go)
- **Workflows**: 2 files (ci.yml, release.yml)

**Total**: ~25 files touched

## Verification

✅ **Build**: Successfully builds as `codegate`

```bash
$ make build
Building codegate dev...
✓ Build complete: ./codegate
```

✅ **Version**: Displays correct name

```bash
$ ./codegate version
codegate version dev
commit: none
built: 2026-02-14T00:43:17Z
```

✅ **Help**: Shows updated commands

```bash
$ ./codegate --help
CodeGate analyzes your staged changes using AI...
Available Commands:
  init        Initialize CodeGate configuration
  review      Analyze staged git changes for issues
  version     Print version information
```

✅ **Tests**: All 29 tests pass

```bash
$ make test
✓ Tests complete
```

## Next Steps

1. **Update GitHub Repository**
   - Rename repository to `codegate`
   - Update repository description

2. **Update URLs in Documentation**
   - Replace `DoffuXx` with actual GitHub username
   - Update badge URLs in README.md

3. **Tag First Release**
   ```bash
   git add .
   git commit -m "feat: rename project to codegate"
   git tag -a v1.0.0 -m "First stable release as CodeGate"
   git push origin main --tags
   ```

## Migration Guide for Users

If users have the old `git-audit` installed:

1. **Remove old binary**:

   ```bash
   rm /usr/local/bin/git-audit
   ```

2. **Install codegate**:

   ```bash
   go install github.com/DoffuXx/codegate@latest
   ```

3. **Rename config file**:

   ```bash
   mv ~/.git-audit.yaml ~/.codegate.yaml
   ```

4. **Update environment variables** (if used):

   ```bash
   # Old
   export GIT_AUDIT_PROVIDER=lmstudio

   # New
   export CODEGATE_PROVIDER=lmstudio
   ```

5. **Update git hooks** (if configured):
   Replace `git-audit` with `codegate` in `.git/hooks/pre-commit`

---

**Rename completed successfully! 🎉**
