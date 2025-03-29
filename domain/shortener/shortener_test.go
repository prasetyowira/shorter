package shortener

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/prasetyowira/shorter/constant"
	"github.com/prasetyowira/shorter/infrastructure/cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock repository for testing
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) Store(ctx context.Context, url *URL) error {
	args := m.Called(ctx, url)
	return args.Error(0)
}

func (m *MockRepository) FindByShortCode(ctx context.Context, shortCode string) (*URL, error) {
	args := m.Called(ctx, shortCode)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*URL), args.Error(1)
}

func (m *MockRepository) IncrementVisits(ctx context.Context, shortCode string) error {
	args := m.Called(ctx, shortCode)
	return args.Error(0)
}

// MockCache is a mock implementation of cache.NamespaceLRU
type MockCache struct {
	mock.Mock
}

func (m *MockCache) Get(namespace, key string) (interface{}, bool) {
	args := m.Called(namespace, key)
	return args.Get(0), args.Bool(1)
}

func (m *MockCache) Set(namespace, key string, value interface{}) {
	m.Called(namespace, key, value)
}

func TestNewService(t *testing.T) {
	// Arrange
	mockRepo := new(MockRepository)
	mockCache := new(MockCache)
	
	// Act
	service := NewService(mockRepo, mockCache)
	
	// Assert
	assert.NotNil(t, service)
	assert.Equal(t, mockRepo, service.repo)
	assert.Equal(t, mockCache, service.cache)
}

func TestCreateShortURL_EmptyLongURL(t *testing.T) {
	// Arrange
	mockRepo := new(MockRepository)
	mockCache := new(MockCache)
	service := NewService(mockRepo, mockCache)
	ctx := context.Background()
	
	// Act
	url, err := service.CreateShortURL(ctx, "", "")
	
	// Assert
	assert.Error(t, err)
	assert.Equal(t, constant.ErrEmptyLongURL, err.Error())
	assert.Nil(t, url)
	mockRepo.AssertNotCalled(t, "Store")
	mockCache.AssertNotCalled(t, "Set")
}

func TestCreateShortURL_WithCustomShortCode(t *testing.T) {
	// Arrange
	mockRepo := new(MockRepository)
	mockCache := new(MockCache)
	service := NewService(mockRepo, mockCache)
	ctx := context.Background()
	
	customShortCode := "custom"
	longURL := "https://example.com"
	
	mockRepo.On("Store", ctx, mock.MatchedBy(func(url *URL) bool {
		return url.LongURL == longURL && url.ShortCode == customShortCode
	})).Return(nil)
	
	mockCache.On("Set", constant.ShortURLNamespace, customShortCode, mock.AnythingOfType("*shortener.URL")).Return()
	
	// Act
	url, err := service.CreateShortURL(ctx, longURL, customShortCode)
	
	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, url)
	assert.Equal(t, longURL, url.LongURL)
	assert.Equal(t, customShortCode, url.ShortCode)
	assert.Equal(t, uint(0), url.Visits)
	mockRepo.AssertExpectations(t)
	mockCache.AssertExpectations(t)
}

func TestCreateShortURL_WithGeneratedShortCode(t *testing.T) {
	// Arrange
	mockRepo := new(MockRepository)
	mockCache := new(MockCache)
	service := NewService(mockRepo, mockCache)
	ctx := context.Background()
	
	longURL := "https://example.com"
	
	mockRepo.On("Store", ctx, mock.MatchedBy(func(url *URL) bool {
		return url.LongURL == longURL && len(url.ShortCode) == 6
	})).Return(nil)
	
	mockCache.On("Set", constant.ShortURLNamespace, mock.AnythingOfType("string"), mock.AnythingOfType("*shortener.URL")).Return()
	
	// Act
	url, err := service.CreateShortURL(ctx, longURL, "")
	
	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, url)
	assert.Equal(t, longURL, url.LongURL)
	assert.Equal(t, 6, len(url.ShortCode))
	assert.Equal(t, uint(0), url.Visits)
	mockRepo.AssertExpectations(t)
	mockCache.AssertExpectations(t)
}

func TestCreateShortURL_StoreError(t *testing.T) {
	// Arrange
	mockRepo := new(MockRepository)
	mockCache := new(MockCache)
	service := NewService(mockRepo, mockCache)
	ctx := context.Background()
	
	longURL := "https://example.com"
	expectedError := errors.New("store error")
	
	mockRepo.On("Store", ctx, mock.AnythingOfType("*shortener.URL")).Return(expectedError)
	
	// Act
	url, err := service.CreateShortURL(ctx, longURL, "")
	
	// Assert
	assert.Error(t, err)
	assert.Equal(t, expectedError, err)
	assert.Nil(t, url)
	mockRepo.AssertExpectations(t)
	mockCache.AssertNotCalled(t, "Set")
}

func TestGetLongURL_EmptyShortCode(t *testing.T) {
	// Arrange
	mockRepo := new(MockRepository)
	mockCache := new(MockCache)
	service := NewService(mockRepo, mockCache)
	ctx := context.Background()
	
	// Act
	url, err := service.GetLongURL(ctx, "")
	
	// Assert
	assert.Error(t, err)
	assert.Equal(t, constant.ErrEmptyShortCode, err.Error())
	assert.Nil(t, url)
	mockRepo.AssertNotCalled(t, "FindByShortCode")
	mockCache.AssertNotCalled(t, "Get")
}

func TestGetLongURL_CacheHit(t *testing.T) {
	// Arrange
	mockRepo := new(MockRepository)
	mockCache := new(MockCache)
	service := NewService(mockRepo, mockCache)
	ctx := context.Background()
	
	shortCode := "abc123"
	cachedURL := &URL{
		ID:        1,
		LongURL:   "https://example.com",
		ShortCode: shortCode,
		CreatedAt: time.Now(),
		Visits:    5,
	}
	
	mockCache.On("Get", constant.ShortURLNamespace, shortCode).Return(cachedURL, true)
	mockRepo.On("IncrementVisits", ctx, shortCode).Return(nil)
	
	// Act
	url, err := service.GetLongURL(ctx, shortCode)
	
	// Assert
	assert.NoError(t, err)
	assert.Equal(t, cachedURL, url)
	mockCache.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
	mockRepo.AssertNotCalled(t, "FindByShortCode")
}

func TestGetLongURL_ShortCodeNotFound(t *testing.T) {
	// Arrange
	mockRepo := new(MockRepository)
	mockCache := new(MockCache)
	service := NewService(mockRepo, mockCache)
	ctx := context.Background()
	
	shortCode := "notfound"
	expectedError := errors.New(constant.ErrShortCodeNotFound)
	
	mockCache.On("Get", constant.ShortURLNamespace, shortCode).Return(nil, false)
	mockRepo.On("FindByShortCode", ctx, shortCode).Return(nil, expectedError)
	
	// Act
	url, err := service.GetLongURL(ctx, shortCode)
	
	// Assert
	assert.Error(t, err)
	assert.Equal(t, expectedError, err)
	assert.Nil(t, url)
	mockRepo.AssertExpectations(t)
	mockCache.AssertExpectations(t)
}

func TestGetLongURL_Success(t *testing.T) {
	// Arrange
	mockRepo := new(MockRepository)
	mockCache := new(MockCache)
	service := NewService(mockRepo, mockCache)
	ctx := context.Background()
	
	shortCode := "abc123"
	expectedURL := &URL{
		ID:        1,
		LongURL:   "https://example.com",
		ShortCode: shortCode,
		CreatedAt: time.Now(),
		Visits:    5,
	}
	
	mockCache.On("Get", constant.ShortURLNamespace, shortCode).Return(nil, false)
	mockRepo.On("FindByShortCode", ctx, shortCode).Return(expectedURL, nil)
	mockRepo.On("IncrementVisits", ctx, shortCode).Return(nil)
	
	// Act
	url, err := service.GetLongURL(ctx, shortCode)
	
	// Assert
	assert.NoError(t, err)
	assert.Equal(t, expectedURL, url)
	mockRepo.AssertExpectations(t)
	mockCache.AssertExpectations(t)
}

func TestGetLongURL_IncrementVisitsError(t *testing.T) {
	// Arrange
	mockRepo := new(MockRepository)
	mockCache := new(MockCache)
	service := NewService(mockRepo, mockCache)
	ctx := context.Background()
	
	shortCode := "abc123"
	expectedURL := &URL{
		ID:        1,
		LongURL:   "https://example.com",
		ShortCode: shortCode,
		CreatedAt: time.Now(),
		Visits:    5,
	}
	incrementError := errors.New("increment error")
	
	mockCache.On("Get", constant.ShortURLNamespace, shortCode).Return(nil, false)
	mockRepo.On("FindByShortCode", ctx, shortCode).Return(expectedURL, nil)
	mockRepo.On("IncrementVisits", ctx, shortCode).Return(incrementError)
	
	// Act
	url, err := service.GetLongURL(ctx, shortCode)
	
	// Assert
	assert.NoError(t, err) // Should still succeed despite increment error
	assert.Equal(t, expectedURL, url)
	mockRepo.AssertExpectations(t)
	mockCache.AssertExpectations(t)
}

func TestGenerateShortCode(t *testing.T) {
	// Test that generated codes have the expected length
	code1 := generateShortCode(6)
	assert.Equal(t, 6, len(code1))
	
	// Test that generated codes are different
	code2 := generateShortCode(6)
	assert.NotEqual(t, code1, code2)
	
	// Test with different lengths
	code3 := generateShortCode(8)
	assert.Equal(t, 8, len(code3))
} 