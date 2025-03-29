package middleware

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/prasetyowira/shorter/constant"
	appLogger "github.com/prasetyowira/shorter/infrastructure/logger"
)

// RequestLogger is middleware that adds request ID to the context and logs request/response info
func RequestLogger() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Generate a unique request ID
			requestID := uuid.New().String()

			// Set request ID in context
			ctx := appLogger.WithRequestID(r.Context(), requestID)

			// Add request ID to the response headers
			w.Header().Set(constant.HeaderRequestID, requestID)

			// Log request
			appLogger.CtxInfo(ctx, constant.MsgRequestReceived, appLogger.LoggerInfo{
				ContextFunction: constant.CtxAPI,
				Data: map[string]interface{}{
					constant.DataMethod:     r.Method,
					constant.DataPath:       r.URL.Path,
					constant.DataRemoteAddr: r.RemoteAddr,
					constant.DataUserAgent:  r.UserAgent(),
				},
			})

			// Create a response wrapper to capture status code
			ww := newStatusResponseWriter(w)

			// Process request
			startTime := time.Now()
			next.ServeHTTP(ww, r.WithContext(ctx))

			// Calculate latency
			latency := time.Since(startTime)

			// Log response
			statusCode := ww.status
			logFunc := appLogger.CtxInfo

			if statusCode >= 400 && statusCode < 500 {
				logFunc = appLogger.CtxWarn
			} else if statusCode >= 500 {
				logFunc = appLogger.CtxError
			}

			logFunc(ctx, "Request completed", appLogger.LoggerInfo{
				ContextFunction: constant.CtxAPI,
				Data: map[string]interface{}{
					"status":  statusCode,
					"latency": latency.String(),
					"method":  r.Method,
					"path":    r.URL.Path,
					"size":    ww.size,
				},
			})
		})
	}
}

// statusResponseWriter is a custom response writer that captures the status code and response size
type statusResponseWriter struct {
	http.ResponseWriter
	status int
	size   int
}

// newStatusResponseWriter creates a new statusResponseWriter
func newStatusResponseWriter(w http.ResponseWriter) *statusResponseWriter {
	return &statusResponseWriter{
		ResponseWriter: w,
		status:         http.StatusOK, // Default status code
	}
}

// WriteHeader captures the status code
func (w *statusResponseWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

// Write captures the response size
func (w *statusResponseWriter) Write(b []byte) (int, error) {
	size, err := w.ResponseWriter.Write(b)
	w.size += size
	return size, err
}
