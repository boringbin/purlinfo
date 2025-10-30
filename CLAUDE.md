# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`purlinfo` is a Go CLI tool that retrieves package information from Package URLs (purls). The project uses the [purl specification](https://github.com/package-url/purl-spec) to identify and fetch metadata about packages across different ecosystems.

Currently only supported service is [Ecosyste.ms](https://ecosyste.ms/).

## Usage

```bash
# Basic usage
bin/purlinfo 'pkg:npm/lodash@4.17.21'

# JSON output
bin/purlinfo -json 'pkg:pypi/requests@2.28.0'

# Verbose mode for debugging
bin/purlinfo -v 'pkg:npm/lodash@4.17.21'

# Custom timeout
bin/purlinfo -timeout 10s 'pkg:npm/lodash@4.17.21'

# Show version
bin/purlinfo -version
```

### CLI Flags

- `-json` (bool): Output as JSON instead of human-readable format
- `-v` (bool): Verbose output with debug logging
- `-version` (bool): Show version and exit
- `-timeout` (duration): HTTP request timeout (default: 30s)

**Note:** Go's `flag` package uses single dash for all flags (e.g., `-json`, not `--json`). The `-v` flag for verbose follows Go's idiomatic CLI conventions.

### Exit Codes

- `0`: Success
- `1`: Invalid arguments
- `2`: Invalid purl format
- `3`: Runtime error (API failure, network error, etc.)

## Development Commands

### Building
```bash
make          # Build the binary to bin/purlinfo
make all      # Same as make
```

The build compiles from the root directory (`.`) to `bin/purlinfo`.

### Code Quality
```bash
make vet           # Run go vet static analysis
make lint-check    # Run golangci-lint (strict configuration)
make lint-fix      # Auto-fix lint issues where possible
make format-check  # Verify code is gofmt'd
make format-fix    # Auto-format code with gofmt
make check         # Run both format-check and lint-check
make fix           # Run both format-fix and lint-fix
```

### Testing
```bash
make test                               # Run unit tests with race detection (excludes integration tests)
make test-integration                   # Run integration tests (requires network, hits real APIs)
make test-coverage                      # Run unit tests and generate coverage report
make test-all                           # Run both unit and integration tests

# Direct go test commands
go test -v -short -race ./...           # Unit tests only
go test -v -tags=integration ./...      # Integration tests only
go test -v ./path/to/package            # Run tests for a specific package
```

**Test Organization:**
- **Unit tests** (`*_test.go`): Fast, use mocks, run by default
- **Integration tests** (`*_integration_test.go`): Require network, use `//go:build integration` tag
- Current test coverage: **~93%**

**Note:** Integration tests hit the real Ecosyste.ms API and are excluded from regular test runs. Use `make test-integration` to run them explicitly.

### Cleanup
```bash
make clean         # Remove bin/purlinfo, coverage.out, coverage.html
```

### Dependencies
```bash
make tidy          # Run go mod tidy
```

## Architecture

### Core Abstractions

The project is built around a service-based architecture with a well-defined interface pattern:

**`Service` interface** (service.go:22-25)
- All package registry integrations must implement `GetPackageInfo(ctx context.Context, purl packageurl.PackageURL) (PackageInfo, error)`
- Context is used for timeout control and request cancellation
- Returns standardized `PackageInfo` with name, version, and licenses

**`PackageInfo` struct** (service.go:8-19)
- Unified response format across all services
- Fields: `Name`, `Version`, `Licenses []string`
- JSON-serializable with struct tags

### Current Implementation

**EcosystemsService** (ecosystems.go)
- Implements the `Service` interface with context support
- Uses the `/api/v1/packages/lookup?purl=` endpoint (API path extracted as constant)
- Constructor: `NewEcosystemsService(opts EcosystemsServiceOptions)` with simplified options:
  - `BaseURL string` - Empty string uses default, no pointer indirection
  - `Client *http.Client` - Nil uses `http.DefaultClient`
- Maps Ecosyste.ms API response via unexported `ecosystemsPackagesLookupResponse` type:
  - `name` → `Name`
  - `latest_release_number` → `Version`
  - `normalized_licenses` → `Licenses`
- Includes detailed HTTP error handling:
  - 404 → Returns `ErrPackageNotFound` (wrapped sentinel error)
  - 429 → Rate limit error
  - 502/503/504 → Service unavailable error
  - Other non-200 → Generic API error
- Empty results return `ErrPackageNotFound`
- Invalid JSON returns `ErrInvalidResponse`
- Uses `http.NewRequestWithContext` for proper context-aware requests

**Sentinel Errors** (service.go)
- `ErrPackageNotFound` - Package not found (404 or empty results)
- `ErrInvalidResponse` - Invalid API response format
- Use with `errors.Is()` for robust error handling

**CLI Implementation** (main.go)
- Uses standard `flag` package for argument parsing (Go idioms: `-v`, `-json`, `-timeout`)
- Main function delegates to `run() int` to properly handle deferred cleanup and exit codes
- Helper functions for separation of concerns:
  - `printUsage()`: Custom usage message
  - `setupLogger(verbose bool)`: Creates slog.Logger with appropriate level
  - `createService(httpClient)`: Creates EcosystemsService (no service selection)
  - `printOutput(info, outputJSON)`: Handles both output formats
- Structured logging with `log/slog` (required by linter rules)
- HTTP client configured with timeout from CLI flag

### Code Organization

All code is currently in the root package (`main`). Files:
- `main.go` - CLI implementation with flag parsing and main logic
- `service.go` - Core interfaces, types, and sentinel errors
- `ecosystems.go` - Ecosyste.ms service implementation

## Linting Configuration

This project uses an **extremely strict** golangci-lint configuration based on the "golden config" by Marat Reymers. Key points:

- **50+ linters enabled** including staticcheck, revive, gosec, govet (with all analyzers)
- **Line length**: Maximum 120 characters (enforced by golines)
- **Complexity limits**: cyclop (max 30), gocognit (min 20), funlen (100 lines/50 statements)
- **Import ordering**: Uses goimports with local prefix `github.com/boringbin/purlinfo`
- **Forbidden patterns**:
  - `fmt.Printf` and similar (use `fmt.Fprintf` with explicit writer)
  - `log` package in non-main files (use `log/slog`)
  - `math/rand` in non-test files (use `math/rand/v2`)
  - Deprecated packages like `github.com/golang/protobuf`, `github.com/satori/go.uuid`
  - Reassigning `flag.Usage` directly (must assign to a separate variable first)
- **Strict checks enabled**:
  - Type assertion error checking
  - No naked returns
  - No global variables (`gochecknoglobals`)
  - No init functions (`gochecknoinits`)
  - Exhaustive switch/map cases
  - HTTP body close verification
  - SQL row close verification
  - Shadow detection (no variable shadowing)
  - No magic numbers (must use named constants)
  - Error strings must not be capitalized

Test files have relaxed rules (no funlen, cyclop, dupl, gosec enforcement).

### Common Linting Fixes

When adding code, watch out for:
1. **exitAfterDefer**: Don't call `os.Exit()` after `defer` - use the `main() { os.Exit(run()) }` pattern
2. **Shadow**: Don't redeclare variables with `:=` in inner scopes - use unique names
3. **Magic numbers**: Define all numeric constants at the top of the file
4. **Error strings**: Start error messages with lowercase (unless proper noun)
5. **forbidigo**: Use `fmt.Fprintf(os.Stdout, ...)` instead of `fmt.Printf`
6. **perfsprint**: Use `errors.New()` for static strings instead of `fmt.Errorf()` without format specifiers

## Go Version

Uses **Go 1.25.0** (go.mod:3)

## Dependencies

- `github.com/package-url/packageurl-go v0.1.3` - Official purl parser

## Development Patterns

### Design Philosophy

This project follows a **minimal and simple** design philosophy:
- Only one package registry service is currently supported (Ecosyste.ms)
- No service selection flags until a second service is added (YAGNI principle)
- Smaller public API surface for easier long-term maintenance
- When adding a second service, then add service selection (`-service` flag)

### Adding New Package Registry Services

When adding a second package registry service:

1. Create a new file (e.g., `servicename.go`)
2. Define a service struct with `baseURL` and `client` fields
3. Implement the `Service` interface with context support:
   ```go
   func (s *YourService) GetPackageInfo(ctx context.Context, purl packageurl.PackageURL) (PackageInfo, error)
   ```
4. Add `var _ Service = (*YourService)(nil)` compile-time check
5. Provide a constructor with simplified options pattern:
   ```go
   type YourServiceOptions struct {
       BaseURL string        // Empty = default, no pointer
       Client  *http.Client  // Nil = http.DefaultClient
   }
   ```
6. Keep internal response types unexported (lowercase names)
7. Use sentinel errors from `service.go` where appropriate (`ErrPackageNotFound`, `ErrInvalidResponse`)
8. Provide specific error messages for different HTTP status codes
9. Use `http.NewRequestWithContext(ctx, ...)` for HTTP requests (not `client.Get`)
10. Map the external API response to `PackageInfo` struct
11. Update `createService()` in main.go to accept service name and return appropriate service
12. Add `-service` flag back to main.go
13. Update help text in `printUsage()` function

### Testing Patterns

**Dependency Injection for Testability:**

The CLI uses a function extraction pattern to enable testing with mock services:
- `run()` - Production entry point, handles setup and creates real services
- `runWithService()` - Core logic, accepts Service as parameter for easy testing

This pattern allows E2E testing without complex dependency injection frameworks:

```go
// In tests, inject a mock service
mockSvc := &mockService{info: PackageInfo{...}, err: nil}
exitCode := runWithService(mockSvc, logger, purl, "purl-string", false, false, 30*time.Second)
```

When adding new functions that need external dependencies:
1. Extract core logic into a `*WithDependency()` function
2. Keep the public function for production use
3. Test via the extracted function with mocks

### API Integration Notes

- **Ecosyste.ms**: Use the `/api/v1/packages/lookup?purl=` endpoint, not `/api/v1/packages/{purl}`
  - The lookup endpoint returns an array, even for single results
  - Use `latest_release_number` field for version (not `version`)
  - Always check for empty results before accessing array index

## Output Formats

### Human-Readable (Default)
```
Name:     lodash
Version:  4.17.21
Licenses: MIT
```

### JSON (--json flag)
```json
{
  "name": "lodash",
  "version": "4.17.21",
  "licenses": [
    "MIT"
  ]
}
```

## Build Output

The binary is built to `bin/purlinfo` from the root directory source files.
