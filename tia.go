package vcsgo

type tia struct {
	Screen [160 * 192 * 4]byte

	ScreenX int
	ScreenY int

	Collisions collisions

	P0, P1 sprite
	M0, M1 sprite
	BL     sprite

	HideM0 bool
	HideM1 bool

	// loaded in such that a screen half is Bits 19-0
	Playfield               uint32
	PFAndBLHavePriority     bool
	PlayfieldScoreColorMode bool
	PlayfieldReflect        bool

	PlayfieldAndBallColorLuma byte

	BGColorLuma byte

	DelayGRP0 bool // until P1
	DelayGRP1 bool // until P0
	DelayGRBL bool // until P1

	flipRequested bool

	InHBlank      bool
	WaitForHBlank bool

	InVBlank    bool
	InVSync     bool
	WasInVSync  bool
	WasInVBlank bool
}

type sprite struct {
	X  byte
	Vx int8

	ColorLuma byte

	// only for P0/P1
	Shape   byte
	Reflect bool

	// only for P0/P1/M0/M1
	RepeatMode byte

	// only for BL/M0/M1
	Size byte
	Show bool
}

func (s *sprite) move() {
	x := int(s.X) - int(s.Vx)
	if x < 0 {
		x += 160
	} else if x > 160 {
		x -= 160
	}
	s.X = byte(x)
}

func (tia *tia) resetP0() { tia.resetPlayer(&tia.P0) }
func (tia *tia) resetP1() { tia.resetPlayer(&tia.P1) }
func (tia *tia) resetM0() { tia.resetObject(&tia.M0) }
func (tia *tia) resetM1() { tia.resetObject(&tia.M1) }
func (tia *tia) resetBL() { tia.resetObject(&tia.BL) }

func (tia *tia) resetPlayer(player *sprite) {
	if tia.InHBlank {
		player.X = 3
	} else {
		player.X = byte(tia.ScreenX)
	}
}
func (tia *tia) resetObject(obj *sprite) {
	if tia.InHBlank {
		obj.X = 2
	} else {
		obj.X = byte(tia.ScreenX)
	}
}

type collisions struct {
	M0P1, M0P0 bool
	M1P0, M1P1 bool

	P0PF, P0BL bool
	P1PF, P1BL bool
	M0PF, M0BL bool
	M1PF, M1BL bool

	BLPF bool

	P0P1, M0M1 bool
}

func (tia *tia) clearCollisions() {
	tia.Collisions = collisions{}
}

func (tia *tia) applyHorizMotion() {
	tia.P0.move()
	tia.P1.move()
	tia.M0.move()
	tia.M1.move()
	tia.BL.move()
}

func (tia *tia) clearHorizMotion() {
	tia.P0.Vx = 0
	tia.P1.Vx = 0
	tia.M0.Vx = 0
	tia.M1.Vx = 0
	tia.BL.Vx = 0
}

func (tia *tia) resetHorizCounter() {
	// TODO: see if this is correct behavior
	tia.ScreenX = -68
	tia.InHBlank = true
}

func (tia *tia) getPlayfieldBit() bool {
	pfX := tia.ScreenX >> 2
	if pfX < 0 || pfX >= 40 {
		return false
	}
	if pfX >= 20 {
		pfX -= 20
		if tia.PlayfieldReflect {
			pfX = 19 - pfX
		}
	}
	return (tia.Playfield & (1 << byte(19-pfX))) != 0
}

func (tia *tia) getBallBit() bool {
	x := tia.ScreenX
	blX := int(tia.BL.X)
	blSz := int(tia.BL.Size)
	return x >= blX && x < blX+blSz
}

var repeatModeTable = [8][9]byte{
	{1, 0, 0, 0, 0, 0, 0, 0, 0},
	{1, 0, 1, 0, 0, 0, 0, 0, 0},
	{1, 0, 0, 0, 1, 0, 0, 0, 0},
	{1, 0, 1, 0, 1, 0, 0, 0, 0},
	{1, 0, 0, 0, 0, 0, 0, 0, 1},
	{1, 1, 0, 0, 0, 0, 0, 0, 0},
	{1, 0, 0, 0, 1, 0, 0, 0, 1},
	{1, 1, 1, 1, 0, 0, 0, 0, 0},
}

func (tia *tia) getPlayerBit(player *sprite) bool {

	row := repeatModeTable[player.RepeatMode]
	pX := tia.ScreenX - int(player.X)
	if pX < 0 || pX >= 9*8 {
		return false
	}

	col := pX >> 3
	if row[col] == 0 {
		return false
	}

	shapeX := uint(pX)
	if player.RepeatMode == 5 {
		shapeX >>= 1 // double width
	} else if player.RepeatMode == 7 {
		shapeX >>= 2 // quad width
	}
	shapeX &= 7

	if player.Reflect {
		return (player.Shape>>shapeX)&1 == 1
	}
	return (player.Shape<<shapeX)&0x80 == 0x80
}

func (tia *tia) drawColor(colorLuma byte) {
	x, y := int(tia.ScreenX), tia.ScreenY
	col := ntscPalette[colorLuma>>1]
	tia.Screen[y*160*4+x*4] = col[0]
	tia.Screen[y*160*4+x*4+1] = col[1]
	tia.Screen[y*160*4+x*4+2] = col[2]
	tia.Screen[y*160*4+x*4+3] = 0xff
}

func (tia *tia) runCycle() {

	if !tia.WasInVSync && tia.InVSync {
		tia.WasInVSync = true
		tia.flipRequested = true
	} else if tia.WasInVSync && !tia.InVSync {
		tia.WasInVSync = false
		// upper border
		tia.ScreenY = -37
		tia.ScreenX = -68
	}

	if !tia.WasInVBlank && tia.InVBlank {
		tia.WasInVBlank = true
		//fmt.Printf("Enter VBlank at %3d", tia.ScreenY)
	} else if tia.WasInVBlank && !tia.InVBlank {
		tia.WasInVBlank = false
		//fmt.Println(" / Exit VBlank at", tia.ScreenY)
	}

	if tia.ScreenX == 160 {
		tia.ScreenX = -68
		tia.WaitForHBlank = false
		tia.InHBlank = true
		if tia.ScreenY++; tia.ScreenY > 221 {
			// if a program doesn't vsync, lets just hang
			// out here at the end of the screen...
			tia.ScreenY = 221
		}
	} else if tia.ScreenX == 0 {
		tia.InHBlank = false
	}

	if tia.ScreenY >= 0 && tia.ScreenY < 192 {
		if tia.ScreenX >= 0 && tia.ScreenX < 160 {

			playfieldBit := tia.getPlayfieldBit()
			ballBit := tia.getBallBit()
			//m0Bit := tia.getM0Bit()
			//m1Bit := tia.getM1Bit()
			p0Bit := tia.getPlayerBit(&tia.P0)
			p1Bit := tia.getPlayerBit(&tia.P1)

			updateCollision := func(collision *bool, test bool) {
				if test {
					*collision = true
				}
			}

			updateCollision(&tia.Collisions.BLPF, ballBit && playfieldBit)

			updateCollision(&tia.Collisions.P0PF, p0Bit && playfieldBit)
			updateCollision(&tia.Collisions.P0BL, p0Bit && ballBit)
			updateCollision(&tia.Collisions.P1PF, p1Bit && playfieldBit)
			updateCollision(&tia.Collisions.P1BL, p1Bit && ballBit)

			updateCollision(&tia.Collisions.P0P1, p0Bit && p1Bit)

			var colorLuma byte
			if tia.InVBlank {
				colorLuma = 0
			} else if tia.PFAndBLHavePriority {
				if playfieldBit || (ballBit && tia.BL.Show) {
					colorLuma = tia.PlayfieldAndBallColorLuma
				} else if p0Bit {
					colorLuma = tia.P0.ColorLuma
				} else if p1Bit {
					colorLuma = tia.P1.ColorLuma
				} else {
					colorLuma = tia.BGColorLuma
				}
			} else {
				if tia.PlayfieldScoreColorMode && playfieldBit {
					if tia.ScreenX < 80 {
						colorLuma = tia.P0.ColorLuma
					} else {
						colorLuma = tia.P1.ColorLuma
					}
				} else if p0Bit {
					colorLuma = tia.P0.ColorLuma
				} else if p1Bit {
					colorLuma = tia.P1.ColorLuma
				} else if playfieldBit || (ballBit && tia.BL.Show) {
					colorLuma = tia.PlayfieldAndBallColorLuma
				} else {
					colorLuma = tia.BGColorLuma
				}
			}

			tia.drawColor(colorLuma)
		}
	}

	tia.ScreenX++
}
