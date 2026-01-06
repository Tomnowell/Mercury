package media

import (
	"encoding/binary"
	"errors"
)

func Decode(buffer []byte) (Packet, error) {
	var packet Packet

	if len(buffer) < 8 {
		return packet, errors.New("packet too small")
	}
	flags := buffer[0]

	packet.Version = flags >> 4
	packet.IsControl = (flags & 0x01) != 0
	packet.Restart = (flags & 0x02) != 0

	if packet.IsControl {
		packet.CtrlType = ControlType(buffer[1])
	} else {
		packet.Payload = PayloadType(buffer[1])
	}

	packet.Sequence = binary.BigEndian.Uint16(buffer[2:4])
	packet.Timestamp = binary.BigEndian.Uint32(buffer[4:8])

	if packet.IsControl {
		if len(buffer) >= 9 {
			packet.Payload = PayloadType(buffer[8])
		}
		return packet, nil
	}

	sampleBytes := buffer[8:]

	if len(sampleBytes)%2 != 0 {
		return packet, errors.New("sample bytes must be an even length array")
	}

	numSamples := len(sampleBytes) / 2
	packet.Samples = make([]int16, numSamples)

	for i := 0; i < numSamples; i++ {
		packet.Samples[i] = int16(binary.LittleEndian.Uint16(sampleBytes[i*2:]))
	}
	return packet, nil
}
