package media

import (
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Tomnowell/Mercury/internal/registry"
)

const (
	MTU int = 1500
)

type SessionState int

const (
	StateIdle SessionState = iota
	StateInviting
	StateRinging
	StateEstablished
)

type IncomingCall struct {
	From   registry.PhoneNumber
	Packet Packet
}

type MediaSession struct {
	pump  *Pump
	role  Role
	state SessionState

	mu                sync.Mutex
	outputLoopStarted bool
	mediaActive       atomic.Bool

	localFormats   []PayloadType
	SelectedFormat PayloadType

	onIncomingCall func(IncomingCall)
}

func NewMediaSession(pump *Pump, role Role) *MediaSession {
	session := &MediaSession{
		pump:  pump,
		state: StateIdle,
		role:  role,
		localFormats: []PayloadType{
			PCM16_8K,
		},
	}
	return session

}

func (session *MediaSession) SetRole(role Role) {
	session.mu.Lock()
	defer session.mu.Unlock()
	session.role = role
}

func (session *MediaSession) GetRole() Role {
	session.mu.Lock()
	defer session.mu.Unlock()
	return session.role
}

func (session *MediaSession) Run() {
	session.Start()

	packetBuffer := make([]byte, MTU)

	for {
		n, addr, err := session.pump.ReadPacket(packetBuffer)

		if n < 8 {
			log.Println("Malformed Packet of size: ", n)
			continue
		}

		if err != nil {
			log.Println(err)
			continue
		}

		packet, err := Decode(packetBuffer[:n], n)

		if err != nil {
			log.Println(err)
			continue
		}

		if packet.IsControl {

			log.Println("Control Packet ", packet.CtrlType)
			session.HandleControl(packet, addr)

			continue
		}

		if !session.IsEstablished() {
			// Drop packets
			log.Println("Media Packet but state not Established ", session.state)
			continue
		}

		session.handleMedia(packet)
	}
}

func (session *MediaSession) EnterEstablished() {
	log.Println("Trying to enter Established State")
	if session.state == StateEstablished {
		return
	}
	session.state = StateEstablished
	session.mediaActive.Store(true)
}

func (session *MediaSession) Start() {
	switch session.role {
	case RoleCaller:
		log.Println("Media Session started as caller")
		return

	case RoleCallee:
		session.state = StateIdle
		log.Println("Waiting For call...")
		return
	case RoleUnknown:
		log.Println("Error: RoleUnknown assigned to client")
		return
	default:
		log.Println("Error: No Role assigned to client (not even RoleUnknown)")
		return
	}
}

func (session *MediaSession) SendDecline() {
	if session.pump == nil {
		return
	}
	err := session.SendControl(CtrlDecline, session.SelectedFormat)
	if err != nil {
		log.Println("Send decline failed...")
	}
}

func (session *MediaSession) OkLoop() {
	timer := time.NewTicker(500 * time.Millisecond)
	defer timer.Stop()

	for range timer.C {
		if session.state == StateEstablished {
			return
		}

		if session.state == StateRinging {
			session.sendOK()
		}
	}
}

func (session *MediaSession) IsEstablished() bool {
	return session.state == StateEstablished
}

func (session *MediaSession) IsRinging() bool {
	return session.state == StateRinging
}

func (session *MediaSession) IsInviting() bool {
	return session.state == StateInviting
}

func (session *MediaSession) IsIdle() bool {
	return session.state == StateIdle
}

func (session *MediaSession) Reset() {
	session.mu.Lock()
	defer session.mu.Unlock()

	if session.state == StateIdle {
		return
	}

	log.Println("Resetting Session...")
	session.state = StateIdle
	session.mediaActive.Store(false)

	if session.pump != nil && session.pump.jb != nil {
		session.pump.jb.Reset()
	}
	session.pump.SetRemote(nil)
	session.role = RoleUnknown
}

func (session *MediaSession) SetIncomingCallHandler(fn func(IncomingCall)) {
	session.onIncomingCall = fn
}

func (session *MediaSession) HandleControl(packet Packet, addr *net.UDPAddr) {
	log.Println("Packet IsControl: Handling...")
	switch packet.CtrlType {
	case CtrlInvite:
		if session.pump.Remote() == nil {
			session.pump.SetRemote(addr)
		}
		session.handleInvite(packet)
	case CtrlOK:
		session.handleOK(packet)
	case CtrlAck:
		session.handleAck()
	}
}

//func (session *MediaSession) SendInvite() {
//	if session.state != StateIdle && session.state != StateInviting {
//		return
//	}
//	log.Println("Sending INVITE...")
//	err := session.SendControl(CtrlInvite, session.SelectedFormat)
//	if err != nil {
//		log.Println(err)
//	}
// }

func (session *MediaSession) handleMedia(packet Packet) {
	if packet.Restart {
		session.pump.SetFormat(packet.Payload)
		session.pump.jb.Reset()
		log.Println("Packet Restart")
		return
	}

	session.pump.jb.Push(packet.Sequence, packet.Samples)
}

func (session *MediaSession) SendControl(ctrl ControlType, payload PayloadType) error {
	packet := Packet{
		Version:   1,
		IsControl: true,
		CtrlType:  ctrl,
		Payload:   payload,
	}
	return session.pump.SendPacket(packet)
}

func (session *MediaSession) sendOK() {
	if session.state != StateRinging {
		return
	}
	log.Println("Sending OK...")
	err := session.SendControl(CtrlOK, session.SelectedFormat)
	if err != nil {
		log.Println(err)
	}
}

func (session *MediaSession) sendAck() {
	log.Println("Sending ACK...")
	err := session.SendControl(CtrlAck, session.SelectedFormat)
	if err != nil {
		log.Println(err)
	}
}

func (session *MediaSession) handleInvite(packet Packet) {
	if !session.IsIdle() {
		return
	}
	log.Println("INVITE received")
	session.state = StateRinging
	if session.onIncomingCall != nil {
		sourceNumber, err := registry.ParsePhoneNumber(packet.Source)
		if err != nil {
			log.Println(err)
			return
		}
		session.onIncomingCall(IncomingCall{
			From:   sourceNumber,
			Packet: packet,
		})
	}
}

func (session *MediaSession) handleOK(packet Packet) {
	if session.role != RoleCaller {
		return
	}
	if session.IsEstablished() {
		return
	}
	session.SelectedFormat = packet.Payload
	session.EnterEstablished()
	session.mediaActive.Store(true)
	log.Println("OK received")
	session.sendAck()
}

func (session *MediaSession) handleAck() {
	if session.role != RoleCallee {
		return
	}
	if session.IsEstablished() {
		return
	}
	log.Println("ACK received")
	session.EnterEstablished()
	session.mediaActive.Store(true)
}
