package logger

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const maxRequestBody = 1 << 20

// Config configures a Logger.
type Config struct {
	Service     string
	Mode        string
	Development bool
}

// Logger owns a Zap logger and its runtime-adjustable level.
type Logger struct {
	logger *zap.Logger
	level  zap.AtomicLevel
}

type modeResponse struct {
	Mode string `json:"mode"`
}

// New constructs a logger. Mode defaults to info.
func New(cfg Config, opts ...zap.Option) (*Logger, error) {
	level, err := parseMode(cfg.Mode)
	if err != nil {
		return nil, err
	}

	atomicLevel := zap.NewAtomicLevelAt(level)
	zapConfig := zap.NewProductionConfig()
	if cfg.Development {
		zapConfig = zap.NewDevelopmentConfig()
	}
	zapConfig.Level = atomicLevel

	zapLogger, err := zapConfig.Build(opts...)
	if err != nil {
		return nil, fmt.Errorf("build zap logger: %w", err)
	}
	if cfg.Service != "" {
		zapLogger = zapLogger.Named(cfg.Service)
	}

	return &Logger{
		logger: zapLogger,
		level:  atomicLevel,
	}, nil
}

// Zap returns the underlying Zap logger for packages that require *zap.Logger.
func (l *Logger) Zap() *zap.Logger {
	return l.logger
}

// Mode returns the active logging level.
func (l *Logger) Mode() string {
	return l.level.Level().String()
}

// SetMode changes the logging level for this logger and all loggers derived
// from it with Named or With.
func (l *Logger) SetMode(mode string) error {
	level, err := parseMode(mode)
	if err != nil {
		return err
	}

	l.level.SetLevel(level)
	return nil
}

// Sync flushes buffered log entries.
func (l *Logger) Sync() error {
	return l.logger.Sync()
}

// Handler returns an HTTP handler for reading and changing the runtime mode.
// Mount it on an internal management server at /log/mode.
func (l *Logger) Handler() http.Handler {
	return http.HandlerFunc(l.handleMode)
}

// RegisterHandler registers the runtime log-level endpoint on mux.
func (l *Logger) RegisterHandler(mux *http.ServeMux) {
	mux.Handle("GET /log/mode", l.Handler())
	mux.Handle("PUT /log/mode", l.Handler())
}

func (l *Logger) handleMode(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, modeResponse{Mode: l.Mode()})
	case http.MethodPut:
		l.updateMode(w, r)
	default:
		w.Header().Set("Allow", "GET, PUT")
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (l *Logger) updateMode(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	decoder := json.NewDecoder(io.LimitReader(r.Body, maxRequestBody))
	decoder.DisallowUnknownFields()

	var request modeResponse
	if err := decoder.Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if err := ensureSingleJSONValue(decoder); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if err := l.SetMode(request.Mode); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, modeResponse{Mode: l.Mode()})
}

func parseMode(mode string) (zapcore.Level, error) {
	if strings.TrimSpace(mode) == "" {
		return zap.InfoLevel, nil
	}

	var level zapcore.Level
	if err := level.UnmarshalText([]byte(strings.ToLower(strings.TrimSpace(mode)))); err != nil {
		return zap.InfoLevel, fmt.Errorf("unsupported log mode %q", mode)
	}

	switch level {
	case zap.DebugLevel, zap.InfoLevel, zap.WarnLevel, zap.ErrorLevel:
		return level, nil
	default:
		return zap.InfoLevel, fmt.Errorf(
			"unsupported log mode %q: use debug, info, warn, or error",
			mode,
		)
	}
}

func ensureSingleJSONValue(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("multiple JSON values")
		}
		return err
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
