package main

import (
	"flag"
	"fmt"
	"log"
	"net"

	"github.com/Tomnowell/Mercury/internal/audio"
	"github.com/Tomnowell/Mercury/internal/audio/portaudio"
	"github.com/Tomnowell/Mercury/internal/media"
)

func main() {
	local_address := flag.String("listen", "[::1]:5000", "local address")
	remote_address := flag.String("remote", "[::1]:5001", "remote address")
	flag.Parse()
	backend := portaudio.New()

	if err := audio.Init(backend); err != nil {
		log.Fatal(err)
	}

	defer func() {
		err := audio.Terminate()
		if err != nil {
			log.Println(err)
		}
	}()

	format := audio.Format{
		SampleRate: audio.SampleRate16K,
		Channels:   1,
	}
	input, _ := audio.NewInput(format)
	output, _ := audio.NewOutput(format)

	fmt.Println("Audio initialised")

	defer func(input audio.Input) {
		err := input.Close()
		if err != nil {
			log.Println(err)
		}
	}(input)
	defer func(output audio.Output) {
		err := output.Close()
		if err != nil {
			log.Println(err)
		}
	}(output)

	localAddr, err := net.ResolveUDPAddr("udp", *local_address)
	if err != nil {
		log.Fatal(err)
	}
	remoteAddr, err := net.ResolveUDPAddr("udp", *remote_address)
	if err != nil {
		log.Fatal(err)
	}
	conn, err := net.ListenUDP("udp6", localAddr)
	if err != nil {
		log.Fatal(err)
	}

	pump := media.NewPump(conn, remoteAddr, input, output, format)

	go func() {
		err := pump.SendLoop()
		if err != nil {
			log.Println(err)
		}
	}()

	go pump.OutputLoop()

	err = pump.ReceiveLoop()
	if err != nil {
		log.Println(err)
		return
	}

}
