package call

import (
	"log"
	"net"
	"strconv"
	"sync"

	"github.com/Tomnowell/Mercury/internal/audio"
	"github.com/Tomnowell/Mercury/internal/media"
	"github.com/Tomnowell/Mercury/internal/registry"
)

type Controller struct {
	number registry.PhoneNumber
	role   media.Role
	format audio.Format

	registry registry.Client

	localAddr *net.UDPAddr

	input  audio.Input
	output audio.Output

	pump    *media.Pump
	session *media.MediaSession

	conn net.Conn

	mu sync.Mutex
}

func NewController(
	number registry.PhoneNumber,
	role media.Role,
	format audio.Format,
	localAddr *net.UDPAddr,
	registryURL string,
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

	reg := *registry.NewClient(registryURL)

	conn, err := net.ListenUDP("udp6", localAddr)
	if err != nil {
		log.Println("Error: Could not create UDP listener")
	}

	pump := media.NewPump(conn, nil, input, output, format)

	session := media.NewMediaSession(pump, role)

	pump.SetMediaGate(func() bool {
		return session.IsEstablished()
	})

	controller := &Controller{
		number:    number,
		role:      role,
		format:    format,
		localAddr: localAddr,
		input:     input,
		output:    output,
		pump:      pump,
		session:   session,
		conn:      conn,
		registry:  reg,
	}

	err = controller.register()
	if err != nil {
		return nil, err
	}

	go controller.pump.SendLoop()
	go controller.pump.OutputLoop()

	return controller, nil
}

func (controller *Controller) register() error {
	return controller.registry.RegisterSelf(controller.number, controller.localAddr.Port)
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

func (controller *Controller) SetRole(role media.Role) {
	controller.role = role
}

func (controller *Controller) Answer() {
	if controller.role != media.RoleCallee {
		log.Println("Answer() called on non-callee controller")
		return
	}

	log.Println("Ready to accept incoming calls")

	go controller.session.Run()

}

func (controller *Controller) Dial(number registry.PhoneNumber) error {
	controller.mu.Lock()
	defer controller.mu.Unlock()

	record, err := controller.registry.Lookup(number)

	remoteAddr, err := net.ResolveUDPAddr(
		"udp6",
		net.JoinHostPort(
			record.IP,
			strconv.Itoa(record.Port)),
	)

	if err != nil {
		return err
	}

	controller.pump.SetRemote(remoteAddr)

	go controller.session.Run()

	return nil

}
