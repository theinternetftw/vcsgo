package main

import (
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

	emu := vcsgo.NewEmulator(cartBytes)
	emu.SetDebugContinue(true)

	testSpeed(emu)
}

func testSpeed(emu vcsgo.Emulator) {

	nullInput := vcsgo.Input{}
	lastPrint := time.Now()
	steps := 0
	for {
		emu.SetInput(nullInput)
		emu.Step()
		emu.FlipRequested()
		steps++

		if time.Now().Sub(lastPrint) >= time.Second {
			fmt.Printf("ips: %9d\n", steps)
			lastPrint = time.Now()
			steps = 0
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

