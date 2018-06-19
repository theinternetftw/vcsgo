package vcsgo

import (
	"fmt"
	"os"

	"github.com/theinternetftw/cpugo/virt6502"
)

const (
	showMemReads  = false
	showMemWrites = false
)

const (
	// tiaCyclesPerLine = 228
	// cpuCyclesPerLine = 228/3
	// linesPerFrame = 262
	// framesPerSecond = 60
	tiaCyclesPerSecond = 228 * 262 * 60
)

type emuState struct {
	Mem mem

	CPU virt6502.Virt6502

	TIA tia

	APU apu

	Timer timer

	Input           Input
	LastKeyState    [256]bool
	DebugKeyPressed bool

	Input45LatchMode bool
	Input4LatchVal   bool
	Input5LatchVal   bool

	DDRModeMaskPortA byte
	DDRModeMaskPortB byte

	Cycles uint64
}

type timer struct {
	Val                            byte
	Interval                       int
	BaseClock                      int
	UnderflowSinceLastReadINTIM    bool
	UnderflowSinceLastReadINSTAT   bool
	UnderflowSinceLastWriteAnyTIMT bool
}

func (t *timer) runCycle() {
	t.BaseClock++
	if t.BaseClock >= t.Interval || t.UnderflowSinceLastReadINTIM {
		t.BaseClock = 0
		t.decTimer()
	}
}

func (t *timer) decTimer() {
	t.Val--
	if t.Val == 255 {
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
	t.decTimer()
}

func (t *timer) readINTIM() byte {
	if t.UnderflowSinceLastReadINTIM {
		t.UnderflowSinceLastReadINTIM = false
		t.BaseClock = 0
	}
	return t.Val
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
	for i := uint(0); i < cycles; i++ {
		emu.Cycles++

		emu.Timer.runCycle()
		for j := 0; j < 3; j++ {
			emu.TIA.runCycle()
		}
	}
}

func (emu *emuState) step() {

	// single step w/ debug printout
	if !emu.CPU.RESET {
		/*
			if !cs.DebugKeyPressed {
				cs.runCycles(1)
				return
			}
		*/
		emu.DebugKeyPressed = false
		if showMemReads {
			fmt.Println()
		}
		//fmt.Println(emu.CPU.DebugStatusLine())
	}

	if emu.TIA.WaitForHBlank {
		emu.TIA.runCycle() // 1 or 3? does it re-sync the clock?
		return
	}

	emu.CPU.Step()
}

func (emu *emuState) updateInput(input Input) {

	// just for debug
	for i, down := range input.Keys {
		if i > 127 {
			continue
		}
		if !emu.LastKeyState[i] && down {
			emu.DebugKeyPressed = true
		}
		emu.LastKeyState[i] = down
	}

	emu.Input = input
}

func (emu *emuState) loadBinaryToMem(addr uint16, bin []byte) error {
	fmt.Println("len", len(bin))
	if len(bin)+int(addr) > 0x10000 {
		return fmt.Errorf("binary len %v too big to load at %v", len(bin), addr)
	}
	for i, b := range bin {
		i16 := uint16(i)
		emu.write(addr+i16, b)
	}
	return nil
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

func newState(cart []byte) *emuState {
	emu := emuState{
		Mem: mem{
			ROM: cart,
		},
		Timer: timer{
			Interval: 1,
		},
		TIA: tia{
			ScreenX:  -68,
			InHBlank: true,
		},
	}
	emu.CPU = virt6502.Virt6502{
		RESET:     true,
		RunCycles: emu.runCycles,
		Write:     emu.write,
		Read:      emu.read,
		Err:       func(e error) { emuErr(e) },
	}

	return &emu
}

func emuErr(args ...interface{}) {
	fmt.Println(args...)
	os.Exit(1)
}
