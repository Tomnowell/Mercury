package media

import (
	"encoding/binary"
	"errors"
)

func Decode(buffer []byte, n int) (Packet, error) {
	if n < 8 {
		return Packet{}, errors.New("packet too short")
	}

	var packet Packet

	// === FLAGS ===
	flags := buffer[0]
	packet.Version = flags >> 4
	packet.IsControl = flags&0x01 != 0
	packet.Restart = flags&0x02 != 0

	// === TYPE ===
	packet.CtrlType = ControlType(buffer[1])

	// === SEQUENCE / TIMESTAMP ===
	packet.Sequence = binary.BigEndian.Uint16(buffer[2:])
	packet.Timestamp = binary.BigEndian.Uint32(buffer[4:])

	offset := 8

	// === CONTROL PACKETS ===
	if packet.IsControl {

		if packet.CtrlType == CtrlInvite {
			if offset >= n {
				return Packet{}, errors.New("invite missing source length")
			}

			// Source number
			srcLen := int(buffer[offset])
			offset++

			if offset+srcLen > n {
				return Packet{}, errors.New("invite source truncated")
			}

			packet.Source = string(buffer[offset : offset+srcLen])
			offset += srcLen

			if offset >= n {
				return Packet{}, errors.New("invite missing target length")
			}

			// Target number
			dstLen := int(buffer[offset])
			offset++

			if offset+dstLen > n {
				return Packet{}, errors.New("invite target truncated")
			}

			packet.Target = string(buffer[offset : offset+dstLen])
			offset += dstLen
		}

		return packet, nil
	}

	// === MEDIA PACKETS ===
	if (n-offset)%2 != 0 {
		return Packet{}, errors.New("invalid media payload length")
	}

	sampleCount := (n - offset) / 2
	packet.Samples = make([]int16, sampleCount)

	for i := 0; i < sampleCount; i++ {
		packet.Samples[i] = int16(binary.LittleEndian.Uint16(buffer[offset:]))
		offset += 2
	}

	return packet, nil
}
