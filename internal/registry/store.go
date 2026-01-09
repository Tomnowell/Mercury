package registry

import (
	"errors"
	"net"
	"sync"
	"time"
)

var (
	ErrNoAlreadyRegistered = errors.New("number already registered")
	ErrNumberNotFound      = errors.New("number not found")
)

type Store struct {
	mu      sync.RWMutex
	records map[PhoneNumber]*Record
}

func NewStore() *Store {
	return &Store{
		records: make(map[PhoneNumber]*Record),
	}
}

func (store *Store) Register(record *Record) error {
	store.mu.Lock()
	defer store.mu.Unlock()

	exists := store.records[record.Number]

	if exists != nil {
		return ErrNoAlreadyRegistered
	}
	store.records[record.Number] = record
	return nil
}

func (store *Store) LookUp(number PhoneNumber) (*Record, error) {
	store.mu.RLock()
	defer store.mu.RUnlock()

	record, exists := store.records[number]

	if !exists {
		return nil, ErrNumberNotFound
	}

	recordCopy := *record
	return &recordCopy, nil

}

func (store *Store) Touch(number PhoneNumber, observedIP net.IP) error {

	store.mu.Lock()
	defer store.mu.Unlock()

	record, exists := store.records[number]

	if !exists {
		return ErrNumberNotFound
	}

	record.LastSeen = time.Now()
	record.ObservedIP = observedIP
	return nil
}

func (store *Store) DeRegister(number PhoneNumber) error {
	store.mu.Lock()
	defer store.mu.Unlock()

	_, exists := store.records[number]

	if !exists {
		return ErrNumberNotFound
	}

	delete(store.records, number)
	return nil
}

func (store *Store) All() []*Record {
	store.mu.RLock()
	defer store.mu.RUnlock()

	allRecords := make([]*Record, len(store.records))

	for _, record := range store.records {
		recordCopy := *record
		allRecords = append(allRecords, &recordCopy)
	}
	return allRecords
}
