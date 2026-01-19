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
	sequence     uint16
	timestamp    uint32
	packetBuffer []byte

	audioFIFO  *AudioFIFO
	jb         *JitterBuffer
	underflows int
	fallback   bool
	mu         sync.Mutex
	allowMedia func() bool
	outputDone chan struct{}
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
		audioFIFO:    NewAudioFifo(SamplesPerFrame(format.SampleRate) * 10),
		jb: NewJitterBuffer(
			0,
			3,
			SamplesPerFrame(format.SampleRate),
		),
		allowMedia: func() bool { return false },
	}
}

func (pump *Pump) AudioIngestLoop() {
	buffer := make([]int16, SamplesPerFrame(pump.format.SampleRate))

	for {
		pump.mu.Lock()
		in := pump.input
		pump.mu.Unlock()
		n, err := in.Read(buffer)
		if err != nil {
			log.Println("Audio ingest error: ", err)
			time.Sleep(10 * time.Millisecond)
			continue
		}

		samples := make([]int16, n)
		copy(samples, buffer[:n])
		pump.audioFIFO.Push(samples)
	}
}

func (pump *Pump) SendLoop() {

	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()

	samplesPerFrame := SamplesPerFrame(pump.format.SampleRate)

	for range ticker.C {
		if pump.allowMedia == nil || !pump.allowMedia() {
			continue
		}

		samples := pump.audioFIFO.Pop(samplesPerFrame)

		if samples == nil || len(samples) < samplesPerFrame {
			continue
		}

		packet := Packet{
			Version:   1,
			Payload:   PCM16_8K,
			Sequence:  pump.sequence,
			Timestamp: pump.timestamp,
			Samples:   samples,
		}

		err := pump.SendPacket(packet)

		if err != nil {
			log.Println("Error: Could not send packet")
			continue
		}
		pump.sequence++
		pump.timestamp += uint32(len(samples))
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
			if pump.underflows > 50 && !pump.fallback && pump.allowMedia() {
				// log.Println("Alert: Audio underflows > 50")
			}
		}
	}
}

func (pump *Pump) ResetSequence() {
	pump.mu.Lock()
	defer pump.mu.Unlock()
	pump.sequence = 0
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
	return pump.SendPacket(packet)
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

func (pump *Pump) SendPacket(packet Packet) error {
	n := Encode(packet, pump.packetBuffer)
	_, err := pump.conn.WriteToUDP(pump.packetBuffer[:n], pump.remote)
	return err
}

func (pump *Pump) SetRemote(remote *net.UDPAddr) {
	pump.mu.Lock()
	defer pump.mu.Unlock()
	pump.remote = remote
}

func (pump *Pump) Remote() *net.UDPAddr {
	pump.mu.Lock()
	defer pump.mu.Unlock()
	return pump.remote
}
