package media

import (
	"log"

	"github.com/Tomnowell/Mercury/internal/audio"
)

const frameDurationMs = 20

func SamplesPerFrame(rate audio.SampleRate) int {
	log.Println("Rate %w", rate)
	log.Println("Calculated SamplesPerFrame: %w", (int(rate) * frameDurationMs / 1000))
	return int(rate) * frameDurationMs / 1000
}
