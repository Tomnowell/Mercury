package media

import (
	"github.com/Tomnowell/Mercury/internal/audio"
)

const frameDurationMs = 20

func SamplesPerFrame(rate audio.SampleRate) int {
	return int(rate) * frameDurationMs / 1000
}

func FormatFromPayload(payload PayloadType) audio.Format {
	switch payload {
	case PCM16_8K:
		return audio.Format{
			SampleRate: audio.SampleRate8K,
			Channels:   1,
		}
	default:
		return audio.Format{
			SampleRate: audio.SampleRate8K,
			Channels:   1,
		}
	}
}

func (pump *Pump) SetMediaGate(f func() bool) {
	pump.allowMedia = f
}
