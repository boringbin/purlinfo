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
}

// Service is the interface that each service must implement.
type Service interface {
	// GetPackageInfo returns the information about a package.
	GetPackageInfo(ctx context.Context, purl packageurl.PackageURL) (PackageInfo, error)
}
