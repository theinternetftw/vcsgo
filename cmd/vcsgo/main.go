package main

import (
	"github.com/theinternetftw/vcsgo"
	"github.com/theinternetftw/vcsgo/profiling"
	"github.com/theinternetftw/vcsgo/platform"

	"golang.org/x/mobile/event/key"

	"fmt"
	"io/ioutil"
	"os"
	"time"

	"runtime/debug"
)

func main() {

	defer profiling.Start().Stop()

	assert(len(os.Args) == 2, "usage: ./vcsgo ROM_FILENAME")
	cartFilename := os.Args[1]

	cartBytes, err := ioutil.ReadFile(cartFilename)
	dieIf(err)

	emu := vcsgo.NewEmulator(cartBytes)

	screenW := 320
	screenH := 264
	platform.InitDisplayLoop("vcsgo", screenW*2, screenH*2, screenW, screenH, func(sharedState *platform.WindowState) {
		startEmu(cartFilename, sharedState, emu)
	})
}

func clamp(min, x, max float32) float32 {
	if x < min {
		return min
	} else if x > max {
		return max
	}
	return x
}
type paddlePhys struct {
	pos, vel float32
}
func (p *paddlePhys) move(dir, dt float32) {
	const maxvel = 135 // degrees/s^2
	if dir < 0 {
		p.vel = -maxvel
	} else if dir > 0 {
		p.vel = maxvel
	} else {
		p.vel = 0
	}
	p.pos += p.vel*dt
	p.pos = clamp(-135, p.pos, 135)
}
func (p *paddlePhys) left(dt float32) { p.move(-1, dt) }
func (p *paddlePhys) right(dt float32) { p.move(1, dt) }
func (p *paddlePhys) noMove(dt float32) { p.move(0, dt) }

func startEmu(filename string, window *platform.WindowState, emu vcsgo.Emulator) {

	// FIXME: settings are for debug right now
	lastFlipTime := time.Now()
	lastInputUpdateTime := time.Now()

	snapshotPrefix := filename + ".snapshot"

	audio, err := platform.OpenAudioBuffer(4, 4096, 44100, 16, 2)
	workingAudioBuffer := make([]byte, audio.BufferSize())
	dieIf(err)

	timer := time.NewTimer(0)
	<-timer.C

	maxRDiff := time.Duration(0)
	maxFDiff := 0.0
	frameCount := 0

	lastNumGC := int64(0)
	gcStats := debug.GCStats{}

	paddles := []paddlePhys{
		paddlePhys{}, paddlePhys{},
	}

	frametimeGoal := map[byte]float64 {
		vcsgo.FormatNTSC: 1.0/60.0,
		vcsgo.FormatPAL: 1.0/50.0,
	}[emu.GetTVFormat()]

	snapshotMode := 'x'

	for {
		newInput := vcsgo.Input {}
		numDown := 'x'

		inputDt := float32(time.Now().Sub(lastInputUpdateTime).Seconds())
		if inputDt > 0.001 {
			window.Mutex.Lock()
			{
				window.CopyKeyCharArray(newInput.Keys[:])

				newInput.ResetButton = window.CodeIsDown(key.CodeF1)
				newInput.SelectButton = window.CodeIsDown(key.CodeF2)

				newInput.JoyP0.Up = window.CodeIsDown(key.CodeW)
				newInput.JoyP0.Down = window.CodeIsDown(key.CodeS)
				newInput.JoyP0.Left = window.CodeIsDown(key.CodeA)
				newInput.JoyP0.Right = window.CodeIsDown(key.CodeD)
				newInput.JoyP0.Button = window.CodeIsDown(key.CodeJ)

				// TODO: switch between input methods (arg switch, plus
				//  about 20 MD5s for the games that use paddles)
				if window.CodeIsDown(key.CodeA) {
					paddles[0].left(inputDt)
				} else if window.CodeIsDown(key.CodeD) {
					paddles[0].right(inputDt)
				} else {
					paddles[0].noMove(inputDt)
				}
				newInput.Paddle0.Button = window.CodeIsDown(key.CodeJ)
				newInput.Paddle0.Position = int16(paddles[0].pos)

				if window.CodeIsDown(key.CodeLeftArrow) {
					paddles[1].left(inputDt)
				} else if window.CodeIsDown(key.CodeRightArrow) {
					paddles[1].right(inputDt)
				} else {
					paddles[1].noMove(inputDt)
				}
				newInput.Paddle1.Button = window.CodeIsDown(key.CodeSpacebar)
				newInput.Paddle1.Position = int16(paddles[1].pos)

				newInput.JoyP1.Up = window.CodeIsDown(key.CodeUpArrow)
				newInput.JoyP1.Down = window.CodeIsDown(key.CodeDownArrow)
				newInput.JoyP1.Left = window.CodeIsDown(key.CodeLeftArrow)
				newInput.JoyP1.Right = window.CodeIsDown(key.CodeRightArrow)
				newInput.JoyP1.Button = window.CodeIsDown(key.CodeSpacebar)
			}
			window.Mutex.Unlock()

			emu.UpdateInput(newInput)

			lastInputUpdateTime = time.Now()

			for r := '0'; r <= '9'; r++ {
				if newInput.Keys[r] {
					numDown = r
					break
				}
			}
			if newInput.Keys['m'] {
				snapshotMode = 'm'
			} else if newInput.Keys['l'] {
				snapshotMode = 'l'
			}
		}

		if numDown > '0' && numDown <= '9' {
			snapFilename := snapshotPrefix+string(numDown)
			if snapshotMode == 'm' {
				snapshotMode = 'x'
				numDown = 'x'
				snapshot := emu.MakeSnapshot()
				fmt.Println("writing snap!")
				err := ioutil.WriteFile(snapFilename, snapshot, os.FileMode(0644))
				if err != nil {
					fmt.Println("failed to write snapshot:", err)
				}
			} else if snapshotMode == 'l' {
				snapshotMode = 'x'
				numDown = 'x'
				snapBytes, err := ioutil.ReadFile(snapFilename)
				fmt.Println("loading snap!")
				if err != nil {
					fmt.Println("failed to load snapshot:", err)
					continue
				}
				newEmu, err := emu.LoadSnapshot(snapBytes)
				if err != nil {
					fmt.Println("failed to load snapshot:", err)
					continue
				}
				emu = newEmu
			}
		}

		emu.Step()

		bufferAvailable := audio.BufferAvailable()

		audioBufSlice := workingAudioBuffer[:bufferAvailable]
		audio.Write(emu.ReadSoundBuffer(audioBufSlice))

		if emu.FlipRequested() {
			window.Mutex.Lock()
			copy(window.Pix, emu.Framebuffer())
			window.RequestDraw()
			window.Mutex.Unlock()

			debug.ReadGCStats(&gcStats)
			if gcStats.NumGC != lastNumGC {
				lastNumGC = gcStats.NumGC
				//fmt.Println("GC!")
			}
			frameCount++
			if frameCount & 0x1ff == 0 {
				//fmt.Printf("maxRTime %.4f, maxFTime %.4f\n", maxRDiff.Seconds(), maxFDiff)
				maxRDiff = 0
				maxFDiff = 0
			}

			if frameCount & 0x1f == 0 {
				//fmt.Println("cmd-paddlePos", paddle0Position)
			}

			rDiff := time.Now().Sub(lastFlipTime)
			const accuracyProtection = 2*time.Millisecond
			ftGoalAsDuration := time.Duration(frametimeGoal*1000)*time.Millisecond
			maxSleep := ftGoalAsDuration - accuracyProtection
			toSleep := maxSleep - rDiff
			if toSleep > accuracyProtection {
				timer.Reset(toSleep)
				<-timer.C
			}

			fDiff := 0.0
			for fDiff < frametimeGoal-0.0005 { // seems to be about 0.0005 resolution? so leave a bit of play
				fDiff = time.Now().Sub(lastFlipTime).Seconds()
			}
			if rDiff > maxRDiff {
				maxRDiff = rDiff
			}
			if fDiff > maxFDiff {
				maxFDiff = fDiff
			}

			lastFlipTime = time.Now()
		}
	}
}

func assert(test bool, msg string) {
	if !test {
		fmt.Println(msg)
		os.Exit(1)
	}
}

func dieIf(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
