package call

import (
	"errors"
	"log"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/Tomnowell/Mercury/internal/audio"
	"github.com/Tomnowell/Mercury/internal/media"
	"github.com/Tomnowell/Mercury/internal/registry"
)

type Controller struct {
	number registry.PhoneNumber
	format audio.Format

	registry registry.Client

	localAddr *net.UDPAddr

	input  audio.Input
	output audio.Output

	pump    *media.Pump
	session *media.MediaSession

	conn net.Conn

	mu sync.Mutex

	pendingCall *media.IncomingCall
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
	session := media.NewMediaSession(pump, media.RoleUnknown)

	controller := &Controller{
		number:    number,
		format:    format,
		localAddr: localAddr,
		input:     input,
		output:    output,
		pump:      pump,
		conn:      conn,
		session:   session,
		registry:  reg,
	}
	pump.SetMediaGate(func() bool {
		controller.mu.Lock()
		defer controller.mu.Unlock()

		return controller.session != nil && controller.session.IsEstablished()
	})

	err = controller.register()
	if err != nil {
		return nil, err
	}

	go controller.pump.SendLoop()
	go controller.pump.OutputLoop()

	go controller.session.Run()
	go controller.listen()

	return controller, nil
}

func (controller *Controller) register() error {
	return controller.registry.RegisterSelf(controller.number, controller.localAddr.Port)
}

func (controller *Controller) listen() {
	buffer := make([]byte, media.MTU)
	for {
		n, addr, err := controller.pump.ReadPacket(buffer)
		if err != nil || n < 8 {
			continue
		}
		packet, err := media.Decode(buffer[:n], n)
		if err != nil {
			continue
		}
		if packet.IsControl {
			controller.handleControl(packet, addr)
		}
	}
}

func (controller *Controller) handleControl(packet media.Packet, addr *net.UDPAddr) {
	controller.mu.Lock()
	defer controller.mu.Unlock()

	if controller.session == nil || controller.pump == nil {
		log.Println("Dropping control packet: Controller not ready...")
		return
	}

	switch packet.CtrlType {
	case media.CtrlInvite:
		if controller.pendingCall != nil {
			log.Println("Busy: Declining Call")
			controller.sendDecline()
			return
		}

		from, err := registry.ParsePhoneNumber(packet.Source)
		if err != nil {
			log.Println("Invalid source number", packet.Source, len(packet.Source))
			return
		}

		if controller.pump.Remote() == nil {
			controller.pump.SetRemote(addr)
		}

		controller.pendingCall = &media.IncomingCall{
			From:   from,
			Packet: packet,
		}

		log.Println("Incoming call from: ", from.String())
		log.Println("'a' to accept, 'd' to decline")

	case media.CtrlOK, media.CtrlAck, media.CtrlDecline:
		if controller.session != nil {
			controller.session.HandleControl(packet, addr)
		}
	}

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
	controller.mu.Lock()
	defer controller.mu.Unlock()

	controller.session.Reset()
	controller.session.SetRole(media.RoleCallee)
	log.Println("Ready to accept incoming calls")
}

func (controller *Controller) Dial(number registry.PhoneNumber) error {
	controller.mu.Lock()
	defer controller.mu.Unlock()

	if controller.pendingCall != nil {
		return errors.New("Already on a call: Busy")
	}

	log.Println("Looking up number", number.String())
	record, err := controller.registry.Lookup(number)

	log.Println("Found record: ", record)

	remoteAddr, err := net.ResolveUDPAddr(
		"udp6",
		net.JoinHostPort(
			record.IP,
			strconv.Itoa(record.Port)),
	)

	if err != nil {
		log.Println("Error resolving remote address")
		return err
	}

	controller.pump.SetRemote(remoteAddr)

	controller.session.SetRole(media.RoleCaller)
	log.Println("Calling: ", number.String())
	go controller.inviteLoop(number)

	return nil
}

func (controller *Controller) onIncomingCall(call media.IncomingCall) {
	controller.mu.Lock()
	defer controller.mu.Unlock()

	if controller.pendingCall != nil {
		log.Println("Busy: Already Ringing")
		return
	}

	controller.pendingCall = &call

	log.Printf("Incoming call From: %s", call.From)
	log.Println("Type 'a' to accept or 'd' to decline")
}

func (controller *Controller) AcceptCall() error {
	controller.mu.Lock()
	defer controller.mu.Unlock()

	if controller.pendingCall == nil {
		return errors.New("no incoming call")
	}

	controller.session.SetRole(media.RoleCallee)
	controller.session.OkLoop()
	controller.pendingCall = nil

	log.Println("Call accepted")
	return nil
}

func (controller *Controller) DeclineCall() error {
	controller.mu.Lock()
	defer controller.mu.Unlock()

	if controller.pendingCall == nil {
		return errors.New("no incoming call")
	}

	err := controller.sendDecline()
	if err != nil {
		return errors.New("Could not send decline")
	}

	controller.pendingCall = nil

	log.Println("Call declined")

	return nil
}

func (controller *Controller) HangUp() {
	controller.mu.Lock()
	defer controller.mu.Unlock()

	if controller.session == nil {
		return
	}

	controller.session = nil
	controller.pump.SetRemote(nil)

	log.Println("Call ended")

}

func (controller *Controller) sendDecline() error {
	packet := media.Packet{
		Version:   1,
		IsControl: true,
		CtrlType:  media.CtrlDecline,
	}

	return controller.pump.SendPacket(packet)
}

func (controller *Controller) sendInvite(target registry.PhoneNumber) error {
	packet := media.Packet{
		Version:   1,
		IsControl: true,
		CtrlType:  media.CtrlInvite,
		Payload:   controller.session.SelectedFormat,
		Source:    controller.number.String(),
		Target:    target.String(),
	}
	return controller.pump.SendPacket(packet)
}

func (controller *Controller) inviteLoop(target registry.PhoneNumber) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if controller.session.IsEstablished() {
				return
			}
			if controller.session.IsRinging() {
				return
			}

			err := controller.sendInvite(target)
			if err != nil {
				log.Println(err)

			}

		}

	}

}
