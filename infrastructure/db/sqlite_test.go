package db

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/prasetyowira/shorter/constant"
	"github.com/prasetyowira/shorter/domain/shortener"
	"github.com/prasetyowira/shorter/infrastructure/cache"
	"github.com/stretchr/testify/assert"
)

// testDBPath is the path to the test database file
const testDBPath = "test.db"

// Helper function to clean up test database
func cleanupTestDB(t *testing.T) {
	err := os.Remove(testDBPath)
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("Failed to clean up test database: %v", err)
	}
}

// Helper function to create a test repository
func createTestRepository(t *testing.T) *SQLiteRepository {
	cleanupTestDB(t)
	
	cacheLRU := cache.NewNamespaceLRU(100)
	repo, err := NewSQLiteRepository(testDBPath, cacheLRU)
	if err != nil {
		t.Fatalf("Failed to create test repository: %v", err)
	}
	
	return repo
}

func TestNewSQLiteRepository(t *testing.T) {
	// Cleanup after test
	defer cleanupTestDB(t)
	
	// Act
	cacheLRU := cache.NewNamespaceLRU(100)
	repo, err := NewSQLiteRepository(testDBPath, cacheLRU)
	
	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, repo)
	assert.NotNil(t, repo.db)
	
	// Clean up
	err = repo.Close()
	assert.NoError(t, err)
}

func TestNewSQLiteRepository_InvalidPath(t *testing.T) {
	// Act - Try to create a repository with an invalid path
	cacheLRU := cache.NewNamespaceLRU(100)
	repo, err := NewSQLiteRepository("/invalid/path/db.sqlite", cacheLRU)
	
	// Assert
	assert.Error(t, err)
	assert.Nil(t, repo)
}

func TestSQLiteRepository_Store(t *testing.T) {
	// Arrange
	repo := createTestRepository(t)
	defer cleanupTestDB(t)
	defer repo.Close()
	ctx := context.Background()
	
	url := &shortener.URL{
		LongURL:   "https://example.com",
		ShortCode: "abc123",
		CreatedAt: time.Now().Truncate(time.Second), // SQLite may not preserve nanoseconds
		Visits:    0,
	}
	
	// Act
	err := repo.Store(ctx, url)
	
	// Assert
	assert.NoError(t, err)
	
	// Verify that the URL was stored correctly
	foundURL, err := repo.FindByShortCode(ctx, url.ShortCode)
	assert.NoError(t, err)
	assert.NotNil(t, foundURL)
	assert.Equal(t, url.LongURL, foundURL.LongURL)
	assert.Equal(t, url.ShortCode, foundURL.ShortCode)
	assert.Equal(t, url.Visits, foundURL.Visits)
}

func TestSQLiteRepository_Store_DuplicateShortCode(t *testing.T) {
	// Arrange
	repo := createTestRepository(t)
	defer cleanupTestDB(t)
	defer repo.Close()
	ctx := context.Background()
	
	url1 := &shortener.URL{
		LongURL:   "https://example.com",
		ShortCode: "abc123",
		CreatedAt: time.Now(),
		Visits:    0,
	}
	
	url2 := &shortener.URL{
		LongURL:   "https://another-example.com",
		ShortCode: "abc123", // Same short code
		CreatedAt: time.Now(),
		Visits:    0,
	}
	
	// Act
	err1 := repo.Store(ctx, url1)
	err2 := repo.Store(ctx, url2)
	
	// Assert
	assert.NoError(t, err1)
	assert.Error(t, err2)
	assert.Equal(t, constant.ErrShortCodeExists, err2.Error())
}

func TestSQLiteRepository_FindByShortCode(t *testing.T) {
	// Arrange
	repo := createTestRepository(t)
	defer cleanupTestDB(t)
	defer repo.Close()
	ctx := context.Background()
	
	originalURL := &shortener.URL{
		LongURL:   "https://example.com",
		ShortCode: "abc123",
		CreatedAt: time.Now().Truncate(time.Second), // SQLite may not preserve nanoseconds
		Visits:    0,
	}
	
	err := repo.Store(ctx, originalURL)
	assert.NoError(t, err)
	
	// Act
	foundURL, err := repo.FindByShortCode(ctx, originalURL.ShortCode)
	
	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, foundURL)
	assert.Equal(t, originalURL.LongURL, foundURL.LongURL)
	assert.Equal(t, originalURL.ShortCode, foundURL.ShortCode)
	assert.Equal(t, originalURL.Visits, foundURL.Visits)
	// Not comparing CreatedAt as it may have minor differences due to storage
}

func TestSQLiteRepository_FindByShortCode_NotFound(t *testing.T) {
	// Arrange
	repo := createTestRepository(t)
	defer cleanupTestDB(t)
	defer repo.Close()
	ctx := context.Background()
	
	// Act
	foundURL, err := repo.FindByShortCode(ctx, "nonexistent")
	
	// Assert
	assert.Error(t, err)
	assert.Equal(t, constant.ErrShortCodeNotFound, err.Error())
	assert.Nil(t, foundURL)
}

func TestSQLiteRepository_IncrementVisits(t *testing.T) {
	// Arrange
	repo := createTestRepository(t)
	defer cleanupTestDB(t)
	defer repo.Close()
	ctx := context.Background()
	
	originalURL := &shortener.URL{
		LongURL:   "https://example.com",
		ShortCode: "abc123",
		CreatedAt: time.Now(),
		Visits:    0,
	}
	
	err := repo.Store(ctx, originalURL)
	assert.NoError(t, err)
	
	// Act - Increment visits
	err = repo.IncrementVisits(ctx, originalURL.ShortCode)
	assert.NoError(t, err)
	
	// Assert - Verify that visits were incremented
	foundURL, err := repo.FindByShortCode(ctx, originalURL.ShortCode)
	assert.NoError(t, err)
	assert.NotNil(t, foundURL)
	assert.Equal(t, uint(1), foundURL.Visits)
	
	// Act - Increment again
	err = repo.IncrementVisits(ctx, originalURL.ShortCode)
	assert.NoError(t, err)
	
	// Assert - Verify visits incremented to 2
	foundURL, err = repo.FindByShortCode(ctx, originalURL.ShortCode)
	assert.NoError(t, err)
	assert.Equal(t, uint(2), foundURL.Visits)
}

func TestSQLiteRepository_IncrementVisits_NonexistentShortCode(t *testing.T) {
	// Arrange
	repo := createTestRepository(t)
	defer cleanupTestDB(t)
	defer repo.Close()
	ctx := context.Background()
	
	// Act
	err := repo.IncrementVisits(ctx, "nonexistent")
	
	// Assert
	assert.NoError(t, err) // Should not return error, just log warning
}

func TestSQLiteRepository_Close(t *testing.T) {
	// Arrange
	repo := createTestRepository(t)
	defer cleanupTestDB(t)
	
	// Act
	err := repo.Close()
	
	// Assert
	assert.NoError(t, err)
}

func TestSQLiteRepository_UpdateLongURL(t *testing.T) {
	// Arrange
	repo := createTestRepository(t)
	defer cleanupTestDB(t)
	defer repo.Close()
	ctx := context.Background()
	
	// Create a URL to update
	originalURL := &shortener.URL{
		LongURL:   "https://example.com",
		ShortCode: "abc123",
		CreatedAt: time.Now().Truncate(time.Second),
		Visits:    0,
	}
	
	err := repo.Store(ctx, originalURL)
	assert.NoError(t, err)
	
	// Act - Update the long URL
	newLongURL := "https://example.com/updated"
	err = repo.UpdateLongURL(ctx, originalURL.ShortCode, newLongURL)
	
	// Assert
	assert.NoError(t, err)
	
	// Verify that the URL was updated
	foundURL, err := repo.FindByShortCode(ctx, originalURL.ShortCode)
	assert.NoError(t, err)
	assert.NotNil(t, foundURL)
	assert.Equal(t, newLongURL, foundURL.LongURL)
	assert.Equal(t, originalURL.ShortCode, foundURL.ShortCode)
	assert.Equal(t, originalURL.Visits, foundURL.Visits)
}

func TestSQLiteRepository_UpdateLongURL_NonexistentShortCode(t *testing.T) {
	// Arrange
	repo := createTestRepository(t)
	defer cleanupTestDB(t)
	defer repo.Close()
	ctx := context.Background()
	
	// Act - Try to update a nonexistent short code
	err := repo.UpdateLongURL(ctx, "nonexistent", "https://example.com/updated")
	
	// Assert
	assert.Error(t, err)
	assert.Equal(t, constant.ErrShortCodeNotFound, err.Error())
}

func TestGormLogger_LogMode(t *testing.T) {
	// Arrange
	logger := &GormLogger{}
	
	// Act
	result := logger.LogMode(0)
	
	// Assert
	assert.Equal(t, logger, result)
}

// Note: The remaining GormLogger methods (Info, Warn, Error, Trace)
// primarily call the application logger and don't need extensive testing.
// They rely on appLogger, which would need to be mocked for thorough testing. 