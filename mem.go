package vcsgo

import "fmt"

type mem struct {
	RAM [0x80]byte // jesus christ

	rom    []byte
	mapper mapper

	lastWriteAddr uint16 // unfortunately necessary for a mapper hack
}

func (emu *emuState) read(addr uint16) byte {
	origAddr := addr

	if emu.Mem.mapper.getMapperNum() == 0 && len(emu.Mem.rom) > 4096 {
		emu.Mem.mapper = emu.guessMapperFromAddr(addr)
	}

	bitOn := func(bit byte) bool {
		return addr&(1<<bit) != 0
	}

	var val byte
	switch {
	case !bitOn(12) && !bitOn(7):
		// TIA
		maskedAddr := addr & 0x0f
		switch maskedAddr {
		case 0x00:
			val = byte(addr) &^ 0xc0
			val |= boolBit(7, emu.TIA.Collisions.M0P1)
			val |= boolBit(6, emu.TIA.Collisions.M0P0)
		case 0x01:
			val = byte(addr) &^ 0xc0
			val |= boolBit(7, emu.TIA.Collisions.M1P0)
			val |= boolBit(6, emu.TIA.Collisions.M1P1)
		case 0x02:
			val = byte(addr) &^ 0xc0
			val |= boolBit(7, emu.TIA.Collisions.P0PF)
			val |= boolBit(6, emu.TIA.Collisions.P0BL)
		case 0x03:
			val = byte(addr) &^ 0xc0
			val |= boolBit(7, emu.TIA.Collisions.P1PF)
			val |= boolBit(6, emu.TIA.Collisions.P1BL)
		case 0x04:
			val = byte(addr) &^ 0xc0
			val |= boolBit(7, emu.TIA.Collisions.M0PF)
			val |= boolBit(6, emu.TIA.Collisions.M0BL)
		case 0x05:
			val = byte(addr) &^ 0xc0
			val |= boolBit(7, emu.TIA.Collisions.M1PF)
			val |= boolBit(6, emu.TIA.Collisions.M1BL)
		case 0x06:
			val = byte(addr) &^ 0x80
			val |= boolBit(7, emu.TIA.Collisions.BLPF)
		case 0x07:
			val = byte(addr) &^ 0xc0
			val |= boolBit(7, emu.TIA.Collisions.P0P1)
			val |= boolBit(6, emu.TIA.Collisions.M0M1)

		case 0x08:
			val = boolBit(7, emu.Paddle0InputCharged)
			// TODO: could you sel, change DDR, and still check these bits?
			if emu.DDRModeMaskPortA == 0xff {
				val = boolBit(7,
					!((emu.RowSelKeypad0&1 > 0 && emu.Input.Keypad0[0]) ||
						(emu.RowSelKeypad0&2 > 0 && emu.Input.Keypad0[3]) ||
						(emu.RowSelKeypad0&4 > 0 && emu.Input.Keypad0[6]) ||
						(emu.RowSelKeypad0&8 > 0 && emu.Input.Keypad0[9])))
			} else if emu.InputTimingPots {
				emu.PaddleChecksThisFrame++
			}
		case 0x09:
			val = boolBit(7, emu.Paddle1InputCharged)
			if emu.DDRModeMaskPortA == 0xff {
				val = boolBit(7,
					!((emu.RowSelKeypad0&1 > 0 && emu.Input.Keypad0[1]) ||
						(emu.RowSelKeypad0&2 > 0 && emu.Input.Keypad0[4]) ||
						(emu.RowSelKeypad0&4 > 0 && emu.Input.Keypad0[7]) ||
						(emu.RowSelKeypad0&8 > 0 && emu.Input.Keypad0[10])))
			} else if emu.InputTimingPots {
				emu.PaddleChecksThisFrame++
			}
		case 0x0a:
			val = boolBit(7, emu.Paddle2InputCharged)
			if emu.DDRModeMaskPortA == 0xff {
				val = boolBit(7,
					!((emu.RowSelKeypad1&1 > 0 && emu.Input.Keypad1[0]) ||
						(emu.RowSelKeypad1&2 > 0 && emu.Input.Keypad1[3]) ||
						(emu.RowSelKeypad1&4 > 0 && emu.Input.Keypad1[6]) ||
						(emu.RowSelKeypad1&8 > 0 && emu.Input.Keypad1[9])))
			} else if emu.InputTimingPots {
				emu.PaddleChecksThisFrame++
			}
		case 0x0b:
			val = boolBit(7, emu.Paddle3InputCharged)
			if emu.DDRModeMaskPortA == 0xff {
				val = boolBit(7,
					!((emu.RowSelKeypad1&1 > 0 && emu.Input.Keypad1[1]) ||
						(emu.RowSelKeypad1&2 > 0 && emu.Input.Keypad1[4]) ||
						(emu.RowSelKeypad1&4 > 0 && emu.Input.Keypad1[7]) ||
						(emu.RowSelKeypad1&8 > 0 && emu.Input.Keypad1[10])))
			} else if emu.InputTimingPots {
				emu.PaddleChecksThisFrame++
			}

		case 0x0c:
			if emu.Input45LatchMode {
				val = boolBit(7, emu.Input4LatchVal)
			} else {
				val = boolBit(7, !emu.Input.JoyP0.Button)
			}
			if emu.DDRModeMaskPortA == 0xff {
				val = boolBit(7,
					!((emu.RowSelKeypad0&1 > 0 && emu.Input.Keypad0[2]) ||
						(emu.RowSelKeypad0&2 > 0 && emu.Input.Keypad0[5]) ||
						(emu.RowSelKeypad0&4 > 0 && emu.Input.Keypad0[8]) ||
						(emu.RowSelKeypad0&8 > 0 && emu.Input.Keypad0[11])))
			}

		case 0x0d:
			if emu.Input45LatchMode {
				val = boolBit(7, emu.Input5LatchVal)
			} else {
				val = boolBit(7, !emu.Input.JoyP1.Button)
			}
			if emu.DDRModeMaskPortA == 0xff {
				val = boolBit(7,
					!((emu.RowSelKeypad1&1 > 0 && emu.Input.Keypad1[2]) ||
						(emu.RowSelKeypad1&2 > 0 && emu.Input.Keypad1[5]) ||
						(emu.RowSelKeypad1&4 > 0 && emu.Input.Keypad1[8]) ||
						(emu.RowSelKeypad1&8 > 0 && emu.Input.Keypad1[11])))
			}

			//case 0x0e, 0x0f:
			// TODO: return garbage? switch to crash for debug purposes?
		}

	case !bitOn(12) && !bitOn(9) && bitOn(7):
		val = emu.Mem.RAM[addr&0x7f]

	case !bitOn(12) && bitOn(9) && bitOn(7):
		// IO
		maskedAddr := addr & 0x7
		switch maskedAddr {
		case 0x0: // 0x280
			if emu.DDRModeMaskPortA == 0 {
				val = byteFromBools(
					!emu.Input.JoyP0.Right && !emu.Input.Paddle0.Button,
					!emu.Input.JoyP0.Left && !emu.Input.Paddle1.Button,
					!emu.Input.JoyP0.Down,
					!emu.Input.JoyP0.Up,
					!emu.Input.JoyP1.Right && !emu.Input.Paddle2.Button,
					!emu.Input.JoyP1.Left && !emu.Input.Paddle3.Button,
					!emu.Input.JoyP1.Down,
					!emu.Input.JoyP1.Up,
				)
			}
		case 0x1: // 0x281
			val = emu.DDRModeMaskPortA
		case 0x2: // 0x282
			val = byteFromBools(
				emu.Input.P1DifficultySwitch,
				emu.Input.P0DifficultySwitch,
				emu.SwtchBUnusedBit2,
				emu.SwtchBUnusedBit1,
				!emu.Input.TVBWSwitch,
				emu.SwtchBUnusedBit0,
				!emu.Input.SelectButton,
				!emu.Input.ResetButton,
			)
		case 0x3: // 0x283
			val = emu.DDRModeMaskPortB
		case 0x4, 0x06: // 0x284, 0x286
			val = emu.Timer.readINTIM()
		case 0x5, 0x07: // 0x285, 0x287
			val = emu.Timer.readINSTAT()
		default:
			emuErr(fmt.Sprintf("impossible io read 0x%04x 0x%04x", addr, maskedAddr))
		}

	case bitOn(12):
		val = emu.Mem.mapper.read(&emu.Mem, addr)

	default:
		emuErr(fmt.Sprintf("unimplemented read: 0x%04x", addr))
	}
	if showMemReads {
		fmt.Printf("read(0x%04x) = 0x%02x\n", origAddr, val)
	}
	return val
}

func (emu *emuState) write(addr uint16, val byte) {
	origAddr := addr

	if emu.Mem.mapper.getMapperNum() == 0 && len(emu.Mem.rom) > 4096 {
		emu.Mem.mapper = emu.guessMapperFromAddr(addr)
	}

	if showMemWrites {
		fmt.Printf("write(0x%04x, 0x%02x)\n", addr, val)
	}

	bitOn := func(bit byte) bool {
		return addr&(1<<bit) != 0
	}

	switch {
	case !bitOn(12) && !bitOn(7):

		// TIA

		maskedAddr := addr & 0x3f

		if emu.Mem.mapper.getMapperNum() == 0x3f {
			emu.Mem.mapper.write(&emu.Mem, addr, val)
		}

		switch maskedAddr {
		case 0x00:
			emu.TIA.InVSync = val&0x02 != 0
		case 0x01:
			wasLatched := emu.Input45LatchMode
			emu.Input45LatchMode = val&0x40 != 0
			if !(wasLatched && emu.Input45LatchMode) {
				emu.Input4LatchVal = true
				emu.Input5LatchVal = true
			}

			// TODO: paddle controllers, 3button adapter
			wasTied := emu.Input03TiedToLow
			emu.Input03TiedToLow = val&0x80 != 0
			if wasTied && !emu.Input03TiedToLow {
				emu.InputTimingPots = true
				emu.InputTimingPotsStartCycles = emu.Cycles
			}
			if emu.Input03TiedToLow {
				emu.Paddle0InputCharged = false
				emu.Paddle1InputCharged = false
				emu.Paddle2InputCharged = false
				emu.Paddle3InputCharged = false
				emu.InputTimingPots = false
			}

			emu.TIA.InVBlank = val&0x02 != 0
		case 0x02:
			emu.TIA.WaitForHBlank = true
		case 0x03:
			emu.TIA.resetHorizCounter()
		case 0x04:
			emu.TIA.P0.RepeatMode = val & 0x07
			emu.TIA.M0.RepeatMode = val & 0x07
			emu.TIA.M0.Size = 1 << ((val >> 4) & 3)
		case 0x05:
			emu.TIA.P1.RepeatMode = val & 0x07
			emu.TIA.M1.RepeatMode = val & 0x07
			emu.TIA.M1.Size = 1 << ((val >> 4) & 3)
		case 0x06:
			emu.TIA.P0.ColorLuma = val & 0xfe
		case 0x07:
			emu.TIA.P1.ColorLuma = val & 0xfe
		case 0x08:
			emu.TIA.PlayfieldAndBallColorLuma = val & 0xfe
		case 0x09:
			emu.TIA.BGColorLuma = val & 0xfe
		case 0x0a:
			boolsFromByte(val,
				nil, nil, nil, nil, nil,
				&emu.TIA.PFAndBLHavePriority,
				&emu.TIA.PlayfieldScoreColorMode,
				&emu.TIA.PlayfieldReflect,
			)
			emu.TIA.BL.Size = 1 << ((val >> 4) & 3)
		case 0x0b:
			emu.TIA.P0.Reflect = val&0x08 != 0
		case 0x0c:
			emu.TIA.P1.Reflect = val&0x08 != 0
		case 0x0d:
			emu.TIA.PlayfieldToLoad &^= 0x0f0000
			emu.TIA.PlayfieldToLoad |= uint32(reverseByte(val)&0x0f) << 16
		case 0x0e:
			emu.TIA.PlayfieldToLoad &^= 0x00ff00
			emu.TIA.PlayfieldToLoad |= uint32(val) << 8
		case 0x0f:
			emu.TIA.PlayfieldToLoad &^= 0x0000ff
			emu.TIA.PlayfieldToLoad |= uint32(reverseByte(val))

		case 0x10:
			emu.TIA.resetP0()
		case 0x11:
			emu.TIA.resetP1()
		case 0x12:
			emu.TIA.resetM0()
		case 0x13:
			emu.TIA.resetM1()
		case 0x14:
			emu.TIA.resetBL()

		case 0x15:
			emu.APU.Channel0.Control = val & 0x0f
		case 0x16:
			emu.APU.Channel1.Control = val & 0x0f
		case 0x17:
			emu.APU.Channel0.FreqDiv = (val & 0x1f) + 1
		case 0x18:
			emu.APU.Channel1.FreqDiv = (val & 0x1f) + 1
		case 0x19:
			emu.APU.Channel0.Volume = val & 0x0f
		case 0x1a:
			emu.APU.Channel1.Volume = val & 0x0f

		case 0x1b:
			emu.TIA.loadShapeP0(val)
		case 0x1c:
			emu.TIA.loadShapeP1(val)
		case 0x1d:
			emu.TIA.M0.Show = val&0x02 != 0
		case 0x1e:
			emu.TIA.M1.Show = val&0x02 != 0
		case 0x1f:
			emu.TIA.loadEnablBL(val&0x02 != 0)
		case 0x20:
			emu.TIA.P0.Vx = int8(val&0xf0) >> 4
		case 0x21:
			emu.TIA.P1.Vx = int8(val&0xf0) >> 4
		case 0x22:
			emu.TIA.M0.Vx = int8(val&0xf0) >> 4
		case 0x23:
			emu.TIA.M1.Vx = int8(val&0xf0) >> 4
		case 0x24:
			emu.TIA.BL.Vx = int8(val&0xf0) >> 4
		case 0x25:
			emu.TIA.DelayGRP0 = val&0x01 != 0
		case 0x26:
			emu.TIA.DelayGRP1 = val&0x01 != 0
		case 0x27:
			emu.TIA.DelayGRBL = val&0x01 != 0
		case 0x28:
			emu.TIA.HideM0 = val&0x02 != 0
		case 0x29:
			emu.TIA.HideM1 = val&0x02 != 0
		case 0x2a:
			emu.TIA.applyHorizMotion()
		case 0x2b:
			emu.TIA.clearHorizMotion()
		case 0x2c:
			emu.TIA.clearCollisions()
		}

	case !bitOn(12) && !bitOn(9) && bitOn(7):
		maskedAddr := addr & 0xff
		emu.Mem.RAM[maskedAddr-0x80] = val

	case !bitOn(12) && bitOn(9) && bitOn(7):
		// IO
		maskedAddr := addr & 0x07
		switch maskedAddr {
		case 0x0: // 0x280
			if emu.DDRModeMaskPortA == 0xff {
				emu.RowSelKeypad0 = ^val >> 4
				emu.RowSelKeypad1 = ^val & 0x0f
				if emu.RowSelKeypad0 > 0 && !emu.EverSelectedKeypad0 {
					fmt.Println("Keypad0 activated!")
					emu.EverSelectedKeypad0 = true
				}
				if emu.RowSelKeypad1 > 0 && !emu.EverSelectedKeypad1 {
					fmt.Println("Keypad1 activated!")
					emu.EverSelectedKeypad1 = true
				}
			}
		case 0x1: // 0x281
			emu.DDRModeMaskPortA = val
		case 0x2: // 0x282
			boolsFromByte(val,
				nil,
				nil,
				&emu.SwtchBUnusedBit2,
				&emu.SwtchBUnusedBit1,
				nil,
				&emu.SwtchBUnusedBit0,
				nil,
				nil,
			)
		case 0x3: // 0x283
			// TODO: should this be unsettable and tied to 0?
			emu.DDRModeMaskPortB = val
		case 0x4: // 0x284, 0x294
			emu.Timer.writeAnyTIMT(1, val)
		case 0x5: // 0x285, 0x295
			emu.Timer.writeAnyTIMT(8, val)
		case 0x6: // 0x286, 0x296
			emu.Timer.writeAnyTIMT(64, val)
		case 0x7: // 0x287, 0x297
			emu.Timer.writeAnyTIMT(1024, val)
		default:
			emuErr(fmt.Sprintf("TODO: io write 0x%04x 0x%04x", addr, maskedAddr))
		}

	case bitOn(12):
		// ROM (or Mapper RAM)
		emu.Mem.mapper.write(&emu.Mem, addr, val)

	default:
		emuErr(fmt.Sprintf("unimplemented: write(0x%04x, 0x%02x)", origAddr, val))
	}

	emu.Mem.lastWriteAddr = addr
}
