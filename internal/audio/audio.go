package audio

import (
	"fmt"
)

var backend Backend

func Init(b Backend) error {
	backend = b
	return backend.Init()
}

func Terminate() error {
	if backend == nil {
		return nil
	}
	return backend.Terminate()
}

func NewInput(format Format) (Input, error) {
	if backend == nil {
		return nil, fmt.Errorf("Audio backend not initialized")
	}
	return backend.NewInput(format)
}

func NewOutput(format Format) (Output, error) {
	if backend == nil {
		return nil, fmt.Errorf("Audio backend not initialized")
	}
	return backend.NewOutput(format)
}

type Input interface {
	Read([]int16) (int, error)
	Reconfigure(format Format) error
	Close() error
}

type Output interface {
	Write([]int16) (int, error)
	Reconfigure(format Format) error
	Close() error
}
