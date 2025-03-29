package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/prasetyowira/shorter/domain/shortener"
	"github.com/prasetyowira/shorter/infrastructure/cache"
	"github.com/prasetyowira/shorter/infrastructure/logger"
	"github.com/prasetyowira/shorter/infrastructure/qrcode"
)

// ShortenRequest represents the request body for creating a short URL
type ShortenRequest struct {
	URL      string `json:"url" binding:"required"`
	ShortURL string `json:"short_url,omitempty"`
}

// ShortenResponse represents the response for a shortened URL
type ShortenResponse struct {
	ShortURL string `json:"short_url"`
	LongURL  string `json:"long_url"`
}

// ShortenerHandler handles URL shortening HTTP requests
type ShortenerHandler struct {
	service     *shortener.Service
	cache       *cache.NamespaceLRU
	qrGenerator *qrcode.Generator
	baseURL     string
}

// NewShortenerHandler creates a new shortener handler
func NewShortenerHandler(service *shortener.Service, cache *cache.NamespaceLRU, qrGenerator *qrcode.Generator, baseURL string) *ShortenerHandler {
	return &ShortenerHandler{
		service:     service,
		cache:       cache,
		qrGenerator: qrGenerator,
		baseURL:     baseURL,
	}
}

// ShortenURL handles the request to create a short URL
func (h *ShortenerHandler) ShortenURL(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req ShortenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.CtxWarn(ctx, "Invalid request payload", logger.LoggerInfo{
			ContextFunction: "ShortenURL",
			Error: &logger.CustomError{
				Code:    "REQ001",
				Message: err.Error(),
				Type:    "validation",
			},
			Data: map[string]interface{}{
				"error": err.Error(),
			},
		})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request payload"})
		return
	}

	logger.CtxDebug(ctx, "Processing URL shortening request", logger.LoggerInfo{
		ContextFunction: "ShortenURL",
		Data: map[string]interface{}{
			"long_url":  req.URL,
			"short_url": req.ShortURL,
		},
	})

	url, err := h.service.CreateShortURL(ctx, req.URL, req.ShortURL)
	if err != nil {
		logger.CtxError(ctx, "Failed to create short URL", logger.LoggerInfo{
			ContextFunction: "ShortenURL",
			Error: &logger.CustomError{
				Code:    "URL001",
				Message: err.Error(),
				Type:    "business",
			},
			Data: map[string]interface{}{
				"long_url":  req.URL,
				"short_url": req.ShortURL,
			},
		})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	// Cache the newly created short URL
	h.cache.Set("urls", url.ShortCode, url.LongURL)

	shortURL := h.baseURL + "/" + url.ShortCode
	logger.CtxInfo(ctx, "Short URL created", logger.LoggerInfo{
		ContextFunction: "ShortenURL",
		Data: map[string]interface{}{
			"short_code": url.ShortCode,
			"short_url":  shortURL,
			"long_url":   url.LongURL,
		},
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(ShortenResponse{
		ShortURL: shortURL,
		LongURL:  url.LongURL,
	})
}

// RedirectToLongURL handles the redirection from short URL to long URL
func (h *ShortenerHandler) RedirectToLongURL(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	shortCode := chi.URLParam(r, "short")

	logger.CtxDebug(ctx, "Processing URL redirection request", logger.LoggerInfo{
		ContextFunction: "RedirectToLongURL",
		Data: map[string]interface{}{
			"short_code": shortCode,
		},
	})

	// If not in cache, get from database
	url, err := h.service.GetLongURL(ctx, shortCode)
	if err != nil {
		logger.CtxWarn(ctx, "Short URL not found", logger.LoggerInfo{
			ContextFunction: "RedirectToLongURL",
			Error: &logger.CustomError{
				Code:    "URL003",
				Message: err.Error(),
				Type:    "business",
			},
			Data: map[string]interface{}{
				"short_code": shortCode,
			},
		})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Short URL not found"})
		return
	}

	// Cache the result for future requests
	h.cache.Set("urls", shortCode, url)

	logger.CtxInfo(ctx, "URL found in database", logger.LoggerInfo{
		ContextFunction: "RedirectToLongURL",
		Data: map[string]interface{}{
			"short_code": shortCode,
			"long_url":   url,
			"cache_hit":  false,
		},
	})

	http.Redirect(w, r, url.LongURL, http.StatusFound)
}

// GenerateQRCode generates a QR code for a short URL
func (h *ShortenerHandler) GenerateQRCode(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	shortCode := chi.URLParam(r, "short")

	logger.CtxDebug(ctx, "Processing QR code generation request", logger.LoggerInfo{
		ContextFunction: "GenerateQRCode",
		Data: map[string]interface{}{
			"short_code": shortCode,
		},
	})

	// Check if the short code exists
	_, err := h.service.GetLongURL(ctx, shortCode)
	if err != nil {
		logger.CtxWarn(ctx, "Short URL not found for QR code", logger.LoggerInfo{
			ContextFunction: "GenerateQRCode",
			Error: &logger.CustomError{
				Code:    "QR001",
				Message: err.Error(),
				Type:    "business",
			},
			Data: map[string]interface{}{
				"short_code": shortCode,
			},
		})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Short URL not found"})
		return
	}

	// Generate QR code
	qrCode, err := h.qrGenerator.GenerateQRCode(shortCode, 256)
	if err != nil {
		logger.CtxError(ctx, "Failed to generate QR code", logger.LoggerInfo{
			ContextFunction: "GenerateQRCode",
			Error: &logger.CustomError{
				Code:    "QR002",
				Message: err.Error(),
				Type:    "system",
			},
			Data: map[string]interface{}{
				"short_code": shortCode,
			},
		})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to generate QR code"})
		return
	}

	logger.CtxInfo(ctx, "QR code generated", logger.LoggerInfo{
		ContextFunction: "GenerateQRCode",
		Data: map[string]interface{}{
			"short_code": shortCode,
			"size":       len(qrCode),
		},
	})

	w.Header().Set("Content-Type", "image/png")
	w.WriteHeader(http.StatusOK)
	w.Write(qrCode)
}
