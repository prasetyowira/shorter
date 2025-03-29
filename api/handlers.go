package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/prasetyowira/shorter/constant"
	"github.com/prasetyowira/shorter/domain/shortener"
	appLogger "github.com/prasetyowira/shorter/infrastructure/logger"
)

// Handler contains service dependencies for API handlers
type Handler struct {
	service *shortener.Service
}

// CreateShortURLRequest is the request object for CreateShortURL endpoint
type CreateShortURLRequest struct {
	LongURL        string `json:"long_url"`
	CustomShortURL string `json:"custom_short_url"`
}

// ShortURLResponse is the response object for short URL operations
type ShortURLResponse struct {
	ShortCode string `json:"short_code"`
	LongURL   string `json:"long_url"`
}

// URLStatsResponse is the response for URL stats
type URLStatsResponse struct {
	ShortCode string `json:"short_code"`
	Visits    uint   `json:"visits"`
}

// ErrorResponse represents an API error response
type ErrorResponse struct {
	Error string `json:"error"`
	Code  int    `json:"code"`
}

// NewHandler creates a new API handler
func NewHandler(service *shortener.Service) *Handler {
	return &Handler{
		service: service,
	}
}

// withRequestID adds a request ID to the context and response headers
func withRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := uuid.New().String()
		ctx := appLogger.WithRequestID(r.Context(), requestID)

		w.Header().Set(constant.HeaderRequestID, requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// logRequest logs incoming requests
func logRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		appLogger.CtxInfo(r.Context(), constant.MsgRequestReceived, appLogger.LoggerInfo{
			ContextFunction: constant.CtxAPI,
			Data: map[string]interface{}{
				constant.DataMethod:     r.Method,
				constant.DataPath:       r.URL.Path,
				constant.DataRemoteAddr: r.RemoteAddr,
				constant.DataUserAgent:  r.UserAgent(),
			},
		})
		next.ServeHTTP(w, r)
	})
}

// CreateShortURL handles short URL creation
func (h *Handler) CreateShortURL(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	appLogger.CtxDebug(ctx, constant.MsgHandlingCreateRequest, appLogger.LoggerInfo{
		ContextFunction: constant.CtxCreateShortURL,
	})

	var req CreateShortURLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		appLogger.CtxError(ctx, "Error decoding request body", appLogger.LoggerInfo{
			ContextFunction: constant.CtxCreateShortURL,
			Error: &appLogger.CustomError{
				Code:    constant.ErrCodeAPIDecodeRequest,
				Message: err.Error(),
				Type:    constant.ErrTypeAPI,
			},
		})

		WriteJSONError(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	url, err := h.service.CreateShortURL(ctx, req.LongURL, req.CustomShortURL)
	if err != nil {
		// Check for specific error messages
		if err.Error() == constant.ErrEmptyLongURL {
			WriteJSONError(w, "URL cannot be empty", http.StatusBadRequest)
			return
		}

		appLogger.CtxError(ctx, "Error creating short URL", appLogger.LoggerInfo{
			ContextFunction: constant.CtxCreateShortURL,
			Error: &appLogger.CustomError{
				Code:    constant.ErrCodeAPIServiceError,
				Message: err.Error(),
				Type:    constant.ErrTypeAPI,
			},
			Data: map[string]interface{}{
				constant.DataLongURL: req.LongURL,
			},
		})

		WriteJSONError(w, "Failed to create short URL", http.StatusInternalServerError)
		return
	}

	resp := ShortURLResponse{
		ShortCode: url.ShortCode,
		LongURL:   url.LongURL,
	}

	appLogger.CtxInfo(ctx, "Created short URL successfully", appLogger.LoggerInfo{
		ContextFunction: constant.CtxCreateShortURL,
		Data: map[string]interface{}{
			constant.DataLongURL:   url.LongURL,
			constant.DataShortCode: url.ShortCode,
		},
	})

	WriteJSON(w, resp, http.StatusCreated)
}

// RedirectToLongURL handles redirection to the original URL
func (h *Handler) RedirectToLongURL(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	shortCode := chi.URLParam(r, "shortCode")

	appLogger.CtxDebug(ctx, constant.MsgProcessingRedirectRequest, appLogger.LoggerInfo{
		ContextFunction: constant.CtxRedirectToLongURL,
		Data: map[string]interface{}{
			constant.DataShortCode: shortCode,
		},
	})

	url, err := h.service.GetLongURL(ctx, shortCode)
	if err != nil {
		if err.Error() == constant.ErrShortCodeNotFound {
			appLogger.CtxInfo(ctx, "Short code not found", appLogger.LoggerInfo{
				ContextFunction: constant.CtxRedirectToLongURL,
				Data: map[string]interface{}{
					constant.DataShortCode: shortCode,
				},
			})

			http.NotFound(w, r)
			return
		}

		appLogger.CtxError(ctx, "Error retrieving long URL", appLogger.LoggerInfo{
			ContextFunction: constant.CtxRedirectToLongURL,
			Error: &appLogger.CustomError{
				Code:    constant.ErrCodeAPIServiceError,
				Message: err.Error(),
				Type:    constant.ErrTypeAPI,
			},
			Data: map[string]interface{}{
				constant.DataShortCode: shortCode,
			},
		})

		WriteJSONError(w, "Error retrieving URL", http.StatusInternalServerError)
		return
	}

	appLogger.CtxInfo(ctx, "Redirecting to long URL", appLogger.LoggerInfo{
		ContextFunction: constant.CtxRedirectToLongURL,
		Data: map[string]interface{}{
			constant.DataShortCode: shortCode,
			constant.DataLongURL:   url,
		},
	})

	http.Redirect(w, r, url.LongURL, http.StatusFound)
}

// GetURLStats handles retrieving URL stats
func (h *Handler) GetURLStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	shortCode := chi.URLParam(r, "shortCode")

	appLogger.CtxDebug(ctx, "Processing URL stats request", appLogger.LoggerInfo{
		ContextFunction: constant.CtxGetURLStats,
		Data: map[string]interface{}{
			constant.DataShortCode: shortCode,
		},
	})

	url, err := h.service.GetLongURL(ctx, shortCode)
	if err != nil {
		if err.Error() == constant.ErrShortCodeNotFound {
			appLogger.CtxInfo(ctx, "Short code not found for stats", appLogger.LoggerInfo{
				ContextFunction: constant.CtxGetURLStats,
				Data: map[string]interface{}{
					constant.DataShortCode: shortCode,
				},
			})

			http.NotFound(w, r)
			return
		}

		appLogger.CtxError(ctx, "Error retrieving URL stats", appLogger.LoggerInfo{
			ContextFunction: constant.CtxGetURLStats,
			Error: &appLogger.CustomError{
				Code:    constant.ErrCodeAPIServiceError,
				Message: err.Error(),
				Type:    constant.ErrTypeAPI,
			},
			Data: map[string]interface{}{
				constant.DataShortCode: shortCode,
			},
		})

		WriteJSONError(w, "Error retrieving URL stats", http.StatusInternalServerError)
		return
	}

	resp := URLStatsResponse{
		ShortCode: url.ShortCode,
		Visits:    url.Visits,
	}

	appLogger.CtxInfo(ctx, "URL stats retrieved successfully", appLogger.LoggerInfo{
		ContextFunction: constant.CtxGetURLStats,
		Data: map[string]interface{}{
			constant.DataShortCode: shortCode,
			constant.DataVisits:    url.Visits,
		},
	})

	WriteJSON(w, resp, http.StatusOK)
}

// WriteJSON writes a JSON response
func WriteJSON(w http.ResponseWriter, data interface{}, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	err := json.NewEncoder(w).Encode(data)
	if err != nil {
		return
	}
}

// WriteJSONError writes a JSON error response
func WriteJSONError(w http.ResponseWriter, message string, statusCode int) {
	WriteJSON(w, ErrorResponse{
		Error: message,
		Code:  statusCode,
	}, statusCode)
}
