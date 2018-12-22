package vcsgo

import (
	"crypto/md5"
	"fmt"
)

func findHash(hash string, list []string) bool {
	for i := range list {
		if hash == list[i] {
			return true
		}
	}
	return false
}
func loadMapperFromRomInfo(rom []byte) mapper {
	hash := fmt.Sprintf("%x", md5.Sum(rom))
	if findHash(hash, mapperListE0) {
		return &mapperE0{}
	}
	if findHash(hash, mapperListE7) {
		return &mapperE7{}
	}
	if findHash(hash, mapperListFE) {
		return &mapperFE{}
	}
	if findHash(hash, mapperListDC) {
		return makeMapperDC()
	}
	if findHash(hash, mapperListC0) {
		return &mapperC0{}
	}
	switch len(rom) {
	case 8*1024 + 256:
		return &mapper66{}
	case 12 * 1024:
		return &mapperFA{}
	}
	return &mapperUnknown{}
}

func (emu *emuState) guessMapperFromAddr(addr uint16) mapper {

	addr &= 0x1fff

	is3F := func(addr uint16) bool {
		// NOTE: it could be writing anywhere from 00-3f if they don't
		// care about trashing TIA, but let's only check for 3f for now
		if addr != 0x003f {
			return false
		}
		// HACK: unfortunately needed to get past clear-all-regs routines
		lastAddr := emu.Mem.lastWriteAddr
		return lastAddr != 0x003e && lastAddr != 0x0040
	}

	switch len(emu.Mem.rom) {
	case 8 * 1024:
		if addr == 0x1ff8 || addr == 0x1ff9 {
			return makeMapperF8()
		}
		if is3F(addr) {
			return &mapper3F{}
		}

	case 16 * 1024:
		if addr >= 0x1ff6 && addr <= 0x1ff9 {
			return makeMapperF6()
		}
		if is3F(addr) {
			return &mapper3F{}
		}

	case 32 * 1024:
		if addr >= 0x1ff4 && addr <= 0x1ffb {
			return makeMapperF4()
		}
		if is3F(addr) {
			return &mapper3F{}
		}

	case 64 * 1024:
		if addr == 0x1ff0 {
			return &mapperF0{}
		}
		if is3F(addr) {
			return &mapper3F{}
		}

	default:
		emuErr(fmt.Sprint("unknown rom size", len(emu.Mem.rom)))
	}
	return emu.Mem.mapper
}
