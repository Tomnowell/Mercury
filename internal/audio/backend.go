package audio

type Backend interface {
	Init() error
	Terminate() error

	NewInput(format Format) (Input, error)
	NewOutput(format Format) (Output, error)
}
