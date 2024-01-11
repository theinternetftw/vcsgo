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
	glimmer.InitDisplayLoop(glimmer.InitDisplayLoopOptions{
		WindowTitle:  "vcsgo",
		WindowWidth:  screenW * 2,
		WindowHeight: screenH * 2,
		RenderWidth:  screenW,
		RenderHeight: screenH,
		InitCallback: func(sharedState *glimmer.WindowState) {
			startEmu(cartFilename, sharedState, emu)
		},
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

	lastInputPollTime := time.Now()

	snapshotPrefix := filename + ".snapshot"

	audio, audioErr := glimmer.OpenAudioBuffer(glimmer.OpenAudioBufferOptions{
		OutputBufDuration: 25 * time.Millisecond,
		SamplesPerSecond:  44100,
		BitsPerSample:     16,
		ChannelCount:      2,
	})
	dieIf(audioErr)
	workingAudioBuffer := make([]byte, audio.GetPrevCallbackReadLen())
	audioToGen := audio.GetPrevCallbackReadLen()

	paddles := []paddlePhys{
		paddlePhys{}, paddlePhys{},
	}

	frameTimer := glimmer.MakeFrameTimer()

	snapshotMode := 'x'

	newInput := vcsgo.Input{}

	for {

		now := time.Now()

		inputDiff := now.Sub(lastInputPollTime)
		if inputDiff > 8*time.Millisecond {

			numDown := 'x'

			inputDt := float32(inputDiff.Seconds())
			newInput = vcsgo.Input{}

			window.InputMutex.Lock()
			{
				window.CopyKeyCharArray(newInput.Keys[:])

				cid := func(c glimmer.KeyCode) bool { return window.CodeIsDown(c) }

				newInput.ResetButton = cid(glimmer.KeyCodeF1)
				newInput.SelectButton = cid(glimmer.KeyCodeF2)

				newInput.JoyP0.Up = cid(glimmer.KeyCodeW)
				newInput.JoyP0.Down = cid(glimmer.KeyCodeS)
				newInput.JoyP0.Left = cid(glimmer.KeyCodeA)
				newInput.JoyP0.Right = cid(glimmer.KeyCodeD)
				newInput.JoyP0.Button = cid(glimmer.KeyCodeJ)

				// TODO: switch between input methods (arg switch, plus
				//  about 20 MD5s for the games that use paddles)
				if cid(glimmer.KeyCodeA) {
					paddles[0].left(inputDt)
				} else if cid(glimmer.KeyCodeD) {
					paddles[0].right(inputDt)
				} else {
					paddles[0].noMove(inputDt)
				}
				newInput.Paddle0.Button = window.CodeIsDown(glimmer.KeyCodeJ)
				newInput.Paddle0.Position = int16(paddles[0].pos)

				newInput.Keypad0 = [12]bool{
					cid(glimmer.KeyCodeDigit1), cid(glimmer.KeyCodeDigit2), cid(glimmer.KeyCodeDigit3),
					cid(glimmer.KeyCodeQ), cid(glimmer.KeyCodeW), cid(glimmer.KeyCodeE),
					cid(glimmer.KeyCodeA), cid(glimmer.KeyCodeS), cid(glimmer.KeyCodeD),
					cid(glimmer.KeyCodeZ), cid(glimmer.KeyCodeX), cid(glimmer.KeyCodeC),
				}

				newInput.Keypad1 = [12]bool{
					cid(glimmer.KeyCodeDigit4), cid(glimmer.KeyCodeDigit5), cid(glimmer.KeyCodeDigit6),
					cid(glimmer.KeyCodeR), cid(glimmer.KeyCodeT), cid(glimmer.KeyCodeY),
					cid(glimmer.KeyCodeF), cid(glimmer.KeyCodeG), cid(glimmer.KeyCodeH),
					cid(glimmer.KeyCodeV), cid(glimmer.KeyCodeB), cid(glimmer.KeyCodeN),
				}

				if window.CodeIsDown(glimmer.KeyCodeArrowLeft) {
					paddles[1].left(inputDt)
				} else if window.CodeIsDown(glimmer.KeyCodeArrowRight) {
					paddles[1].right(inputDt)
				} else {
					paddles[1].noMove(inputDt)
				}
				newInput.Paddle1.Button = window.CodeIsDown(glimmer.KeyCodeSpace)
				newInput.Paddle1.Position = int16(paddles[1].pos)

				newInput.JoyP1.Up = window.CodeIsDown(glimmer.KeyCodeArrowUp)
				newInput.JoyP1.Down = window.CodeIsDown(glimmer.KeyCodeArrowDown)
				newInput.JoyP1.Left = window.CodeIsDown(glimmer.KeyCodeArrowLeft)
				newInput.JoyP1.Right = window.CodeIsDown(glimmer.KeyCodeArrowRight)
				newInput.JoyP1.Button = window.CodeIsDown(glimmer.KeyCodeSpace)
			}
			window.InputMutex.Unlock()

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

		if emu.GetSoundBufferUsed() >= audioToGen {
			if cap(workingAudioBuffer) < audioToGen {
				workingAudioBuffer = make([]byte, audioToGen)
			}
			workingAudioBuffer = workingAudioBuffer[:audioToGen]
			audio.Write(emu.ReadSoundBuffer(workingAudioBuffer))
		}

		if emu.FlipRequested() {
			window.RenderMutex.Lock()
			copy(window.Pix, emu.Framebuffer())
			window.RenderMutex.Unlock()

			frameTimer.MarkRenderComplete()

			audio.WaitForPlaybackIfAhead()

			frameTimer.MarkFrameComplete()

			if emu.InDevMode() {
				frameTimer.PrintStatsEveryXFrames(60 * 5)
			}
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
