package vcsgo

import "fmt"

type mem struct {
	RAM [0x80]byte // jesus christ
	ROM []byte

	MapperNum     byte
	SelectedBanks [8]byte    // maximum slices needed
	MapperRAM     [2048]byte // maximum ram needed
}

func (emu *emuState) read(addr uint16) byte {

	addr &= 0x1fff

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
			val = boolBit(7, emu.TIA.Collisions.M0P1)
			val |= boolBit(6, emu.TIA.Collisions.M0P0)
		case 0x01:
			val = boolBit(7, emu.TIA.Collisions.M1P0)
			val |= boolBit(6, emu.TIA.Collisions.M1P1)
		case 0x02:
			val = boolBit(7, emu.TIA.Collisions.P0PF)
			val |= boolBit(6, emu.TIA.Collisions.P0BL)
		case 0x03:
			val = boolBit(7, emu.TIA.Collisions.P1PF)
			val |= boolBit(6, emu.TIA.Collisions.P1BL)
		case 0x04:
			val = boolBit(7, emu.TIA.Collisions.M0PF)
			val |= boolBit(6, emu.TIA.Collisions.M0BL)
		case 0x05:
			val = boolBit(7, emu.TIA.Collisions.M1PF)
			val |= boolBit(6, emu.TIA.Collisions.M1BL)
		case 0x06:
			val = boolBit(7, emu.TIA.Collisions.BLPF)
		case 0x07:
			val = boolBit(7, emu.TIA.Collisions.P0P1)
			val |= boolBit(6, emu.TIA.Collisions.M0M1)
		case 0x0c:
			// TODO: other controllers!
			if emu.Input45LatchMode {
				val = boolBit(7, emu.Input4LatchVal)
			} else {
				val = boolBit(7, emu.Input.JoyP0.Button)
			}
		default:
			emuErr(fmt.Sprintf("TODO: tia read 0x%04x 0x%04x", addr, maskedAddr))
		}

	case !bitOn(12) && !bitOn(9) && bitOn(7):
		maskedAddr := addr & 0xff
		val = emu.Mem.RAM[maskedAddr-0x80]

	case !bitOn(12) && bitOn(9) && bitOn(7):
		// IO
		maskedAddr := addr & 0x7
		switch maskedAddr {
		case 0x0: // 0x280
			// TODO: support other input methods
			// TODO: does moved/not moved mean pressed or not?
			val = byteFromBools(
				!emu.Input.JoyP0.Right,
				!emu.Input.JoyP0.Left,
				!emu.Input.JoyP0.Down,
				!emu.Input.JoyP0.Up,
				!emu.Input.JoyP1.Right,
				!emu.Input.JoyP1.Left,
				!emu.Input.JoyP1.Down,
				!emu.Input.JoyP1.Up,
			)
		case 0x2: // 0x282
			val = byteFromBools(
				emu.Input.P1DifficultySwitch,
				emu.Input.P0DifficultySwitch,
				false,
				false,
				!emu.Input.TVBWSwitch,
				false,
				!emu.Input.SelectButton,
				!emu.Input.ResetButton,
			)
		case 0x4, 0x06: // 0x284, 0x286
			val = emu.Timer.readINTIM()
		case 0x5, 0x07: // 0x285, 0x287
			val = byteFromBools(
				emu.Timer.UnderflowSinceLastWriteAnyTIMT,
				emu.Timer.UnderflowSinceLastReadINSTAT,
				false, false, false, false, false, false,
			)
			emu.Timer.UnderflowSinceLastReadINSTAT = false
			val = emu.Timer.readINTIM()
		default:
			emuErr(fmt.Sprintf("TODO: io read 0x%04x 0x%04x", addr, maskedAddr))
		}

	case bitOn(12):

		maskedAddr := addr & 0xfff
		if len(emu.Mem.ROM) < 0xfff {
			// should be a power of two but let's handle weird homebrew bins
			maskedAddr %= uint16(len(emu.Mem.ROM))
		}

		switch emu.Mem.MapperNum {
		case 0xf8, 0xf6, 0xf4:
			bank := uint16(emu.Mem.SelectedBanks[0])
			val = emu.Mem.ROM[bank*4096+maskedAddr]
		case 0xe0:
			if maskedAddr < 0xc00 {
				selector := maskedAddr >> 10
				bank := uint16(emu.Mem.SelectedBanks[selector])
				val = emu.Mem.ROM[bank*1024+(maskedAddr&0x3ff)]
			} else {
				val = emu.Mem.ROM[7*1024+(maskedAddr&0x3ff)]
			}
		case 0x3f:
			if maskedAddr < 0x800 {
				bank := uint16(emu.Mem.SelectedBanks[0])
				val = emu.Mem.ROM[bank*2048+maskedAddr]
			} else {
				val = emu.Mem.ROM[3*2048+(maskedAddr&0x7ff)]
			}
		case 0xfa:
			if addr >= 0x1100 && addr <= 0x11ff {
				val = emu.Mem.MapperRAM[addr&0xff]
				// TODO(?): another clause for RAM write zone?
				// would that write garbage data?
			} else {
				bank := uint16(emu.Mem.SelectedBanks[0])
				val = emu.Mem.ROM[bank*4096+maskedAddr]
			}
		case 0xe7:
			emuErr("e7 mapper reads not yet implemented")
		case 0xf0:
			if addr == 0x1fec {
				val = emu.Mem.SelectedBanks[0]
			} else {
				bank := uint16(emu.Mem.SelectedBanks[0])
				val = emu.Mem.ROM[bank*4096+maskedAddr]
			}
		case 0x00:
			val = emu.Mem.ROM[maskedAddr]
		default:
			emuErr(fmt.Sprintf("read with unknown mapper 0x%02x", emu.Mem.MapperNum))
		}

	default:
		emuErr(fmt.Sprintf("unimplemented read: 0x%04x", addr))
	}
	if showMemReads {
		fmt.Printf("read(0x%04x) = 0x%02x\n", addr, val)
	}
	return val
}

func (emu *emuState) write(addr uint16, val byte) {

	addr &= 0x1fff

	if emu.Mem.MapperNum == 0x00 && len(emu.Mem.ROM) > 4096 {
		emu.Mem.MapperNum = emu.guessMapperFromWrite(addr)
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
		if emu.Mem.MapperNum == 0x3f {
			emu.Mem.SelectedBanks[0] = val & 0x03
		}

		maskedAddr := addr & 0x3f

		if addr <= 0x2c {
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
				if val&0x80 != 0 {
					emuErr("input0-3 control on mem 0x01 not yet implemented")
				}
				emu.TIA.InVBlank = val&0x02 != 0
			case 0x02:
				emu.TIA.WaitForHBlank = true
			case 0x03:
				emu.TIA.resetHorizCounter()
			case 0x04:
				emu.TIA.P0.RepeatMode = val & 0x07
				emu.TIA.M0.Size = 1 << ((val >> 4) & 3)
			case 0x05:
				emu.TIA.P1.RepeatMode = val & 0x07
				emu.TIA.M1.Size = 1 << ((val >> 4) & 3)
			case 0x06:
				emu.TIA.P0.Color = val >> 4
				emu.TIA.P0.Luma = (val >> 1) & 0x07
				emu.TIA.M0.Color = val >> 4
				emu.TIA.M0.Luma = (val >> 1) & 0x07
			case 0x07:
				emu.TIA.P1.Color = val >> 4
				emu.TIA.P1.Luma = (val >> 1) & 0x07
				emu.TIA.M1.Color = val >> 4
				emu.TIA.M1.Luma = (val >> 1) & 0x07
			case 0x08:
				emu.TIA.PlayfieldColor = val >> 4
				emu.TIA.PlayfieldLuma = (val >> 1) & 0x07
				emu.TIA.BL.Color = val >> 4
				emu.TIA.BL.Luma = (val >> 1) & 0x07
			case 0x09:
				emu.TIA.BGColor = val >> 4
				emu.TIA.BGLuma = (val >> 1) & 0x07
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
				emu.TIA.Playfield &^= 0x0f0000
				emu.TIA.Playfield |= uint32(reverseByte(val)&0x0f) << 16
			case 0x0e:
				emu.TIA.Playfield &^= 0x00ff00
				emu.TIA.Playfield |= uint32(val) << 8
			case 0x0f:
				emu.TIA.Playfield &^= 0x0000ff
				emu.TIA.Playfield |= uint32(reverseByte(val))

			case 0x10:
				emu.TIA.resetPlayer(&emu.TIA.P0)
			case 0x11:
				emu.TIA.resetPlayer(&emu.TIA.P1)
			case 0x12:
				emu.TIA.resetObject(&emu.TIA.M0)
			case 0x13:
				emu.TIA.resetObject(&emu.TIA.M1)
			case 0x14:
				emu.TIA.resetObject(&emu.TIA.BL)

			case 0x15:
				emu.APU.Channel0.Control = val & 0x0f
			case 0x16:
				emu.APU.Channel1.Control = val & 0x0f
			case 0x17:
				emu.APU.Channel0.FreqDiv = val & 0x1f
			case 0x18:
				emu.APU.Channel1.FreqDiv = val & 0x1f
			case 0x19:
				emu.APU.Channel0.Volume = val & 0x0f
			case 0x1a:
				emu.APU.Channel1.Volume = val & 0x0f

			case 0x1b:
				emu.TIA.P0.Shape = val
			case 0x1c:
				emu.TIA.P1.Shape = val
			case 0x1d:
				emu.TIA.M0.Show = val&0x02 != 0
			case 0x1e:
				emu.TIA.M1.Show = val&0x02 != 0
			case 0x1f:
				emu.TIA.BL.Show = val&0x02 != 0
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
			default:
				emuErr(fmt.Sprintf("TODO: tia write 0x%04x 0x%04x", addr, maskedAddr))
			}
		}

	case !bitOn(12) && !bitOn(9) && bitOn(7):
		maskedAddr := addr & 0xff
		emu.Mem.RAM[maskedAddr-0x80] = val

	case !bitOn(12) && bitOn(9) && bitOn(7):
		// IO
		maskedAddr := addr & 0x07
		switch maskedAddr {
		case 0x1: // 0x281
			emu.DDRModeMaskPortA = val
		case 0x3: // 0x283
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
		// ROM
		switch emu.Mem.MapperNum {
		case 0xf8:
			if addr >= 0x1ff8 && addr <= 0x1ff9 {
				emu.Mem.SelectedBanks[0] = byte(addr - 0x1ff8)
			}
		case 0xe0:
			if addr >= 0x1fe0 && addr <= 0x1ff8 {
				sliceNum := ((addr - 0x1fe0) >> 3) & 3
				bankNum := byte((addr - 0x1fe0) & 7)
				emu.Mem.SelectedBanks[sliceNum] = bankNum
			}
		case 0x3f:
			// pass (handled in TIA write section)
		case 0xfa:
			if addr >= 0x1ff8 && addr <= 0x1ffa {
				emu.Mem.SelectedBanks[0] = byte(addr - 0x1ff8)
			} else if addr >= 0x1000 && addr <= 0x10ff {
				emu.Mem.MapperRAM[addr-0x1000] = val
			}
		case 0xf6:
			if addr >= 0x1ff6 && addr <= 0x1ff9 {
				emu.Mem.SelectedBanks[0] = byte(addr - 0x1ff6)
			}
		case 0xf4:
			if addr >= 0x1ff4 && addr <= 0x1ffb {
				emu.Mem.SelectedBanks[0] = byte(addr - 0x1ff4)
			}
		case 0xf0:
			if addr == 0x1ff0 {
				emu.Mem.SelectedBanks[0]++
				emu.Mem.SelectedBanks[0] &= 0x0f
			}
		case 0x00:
			// pass
		default:
			emuErr(fmt.Sprintf("rom bank switch or nop writes unimplemented for mapper 0x%02x: write(0x%04x, 0x%02x)", emu.Mem.MapperNum, addr, val))
		}

	default:
		emuErr(fmt.Sprintf("unimplemented: write(0x%04x, 0x%02x)", addr, val))
	}
}

func (emu *emuState) guessMapperFromWrite(addr uint16) byte {
	switch len(emu.Mem.ROM) {
	case 8 * 1024:
		if addr == 0x003f {
			// NOTE: it could be writing anywhere from 00-3f if they don't
			// care about trashing TIA, but lets only check for 3f for now
			return 0x3f
		} else if addr >= 0x1fe0 && addr < 0x1ff8 {
			// NOTE: FIXME(?): Assuming that E0 mappers will *not* write to 0x1ff8 first. Basically just hoping to get lucky.
			return 0xe0
		} else if addr == 0x1ff8 || addr == 0x1ff9 {
			return 0xf8
		}
	case 12 * 1024:
		return 0xfa
	case 16 * 1024:
		if addr >= 0x1ff6 && addr <= 0x1ff9 {
			return 0xf6
		}
		if addr >= 0x1fe0 && addr < 0x1fe7 {
			emuErr("0xe7 mapper not yet supported")
			//return 0xe7
		}
	case 32 * 1024:
		return 0xf4
	case 64 * 1024:
		return 0xf0
	default:
		emuErr(fmt.Sprint("unknown rom size", len(emu.Mem.ROM)))
	}
	return 0x00
}
