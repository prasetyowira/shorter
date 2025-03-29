package shortener

import (
	"errors"
	"testing"
	"time"

	"github.com/prasetyowira/shorter/constant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock repository for testing
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) Store(url *URL) error {
	args := m.Called(url)
	return args.Error(0)
}

func (m *MockRepository) FindByShortCode(shortCode string) (*URL, error) {
	args := m.Called(shortCode)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*URL), args.Error(1)
}

func (m *MockRepository) IncrementVisits(shortCode string) error {
	args := m.Called(shortCode)
	return args.Error(0)
}

func TestNewService(t *testing.T) {
	// Arrange
	mockRepo := new(MockRepository)
	
	// Act
	service := NewService(mockRepo)
	
	// Assert
	assert.NotNil(t, service)
	assert.Equal(t, mockRepo, service.repo)
	assert.NotNil(t, service.ctx)
}

func TestCreateShortURL_EmptyLongURL(t *testing.T) {
	// Arrange
	mockRepo := new(MockRepository)
	service := NewService(mockRepo)
	
	// Act
	url, err := service.CreateShortURL("", "")
	
	// Assert
	assert.Error(t, err)
	assert.Equal(t, constant.ErrEmptyLongURL, err.Error())
	assert.Nil(t, url)
	mockRepo.AssertNotCalled(t, "Store")
}

func TestCreateShortURL_WithCustomShortCode(t *testing.T) {
	// Arrange
	mockRepo := new(MockRepository)
	service := NewService(mockRepo)
	
	customShortCode := "custom"
	longURL := "https://example.com"
	
	mockRepo.On("Store", mock.MatchedBy(func(url *URL) bool {
		return url.LongURL == longURL && url.ShortCode == customShortCode
	})).Return(nil)
	
	// Act
	url, err := service.CreateShortURL(longURL, customShortCode)
	
	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, url)
	assert.Equal(t, longURL, url.LongURL)
	assert.Equal(t, customShortCode, url.ShortCode)
	assert.Equal(t, uint(0), url.Visits)
	mockRepo.AssertExpectations(t)
}

func TestCreateShortURL_WithGeneratedShortCode(t *testing.T) {
	// Arrange
	mockRepo := new(MockRepository)
	service := NewService(mockRepo)
	
	longURL := "https://example.com"
	
	mockRepo.On("Store", mock.MatchedBy(func(url *URL) bool {
		return url.LongURL == longURL && len(url.ShortCode) == 6
	})).Return(nil)
	
	// Act
	url, err := service.CreateShortURL(longURL, "")
	
	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, url)
	assert.Equal(t, longURL, url.LongURL)
	assert.Equal(t, 6, len(url.ShortCode))
	assert.Equal(t, uint(0), url.Visits)
	mockRepo.AssertExpectations(t)
}

func TestCreateShortURL_StoreError(t *testing.T) {
	// Arrange
	mockRepo := new(MockRepository)
	service := NewService(mockRepo)
	
	longURL := "https://example.com"
	expectedError := errors.New("store error")
	
	mockRepo.On("Store", mock.AnythingOfType("*shortener.URL")).Return(expectedError)
	
	// Act
	url, err := service.CreateShortURL(longURL, "")
	
	// Assert
	assert.Error(t, err)
	assert.Equal(t, expectedError, err)
	assert.Nil(t, url)
	mockRepo.AssertExpectations(t)
}

func TestGetLongURL_EmptyShortCode(t *testing.T) {
	// Arrange
	mockRepo := new(MockRepository)
	service := NewService(mockRepo)
	
	// Act
	longURL, err := service.GetLongURL("")
	
	// Assert
	assert.Error(t, err)
	assert.Equal(t, constant.ErrEmptyShortCode, err.Error())
	assert.Empty(t, longURL)
	mockRepo.AssertNotCalled(t, "FindByShortCode")
}

func TestGetLongURL_ShortCodeNotFound(t *testing.T) {
	// Arrange
	mockRepo := new(MockRepository)
	service := NewService(mockRepo)
	
	shortCode := "notfound"
	expectedError := errors.New(constant.ErrShortCodeNotFound)
	
	mockRepo.On("FindByShortCode", shortCode).Return(nil, expectedError)
	
	// Act
	longURL, err := service.GetLongURL(shortCode)
	
	// Assert
	assert.Error(t, err)
	assert.Equal(t, expectedError, err)
	assert.Empty(t, longURL)
	mockRepo.AssertExpectations(t)
}

func TestGetLongURL_Success(t *testing.T) {
	// Arrange
	mockRepo := new(MockRepository)
	service := NewService(mockRepo)
	
	shortCode := "abc123"
	expectedURL := &URL{
		ID:        1,
		LongURL:   "https://example.com",
		ShortCode: shortCode,
		CreatedAt: time.Now(),
		Visits:    5,
	}
	
	mockRepo.On("FindByShortCode", shortCode).Return(expectedURL, nil)
	mockRepo.On("IncrementVisits", shortCode).Return(nil)
	
	// Act
	longURL, err := service.GetLongURL(shortCode)
	
	// Assert
	assert.NoError(t, err)
	assert.Equal(t, expectedURL.LongURL, longURL)
	mockRepo.AssertExpectations(t)
}

func TestGetLongURL_IncrementVisitsError(t *testing.T) {
	// Arrange
	mockRepo := new(MockRepository)
	service := NewService(mockRepo)
	
	shortCode := "abc123"
	expectedURL := &URL{
		ID:        1,
		LongURL:   "https://example.com",
		ShortCode: shortCode,
		CreatedAt: time.Now(),
		Visits:    5,
	}
	incrementError := errors.New("increment error")
	
	mockRepo.On("FindByShortCode", shortCode).Return(expectedURL, nil)
	mockRepo.On("IncrementVisits", shortCode).Return(incrementError)
	
	// Act
	longURL, err := service.GetLongURL(shortCode)
	
	// Assert
	assert.NoError(t, err) // Should still succeed despite increment error
	assert.Equal(t, expectedURL.LongURL, longURL)
	mockRepo.AssertExpectations(t)
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