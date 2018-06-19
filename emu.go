package vcsgo

// Emulator exposes the public facing fns for an emulation session
type Emulator interface {
	Step()

	LoadBinaryToMem(addr uint16, bin []byte) error

	MakeSnapshot() []byte
	LoadSnapshot([]byte) (Emulator, error)

	Framebuffer() []byte
	FlipRequested() bool

	ReadSoundBuffer([]byte) []byte

	UpdateInput(input Input)

	GetCycles() uint64
	GetCyclesPerSecond() uint64
}

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
}

// Joystick represents the buttons on a joystick
type Joystick struct {
	Up     bool
	Down   bool
	Left   bool
	Right  bool
	Button bool
}

func (emu *emuState) UpdateInput(input Input) {
	emu.updateInput(input)
}

// NewEmulator creates an emulation session
func NewEmulator(cart []byte) Emulator {
	return newState(cart)
}

func (emu *emuState) LoadBinaryToMem(addr uint16, bin []byte) error {
	return emu.loadBinaryToMem(addr, bin)
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

func (emu *emuState) GetCycles() uint64 {
	return emu.Cycles
}
func (emu *emuState) GetCyclesPerSecond() uint64 {
	return TIACyclesPerSecond
}
