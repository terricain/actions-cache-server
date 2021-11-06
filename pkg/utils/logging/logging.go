package logging

import (
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func SetupLogging(level string) {
	zerologLevel, err := zerolog.ParseLevel(level)
	if err != nil {
		zerologLevel = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(zerologLevel)
	// zerolog.TimeFieldFormat

	if err != nil {
		log.Warn().Err(err).Msg("Failed to parse log level, defaulting to info")
	}
}
