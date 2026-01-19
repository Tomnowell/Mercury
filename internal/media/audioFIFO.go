package media

import "sync"

type AudioFIFO struct {
	mu       sync.Mutex
	buffer   []int16
	capacity int
}

func NewAudioFifo(capacity int) *AudioFIFO {
	return &AudioFIFO{
		buffer:   make([]int16, 0, capacity),
		capacity: capacity,
	}
}

func (fifo *AudioFIFO) Push(samples []int16) {
	fifo.mu.Lock()
	defer fifo.mu.Unlock()

	if len(samples) >= fifo.capacity {
		fifo.buffer = append(fifo.buffer[:0], samples[len(samples)-fifo.capacity:]...)
		return
	}

	overflow := len(fifo.buffer) + len(samples) - fifo.capacity

	if overflow > 0 {
		fifo.buffer = fifo.buffer[overflow:]
	}

	fifo.buffer = append(fifo.buffer, samples...)
}

func (fifo *AudioFIFO) Pop(n int) []int16 {
	fifo.mu.Lock()
	defer fifo.mu.Unlock()

	if len(fifo.buffer) < n {
		return nil
	}

	out := make([]int16, n)
	copy(out, fifo.buffer[:n])
	fifo.buffer = fifo.buffer[n:]

	return out
}

func (fifo *AudioFIFO) Len() int {
	fifo.mu.Lock()
	defer fifo.mu.Unlock()

	return len(fifo.buffer)
}

func (fifo *AudioFIFO) Clear() {
	fifo.mu.Lock()
	defer fifo.mu.Unlock()

	fifo.buffer = fifo.buffer[:0]
}
