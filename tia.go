package vcsgo

import "fmt"

type tia struct {
	Screen [320 * 264 * 4]byte

	Palette [128][3]byte

	TVFormat  TVFormat
	FormatSet bool

	FrameCount int

	PALFrameCountStart  int
	NTSCFrameCountStart int

	ScreenX int
	ScreenY int

	Collisions collisions

	P0, P1 sprite
	M0, M1 sprite
	BL     sprite

	HideM0 bool
	HideM1 bool

	HMoveRequested   bool
	HMoveCombEnabled bool

	// loaded in such that a screen half is Bits 19-0
	Playfield               uint32
	PlayfieldToLoad         uint32
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

	ShowDebugPuck bool
}

type sprite struct {
	X  byte
	Vx int8

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

func bitsToStr(val byte) string {
	return fmt.Sprintf("%v%v%v%v%v%v%v%v",
		val>>7&1, val>>6&1, val>>5&1, val>>4&1,
		val>>3&1, val>>2&1, val>>1&1, val>>0&1)
}

func (tia *tia) loadShapeP0(val byte) {
	tia.P0.Shape = val
	tia.P1.LatchedShape = tia.P1.Shape
	//fmt.Printf("P0Load!: P0LatchedShape:%v P0Shape:%v ", bitsToStr(tia.P0.LatchedShape), bitsToStr(tia.P0.Shape))
	//fmt.Printf("P1LatchedShape:%v P1Shape:%v\n", bitsToStr(tia.P1.LatchedShape), bitsToStr(tia.P1.Shape))
}
func (tia *tia) loadShapeP1(val byte) {
	tia.P1.Shape = val
	tia.P0.LatchedShape = tia.P0.Shape
	tia.BL.LatchedShow = tia.BL.Show
	//fmt.Printf("P1Load!: P0LatchedShape:%v P0Shape:%v ", bitsToStr(tia.P0.LatchedShape), bitsToStr(tia.P0.Shape))
	//fmt.Printf("P1LatchedShape:%v P1Shape:%v\n", bitsToStr(tia.P1.LatchedShape), bitsToStr(tia.P1.Shape))
}
func (tia *tia) loadEnablBL(val bool) {
	tia.BL.Show = val
}

func (s *sprite) move(vx int8) {
	x := int(s.X) - int(vx)
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
		player.X = 3
	} else {
		player.X = byte(tia.ScreenX+5) % 160
	}
}
func (tia *tia) resetMissile(missile *sprite) {
	if tia.InHBlank {
		missile.X = 2
	} else {
		missile.X = byte(tia.ScreenX+4) % 160
	}
}
func (tia *tia) resetBall(ball *sprite) {
	if tia.InHBlank {
		ball.X = 2
	} else {
		ball.X = byte(tia.ScreenX+4) % 160
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
	tia.HMoveRequested = true
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

var playerModeTable = [8][9]byte{
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

	shape := player.Shape
	if delay {
		shape = player.LatchedShape
	}

	if shape == 0 {
		return false
	}

	pX := tia.ScreenX - int(player.X)
	if pX < 0 || pX >= 9*8 {
		// NOTE: wrapping may not work this way? Might just be
		// the single copy of a player that's crossing the 160
		// border, not all copies...
		pXWrap := tia.ScreenX - (int(player.X) - 160)
		if pXWrap >= 0 && pXWrap < 9*8 {
			pX = pXWrap
		} else {
			return false
		}
	}

	col := pX >> 3
	if playerModeTable[player.RepeatMode][col] == 0 {
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
		return (shape>>shapeX)&1 > 0
	}
	return shape<<shapeX >= 0x80
}

// same as player, save no double or quad mode
var missileModeTable = [8][9]byte{
	{1, 0, 0, 0, 0, 0, 0, 0, 0},
	{1, 0, 1, 0, 0, 0, 0, 0, 0},
	{1, 0, 0, 0, 1, 0, 0, 0, 0},
	{1, 0, 1, 0, 1, 0, 0, 0, 0},
	{1, 0, 0, 0, 0, 0, 0, 0, 1},
	{1, 0, 0, 0, 0, 0, 0, 0, 0},
	{1, 0, 0, 0, 1, 0, 0, 0, 1},
	{1, 0, 0, 0, 0, 0, 0, 0, 0},
}

func (tia *tia) getMissileBit(missile *sprite) bool {

	mX := tia.ScreenX - int(missile.X)
	if mX < 0 || mX >= 9*8 {
		return false
	}

	col := mX >> 3
	if missileModeTable[missile.RepeatMode][col] == 0 {
		return false
	}

	shapeX := mX
	shapeX &= 7
	return shapeX <= int(missile.Size)-1
}

func (tia *tia) drawRGB(x, y int, r, g, b byte) {

	pix := (y*320 + 2*x) * 4

	tia.Screen[pix] = r
	tia.Screen[pix+1] = g
	tia.Screen[pix+2] = b

	pix2 := pix + 4

	tia.Screen[pix2] = r
	tia.Screen[pix2+1] = g
	tia.Screen[pix2+2] = b

	if tia.ShowDebugPuck && x+2 < 160 {
		pix3 := pix + 8
		tia.Screen[pix3] = 0xff
		tia.Screen[pix3+1] = 0xff
		tia.Screen[pix3+2] = 0xff
	}
}

func (tia *tia) drawColor(colorLuma byte) {
	col := tia.Palette[colorLuma>>1]
	tia.drawRGB(tia.ScreenX, tia.ScreenY, col[0], col[1], col[2])
}

func (s *sprite) lockMissileToPlayer(player *sprite) {
	s.X = player.X
	if player.RepeatMode == 5 {
		s.X += 8
	} else if player.RepeatMode == 7 {
		s.X += 16
	} else {
		s.X += 4
	}
	// TODO: should it wrap?
	s.X %= 160
}

func (tia *tia) setTVFormat(format TVFormat) {
	tia.TVFormat = format
	tia.FormatSet = true
	if format == FormatPAL {
		tia.Palette = palPalette
	} else {
		tia.Palette = ntscPalette
	}
}
func (tia *tia) init(emu *emuState) {
	for i := 0; i < len(tia.Screen); i += 4 {
		tia.Screen[i] = 0xff
	}
}

func (tia *tia) startVSync() {
	tia.WasInVSync = true
	tia.flipRequested = true
	tia.FrameCount++

	if !tia.FormatSet {
		if tia.ScreenY >= 263 {
			if tia.PALFrameCountStart == 0 {
				tia.PALFrameCountStart = tia.FrameCount
			} else if tia.FrameCount-tia.PALFrameCountStart >= 20 {
				fmt.Println("PAL!")
				tia.setTVFormat(FormatPAL)
			}
		} else {
			if tia.NTSCFrameCountStart == 0 {
				tia.NTSCFrameCountStart = tia.FrameCount
			} else if tia.FrameCount-tia.NTSCFrameCountStart >= 20 {
				fmt.Println("NTSC!")
				tia.setTVFormat(FormatNTSC)
			}
		}
	}
}

func (tia *tia) hmove() {
	tia.HMoveRequested = false
	if tia.InHBlank {
		// tia.HMoveCombEnabled = true
		tia.P0.move(tia.P0.Vx)
		tia.P1.move(tia.P1.Vx)
		tia.M0.move(tia.M0.Vx)
		tia.M1.move(tia.M1.Vx)
		tia.BL.move(tia.BL.Vx)
	} else {
		// FIXME: right now I'm just assuming that anything
		// that's HMOVEing early is trying the no-comb trick,
		// which is not necessarily true...
		tia.P0.move(tia.P0.Vx + 8)
		tia.P1.move(tia.P1.Vx + 8)
		tia.M0.move(tia.M0.Vx + 8)
		tia.M1.move(tia.M1.Vx + 8)
		tia.BL.move(tia.BL.Vx + 8)
	}
}

func (tia *tia) computeColorAndUpdateCollision() byte {

	blShow := tia.BL.Show
	if tia.DelayGRBL {
		blShow = tia.BL.LatchedShow
	}

	playfieldBit := tia.getPlayfieldBit()
	ballBit := tia.getBallBit() && blShow
	p0Bit := tia.getPlayerBit(&tia.P0, tia.DelayGRP0)
	p1Bit := tia.getPlayerBit(&tia.P1, tia.DelayGRP1)
	m0Bit := tia.getMissileBit(&tia.M0) && tia.M0.Show && !tia.HideM0
	m1Bit := tia.getMissileBit(&tia.M1) && tia.M1.Show && !tia.HideM1

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

	drawPFBL := playfieldBit || ballBit
	drawP0M0 := p0Bit || m0Bit
	drawP1M1 := p1Bit || m1Bit

	var colorLuma byte
	if tia.PFAndBLHavePriority {
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

	return colorLuma
}

func (tia *tia) runThreeCycles() {

	if tia.HideM0 {
		tia.M0.lockMissileToPlayer(&tia.P0)
	}
	if tia.HideM1 {
		tia.M1.lockMissileToPlayer(&tia.P1)
	}
	if tia.HMoveRequested {
		tia.hmove()
	}

	if !tia.WasInVSync && tia.InVSync {
		tia.startVSync()
	} else if tia.WasInVSync && !tia.InVSync {
		tia.WasInVSync = false
		// NOTE: Found PAL roms that expect less than 45 lines
		// of upper border, so leaving this as is for now.
		tia.ScreenY = -37
	}

	/*
		if !tia.WasInVBlank && tia.InVBlank {
			tia.WasInVBlank = true
			fmt.Printf("Enter VBlank at %3d", tia.ScreenY)
		} else if tia.WasInVBlank && !tia.InVBlank {
			tia.WasInVBlank = false
			fmt.Println(" / Exit VBlank at", tia.ScreenY)
		}
	*/

	for i := 0; i < 3; i++ {

		if tia.ScreenX >= 0 && tia.ScreenX < 160 {

			if tia.ScreenX == 0 {
				tia.InHBlank = false
			} else if tia.ScreenX == 8 {
				tia.HMoveCombEnabled = false
			}

			colorLuma := byte(0)
			if !tia.InVBlank && !tia.HMoveCombEnabled {
				colorLuma = tia.computeColorAndUpdateCollision()
			}

			if tia.ScreenY >= 0 && tia.ScreenY < 264 {
				tia.drawColor(colorLuma)
			}

		} else if tia.ScreenX == 160 {
			tia.ScreenX = -68
			tia.WaitForHBlank = false
			tia.InHBlank = true
			if tia.ScreenY++; tia.ScreenY > 275 {
				// if a program doesn't vsync, lets just hang
				// out here at the end of the screen...
				tia.ScreenY = 275
			}
		}

		// NOTE: Load every four pixels is correct, but don't be surprised
		// if what offset we do it at changes when other things are fixed...
		if tia.ScreenX&3 == 3 {
			tia.Playfield = tia.PlayfieldToLoad
		}

		tia.ScreenX++
	}
}
