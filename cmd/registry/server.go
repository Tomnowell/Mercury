package main

import (
	"log"
	"net/http"

	"github.com/Tomnowell/Mercury/internal/registry"
)

func NewServer(addr string) error {
	store := registry.NewStore()
	handlers := NewRegistryHandlers(store)

	mux := http.NewServeMux()
	mux.HandleFunc("/register/", handlers.Register)
	mux.HandleFunc("/lookup/", handlers.LookUp)
	mux.HandleFunc("/health/", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Debug Only
	mux.HandleFunc("/records/", handlers.ListRecords)

	log.Println("Registry listening on: ", addr)
	return http.ListenAndServe(addr, mux)

}
