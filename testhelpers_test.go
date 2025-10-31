package main

// stringPtr returns a pointer to a string.
// This helper is used across multiple test files.
func stringPtr(s string) *string {
	return &s
}
