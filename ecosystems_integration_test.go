//go:build integration

package main

import (
	"context"
	"testing"
	"time"

	"github.com/package-url/packageurl-go"
)

// TestEcosystemsService_Integration tests the actual Ecosyste.ms API.
func TestEcosystemsService_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tests := []struct {
		name        string
		purl        string
		wantName    string
		wantVersion string // This might change as packages are updated
		wantLicense string // At least one license should contain this
	}{
		{
			name:        "npm package",
			purl:        "pkg:npm/lodash@4.17.21",
			wantName:    "lodash",
			wantVersion: "4.17.21", // Latest version might be higher
			wantLicense: "MIT",
		},
		{
			name:        "pypi package",
			purl:        "pkg:pypi/requests@2.28.0",
			wantName:    "requests",
			wantLicense: "Apache",
		},
		{
			name:        "npm scoped package",
			purl:        "pkg:npm/%40types/node@18.0.0",
			wantName:    "@types/node",
			wantLicense: "MIT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create real service
			service := NewEcosystemsService(EcosystemsServiceOptions{})

			// Parse purl
			purl, err := packageurl.FromString(tt.purl)
			if err != nil {
				t.Fatalf("failed to parse purl: %v", err)
			}

			// Call with reasonable timeout
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			got, err := service.GetPackageInfo(ctx, purl)
			if err != nil {
				t.Fatalf("GetPackageInfo() error = %v", err)
			}

			// Verify name (should match exactly)
			if got.Name != tt.wantName {
				t.Errorf("GetPackageInfo() Name = %q, want %q", got.Name, tt.wantName)
			}

			// Verify version is not empty
			if got.Version == "" {
				t.Error("GetPackageInfo() Version is empty")
			}

			// If we specified a version to check, verify it
			if tt.wantVersion != "" && got.Version != tt.wantVersion {
				t.Logf("Note: Version = %q, expected %q (version may have been updated)", got.Version, tt.wantVersion)
			}

			// Verify at least one license contains expected string
			if tt.wantLicense != "" {
				found := false
				for _, lic := range got.Licenses {
					if contains(lic, tt.wantLicense) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("GetPackageInfo() Licenses = %v, want at least one containing %q", got.Licenses, tt.wantLicense)
				}
			}

			t.Logf("Successfully retrieved: %s v%s (licenses: %v)", got.Name, got.Version, got.Licenses)
		})
	}
}

// TestEcosystemsService_Integration_NotFound tests if the Ecosyste.ms API returns an error for a non-existent package.
func TestEcosystemsService_Integration_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	service := NewEcosystemsService(EcosystemsServiceOptions{})

	// Use a package that definitely doesn't exist
	purl, err := packageurl.FromString("pkg:npm/this-package-definitely-does-not-exist-12345@999.999.999")
	if err != nil {
		t.Fatalf("failed to parse purl: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err = service.GetPackageInfo(ctx, purl)
	if err == nil {
		t.Error("GetPackageInfo() for nonexistent package should return error")
	}

	if !contains(err.Error(), "package not found") {
		t.Errorf("GetPackageInfo() error = %q, want error containing 'package not found'", err.Error())
	}
}
