package transport

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func NewRouter(handler *Handler) http.Handler {
	r := chi.NewRouter()

	r.Use(RecoveryMiddleware)
	r.Use(LoggingMiddleware)
	r.Use(CORSMiddleware)

	r.Get("/health", handler.HandleHealth)
	r.Post("/api/games", handler.HandleCreateGame)
	r.Get("/api/games/{code}", handler.HandleGetGame)
	r.Get("/ws", handler.HandleWebSocket)

	return r
}
