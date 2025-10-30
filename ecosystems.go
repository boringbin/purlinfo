package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/package-url/packageurl-go"
)

const (
	// ecosystemsBaseURL is the base URL for the Ecosystems API.
	//
	// See https://packages.ecosyste.ms/docs/index.html
	ecosystemsBaseURL = "https://packages.ecosyste.ms"
	// ecosystemsAPIPath is the API path for package lookup.
	ecosystemsAPIPath = "/api/v1/packages/lookup"
)

// EcosystemsService is the service for the Ecosystems API.
type EcosystemsService struct {
	baseURL string
	client  *http.Client
}

var _ Service = (*EcosystemsService)(nil)

// EcosystemsServiceOptions are the options for the EcosystemsService.
type EcosystemsServiceOptions struct {
	// BaseURL is the base URL for the Ecosystems API.
	// If empty, defaults to the public Ecosystems API.
	BaseURL string
	// Client is the HTTP client to use for the Ecosystems API.
	// If nil, defaults to http.DefaultClient.
	Client *http.Client
}

// NewEcosystemsService creates a new EcosystemsService.
func NewEcosystemsService(opts EcosystemsServiceOptions) *EcosystemsService {
	// Default to the Ecosystems API base URL.
	baseURL := ecosystemsBaseURL
	if opts.BaseURL != "" {
		baseURL = opts.BaseURL
	}
	// Default to the default HTTP client.
	client := opts.Client
	if client == nil {
		client = http.DefaultClient
	}

	return &EcosystemsService{
		baseURL: baseURL,
		client:  client,
	}
}

// ecosystemsPackagesLookupResponse is the response from the Ecosystems API.
type ecosystemsPackagesLookupResponse struct {
	Name                string   `json:"name"`
	LatestReleaseNumber string   `json:"latest_release_number"`
	NormalizedLicenses  []string `json:"normalized_licenses"`
}

// GetPackageInfo returns the information about a package.
func (s *EcosystemsService) GetPackageInfo(ctx context.Context, purl packageurl.PackageURL) (PackageInfo, error) {
	apiURL := fmt.Sprintf("%s%s?purl=%s", s.baseURL, ecosystemsAPIPath, url.QueryEscape(purl.String()))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return PackageInfo{}, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	response, err := s.client.Do(req)
	if err != nil {
		return PackageInfo{}, fmt.Errorf("failed to make HTTP request: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		switch response.StatusCode {
		case http.StatusNotFound:
			return PackageInfo{}, fmt.Errorf("%w: HTTP 404", ErrPackageNotFound)
		case http.StatusTooManyRequests:
			return PackageInfo{}, errors.New("rate limited by API: HTTP 429")
		case http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
			return PackageInfo{}, fmt.Errorf("API service unavailable: HTTP %d", response.StatusCode)
		default:
			return PackageInfo{}, fmt.Errorf("API error: HTTP %d", response.StatusCode)
		}
	}

	// Parse the response (it's an array)
	var results []ecosystemsPackagesLookupResponse
	err = json.NewDecoder(response.Body).Decode(&results)
	if err != nil {
		return PackageInfo{}, fmt.Errorf("%w: %w", ErrInvalidResponse, err)
	}

	// Check if we got any results
	if len(results) == 0 {
		return PackageInfo{}, fmt.Errorf("%w: %s", ErrPackageNotFound, purl.String())
	}

	// Get the first result
	result := results[0]

	// Convert the response to the PackageInfo struct
	packageInfo := PackageInfo{
		Name:     result.Name,
		Version:  result.LatestReleaseNumber,
		Licenses: result.NormalizedLicenses,
	}

	return packageInfo, nil
}
