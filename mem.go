package vcsgo

import "fmt"

type mem struct {
	RAM [0x80]byte // jesus christ

	rom    []byte
	mapper mapper
}

func (emu *emuState) read(addr uint16) byte {
	origAddr := addr
	addr &= 0x1fff

	if emu.Mem.mapper == nil && len(emu.Mem.rom) > 4096 {
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

		// TODO: paddles, other controllers
		// (currently simulating no paddles plugged in)
		case 0x08:
			val = boolBit(0, !emu.Input03TiedToLow)
		case 0x09:
			val = boolBit(0, !emu.Input03TiedToLow)
		case 0x0a:
			val = boolBit(0, !emu.Input03TiedToLow)
		case 0x0b:
			val = boolBit(0, !emu.Input03TiedToLow)

		case 0x0c:
			if emu.Input45LatchMode {
				val = boolBit(7, !emu.Input4LatchVal)
			} else {
				val = boolBit(7, !emu.Input.JoyP0.Button)
			}
		case 0x0d:
			// TODO: other controllers!
			if emu.Input45LatchMode {
				val = boolBit(7, emu.Input5LatchVal)
			} else {
				val = boolBit(7, !emu.Input.JoyP1.Button)
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
			val = emu.DDRModeMaskPortA
		case 0x4, 0x06: // 0x284, 0x286
			val = emu.Timer.readINTIM()
		case 0x5, 0x07: // 0x285, 0x287
			val = emu.Timer.readINSTAT()
		default:
			emuErr(fmt.Sprintf("impossible io read 0x%04x 0x%04x", addr, maskedAddr))
		}

	case bitOn(12):

		if emu.Mem.mapper != nil {
			val = emu.Mem.mapper.read(&emu.Mem, addr)
		} else {
			maskedAddr := addr & 0xfff
			if len(emu.Mem.rom) < 0xfff {
				// should be a power of two but let's handle weird homebrew bins
				maskedAddr %= uint16(len(emu.Mem.rom))
			}
			val = emu.Mem.rom[maskedAddr]
		}
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
	addr &= 0x1fff

	if emu.Mem.mapper == nil && len(emu.Mem.rom) > 4096 {
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
			emu.Input03TiedToLow = val&0x80 != 0

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
			emu.TIA.Playfield &^= 0x0f0000
			emu.TIA.Playfield |= uint32(reverseByte(val)&0x0f) << 16
		case 0x0e:
			emu.TIA.Playfield &^= 0x00ff00
			emu.TIA.Playfield |= uint32(val) << 8
		case 0x0f:
			emu.TIA.Playfield &^= 0x0000ff
			emu.TIA.Playfield |= uint32(reverseByte(val))

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

		if emu.Mem.mapper != nil {
			emu.Mem.mapper.write(&emu.Mem, addr, val)
		} else {
			// pass
		}

	default:
		emuErr(fmt.Sprintf("unimplemented: write(0x%04x, 0x%02x)", origAddr, val))
	}
}

type mapper interface {
	read(mem *mem, addr uint16) byte
	write(mem *mem, addr uint16, val byte)
}

type superchip struct {
	RAM       [128]byte
	Activated bool
}

// NOTE: assumes addr already limited to 0x1000-0x10ff
func (s *superchip) read(addr uint16) byte {
	var val byte
	if addr >= 0x1080 {
		val = s.RAM[addr-0x1080]
	} else {
		s.RAM[addr-0x1000] = 0xff // write garbage
	}
	return val
}

// NOTE: assumes addr already limited to 0x1000-0x107f
func (s *superchip) write(addr uint16, val byte) {
	if !s.Activated {
		fmt.Println("activating superchip!")
		s.Activated = true
	}
	s.RAM[addr-0x1000] = val
}

type mapperStd struct {
	BankNum      uint16
	Superchip    superchip
	CtrlAddrLow  uint16
	CtrlAddrHigh uint16
}

func (m *mapperStd) read(mem *mem, addr uint16) byte {
	var val byte
	if m.Superchip.Activated && addr >= 0x1000 && addr <= 0x10ff {
		val = m.Superchip.read(addr)
	} else if addr >= m.CtrlAddrLow && addr <= m.CtrlAddrHigh {
		m.BankNum = addr - m.CtrlAddrLow
	} else {
		val = mem.rom[m.BankNum*4096+(addr&0xfff)]
	}
	return val
}
func (m *mapperStd) write(mem *mem, addr uint16, val byte) {
	if addr >= 0x1000 && addr <= 0x107f {
		m.Superchip.write(addr, val)
	} else if addr >= m.CtrlAddrLow && addr <= m.CtrlAddrHigh {
		m.BankNum = addr - m.CtrlAddrLow
	}
}

func makeMapperF8() mapper {
	return &mapperStd{
		CtrlAddrLow: 0x1ff8, CtrlAddrHigh: 0x1ff9,
	}
}
func makeMapperF6() mapper {
	return &mapperStd{
		CtrlAddrLow: 0x1ff6, CtrlAddrHigh: 0x1ff9,
	}
}
func makeMapperF4() mapper {
	return &mapperStd{
		CtrlAddrLow: 0x1ff4, CtrlAddrHigh: 0x1ffb,
	}
}

type mapperFA struct {
	BankNum   uint16
	MapperRAM [256]byte
}

func (m *mapperFA) read(mem *mem, addr uint16) byte {
	var val byte
	if addr >= 0x1100 && addr <= 0x11ff {
		val = m.MapperRAM[addr&0xff]
	} else if addr >= 0x1000 && addr <= 0x10ff {
		m.MapperRAM[addr&0xff] = 0xff // trash ram
	} else if addr >= 0x1ff8 && addr <= 0x1ffa {
		m.BankNum = addr - 0x1ff8
	} else {
		val = mem.rom[m.BankNum*4096+(addr&0xfff)]
	}
	return val
}
func (m *mapperFA) write(mem *mem, addr uint16, val byte) {
	if addr >= 0x1000 && addr <= 0x10ff {
		m.MapperRAM[addr-0x1000] = val
	} else if addr >= 0x1ff8 && addr <= 0x1ffa {
		m.BankNum = addr - 0x1ff8
	}
}

type mapperF0 struct {
	BankNum uint16
}

func (m *mapperF0) read(mem *mem, addr uint16) byte {
	var val byte
	if addr == 0x1fec {
		val = byte(m.BankNum)
	} else if addr == 0x1ff0 {
		m.BankNum = (m.BankNum + 1) & 0x0f
	} else {
		val = mem.rom[m.BankNum*4096+(addr&0xfff)]
	}
	return val
}
func (m *mapperF0) write(mem *mem, addr uint16, val byte) {
	if addr == 0x1ff0 {
		m.BankNum = (m.BankNum + 1) & 0x0f
	}
}

// TODO: REMAINING MAPPERS
/*
	WRITES
		switch emu.Mem.MapperNum {
		case 0xe0:
			if addr >= 0x1fe0 && addr <= 0x1ff8 {
				sliceNum := ((addr - 0x1fe0) >> 3) & 3
				bankNum := byte((addr - 0x1fe0) & 7)
				emu.Mem.SelectedBanks[sliceNum] = bankNum
			}
		case 0x3f:
			// in tia write section:
			// TODO: (have to have mapper control entire write?)
		    if emu.Mem.MapperNum == 0x3f {
				emu.Mem.SelectedBanks[0] = val & 0x03
			}
	READS
		switch emu.Mem.MapperNum {
		case 0xe0:
			if maskedAddr < 0xc00 {
				selector := maskedAddr >> 10
				bank := uint16(emu.Mem.SelectedBanks[selector])
				val = emu.Mem.rom[bank*1024+(maskedAddr&0x3ff)]
			} else {
				val = emu.Mem.rom[7*1024+(maskedAddr&0x3ff)]
			}
		case 0x3f:
			if maskedAddr < 0x800 {
				bank := uint16(emu.Mem.SelectedBanks[0])
				val = emu.Mem.rom[bank*2048+maskedAddr]
			} else {
				val = emu.Mem.rom[3*2048+(maskedAddr&0x7ff)]
			}
*/

func (emu *emuState) guessMapperFromAddr(addr uint16) mapper {

	addr &= 0x1fff

	switch len(emu.Mem.rom) {
	case 8 * 1024:
		/*
			if addr == 0x003f {
				fmt.Println("MAPPER: 8k 3f")
				// NOTE: it could be writing anywhere from 00-3f if they don't
				// care about trashing TIA, but lets only check for 3f for now
				return 0x3f
			} else if addr >= 0x1fe0 && addr < 0x1ff8 {
				fmt.Println("MAPPER: 8k e0")
				// NOTE: FIXME(?): Assuming that E0 mappers will *not* write to 0x1ff8 first. Basically just hoping to get lucky.
				return 0xe0
			}
		*/
		if addr == 0x1ff8 || addr == 0x1ff9 {
			fmt.Println("MAPPER: 8k f8")
			return makeMapperF8()
		}

	case 12 * 1024:
		fmt.Println("MAPPER: 12k fa")
		return &mapperFA{}

	case 16 * 1024:
		if addr >= 0x1ff6 && addr <= 0x1ff9 {
			fmt.Println("MAPPER: 16k f6")
			return makeMapperF6()
		}
		/*
			if addr >= 0x1fe0 && addr < 0x1fe7 {
				emuErr("0xe7 mapper not yet supported")
				//return 0xe7
			}
		*/

	case 32 * 1024:
		fmt.Println("MAPPER: 32k f4")
		return makeMapperF4()

	case 64 * 1024:
		fmt.Println("MAPPER: 64k f0")
		return &mapperF0{}

	default:
		emuErr(fmt.Sprint("unknown rom size", len(emu.Mem.rom)))
	}
	return nil
}
