package db

import (
	"context"
	"errors"
	"github.com/prasetyowira/shorter/infrastructure/cache"
	"time"

	"github.com/prasetyowira/shorter/constant"
	"github.com/prasetyowira/shorter/domain/shortener"
	appLogger "github.com/prasetyowira/shorter/infrastructure/logger"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
)

// SQLiteRepository implements shortener.Repository interface
type SQLiteRepository struct {
	db    *gorm.DB
	cache *cache.NamespaceLRU
}

// URLModel is the GORM model for URL entity
type URLModel struct {
	ID        uint   `gorm:"primaryKey"`
	LongURL   string `gorm:"not null"`
	ShortCode string `gorm:"uniqueIndex;not null"`
	CreatedAt time.Time
	Visits    uint
}

// GormLogger implements GORM's logger.Interface
type GormLogger struct{}

// LogMode implements the log.Interface method
func (l *GormLogger) LogMode(level gormLogger.LogLevel) gormLogger.Interface {
	return l
}

// Info logs info messages
func (l *GormLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	appLogger.CtxInfo(ctx, msg, appLogger.LoggerInfo{
		ContextFunction: constant.CtxDB,
		Data: map[string]interface{}{
			constant.DataData: data,
		},
	})
}

// Warn logs warn messages
func (l *GormLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	appLogger.CtxWarn(ctx, msg, appLogger.LoggerInfo{
		ContextFunction: constant.CtxDB,
		Data: map[string]interface{}{
			constant.DataData: data,
		},
	})
}

// Error logs error messages
func (l *GormLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	appLogger.CtxError(ctx, msg, appLogger.LoggerInfo{
		ContextFunction: constant.CtxDB,
		Error: &appLogger.CustomError{
			Code:    constant.ErrCodeDBGeneral,
			Message: msg,
			Type:    constant.ErrTypeDB,
		},
		Data: map[string]interface{}{
			constant.DataData: data,
		},
	})
}

// Trace logs SQL operations
func (l *GormLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	elapsed := time.Since(begin)
	sql, rows := fc()

	if err != nil {
		appLogger.CtxError(ctx, "SQL error", appLogger.LoggerInfo{
			ContextFunction: constant.CtxDB,
			Error: &appLogger.CustomError{
				Code:    constant.ErrCodeDBGeneral,
				Message: err.Error(),
				Type:    constant.ErrTypeDB,
			},
			Data: map[string]interface{}{
				constant.DataElapsed: elapsed.String(),
				constant.DataRows:    rows,
				constant.DataSQL:     sql,
			},
		})
		return
	}

	// Only log SQL queries if in debug mode
	appLogger.CtxDebug(ctx, "SQL query", appLogger.LoggerInfo{
		ContextFunction: constant.CtxDB,
		Data: map[string]interface{}{
			constant.DataElapsed: elapsed.String(),
			constant.DataRows:    rows,
			constant.DataSQL:     sql,
		},
	})
}

// NewSQLiteRepository creates a new SQLite repository
func NewSQLiteRepository(dbPath string, cacheObj *cache.NamespaceLRU) (*SQLiteRepository, error) {
	ctx := appLogger.NewRequestContext()

	appLogger.CtxDebug(ctx, "Opening SQLite database", appLogger.LoggerInfo{
		ContextFunction: constant.CtxDB,
		Data: map[string]interface{}{
			constant.DataPath: dbPath,
		},
	})

	dbLogger := &GormLogger{}

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: dbLogger,
	})
	if err != nil {
		appLogger.CtxError(ctx, "Failed to open database", appLogger.LoggerInfo{
			ContextFunction: constant.CtxDB,
			Error: &appLogger.CustomError{
				Code:    constant.ErrCodeDBOpen,
				Message: err.Error(),
				Type:    constant.ErrTypeDB,
			},
			Data: map[string]interface{}{
				constant.DataPath: dbPath,
			},
		})
		return nil, err
	}

	// Auto-migrate the schema
	if err := db.AutoMigrate(&URLModel{}); err != nil {
		appLogger.CtxError(ctx, "Failed to migrate database schema", appLogger.LoggerInfo{
			ContextFunction: constant.CtxDB,
			Error: &appLogger.CustomError{
				Code:    constant.ErrCodeDBMigrate,
				Message: err.Error(),
				Type:    constant.ErrTypeDB,
			},
		})
		return nil, err
	}

	appLogger.CtxInfo(ctx, "Database initialized successfully", appLogger.LoggerInfo{
		ContextFunction: constant.CtxDB,
		Data: map[string]interface{}{
			constant.DataPath: dbPath,
		},
	})

	return &SQLiteRepository{db: db, cache: cacheObj}, nil
}

// Store persists a URL to the database
func (r *SQLiteRepository) Store(ctx context.Context, url *shortener.URL) error {
	// Check if shortcode already exists
	var count int64
	err := r.db.Raw(`SELECT COUNT(*) FROM url_models WHERE short_code = ?`, url.ShortCode).Count(&count).Error
	if err != nil {
		appLogger.CtxError(ctx, "Error checking for existing short code", appLogger.LoggerInfo{
			ContextFunction: constant.CtxStore,
			Error: &appLogger.CustomError{
				Code:    constant.ErrCodeDBCheckExists,
				Message: err.Error(),
				Type:    constant.ErrTypeDB,
			},
			Data: map[string]interface{}{
				constant.DataShortCode: url.ShortCode,
			},
		})
		return err
	}

	if count > 0 {
		appLogger.CtxWarn(ctx, "Short code already exists", appLogger.LoggerInfo{
			ContextFunction: constant.CtxStore,
			Data: map[string]interface{}{
				constant.DataShortCode: url.ShortCode,
			},
		})
		return errors.New(constant.ErrShortCodeExists)
	}

	model := URLModel{
		LongURL:   url.LongURL,
		ShortCode: url.ShortCode,
		CreatedAt: url.CreatedAt,
		Visits:    url.Visits,
	}

	result := r.db.Exec(`INSERT INTO url_models (long_url, short_code, created_at, visits) VALUES (?, ?, ?, ?)`,
		model.LongURL, model.ShortCode, model.CreatedAt, model.Visits)

	if result.Error != nil {
		appLogger.CtxError(ctx, "Failed to insert URL", appLogger.LoggerInfo{
			ContextFunction: constant.CtxStore,
			Error: &appLogger.CustomError{
				Code:    constant.ErrCodeDBInsert,
				Message: result.Error.Error(),
				Type:    constant.ErrTypeDB,
			},
			Data: map[string]interface{}{
				constant.DataShortCode: url.ShortCode,
				constant.DataLongURL:   url.LongURL,
			},
		})
		return result.Error
	}

	url.ID = model.ID

	appLogger.CtxInfo(ctx, "URL stored successfully", appLogger.LoggerInfo{
		ContextFunction: constant.CtxStore,
		Data: map[string]interface{}{
			constant.DataShortCode: url.ShortCode,
			constant.DataLongURL:   url.LongURL,
		},
	})

	return nil
}

// FindByShortCode retrieves a URL by its short code
func (r *SQLiteRepository) FindByShortCode(ctx context.Context, shortCode string) (*shortener.URL, error) {
	var model URLModel

	appLogger.CtxDebug(ctx, "Looking up short code", appLogger.LoggerInfo{
		ContextFunction: constant.CtxFindByShortCode,
		Data: map[string]interface{}{
			constant.DataShortCode: shortCode,
		},
	})

	rows, err := r.db.Raw(`SELECT id, long_url, short_code, created_at, visits FROM url_models WHERE short_code = ? LIMIT 1`, shortCode).Rows()
	if err != nil {
		appLogger.CtxError(ctx, "Database error while looking up short code", appLogger.LoggerInfo{
			ContextFunction: constant.CtxFindByShortCode,
			Error: &appLogger.CustomError{
				Code:    constant.ErrCodeDBLookup,
				Message: err.Error(),
				Type:    constant.ErrTypeDB,
			},
			Data: map[string]interface{}{
				constant.DataShortCode: shortCode,
			},
		})
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		appLogger.CtxInfo(ctx, "Short code not found", appLogger.LoggerInfo{
			ContextFunction: constant.CtxFindByShortCode,
			Data: map[string]interface{}{
				constant.DataShortCode: shortCode,
			},
		})
		return nil, errors.New(constant.ErrShortCodeNotFound)
	}

	if err := r.db.ScanRows(rows, &model); err != nil {
		appLogger.CtxError(ctx, "Failed to scan database rows", appLogger.LoggerInfo{
			ContextFunction: constant.CtxFindByShortCode,
			Error: &appLogger.CustomError{
				Code:    constant.ErrCodeDBScanRows,
				Message: err.Error(),
				Type:    constant.ErrTypeDB,
			},
			Data: map[string]interface{}{
				constant.DataShortCode: shortCode,
			},
		})
		return nil, err
	}

	if err := rows.Err(); err != nil {
		appLogger.CtxError(ctx, "Row iteration error", appLogger.LoggerInfo{
			ContextFunction: constant.CtxFindByShortCode,
			Error: &appLogger.CustomError{
				Code:    constant.ErrCodeDBRowIterate,
				Message: err.Error(),
				Type:    constant.ErrTypeDB,
			},
			Data: map[string]interface{}{
				constant.DataShortCode: shortCode,
			},
		})
		return nil, err
	}

	appLogger.CtxDebug(ctx, "Short code found", appLogger.LoggerInfo{
		ContextFunction: constant.CtxFindByShortCode,
		Data: map[string]interface{}{
			constant.DataShortCode: shortCode,
			constant.DataLongURL:   model.LongURL,
			constant.DataVisits:    model.Visits,
		},
	})

	return &shortener.URL{
		ID:        model.ID,
		LongURL:   model.LongURL,
		ShortCode: model.ShortCode,
		CreatedAt: model.CreatedAt,
		Visits:    model.Visits,
	}, nil
}

// IncrementVisits increments the visit count for a URL
func (r *SQLiteRepository) IncrementVisits(ctx context.Context, shortCode string) error {
	result := r.db.Exec(`UPDATE url_models SET visits = visits + 1 WHERE short_code = ?`, shortCode)

	if result.Error != nil {
		appLogger.CtxError(ctx, "Failed to increment visit count", appLogger.LoggerInfo{
			ContextFunction: constant.CtxIncrementVisits,
			Error: &appLogger.CustomError{
				Code:    constant.ErrCodeDBIncrement,
				Message: result.Error.Error(),
				Type:    constant.ErrTypeDB,
			},
			Data: map[string]interface{}{
				constant.DataShortCode: shortCode,
			},
		})
		return result.Error
	}

	if result.RowsAffected == 0 {
		appLogger.CtxWarn(ctx, "No rows affected when incrementing visits", appLogger.LoggerInfo{
			ContextFunction: constant.CtxIncrementVisits,
			Data: map[string]interface{}{
				constant.DataShortCode: shortCode,
			},
		})
	} else {
		appLogger.CtxDebug(ctx, "Visit count incremented", appLogger.LoggerInfo{
			ContextFunction: constant.CtxIncrementVisits,
			Data: map[string]interface{}{
				constant.DataShortCode:    shortCode,
				constant.DataRowsAffected: result.RowsAffected,
			},
		})
		// Get url obj from cache
		urlObj, found := r.cache.Get(constant.ShortURLNamespace, shortCode)
		if found {
			if url, ok := urlObj.(*shortener.URL); ok {
				url.Visits++
				// Update the cache
				r.cache.Set(constant.ShortURLNamespace, shortCode, url)
			}
		}
	}

	return nil
}

// UpdateLongURL updates the long URL for an existing short code
func (r *SQLiteRepository) UpdateLongURL(ctx context.Context, shortCode string, newLongURL string) error {
	appLogger.CtxDebug(ctx, "Updating long URL in database", appLogger.LoggerInfo{
		ContextFunction: constant.CtxUpdateLongURL,
		Data: map[string]interface{}{
			constant.DataShortCode: shortCode,
			constant.DataLongURL:   newLongURL,
		},
	})

	// Check if shortcode exists
	var count int64
	err := r.db.Raw(`SELECT COUNT(*) FROM url_models WHERE short_code = ?`, shortCode).Count(&count).Error
	if err != nil {
		appLogger.CtxError(ctx, "Error checking for existing short code", appLogger.LoggerInfo{
			ContextFunction: constant.CtxUpdateLongURL,
			Error: &appLogger.CustomError{
				Code:    constant.ErrCodeDBCheckExists,
				Message: err.Error(),
				Type:    constant.ErrTypeDB,
			},
			Data: map[string]interface{}{
				constant.DataShortCode: shortCode,
			},
		})
		return err
	}

	if count == 0 {
		appLogger.CtxWarn(ctx, "Short code not found", appLogger.LoggerInfo{
			ContextFunction: constant.CtxUpdateLongURL,
			Error: &appLogger.CustomError{
				Code:    constant.ErrCodeShortCodeNotFound,
				Message: constant.ErrShortCodeNotFound,
				Type:    constant.ErrTypeDB,
			},
			Data: map[string]interface{}{
				constant.DataShortCode: shortCode,
			},
		})
		return errors.New(constant.ErrShortCodeNotFound)
	}

	// Update the long URL
	result := r.db.Exec(`UPDATE url_models SET long_url = ? WHERE short_code = ?`, newLongURL, shortCode)
	if result.Error != nil {
		appLogger.CtxError(ctx, "Failed to update long URL in database", appLogger.LoggerInfo{
			ContextFunction: constant.CtxUpdateLongURL,
			Error: &appLogger.CustomError{
				Code:    constant.ErrCodeUpdateFailure,
				Message: result.Error.Error(),
				Type:    constant.ErrTypeDB,
			},
			Data: map[string]interface{}{
				constant.DataShortCode: shortCode,
				constant.DataLongURL:   newLongURL,
			},
		})
		return result.Error
	}

	if result.RowsAffected == 0 {
		appLogger.CtxWarn(ctx, "No rows updated", appLogger.LoggerInfo{
			ContextFunction: constant.CtxUpdateLongURL,
			Data: map[string]interface{}{
				constant.DataShortCode: shortCode,
				constant.DataRowsAffected: 0,
			},
		})
		return errors.New(constant.ErrShortCodeNotFound)
	}

	appLogger.CtxInfo(ctx, "Long URL updated successfully in database", appLogger.LoggerInfo{
		ContextFunction: constant.CtxUpdateLongURL,
		Data: map[string]interface{}{
			constant.DataShortCode: shortCode,
			constant.DataLongURL:   newLongURL,
			constant.DataRowsAffected: result.RowsAffected,
		},
	})

	return nil
}

// Close closes the database connection
func (r *SQLiteRepository) Close() error {
	ctx := context.Background()
	sqlDB, err := r.db.DB()
	if err != nil {
		appLogger.CtxError(ctx, "Failed to get database connection", appLogger.LoggerInfo{
			ContextFunction: constant.CtxClose,
			Error: &appLogger.CustomError{
				Code:    constant.ErrCodeDBClose,
				Message: err.Error(),
				Type:    constant.ErrTypeDB,
			},
		})
		return err
	}

	appLogger.CtxInfo(ctx, "Closing database connection", appLogger.LoggerInfo{
		ContextFunction: constant.CtxClose,
	})

	return sqlDB.Close()
}
