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
	case 0xdc:
		mapper = &mapperDC{}
	case 0xe0:
		mapper = &mapperE0{}
	case 0xe7:
		mapper = &mapperE7{}
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
	case 0xfe:
		mapper = &mapperFE{}
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
	} else if len(mem.rom) > 0x1000 {
		// match those mappers that set some last chunk to the last bit of ROM
		bankStart := uint16(len(mem.rom) - 0x1000)
		return mem.rom[bankStart+maskedAddr]
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
	addr &= 0x1fff
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
	addr &= 0x1fff
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
	addr &= 0x1fff
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
	addr &= 0x1fff
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
	addr &= 0x1fff
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
	addr &= 0x1fff
	if addr == 0x1ff0 {
		m.BankNum = (m.BankNum + 1) & 0x0f
	}
}
func (m *mapperF0) getMapperNum() uint16 {
	return 0xf0
}

type mapperE0 struct {
	SelectedBanks [4]uint16
}

func (m *mapperE0) read(mem *mem, addr uint16) byte {
	addr &= 0x1fff
	if addr >= 0x1c00 {
		m.setBankIf(addr)
		return mem.rom[7*1024+(addr&0x3ff)]
	}
	bankZone := (addr & 0xfff) >> 10
	bank := uint16(m.SelectedBanks[bankZone])
	return mem.rom[bank*1024+(addr&0x3ff)]
}
func (m *mapperE0) setBankIf(addr uint16) {
	if addr >= 0x1fe0 && addr <= 0x1ff8 {
		sliceNum := ((addr - 0x1fe0) >> 3) & 3
		bankNum := (addr - 0x1fe0) & 7
		m.SelectedBanks[sliceNum] = bankNum
	}
}
func (m *mapperE0) write(mem *mem, addr uint16, val byte) {
	addr &= 0x1fff
	m.setBankIf(addr)
}
func (m *mapperE0) getMapperNum() uint16 {
	return 0xe0
}

type mapper3F struct {
	BankNum uint16
}

func (m *mapper3F) read(mem *mem, addr uint16) byte {
	addr &= 0x1fff
	if addr < 0x1800 {
		return mem.rom[m.BankNum*2048+(addr&0x7ff)]
	}
	romLen := uint16(len(mem.rom))
	return mem.rom[romLen-2048+(addr&0x7ff)]
}
func (m *mapper3F) write(mem *mem, addr uint16, val byte) {
	addr &= 0x1fff
	if addr >= 0 && addr <= 0x3f {
		m.BankNum = uint16(val)
	}
}
func (m *mapper3F) getMapperNum() uint16 {
	return 0x3f
}

type mapperFE struct {
	BankNum uint16
}

func (m *mapperFE) read(mem *mem, addr uint16) byte {
	if addr&0x2000 > 0 {
		return mem.rom[addr&0xfff]
	}
	return mem.rom[4096+(addr&0xfff)]
}
func (m *mapperFE) write(mem *mem, addr uint16, val byte) {}
func (m *mapperFE) getMapperNum() uint16 {
	return 0xfe
}

type mapperE7 struct {
	BankNum      uint16
	RAMBankNum   uint16
	RAM          [2048]byte
	BigRAMBankOn bool
}

func (m *mapperE7) read(mem *mem, addr uint16) byte {
	addr &= 0x1fff
	var val byte
	if addr >= 0x1400 && addr <= 0x17ff && m.BankNum == 8 {
		val = m.RAM[addr-0x1000]
	} else if addr >= 0x1000 && addr <= 0x17ff {
		val = mem.rom[m.BankNum*2048+(addr-0x1000)]
	} else if addr >= 0x1800 && addr <= 0x18ff {
		val = 0 // write area of RAM, TODO: actual behavior
	} else if addr >= 0x1900 && addr <= 0x19ff {
		ramStart := 1024 + m.RAMBankNum*256
		val = m.RAM[ramStart+(addr-0x1900)]
	} else { // all that's left is >= 0x1a00
		romLen := uint16(len(mem.rom))
		val = mem.rom[romLen-1536+(addr-0x1a00)]
		m.setBankIf(addr)
	}
	return val
}
func (m *mapperE7) write(mem *mem, addr uint16, val byte) {
	addr &= 0x1fff
	if addr >= 0x1000 && addr <= 0x13ff && m.BankNum == 8 {
		m.RAM[addr-0x1000] = val
	} else if addr >= 0x1800 && addr <= 0x18ff {
		ramStart := 1024 + m.RAMBankNum*256
		m.RAM[ramStart+(addr-0x1800)] = val
	}
	m.setBankIf(addr)
}
func (m *mapperE7) setBankIf(addr uint16) {
	if addr >= 0x1fe0 && addr <= 0x1fe6 {
		m.BankNum = (addr - 0x1fe0)
	} else if addr == 0x1fe7 {
		m.BankNum = 8 // RAM
	} else if addr >= 0x1fe8 && addr <= 0x1feb {
		m.RAMBankNum = addr - 0x1fe8
	}
}
func (m *mapperE7) getMapperNum() uint16 {
	return 0xe7
}

type mapperDC struct {
	BankNum uint16
}

func (m *mapperDC) read(mem *mem, addr uint16) byte {
	return 0
}
func (m *mapperDC) write(mem *mem, addr uint16, val byte) {
}
func (m *mapperDC) getMapperNum() uint16 {
	return 0xdc
}
