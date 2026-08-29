package handlers

import (
	"encoding/json"
	"net/http"
	"qwrttqr-rate-limiter/core/server/internal/config"
)

func HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func ParseConfig(w http.ResponseWriter, r *http.Request) {
	conf, err := config.ReadConfig()
	if err != nil {
		http.Error(w, "Failed to parse config", http.StatusInternalServerError)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(conf)
}
