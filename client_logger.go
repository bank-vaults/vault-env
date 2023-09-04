package main

import "log/slog"

type clientLogger struct {
	logger *slog.Logger
}

func (l clientLogger) Trace(msg string, args ...map[string]interface{}) {
	l.Debug(msg, args...)
}

func (l clientLogger) Debug(msg string, args ...map[string]interface{}) {
	l.logger.Debug(msg, l.argsToAttrs(args...)...)
}

func (l clientLogger) Info(msg string, args ...map[string]interface{}) {
	l.logger.Info(msg, l.argsToAttrs(args...)...)
}

func (l clientLogger) Warn(msg string, args ...map[string]interface{}) {
	l.logger.Warn(msg, l.argsToAttrs(args...)...)
}

func (l clientLogger) Error(msg string, args ...map[string]interface{}) {
	l.logger.Error(msg, l.argsToAttrs(args...)...)
}

func (clientLogger) argsToAttrs(args ...map[string]interface{}) []any {
	var attrs []any

	for _, arg := range args {
		for key, value := range arg {
			attrs = append(attrs, slog.Any(key, value))
		}
	}

	return attrs
}
