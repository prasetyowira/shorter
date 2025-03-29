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
	"github.com/prasetyowira/shorter/infrastructure/qrcode"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock service for testing
type MockService struct {
	mock.Mock
}

func (m *MockService) CreateShortURL(ctx context.Context, longURL string, customShortURL string) (*shortener.URL, error) {
	args := m.Called(ctx, longURL, customShortURL)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*shortener.URL), args.Error(1)
}

func (m *MockService) GetLongURL(ctx context.Context, shortCode string) (*shortener.URL, error) {
	args := m.Called(ctx, shortCode)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*shortener.URL), args.Error(1)
}

// Mock QR code generator for testing
type MockQRGenerator struct {
	mock.Mock
}

func (m *MockQRGenerator) GenerateQRCode(shortCode string, size int) ([]byte, error) {
	args := m.Called(shortCode, size)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}

func TestNewHandler(t *testing.T) {
	// Arrange
	mockService := new(MockService)
	mockQRGenerator := new(MockQRGenerator)
	baseURL := "http://localhost:8080"
	
	// Act
	handler := NewHandler(mockService, mockQRGenerator, baseURL)
	
	// Assert
	assert.NotNil(t, handler)
	assert.Equal(t, mockService, handler.service)
	assert.Equal(t, mockQRGenerator, handler.qrGenerator)
	assert.Equal(t, baseURL, handler.baseURL)
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
	
	mockService.On("CreateShortURL", mock.Anything, longURL, mock.Anything).Return(expectedURL, nil)
	
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
	
	mockService.On("CreateShortURL", mock.Anything, "", mock.Anything).
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
	mockService.On("CreateShortURL", mock.Anything, longURL, mock.Anything).
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
	mockQRGenerator := new(MockQRGenerator)
	baseURL := "http://localhost:8080"
	handler := NewHandler(mockService, mockQRGenerator, baseURL)
	
	shortCode := "abc123"
	mockURL := &shortener.URL{
		ID:        1,
		LongURL:   "https://example.com",
		ShortCode: shortCode,
		CreatedAt: time.Now(),
		Visits:    5,
	}
	
	mockService.On("GetLongURL", mock.Anything, shortCode).Return(mockURL, nil)
	
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
	assert.Equal(t, mockURL.LongURL, w.Header().Get("Location"))
	
	mockService.AssertExpectations(t)
}

func TestRedirectToLongURL_NotFound(t *testing.T) {
	// Arrange
	mockService := new(MockService)
	mockQRGenerator := new(MockQRGenerator)
	baseURL := "http://localhost:8080"
	handler := NewHandler(mockService, mockQRGenerator, baseURL)
	
	shortCode := "nonexistent"
	
	mockService.On("GetLongURL", mock.Anything, shortCode).
		Return(nil, errors.New(constant.ErrShortCodeNotFound))
	
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
	mockQRGenerator := new(MockQRGenerator)
	baseURL := "http://localhost:8080"
	handler := NewHandler(mockService, mockQRGenerator, baseURL)
	
	shortCode := "abc123"
	expectedError := errors.New("service error")
	
	mockService.On("GetLongURL", mock.Anything, shortCode).
		Return(nil, expectedError)
	
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
	mockQRGenerator := new(MockQRGenerator)
	baseURL := "http://localhost:8080"
	handler := NewHandler(mockService, mockQRGenerator, baseURL)
	
	shortCode := "abc123"
	visits := uint(42)
	mockURL := &shortener.URL{
		ID:        1,
		LongURL:   "https://example.com",
		ShortCode: shortCode,
		CreatedAt: time.Now(),
		Visits:    visits,
	}
	
	mockService.On("GetLongURL", mock.Anything, shortCode).Return(mockURL, nil)
	
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
	assert.Equal(t, visits, response.Visits)
	
	mockService.AssertExpectations(t)
}

func TestGetURLStats_NotFound(t *testing.T) {
	// Arrange
	mockService := new(MockService)
	mockQRGenerator := new(MockQRGenerator)
	baseURL := "http://localhost:8080"
	handler := NewHandler(mockService, mockQRGenerator, baseURL)
	
	shortCode := "nonexistent"
	
	mockService.On("GetLongURL", mock.Anything, shortCode).
		Return(nil, errors.New(constant.ErrShortCodeNotFound))
	
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

func TestGetURLStats_ServiceError(t *testing.T) {
	// Arrange
	mockService := new(MockService)
	mockQRGenerator := new(MockQRGenerator)
	baseURL := "http://localhost:8080"
	handler := NewHandler(mockService, mockQRGenerator, baseURL)
	
	shortCode := "abc123"
	expectedError := errors.New("service error")
	
	mockService.On("GetLongURL", mock.Anything, shortCode).
		Return(nil, expectedError)
	
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
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	
	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Failed to retrieve URL stats", response.Error)
	
	mockService.AssertExpectations(t)
}

func TestGenerateQRCode_Success(t *testing.T) {
	// Arrange
	mockService := new(MockService)
	mockQRGenerator := new(MockQRGenerator)
	baseURL := "http://localhost:8080"
	handler := NewHandler(mockService, mockQRGenerator, baseURL)
	
	shortCode := "abc123"
	mockQRData := []byte("fake-qr-code-data")
	mockURL := &shortener.URL{
		ID:        1,
		LongURL:   "https://example.com",
		ShortCode: shortCode,
		CreatedAt: time.Now(),
		Visits:    5,
	}
	
	mockService.On("GetLongURL", mock.Anything, shortCode).Return(mockURL, nil)
	mockQRGenerator.On("GenerateQRCode", shortCode, 256).Return(mockQRData, nil)
	
	// Chi router context setup
	req := httptest.NewRequest("GET", "/api/urls/"+shortCode+"/qrcode", nil)
	w := httptest.NewRecorder()
	
	chiCtx := chi.NewRouteContext()
	chiCtx.URLParams.Add("shortCode", shortCode)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, chiCtx))
	
	// Act
	handler.GenerateQRCode(w, req)
	
	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "image/png", w.Header().Get("Content-Type"))
	assert.Equal(t, mockQRData, w.Body.Bytes())
	
	mockService.AssertExpectations(t)
	mockQRGenerator.AssertExpectations(t)
}

func TestGenerateQRCode_ShortCodeNotFound(t *testing.T) {
	// Arrange
	mockService := new(MockService)
	mockQRGenerator := new(MockQRGenerator)
	baseURL := "http://localhost:8080"
	handler := NewHandler(mockService, mockQRGenerator, baseURL)
	
	shortCode := "nonexistent"
	
	mockService.On("GetLongURL", mock.Anything, shortCode).
		Return(nil, errors.New(constant.ErrShortCodeNotFound))
	
	// Chi router context setup
	req := httptest.NewRequest("GET", "/api/urls/"+shortCode+"/qrcode", nil)
	w := httptest.NewRecorder()
	
	chiCtx := chi.NewRouteContext()
	chiCtx.URLParams.Add("shortCode", shortCode)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, chiCtx))
	
	// Act
	handler.GenerateQRCode(w, req)
	
	// Assert
	assert.Equal(t, http.StatusNotFound, w.Code)
	
	mockService.AssertExpectations(t)
	mockQRGenerator.AssertNotCalled(t, "GenerateQRCode")
}

func TestGenerateQRCode_ServiceError(t *testing.T) {
	// Arrange
	mockService := new(MockService)
	mockQRGenerator := new(MockQRGenerator)
	baseURL := "http://localhost:8080"
	handler := NewHandler(mockService, mockQRGenerator, baseURL)
	
	shortCode := "abc123"
	expectedError := errors.New("service error")
	
	mockService.On("GetLongURL", mock.Anything, shortCode).
		Return(nil, expectedError)
	
	// Chi router context setup
	req := httptest.NewRequest("GET", "/api/urls/"+shortCode+"/qrcode", nil)
	w := httptest.NewRecorder()
	
	chiCtx := chi.NewRouteContext()
	chiCtx.URLParams.Add("shortCode", shortCode)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, chiCtx))
	
	// Act
	handler.GenerateQRCode(w, req)
	
	// Assert
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	
	mockService.AssertExpectations(t)
	mockQRGenerator.AssertNotCalled(t, "GenerateQRCode")
}

func TestGenerateQRCode_QRGenerationError(t *testing.T) {
	// Arrange
	mockService := new(MockService)
	mockQRGenerator := new(MockQRGenerator)
	baseURL := "http://localhost:8080"
	handler := NewHandler(mockService, mockQRGenerator, baseURL)
	
	shortCode := "abc123"
	qrError := errors.New("qr generation error")
	mockURL := &shortener.URL{
		ID:        1,
		LongURL:   "https://example.com",
		ShortCode: shortCode,
		CreatedAt: time.Now(),
		Visits:    5,
	}
	
	mockService.On("GetLongURL", mock.Anything, shortCode).Return(mockURL, nil)
	mockQRGenerator.On("GenerateQRCode", shortCode, 256).Return(nil, qrError)
	
	// Chi router context setup
	req := httptest.NewRequest("GET", "/api/urls/"+shortCode+"/qrcode", nil)
	w := httptest.NewRecorder()
	
	chiCtx := chi.NewRouteContext()
	chiCtx.URLParams.Add("shortCode", shortCode)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, chiCtx))
	
	// Act
	handler.GenerateQRCode(w, req)
	
	// Assert
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	
	mockService.AssertExpectations(t)
	mockQRGenerator.AssertExpectations(t)
} 