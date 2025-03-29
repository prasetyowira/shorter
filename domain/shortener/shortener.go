package shortener

import (
	"context"
	"errors"
	"github.com/prasetyowira/shorter/infrastructure/cache"
	"time"

	"github.com/prasetyowira/shorter/constant"
	"github.com/prasetyowira/shorter/infrastructure/logger"
)

// URL represents the core domain model for a shortened URL
type URL struct {
	ID        uint      `json:"id"`
	LongURL   string    `json:"long_url"`
	ShortCode string    `json:"short_code"`
	CreatedAt time.Time `json:"created_at"`
	Visits    uint      `json:"visits"`
}

// Repository defines the interface for data persistence operations
type Repository interface {
	Store(ctx context.Context, url *URL) error
	FindByShortCode(ctx context.Context, shortCode string) (*URL, error)
	IncrementVisits(ctx context.Context, shortCode string) error
	UpdateLongURL(ctx context.Context, shortCode string, newLongURL string) error
}

// Service represents the domain service for URL shortening
type Service struct {
	repo  Repository
	cache *cache.NamespaceLRU
}

// NewService creates a new shortener service
func NewService(repo Repository, lru *cache.NamespaceLRU) *Service {
	ctx := logger.NewRequestContext()

	logger.CtxDebug(ctx, "Creating shortener service", logger.LoggerInfo{
		ContextFunction: constant.CtxDomain,
		Data: map[string]interface{}{
			constant.DataService: "shortener",
		},
	})

	return &Service{
		repo:  repo,
		cache: lru,
	}
}

// CreateShortURL creates a new shortened URL
func (s *Service) CreateShortURL(ctx context.Context, longURL, customShort string) (*URL, error) {
	logger.CtxDebug(ctx, "Creating short URL", logger.LoggerInfo{
		ContextFunction: constant.CtxCreateShortURL,
		Data: map[string]interface{}{
			constant.DataLongURL:     longURL,
			constant.DataCustomShort: customShort != "",
		},
	})

	if longURL == "" {
		logger.CtxWarn(ctx, "Long URL cannot be empty", logger.LoggerInfo{
			ContextFunction: constant.CtxCreateShortURL,
			Error: &logger.CustomError{
				Code:    constant.ErrCodeEmptyLongURL,
				Message: constant.ErrEmptyLongURL,
				Type:    constant.ErrTypeValidation,
			},
		})
		return nil, errors.New(constant.ErrEmptyLongURL)
	}

	shortCode := customShort
	if shortCode == "" {
		shortCode = generateShortCode(6)
		logger.CtxDebug(ctx, "Generated random short code", logger.LoggerInfo{
			ContextFunction: constant.CtxCreateShortURL,
			Data: map[string]interface{}{
				constant.DataShortCode: shortCode,
			},
		})
	}

	url := &URL{
		LongURL:   longURL,
		ShortCode: shortCode,
		CreatedAt: time.Now(),
		Visits:    0,
	}

	if err := s.repo.Store(ctx, url); err != nil {
		logger.CtxError(ctx, "Failed to store URL", logger.LoggerInfo{
			ContextFunction: constant.CtxCreateShortURL,
			Error: &logger.CustomError{
				Code:    constant.ErrCodeStorageFailure,
				Message: err.Error(),
				Type:    constant.ErrTypeStorage,
			},
			Data: map[string]interface{}{
				constant.DataLongURL:   longURL,
				constant.DataShortCode: shortCode,
			},
		})
		return nil, err
	}

	// ShortURLNamespace
	s.cache.Set(constant.ShortURLNamespace, shortCode, url)

	logger.CtxInfo(ctx, "URL successfully shortened", logger.LoggerInfo{
		ContextFunction: constant.CtxCreateShortURL,
		Data: map[string]interface{}{
			constant.DataShortCode: url.ShortCode,
			constant.DataLongURL:   url.LongURL,
			constant.DataCustom:    customShort != "",
		},
	})

	return url, nil
}

// GetLongURL retrieves the original URL from a short code
func (s *Service) GetLongURL(ctx context.Context, shortCode string) (*URL, error) {

	logger.CtxDebug(ctx, "Retrieving long URL", logger.LoggerInfo{
		ContextFunction: constant.CtxGetLongURL,
		Data: map[string]interface{}{
			constant.DataShortCode: shortCode,
		},
	})

	if shortCode == "" {
		logger.CtxWarn(ctx, "Short code cannot be empty", logger.LoggerInfo{
			ContextFunction: constant.CtxGetLongURL,
			Error: &logger.CustomError{
				Code:    constant.ErrCodeEmptyShortCode,
				Message: constant.ErrEmptyShortCode,
				Type:    constant.ErrTypeValidation,
			},
		})
		return nil, errors.New(constant.ErrEmptyShortCode)
	}

	val, found := s.cache.Get(constant.ShortURLNamespace, shortCode)
	if found {
		if urlObj, ok := val.(*URL); ok {
			// Cache hit, log and return
			logger.CtxInfo(ctx, "Long URL retrieved from cache", logger.LoggerInfo{
				ContextFunction: constant.CtxGetLongURL,
				Data: map[string]interface{}{
					constant.DataShortCode: shortCode,
					constant.DataLongURL:   urlObj.LongURL,
					constant.DataVisits:    urlObj.Visits,
				},
			})
			if err := s.repo.IncrementVisits(ctx, shortCode); err != nil {
				// Log error but continue with the redirect
				logger.CtxWarn(ctx, "Failed to increment visit count", logger.LoggerInfo{
					ContextFunction: constant.CtxGetLongURL,
					Error: &logger.CustomError{
						Code:    constant.ErrCodeIncrementVisits,
						Message: err.Error(),
						Type:    constant.ErrTypeStats,
					},
					Data: map[string]interface{}{
						constant.DataShortCode: shortCode,
					},
				})
			} else {
				logger.CtxDebug(ctx, "Visit count incremented", logger.LoggerInfo{
					ContextFunction: constant.CtxGetLongURL,
					Data: map[string]interface{}{
						constant.DataShortCode: shortCode,
					},
				})
			}
			return urlObj, nil
		}
	}

	url, err := s.repo.FindByShortCode(ctx, shortCode)
	if err != nil {
		logger.CtxWarn(ctx, "Failed to find URL by short code", logger.LoggerInfo{
			ContextFunction: constant.CtxGetLongURL,
			Error: &logger.CustomError{
				Code:    constant.ErrCodeShortCodeNotFound,
				Message: err.Error(),
				Type:    constant.ErrTypeRetrieval,
			},
			Data: map[string]interface{}{
				constant.DataShortCode: shortCode,
			},
		})
		return nil, err
	}

	if err := s.repo.IncrementVisits(ctx, shortCode); err != nil {
		// Log error but continue with the redirect
		logger.CtxWarn(ctx, "Failed to increment visit count", logger.LoggerInfo{
			ContextFunction: constant.CtxGetLongURL,
			Error: &logger.CustomError{
				Code:    constant.ErrCodeIncrementVisits,
				Message: err.Error(),
				Type:    constant.ErrTypeStats,
			},
			Data: map[string]interface{}{
				constant.DataShortCode: shortCode,
			},
		})
	} else {
		logger.CtxDebug(ctx, "Visit count incremented", logger.LoggerInfo{
			ContextFunction: constant.CtxGetLongURL,
			Data: map[string]interface{}{
				constant.DataShortCode: shortCode,
			},
		})
	}

	logger.CtxInfo(ctx, "Long URL retrieved successfully", logger.LoggerInfo{
		ContextFunction: constant.CtxGetLongURL,
		Data: map[string]interface{}{
			constant.DataShortCode: shortCode,
			constant.DataLongURL:   url.LongURL,
			constant.DataVisits:    url.Visits,
		},
	})

	return url, nil
}

// UpdateLongURL updates the long URL for an existing short code
func (s *Service) UpdateLongURL(ctx context.Context, shortCode, newLongURL string) (*URL, error) {
	logger.CtxDebug(ctx, "Updating long URL", logger.LoggerInfo{
		ContextFunction: constant.CtxUpdateLongURL,
		Data: map[string]interface{}{
			constant.DataShortCode: shortCode,
			constant.DataLongURL:   newLongURL,
		},
	})

	if shortCode == "" {
		logger.CtxWarn(ctx, "Short code cannot be empty", logger.LoggerInfo{
			ContextFunction: constant.CtxUpdateLongURL,
			Error: &logger.CustomError{
				Code:    constant.ErrCodeEmptyShortCode,
				Message: constant.ErrEmptyShortCode,
				Type:    constant.ErrTypeValidation,
			},
		})
		return nil, errors.New(constant.ErrEmptyShortCode)
	}

	if newLongURL == "" {
		logger.CtxWarn(ctx, "Long URL cannot be empty", logger.LoggerInfo{
			ContextFunction: constant.CtxUpdateLongURL,
			Error: &logger.CustomError{
				Code:    constant.ErrCodeEmptyLongURL,
				Message: constant.ErrEmptyLongURL,
				Type:    constant.ErrTypeValidation,
			},
		})
		return nil, errors.New(constant.ErrEmptyLongURL)
	}

	// First check if the short code exists
	url, err := s.repo.FindByShortCode(ctx, shortCode)
	if err != nil {
		logger.CtxWarn(ctx, "Failed to find URL by short code", logger.LoggerInfo{
			ContextFunction: constant.CtxUpdateLongURL,
			Error: &logger.CustomError{
				Code:    constant.ErrCodeShortCodeNotFound,
				Message: err.Error(),
				Type:    constant.ErrTypeRetrieval,
			},
			Data: map[string]interface{}{
				constant.DataShortCode: shortCode,
			},
		})
		return nil, err
	}

	// Update the long URL
	err = s.repo.UpdateLongURL(ctx, shortCode, newLongURL)
	if err != nil {
		logger.CtxError(ctx, "Failed to update long URL", logger.LoggerInfo{
			ContextFunction: constant.CtxUpdateLongURL,
			Error: &logger.CustomError{
				Code:    constant.ErrCodeUpdateFailure,
				Message: err.Error(),
				Type:    constant.ErrTypeStorage,
			},
			Data: map[string]interface{}{
				constant.DataShortCode: shortCode,
				constant.DataLongURL:   newLongURL,
			},
		})
		return nil, err
	}

	// Update the URL object with the new long URL
	url.LongURL = newLongURL

	// Update the cache
	s.cache.Set(constant.ShortURLNamespace, shortCode, url)

	logger.CtxInfo(ctx, "URL successfully updated", logger.LoggerInfo{
		ContextFunction: constant.CtxUpdateLongURL,
		Data: map[string]interface{}{
			constant.DataShortCode: shortCode,
			constant.DataLongURL:   newLongURL,
		},
	})

	return url, nil
}

// generateShortCode generates a random short code of specified length
func generateShortCode(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[time.Now().UnixNano()%int64(len(charset))]
		time.Sleep(1 * time.Nanosecond) // Ensure uniqueness
	}
	return string(result)
}
