package main

import (
	"github.com/theinternetftw/glimmer"
	"github.com/theinternetftw/vcsgo"
	"github.com/theinternetftw/vcsgo/profiling"

	"fmt"
	"io/ioutil"
	"os"
	"time"
)

func main() {

	defer profiling.Start().Stop()

	assert(len(os.Args) == 2, "usage: ./vcsgo ROM_FILENAME")
	cartFilename := os.Args[1]

	cartBytes, err := ioutil.ReadFile(cartFilename)
	dieIf(err)

	devMode := fileExists("devmode")

	emu := vcsgo.NewEmulator(cartBytes, devMode)

	screenW := 320
	screenH := 264
	glimmer.InitDisplayLoop("vcsgo", screenW*2, screenH*2, screenW, screenH, func(sharedState *glimmer.WindowState) {
		startEmu(cartFilename, sharedState, emu)
	})
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
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
	p.pos += p.vel * dt
	p.pos = clamp(-135, p.pos, 135)
}
func (p *paddlePhys) left(dt float32)   { p.move(-1, dt) }
func (p *paddlePhys) right(dt float32)  { p.move(1, dt) }
func (p *paddlePhys) noMove(dt float32) { p.move(0, dt) }

func startEmu(filename string, window *glimmer.WindowState, emu vcsgo.Emulator) {

	// FIXME: settings are for debug right now
	lastFlipTime := time.Now()
	lastInputPollTime := time.Now()

	snapshotPrefix := filename + ".snapshot"

	audio, err := glimmer.OpenAudioBuffer(1, 8192, 44100, 16, 2)
	workingAudioBuffer := make([]byte, audio.BufferSize())
	dieIf(err)

	timer := time.NewTimer(0)
	<-timer.C

	maxRDiff := time.Duration(0)
	maxFDiff := 0.0
	frameCount := 0

	paddles := []paddlePhys{
		paddlePhys{}, paddlePhys{},
	}

	frametimeGoal := map[vcsgo.TVFormat]float64{
		vcsgo.FormatNTSC: 1.0 / 60.0,
		vcsgo.FormatPAL:  1.0 / 50.0,
	}[emu.GetTVFormat()]

	snapshotMode := 'x'

	newInput := vcsgo.Input{}

	for {

		now := time.Now()

		inputDiff := now.Sub(lastInputPollTime)
		if inputDiff > 8*time.Millisecond {

			numDown := 'x'

			inputDt := float32(inputDiff.Seconds())
			newInput = vcsgo.Input{}

			window.Mutex.Lock()
			{
				window.CopyKeyCharArray(newInput.Keys[:])

				cid := func(c glimmer.KeyCode) bool { return window.CodeIsDown(c) }

				newInput.ResetButton = cid(glimmer.CodeF1)
				newInput.SelectButton = cid(glimmer.CodeF2)

				newInput.JoyP0.Up = cid(glimmer.CodeW)
				newInput.JoyP0.Down = cid(glimmer.CodeS)
				newInput.JoyP0.Left = cid(glimmer.CodeA)
				newInput.JoyP0.Right = cid(glimmer.CodeD)
				newInput.JoyP0.Button = cid(glimmer.CodeJ)

				// TODO: switch between input methods (arg switch, plus
				//  about 20 MD5s for the games that use paddles)
				if cid(glimmer.CodeA) {
					paddles[0].left(inputDt)
				} else if cid(glimmer.CodeD) {
					paddles[0].right(inputDt)
				} else {
					paddles[0].noMove(inputDt)
				}
				newInput.Paddle0.Button = window.CodeIsDown(glimmer.CodeJ)
				newInput.Paddle0.Position = int16(paddles[0].pos)

				newInput.Keypad0 = [12]bool{
					cid(glimmer.Code1), cid(glimmer.Code2), cid(glimmer.Code3),
					cid(glimmer.CodeQ), cid(glimmer.CodeW), cid(glimmer.CodeE),
					cid(glimmer.CodeA), cid(glimmer.CodeS), cid(glimmer.CodeD),
					cid(glimmer.CodeZ), cid(glimmer.CodeX), cid(glimmer.CodeC),
				}

				newInput.Keypad1 = [12]bool{
					cid(glimmer.Code4), cid(glimmer.Code5), cid(glimmer.Code6),
					cid(glimmer.CodeR), cid(glimmer.CodeT), cid(glimmer.CodeY),
					cid(glimmer.CodeF), cid(glimmer.CodeG), cid(glimmer.CodeH),
					cid(glimmer.CodeV), cid(glimmer.CodeB), cid(glimmer.CodeN),
				}

				if window.CodeIsDown(glimmer.CodeLeftArrow) {
					paddles[1].left(inputDt)
				} else if window.CodeIsDown(glimmer.CodeRightArrow) {
					paddles[1].right(inputDt)
				} else {
					paddles[1].noMove(inputDt)
				}
				newInput.Paddle1.Button = window.CodeIsDown(glimmer.CodeSpacebar)
				newInput.Paddle1.Position = int16(paddles[1].pos)

				newInput.JoyP1.Up = window.CodeIsDown(glimmer.CodeUpArrow)
				newInput.JoyP1.Down = window.CodeIsDown(glimmer.CodeDownArrow)
				newInput.JoyP1.Left = window.CodeIsDown(glimmer.CodeLeftArrow)
				newInput.JoyP1.Right = window.CodeIsDown(glimmer.CodeRightArrow)
				newInput.JoyP1.Button = window.CodeIsDown(glimmer.CodeSpacebar)
			}
			window.Mutex.Unlock()

			lastInputPollTime = time.Now()

			emu.SetInput(newInput)

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

			if numDown > '0' && numDown <= '9' {
				snapFilename := snapshotPrefix + string(numDown)
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

			frameCount++
			if frameCount&0xff == 0 {
				if emu.InDevMode() {
					fmt.Printf("maxRTime %.4f, maxFTime %.4f\n", maxRDiff.Seconds(), maxFDiff)
				}
				maxRDiff = 0
				maxFDiff = 0
			}

			if frameCount&0x1f == 0 {
				//fmt.Println("cmd-paddlePos", paddle0Position)
			}

			rDiff := time.Now().Sub(lastFlipTime)
			const accuracyProtection = 2 * time.Millisecond
			ftGoalAsDuration := time.Duration(frametimeGoal*1000) * time.Millisecond
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
