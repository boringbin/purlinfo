// Package main provides the `purlinfo` CLI.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/package-url/packageurl-go"
)

const (
	// version is the version of the `purlinfo` CLI.
	version = "0.1.0-dev"
	// exitSuccess is the exit code for success.
	exitSuccess = 0
	// exitInvalidArgs is the exit code for invalid arguments.
	exitInvalidArgs = 1
	// exitInvalidPurl is the exit code for invalid purl.
	exitInvalidPurl = 2
	// exitRuntimeError is the exit code for runtime error.
	exitRuntimeError = 3
	// defaultTimeoutSec is the default timeout in seconds.
	defaultTimeoutSec = 30
)

func main() {
	os.Exit(run())
}

func run() int {
	var (
		outputJSON  = flag.Bool("json", false, "Output as JSON")
		verbose     = flag.Bool("v", false, "Verbose output (debug mode)")
		showVersion = flag.Bool("version", false, "Show version and exit")
		timeout     = flag.Duration("timeout", defaultTimeoutSec*time.Second, "HTTP request timeout")
		email       = flag.String("email", "", "Email for polite pool (optional)")
	)

	// Customize usage message
	printUsageFunc := func() {
		printUsage()
	}
	flag.CommandLine.Usage = printUsageFunc

	flag.Parse()

	// Handle version flag
	if *showVersion {
		fmt.Fprintf(os.Stdout, "purlinfo version %s\n", version)
		return exitSuccess
	}

	// Setup logger based on verbose flag
	logger := setupLogger(*verbose)

	// Get the purl from remaining arguments
	args := flag.Args()
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Error: purl argument is required\n\n")
		printUsage()
		return exitInvalidArgs
	}
	if len(args) > 1 {
		fmt.Fprintf(os.Stderr, "Error: Too many arguments. Expected 1 purl, got %d\n\n", len(args))
		printUsage()
		return exitInvalidArgs
	}

	purlString := args[0]

	// Parse the purl
	logger.Debug("parsing purl", "purl", purlString)
	purl, err := packageurl.FromString(purlString)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Invalid purl format: %v\n", err)
		return exitInvalidPurl
	}

	// Create HTTP client with timeout
	httpClient := &http.Client{
		Timeout: *timeout,
	}

	// Create service
	service := createService(httpClient, *email)

	// Delegate to runWithService for the core logic
	return runWithService(service, logger, purl, purlString, *verbose, *outputJSON, *timeout)
}

// runWithService contains the core logic for fetching and displaying package info.
// This function is separated to enable testing with mock services.
func runWithService(
	service Service,
	logger *slog.Logger,
	purl packageurl.PackageURL,
	purlString string,
	verbose bool,
	outputJSON bool,
	timeout time.Duration,
) int {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Get package info
	logger.Debug("fetching package info", "purl", purlString)
	info, err := service.GetPackageInfo(ctx, purl)
	if err != nil {
		if verbose {
			fmt.Fprintf(os.Stderr, "Error: Failed to get package info: %v\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "Error: Failed to get package info\n")
			fmt.Fprintf(os.Stderr, "Use -v flag for more details\n")
		}
		return exitRuntimeError
	}

	// Output the result
	if printErr := printOutput(info, outputJSON); printErr != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", printErr)
		return exitRuntimeError
	}

	return exitSuccess
}

// printUsage prints the usage message.
func printUsage() {
	fmt.Fprintf(os.Stderr, "Usage: %s [OPTIONS] purl\n\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "Get package information from a package URL (purl).\n\n")
	fmt.Fprintf(os.Stderr, "Arguments:\n")
	fmt.Fprintf(os.Stderr, "  purl    Package URL (e.g., pkg:npm/lodash@4.17.21)\n\n")
	fmt.Fprintf(os.Stderr, "Options:\n")
	flag.PrintDefaults()
}

// setupLogger sets up the logger based on the verbose flag.
func setupLogger(verbose bool) *slog.Logger {
	logLevel := slog.LevelError
	if verbose {
		// If verbose is true, set the log level to debug
		// This will log all messages, including debug messages
		logLevel = slog.LevelDebug
	}
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: logLevel,
	}))
}

// createService creates the service.
func createService(httpClient *http.Client, email string) Service {
	return NewEcosystemsService(EcosystemsServiceOptions{
		Client: httpClient,
		Email:  email,
	})
}

// printOutput prints the output based on the outputJSON flag.
func printOutput(info PackageInfo, outputJSON bool) error {
	if outputJSON {
		return printJSONOutput(info)
	}
	return printHumanReadableOutput(info)
}

// printJSONOutput prints the package info as JSON.
func printJSONOutput(info PackageInfo) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if encodeErr := encoder.Encode(info); encodeErr != nil {
		return fmt.Errorf("failed to encode JSON: %w", encodeErr)
	}
	return nil
}

// printHumanReadableOutput prints the package info in human-readable format.
func printHumanReadableOutput(info PackageInfo) error {
	fmt.Fprintf(os.Stdout, "Name:            %s\n", info.Name)
	fmt.Fprintf(os.Stdout, "Version:         %s\n", info.Version)
	fmt.Fprintf(os.Stdout, "Ecosystem:       %s\n", info.Ecosystem)

	printLicenses(info.Licenses)
	printOptionalField("Description:", info.Description)
	printOptionalField("Homepage:", info.Homepage)
	printOptionalField("RepositoryURL:", info.RepositoryURL)
	printOptionalField("DocumentationURL:", info.DocumentationURL)

	return nil
}

// printLicenses prints the licenses field.
func printLicenses(licenses []string) {
	if len(licenses) > 0 {
		fmt.Fprintf(os.Stdout, "Licenses:        %s\n", strings.Join(licenses, ", "))
	} else {
		fmt.Fprintf(os.Stdout, "Licenses:        (none)\n")
	}
}

// printOptionalField prints an optional field (nullable string pointer).
func printOptionalField(label string, value *string) {
	// labelColumnWidth is set to 17 to match the longest label "DocumentationURL:" (17 chars).
	// This ensures all field values are aligned at the same column.
	const labelColumnWidth = 17
	padding := labelColumnWidth - len(label)

	if value != nil && *value != "" {
		fmt.Fprintf(os.Stdout, "%s%*s%s\n", label, padding, "", *value)
	} else {
		fmt.Fprintf(os.Stdout, "%s%*s(none)\n", label, padding, "")
	}
}
