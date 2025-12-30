package media

import (
	"log"
	"net"

	"github.com/Tomnowell/Mercury/internal/audio"
)

const (
	maxPacketSize = 1500
)

type Pump struct {
	conn   *net.UDPConn
	remote *net.UDPAddr
	input  audio.Input
	output audio.Output
	format audio.Format

	sequence  uint16
	timestamp uint32

	packetBuffer []byte
	rx           chan []int16
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
		rx:           make(chan []int16, 3),
	}
}

func (pump *Pump) SendLoop() error {
	samplesPerFrame := SamplesPerFrame(pump.format.SampleRate)
	buffer := make([]int16, samplesPerFrame)
	payload := PCM16_16K

	if pump.format.SampleRate == audio.SampleRate8K {
		payload = PCM16_8K
	}

	for {
		n, err := pump.input.Read(buffer)
		if err != nil {
			return err
		}

		if n == 0 {
			// Buffer is empty
			continue
		}

		packet := Packet{
			Version:   1,
			Payload:   payload,
			Sequence:  pump.sequence,
			Timestamp: pump.timestamp,
			Samples:   buffer[:n],
		}

		if err := pump.sendPacket(packet); err != nil {
			return err
		}
		pump.sequence++
		pump.timestamp += uint32(n)
	}
}

func (pump *Pump) ReceiveLoop() error {
	buffer := make([]byte, maxPacketSize)

	for {
		n, _, err := pump.conn.ReadFromUDP(buffer)
		if err != nil {
			return err
		}

		packet, err := Decode(buffer[:n])
		if err != nil {
			return err
		}

		select {
		case pump.rx <- packet.Samples:
			//OK
		default:
			// Drop packet buffer full
		}

	}
}

func (pump *Pump) OutputLoop() {
	buffer := make([]int16, SamplesPerFrame(pump.format.SampleRate))

	for {
		select {
		case samples := <-pump.rx:
			if len(samples) == len(buffer) {
				copy(buffer, samples)
			} else {
				clear(buffer)

			}
		default:
			clear(buffer)
		}
		_, err := pump.output.Write(buffer)

		if err != nil {
			log.Println("Could not write to output")
		}

	}
}

func (pump *Pump) sendPacket(pkt Packet) error {
	n := Encode(pkt, pump.packetBuffer)
	_, err := pump.conn.WriteToUDP(pump.packetBuffer[:n], pump.remote)
	return err
}
