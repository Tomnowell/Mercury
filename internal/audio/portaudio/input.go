package portaudio

import (
	"errors"
	"fmt"

	"github.com/Tomnowell/Mercury/internal/audio"
	pa "github.com/gordonklaus/portaudio"
)

type Input struct {
	stream *pa.Stream
	format audio.Format
	buffer []int16
}

func NewInput(format audio.Format) (*Input, error) {
	in := &Input{
		format: format,
		buffer: make([]int16, format.Channels*256),
	}

	stream, err := pa.OpenDefaultStream(format.Channels, 0, float64(format.SampleRate), len(in.buffer), in.buffer)

	if err != nil {
		return nil, fmt.Errorf("Error: Failed to open PortAudio input stream %w", err)
	}

	in.stream = stream

	if err := stream.Start(); err != nil {
		return nil, fmt.Errorf("Error: Failed to start PortAudio input stream %w", err)
	}

	return in, nil

}

func (input *Input) Read(buffer []int16) (int, error) {
	n := len(buffer)

	if err := input.stream.Read(); err != nil {
		return 0, err
	}
	copy(buffer, input.buffer[:n])
	return n, nil
}

func (input *Input) Close() error {
	if input.stream != nil {
		_ = input.stream.Stop()
		_ = input.stream.Close()
		return nil
	}
	return errors.New("input stream nil, cannot Stop or Close")
}
