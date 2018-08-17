package vcsgo

import (
	"crypto/md5"
	"fmt"
)

// NOTE: only using hashes for mappers that are no longer
// used and are tough to add to heuristics without stepping
// on other mappers (or are tough to use heuristics on).
var mapperListE0 = []string{
	"27c6a2ca16ad7d814626ceea62fa8fb4", // Frogger II (NTSC)
	"fb91dfc36cddaa54b09924ae8fd96199", // Frogger II (PAL)

	"b311ab95e85bc0162308390728a7361d", // Gyruss (NTSC)
	"e600f5e98a20fafa47676198efe6834d", // Gyruss (PAL)

	"e51030251e440cffaab1ac63438b44ae", // James Bond 007 (NTSC)

	"e24d7d879281ffec0641e9c3f52e505a", // Lord of the Rings (Prototype)

	"3347a6dd59049b15a38394aa2dafa585", // Montezuma's Revenge (NTSC)
	"9f59eddf9ba91a7d93bce7ee4b7693bc", // Montezuma's Revenge (PAL) (Hack)

	"b7a7e34e304e4b7bc565ec01ba33ea27", // Mr. Do's Castle (NTSC)

	"c7f13ef38f61ee2367ada94fdcc6d206", // Popeye (NTSC)
	"e9cb18770a41a16de63b124c1e8bd493", // Popeye (PAL)

	"72b8dc752befbfb3ffda120eb98b2dd0", // Q-bert's Qubes (NTSC)
	"517592e6e0c71731019c0cebc2ce044f", // Q-bert's Qubes (NTSC) [a1]

	"5336f86f6b982cc925532f2e80aa1e17", // Star Wars - Death Star Battle (NTSC)
	"cb9b2e9806a7fbab3d819cfe15f0f05a", // Star Wars - Death Star Battle (PAL)

	"c246e05b52f68ab2e9aee40f278cd158", // Star Wars - Ewok Adventure (Prototype) (NTSC) (Hack)
	"6dfad2dd2c7c16ac0fa257b6ce0be2f0", // Star Wars - Ewok Adventure (Prototype) (PAL)

	"6339d28c9a7f92054e70029eb0375837", // Star Wars - The Arcade Game (NTSC)
	"6cf054cd23a02e09298d2c6f787eb21d", // Star Wars - The Arcade Game (PAL)
	"6651e2791d38edc02c5a5fd7b47a1627", // Star Wars - The Arcade Game (NTSC) (Prototype)

	"c29f8db680990cb45ef7fef6ab57a2c2", // Super Cobra (NTSC)
	"d326db524d93fa2897ab69c42d6fb698", // Super Cobra (PAL)

	"fa2be8125c3c60ab83e1c0fe56922fcb", // Tooth Protectors (NTSC)

	"085322bae40d904f53bdcc56df0593fc", // Tutankham (NTSC)
	"66c2380c71709efa7b166621e5bb4558", // Tutankham (PAL)
}
var mapperListE7 = []string{
	"76f53abbbf39a0063f24036d6ee0968a", // Bump 'n' Jump (NTSC)
	"4dbf47c7f5ac767a3b07843a530d29a5", // Breaking News (Hack)

	"0443cfa9872cdb49069186413275fa21", // Burgertime (NTSC)

	"3b76242691730b2dd22ec0ceab351bc6", // Masters of the Universe (NTSC)
}
var mapperListFE = []string{
	"ac7c2260378975614192ca2bc3d20e0b", // Decathalon (NTSC)
	"883258dcd68cefc6cd4d40b1185116dc", // Decathalon (PAL)

	"4f618c2429138e0280969193ed6c107e", // Robot Tank (NTSC)
	"f687ec4b69611a7f78bd69b8a567937a", // Robot Tank (PAL)
	"fbb0151ea2108e33b2dbaae14a1831dd", // Robot Tank TV (Hack)

	"c032c2bd7017fdfbba9a105ec50f800e", // Thwocker (Prototype)
}
var mapperListDC = []string{
	"448c2a175afc8df174d6ff4cce12c794", // Pitfall 2 (NTSC)
	"e34c236630c945089fcdef088c4b6e06", // Pitfall 2 (PAL)
	"39a6a5a2e1f6297cceaa48bb03af02e9", // Pitfall 2 (Hack)
}

func findHash(hash string, list []string) bool {
	for i := range list {
		if hash == list[i] {
			return true
		}
	}
	return false
}
func loadMapperFromRomHash(rom []byte) mapper {
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

	case 12 * 1024:
		return &mapperFA{}

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
