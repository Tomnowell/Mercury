package media

type PayloadType uint8

const (
	PCM16_8K PayloadType = 0
	// PCM16_16K PayloadType = 1
)

type ControlType uint8

const (
	CtrlInvite ControlType = iota
	CtrlOK
	CtrlAck
)

type Packet struct {
	// Packets can be either Control or Media
	// Control packets must not contain samples
	// media packets must not set IsControl

	Version   uint8
	IsControl bool

	CtrlType ControlType

	Payload   PayloadType
	Restart   bool
	Sequence  uint16
	Timestamp uint32
	Samples   []int16
}
