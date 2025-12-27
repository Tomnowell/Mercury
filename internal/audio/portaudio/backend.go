package portaudio

import (
	"github.com/Tomnowell/Mercury/internal/audio"
	pa "github.com/gordonklaus/portaudio"
)

type Backend struct{}

func New() audio.Backend {
	return &Backend{}
}

func (b *Backend) Init() error {
	return pa.Initialize()
}

func (b *Backend) Terminate() error {
	return pa.Terminate()
}

func (b *Backend) NewInput(format audio.Format) (audio.Input, error) {
	return NewInput(format)
}

func (b *Backend) NewOutput(format audio.Format) (audio.Output, error) {
	return NewOutput(format)
}
