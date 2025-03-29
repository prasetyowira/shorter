package shortener

import (
	"context"
	"errors"
	"testing"

	"github.com/prasetyowira/shorter/constant"
	"github.com/prasetyowira/shorter/infrastructure/cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockRepository is a mock implementation of the Repository interface
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) Store(ctx context.Context, url *URL) error {
	args := m.Called(ctx, url)
	return args.Error(0)
}

func (m *MockRepository) FindByShortCode(ctx context.Context, shortCode string) (*URL, error) {
	args := m.Called(ctx, shortCode)
	return args.Get(0).(*URL), args.Error(1)
}

func (m *MockRepository) IncrementVisits(ctx context.Context, shortCode string) error {
	args := m.Called(ctx, shortCode)
	return args.Error(0)
}

func (m *MockRepository) UpdateLongURL(ctx context.Context, shortCode string, newLongURL string) error {
	args := m.Called(ctx, shortCode, newLongURL)
	return args.Error(0)
}

func (m *MockRepository) Close() error {
	args := m.Called()
	return args.Error(0)
}

func TestService_UpdateLongURL(t *testing.T) {
	// Create cache and mock repository
	cacheLRU := cache.NewNamespaceLRU(100)
	mockRepo := new(MockRepository)
	
	// Create service with mock repository
	service := NewService(mockRepo, cacheLRU)
	
	// Test cases
	tests := []struct {
		name         string
		shortCode    string
		newLongURL   string
		setupMock    func()
		expectedURL  *URL
		expectedErr  error
	}{
		{
			name:       "Success",
			shortCode:  "abc123",
			newLongURL: "https://example.com/updated",
			setupMock: func() {
				existingURL := &URL{
					ID:        1,
					ShortCode: "abc123",
					LongURL:   "https://example.com/original",
					Visits:    5,
				}
				mockRepo.On("FindByShortCode", mock.Anything, "abc123").Return(existingURL, nil)
				mockRepo.On("UpdateLongURL", mock.Anything, "abc123", "https://example.com/updated").Return(nil)
			},
			expectedURL: &URL{
				ID:        1,
				ShortCode: "abc123",
				LongURL:   "https://example.com/updated",
				Visits:    5,
			},
			expectedErr: nil,
		},
		{
			name:       "Empty ShortCode",
			shortCode:  "",
			newLongURL: "https://example.com/updated",
			setupMock:  func() {},
			expectedURL: nil,
			expectedErr: errors.New(constant.ErrEmptyShortCode),
		},
		{
			name:       "Empty LongURL",
			shortCode:  "abc123",
			newLongURL: "",
			setupMock:  func() {},
			expectedURL: nil,
			expectedErr: errors.New(constant.ErrEmptyLongURL),
		},
		{
			name:       "ShortCode Not Found",
			shortCode:  "nonexistent",
			newLongURL: "https://example.com/updated",
			setupMock: func() {
				mockRepo.On("FindByShortCode", mock.Anything, "nonexistent").Return((*URL)(nil), errors.New(constant.ErrShortCodeNotFound))
			},
			expectedURL: nil,
			expectedErr: errors.New(constant.ErrShortCodeNotFound),
		},
		{
			name:       "Update Error",
			shortCode:  "abc123",
			newLongURL: "https://example.com/updated",
			setupMock: func() {
				existingURL := &URL{
					ID:        1,
					ShortCode: "abc123",
					LongURL:   "https://example.com/original",
					Visits:    5,
				}
				mockRepo.On("FindByShortCode", mock.Anything, "abc123").Return(existingURL, nil)
				mockRepo.On("UpdateLongURL", mock.Anything, "abc123", "https://example.com/updated").
					Return(errors.New("database error"))
			},
			expectedURL: nil,
			expectedErr: errors.New("database error"),
		},
	}
	
	// Run tests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset mock and setup for this test case
			mockRepo.ExpectedCalls = nil
			tt.setupMock()
			
			// Call the function
			ctx := context.Background()
			url, err := service.UpdateLongURL(ctx, tt.shortCode, tt.newLongURL)
			
			// Verify results
			if tt.expectedErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedErr.Error(), err.Error())
				assert.Nil(t, url)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedURL.ID, url.ID)
				assert.Equal(t, tt.expectedURL.ShortCode, url.ShortCode)
				assert.Equal(t, tt.expectedURL.LongURL, url.LongURL)
				assert.Equal(t, tt.expectedURL.Visits, url.Visits)
			}
			
			// Verify all mock expectations were met
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestService_UpdateLongURL_Cache(t *testing.T) {
	// Create cache and mock repository
	cacheLRU := cache.NewNamespaceLRU(100)
	mockRepo := new(MockRepository)
	
	// Create service with mock repository
	service := NewService(mockRepo, cacheLRU)
	
	// Create test URL
	existingURL := &URL{
		ID:        1,
		ShortCode: "abc123",
		LongURL:   "https://example.com/original",
		Visits:    5,
	}
	
	// Setup mock
	mockRepo.On("FindByShortCode", mock.Anything, "abc123").Return(existingURL, nil)
	mockRepo.On("UpdateLongURL", mock.Anything, "abc123", "https://example.com/updated").Return(nil)
	
	// Put the original URL in cache
	cacheLRU.Set(constant.ShortURLNamespace, "abc123", existingURL)
	
	// Call the function
	ctx := context.Background()
	url, err := service.UpdateLongURL(ctx, "abc123", "https://example.com/updated")
	
	// Verify results
	assert.NoError(t, err)
	assert.Equal(t, "https://example.com/updated", url.LongURL)
	
	// Verify cache was updated
	cachedURL, found := cacheLRU.Get(constant.ShortURLNamespace, "abc123")
	assert.True(t, found, "URL should be in cache")
	assert.Equal(t, "https://example.com/updated", cachedURL.(*URL).LongURL)
	
	// Verify all mock expectations were met
	mockRepo.AssertExpectations(t)
} 