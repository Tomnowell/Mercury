package media

import (
	"log"
	"sync"
	"sync/atomic"
	"time"
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

type MediaSession struct {
	pump  *Pump
	role  Role
	state SessionState

	mu                sync.Mutex
	outputLoopStarted bool
	mediaActive       atomic.Bool

	localFormats   []PayloadType
	selectedFormat PayloadType
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

	pump.SetPacketHandler(session.handlePacket)
	return session

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

		packet, err := Decode(packetBuffer[:n])

		if err != nil {
			log.Println(err)
			continue
		}

		if packet.IsControl {
			if session.pump.Remote() == nil {
				log.Println("Binding remote address to: ", addr)
				session.pump.SetRemote(addr)
			}

			log.Println("Control Packet ", packet.CtrlType)
			session.handleControl(packet)

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

func (session *MediaSession) InviteLoop() {
	timer := time.NewTicker(500 * time.Millisecond)
	defer timer.Stop()

	for range timer.C {
		if session.state != StateIdle && session.state != StateInviting {
			return
		}
		session.sendInvite()
	}

}

func (session *MediaSession) Start() {
	switch session.role {
	case RoleCaller:
		session.state = StateInviting
		go session.InviteLoop()
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

func (session *MediaSession) sendControl(ctrl ControlType, payload PayloadType) error {
	packet := Packet{
		Version:   1,
		IsControl: true,
		CtrlType:  ctrl,
		Payload:   payload,
	}

	return session.pump.sendPacket(packet)
}

func (session *MediaSession) sendInvite() {
	if session.state != StateIdle && session.state != StateInviting {
		return
	}
	log.Println("Sending INVITE...")

	err := session.sendControl(CtrlInvite, session.selectedFormat)
	if err != nil {
		log.Println(err)
	}

}

func (session *MediaSession) sendOK() {
	if session.state != StateRinging {
		return
	}

	log.Println("Sending OK...")

	err := session.sendControl(CtrlOK, session.selectedFormat)
	if err != nil {
		log.Println(err)
	}
}

func (session *MediaSession) sendAck() {
	log.Println("Sending ACK...")

	err := session.sendControl(CtrlAck, session.selectedFormat)

	if err != nil {
		log.Println(err)
	}
}

func (session *MediaSession) okLoop() {
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

func (session *MediaSession) HandleInvite(packet Packet) {

	if !session.IsIdle() {
		return
	}
	log.Println("INVITE received")
	session.selectedFormat = packet.Payload
	session.state = StateRinging

	if session.role == RoleCallee {
		go session.okLoop()
	}

}

func (session *MediaSession) HandleOK(packet Packet) {
	if session.role != RoleCaller {
		return
	}
	if session.IsEstablished() {
		return
	}
	session.selectedFormat = packet.Payload
	session.EnterEstablished()
	session.mediaActive.Store(true)

	log.Println("OK received")

	session.sendAck()

}

func (session *MediaSession) HandleAck(packet Packet) {
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

func (session *MediaSession) handlePacket(packet Packet) {
	log.Println("Handling packet:")
	if packet.IsControl {
		log.Print("Session State: ", session.state)
		log.Println("Packet IsControl, sending to handleControl")
		session.handleControl(packet)
		return
	}

	if session.state != StateEstablished {
		log.Println("Session State: ", session.state)
		log.Println("Packet IsMedia, but State is not Established yet...returning")
		return
	}

	session.handleMedia(packet)
}

func (session *MediaSession) handleMedia(packet Packet) {
	if packet.Restart {
		session.pump.SetFormat(packet.Payload)
		session.pump.jb.Reset()
		log.Println("Packet Restart")
		return
	}

	session.pump.jb.Push(packet.Sequence, packet.Samples)
}

func (session *MediaSession) handleControl(packet Packet) {
	log.Println("Packet IsControl: Handling...")
	switch packet.CtrlType {
	case CtrlInvite:
		session.HandleInvite(packet)
	case CtrlOK:
		session.HandleOK(packet)
	case CtrlAck:
		session.HandleAck(packet)
	}
}
