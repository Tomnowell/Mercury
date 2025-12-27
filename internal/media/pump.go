package media

import (
	"encoding/binary"
	"net"

	"github.com/Tomnowell/Mercury/internal/audio"
)

type Pump struct {
	conn   *net.UDPConn
	input  audio.Input
	output audio.Output
	format audio.Format

	sequence  uint16
	timestamp uint32
}

func NewPump(
	conn *net.UDPConn,
	input audio.Input,
	output audio.Output,
) *Pump {
	return &Pump{
		conn:   conn,
		input:  input,
		output: output,
	}
}

func (pump *Pump) SendLoop() error {
	samplesPerFrame := SamplesPerFrame(pump.format.SampleRate)
	buffer := make([]int16, samplesPerFrame)
	packetBuffer := make([]byte, 1500) // Assuming MTU 1500

	payload := PCM16_16K

	if pump.format.SampleRate == audio.SampleRate8K {
		payload = PCM16_8K
	}

	for {
		_, err := pump.input.Read(buffer)
		if err != nil {
			return err
		}

		packet := Packet{
			Version:   1,
			Payload:   payload,
			Sequence:  pump.sequence,
			Timestamp: pump.timestamp,
			Samples:   buffer,
		}

		n := Encode(packet, packetBuffer)

		_, err = pump.conn.Write(packetBuffer[:n])

		if err != nil {
			return err
		}
		pump.sequence++
		pump.timestamp += uint32(len(buffer))
	}
}

func (pump *Pump) ReceiveLoop() error {
	buffer := make([]byte, 1500)
	samples := make([]int16, SamplesPerFrame(pump.format.SampleRate))

	for {
		_, err := pump.conn.Read(buffer)
		if err != nil {
			return err
		}

		offset := 7

		for i := range samples {
			samples[i] = int16(binary.LittleEndian.Uint16(buffer[offset:]))
			offset += 2
		}

		_, err = pump.output.Write(samples)
		if err != nil {
			return err
		}

	}

}
