package media

import (
	"log"
	"net"
	"sync"
	"sync/atomic"
)

const (
	MTU int = 1500
)

type MediaSession struct {
	pump *Pump

	mu                sync.Mutex
	outputLoopStarted bool
	mediaActive       atomic.Bool

	localFormats   []PayloadType
	SelectedFormat PayloadType

	controlHandler func(Packet, *net.UDPAddr)
}

func NewMediaSession(pump *Pump) *MediaSession {
	session := &MediaSession{
		pump: pump,
		localFormats: []PayloadType{
			PCM16_8K,
		},
	}
	return session

}

func (session *MediaSession) SetControlHandler(fn func(Packet, *net.UDPAddr)) {
	session.controlHandler = fn
}

func (session *MediaSession) Run() {

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
		log.Printf("RX seq=%d ts=%d", packet.Sequence, packet.Timestamp)

		if err != nil {
			log.Println(err)
			continue
		}

		if packet.IsControl {
			log.Println("Control Packet ", packet.CtrlType)
			if session.controlHandler != nil {
				session.controlHandler(packet, addr)
			}
			continue
		}

		session.handleMedia(packet)
	}
}

func (session *MediaSession) StartMedia() {
	session.pump.allowMedia = func() bool {
		return true
	}
	session.pump.jb.Reset()
}

func (session *MediaSession) StopMedia() {
	session.pump.allowMedia = func() bool {
		return false
	}
	session.pump.jb.Reset()
}

func (session *MediaSession) Reset() {
	session.mu.Lock()
	defer session.mu.Unlock()

	log.Println("Resetting Session...")
	session.mediaActive.Store(false)

	if session.pump != nil && session.pump.jb != nil {
		session.pump.jb.Reset()
	}
	session.pump.SetRemote(nil)
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
