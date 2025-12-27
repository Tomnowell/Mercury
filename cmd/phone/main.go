package main

import (
	"fmt"
	"log"

	"github.com/Tomnowell/Mercury/internal/audio"
	"github.com/Tomnowell/Mercury/internal/audio/portaudio"
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

	// Prototype V1 hardcoded buffer size - this should be adjustable or at least programatically adaptable.
	buffer := make([]int16, 256)

	log.Println("Loopback Started: Please talk into the mic to test (Ctrl+C) to stop")

	for {
		n, err := input.Read(buffer)
		if err != nil {
			log.Fatal(err)
		}

		_, err = output.Write(buffer[:n])
		if err != nil {
			log.Fatal(err)
		}
	}

}
