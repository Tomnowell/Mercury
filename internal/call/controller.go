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

type CallState int

const (
	StateIdle CallState = iota
	StateCalling
	StateRinging
	StateEstablished
)

type IncomingCall struct {
	From   registry.PhoneNumber
	Packet media.Packet
}

type Controller struct {
	number registry.PhoneNumber
	format audio.Format

	registry registry.Client

	localAddr *net.UDPAddr

	input  audio.Input
	output audio.Output

	pump    *media.Pump
	session *media.MediaSession

	state CallState

	conn net.Conn

	mu sync.Mutex

	pendingCall *IncomingCall
}

func NewController(
	number registry.PhoneNumber,
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
		return nil, err
	}

	pump := media.NewPump(conn, nil, input, output, format)
	session := media.NewMediaSession(pump)

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
	controller.session.SetControlHandler(controller.handleControl)

	pump.SetMediaGate(func() bool {
		controller.mu.Lock()
		defer controller.mu.Unlock()

		return controller.session != nil && controller.state == StateEstablished
	})

	err = controller.register()
	if err != nil {
		return nil, err
	}

	go controller.pump.AudioIngestLoop()
	go controller.pump.SendLoop()
	go controller.pump.OutputLoop()
	go controller.session.Run()

	return controller, nil
}

func (controller *Controller) register() error {
	return controller.registry.RegisterSelf(controller.number, controller.localAddr.Port)
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
		if controller.state != StateIdle {
			controller.sendDecline()
		}

		controller.state = StateRinging

		from, err := registry.ParsePhoneNumber(packet.Source)
		if err != nil {
			log.Println("Invalid source number", packet.Source, len(packet.Source))
			return
		}
		to, err := registry.ParsePhoneNumber(packet.Target)
		if to.String() != controller.number.String() {

			log.Println("Wrong number!", to, controller.number)
			return
		}

		if controller.pump.Remote() == nil {
			controller.pump.SetRemote(addr)
		}

		controller.pendingCall = &IncomingCall{
			From:   from,
			Packet: packet,
		}
		log.Println("Incoming call from: ", from.String())
		log.Println("'a' to accept, 'd' to decline")

	case media.CtrlOK:

		if controller.state != StateCalling {
			return
		}

		controller.state = StateEstablished

		log.Println("Ok Received...Establishing State")
		err := controller.sendAck()
		if err != nil {
			log.Println("Error: Could not send ACK...", err)
			return
		}
		controller.session.StartMedia()

	case media.CtrlAck:
		log.Println("ACK Received")
		if controller.state != StateRinging {
			log.Println("...But state wasnt StateRinging")
			return
		}

		controller.state = StateEstablished
		controller.session.StartMedia()

	case media.CtrlDecline:
		// TODO: Consider what needs to happen when a call is declined...session reset is not quite right.
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

func (controller *Controller) Reset() {
	controller.session.Reset()

}

func (controller *Controller) Dial(number registry.PhoneNumber) error {
	controller.mu.Lock()
	defer controller.mu.Unlock()

	controller.state = StateCalling
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

	log.Println("Setting remote: ", remoteAddr)
	controller.pump.SetRemote(remoteAddr)

	log.Println("Calling: ", number.String())
	go controller.inviteLoop(number)

	return nil
}

func (controller *Controller) AcceptCall() error {
	controller.mu.Lock()
	defer controller.mu.Unlock()

	if controller.pendingCall == nil {
		return errors.New("no incoming call")
	}

	go controller.okLoop()
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
	controller.state = StateIdle

	log.Println("Call declined")

	return nil
}

func (controller *Controller) HangUp() {
	controller.mu.Lock()
	defer controller.mu.Unlock()

	if controller.session == nil {
		return
	}

	controller.session.Reset()

	controller.pump.SetRemote(nil)
	controller.state = StateIdle

	log.Println("Call ended")

}

func (controller *Controller) sendDecline() error {
	return controller.sendControl(media.CtrlDecline, controller.session.SelectedFormat)
}

func (controller *Controller) sendAck() error {
	log.Println("Sending ACK")
	return controller.sendControl(media.CtrlAck, controller.session.SelectedFormat)
}

func (controller *Controller) sendOK() error {
	return controller.sendControl(media.CtrlOK, controller.session.SelectedFormat)
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

	log.Println("Remote: ", controller.pump.Remote())

	if controller.pump.Remote() == nil {
		log.Println("Remote is nil??? ", controller.pump.Remote())
		return errors.New("Pump remote not set: Cannot send invite")
	}

	return controller.pump.SendPacket(packet)
}

func (controller *Controller) sendControl(ctrl media.ControlType, payload media.PayloadType) error {
	log.Println("sendControl called with type: ", ctrl)
	packet := media.Packet{
		Version:   1,
		IsControl: true,
		CtrlType:  ctrl,
		Payload:   payload,
	}

	if controller.pump.Remote() == nil {
		return errors.New("Pump remote not set: Cannot send control packet")
	}

	return controller.pump.SendPacket(packet)
}

func (controller *Controller) inviteLoop(target registry.PhoneNumber) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if controller.state != StateCalling {
				return
			}

			err := controller.sendInvite(target)
			if err != nil {
				log.Println(err)
			}
		}
	}
}

func (controller *Controller) okLoop() {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if controller.state == StateEstablished {
				return
			}

			err := controller.sendOK()
			if err != nil {
				log.Println("Error sending OK", err)
			}
			log.Println("Sending OK...")
		}

	}
}
