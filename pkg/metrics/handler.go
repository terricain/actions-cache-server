package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
)

func Server(listenAddr string) {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	log.Info().Msgf("Serving metrics on %s", listenAddr)
	if err := http.ListenAndServe(listenAddr, mux); err != nil {
		log.Error().Err(err).Msg("Caught error listening for metrics")
	} else {
		log.Info().Msg("Finished listening for metrics")
	}
}
