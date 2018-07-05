package vcsgo

// Emulator exposes the public facing fns for an emulation session
type Emulator interface {
	Step()

	MakeSnapshot() []byte
	LoadSnapshot([]byte) (Emulator, error)

	Framebuffer() []byte
	FlipRequested() bool

	ReadSoundBuffer([]byte) []byte

	UpdateInput(input Input)

	GetTVFormat() byte
}

const (
	// FormatNTSC represents NTSC 60fps games
	FormatNTSC = iota
	// FormatPAL represents PAL 50fps games
	FormatPAL
)

// Input covers all outside info sent to the Emulator
// TODO: add dt?
type Input struct {
	// Keys is a bool array of keydown state
	Keys [256]bool

	ResetButton        bool
	SelectButton       bool
	TVBWSwitch         bool
	P0DifficultySwitch bool
	P1DifficultySwitch bool

	JoyP0 Joystick
	JoyP1 Joystick

	Paddle0 Paddle
	Paddle1 Paddle
	Paddle2 Paddle
	Paddle3 Paddle
}

// Joystick represents the buttons on a joystick
type Joystick struct {
	Up     bool
	Down   bool
	Left   bool
	Right  bool
	Button bool
}

// Paddle represents a paddle controller
type Paddle struct {
	Button bool
	// Position should range from -135 to +135
	Position int16
}

func (emu *emuState) UpdateInput(input Input) {
	emu.updateInput(input)
}

// NewEmulator creates an emulation session
func NewEmulator(cart []byte) Emulator {
	return newState(cart)
}

func (emu *emuState) MakeSnapshot() []byte {
	return emu.makeSnapshot()
}

func (emu *emuState) LoadSnapshot(snapBytes []byte) (Emulator, error) {
	return emu.loadSnapshot(snapBytes)
}

// Framebuffer returns the current state of the screen
func (emu *emuState) Framebuffer() []byte {
	return emu.framebuffer()
}

// FlipRequested indicates if a draw request is pending
// and clears it before returning
func (emu *emuState) FlipRequested() bool {
	return emu.flipRequested()
}

func (emu *emuState) Step() {
	emu.step()
}

func (emu *emuState) GetTVFormat() byte {
	return emu.TIA.TVFormat
}
