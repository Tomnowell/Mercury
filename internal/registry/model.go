package registry

import (
	"net"
	"time"
)

type PhoneNumber string

func (number PhoneNumber) String() string {
	return string(number)
}

type Endpoint struct {
	IP   string `json:"ip"`
	Port int    `json:"port"`
}

type Record struct {
	Number     PhoneNumber
	Endpoint   Endpoint
	ObservedIP net.IP
	LastSeen   time.Time
}
