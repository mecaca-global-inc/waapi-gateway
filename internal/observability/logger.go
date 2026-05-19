package observability

import (
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func Setup(level, format string) {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	lvl, err := zerolog.ParseLevel(level)
	if err != nil {
		lvl = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(lvl)
	if format == "json" {
		log.Logger = log.Output(os.Stderr)
		return
	}
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
}
