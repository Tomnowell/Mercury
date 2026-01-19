package main

import (
	"flag"
	"log"
	"net"

	"github.com/Tomnowell/Mercury/internal/audio"
	"github.com/Tomnowell/Mercury/internal/audio/portaudio"
	"github.com/Tomnowell/Mercury/internal/call"
	"github.com/Tomnowell/Mercury/internal/registry"
)

func main() {
	// Mark -- Flags ----------------------------------------------------------------------------------------------

	listenAddressString := flag.String("listen", "[::1]:5000", "local address")
	dialString := flag.String("dial", "", "number to dial")
	numberString := flag.String("number", "", "local phone number to register")
	registryURL := flag.String("registry", "http://[::1]:5678", "registry base URL")
	flag.Parse()

	// Mark -- Audio Init -----------------------------------------------------------------------------------------
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
		SampleRate: audio.SampleRate8K,
		Channels:   1,
	}

	// Mark -- LocalAddress ---------------------------------------------------------------------------------------------

	localAddr, err := net.ResolveUDPAddr("udp", *listenAddressString)
	if err != nil {
		log.Fatal(err)
	}

	// Mark -- Local Number (Required) ----------------------------------------------------------------------------------
	if *numberString == "" {
		log.Fatal("You must provide a -number for this device")
	}

	localNumber, err := registry.ParsePhoneNumber(*numberString)
	if err != nil {
		log.Fatal("Invalid local Number", err)
	}

	// Mark -- Determine Role -------------------------------------------------------------------------------------------
	role := call.RoleUnknown
	//if *dialString != "" {
	//		role = media.RoleCaller
	//	}

	// Mark -- Controller Setup -----------------------------------------------------------------------------------------
	controller, err := call.NewController(localNumber, format, localAddr, *registryURL)
	if err != nil {
		log.Fatal("Could not start Controller", err)
	}

	// Mark -- Role Specific Behaviour ----------------------------------------------------------------------------------
	switch role {
	case call.RoleCaller:
		// === CALLER ===
		targetNumber, err := registry.ParsePhoneNumber(*dialString)
		if err != nil {
			log.Fatal("Invalid number to dial", err)
		}

		log.Println("Dialing: ", targetNumber)
		err = controller.Dial(targetNumber)
		if err != nil {
			log.Fatal("Dialling failed: ", err)
		}

	case call.RoleUnknown:
		controller.RunCLI()
	}

	select {}
}
