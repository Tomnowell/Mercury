package call

import (
	"log"
	"net"

	"github.com/Tomnowell/Mercury/internal/audio"
	"github.com/Tomnowell/Mercury/internal/media"
)

type Controller struct {
	role   media.Role
	format audio.Format

	input   audio.Input
	output  audio.Output
	pump    *media.Pump
	session *media.MediaSession
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
