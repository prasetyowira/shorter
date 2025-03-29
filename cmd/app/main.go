package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/prasetyowira/shorter/api"
	"github.com/prasetyowira/shorter/config"
	"github.com/prasetyowira/shorter/constant"
	"github.com/prasetyowira/shorter/domain/shortener"
	"github.com/prasetyowira/shorter/infrastructure/cache"
	"github.com/prasetyowira/shorter/infrastructure/db"
	appLogger "github.com/prasetyowira/shorter/infrastructure/logger"
)

func main() {
	// Load configuration from environment variables
	cfg := config.LoadConfig()

	// Initialize logger based on environment
	isProduction := cfg.LogLevel == "INFO"
	appLogger.Initialize(isProduction)
	defer appLogger.Close()

	appLogger.Info(constant.MsgApplicationStarting, appLogger.LoggerInfo{
		ContextFunction: constant.CtxMain,
		Data: map[string]interface{}{
			constant.DataPort:        cfg.Port,
			constant.DataDBPath:      cfg.DatabaseURL,
			constant.DataEnvironment: cfg.LogLevel,
		},
	})

	// Create SQLite repository
	repository, err := db.NewSQLiteRepository(cfg.DatabaseURL)
	if err != nil {
		appLogger.Fatal(constant.MsgFailedToInitDB, appLogger.LoggerInfo{
			ContextFunction: constant.CtxMain,
			Error: &appLogger.CustomError{
				Code:    constant.ErrCodeAppDBInit,
				Message: err.Error(),
				Type:    constant.ErrTypeApp,
			},
			Data: map[string]interface{}{
				constant.DataDBPath: cfg.DatabaseURL,
			},
		})
	}
	defer repository.Close()

	cacheLRU := cache.NewNamespaceLRU(cfg.CacheSize)
	// Create shortener service
	service := shortener.NewService(repository, cacheLRU)

	// Create API handler and router
	handler := api.NewHandler(service)
	router := api.NewRouter(handler, cfg.AuthUser, cfg.AuthPass)
	router.SetupRoutes()

	// Configure HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		appLogger.Info(constant.MsgServerStarting, appLogger.LoggerInfo{
			ContextFunction: constant.CtxMain,
			Data: map[string]interface{}{
				constant.DataPort: cfg.Port,
			},
		})

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			appLogger.Fatal(constant.MsgServerFailedToStart, appLogger.LoggerInfo{
				ContextFunction: constant.CtxMain,
				Error: &appLogger.CustomError{
					Code:    constant.ErrCodeAppServerStart,
					Message: err.Error(),
					Type:    constant.ErrTypeApp,
				},
				Data: map[string]interface{}{
					constant.DataPort: cfg.Port,
				},
			})
		}
	}()

	// Set up graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit

	appLogger.Info(constant.MsgServerShuttingDown, appLogger.LoggerInfo{
		ContextFunction: constant.CtxMain,
	})

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		appLogger.Error(constant.MsgServerShutdownError, appLogger.LoggerInfo{
			ContextFunction: constant.CtxMain,
			Error: &appLogger.CustomError{
				Code:    constant.ErrCodeAppServerShutdown,
				Message: err.Error(),
				Type:    constant.ErrTypeApp,
			},
		})
	}

	appLogger.Info(constant.MsgServerStopped, appLogger.LoggerInfo{
		ContextFunction: constant.CtxMain,
	})
}
