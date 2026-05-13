## task-31-ci-config

### Patterns
- GitHub Actions matrix for Go versions: `strategy.matrix.go-version`
- `defaults.run.working-directory` used to set workspace since Go module is in `gst/` subdirectory
- Standard Go CI steps: checkout → setup-go → vet → test → build

### Conventions
- Workflow file: `.github/workflows/gst-ci.yml`
- Triggers: push + pull_request targeting `main`
- Go versions: 1.22, 1.23 (matching go.mod's go 1.22 directive)
