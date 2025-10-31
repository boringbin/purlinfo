package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/package-url/packageurl-go"
)

// TestNewEcosystemsService tests the NewEcosystemsService function.
func TestNewEcosystemsService(t *testing.T) {
	t.Parallel()

	t.Run("default options", func(t *testing.T) {
		t.Parallel()

		service := NewEcosystemsService(EcosystemsServiceOptions{})

		if service.baseURL != ecosystemsBaseURL {
			t.Errorf("baseURL = %q, want %q", service.baseURL, ecosystemsBaseURL)
		}
		if service.client != http.DefaultClient {
			t.Error("client should be http.DefaultClient when not provided")
		}
	})

	t.Run("custom base URL", func(t *testing.T) {
		t.Parallel()

		customURL := "https://example.com"
		service := NewEcosystemsService(EcosystemsServiceOptions{
			BaseURL: customURL,
		})

		if service.baseURL != customURL {
			t.Errorf("baseURL = %q, want %q", service.baseURL, customURL)
		}
	})

	t.Run("custom HTTP client", func(t *testing.T) {
		t.Parallel()

		customClient := &http.Client{Timeout: 5 * time.Second}
		service := NewEcosystemsService(EcosystemsServiceOptions{
			Client: customClient,
		})

		if service.client != customClient {
			t.Error("client should be the provided custom client")
		}
	})
}

// TestEcosystemsService_GetPackageInfo tests the GetPackageInfo method.
func TestEcosystemsService_GetPackageInfo(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		mockResponse   string
		mockStatusCode int
		purl           string
		want           PackageInfo
		wantErr        bool
		errContains    string
	}{
		{
			name: "success with licenses",
			mockResponse: `[{
				"name": "lodash",
				"latest_release_number": "4.17.21",
				"normalized_licenses": ["MIT"],
				"homepage": "https://lodash.com/",
				"repository_url": "https://github.com/lodash/lodash",
				"description": "Lodash modular utilities.",
				"documentation_url": "https://lodash.com/docs"
			}]`,
			mockStatusCode: http.StatusOK,
			purl:           "pkg:npm/lodash@4.17.21",
			want: PackageInfo{
				Name:             "lodash",
				Version:          "4.17.21",
				Licenses:         []string{"MIT"},
				Homepage:         stringPtr("https://lodash.com/"),
				RepositoryURL:    stringPtr("https://github.com/lodash/lodash"),
				Description:      stringPtr("Lodash modular utilities."),
				Ecosystem:        "npm",
				DocumentationURL: stringPtr("https://lodash.com/docs"),
			},
			wantErr: false,
		},
		{
			name: "success with multiple licenses",
			mockResponse: `[{
				"name": "requests",
				"latest_release_number": "2.32.5",
				"normalized_licenses": ["Apache-2.0", "MIT"],
				"homepage": "https://requests.readthedocs.io",
				"repository_url": "https://github.com/psf/requests",
				"description": "Python HTTP for Humans."
			}]`,
			mockStatusCode: http.StatusOK,
			purl:           "pkg:pypi/requests@2.28.0",
			want: PackageInfo{
				Name:             "requests",
				Version:          "2.32.5",
				Licenses:         []string{"Apache-2.0", "MIT"},
				Homepage:         stringPtr("https://requests.readthedocs.io"),
				RepositoryURL:    stringPtr("https://github.com/psf/requests"),
				Description:      stringPtr("Python HTTP for Humans."),
				Ecosystem:        "pypi",
				DocumentationURL: nil,
			},
			wantErr: false,
		},
		{
			name: "success with no licenses",
			mockResponse: `[{
				"name": "testpkg",
				"latest_release_number": "1.0.0",
				"normalized_licenses": []
			}]`,
			mockStatusCode: http.StatusOK,
			purl:           "pkg:npm/testpkg@1.0.0",
			want: PackageInfo{
				Name:             "testpkg",
				Version:          "1.0.0",
				Licenses:         []string{},
				Homepage:         nil,
				RepositoryURL:    nil,
				Description:      nil,
				Ecosystem:        "npm",
				DocumentationURL: nil,
			},
			wantErr: false,
		},
		{
			name:           "empty results",
			mockResponse:   `[]`,
			mockStatusCode: http.StatusOK,
			purl:           "pkg:npm/nonexistent@1.0.0",
			wantErr:        true,
			errContains:    "package not found",
		},
		{
			name:           "HTTP 404 error",
			mockResponse:   `{"error": "not found"}`,
			mockStatusCode: http.StatusNotFound,
			purl:           "pkg:npm/test@1.0.0",
			wantErr:        true,
			errContains:    "package not found",
		},
		{
			name:           "HTTP 500 error",
			mockResponse:   `{"error": "internal server error"}`,
			mockStatusCode: http.StatusInternalServerError,
			purl:           "pkg:npm/test@1.0.0",
			wantErr:        true,
			errContains:    "API error",
		},
		{
			name:           "malformed JSON",
			mockResponse:   `[{invalid json}]`,
			mockStatusCode: http.StatusOK,
			purl:           "pkg:npm/test@1.0.0",
			wantErr:        true,
			errContains:    "invalid API response",
		},
		{
			name:           "not an array",
			mockResponse:   `{"name": "test"}`,
			mockStatusCode: http.StatusOK,
			purl:           "pkg:npm/test@1.0.0",
			wantErr:        true,
			errContains:    "invalid API response",
		},
		{
			name:           "HTTP 429 rate limit error",
			mockResponse:   `{"error": "too many requests"}`,
			mockStatusCode: http.StatusTooManyRequests,
			purl:           "pkg:npm/test@1.0.0",
			wantErr:        true,
			errContains:    "rate limited",
		},
		{
			name:           "HTTP 502 bad gateway error",
			mockResponse:   `{"error": "bad gateway"}`,
			mockStatusCode: http.StatusBadGateway,
			purl:           "pkg:npm/test@1.0.0",
			wantErr:        true,
			errContains:    "service unavailable",
		},
		{
			name:           "HTTP 503 service unavailable error",
			mockResponse:   `{"error": "service unavailable"}`,
			mockStatusCode: http.StatusServiceUnavailable,
			purl:           "pkg:npm/test@1.0.0",
			wantErr:        true,
			errContains:    "service unavailable",
		},
		{
			name:           "HTTP 504 gateway timeout error",
			mockResponse:   `{"error": "gateway timeout"}`,
			mockStatusCode: http.StatusGatewayTimeout,
			purl:           "pkg:npm/test@1.0.0",
			wantErr:        true,
			errContains:    "service unavailable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create mock server.
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify the request.
				if r.Method != http.MethodGet {
					t.Errorf("expected GET request, got %s", r.Method)
				}

				// Check that purl query parameter exists.
				if r.URL.Query().Get("purl") == "" {
					t.Error("expected purl query parameter")
				}

				// Send mock response.
				w.WriteHeader(tt.mockStatusCode)
				_, _ = w.Write([]byte(tt.mockResponse))
			}))
			t.Cleanup(server.Close)

			// Create service with mock server URL.
			service := NewEcosystemsService(EcosystemsServiceOptions{
				BaseURL: server.URL,
			})

			// Parse purl.
			purl, err := packageurl.FromString(tt.purl)
			if err != nil {
				t.Fatalf("failed to parse purl: %v", err)
			}

			// Call GetPackageInfo.
			ctx := context.Background()
			got, err := service.GetPackageInfo(ctx, purl)

			// Check error.
			if tt.wantErr {
				if err == nil {
					t.Errorf("GetPackageInfo() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("GetPackageInfo() error = %q, want error containing %q", err.Error(), tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("GetPackageInfo() unexpected error = %v", err)
				return
			}

			// Check result.
			if got.Name != tt.want.Name {
				t.Errorf("GetPackageInfo() Name = %q, want %q", got.Name, tt.want.Name)
			}
			if got.Version != tt.want.Version {
				t.Errorf("GetPackageInfo() Version = %q, want %q", got.Version, tt.want.Version)
			}
			if got.Ecosystem != tt.want.Ecosystem {
				t.Errorf("GetPackageInfo() Ecosystem = %q, want %q", got.Ecosystem, tt.want.Ecosystem)
			}
			if !equalStringSlices(got.Licenses, tt.want.Licenses) {
				t.Errorf("GetPackageInfo() Licenses = %v, want %v", got.Licenses, tt.want.Licenses)
			}
			if !equalStringPtrs(got.Homepage, tt.want.Homepage) {
				t.Errorf(
					"GetPackageInfo() Homepage = %v, want %v",
					stringPtrToString(got.Homepage),
					stringPtrToString(tt.want.Homepage),
				)
			}
			if !equalStringPtrs(got.RepositoryURL, tt.want.RepositoryURL) {
				t.Errorf(
					"GetPackageInfo() RepositoryURL = %v, want %v",
					stringPtrToString(got.RepositoryURL),
					stringPtrToString(tt.want.RepositoryURL),
				)
			}
			if !equalStringPtrs(got.Description, tt.want.Description) {
				t.Errorf(
					"GetPackageInfo() Description = %v, want %v",
					stringPtrToString(got.Description),
					stringPtrToString(tt.want.Description),
				)
			}
			if !equalStringPtrs(got.DocumentationURL, tt.want.DocumentationURL) {
				t.Errorf(
					"GetPackageInfo() DocumentationURL = %v, want %v",
					stringPtrToString(got.DocumentationURL),
					stringPtrToString(tt.want.DocumentationURL),
				)
			}
		})
	}
}

// TestEcosystemsService_GetPackageInfo_ContextCancellation tests the GetPackageInfo method with a cancelled context.
func TestEcosystemsService_GetPackageInfo_ContextCancellation(t *testing.T) {
	t.Parallel()

	// Create a server that delays response.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[{"name":"test","latest_release_number":"1.0.0","normalized_licenses":[]}]`))
	}))
	t.Cleanup(server.Close)

	service := NewEcosystemsService(EcosystemsServiceOptions{
		BaseURL: server.URL,
	})

	purl, err := packageurl.FromString("pkg:npm/test@1.0.0")
	if err != nil {
		t.Fatalf("failed to parse purl: %v", err)
	}

	// Create context that will be cancelled.
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	_, err = service.GetPackageInfo(ctx, purl)
	if err == nil {
		t.Error("GetPackageInfo() with cancelled context should return error")
	}
}

// TestEcosystemsService_GetPackageInfo_Timeout tests the GetPackageInfo method with a timeout.
func TestEcosystemsService_GetPackageInfo_Timeout(t *testing.T) {
	t.Parallel()

	// Create a server that delays response.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[{"name":"test","latest_release_number":"1.0.0","normalized_licenses":[]}]`))
	}))
	t.Cleanup(server.Close)

	service := NewEcosystemsService(EcosystemsServiceOptions{
		BaseURL: server.URL,
		Client:  &http.Client{Timeout: 50 * time.Millisecond},
	})

	purl, err := packageurl.FromString("pkg:npm/test@1.0.0")
	if err != nil {
		t.Fatalf("failed to parse purl: %v", err)
	}

	ctx := context.Background()
	_, err = service.GetPackageInfo(ctx, purl)
	if err == nil {
		t.Error("GetPackageInfo() with timeout should return error")
	}
}

// contains checks if a string contains a substring.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && containsHelper(s, substr)))
}

// containsHelper is a helper function to check if a string contains a substring.
func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// equalStringSlices compares string slices.
func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// stringPtr returns a pointer to a string.
func stringPtr(s string) *string {
	return &s
}

// equalStringPtrs compares two string pointers.
func equalStringPtrs(a, b *string) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

// stringPtrToString converts a string pointer to a string for display.
func stringPtrToString(s *string) string {
	if s == nil {
		return "<nil>"
	}
	return *s
}
