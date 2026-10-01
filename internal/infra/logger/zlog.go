package logger

import (
	"os"

	"github.com/rs/zerolog"
)

type Logger struct {
	logger zerolog.Logger
}

// New returns a JSON logger writing to stdout. An unknown level means info.
func New(level string) *Logger {
	lvl, err := zerolog.ParseLevel(level)
	if err != nil || lvl == zerolog.NoLevel {
		lvl = zerolog.InfoLevel
	}

	return &Logger{
		logger: zerolog.New(os.Stdout).Level(lvl).With().Timestamp().Logger(),
	}
}

func (l *Logger) Info(msg string, keysAndValues ...interface{}) {
	l.logger.Info().Fields(keyValuesToMap(keysAndValues...)).Msg(msg)
}

func (l *Logger) Error(msg string, keysAndValues ...interface{}) {
	l.logger.Error().Fields(keyValuesToMap(keysAndValues...)).Msg(msg)
}

func (l *Logger) Warn(msg string, keysAndValues ...interface{}) {
	l.logger.Warn().Fields(keyValuesToMap(keysAndValues...)).Msg(msg)
}

func (l *Logger) Debug(msg string, keysAndValues ...interface{}) {
	l.logger.Debug().Fields(keyValuesToMap(keysAndValues...)).Msg(msg)
}

func keyValuesToMap(keysAndValues ...interface{}) map[string]interface{} {
	if len(keysAndValues) == 0 {
		return nil
	}

	result := make(map[string]interface{})

	for i := 0; i < len(keysAndValues); i += 2 {
		if i+1 >= len(keysAndValues) {
			if key, ok := keysAndValues[i].(string); ok {
				result[key] = "[MISSING_VALUE]"
			}
			break
		}

		if key, ok := keysAndValues[i].(string); ok {
			result[key] = keysAndValues[i+1]
		}
	}

	return result
}
