package call

import (
	"net"
)

type Dialler interface {
	Dial(deviceID string) (*Controller, error)
	Register(id string, address *net.UDPAddr) error
	Lookup(id string) (*net.UDPAddr, error)
}

type UDPRegistryDialer struct {
	registryAddr string
}
