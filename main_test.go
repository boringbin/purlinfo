package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/package-url/packageurl-go"
)

// mockService is a mock implementation of the Service interface for testing.
type mockService struct {
	info PackageInfo
	err  error
}

func (m *mockService) GetPackageInfo(_ context.Context, _ packageurl.PackageURL) (PackageInfo, error) {
	return m.info, m.err
}

// TestPrintUsage tests the printUsage function.
func TestPrintUsage(t *testing.T) {
	// Note: Cannot use t.Parallel() because test modifies global os.Stderr

	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	printUsage()

	_ = w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	// Check that usage contains expected strings
	expectedStrings := []string{
		"Usage:",
		"Get package information",
		"Arguments:",
		"pkg:npm/lodash",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("printUsage() output missing %q\nGot output:\n%s", expected, output)
		}
	}
}

// TestSetupLogger tests the setupLogger function.
func TestSetupLogger(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		verbose bool
		want    slog.Level
	}{
		{
			name:    "verbose mode",
			verbose: true,
			want:    slog.LevelDebug,
		},
		{
			name:    "non-verbose mode",
			verbose: false,
			want:    slog.LevelError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			logger := setupLogger(tt.verbose)
			if logger == nil {
				t.Fatal("setupLogger() returned nil")
			}

			// Logger should be configured but we can't easily inspect the level
			// We mainly test that it doesn't panic and returns a logger
		})
	}
}

// TestPrintOutput tests the printOutput function.
func TestPrintOutput(t *testing.T) {
	// Note: Cannot use t.Parallel() because subtests modify global os.Stdout

	tests := []struct {
		name       string
		info       PackageInfo
		outputJSON bool
		wantStdout []string // Strings that should appear in output
	}{
		{
			name: "human-readable with licenses",
			info: PackageInfo{
				Name:     "lodash",
				Version:  "4.17.21",
				Licenses: []string{"MIT"},
			},
			outputJSON: false,
			wantStdout: []string{"Name:", "lodash", "Version:", "4.17.21", "Licenses:", "MIT"},
		},
		{
			name: "human-readable without licenses",
			info: PackageInfo{
				Name:     "testpkg",
				Version:  "1.0.0",
				Licenses: []string{},
			},
			outputJSON: false,
			wantStdout: []string{"Name:", "testpkg", "Version:", "1.0.0", "Licenses:", "(none)"},
		},
		{
			name: "human-readable with multiple licenses",
			info: PackageInfo{
				Name:     "requests",
				Version:  "2.32.5",
				Licenses: []string{"Apache-2.0", "MIT"},
			},
			outputJSON: false,
			wantStdout: []string{"Name:", "requests", "Version:", "2.32.5", "Licenses:", "Apache-2.0", "MIT"},
		},
		{
			name: "JSON output",
			info: PackageInfo{
				Name:     "lodash",
				Version:  "4.17.21",
				Licenses: []string{"MIT"},
			},
			outputJSON: true,
			wantStdout: []string{`"name"`, `"lodash"`, `"version"`, `"4.17.21"`, `"licenses"`, `"MIT"`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: Cannot use t.Parallel() because test modifies global os.Stdout

			// Capture stdout.
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			err := printOutput(tt.info, tt.outputJSON)

			_ = w.Close()
			os.Stdout = oldStdout

			if err != nil {
				t.Errorf("printOutput() unexpected error = %v", err)
				return
			}

			var buf bytes.Buffer
			_, _ = io.Copy(&buf, r)
			output := buf.String()

			// Check for expected strings in output
			for _, want := range tt.wantStdout {
				if !strings.Contains(output, want) {
					t.Errorf("printOutput() output missing %q\nGot: %s", want, output)
				}
			}

			// For JSON output, validate it's actually valid JSON
			if tt.outputJSON {
				var result PackageInfo
				if jsonErr := json.Unmarshal([]byte(output), &result); jsonErr != nil {
					t.Errorf("printOutput() produced invalid JSON: %v\nOutput: %s", jsonErr, output)
				}
			}
		})
	}
}

// TestRun_Version tests the run function with the --version flag.
func TestRun_Version(t *testing.T) {
	// Note: Cannot use t.Parallel() because run() modifies global flag.CommandLine

	// Save and restore os.Args and flag.CommandLine
	oldArgs := os.Args
	oldCommandLine := flag.CommandLine
	t.Cleanup(func() {
		os.Args = oldArgs
		flag.CommandLine = oldCommandLine
	})

	// Reset flag.CommandLine for this test
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	os.Args = []string{"purlinfo", "--version"}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	exitCode := run()

	_ = w.Close()
	os.Stdout = oldStdout

	if exitCode != exitSuccess {
		t.Errorf("run() with --version returned exit code %d, want %d", exitCode, exitSuccess)
	}

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if !strings.Contains(output, "purlinfo version") {
		t.Errorf("run() --version output = %q, want to contain 'purlinfo version'", output)
	}
	if !strings.Contains(output, version) {
		t.Errorf("run() --version output = %q, want to contain version %q", output, version)
	}
}

// TestRun_NoArguments tests the run function with no arguments.
func TestRun_NoArguments(t *testing.T) {
	// Note: Cannot use t.Parallel() because run() modifies global flag.CommandLine

	// Save and restore os.Args and flag.CommandLine
	oldArgs := os.Args
	oldCommandLine := flag.CommandLine
	t.Cleanup(func() {
		os.Args = oldArgs
		flag.CommandLine = oldCommandLine
	})

	// Reset flag.CommandLine for this test
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	os.Args = []string{"purlinfo"}

	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	exitCode := run()

	_ = w.Close()
	os.Stderr = oldStderr

	if exitCode != exitInvalidArgs {
		t.Errorf("run() with no args returned exit code %d, want %d", exitCode, exitInvalidArgs)
	}

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if !strings.Contains(output, "purl argument is required") {
		t.Errorf("run() no args output = %q, want to contain 'purl argument is required'", output)
	}
}

// TestRun_InvalidPURL tests the run function with an invalid purl.
func TestRun_InvalidPURL(t *testing.T) {
	// Note: Cannot use t.Parallel() because run() modifies global flag.CommandLine

	// Save and restore os.Args and flag.CommandLine
	oldArgs := os.Args
	oldCommandLine := flag.CommandLine
	t.Cleanup(func() {
		os.Args = oldArgs
		flag.CommandLine = oldCommandLine
	})

	// Reset flag.CommandLine for this test
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	os.Args = []string{"purlinfo", "not-a-valid-purl"}

	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	exitCode := run()

	_ = w.Close()
	os.Stderr = oldStderr

	if exitCode != exitInvalidPurl {
		t.Errorf("run() with invalid purl returned exit code %d, want %d", exitCode, exitInvalidPurl)
	}

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if !strings.Contains(output, "Invalid purl format") {
		t.Errorf("run() invalid purl output = %q, want to contain 'Invalid purl format'", output)
	}
}

// TestRun_TooManyArguments tests the run function with too many arguments.
func TestRun_TooManyArguments(t *testing.T) {
	// Note: Cannot use t.Parallel() because run() modifies global flag.CommandLine

	// Save and restore os.Args and flag.CommandLine
	oldArgs := os.Args
	oldCommandLine := flag.CommandLine
	t.Cleanup(func() {
		os.Args = oldArgs
		flag.CommandLine = oldCommandLine
	})

	// Reset flag.CommandLine for this test
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	os.Args = []string{"purlinfo", "pkg:npm/test@1.0.0", "extra-arg"}

	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	exitCode := run()

	_ = w.Close()
	os.Stderr = oldStderr

	if exitCode != exitInvalidArgs {
		t.Errorf("run() with too many args returned exit code %d, want %d", exitCode, exitInvalidArgs)
	}

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if !strings.Contains(output, "Too many arguments") {
		t.Errorf("run() too many args output = %q, want to contain 'Too many arguments'", output)
	}
}

// TestRunWithService_Success tests the runWithService function with a successful mock service.
func TestRunWithService_Success(t *testing.T) {
	// Note: Cannot use t.Parallel() because test modifies global os.Stdout

	// Create mock service that returns success.
	mockSvc := &mockService{
		info: PackageInfo{
			Name:     "test-package",
			Version:  "1.0.0",
			Licenses: []string{"MIT"},
		},
		err: nil,
	}

	// Parse a valid purl.
	purl, err := packageurl.FromString("pkg:npm/test@1.0.0")
	if err != nil {
		t.Fatalf("failed to parse purl: %v", err)
	}

	// Setup logger.
	logger := setupLogger(false)

	// Capture stdout.
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Call runWithService with mock.
	exitCode := runWithService(mockSvc, logger, purl, "pkg:npm/test@1.0.0", false, false, 30*time.Second)

	_ = w.Close()
	os.Stdout = oldStdout

	// Verify exit code.
	if exitCode != exitSuccess {
		t.Errorf("runWithService() = %d, want %d", exitCode, exitSuccess)
	}

	// Verify output.
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	expectedStrings := []string{"test-package", "1.0.0", "MIT"}
	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("output missing %q\nGot: %s", expected, output)
		}
	}
}

// TestRunWithService_JSONOutput tests the runWithService function with JSON output.
func TestRunWithService_JSONOutput(t *testing.T) {
	// Note: Cannot use t.Parallel() because test modifies global os.Stdout

	// Create mock service.
	mockSvc := &mockService{
		info: PackageInfo{
			Name:     "json-test",
			Version:  "2.0.0",
			Licenses: []string{"Apache-2.0", "MIT"},
		},
		err: nil,
	}

	purl, _ := packageurl.FromString("pkg:npm/test@2.0.0")
	logger := setupLogger(false)

	// Capture stdout.
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Call with JSON output enabled.
	exitCode := runWithService(mockSvc, logger, purl, "pkg:npm/test@2.0.0", false, true, 30*time.Second)

	_ = w.Close()
	os.Stdout = oldStdout

	if exitCode != exitSuccess {
		t.Errorf("runWithService() = %d, want %d", exitCode, exitSuccess)
	}

	// Verify JSON output.
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	var result PackageInfo
	if jsonErr := json.Unmarshal([]byte(output), &result); jsonErr != nil {
		t.Errorf("runWithService() produced invalid JSON: %v\nOutput: %s", jsonErr, output)
	}

	if result.Name != "json-test" || result.Version != "2.0.0" {
		t.Errorf("runWithService() JSON = %+v, want name=json-test version=2.0.0", result)
	}
}

// TestRunWithService_ServiceError tests the runWithService function when service returns an error.
func TestRunWithService_ServiceError(t *testing.T) {
	// Note: Cannot use t.Parallel() because test modifies global os.Stderr

	// Create mock service that returns error.
	mockSvc := &mockService{
		err: errors.New("service error: package not found"),
	}

	purl, _ := packageurl.FromString("pkg:npm/test@1.0.0")
	logger := setupLogger(false)

	// Capture stderr.
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	exitCode := runWithService(mockSvc, logger, purl, "pkg:npm/test@1.0.0", false, false, 30*time.Second)

	_ = w.Close()
	os.Stderr = oldStderr

	// Verify exit code.
	if exitCode != exitRuntimeError {
		t.Errorf("runWithService() = %d, want %d", exitCode, exitRuntimeError)
	}

	// Verify error message in stderr.
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if !strings.Contains(output, "Failed to get package info") {
		t.Errorf("output missing error message\nGot: %s", output)
	}
}

// TestRunWithService_ServiceErrorVerbose tests error output in verbose mode.
func TestRunWithService_ServiceErrorVerbose(t *testing.T) {
	// Note: Cannot use t.Parallel() because test modifies global os.Stderr

	// Create mock service with specific error.
	mockSvc := &mockService{
		err: errors.New("specific error message"),
	}

	purl, _ := packageurl.FromString("pkg:npm/test@1.0.0")
	logger := setupLogger(true)

	// Capture stderr.
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	// Call with verbose=true.
	exitCode := runWithService(mockSvc, logger, purl, "pkg:npm/test@1.0.0", true, false, 30*time.Second)

	_ = w.Close()
	os.Stderr = oldStderr

	if exitCode != exitRuntimeError {
		t.Errorf("runWithService() = %d, want %d", exitCode, exitRuntimeError)
	}

	// In verbose mode, should include the actual error.
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if !strings.Contains(output, "specific error message") {
		t.Errorf("verbose output missing specific error\nGot: %s", output)
	}
}
