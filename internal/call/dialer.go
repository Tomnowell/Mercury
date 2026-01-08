package call

import (
	"github.com/Tomnowell/internal/call"
	"github.com/Tomnowell/internal/media"
	"github.com/Tomnowell/internal/audio

)

type Dialler interface {
	Dial(deviceID string) (*Controller, error) 
	Register(id string, address *net.UDPAddr) error
	Lookup(id string) (*net.UDPAddr, error)
}


type UDPRegistryDialer {
	registryAddr string
}


func (dialer *UDPRegistryDialer) Dial(deviceID string) (*Controller, error) {
	target, err := queryRegistry(dialer.registryAddr, deviceID)
	if err != nil {
		reutrn nil, err 
	}

	controller := call.NewController(Role

 

}

