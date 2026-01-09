package main

import (
	"encoding/json"
	"github.com/Tomnowell/Mercury/internal/registry"
	"net"
	"net/http"
	"strings"
	"time"
)

type RegistryHandlers struct {
	store *registry.Store
}

func NewRegistryHandlers(store *registry.Store) *RegistryHandlers {
	return &RegistryHandlers{store: store}
}

func (handler *RegistryHandlers) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request RegisterRequest

	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	number, err := registry.ParsePhoneNumber(request.Number)
	if err != nil {
		http.Error(w, "Invalid number", http.StatusBadRequest)
		return
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		http.Error(w, "Invalid remote address", http.StatusBadRequest)
		return
	}

	observedIP := net.ParseIP(host)

	if observedIP == nil {
		http.Error(w, "Invalid remote IP: Could not parse address", http.StatusBadRequest)
		return
	}
	record := &registry.Record{
		Number: number,
		Endpoint: registry.Endpoint{
			IP:   observedIP.String(),
			Port: request.Port,
		},
		ObservedIP: observedIP,
		LastSeen:   time.Now(),
	}

	handler.store.Register(record)
}

func (handler *RegistryHandlers) LookUp(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")

	if len(parts) != 2 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	number, err := registry.ParsePhoneNumber(parts[1])
	if err != nil {
		http.Error(w, "Invalid number", http.StatusBadRequest)
		return
	}

	record, err := handler.store.LookUp(number)

	if err != nil {
		http.Error(w, "Number not found", http.StatusNotFound)
		return
	}

	response := LookupResponse{
		Number: record.Number.String(),
		IP:     record.Endpoint.IP,
		Port:   record.Endpoint.Port,
	}

	_ = json.NewEncoder(w).Encode(response)

}

func (handler *RegistryHandlers) ListRecords(w http.ResponseWriter, r *http.Request) {
	records := handler.store.All()
	_ = json.NewEncoder(w).Encode(records)
}
