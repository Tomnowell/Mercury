package media

type JitterBuffer struct {
	size      uint16
	expected  uint16
	buffer    map[uint16][]int16
	frameSize int
	started   bool
}

func NewJitterBuffer(startSeq uint16, size int, frameSize int) *JitterBuffer {
	return &JitterBuffer{
		size:      uint16(size),
		expected:  startSeq,
		buffer:    make(map[uint16][]int16),
		frameSize: frameSize,
	}
}

func (jb *JitterBuffer) Push(seq uint16, samples []int16) {
	if !jb.started {
		jb.expected = seq
		jb.started = true
	}

	if seq < jb.expected {
		// Drop, samples ar too old
		return
	}

	if seq >= jb.expected+jb.size {
		// Drop, too far ahead
		return
	}

	buffer := make([]int16, len(samples))
	copy(buffer, samples)

	jb.buffer[seq] = buffer
}

func (jb *JitterBuffer) Pop() []int16 {

	if !jb.started {
		return nil
	}

	samples, ok := jb.buffer[jb.expected]
	if ok {
		delete(jb.buffer, jb.expected)
		jb.expected++
		return samples
	}

	// Missing packet, adbance
	jb.expected++
	return nil
}
