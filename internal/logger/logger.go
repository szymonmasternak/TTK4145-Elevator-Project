package logger

import (
	"os"
	"sync"

	"github.com/rs/zerolog"
)

var once sync.Once
var Log zerolog.Logger

func configureLogger() {
	customTimeFormat := "2006-01-02T15:04:05.000Z07:00"
	zerolog.TimeFieldFormat = customTimeFormat

	output := zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: customTimeFormat,
	}

	Log = zerolog.New(output).With().Timestamp().Logger()
}

func GetLoggerConfigured(level zerolog.Level) *zerolog.Logger {
	once.Do(func() {
		configureLogger()
		zerolog.SetGlobalLevel(level)
	})
	return &Log
}

func GetLogger() *zerolog.Logger {
	once.Do(func() {
		configureLogger()
	})
	return &Log
}
