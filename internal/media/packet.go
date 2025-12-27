package media

import (
	"encoding/binary"
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
