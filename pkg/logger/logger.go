package logger

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type defaultHandler struct {
	slog.Handler
}

func New(level slog.Level) *slog.Logger {
	handlerOptions := slog.HandlerOptions{
		AddSource:   true,
		Level:       level,
		ReplaceAttr: replaceAttr,
	}

	var defaultAttrs []slog.Attr
	//defaultAttrs = append(defaultAttrs, slog.String("app", appName))

	handler := setDefaultHandler(handlerOptions, defaultAttrs)
	slog.SetDefault(setDefaultHandler(handlerOptions, defaultAttrs))
	return handler
}

func setDefaultHandler(handlerOptions slog.HandlerOptions, attrs []slog.Attr) *slog.Logger {
	return slog.New(&defaultHandler{
		Handler: slog.NewJSONHandler(os.Stdout, &handlerOptions).WithAttrs(attrs),
	})
}

func replaceAttr(groups []string, a slog.Attr) slog.Attr {
	if a.Key == slog.TimeKey {
		v := a.Value.Time().Format(time.RFC3339)
		return slog.Attr{Key: "ts", Value: slog.StringValue(v)}
	}

	if a.Key == slog.LevelKey {
		v := strings.ToLower(a.Value.String())
		return slog.Attr{Key: "level", Value: slog.StringValue(v)}
	}

	if a.Key == slog.SourceKey {
		if source, ok := a.Value.Any().(*slog.Source); ok {
			v := fmt.Sprintf("%s:%d", filepath.Base(source.File), source.Line)
			return slog.Attr{Key: "file", Value: slog.StringValue(v)}
		}
	}

	return a
}
