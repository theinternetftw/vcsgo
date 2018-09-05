package vcsgo

import (
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/theinternetftw/cpugo/virt6502"
)

const (
	showMemReads  = false
	showMemWrites = false
)

const (
	tiaCyclesPerScanline = 228
	cpuCyclesPerScanline = 228 / 3
	// scanlinesPerFrame = 262
	// framesPerSecond = 60
	tiaCyclesPerSecond = 1193182 * 3
)

type emuState struct {
	Mem mem

	CPU virt6502.Virt6502

	TIA tia

	APU apu

	Timer timer

	Input        Input
	LastKeyState [256]bool

	DebugKeyPressed bool
	DebugKeyVal     byte
	DebugContinue   bool
	DebugCmdStr     []byte
	DebugLastCmd    dbgCmd

	Input03TiedToLow           bool
	InputTimingPots            bool
	InputTimingPotsStartCycles uint64

	JoystickButtonChecksThisFrame int
	JoystickButtonChecksLastFrame int
	PaddleChecksThisFrame         int
	PaddleChecksLastFrame         int
	PaddleCodeFrames              int
	LastPaddleFrameReset          int
	InputPotsBeingUsed            bool

	Paddle0InputCharged bool
	Paddle1InputCharged bool
	Paddle2InputCharged bool
	Paddle3InputCharged bool

	Input45LatchMode bool
	Input4LatchVal   bool
	Input5LatchVal   bool

	SwtchBUnusedBit0 bool
	SwtchBUnusedBit1 bool
	SwtchBUnusedBit2 bool

	DDRModeMaskPortA byte
	DDRModeMaskPortB byte

	DirectionPortA byte

	RowSelKeypad0       byte
	RowSelKeypad1       byte
	EverSelectedKeypad0 bool
	EverSelectedKeypad1 bool

	Cycles uint64
}

type timer struct {
	Val                            byte
	Interval                       int
	BaseClock                      int
	Underflow                      bool
	UnderflowSinceLastReadINSTAT   bool
	UnderflowSinceLastWriteAnyTIMT bool
}

func (t *timer) runCycle() {
	t.BaseClock++
	if t.BaseClock >= t.Interval || t.Underflow {
		t.BaseClock = 0
		t.decTimer()
	}
}

func (t *timer) decTimer() {
	t.Val--
	if t.Val == 255 && !t.Underflow {
		t.Underflow = true
		t.UnderflowSinceLastReadINSTAT = true
		t.UnderflowSinceLastWriteAnyTIMT = true
	}
}

func (t *timer) writeAnyTIMT(interval int, val byte) {
	t.Val = val
	t.BaseClock = 0
	t.Interval = interval
	t.UnderflowSinceLastWriteAnyTIMT = false
	t.Underflow = false
	t.decTimer() // NOTE: does this happen now or the next tick after write?
}

func (t *timer) readINTIM() byte {
	if t.Underflow {
		t.Underflow = false
		t.BaseClock = 0 // NOTE: is this right?
	}
	return t.Val
}

func (t *timer) readINSTAT() byte {
	val := byteFromBools(
		t.UnderflowSinceLastWriteAnyTIMT,
		t.UnderflowSinceLastReadINSTAT,
		false, false, false, false, false, false,
	)
	t.UnderflowSinceLastReadINSTAT = false
	return val
}

func (emu *emuState) flipRequested() bool {
	result := emu.TIA.flipRequested
	emu.TIA.flipRequested = false
	return result
}

func (emu *emuState) framebuffer() []byte {
	return emu.TIA.Screen[:]
}

func (emu *emuState) runCycles(cycles uint) {

	for i := uint(0); emu.TIA.WaitForHBlank || i < cycles; i++ {
		emu.Cycles++

		emu.Timer.runCycle()
		emu.Mem.mapper.runCycle(emu)

		emu.TIA.runThreeCycles()
		emu.APU.runThreeCycles()
	}

	if emu.Input45LatchMode {
		emu.handleLatchMode()
	}

	if emu.InputTimingPots {
		emu.handlePotTiming()
		emu.doPotHeuristics()
	}
}

func (emu *emuState) handleLatchMode() {
	if emu.Input4LatchVal {
		emu.Input4LatchVal = !emu.Input.JoyP0.Button
	}
	if emu.Input5LatchVal {
		emu.Input5LatchVal = !emu.Input.JoyP1.Button
	}
}

func (emu *emuState) handlePotTiming() {
	diff := emu.Cycles - emu.InputTimingPotsStartCycles
	scanlines := int16(diff / cpuCyclesPerScanline)
	paddles, regs := []Paddle{
		emu.Input.Paddle0, emu.Input.Paddle1,
		emu.Input.Paddle2, emu.Input.Paddle3,
	}, []*bool{
		&emu.Paddle0InputCharged, &emu.Paddle1InputCharged,
		&emu.Paddle2InputCharged, &emu.Paddle3InputCharged,
	}
	for i, paddle := range paddles {
		scanLimit := paddlePosToScanlines(paddle.Position)
		if scanlines >= scanLimit {
			*regs[i] = true
		}
	}
}

func (emu *emuState) doPotHeuristics() {
	if !emu.InputPotsBeingUsed {
		if emu.LastPaddleFrameReset != emu.TIA.FrameCount {
			emu.LastPaddleFrameReset = emu.TIA.FrameCount
			emu.PaddleChecksLastFrame = emu.PaddleChecksThisFrame
			emu.PaddleChecksThisFrame = 0
			emu.JoystickButtonChecksLastFrame = emu.JoystickButtonChecksThisFrame
			emu.JoystickButtonChecksThisFrame = 0
		}
		fewChecksButNoJoy := emu.PaddleChecksLastFrame >= 20 && emu.JoystickButtonChecksLastFrame == 0
		if fewChecksButNoJoy || emu.PaddleChecksLastFrame >= 60 {
			emu.PaddleCodeFrames++
			if emu.PaddleCodeFrames >= 20 {
				fmt.Println("Paddle code found: Joysticks disabled", emu.PaddleChecksLastFrame)
				emu.InputPotsBeingUsed = true
			}
		} else {
			emu.PaddleCodeFrames = 0
		}
	}
}

func paddlePosToScanlines(pos int16) int16 {
	const outRange = 380
	v := int16(float32(135-pos) / 270.0 * outRange)
	if v < 0 {
		return 0
	} else if v > outRange {
		return outRange
	}
	return v
}

type dbgCmdType int

const (
	cmdStep dbgCmdType = iota
	cmdRunto
	cmdContinue
	cmdDisplay
	cmdStepLine
	cmdStepFrame
	cmdRepeat
	cmdErr
)

type dbgCmd struct {
	cType  dbgCmdType
	argPC  uint16
	errMsg error
}

func getDbgCmd(cmdStr string) dbgCmd {
	parts := strings.Split(string(cmdStr), " ")
	if len(parts) == 1 && parts[0] == "" {
		return dbgCmd{cType: cmdRepeat}
	}
	switch parts[0] {
	case "s":
		return dbgCmd{cType: cmdStep}
	case "c":
		return dbgCmd{cType: cmdContinue}
	case "l":
		return dbgCmd{cType: cmdStepLine}
	case "f":
		return dbgCmd{cType: cmdStepFrame}
	case "d":
		return dbgCmd{cType: cmdDisplay}
	case "r":
		if len(parts) < 2 {
			return dbgCmd{cType: cmdErr, errMsg: fmt.Errorf("need pc arg")}
		}
		i, err := strconv.ParseUint(parts[1], 16, 16)
		if err != nil {
			return dbgCmd{cType: cmdErr, errMsg: fmt.Errorf("bad pc arg: %v", err)}
		}
		return dbgCmd{cType: cmdRunto, argPC: uint16(i)}
	default:
		return dbgCmd{cType: cmdErr, errMsg: fmt.Errorf("unknown command %q", parts[0])}
	}
}

func (emu *emuState) step() {

	if emu.DebugKeyVal == '`' {
		emu.DebugKeyPressed = false
		emu.DebugContinue = false
		emu.TIA.ShowDebugPuck = true
		return
	}

	if !emu.DebugContinue {
		// single step w/ debug printout
		if !emu.DebugKeyPressed {
			//emu.runCycles(1)
			return
		}
		emu.DebugKeyPressed = false

		if emu.DebugKeyVal != '\r' {
			fmt.Printf("%c", emu.DebugKeyVal)
			emu.DebugCmdStr = append(emu.DebugCmdStr, emu.DebugKeyVal)
			return
		}

		if len(emu.DebugCmdStr) > 0 {
			fmt.Println()
		}
		cmdStr := string(emu.DebugCmdStr)
		emu.DebugCmdStr = emu.DebugCmdStr[:0]

		runUntil := func(cond func() bool, timeout time.Duration) {
			start := time.Now()
			for !cond() {
				emu.stepNoDbg()
				if time.Now().Sub(start) > timeout {
					fmt.Printf("TIMED OUT: ran to 0x%04x\n", emu.CPU.PC)
					break
				}
			}
		}

		cmd := getDbgCmd(cmdStr)

		if cmd.cType == cmdRepeat {
			cmd = emu.DebugLastCmd
		}

		switch cmd.cType {
		case cmdContinue:
			emu.DebugContinue = true
			emu.TIA.ShowDebugPuck = false
		case cmdDisplay:
			fmt.Println(emu.debugStatusLine())
		case cmdStep:
			emu.stepNoDbg()
			fmt.Println(emu.debugStatusLine())
		case cmdStepLine:
			fmt.Println("stepping to next scanline")
			startLine := emu.TIA.ScreenY
			runUntil(func() bool {
				return startLine != emu.TIA.ScreenY
			}, 5*time.Second)
			fmt.Println(emu.debugStatusLine())
		case cmdStepFrame:
			fmt.Println("stepping to next frame")
			startFrame := emu.TIA.FrameCount
			runUntil(func() bool {
				return startFrame != emu.TIA.FrameCount
			}, 5*time.Second)
			fmt.Println(emu.debugStatusLine())
		case cmdRunto:
			fmt.Printf("running to 0x%04x\n", cmd.argPC)
			runUntil(func() bool {
				return emu.CPU.PC == cmd.argPC
			}, 5*time.Second)
			fmt.Println(emu.debugStatusLine())
		case cmdErr:
			fmt.Printf("* ERR - %v\n", cmd.errMsg)
		default:
			fmt.Println("* ERR - unknown BUT PARSED command")
		}
		emu.DebugLastCmd = cmd

		emu.TIA.flipRequested = true
	} else {
		//if showMemReads { fmt.Println() }
		emu.stepNoDbg()
	}
}

func (emu *emuState) stepNoDbg() {

	emu.CPU.Step()
}

func (emu *emuState) debugStatusLine() string {
	return fmt.Sprintf("%sT:0x%02x Tstep:%04d, sX:%03d sY:%03d, p1X:%03d, p1Vx:%03d",
		emu.CPU.DebugStatusLine(),
		emu.Timer.Val,
		emu.Timer.Interval,
		emu.TIA.ScreenX,
		emu.TIA.ScreenY,
		emu.TIA.P1.X,
		emu.TIA.P1.Vx,
	)
}

func (emu *emuState) setInput(input Input) {

	// just for debug
	for i := 0; i < 128; i++ {
		down := input.Keys[i]
		if !emu.LastKeyState[i] && down {
			emu.DebugKeyPressed = true
			emu.DebugKeyVal = byte(i)
		}
		emu.LastKeyState[i] = down
	}

	if emu.InputPotsBeingUsed {
		input.JoyP0 = Joystick{}
		input.JoyP1 = Joystick{}
	} else {
		input.Paddle0 = Paddle{}
		input.Paddle1 = Paddle{}
		input.Paddle2 = Paddle{}
		input.Paddle3 = Paddle{}
	}

	// NOTE: just thanks to current keypad/joystick keybindings
	if emu.EverSelectedKeypad0 {
		input.JoyP0 = Joystick{}
	}

	emu.Input = input
}

func (emu *emuState) reset() {
	emu.CPU.RESET = true
}

// ReadSoundBuffer returns a 44100hz * 16bit * 2ch sound buffer.
// A pre-sized buffer must be provided, which is returned resized
// if the buffer was less full than the length requested.
func (emu *emuState) ReadSoundBuffer(toFill []byte) []byte {
	return emu.APU.buffer.read(toFill)
}

func initEmuState(emu *emuState, cart []byte) {
	*emu = emuState{
		Mem: mem{
			mapper: loadMapperFromRomInfo(cart),
			rom:    cart,
		},
		Timer: timer{
			Interval: 1024,

			// NOTE: correct and important for
			// random number generation in games,
			// still, lets have determinism every
			// start for the moment.
			Val: byte(rand.Uint32()),
		},
		TIA: tia{
			ScreenX:       -68,
			InHBlank:      true,
			Palette:       ntscPalette,
			M0:            sprite{Size: 1},
			M1:            sprite{Size: 1},
			ShowDebugPuck: true,
		},
	}

	emu.CPU = virt6502.Virt6502{
		RESET:     true,
		RunCycles: emu.runCycles,
		Write:     emu.write,
		Read:      emu.read,
		Err:       func(e error) { emuErr(e) },
	}
	emu.APU.init(emu)
	emu.TIA.init(emu)

	// NOTE: random fill the RAM, but but still keep it
	// deterministic every start for the moment...
	rand.Read(emu.Mem.RAM[:])

	emu.Mem.mapper.init(emu)
}

func newState(cart []byte) *emuState {
	var emu emuState

	initEmuState(&emu, cart)
	fmt.Println("ROM Size:", len(emu.Mem.rom))
	fmt.Printf("Mapper: 0x%02x\n", emu.Mem.mapper.getMapperNum())

	tvFormat := discoverTVFormat(&emu)
	// start fresh with correct format
	initEmuState(&emu, cart)
	emu.TIA.setTVFormat(tvFormat)
	fmt.Println()

	return &emu
}

// discoverTVFormat runs a headless version of emulation for
// a few frames and returns whether it thinks its PAL or not
func discoverTVFormat(emu *emuState) TVFormat {

	startTime := time.Now()

	frames := 0
	nullInput := Input{}
	emu.DebugContinue = true
	for !emu.TIA.FormatSet {
		if time.Now().Sub(startTime) > 2*time.Second {
			break
		}
		emu.SetInput(nullInput)
		emu.Step()
		if emu.FlipRequested() {
			frames++
		}
	}
	return emu.TIA.TVFormat
}

func emuErr(args ...interface{}) {
	fmt.Println(args...)
	os.Exit(1)
}
