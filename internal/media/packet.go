package media

import (
	"encoding/binary"
	"errors"
)

type PayloadType uint8

const (
	PCM16_8K  PayloadType = 0
	PCM16_16K PayloadType = 1
)

type Packet struct {
	Version   uint8
	Payload   PayloadType
	Restart   bool
	Sequence  uint16
	Timestamp uint32
	Samples   []int16
}

func Encode(packet Packet, buffer []byte) int {
	flags := packet.Version << 4
	flags |= uint8(packet.Payload & 0x0F)
	if packet.Restart {
		flags |= 0x08
	}
	buffer[0] = flags
	binary.BigEndian.PutUint16(buffer[1:], packet.Sequence)
	binary.BigEndian.PutUint32(buffer[3:], packet.Timestamp)

	offset := 7

	for _, sample := range packet.Samples {
		binary.LittleEndian.PutUint16(buffer[offset:], uint16(sample))
		offset += 2
	}
	return offset
}

func Decode(buffer []byte) (Packet, error) {
	var packet Packet

	if len(buffer) < 7 {
		return packet, errors.New("packet too small")
	}
	flags := buffer[0]

	packet.Version = flags >> 4
	packet.Payload = PayloadType(flags & 0x0F)
	packet.Restart = (flags & 0x08) != 0

	packet.Sequence = binary.BigEndian.Uint16(buffer[1:3])
	packet.Timestamp = binary.BigEndian.Uint32(buffer[3:7])
	sampleBytes := buffer[7:]

	if len(sampleBytes)%2 != 0 {
		return packet, errors.New("sample bytes must be an even length array")
	}

	numSamples := len(sampleBytes) / 2
	packet.Samples = make([]int16, numSamples)

	offset := 0

	for i := 0; i < numSamples; i++ {
		packet.Samples[i] = int16(binary.LittleEndian.Uint16(sampleBytes[offset:]))
		offset += 2
	}
	return packet, nil
}
