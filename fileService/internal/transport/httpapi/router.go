package httpapi

import (
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

func NewRouter(handler *Handler) *mux.Router {
	router := mux.NewRouter()
	router.Use(jsonContentTypeMiddleware)

	router.HandleFunc("/files", handler.Upload).Methods(http.MethodPost)
	router.HandleFunc("/files/{id}", handler.GetMetadata).Methods(http.MethodGet)
	router.HandleFunc("/files/{id}", handler.Delete).Methods(http.MethodDelete)
	router.HandleFunc("/files/{id}/content", handler.GetContent).Methods(http.MethodGet)
	router.HandleFunc("/files/{id}/download", handler.GetDownloadLink).Methods(http.MethodGet)

	return router
}

func jsonContentTypeMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Request-Started-At", time.Now().UTC().Format(time.RFC3339))
		next.ServeHTTP(w, r)
	})
}
