package vcsgo

import (
	"encoding/json"
	"fmt"
)

type mapper interface {
	read(mem *mem, addr uint16) byte
	write(mem *mem, addr uint16, val byte)

	getMapperNum() uint16
	getBankNum() uint16

	runCycle(emu *emuState)
	init(emu *emuState)
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
	case 0x3f:
		mapper = &mapper3F{}
	case 0x66:
		mapper = &mapper66{}
	case 0xc0:
		mapper = &mapperC0{}
	case 0xdc:
		mapper = makeMapperDC()
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
	romSize := len(mem.rom)
	if romSize < 0x1000 {
		// should be a power of two but let's handle weird homebrew bins
		maskedAddr %= uint16(romSize)
	} else if romSize > 0x1000 {
		// match those mappers that set some last chunk to the last bit of ROM
		bankStart := uint16(romSize - 0x1000)
		return mem.rom[bankStart+maskedAddr]
	}
	return mem.rom[maskedAddr]
}
func (m *mapperUnknown) write(mem *mem, addr uint16, val byte) {
	return // no writes to ROM addrs
}
func (m *mapperUnknown) getMapperNum() uint16   { return 0x00 }
func (m *mapperUnknown) getBankNum() uint16     { return 0 }
func (m *mapperUnknown) runCycle(emu *emuState) {}
func (m *mapperUnknown) init(emu *emuState)     {}

type mapperStd struct {
	MapperNum    uint16
	BankNum      uint16
	Superchip    superchip
	NoSuperchip  bool
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
	if !m.NoSuperchip && addr >= 0x1000 && addr <= 0x107f {
		m.Superchip.write(addr, val)
	} else if addr >= m.CtrlAddrLow && addr <= m.CtrlAddrHigh {
		m.BankNum = addr - m.CtrlAddrLow
	}
}
func (m *mapperStd) getMapperNum() uint16   { return m.MapperNum }
func (m *mapperStd) getBankNum() uint16     { return m.BankNum }
func (m *mapperStd) runCycle(emu *emuState) {}
func (m *mapperStd) init(emu *emuState)     {}

func makeMapperF8() mapper {
	return &mapperStd{
		MapperNum: 0xf8, CtrlAddrLow: 0x1ff8, CtrlAddrHigh: 0x1ff9,
	}
}
func makeMapperF8NoSC() mapper {
	return &mapperStd{
		MapperNum: 0xf8, CtrlAddrLow: 0x1ff8, CtrlAddrHigh: 0x1ff9,
		NoSuperchip: true,
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
func (m *mapperFA) getMapperNum() uint16   { return 0xfa }
func (m *mapperFA) getBankNum() uint16     { return m.BankNum }
func (m *mapperFA) runCycle(emu *emuState) {}
func (m *mapperFA) init(emu *emuState)     {}

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
func (m *mapperF0) getMapperNum() uint16   { return 0xf0 }
func (m *mapperF0) getBankNum() uint16     { return m.BankNum }
func (m *mapperF0) runCycle(emu *emuState) {}
func (m *mapperF0) init(emu *emuState)     {}

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
func (m *mapperE0) getMapperNum() uint16 { return 0xe0 }
func (m *mapperE0) getBankNum() uint16 {
	return m.SelectedBanks[0]<<12 | m.SelectedBanks[1]<<8 | m.SelectedBanks[2]<<4 | m.SelectedBanks[3]
}
func (m *mapperE0) runCycle(emu *emuState) {}
func (m *mapperE0) init(emu *emuState)     {}

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
func (m *mapper3F) getMapperNum() uint16   { return 0x3f }
func (m *mapper3F) getBankNum() uint16     { return m.BankNum }
func (m *mapper3F) runCycle(emu *emuState) {}
func (m *mapper3F) init(emu *emuState)     {}

type mapperC0 struct {
	RAM [1024]byte
}

func (m *mapperC0) read(mem *mem, addr uint16) byte {
	addr &= 0x1fff
	if addr >= 0x1000 && addr <= 0x13ff {
		return m.RAM[addr-0x1000]
	} else if addr >= 0x1400 && addr <= 0x17ff {
		return 0xff
	}
	return mem.rom[addr-0x1800]
}
func (m *mapperC0) write(mem *mem, addr uint16, val byte) {
	addr &= 0x1fff
	if addr >= 0x1400 && addr <= 0x17ff {
		m.RAM[addr-0x1400] = val
	}
}
func (m *mapperC0) getMapperNum() uint16   { return 0xc0 }
func (m *mapperC0) getBankNum() uint16     { return 0 }
func (m *mapperC0) runCycle(emu *emuState) {}
func (m *mapperC0) init(emu *emuState)     {}

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
func (m *mapperFE) getMapperNum() uint16                  { return 0xfe }
func (m *mapperFE) getBankNum() uint16                    { return m.BankNum }
func (m *mapperFE) runCycle(emu *emuState)                {}
func (m *mapperFE) init(emu *emuState)                    {}

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
func (m *mapperE7) getMapperNum() uint16   { return 0xe7 }
func (m *mapperE7) getBankNum() uint16     { return m.BankNum }
func (m *mapperE7) runCycle(emu *emuState) {}
func (m *mapperE7) init(emu *emuState)     {}

const dpcROMEnd = 8192 + 2048 - 1

type dpc struct {
	MapperF8   mapper
	Ptrs       [8]dpcPtr
	LFSR       byte
	Oscillator byte
	OscCounter byte
}

type dpcPtr struct {
	Ptr       uint16
	ShowStart byte
	ShowEnd   byte
	Show      bool
	MusicMode bool
}

func (d *dpc) readLFSR() byte {
	val := d.LFSR
	d.LFSR = val<<1 | (^(val>>7 ^ val>>5 ^ val>>4 ^ val>>3) & 1)
	return val
}

func makeMapperDC() mapper {
	return &dpc{MapperF8: makeMapperF8NoSC()}
}
func (d *dpc) read(mem *mem, addr uint16) byte {
	// TODO: add dpc mirrors at 0x1800 (p2 doesn't use it)
	addr &= 0x1fff
	if addr >= 0x1000 && addr <= 0x1003 {
		return d.readLFSR()
	} else if addr >= 0x1004 && addr <= 0x1007 {
		return d.getMusic()
	} else if addr >= 0x1008 && addr <= 0x100F {
		return d.Ptrs[addr-0x1008].read(mem)
	} else if addr >= 0x1010 && addr <= 0x1017 {
		return d.Ptrs[addr-0x1010].readMasked(mem)
	} else if addr >= 0x1018 && addr <= 0x101F {
		fmt.Println("DPC - 0x1018 NOT IMPL!")
	} else if addr >= 0x1020 && addr <= 0x1027 {
		fmt.Println("DPC - 0x1020 NOT IMPL!")
	} else if addr >= 0x1028 && addr <= 0x102F {
		fmt.Println("DPC - 0x1028 NOT IMPL!")
	} else if addr >= 0x1030 && addr <= 0x1037 {
		fmt.Println("DPC - 0x1030 NOT IMPL!")
	} else if addr >= 0x1038 && addr <= 0x103f {
		fmt.Println("DPC - 0x1038 NOT IMPL!")
	} else if addr >= 0x1070 && addr <= 0x1077 {
		d.LFSR = 0
	}
	return d.MapperF8.read(mem, addr)
}

func (d *dpc) write(mem *mem, addr uint16, val byte) {
	// TODO: add dpc mirrors at 0x1800 (p2 doesn't use it)
	addr &= 0x1fff
	if addr >= 0x1040 && addr <= 0x1047 {
		d.Ptrs[addr-0x1040].setShowStart(val)
	} else if addr >= 0x1048 && addr <= 0x104f {
		d.Ptrs[addr-0x1048].setShowEnd(val)
	} else if addr >= 0x1050 && addr <= 0x1057 {
		d.Ptrs[addr-0x1050].setLo(val)
	} else if addr >= 0x1058 && addr <= 0x105c {
		d.Ptrs[addr-0x1058].setHi(val)
	} else if addr >= 0x105d && addr <= 0x105f {
		d.Ptrs[addr-0x1058].setHi(val)
		d.Ptrs[addr-0x1058].MusicMode = val&0x10 != 0 || val&0x20 != 0
	} else if addr >= 0x1060 && addr <= 0x1067 {
		if val != 0 {
			fmt.Println("DPC - 0x1060 NOT IMPL!")
		}
	} else if addr >= 0x1068 && addr <= 0x106f {
		// not used
	} else if addr >= 0x1070 && addr <= 0x1077 {
		d.LFSR = 0
	} else if addr >= 0x1078 && addr <= 0x107f {
		// not used
	} else {
		d.MapperF8.write(mem, addr, val)
	}
}

func (d *dpc) getMapperNum() uint16 { return 0xdc }
func (d *dpc) getBankNum() uint16   { return d.MapperF8.getBankNum() }
func (d *dpc) runCycle(emu *emuState) {
	if d.OscCounter++; d.OscCounter == 60 {
		d.OscCounter = 0
		for i := range d.Ptrs {
			p := &d.Ptrs[i]
			if p.MusicMode {
				p.updateMask()
				p.Ptr--
				if p.Ptr&0xff == 0xff {
					p.setLo(p.ShowStart)
				}
			}
		}
	}
}
func (d *dpc) init(emu *emuState) {}

var dpcMusicMixer = [8]byte{0, 4, 5, 9, 6, 10, 11, 15}

func (d *dpc) getMusic() byte {
	selector := d.Ptrs[5].getChannel()<<2 | d.Ptrs[6].getChannel()<<1 | d.Ptrs[7].getChannel()
	val := dpcMusicMixer[selector]
	return val
}

func (p *dpcPtr) getChannel() byte {
	if p.MusicMode && p.Show {
		return 1
	}
	return 0
}
func (p *dpcPtr) read(mem *mem) byte {
	p.updateMask()
	val := mem.rom[dpcROMEnd-(p.Ptr&0x7ff)]
	p.Ptr--
	return val
}
func (p *dpcPtr) readMasked(mem *mem) byte {
	p.updateMask()
	val := byte(0)
	if p.Show {
		val = mem.rom[dpcROMEnd-(p.Ptr&0x7ff)]
	}
	p.Ptr--
	return val
}
func (p *dpcPtr) updateMask() {
	if byte(p.Ptr) == p.ShowStart {
		p.Show = true
	} else if byte(p.Ptr) == p.ShowEnd {
		p.Show = false
	}
}
func (p *dpcPtr) setLo(lo byte) {
	p.Ptr &= 0xff00
	p.Ptr |= uint16(lo)
}
func (p *dpcPtr) setHi(hi byte) {
	p.Ptr &= 0x00ff
	p.Ptr |= uint16(hi&0x0f) << 8
}
func (p *dpcPtr) setShowStart(val byte) {
	p.ShowStart = val
	p.Show = false
}
func (p *dpcPtr) setShowEnd(val byte) {
	p.ShowEnd = val
	p.Show = false
}

type mapper66 struct {
	BankCtrl    byte
	WriteEnable bool
	ROMEnable   bool

	RAM [6 * 1024]byte

	ByteReg        byte
	ByteRegLatched bool

	LastAddr   uint16
	InWriteSeq bool
	AddrCycles byte
}

var mapper66BankTable = [2][8]uint16{
	{2, 0, 2, 0, 2, 1, 2, 1},
	{3, 3, 0, 2, 3, 3, 1, 2},
}

func (m *mapper66) read(mem *mem, addr uint16) byte {
	addr &= 0x1fff

	bankSlot := addr / 0x1800
	bank := mapper66BankTable[bankSlot][m.BankCtrl]
	offset := [2]uint16{0x1000, 0x1800}[bankSlot]

	if bank == 3 {
		emuErr("0x1000-0x1800 - ROM requested and not impld")
	}

	if m.InWriteSeq {
		if m.LastAddr != addr {
			m.AddrCycles++
		}
		if m.AddrCycles == 5 && m.WriteEnable {
			m.RAM[bank*2048+addr-offset] = m.ByteReg
		}
		if m.AddrCycles >= 5 {
			m.InWriteSeq = false
		}
	}
	m.LastAddr = addr

	if addr >= 0x1000 && addr <= 0x10ff {
		m.ByteReg = byte(addr)
		m.InWriteSeq = true
		m.AddrCycles = 0
	} else if addr == 0x1ff8 {
		m.loadCtrlReg(m.ByteReg)
		m.InWriteSeq = false // ?
	} else if addr == 0x1ff9 {
		emuErr("0x1ff9 - audio reg read not impld")
	}

	return m.RAM[bank*2048+addr-offset]
}
func (m *mapper66) write(mem *mem, addr uint16, val byte) {
	m.read(mem, addr)
}
func (m *mapper66) getMapperNum() uint16 { return 0x66 }
func (m *mapper66) getBankNum() uint16   { return uint16(m.BankCtrl) }
func (m *mapper66) runCycle(emu *emuState) {
	if m.InWriteSeq {
		// catch reads we can't see...
		addr := emu.Mem.lastReadAddr & 0x1fff
		if addr < 0x1000 {
			if addr != m.LastAddr {
				m.AddrCycles++
			}
		}
		m.LastAddr = addr
	}
}
func (m *mapper66) loadCtrlReg(reg byte) {
	m.BankCtrl = reg >> 2 & 7
	m.WriteEnable = reg&2 != 0
	m.ROMEnable = reg&1 == 0
}
func (m *mapper66) init(emu *emuState) {
	copy(m.RAM[:], emu.Mem.rom[:6*1024])

	m.BankCtrl = 2 // temp for dummy step
	emu.CPU.Step() // allows for doReset()

	emu.CPU.PC = uint16(emu.Mem.rom[8*1024+1])<<8 | uint16(emu.Mem.rom[8*1024])
	ctrlReg := emu.Mem.rom[8*1024+2]
	emu.Mem.RAM = [128]byte{}
	emu.Mem.RAM[0] = ctrlReg
	m.loadCtrlReg(ctrlReg)
}
