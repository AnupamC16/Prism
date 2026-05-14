package api

import (
	"context"
	"crypto/rand"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/anupam-chopra/prism/internal/logger"
)

type responseWriter struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int
	wroteHeader  bool
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}
}

func (rw *responseWriter) Header() http.Header {
	return rw.ResponseWriter.Header()
}

func (rw *responseWriter) WriteHeader(code int) {
	if rw.wroteHeader {
		return
	}
	rw.statusCode = code
	rw.wroteHeader = true
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.wroteHeader {
		rw.WriteHeader(http.StatusOK)
	}
	n, err := rw.ResponseWriter.Write(b)
	rw.bytesWritten += n
	return n, err
}

func Recovery(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rw, ok := w.(*responseWriter)
			if !ok {
				rw = newResponseWriter(w)
			}

			defer func() {
				if err := recover(); err != nil {
					log.Error("panic recovered",
						slog.Any("panic_value", err),
						slog.String("stack_trace", string(debug.Stack())),
						slog.String("request_id", logger.GetRequestID(r.Context())),
					)
					if !rw.wroteHeader {
						rw.Header().Set("Content-Type", "application/json")
						rw.WriteHeader(http.StatusInternalServerError)
						_, _ = rw.Write([]byte(`{"success":false,"error":{"code":"INTERNAL_ERROR","message":"internal server error"}}`))
					}
				}
			}()

			next.ServeHTTP(rw, r)
		})
	}
}

func RequestID() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := r.Header.Get("X-Request-ID")
			if id == "" {
				b := make([]byte, 16)
				_, _ = rand.Read(b)
				id = fmt.Sprintf("%x", b)
			}

			ctx := logger.WithRequestID(r.Context(), id)
			w.Header().Set("X-Request-ID", id)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func RequestLogger(log *slog.Logger) func(http.Handler) http.Handler {
	return logger.NewRequestLogger(log)
}

func Timeout(duration time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), duration)
			defer cancel()

			next.ServeHTTP(w, r.WithContext(ctx))

			if ctx.Err() != nil {
				slog.Warn("request timeout",
					slog.String("request_id", logger.GetRequestID(r.Context())),
					slog.String("path", r.URL.Path),
				)
			}
		})
	}
}

func CORS() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID, X-DRM-Token, X-Asset-ID")
			w.Header().Set("Access-Control-Expose-Headers", "X-Request-ID")
			w.Header().Set("Access-Control-Max-Age", "86400")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
