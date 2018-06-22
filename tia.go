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
	X              byte
	Vx             int8
	ResetRequested bool
	ResetX         byte

	ColorLuma byte

	// only for P0/P1
	LatchedShape byte
	Shape        byte
	Reflect      bool

	// only for P0/P1/M0/M1
	RepeatMode byte

	// only for BL/M0/M1
	Size byte
	Show bool

	// only for BL
	LatchedShow bool
}

func (tia *tia) loadShapeP0(val byte) {
	tia.P0.Shape = val
	tia.P1.LatchedShape = tia.P1.Shape
}
func (tia *tia) loadShapeP1(val byte) {
	tia.P1.Shape = val
	tia.P0.LatchedShape = tia.P0.Shape
	tia.BL.LatchedShow = tia.BL.Show
}
func (tia *tia) loadEnablBL(val bool) {
	tia.BL.Show = val
}

func (s *sprite) move() {
	x := int(s.X) - int(s.Vx)
	for x < 0 {
		x += 160
	}
	for x >= 160 {
		x -= 160
	}
	s.X = byte(x)
}

func (tia *tia) resetP0() { tia.resetPlayer(&tia.P0) }
func (tia *tia) resetP1() { tia.resetPlayer(&tia.P1) }
func (tia *tia) resetM0() { tia.resetMissile(&tia.M0) }
func (tia *tia) resetM1() { tia.resetMissile(&tia.M1) }
func (tia *tia) resetBL() { tia.resetBall(&tia.BL) }

func (tia *tia) resetPlayer(player *sprite) {
	if tia.InHBlank {
		player.X = 4
	} else {
		player.ResetRequested = true
		player.ResetX = byte(tia.ScreenX+9) % 160
	}
}
func (tia *tia) resetMissile(missile *sprite) {
	if tia.InHBlank {
		missile.X = 3
	} else {
		missile.ResetRequested = true
		missile.ResetX = byte(tia.ScreenX+9) % 160
	}
}
func (tia *tia) resetBall(ball *sprite) {
	if tia.InHBlank {
		ball.X = 3
	} else {
		ball.X = byte(tia.ScreenX+9) % 160
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

func (tia *tia) getPlayerBit(player *sprite, delay bool) bool {

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

	shape := player.Shape
	if delay {
		shape = player.LatchedShape
	}

	if player.Reflect {
		return (shape>>shapeX)&1 == 1
	}
	return (shape<<shapeX)&0x80 == 0x80
}

func (tia *tia) getMissileBit(missile, player *sprite) bool {

	row := repeatModeTable[player.RepeatMode]
	mX := tia.ScreenX - int(missile.X)
	if mX < 0 || mX >= 9*8 {
		return false
	}

	col := mX >> 3
	if row[col] == 0 {
		return false
	}

	shapeX := mX
	shapeX &= 7
	return shapeX <= int(missile.Size)-1
}

func (tia *tia) drawColor(colorLuma byte) {
	x, y := int(tia.ScreenX), tia.ScreenY
	col := ntscPalette[colorLuma>>1]
	tia.Screen[y*160*4+x*4] = col[0]
	tia.Screen[y*160*4+x*4+1] = col[1]
	tia.Screen[y*160*4+x*4+2] = col[2]
	tia.Screen[y*160*4+x*4+3] = 0xff
}

func (s *sprite) lockMissileToPlayer(player *sprite) {
	s.X = player.X
	if player.RepeatMode == 5 {
		s.X += 3
	} else if player.RepeatMode == 7 {
		s.X += 5
	} else {
		s.X += 10
	}
	// TODO: should it wrap?
}

func (s *sprite) updateFromReset() {
	if s.ResetRequested {
		s.ResetRequested = false
		s.X = s.ResetX
	}
}

func (tia *tia) updatePositionsFromReset() {
	tia.P0.updateFromReset()
	tia.P1.updateFromReset()
	tia.M0.updateFromReset()
	tia.M1.updateFromReset()
}

func (tia *tia) runCycle() {

	if !tia.WasInVSync && tia.InVSync {
		tia.WasInVSync = true
		tia.flipRequested = true
	} else if tia.WasInVSync && !tia.InVSync {
		tia.WasInVSync = false
		// upper border
		// tia.ScreenY = -37
		tia.ScreenY = -37
	}

	if tia.HideM0 {
		tia.M0.lockMissileToPlayer(&tia.P0)
	}
	if tia.HideM1 {
		tia.M1.lockMissileToPlayer(&tia.P1)
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
		tia.updatePositionsFromReset()
	} else if tia.ScreenX == 0 {
		tia.InHBlank = false
	}

	if tia.ScreenY >= 0 && tia.ScreenY < 192 {
		if tia.ScreenX >= 0 && tia.ScreenX < 160 {

			playfieldBit := tia.getPlayfieldBit()
			ballBit := tia.getBallBit()
			p0Bit := tia.getPlayerBit(&tia.P0, tia.DelayGRP0)
			p1Bit := tia.getPlayerBit(&tia.P1, tia.DelayGRP1)
			m0Bit := tia.getMissileBit(&tia.M0, &tia.P0)
			m1Bit := tia.getMissileBit(&tia.M1, &tia.P1)

			updateCollision := func(collision *bool, test bool) {
				if test {
					*collision = true
				}
			}

			updateCollision(&tia.Collisions.BLPF, ballBit && playfieldBit)

			updateCollision(&tia.Collisions.P0PF, p0Bit && playfieldBit)
			updateCollision(&tia.Collisions.P0BL, p0Bit && ballBit)

			updateCollision(&tia.Collisions.M0P0, m0Bit && p0Bit)
			updateCollision(&tia.Collisions.M0P1, m0Bit && p1Bit)
			updateCollision(&tia.Collisions.M0PF, m0Bit && playfieldBit)
			updateCollision(&tia.Collisions.M0BL, m0Bit && ballBit)

			updateCollision(&tia.Collisions.P1PF, p1Bit && playfieldBit)
			updateCollision(&tia.Collisions.P1BL, p1Bit && ballBit)

			updateCollision(&tia.Collisions.M1P0, m1Bit && p0Bit)
			updateCollision(&tia.Collisions.M1P1, m1Bit && p1Bit)
			updateCollision(&tia.Collisions.M1PF, m1Bit && playfieldBit)
			updateCollision(&tia.Collisions.M1BL, m1Bit && ballBit)

			updateCollision(&tia.Collisions.P0P1, p0Bit && p1Bit)
			updateCollision(&tia.Collisions.M0M1, m0Bit && m1Bit)

			blShow := tia.BL.Show
			if tia.DelayGRBL {
				blShow = tia.BL.LatchedShow
			}

			drawPFBL := playfieldBit || (ballBit && blShow)
			drawP0M0 := p0Bit || (m0Bit && tia.M0.Show && !tia.HideM0)
			drawP1M1 := p1Bit || (m1Bit && tia.M1.Show && !tia.HideM1)

			var colorLuma byte
			if tia.InVBlank {
				colorLuma = 0
			} else if tia.PFAndBLHavePriority {
				if drawPFBL {
					colorLuma = tia.PlayfieldAndBallColorLuma
				} else if drawP0M0 {
					colorLuma = tia.P0.ColorLuma
				} else if drawP1M1 {
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
				} else if drawP0M0 {
					colorLuma = tia.P0.ColorLuma
				} else if drawP1M1 {
					colorLuma = tia.P1.ColorLuma
				} else if drawPFBL {
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
