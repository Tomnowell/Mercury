package media

import (
	"log"
	"net"
	"sync"
	"time"

	"github.com/Tomnowell/Mercury/internal/audio"
)

const (
	maxPacketSize = 1500
)

type Pump struct {
	conn         *net.UDPConn
	remote       *net.UDPAddr
	input        audio.Input
	output       audio.Output
	format       audio.Format
	onPacket     func(Packet)
	sequence     uint16
	timestamp    uint32
	packetBuffer []byte
	jb           *JitterBuffer
	underflows   int
	fallback     bool
	mu           sync.Mutex
	allowMedia   func() bool
	outputDone   chan struct{}
}

func NewPump(
	conn *net.UDPConn,
	remote *net.UDPAddr,
	input audio.Input,
	output audio.Output,
	format audio.Format,

) *Pump {
	return &Pump{
		conn:         conn,
		remote:       remote,
		input:        input,
		output:       output,
		format:       format,
		packetBuffer: make([]byte, maxPacketSize),
		jb: NewJitterBuffer(
			0,
			3,
			SamplesPerFrame(format.SampleRate),
		),
		allowMedia: func() bool { return false },
	}
}

func (pump *Pump) SendLoop() {
	samplesPerFrame := SamplesPerFrame(pump.format.SampleRate)
	buffer := make([]int16, samplesPerFrame)
	payload := PCM16_8K

	for {
		pump.mu.Lock()
		in := pump.input
		pump.mu.Unlock()

		n, err := in.Read(buffer)

		if err != nil {
			log.Println("SendLoop input error:", err)
			time.Sleep(10 * time.Millisecond)
			continue
		}

		if n == 0 {
			// Buffer is empty
			log.Println("SendLoop: Buffer empty")
			continue
		}

		if pump.allowMedia != nil && pump.allowMedia() {
			packet := Packet{
				Version:   1,
				Payload:   payload,
				Sequence:  pump.sequence,
				Timestamp: pump.timestamp,
				Samples:   buffer[:n],
			}

			if err := pump.sendPacket(packet); err != nil {
				log.Println("Error: Could not send packet")
				continue
			}
			pump.sequence++
			pump.timestamp += uint32(n)
		}
	}
}

func (pump *Pump) OutputLoop() {
	buffer := make([]int16, SamplesPerFrame(pump.format.SampleRate))

	for {
		select {
		case <-pump.outputDone:
			return
		default:
			var samples []int16
			pump.mu.Lock()
			if pump.allowMedia != nil && pump.allowMedia() {
				samples = pump.jb.Pop()
			}
			pump.mu.Unlock()

			if samples != nil {
				log.Printf("Media Out: samples=%d", samples)
			}

			if samples != nil && len(samples) == len(buffer) {
				copy(buffer, samples)
				pump.underflows = 0
			} else {
				clear(buffer)
				pump.underflows++
			}
			if _, err := pump.output.Write(buffer); err != nil {
				log.Println("Error: Output Write Error")
			}
			if pump.underflows > 20 && !pump.fallback && pump.allowMedia() {
				log.Println("Alert: Audio underflows > 20")
			}
		}
	}
}

func (pump *Pump) RestartOutput() error {
	log.Println("(Re)Starting Audio Output...")
	if pump.outputDone != nil {
		close(pump.outputDone)
	}

	if pump.output != nil {
		_ = pump.output.Close()
	}
	out, err := audio.NewOutput(pump.format)
	if err != nil {
		log.Println("Error: Could not make new output!")
		return err
	}
	pump.output = out
	pump.outputDone = make(chan struct{})

	go pump.OutputLoop()
	return nil
}

func (pump *Pump) RestartMedia() error {
	pump.sequence = 0
	pump.timestamp = 0

	packet := Packet{
		Version: 1,
		Payload: PayloadType(pump.format.SampleRate),
		Restart: true,
	}
	return pump.sendPacket(packet)
}

func (pump *Pump) RequestFallback() error {
	pump.format.SampleRate = audio.SampleRate8K

	return pump.RestartMedia()
}

func (pump *Pump) SetFormat(payload PayloadType) error {

	format := FormatFromPayload(payload)

	pump.mu.Lock()
	defer pump.mu.Unlock()

	pump.format = format
	pump.sequence = 0
	pump.timestamp = 0
	pump.jb.Reset()

	if err := pump.input.Reconfigure(format); err != nil {
		return err
	}

	if err := pump.output.Reconfigure(format); err != nil {
		return err
	}

	return nil
}

func (pump *Pump) ReadPacket(buffer []byte) (int, *net.UDPAddr, error) {
	return pump.conn.ReadFromUDP(buffer)
}

func (pump *Pump) sendPacket(packet Packet) error {
	n := Encode(packet, pump.packetBuffer)
	_, err := pump.conn.WriteToUDP(pump.packetBuffer[:n], pump.remote)
	return err
}

func (pump *Pump) SetRemote(remote *net.UDPAddr) {
	pump.remote = remote
}

func (pump *Pump) SetPacketHandler(fn func(Packet)) {
	pump.onPacket = fn
}
