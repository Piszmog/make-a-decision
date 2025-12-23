package router

import (
	"github.com/Piszmog/make-a-decision/internal/db"
	"github.com/Piszmog/make-a-decision/internal/dist"
	"github.com/Piszmog/make-a-decision/internal/server/handler"
	"github.com/Piszmog/make-a-decision/internal/server/middleware"
	"log/slog"
	"net/http"
)

func New(logger *slog.Logger, database db.Database) http.Handler {
	h := &handler.Handler{
		Logger:   logger,
		Database: database,
	}

	// Create user context middleware
	userContextMiddleware := &middleware.UserContextMiddleware{
		Logger:   logger,
		Database: database,
	}

	mux := http.NewServeMux()

	// Static assets (no middleware needed)
	mux.Handle(newPath(http.MethodGet, "/assets/"), middleware.CacheMiddleware(http.FileServer(http.FS(dist.AssetsDir))))

	// Public routes
	mux.HandleFunc(newPath(http.MethodGet, "/"), h.Home)
	mux.HandleFunc(newPath(http.MethodPost, "/api/random"), h.RandomPicker)

	// Options management endpoints (public for now)
	mux.HandleFunc(newPath(http.MethodGet, "/manage/options"), h.GetOptions)
	mux.HandleFunc(newPath(http.MethodPost, "/api/options"), h.AddOption)
	mux.HandleFunc(newPath(http.MethodPost, "/api/options/update"), h.UpdateOptionDetails)
	mux.HandleFunc(newPath(http.MethodGet, "/expand-option/"), h.ExpandOption)
	mux.HandleFunc(newPath(http.MethodGet, "/collapse-option/"), h.CollapseOption)
	mux.HandleFunc(newPath(http.MethodPost, "/api/weight/increase/"), h.IncreaseWeight)
	mux.HandleFunc(newPath(http.MethodPost, "/api/weight/decrease/"), h.DecreaseWeight)
	mux.HandleFunc(newPath(http.MethodDelete, "/api/options/"), h.DeleteOption)
	mux.HandleFunc(newPath(http.MethodGet, "/close-modal"), h.CloseModal)

	// Authentication endpoints
	mux.HandleFunc(newPath(http.MethodGet, "/signin"), h.SigninPage)
	mux.HandleFunc(newPath(http.MethodPost, "/api/signin"), h.Authenticate)
	mux.HandleFunc(newPath(http.MethodGet, "/signout"), h.Signout)
	mux.HandleFunc(newPath(http.MethodGet, "/signup"), h.SignupPage)
	mux.HandleFunc(newPath(http.MethodPost, "/api/signup"), h.SignupSubmit)

	// Local storage sync endpoint
	mux.HandleFunc(newPath(http.MethodPost, "/api/sync-local-options"), h.SyncLocalOptions)

	// Chain middlewares: logging -> user context -> routes
	return middleware.Chain(
		middleware.NewLoggingMiddleware(logger, nil).Middleware,
		userContextMiddleware.Middleware,
	)(mux)
}

func newPath(method string, path string) string {
	return method + " " + path
}
