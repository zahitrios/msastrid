package middlewares

import (
	"net/http"
	"strings"

	"go.mongodb.org/mongo-driver/mongo"

	"ms-astrid/products/services"
	"ms-astrid/products/utils"
)

func CorsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, PATCH, DELETE, UPDATE")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		w.Header().Set("Access-Control-Expose-Headers", "Content-Length")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}

func LoggerMiddleware(db *mongo.Client) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rw := utils.NewResponseWriter(w, r)
			next.ServeHTTP(rw, r)

			if r.Method != http.MethodGet && r.URL.Path != "/logs" && !strings.HasPrefix(r.UserAgent(), "curl") {
				loggerService := services.NewLoggerService(db)
				loggerService.LogRequest(rw, r)
			}
		})
	}
}
