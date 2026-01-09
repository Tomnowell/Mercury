package call

import (
	"log"
	"net"
	"strconv"

	"github.com/Tomnowell/Mercury/internal/audio"
	"github.com/Tomnowell/Mercury/internal/media"
	"github.com/Tomnowell/Mercury/internal/registry"
	"github.com/Tomnowell/Mercury/internal/signalling"
)

type Controller struct {
	role   media.Role
	format audio.Format

	input   audio.Input
	output  audio.Output
	pump    *media.Pump
	session *media.MediaSession

	dialTarget string
}

func NewController(
	role media.Role,
	format audio.Format,
	localAddr, remoteAddr *net.UDPAddr,
) (*Controller, error) {
	input, err := audio.NewInput(format)

	if err != nil {
		log.Println("Error: Could not create audio input")
		return nil, err
	}

	output, err := audio.NewOutput(format)

	if err != nil {
		log.Println("Error: Could not create audio output")
		return nil, err
	}

	conn, err := net.ListenUDP("udp6", localAddr)
	defer conn.Close()

	if err != nil {
		log.Println("Error: Could not start UDP listener on: ", localAddr)
		return nil, err
	}

	pump := media.NewPump(conn, remoteAddr, input, output, format)

	go pump.SendLoop()

	go pump.OutputLoop()

	session := media.NewMediaSession(pump, role)

	pump.SetMediaGate(func() bool {
		return session.IsEstablished()
	})

	return &Controller{
		role:    role,
		format:  format,
		input:   input,
		output:  output,
		pump:    pump,
		session: session,
	}, nil
}

func (controller *Controller) Run() {
	go controller.session.Run()
}

func (controller *Controller) Close() {
	if controller.input != nil {
		controller.input.Close()
	}

	if controller.output != nil {
		controller.output.Close()
	}
}

func (controller *Controller) Answer() {
	if controller.role != media.RoleCallee {
		log.Println("Answer() called on non-callee controller")
		return
	}

	log.Println("Ready to accept incoming calls")
}
func Dial(
	number registry.PhoneNumber,
	format audio.Format,
	registry signalling.RegistryClient,
	localAddr *net.UDPAddr,
) (*Controller, error) {

	record, err := registry.Lookup(number)

	if err != nil {
		return nil, err
	}

	remoteAddr, err := net.ResolveUDPAddr("udp6", net.JoinHostPort(record.Endpoint.IP, strconv.Itoa(record.Endpoint.Port)))

	if err != nil {
		return nil, err
	}

	controller, err := NewController(
		media.RoleCaller,
		format,
		localAddr,
		remoteAddr,
	)

	if err != nil {
		return nil, err
	}

	controller.Run()

	return controller, nil

}
