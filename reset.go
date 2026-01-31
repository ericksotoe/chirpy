package main

import (
	"context"
	"net/http"
)

func (cfg *apiConfig) requestResetHandler(w http.ResponseWriter, r *http.Request) {
	cfg.fileserverHits.Store(0)
	if cfg.dev == "dev" {
		err := cfg.db.ResetUsers(context.Background())
		if err != nil {
			respondWithError(w, http.StatusForbidden, "Can't reset the db with your permissions")
			return
		}
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Hits reset to 0 and db reset"))

}
