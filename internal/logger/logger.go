package logger

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"
)

type responseWriter struct {
	wrapped      http.ResponseWriter
	statusCode   int
	bytesWritten int
	wroteHeader  bool
}

func (rw *responseWriter) Header() http.Header {
	return rw.wrapped.Header()
}

func (rw *responseWriter) WriteHeader(code int) {
	if rw.wroteHeader {
		return
	}
	rw.statusCode = code
	rw.wroteHeader = true
	rw.wrapped.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.wroteHeader {
		rw.WriteHeader(http.StatusOK)
	}
	n, err := rw.wrapped.Write(b)
	rw.bytesWritten += n
	return n, err
}

func New(level string) *slog.Logger {
	var logLevel slog.Level
	switch level {
	case "debug":
		logLevel = slog.LevelDebug
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	case "info":
		fallthrough
	default:
		logLevel = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		AddSource: false,
		Level:     logLevel,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey && len(groups) == 0 {
				return slog.Int64("ts", a.Value.Time().UnixMilli())
			}
			return a
		},
	}

	handler := slog.NewJSONHandler(os.Stdout, opts)
	return slog.New(handler)
}

type contextKey string

const requestIDKey contextKey = "request_id"

func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

func GetRequestID(ctx context.Context) string {
	val := ctx.Value(requestIDKey)
	if val == nil {
		return ""
	}
	id, ok := val.(string)
	if !ok {
		return ""
	}
	return id
}

func NewRequestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			rw := &responseWriter{
				wrapped:    w,
				statusCode: http.StatusOK, // Default status code if WriteHeader is not called
			}

			next.ServeHTTP(rw, r)

			duration := time.Since(start).Milliseconds()

			level := slog.LevelInfo
			if rw.statusCode >= 400 && rw.statusCode < 500 {
				level = slog.LevelWarn
			} else if rw.statusCode >= 500 {
				level = slog.LevelError
			}

			logger.Log(r.Context(), level, "request completed",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", rw.statusCode),
				slog.Int64("duration_ms", duration),
				slog.String("request_id", GetRequestID(r.Context())),
				slog.Int("bytes", rw.bytesWritten),
				slog.String("remote_addr", r.RemoteAddr),
				slog.String("user_agent", r.Header.Get("User-Agent")),
			)
		})
	}
}

func NewDRMAuditLogger(logger *slog.Logger) *slog.Logger {
	return logger.With("component", "drm_audit")
}
