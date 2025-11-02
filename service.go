package main

import (
	"context"
	"errors"

	"github.com/package-url/packageurl-go"
)

var (
	// ErrPackageNotFound is returned when a package is not found.
	ErrPackageNotFound = errors.New("package not found")
	// ErrInvalidResponse is returned when the API response is invalid.
	ErrInvalidResponse = errors.New("invalid API response")
)

// PackageInfo represents the information about a package.
//
// Each service should return this information.
type PackageInfo struct {
	// The name of the package.
	Name string `json:"name"`
	// The version of the package.
	Version string `json:"version"`
	// The licenses of the package.
	Licenses []string `json:"licenses"`
	// The homepage URL of the package (empty string if not available).
	Homepage string `json:"homepage,omitempty"`
	// The repository URL of the package (empty string if not available).
	RepositoryURL string `json:"repository_url,omitempty"`
	// The description of the package (empty string if not available).
	Description string `json:"description,omitempty"`
	// The ecosystem/type of the package (e.g., npm, pypi, cargo).
	Ecosystem string `json:"ecosystem"`
	// The documentation URL of the package (empty string if not available).
	DocumentationURL string `json:"documentation_url,omitempty"`
}

// Service is the interface that each service must implement.
type Service interface {
	// GetPackageInfo returns the information about a package.
	GetPackageInfo(ctx context.Context, purl packageurl.PackageURL) (PackageInfo, error)
}
