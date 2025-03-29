package shortener_test

import (
	"context"
	"os"
	"testing"

	"github.com/prasetyowira/shorter/constant"
	"github.com/prasetyowira/shorter/domain/shortener"
	"github.com/prasetyowira/shorter/infrastructure/cache"
	"github.com/prasetyowira/shorter/infrastructure/db"
	"github.com/stretchr/testify/assert"
)

const testDBPath = "test_integration.db"

// Helper function to clean up test database
func cleanupIntegrationTestDB(t *testing.T) {
	err := os.Remove(testDBPath)
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("Failed to clean up test database: %v", err)
	}
}

// Helper function to create a test service with real SQLite repository
func createIntegrationTestService(t *testing.T) *shortener.Service {
	cleanupIntegrationTestDB(t)
	
	cacheLRU := cache.NewNamespaceLRU(100)
	repo, err := db.NewSQLiteRepository(testDBPath, cacheLRU)
	if err != nil {
		t.Fatalf("Failed to create test repository: %v", err)
	}
	
	return shortener.NewService(repo, cacheLRU)
}

func TestIntegration_UpdateLongURL(t *testing.T) {
	// Skip in CI environment
	if os.Getenv("CI") == "true" {
		t.Skip("Skipping integration test in CI environment")
	}
	
	// Arrange
	service := createIntegrationTestService(t)
	defer cleanupIntegrationTestDB(t)
	ctx := context.Background()
	
	// First create a URL
	originalURL := "https://example.com"
	shortCode := "abc123"
	
	// Creating a URL with defined short code for testing
	url, err := service.CreateShortURL(ctx, originalURL, shortCode)
	assert.NoError(t, err)
	assert.Equal(t, shortCode, url.ShortCode)
	assert.Equal(t, originalURL, url.LongURL)
	assert.Equal(t, uint(0), url.Visits) // Initially 0 visits
	
	// Act - Update the long URL
	newLongURL := "https://example.com/updated"
	updatedURL, err := service.UpdateLongURL(ctx, shortCode, newLongURL)
	
	// Assert
	assert.NoError(t, err)
	assert.Equal(t, newLongURL, updatedURL.LongURL)
	assert.Equal(t, shortCode, updatedURL.ShortCode)
	// Visits should still be 0 after update
	assert.Equal(t, uint(0), updatedURL.Visits)
	
	// Verify that the update is persisted by getting the URL again
	retrievedURL, err := service.GetLongURL(ctx, shortCode)
	assert.NoError(t, err)
	assert.Equal(t, newLongURL, retrievedURL.LongURL)
	assert.Equal(t, shortCode, retrievedURL.ShortCode)
	
	// GetLongURL increments the visit counter, so it should now be 1
	assert.Equal(t, uint(1), retrievedURL.Visits)
}

func TestIntegration_UpdateLongURL_NotFound(t *testing.T) {
	// Skip in CI environment
	if os.Getenv("CI") == "true" {
		t.Skip("Skipping integration test in CI environment")
	}
	
	// Arrange
	service := createIntegrationTestService(t)
	defer cleanupIntegrationTestDB(t)
	ctx := context.Background()
	
	// Act - Try to update a non-existent URL
	updatedURL, err := service.UpdateLongURL(ctx, "nonexistent", "https://example.com/updated")
	
	// Assert
	assert.Error(t, err)
	assert.Nil(t, updatedURL)
}

func TestIntegration_UpdateLongURL_EmptyShortCode(t *testing.T) {
	// Skip in CI environment
	if os.Getenv("CI") == "true" {
		t.Skip("Skipping integration test in CI environment")
	}
	
	// Arrange
	service := createIntegrationTestService(t)
	defer cleanupIntegrationTestDB(t)
	ctx := context.Background()
	
	// Act - Try to update with empty short code
	updatedURL, err := service.UpdateLongURL(ctx, "", "https://example.com/updated")
	
	// Assert
	assert.Error(t, err)
	assert.Equal(t, constant.ErrEmptyShortCode, err.Error())
	assert.Nil(t, updatedURL)
}

func TestIntegration_UpdateLongURL_EmptyLongURL(t *testing.T) {
	// Skip in CI environment
	if os.Getenv("CI") == "true" {
		t.Skip("Skipping integration test in CI environment")
	}
	
	// Arrange
	service := createIntegrationTestService(t)
	defer cleanupIntegrationTestDB(t)
	ctx := context.Background()
	
	// First create a URL
	originalURL := "https://example.com"
	shortCode := "abc123"
	
	// Creating a URL with defined short code for testing
	_, err := service.CreateShortURL(ctx, originalURL, shortCode)
	assert.NoError(t, err)
	
	// Act - Try to update with empty long URL
	updatedURL, err := service.UpdateLongURL(ctx, shortCode, "")
	
	// Assert
	assert.Error(t, err)
	assert.Equal(t, constant.ErrEmptyLongURL, err.Error())
	assert.Nil(t, updatedURL)
	
	// Verify the original URL is still intact
	retrievedURL, err := service.GetLongURL(ctx, shortCode)
	assert.NoError(t, err)
	assert.Equal(t, originalURL, retrievedURL.LongURL)
}

func TestIntegration_UpdateLongURL_Cache(t *testing.T) {
	// Skip in CI environment
	if os.Getenv("CI") == "true" {
		t.Skip("Skipping integration test in CI environment")
	}
	
	// Arrange
	cacheLRU := cache.NewNamespaceLRU(100)
	repo, err := db.NewSQLiteRepository(testDBPath, cacheLRU)
	if err != nil {
		t.Fatalf("Failed to create test repository: %v", err)
	}
	defer cleanupIntegrationTestDB(t)
	
	service := shortener.NewService(repo, cacheLRU)
	ctx := context.Background()
	
	// First create a URL
	originalURL := "https://example.com"
	shortCode := "abc123"
	
	// Creating a URL with defined short code for testing
	_, err = service.CreateShortURL(ctx, originalURL, shortCode)
	assert.NoError(t, err)
	
	// Get the URL to populate cache
	_, err = service.GetLongURL(ctx, shortCode)
	assert.NoError(t, err)
	
	// Verify URL is in cache
	cachedURL, found := cacheLRU.Get(constant.ShortURLNamespace, shortCode)
	assert.True(t, found, "URL should be in cache")
	assert.Equal(t, originalURL, cachedURL.(*shortener.URL).LongURL)
	
	// Act - Update the long URL
	newLongURL := "https://example.com/updated"
	_, err = service.UpdateLongURL(ctx, shortCode, newLongURL)
	assert.NoError(t, err)
	
	// Verify cache was updated
	updatedCachedURL, found := cacheLRU.Get(constant.ShortURLNamespace, shortCode)
	assert.True(t, found, "URL should still be in cache after update")
	assert.Equal(t, newLongURL, updatedCachedURL.(*shortener.URL).LongURL)
} 