package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prasetyowira/shorter/constant"
	appLogger "github.com/prasetyowira/shorter/infrastructure/logger"
)

// Router represents the application router
type Router struct {
	handler  *Handler
	router   *chi.Mux
	username string
	password string
}

// NewRouter creates a new router
func NewRouter(handler *Handler, username, password string) *Router {
	r := chi.NewRouter()

	// Middleware setup
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(withRequestID)
	r.Use(logRequest)

	return &Router{
		handler:  handler,
		router:   r,
		username: username,
		password: password,
	}
}

// SetupRoutes configures all application routes
func (r *Router) SetupRoutes() {
	appLogger.Info(constant.MsgSettingUpRoutes, appLogger.LoggerInfo{
		ContextFunction: constant.CtxRouter,
	})

	creds := map[string]string{
		r.username: r.password,
	}
	// API routes with Basic Auth
	r.router.With(
		middleware.BasicAuth("shorter", creds),
	).Post(constant.RouteCreateShortURL, r.handler.CreateShortURL)

	r.router.With(
		middleware.BasicAuth("shorter", creds),
	).Put(constant.RouteUpdateLongURL, r.handler.UpdateLongURL)

	// Public routes
	r.router.Get(constant.RouteShortCodeRedirect, r.handler.RedirectToLongURL)
	r.router.Get(constant.RouteURLStats, r.handler.GetURLStats)
	r.router.Get(constant.RouteQRCode, r.handler.GenerateQRCode)

	// Healthcheck
	r.router.Get(constant.RouteHealthcheck, func(w http.ResponseWriter, r *http.Request) {
		appLogger.CtxDebug(r.Context(), constant.MsgHealthcheckRequest, appLogger.LoggerInfo{
			ContextFunction: constant.CtxRouter,
		})

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(constant.MsgHealthy))
	})
}

// ServeHTTP implements the http.Handler interface
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.router.ServeHTTP(w, req)
}
