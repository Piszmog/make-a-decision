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

	mux := http.NewServeMux()

	mux.Handle(newPath(http.MethodGet, "/assets/"), middleware.CacheMiddleware(http.FileServer(http.FS(dist.AssetsDir))))
	mux.HandleFunc(newPath(http.MethodGet, "/"), h.Home)
	mux.HandleFunc(newPath(http.MethodPost, "/api/random"), h.RandomPicker)

	// Options management endpoints
	mux.HandleFunc(newPath(http.MethodGet, "/manage/options"), h.GetOptions)
	mux.HandleFunc(newPath(http.MethodPost, "/api/options"), h.AddOption)
	mux.HandleFunc(newPath(http.MethodPost, "/api/options/update"), h.UpdateOptionDetails)
	mux.HandleFunc(newPath(http.MethodGet, "/expand-option/"), h.ExpandOption)
	mux.HandleFunc(newPath(http.MethodGet, "/collapse-option/"), h.CollapseOption)
	mux.HandleFunc(newPath(http.MethodPost, "/api/weight/increase/"), h.IncreaseWeight)
	mux.HandleFunc(newPath(http.MethodPost, "/api/weight/decrease/"), h.DecreaseWeight)
	mux.HandleFunc(newPath(http.MethodDelete, "/api/options/"), h.DeleteOption)

	mux.HandleFunc(newPath(http.MethodGet, "/close-modal"), h.CloseModal)

	return middleware.NewLoggingMiddleware(logger, mux)
}

func newPath(method string, path string) string {
	return method + " " + path
}
