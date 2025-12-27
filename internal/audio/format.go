package audio

type SampleRate int

const (
	SampleRate8K  SampleRate = 8000
	SampleRate16K SampleRate = 16000
)

type Format struct {
	SampleRate SampleRate
	Channels   int
}
