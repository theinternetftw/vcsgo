package vcsgo

import (
	"encoding/json"
	"fmt"
)

type mapper interface {
	read(mem *mem, addr uint16) byte
	write(mem *mem, addr uint16, val byte)
	getMapperNum() uint16
}

type marshalledMapper struct {
	Number uint16
	Data   []byte
}

func unmarshalMapper(m marshalledMapper) (mapper, error) {
	var mapper mapper
	switch m.Number {
	case 0x00:
		mapper = &mapperUnknown{}
	case 0xf0:
		mapper = &mapperF0{}
	case 0xf4:
		mapper = makeMapperF4()
	case 0xf6:
		mapper = makeMapperF6()
	case 0xf8:
		mapper = makeMapperF8()
	case 0xfa:
		mapper = &mapperFA{}
	default:
		return nil, fmt.Errorf("state contained unknown mapper number 0x%04x", m.Number)
	}
	if err := json.Unmarshal(m.Data, &mapper); err != nil {
		return nil, err
	}
	return mapper, nil
}

func marshalMapper(mapper mapper) marshalledMapper {
	rawJSON, err := json.Marshal(mapper)
	if err != nil {
		panic(err)
	}
	return marshalledMapper{
		Number: mapper.getMapperNum(),
		Data:   rawJSON,
	}
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

type mapperUnknown struct{}

func (m *mapperUnknown) read(mem *mem, addr uint16) byte {
	maskedAddr := addr & 0xfff
	if len(mem.rom) < 0x1000 {
		// should be a power of two but let's handle weird homebrew bins
		maskedAddr %= uint16(len(mem.rom))
	}
	return mem.rom[maskedAddr]
}
func (m *mapperUnknown) write(mem *mem, addr uint16, val byte) {
	return // no writes to ROM addrs
}
func (m *mapperUnknown) getMapperNum() uint16 {
	return 0x00
}

type mapperStd struct {
	MapperNum    uint16
	BankNum      uint16
	Superchip    superchip
	CtrlAddrLow  uint16
	CtrlAddrHigh uint16
}

func (m *mapperStd) read(mem *mem, addr uint16) byte {
	var val byte
	if m.Superchip.Activated && addr >= 0x1000 && addr <= 0x10ff {
		val = m.Superchip.read(addr)
	} else {
		val = mem.rom[m.BankNum*4096+(addr&0xfff)]
	}
	if addr >= m.CtrlAddrLow && addr <= m.CtrlAddrHigh {
		m.BankNum = addr - m.CtrlAddrLow
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
func (m *mapperStd) getMapperNum() uint16 {
	return m.MapperNum
}

func makeMapperF8() mapper {
	return &mapperStd{
		MapperNum: 0xf8, CtrlAddrLow: 0x1ff8, CtrlAddrHigh: 0x1ff9,
	}
}
func makeMapperF6() mapper {
	return &mapperStd{
		MapperNum: 0xf6, CtrlAddrLow: 0x1ff6, CtrlAddrHigh: 0x1ff9,
	}
}
func makeMapperF4() mapper {
	return &mapperStd{
		MapperNum: 0xf4, CtrlAddrLow: 0x1ff4, CtrlAddrHigh: 0x1ffb,
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
	} else {
		val = mem.rom[m.BankNum*4096+(addr&0xfff)]
	}
	if addr >= 0x1ff8 && addr <= 0x1ffa {
		m.BankNum = addr - 0x1ff8
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
func (m *mapperFA) getMapperNum() uint16 {
	return 0xfa
}

type mapperF0 struct {
	BankNum uint16
}

func (m *mapperF0) read(mem *mem, addr uint16) byte {
	var val byte
	if addr == 0x1fec {
		val = byte(m.BankNum)
	} else {
		val = mem.rom[m.BankNum*4096+(addr&0xfff)]
	}
	if addr == 0x1ff0 {
		m.BankNum = (m.BankNum + 1) & 0x0f
	}
	return val
}
func (m *mapperF0) write(mem *mem, addr uint16, val byte) {
	if addr == 0x1ff0 {
		m.BankNum = (m.BankNum + 1) & 0x0f
	}
}
func (m *mapperF0) getMapperNum() uint16 {
	return 0xf0
}

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
	return emu.Mem.mapper
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
