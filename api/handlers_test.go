package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prasetyowira/shorter/constant"
	"github.com/prasetyowira/shorter/domain/shortener"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock service for testing
type MockService struct {
	mock.Mock
}

func (m *MockService) CreateShortURL(ctx context.Context, longURL string) (*shortener.URL, error) {
	args := m.Called(ctx, longURL)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*shortener.URL), args.Error(1)
}

func (m *MockService) GetLongURL(ctx context.Context, shortCode string) (string, error) {
	args := m.Called(ctx, shortCode)
	return args.String(0), args.Error(1)
}

func TestNewHandler(t *testing.T) {
	// Arrange
	mockService := new(MockService)
	
	// Act
	handler := NewHandler(mockService)
	
	// Assert
	assert.NotNil(t, handler)
	assert.Equal(t, mockService, handler.service)
}

func TestWithRequestID(t *testing.T) {
	// Arrange
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if request ID is in context
		requestID := r.Context().Value(constant.RequestIDKey)
		assert.NotNil(t, requestID)
		
		// Check if request ID header is set
		headerRequestID := w.Header().Get(constant.HeaderRequestID)
		assert.NotEmpty(t, headerRequestID)
		assert.Equal(t, requestID, headerRequestID)
		
		w.WriteHeader(http.StatusOK)
	})
	
	middleware := withRequestID(nextHandler)
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	
	// Act
	middleware.ServeHTTP(w, req)
	
	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestLogRequest(t *testing.T) {
	// Arrange
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	
	middleware := logRequest(nextHandler)
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	
	// Act
	middleware.ServeHTTP(w, req)
	
	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	// Note: We can't easily test the logger output without mocking it
}

func TestCreateShortURL_Success(t *testing.T) {
	// Arrange
	mockService := new(MockService)
	handler := NewHandler(mockService)
	
	longURL := "https://example.com"
	createReq := CreateShortURLRequest{
		LongURL: longURL,
	}
	
	expectedURL := &shortener.URL{
		ID:        1,
		LongURL:   longURL,
		ShortCode: "abc123",
		CreatedAt: time.Now(),
		Visits:    0,
	}
	
	mockService.On("CreateShortURL", mock.Anything, longURL).Return(expectedURL, nil)
	
	reqBody, _ := json.Marshal(createReq)
	req := httptest.NewRequest("POST", "/api/urls", bytes.NewBuffer(reqBody))
	w := httptest.NewRecorder()
	
	// Act
	handler.CreateShortURL(w, req)
	
	// Assert
	assert.Equal(t, http.StatusCreated, w.Code)
	
	var response ShortURLResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, expectedURL.ShortCode, response.ShortCode)
	assert.Equal(t, expectedURL.LongURL, response.LongURL)
	
	mockService.AssertExpectations(t)
}

func TestCreateShortURL_InvalidRequestBody(t *testing.T) {
	// Arrange
	mockService := new(MockService)
	handler := NewHandler(mockService)
	
	invalidJSON := []byte(`{"long_url": }`) // Invalid JSON
	req := httptest.NewRequest("POST", "/api/urls", bytes.NewBuffer(invalidJSON))
	w := httptest.NewRecorder()
	
	// Act
	handler.CreateShortURL(w, req)
	
	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)
	
	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Invalid request format", response.Error)
	assert.Equal(t, http.StatusBadRequest, response.Code)
	
	mockService.AssertNotCalled(t, "CreateShortURL")
}

func TestCreateShortURL_EmptyURL(t *testing.T) {
	// Arrange
	mockService := new(MockService)
	handler := NewHandler(mockService)
	
	createReq := CreateShortURLRequest{
		LongURL: "", // Empty URL
	}
	
	mockService.On("CreateShortURL", mock.Anything, "").
		Return(nil, errors.New(constant.ErrEmptyLongURL))
	
	reqBody, _ := json.Marshal(createReq)
	req := httptest.NewRequest("POST", "/api/urls", bytes.NewBuffer(reqBody))
	w := httptest.NewRecorder()
	
	// Act
	handler.CreateShortURL(w, req)
	
	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)
	
	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "URL cannot be empty", response.Error)
	
	mockService.AssertExpectations(t)
}

func TestCreateShortURL_ServiceError(t *testing.T) {
	// Arrange
	mockService := new(MockService)
	handler := NewHandler(mockService)
	
	longURL := "https://example.com"
	createReq := CreateShortURLRequest{
		LongURL: longURL,
	}
	
	expectedError := errors.New("service error")
	mockService.On("CreateShortURL", mock.Anything, longURL).
		Return(nil, expectedError)
	
	reqBody, _ := json.Marshal(createReq)
	req := httptest.NewRequest("POST", "/api/urls", bytes.NewBuffer(reqBody))
	w := httptest.NewRecorder()
	
	// Act
	handler.CreateShortURL(w, req)
	
	// Assert
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	
	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Failed to create short URL", response.Error)
	
	mockService.AssertExpectations(t)
}

func TestRedirectToLongURL_Success(t *testing.T) {
	// Arrange
	mockService := new(MockService)
	handler := NewHandler(mockService)
	
	shortCode := "abc123"
	longURL := "https://example.com"
	
	mockService.On("GetLongURL", mock.Anything, shortCode).Return(longURL, nil)
	
	// Setup Chi router context with URL parameter
	req := httptest.NewRequest("GET", "/"+shortCode, nil)
	w := httptest.NewRecorder()
	
	// Chi router context setup
	chiCtx := chi.NewRouteContext()
	chiCtx.URLParams.Add("shortCode", shortCode)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, chiCtx))
	
	// Act
	handler.RedirectToLongURL(w, req)
	
	// Assert
	assert.Equal(t, http.StatusFound, w.Code)
	assert.Equal(t, longURL, w.Header().Get("Location"))
	
	mockService.AssertExpectations(t)
}

func TestRedirectToLongURL_NotFound(t *testing.T) {
	// Arrange
	mockService := new(MockService)
	handler := NewHandler(mockService)
	
	shortCode := "notfound"
	
	mockService.On("GetLongURL", mock.Anything, shortCode).
		Return("", errors.New(constant.ErrShortCodeNotFound))
	
	// Setup Chi router context with URL parameter
	req := httptest.NewRequest("GET", "/"+shortCode, nil)
	w := httptest.NewRecorder()
	
	// Chi router context setup
	chiCtx := chi.NewRouteContext()
	chiCtx.URLParams.Add("shortCode", shortCode)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, chiCtx))
	
	// Act
	handler.RedirectToLongURL(w, req)
	
	// Assert
	assert.Equal(t, http.StatusNotFound, w.Code)
	
	mockService.AssertExpectations(t)
}

func TestRedirectToLongURL_ServiceError(t *testing.T) {
	// Arrange
	mockService := new(MockService)
	handler := NewHandler(mockService)
	
	shortCode := "abc123"
	expectedError := errors.New("service error")
	
	mockService.On("GetLongURL", mock.Anything, shortCode).
		Return("", expectedError)
	
	// Setup Chi router context with URL parameter
	req := httptest.NewRequest("GET", "/"+shortCode, nil)
	w := httptest.NewRecorder()
	
	// Chi router context setup
	chiCtx := chi.NewRouteContext()
	chiCtx.URLParams.Add("shortCode", shortCode)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, chiCtx))
	
	// Act
	handler.RedirectToLongURL(w, req)
	
	// Assert
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	
	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Error retrieving URL", response.Error)
	
	mockService.AssertExpectations(t)
}

func TestGetURLStats_Success(t *testing.T) {
	// Arrange
	mockService := new(MockService)
	handler := NewHandler(mockService)
	
	shortCode := "abc123"
	url := &shortener.URL{
		ID:        1,
		LongURL:   "https://example.com",
		ShortCode: shortCode,
		CreatedAt: time.Now(),
		Visits:    42,
	}
	
	mockService.On("GetLongURL", mock.Anything, shortCode).Return(url, nil)
	
	// Setup Chi router context with URL parameter
	req := httptest.NewRequest("GET", "/api/urls/"+shortCode+"/stats", nil)
	w := httptest.NewRecorder()
	
	// Chi router context setup
	chiCtx := chi.NewRouteContext()
	chiCtx.URLParams.Add("shortCode", shortCode)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, chiCtx))
	
	// Act
	handler.GetURLStats(w, req)
	
	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response URLStatsResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, shortCode, response.ShortCode)
	assert.Equal(t, uint(42), response.Visits)
	
	mockService.AssertExpectations(t)
}

func TestGetURLStats_NotFound(t *testing.T) {
	// Arrange
	mockService := new(MockService)
	handler := NewHandler(mockService)
	
	shortCode := "notfound"
	
	mockService.On("GetLongURL", mock.Anything, shortCode).
		Return("", errors.New(constant.ErrShortCodeNotFound))
	
	// Setup Chi router context with URL parameter
	req := httptest.NewRequest("GET", "/api/urls/"+shortCode+"/stats", nil)
	w := httptest.NewRecorder()
	
	// Chi router context setup
	chiCtx := chi.NewRouteContext()
	chiCtx.URLParams.Add("shortCode", shortCode)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, chiCtx))
	
	// Act
	handler.GetURLStats(w, req)
	
	// Assert
	assert.Equal(t, http.StatusNotFound, w.Code)
	
	mockService.AssertExpectations(t)
} 