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
	buffer[1] = byte(packet.CtrlType)

	binary.BigEndian.PutUint16(buffer[2:], packet.Sequence)
	binary.BigEndian.PutUint32(buffer[4:], packet.Timestamp)

	offset := 8

	// === CONTROL PACKETS ===
	if packet.IsControl {

		// INVITE carries identity
		if packet.CtrlType == CtrlInvite {
			src := []byte(packet.Source)
			dst := []byte(packet.Target)

			buffer[offset] = byte(len(src))
			offset++
			copy(buffer[offset:], src)
			offset += len(src)

			buffer[offset] = byte(len(dst))
			offset++
			copy(buffer[offset:], dst)
			offset += len(dst)
		}

		return offset
	}

	// === MEDIA PACKETS ===
	for _, sample := range packet.Samples {
		binary.LittleEndian.PutUint16(buffer[offset:], uint16(sample))
		offset += 2
	}

	return offset
}
