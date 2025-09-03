package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type Response struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

func TimeoutMiddleware(timeout time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()

			r = r.WithContext(ctx)

			done := make(chan struct{})
			go func() {
				defer close(done)
				next.ServeHTTP(w, r)
			}()

			select {
			case <-done:
				return
			case <-ctx.Done():
				if errors.Is(ctx.Err(), context.DeadlineExceeded) {
					response := Response{
						Message: "Request timeout",
						Error:   "timeout",
					}
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusRequestTimeout)
					if err := json.NewEncoder(w).Encode(response); err != nil {
						log.Printf("Error encoding response: %v", err)
					}
				} else {
					response := Response{
						Message: "Request cancelled",
						Error:   "cancelled",
					}
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)
					if err := json.NewEncoder(w).Encode(response); err != nil {
						log.Printf("Error encoding response: %v", err)
					}
				}
				return
			}
		})
	}
}

func RequestIDMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := uuid.New().String()
			ctx := context.WithValue(r.Context(), "request_id", requestID)
			r = r.WithContext(ctx)
			w.Header().Set("X-Request-ID", requestID)
			next.ServeHTTP(w, r)
		})
	}
}
