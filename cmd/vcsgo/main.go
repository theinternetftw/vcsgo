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

	var emu vcsgo.Emulator

	emu = vcsgo.NewEmulator(cartBytes)

	screenW := 320
	screenH := 240
	platform.InitDisplayLoop("vcsgo", screenW*2+40, screenH*2+40, screenW, screenH, func(sharedState *platform.WindowState) {
		startEmu(cartFilename, sharedState, emu)
	})
}

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

	paddle0Position := float32(0)
	paddleVel := float32(45) //degrees a second
	clamp := func(min, x, max float32) float32 {
		if x < min {
			return min
		} else if x > max {
			return max
		}
		return x
	}

	for {
		newInput := vcsgo.Input {}
		snapshotMode := 'x'
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

				newInput.Paddle0.Button = newInput.JoyP0.Button
				if newInput.JoyP0.Left {
					paddle0Position -= paddleVel*inputDt
				} else if newInput.JoyP0.Right {
					paddle0Position += paddleVel*inputDt
				}
				paddle0Position = clamp(-135, paddle0Position, 135)
				newInput.Paddle0.Position = int16(paddle0Position)

				newInput.JoyP1.Up = window.CodeIsDown(key.CodeUpArrow)
				newInput.JoyP1.Down = window.CodeIsDown(key.CodeDownArrow)
				newInput.JoyP1.Left = window.CodeIsDown(key.CodeLeftArrow)
				newInput.JoyP1.Right = window.CodeIsDown(key.CodeRightArrow)
				newInput.JoyP1.Button = window.CodeIsDown(key.CodeSpacebar)
				/*
				for r := '0'; r <= '9'; r++ {
					if window.CharIsDown(r) {
						numDown = r
						break
					}
				}
				if window.CharIsDown('m') {
					snapshotMode = 'm'
				} else if window.CharIsDown('l') {
					snapshotMode = 'l'
				}
				*/
			}
			window.Mutex.Unlock()

			if numDown > '0' && numDown <= '9' {
				snapFilename := snapshotPrefix+string(numDown)
				if snapshotMode == 'm' {
					snapshotMode = 'x'
					snapshot := emu.MakeSnapshot()
					if len(snapshot) > 0 {
						ioutil.WriteFile(snapFilename, snapshot, os.FileMode(0644))
					}
				} else if snapshotMode == 'l' {
					snapshotMode = 'x'
					snapBytes, err := ioutil.ReadFile(snapFilename)
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

			emu.UpdateInput(newInput)

			lastInputUpdateTime = time.Now()
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
			if frameCount & 0x3f == 0 {
				//fmt.Printf("maxRTime %.4f, maxFTime %.4f\n", maxRDiff.Seconds(), maxFDiff)
				maxRDiff = 0
				maxFDiff = 0
			}

			if frameCount & 0x1f == 0 {
				//fmt.Println("cmd-paddlePos", paddle0Position)
			}

			rDiff := time.Now().Sub(lastFlipTime)
			toSleep := 15*time.Millisecond - rDiff
			if toSleep > 2*time.Millisecond {
				timer.Reset(toSleep)
				<-timer.C
			}

			fDiff := 0.0
			for fDiff < 0.0161 { // seems to be about 0.005 resolution? so leave a bit of play
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
