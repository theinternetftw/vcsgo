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

	Input03TiedToLow           bool
	InputTimingPots            bool
	InputTimingPotsStartCycles uint64

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

	Cycles uint64
}

type timer struct {
	Val                            byte
	Interval                       int
	BaseClock                      int
	Underflow                      bool
	UnderflowSinceLastReadINTIM    bool
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
	if t.Val == 255 {
		t.Underflow = true
		t.UnderflowSinceLastReadINTIM = true
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
	t.UnderflowSinceLastReadINTIM = false
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

var lastCheck time.Time

func (emu *emuState) runCycles(cycles uint) {
	for i := uint(0); i < cycles; i++ {
		emu.Cycles++

		emu.Timer.runCycle()

		for j := 0; j < 3; j++ {
			emu.TIA.runCycle()
			emu.APU.runCycle()
		}
	}
	if emu.InputTimingPots {
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
			if paddle.Position < 0 && scanlines >= scanLimit {
				if time.Now().Sub(lastCheck).Seconds() > 0.2 {
					fmt.Println("pos:", paddle.Position, "limit:", scanLimit, "scanlines:", scanlines)
					lastCheck = time.Now()
				}
				*regs[i] = true
			}
		}
		allDone := true
		for _, reg := range regs {
			if !(*reg) {
				allDone = false
			}
		}
		if allDone {
			emu.InputTimingPots = false
		}
	}
}

func paddlePosToScanlines(pos int16) int16 {
	v := int16(float32(135-pos) / 270.0 * 380.0)
	if v < 0 {
		return 0
	} else if v > 380 {
		return 380
	}
	return v
}

type debugCmdType int

const (
	cmdStep = iota
	cmdRunto
	cmdContinue
	cmdDisplay
)

type debugCmd struct {
	cType debugCmdType
	argPC uint16
}

func getCmd(cmdStr string) (debugCmd, error) {
	parts := strings.Split(string(cmdStr), " ")
	if len(parts) == 1 && (parts[0] == "" || parts[0] == "s") {
		return debugCmd{cmdStep, 0}, nil
	}
	switch parts[0] {
	case "c":
		if len(parts) > 1 {
			return debugCmd{}, fmt.Errorf("continue takes no args")
		}
		return debugCmd{cmdContinue, 0}, nil
	case "d":
		if len(parts) > 1 {
			return debugCmd{}, fmt.Errorf("display takes no args")
		}
		return debugCmd{cmdDisplay, 0}, nil
	case "r":
		if len(parts) < 2 {
			return debugCmd{}, fmt.Errorf("need pc arg")
		}
		i, err := strconv.ParseUint(parts[1], 16, 16)
		if err != nil {
			return debugCmd{}, fmt.Errorf("bad pc arg: %v", err)
		}
		return debugCmd{cmdRunto, uint16(i)}, nil
	default:
		return debugCmd{}, fmt.Errorf("unknown command %q", parts[0])
	}
}

func (emu *emuState) step() {

	if emu.DebugKeyVal == '`' {
		emu.DebugKeyPressed = false
		emu.DebugContinue = false
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

		if cmd, err := getCmd(cmdStr); err != nil {
			fmt.Printf("* ERR - %v\n", err)
			return
		} else if cmd.cType == cmdContinue {
			emu.DebugContinue = true
		} else if cmd.cType == cmdDisplay {
			fmt.Println(emu.debugStatusLine())
			return
		} else if cmd.cType == cmdStep {
			fmt.Println(emu.debugStatusLine())
		} else if cmd.cType == cmdRunto {
			fmt.Printf("running to 0x%04x\n", cmd.argPC)
			for emu.CPU.PC != uint16(cmd.argPC) {
				emu.stepNoDbg()
			}
			fmt.Println(emu.debugStatusLine())
		} else {
			fmt.Println("* ERR - unknown BUT PARSED command")
			return
		}
	}

	//if showMemReads { fmt.Println() }

	emu.stepNoDbg()
}

func (emu *emuState) stepNoDbg() {
	if emu.TIA.WaitForHBlank {
		emu.runCycles(1) // 1 or 3? does it re-sync the clock?
		return
	}

	emu.CPU.Step()
}

func (emu *emuState) debugStatusLine() string {
	return fmt.Sprintf("%sT:0x%02x Tstep:%04d, Bx:%02x Bvx:%02d",
		emu.CPU.DebugStatusLine(),
		emu.Timer.Val,
		emu.Timer.Interval,
		emu.TIA.BL.X,
		emu.TIA.BL.Vx,
	)
}

func (emu *emuState) updateInput(input Input) {

	// just for debug
	for i := 0; i < 128; i++ {
		down := input.Keys[i]
		if !emu.LastKeyState[i] && down {
			emu.DebugKeyPressed = true
			emu.DebugKeyVal = byte(i)
		}
		emu.LastKeyState[i] = down
	}

	if emu.Input45LatchMode {
		if emu.Input4LatchVal {
			emu.Input4LatchVal = !emu.Input.JoyP0.Button
		}
		if emu.Input5LatchVal {
			emu.Input5LatchVal = !emu.Input.JoyP1.Button
		}
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
			rom: cart,
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
			ScreenX:  -68,
			InHBlank: true,
			M0:       sprite{Size: 1},
			M1:       sprite{Size: 1},
		},
	}

	emu.CPU = virt6502.Virt6502{
		RESET:     true,
		RunCycles: emu.runCycles,
		Write:     emu.write,
		Read:      emu.read,
		Err:       func(e error) { emuErr(e) },
	}
	emu.APU.init()

	// NOTE: random fill the RAM, but but still keep it
	// deterministic every start for the moment...
	rand.Read(emu.Mem.RAM[:])
}

func newState(cart []byte) *emuState {
	var emu emuState

	initEmuState(&emu, cart)
	tvFormat := discoverTVFormat(&emu)
	// start fresh with correct format
	initEmuState(&emu, cart)
	emu.TIA.TVFormat = tvFormat

	return &emu
}

// discoverTVFormat runs a headless version of emulation for
// a few frames and returns whether it thinks its PAL or not
func discoverTVFormat(emu *emuState) byte {

	const numTestFrames = 10

	frames := 0
	nullInput := Input{}
	emu.DebugContinue = true
	for frames < numTestFrames {
		emu.UpdateInput(nullInput)
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
