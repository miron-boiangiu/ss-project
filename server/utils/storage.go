package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func SaveToLocal(data []byte, keyName string) error {
	// keyName is like "photos/123.png"
	// we want to save to "./uploads/photos/123.png"
	
	path := filepath.Join("uploads", keyName)
	
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}
	
	return nil
}

func GetLocalURL(keyName string) string {
	// construct URL assuming the server serves /uploads/ at root
	// e.g. http://localhost:8080/uploads/photos/123.png
	// The frontend runs on a different port, so we need a full URL
	// Get the server's external URL from environment or use default
	baseURL := os.Getenv("API_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}
	// Remove trailing slash if present
	baseURL = strings.TrimSuffix(baseURL, "/")
	return fmt.Sprintf("%s/uploads/%s", baseURL, keyName)
}
