# GitHub Actions Workflows

This document describes the CI/CD pipeline and workflow dependencies.

## Workflow Execution Order

```
┌─────────────────────────────────────────────────────────┐
│                    Push to main/develop                  │
│                   or Pull Request to main                │
└────────────────────┬────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────┐
│                     CI Workflow                          │
│  ┌──────────┐  ┌──────────┐                            │
│  │   Lint   │  │   Test   │  (run in parallel)         │
│  └────┬─────┘  └────┬─────┘                            │
│       │             │                                    │
│       └──────┬──────┘                                    │
│              │                                           │
│       ┌──────┴──────┐                                   │
│       │             │                                    │
│  ┌────▼─────┐  ┌───▼────────┐                          │
│  │  Build   │  │  Security  │  (run in parallel)       │
│  │ (needs:  │  │  (needs:   │                          │
│  │lint,test)│  │ lint,test) │                          │
│  └──────────┘  └────────────┘                          │
└────────────────────┬────────────────────────────────────┘
                     │
         ✓ All jobs succeeded
                     │
                     ▼
┌─────────────────────────────────────────────────────────┐
│               Docker Build Workflow                      │
│  (Triggers only after CI workflow completes)            │
│                                                          │
│  1. Verify CI succeeded (skip for manual/tag triggers)  │
│  2. Build Docker Image (multi-platform)                 │
│  3. Push to GitHub Container Registry                   │
│  4. Scan image with Trivy                               │
│  5. Upload security scan results                        │
└─────────────────────────────────────────────────────────┘
```

## Workflow Files

### `.github/workflows/ci.yml`
**Triggers:**
- Push to `main` or `develop` branches
- Pull requests to `main` branch

**Jobs:**
1. **lint** - Runs golangci-lint with 5-minute timeout
2. **test** - Runs all tests with race detection and generates coverage
3. **build** - Builds all 4 binaries (needs: lint, test)
4. **security** - Runs Gosec security scanner (needs: lint, test)

**Success Criteria:** All jobs must pass (lint, test, build, security)

### `.github/workflows/docker.yml`
**Triggers:**
- `workflow_run` - After CI workflow completes on `main` branch
- `workflow_dispatch` - Manual trigger
- `push` to version tags (`v*`)

**Jobs:**
1. **build-and-push** - Builds and pushes Docker images
   - First step verifies CI succeeded (for workflow_run triggers)
   - Then builds multi-platform images
   - Pushes to registry and scans for vulnerabilities

**Protection:** Docker build only proceeds if:
- CI workflow completed successfully (for workflow_run trigger)
- OR manually triggered (workflow_dispatch) - bypasses CI check
- OR version tag push - bypasses CI check

## Dependency Chain

```
Lint ─┐
      ├──> Build ──┐
Test ─┤            ├──> CI Success ──> Docker Build
      └──> Security┘
```

## Workflow Behaviors

### On Push to `main` or `develop`:
1. CI workflow runs (lint, test, build, security)
2. If all CI jobs succeed AND branch is `main`:
   - Docker workflow automatically triggers
   - Docker image is built and pushed

### On Pull Request to `main`:
1. CI workflow runs (lint, test, build, security)
2. Docker workflow does NOT automatically trigger
3. Docker image can be manually triggered if needed

### On Version Tag (`v*`):
1. Docker workflow triggers directly
2. Builds and publishes release images
3. CI workflow may run separately depending on branch

## Manual Workflow Triggers

You can manually trigger workflows from GitHub Actions UI:
- **Docker Build**: Use "workflow_dispatch" trigger
  - Useful for rebuilding images without code changes
  - Bypasses CI dependency check

## Failure Handling

### If CI workflow fails:
- Docker build workflow will NOT trigger
- No Docker image will be built or pushed
- Fix the issues and push again

### If Docker build fails:
- Does not affect CI workflow status
- Image will not be pushed to registry
- Can be retried manually via workflow_dispatch

## Security Considerations

1. **SARIF uploads** are marked as `continue-on-error: true` to prevent blocking if CodeQL is not configured
2. **Trivy scans** only run on successful builds
3. **Container registry** requires GitHub token with appropriate permissions
4. All workflows use pinned action versions for security

## Environment Variables

### CI Workflow
- `GO_VERSION`: '1.21' (Go version for builds and tests)

### Docker Workflow
- `REGISTRY`: ghcr.io (GitHub Container Registry)
- `IMAGE_NAME`: ${{ github.repository }} (derived from repo)

## Artifacts

### CI Workflow
- **binaries**: All 4 compiled binaries (hyundai-logger, export-data, hyundai-logger-interactive, validate-config)
- **coverage.txt**: Test coverage report (uploaded to Codecov)
- **results.sarif**: Security scan results (uploaded to GitHub Code Scanning)

### Docker Workflow
- **Docker images**: Multi-platform images (linux/amd64, linux/arm64, linux/arm/v7)
- **trivy-results.sarif**: Container vulnerability scan results

## Best Practices

1. ✅ **Always wait for CI**: Docker builds only happen after successful CI
2. ✅ **Parallel execution**: Lint and Test run in parallel for speed
3. ✅ **Clear dependencies**: Build and Security depend on Lint+Test
4. ✅ **Security first**: Both code (Gosec) and container (Trivy) scanning
5. ✅ **Fail fast**: Jobs fail immediately on errors
6. ✅ **Artifact preservation**: Binaries and scan results saved for review

## Troubleshooting

### "Docker workflow didn't trigger"
- Check if CI workflow succeeded on `main` branch
- Verify you're on `main` branch (Docker only auto-triggers on `main`)
- Check GitHub Actions logs for CI workflow status

### "CI is passing but Docker build fails"
- Docker build has separate requirements (Dockerfile, dependencies)
- Check Docker build logs for specific errors
- Ensure all required files are committed

### "Security scan uploads failing"
- This is expected if CodeQL is not set up
- Workflows continue with `continue-on-error: true`
- Not a blocker for deployments

## Monitoring

Watch the Actions tab in GitHub for:
- ✅ Green checkmarks = Success
- ❌ Red X = Failure
- 🟡 Yellow dot = In progress
- ⏸️ Gray circle = Skipped/Not required
