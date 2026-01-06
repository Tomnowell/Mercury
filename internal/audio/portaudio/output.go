package portaudio

import (
	"errors"
	"fmt"
	"log"

	"github.com/Tomnowell/Mercury/internal/audio"
	"github.com/Tomnowell/Mercury/internal/media"
	pa "github.com/gordonklaus/portaudio"
)

type Output struct {
	stream *pa.Stream
	format audio.Format
	buffer []int16
}

func NewOutput(format audio.Format) (*Output, error) {
	out := &Output{
		format: format,
		buffer: make([]int16, format.Channels*media.SamplesPerFrame(format.SampleRate)),
	}

	stream, err := pa.OpenDefaultStream(0, format.Channels, float64(format.SampleRate), len(out.buffer), out.buffer)
	if err != nil {
		return nil, fmt.Errorf("Error: Failed to open audio stream %w", err)
	}

	out.stream = stream

	if err := stream.Start(); err != nil {
		return nil, fmt.Errorf("Error: Failed to start stream %w", err)
	}

	log.Println("PortAudio Output Started")
	return out, nil
}

func (out *Output) Write(buffer []int16) (int, error) {
	n := len(buffer)
	copy(out.buffer[:n], buffer)
	if err := out.stream.Write(); err != nil {
		return 0, err
	}
	return n, nil
}

func (out *Output) Close() error {
	if out.stream != nil {
		_ = out.stream.Stop()
		_ = out.stream.Close()
		out.stream = nil
		return nil
	}
	return errors.New("Error: out stream was nil, can't close non-existent stream")
}

func (out *Output) Reconfigure(format audio.Format) error {
	out.stream.Stop()
	out.stream.Close()

	out.format = format
	out.buffer = make([]int16, format.Channels*media.SamplesPerFrame(format.SampleRate))

	stream, err := pa.OpenDefaultStream(
		format.Channels,
		0,
		float64(format.SampleRate),
		len(out.buffer),
		&out.buffer,
	)
	if err != nil {
		return err
	}

	out.stream = stream

	return out.stream.Start()

}
