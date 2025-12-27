package media

import "github.com/Tomnowell/Mercury/internal/audio"

const frameDurationMs = 20

func SamplesPerFrame(rate audio.SampleRate) int {
	return int(rate) * frameDurationMs / 1000
}
