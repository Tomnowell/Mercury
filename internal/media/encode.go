package media

import (
	"encoding/binary"
)

func Encode(packet Packet, buffer []byte) int {
	flags := packet.Version << 4
	if packet.IsControl {
		flags |= 0x01
	}
	if packet.Restart {
		flags |= 0x02
	}

	buffer[0] = flags

	if packet.IsControl {
		buffer[1] = byte(packet.CtrlType)
	} else {
		buffer[1] = byte(packet.Payload)
	}

	binary.BigEndian.PutUint16(buffer[2:], packet.Sequence)
	binary.BigEndian.PutUint32(buffer[4:], packet.Timestamp)

	offset := 8

	if packet.IsControl {
		buffer[offset] = byte(packet.Payload)
		return offset + 1
	}

	for _, sample := range packet.Samples {
		binary.LittleEndian.PutUint16(buffer[offset:], uint16(sample))
		offset += 2
	}
	return offset
}
