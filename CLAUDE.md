# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Design Philosophy

This project follows a **minimal and simple** design philosophy. The smaller the public API surface, the easier it is to maintain.

## Usage

See [README.md](README.md) for usage, CLI flags, and project overview.

## Exit Codes

- `0`: Success
- `1`: Invalid arguments
- `2`: Invalid purl format
- `3`: Runtime error (API failure, network error, etc.)

## Development

**Quick Commands:** `make test`, `make lint-check`, `make format-check` before submitting changes.

**Test Organization:**
- Unit tests (`*_test.go`): Fast, use mocks, run by default with `make test`
- Integration tests (`*_integration_test.go`): Require network, use `//go:build integration` tag, run with `make test-integration`

**Dependencies:** Go 1.25.0, `github.com/package-url/packageurl-go v0.1.3`

## Architecture

**`Service` interface** (service.go:22-25)
- All package registry integrations implement `GetPackageInfo(ctx context.Context, purl packageurl.PackageURL) (PackageInfo, error)`
- Context handles timeout control and request cancellation
- Returns standardized `PackageInfo{Name, Version, Licenses}`

**`PackageInfo` struct** (service.go:8-19)
- Unified response format: `Name`, `Version`, `Licenses []string`
- JSON-serializable with struct tags

**Sentinel Errors** (service.go)
- `ErrPackageNotFound` - Package not found (404 or empty results)
- `ErrInvalidResponse` - Invalid API response format
- Use with `errors.Is()` for robust error handling

**EcosystemsService** (ecosystems.go)
- Constructor: `NewEcosystemsService(opts EcosystemsServiceOptions)`
  - `BaseURL string` - Empty = default, no pointer
  - `Client *http.Client` - Nil = `http.DefaultClient`
  - `Email string` - Optional for polite pool (sets User-Agent: `purlinfo/VERSION (mailto:EMAIL)`)
- Uses `/api/v1/packages/lookup?purl=` endpoint (NOT `/api/v1/packages/{purl}`)
- Maps API response: `name` → `Name`, `latest_release_number` → `Version`, `normalized_licenses` → `Licenses`
- Always use `http.NewRequestWithContext(ctx, ...)` for context-aware requests

**CLI Implementation** (main.go)
- Delegates `main() { os.Exit(run()) }` to handle deferred cleanup before exit
- Helper functions: `printUsage()`, `setupLogger(verbose)`, `createService(client, email)`, `printOutput(info, json)`
- Structured logging with `log/slog` (required by linter)

**Code Organization** (root package `main`)
- `main.go` - CLI, flag parsing, main logic
- `service.go` - Core interfaces, types, sentinel errors
- `ecosystems.go` - Ecosyste.ms service implementation

## Linting Configuration

Uses **extremely strict** golangci-lint (50+ linters). Common gotchas:

1. **exitAfterDefer**: Don't call `os.Exit()` after `defer` - use `main() { os.Exit(run()) }` pattern
2. **Shadow**: Don't redeclare variables with `:=` in inner scopes - use unique names
3. **Magic numbers**: Define all numeric constants at the top of the file
4. **Error strings**: Start error messages with lowercase (unless proper noun)
5. **forbidigo**: Use `fmt.Fprintf(os.Stdout, ...)` instead of `fmt.Printf`
6. **perfsprint**: Use `errors.New()` for static strings instead of `fmt.Errorf()` without format specifiers
7. **No global variables** (`gochecknoglobals`) or init functions (`gochecknoinits`)
8. **Import ordering**: Uses goimports with local prefix `github.com/boringbin/purlinfo`

Other enforced rules: no naked returns, type assertion error checking, HTTP/SQL body close verification, no variable shadowing, exhaustive switch/map cases. Test files have relaxed rules.

## Testing Patterns

**Dependency Injection for Testability:**

The CLI uses function extraction to enable testing with mock services:
- `run()` - Production entry point, handles setup and creates real services
- `runWithService()` - Core logic, accepts `Service` as parameter for easy testing

Example:
```go
// In tests, inject a mock service
mockSvc := &mockService{info: PackageInfo{...}, err: nil}
exitCode := runWithService(mockSvc, logger, purl, "purl-string", false, false, 30*time.Second)
```

When adding functions with external dependencies:
1. Extract core logic into a `*WithDependency()` function
2. Keep the public function for production use
3. Test via the extracted function with mocks

## API Integration Notes

**Ecosyste.ms:** Use `/api/v1/packages/lookup?purl=` endpoint (NOT `/api/v1/packages/{purl}`)
- Returns an array, even for single results - always check for empty results before accessing index
- Use `latest_release_number` field for version (NOT `version`)
