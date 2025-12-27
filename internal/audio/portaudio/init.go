package portaudio

import "github.com/gordonklaus/portaudio"

func Init() error {
	return portaudio.Initialize()
}

func Terminate() error {
	return portaudio.Terminate()
}
