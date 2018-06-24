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
)

func main() {

	defer profiling.Start().Stop()

	assert(len(os.Args) == 2, "usage: ./vcsgo ROM_FILENAME")
	cartFilename := os.Args[1]

	cartBytes, err := ioutil.ReadFile(cartFilename)
	dieIf(err)

	var emu vcsgo.Emulator

	emu = vcsgo.NewEmulator(cartBytes)

	screenW := 160
	screenH := 222
	platform.InitDisplayLoop("vcsgo", screenW*2+40, screenH*2+40, screenW, screenH, func(sharedState *platform.WindowState) {
		startEmu(cartFilename, sharedState, emu)
	})
}

func startEmu(filename string, window *platform.WindowState, emu vcsgo.Emulator) {

	// FIXME: settings are for debug right now
	lastFlipTime := time.Now()

	snapshotPrefix := filename + ".snapshot"

	audio, err := platform.OpenAudioBuffer(4, 4096, 44100, 16, 2)
	workingAudioBuffer := make([]byte, audio.BufferSize())
	dieIf(err)

	timer := time.NewTimer(0)
	<-timer.C

	for {
		newInput := vcsgo.Input {}
		snapshotMode := 'x'
		numDown := 'x'

		window.Mutex.Lock()
		{
			window.CopyKeyCharArray(newInput.Keys[:])
			if window.CodeIsDown(key.CodeF1) {
				newInput.ResetButton = true
			}
			if window.CodeIsDown(key.CodeF2) {
				newInput.SelectButton = true
			}
			newInput.JoyP0.Up = window.CodeIsDown(key.CodeUpArrow)
			newInput.JoyP0.Down = window.CodeIsDown(key.CodeDownArrow)
			newInput.JoyP0.Left = window.CodeIsDown(key.CodeLeftArrow)
			newInput.JoyP0.Right = window.CodeIsDown(key.CodeRightArrow)
			newInput.JoyP0.Button = window.CodeIsDown(key.CodeSpacebar)
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
		emu.Step()

		bufferAvailable := audio.BufferAvailable()

		audioBufSlice := workingAudioBuffer[:bufferAvailable]
		audio.Write(emu.ReadSoundBuffer(audioBufSlice))

		if emu.FlipRequested() {
			window.Mutex.Lock()
			copy(window.Pix, emu.Framebuffer())
			window.RequestDraw()
			window.Mutex.Unlock()
			now := time.Now()
			toSleep := 17*time.Millisecond - now.Sub(lastFlipTime)
			lastFlipTime = now
			time.Sleep(toSleep)
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
