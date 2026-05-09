package api

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/adamlacasse/freq-show/apps/server/pkg/data"
	"github.com/adamlacasse/freq-show/apps/server/pkg/discovery"
)

const maxDiscoveryQueryRunes = 1000

func discoverHandler(svc *discovery.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !assertMethod(w, r, http.MethodPost) {
			return
		}
		if svc == nil {
			writeJSON(w, http.StatusServiceUnavailable, errorResponse{"discovery service is not configured"})
			return
		}

		var query data.DiscoveryQuery
		if err := json.NewDecoder(r.Body).Decode(&query); err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{"invalid request body"})
			return
		}
		query.Query = strings.TrimSpace(query.Query)
		if query.Query == "" {
			writeJSON(w, http.StatusBadRequest, errorResponse{"query field required"})
			return
		}
		if utf8.RuneCountInString(query.Query) > maxDiscoveryQueryRunes {
			writeJSON(w, http.StatusBadRequest, errorResponse{"query too long (max 1000 characters)"})
			return
		}

		result, err := svc.Run(r.Context(), query)
		if err != nil {
			log.Printf("discovery request failed: %v", err)
			switch {
			case errors.Is(err, discovery.ErrUnconfigured):
				writeJSON(w, http.StatusServiceUnavailable, errorResponse{"discovery service is not configured"})
			case errors.Is(err, discovery.ErrNoEmbeddedAlbums):
				writeJSON(w, http.StatusServiceUnavailable, errorResponse{"discovery index is empty; browse a few albums or run the reindex job first"})
			default:
				writeJSON(w, http.StatusBadGateway, errorResponse{discoveryUserError(err)})
			}
			return
		}
		writeJSON(w, http.StatusOK, result)
	}
}

func discoveryUserError(err error) string {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "embed interpreted query"):
		return "discovery query embedding failed"
	case strings.Contains(msg, "interpret"):
		return "discovery interpretation failed"
	case strings.Contains(msg, "curate"):
		return "discovery curation failed"
	default:
		return "discovery request failed"
	}
}
