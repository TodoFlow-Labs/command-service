package logging

import (
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type Logger = zerolog.Logger

func New(level string) *zerolog.Logger {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	logger := log.With().Timestamp().Logger()
	return &logger
}

