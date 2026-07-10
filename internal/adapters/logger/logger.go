package logger

import (
	"log/slog"
	"os"
	"strings"
)

func Setup(env, serviceName string) {
	opts := &slog.HandlerOptions{
		Level: slog.LevelDebug,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			switch a.Key {
			case slog.TimeKey:
				a.Key = "timestamp"
				a.Value = slog.TimeValue(a.Value.Time().UTC())
			case slog.MessageKey:
				a.Key = "message"
			case slog.LevelKey:
				a.Value = slog.StringValue(strings.ToLower(a.Value.String()))
			}
			return a
		},
	}
	handler := slog.NewJSONHandler(os.Stdout, opts)

	logger := slog.New(handler).With(
		slog.String("service", serviceName),
		slog.String("env", env),
	)

	slog.SetDefault(logger)
}
