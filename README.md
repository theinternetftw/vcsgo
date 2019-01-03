# vcsgo - an atari 2600 emulator in go

My other emulators:
[dmgo](https://github.com/theinternetftw/dmgo),
[famigo](https://github.com/theinternetftw/famigo),
[segmago](https://github.com/theinternetftw/segmago), and
[a1go](https://github.com/theinternetftw/a1go).

#### Features:
 * Audio!
 * Quicksave/Quickload!
 * Glitches are moderately rare!
 * Graphical and auditory cross-platform support!

That last bit relies on [glimmer](https://github.com/theinternetftw/glimmer). Let me know if it fails on your platform.

Tested on windows 10 and ubuntu 18.10.

#### Dependencies:

You can compile on windows with no C dependencies.

Linux users should 'apt install libasound2-dev' or equivalent.

FreeBSD (and Mac?) users should 'pkg install openal-soft' or equivalent.

#### Compile instructions

If you have go version >= 1.11, `go build ./cmd/vcsgo` should be enough. The interested can also see my build script `b` for profiling and such.

Non-windows users will need the dependencies listed above.

#### Important Notes:

 * First player keybindings are WSAD / J / F1 / F2 (arrowpad/paddle, fire, reset switch, select switch)
 * Second player Keybindings are UpDownLeftRight / Space
 * Keypad1 is 123/QWE/ASD/ZXC
 * Keypad2 is 456/RTY/FGH/VBN
 * Quicksave/Quickload is done by pressing m or l (make or load quicksave), followed by a number key
