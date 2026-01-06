package main

import (
	"flag"
	"log"
	"net"

	"github.com/Tomnowell/Mercury/internal/audio"
	"github.com/Tomnowell/Mercury/internal/audio/portaudio"
	"github.com/Tomnowell/Mercury/internal/call"
	"github.com/Tomnowell/Mercury/internal/media"
)

func main() {
	local_address := flag.String("listen", "[::1]:5000", "local address")
	remote_address := flag.String("remote", "[::1]:5001", "remote address")
	roleFlag := flag.String("role", "caller", `call role: "caller" or "callee"`)
	flag.Parse()

	role, err := media.GetRole(*roleFlag)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Starting as role:", role)
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

	localAddr, err := net.ResolveUDPAddr("udp", *local_address)
	if err != nil {
		log.Fatal(err)
	}
	remoteAddr, err := net.ResolveUDPAddr("udp", *remote_address)
	if err != nil {
		log.Fatal(err)
	}

	controller, err := call.NewController(
		role,
		audio.Format{
			SampleRate: audio.SampleRate8K,
			Channels:   1,
		},
		localAddr,
		remoteAddr,
	)

	if err != nil {
		log.Fatal(err)
	}

	defer controller.Close()

	controller.Run()

	select {}

}
