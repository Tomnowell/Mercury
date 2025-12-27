package main

import (
	"fmt"
	"log"
	"net"

	"github.com/Tomnowell/Mercury/internal/audio"
	"github.com/Tomnowell/Mercury/internal/audio/portaudio"
	"github.com/Tomnowell/Mercury/internal/media"
)

func main() {
	backend := portaudio.New()

	if err := audio.Init(backend); err != nil {
		log.Fatal(err)
	}

	defer audio.Terminate()

	fmt.Println("Audio initialised")

	format16k := audio.Format{
		SampleRate: audio.SampleRate16K,
		Channels:   1,
	}
	input, _ := audio.NewInput(format16k)
	output, _ := audio.NewOutput(format16k)

	defer input.Close()
	defer output.Close()

	receive_addr, _ := net.ResolveUDPAddr("udp6", "[::1]:6969")
	conn, _ := net.DialUDP("udp6", nil, receive_addr)

	pump := media.NewPump(conn, input, output)

	go pump.SendLoop()
	pump.ReceiveLoop()

}
