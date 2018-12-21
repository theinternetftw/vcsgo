package vcsgo

// Emulator exposes the public facing fns for an emulation session
type Emulator interface {
	Step()

	MakeSnapshot() []byte
	LoadSnapshot([]byte) (Emulator, error)

	Framebuffer() []byte
	FlipRequested() bool

	ReadSoundBuffer([]byte) []byte

	SetInput(input Input)

	SetDebugContinue(b bool)

	GetTVFormat() TVFormat
}

// TVFormat is for PAL/NTSC determination
type TVFormat byte

const (
	// FormatNTSC represents NTSC 60fps games
	FormatNTSC TVFormat = iota
	// FormatPAL represents PAL 50fps games
	FormatPAL
)

// Input covers all outside info sent to the Emulator
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

	Keypad0 [12]bool
	Keypad1 [12]bool
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

func (emu *emuState) SetInput(input Input) {
	emu.setInput(input)
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

func (emu *emuState) Framebuffer() []byte {
	return emu.framebuffer()
}

func (emu *emuState) SetDebugContinue(b bool) {
	emu.DebugContinue = b
}

// FlipRequested indicates if a draw request is pending
// and clears it before returning
func (emu *emuState) FlipRequested() bool {
	return emu.flipRequested()
}

func (emu *emuState) Step() {
	emu.step()
}

func (emu *emuState) GetTVFormat() TVFormat {
	return emu.TIA.TVFormat
}

// ReadSoundBuffer returns a 44100hz * 16bit * 2ch sound buffer.
// A pre-sized buffer must be provided, which is returned resized
// if the buffer was less full than the length requested.
func (emu *emuState) ReadSoundBuffer(toFill []byte) []byte {
	return emu.APU.readSoundBuffer(toFill)
}
