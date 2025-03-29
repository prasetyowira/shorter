package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockHandler implements api.Handler interface for testing
type MockHandler struct {
	mock.Mock
}

func (m *MockHandler) CreateShortURL(w http.ResponseWriter, r *http.Request) {
	m.Called(w, r)
	w.WriteHeader(http.StatusCreated)
}

func (m *MockHandler) RedirectToLongURL(w http.ResponseWriter, r *http.Request) {
	m.Called(w, r)
	w.WriteHeader(http.StatusFound)
}

func (m *MockHandler) GetURLStats(w http.ResponseWriter, r *http.Request) {
	m.Called(w, r)
	w.WriteHeader(http.StatusOK)
}

func TestNewRouter(t *testing.T) {
	// Arrange
	mockHandler := new(MockHandler)
	
	// Act
	router := NewRouter(mockHandler)
	
	// Assert
	assert.NotNil(t, router)
	assert.Equal(t, mockHandler, router.handler)
	assert.NotNil(t, router.router)
	assert.IsType(t, &chi.Mux{}, router.router)
}

func TestRouter_SetupRoutes(t *testing.T) {
	// Arrange
	mockHandler := new(MockHandler)
	router := NewRouter(mockHandler)
	
	// Act
	router.SetupRoutes()
	
	// Testing POST /api/urls
	mockHandler.On("CreateShortURL", mock.Anything, mock.Anything).Once()
	req := httptest.NewRequest("POST", "/api/urls", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)
	
	// Testing GET /{shortCode}
	mockHandler.On("RedirectToLongURL", mock.Anything, mock.Anything).Once()
	req = httptest.NewRequest("GET", "/abc123", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusFound, w.Code)
	
	// Testing GET /api/urls/{shortCode}/stats
	mockHandler.On("GetURLStats", mock.Anything, mock.Anything).Once()
	req = httptest.NewRequest("GET", "/api/urls/abc123/stats", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	
	// Testing healthcheck route
	req = httptest.NewRequest("GET", "/health", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "Healthy", w.Body.String())
	
	// Assert that all expected calls were made
	mockHandler.AssertExpectations(t)
} 