package vcsgo

var ntscPalette = [128][3]byte{
	{0x00, 0x00, 0x00},
	{0x1A, 0x1A, 0x1A},
	{0x39, 0x39, 0x39},
	{0x5B, 0x5B, 0x5B},
	{0x7E, 0x7E, 0x7E},
	{0xA2, 0xA2, 0xA2},
	{0xC7, 0xC7, 0xC7},
	{0xED, 0xED, 0xED},

	{0x19, 0x02, 0x00},
	{0x3A, 0x1F, 0x00},
	{0x5D, 0x41, 0x00},
	{0x82, 0x64, 0x00},
	{0xA7, 0x88, 0x00},
	{0xCC, 0xAD, 0x00},
	{0xF2, 0xD2, 0x19},
	{0xFE, 0xFA, 0x40},

	{0x37, 0x00, 0x00},
	{0x5E, 0x08, 0x00},
	{0x83, 0x27, 0x00},
	{0xA9, 0x49, 0x00},
	{0xCF, 0x6C, 0x00},
	{0xF5, 0x8F, 0x17},
	{0xFE, 0xB4, 0x38},
	{0xFE, 0xDF, 0x6F},

	{0x47, 0x00, 0x00},
	{0x73, 0x00, 0x00},
	{0x98, 0x13, 0x00},
	{0xBE, 0x32, 0x16},
	{0xE4, 0x53, 0x35},
	{0xFE, 0x76, 0x57},
	{0xFE, 0x9C, 0x81},
	{0xFE, 0xC6, 0xBB},

	{0x44, 0x00, 0x08},
	{0x6F, 0x00, 0x1F},
	{0x96, 0x06, 0x40},
	{0xBB, 0x24, 0x62},
	{0xE1, 0x45, 0x85},
	{0xFE, 0x67, 0xAA},
	{0xFE, 0x8C, 0xD6},
	{0xFE, 0xB7, 0xF6},

	{0x2D, 0x00, 0x4A},
	{0x57, 0x00, 0x67},
	{0x7D, 0x05, 0x8C},
	{0xA1, 0x22, 0xB1},
	{0xC7, 0x43, 0xD7},
	{0xED, 0x65, 0xFE},
	{0xFE, 0x8A, 0xF6},
	{0xFE, 0xB5, 0xF7},

	{0x0D, 0x00, 0x82},
	{0x33, 0x00, 0xA2},
	{0x55, 0x0F, 0xC9},
	{0x78, 0x2D, 0xF0},
	{0x9C, 0x4E, 0xFE},
	{0xC3, 0x72, 0xFE},
	{0xEB, 0x98, 0xFE},
	{0xFE, 0xC0, 0xF9},

	{0x00, 0x00, 0x91},
	{0x0A, 0x05, 0xBD},
	{0x28, 0x22, 0xE4},
	{0x48, 0x42, 0xFE},
	{0x6B, 0x64, 0xFE},
	{0x90, 0x8A, 0xFE},
	{0xB7, 0xB0, 0xFE},
	{0xDF, 0xD8, 0xFE},

	{0x00, 0x00, 0x72},
	{0x00, 0x1C, 0xAB},
	{0x03, 0x3C, 0xD6},
	{0x20, 0x5E, 0xFD},
	{0x40, 0x81, 0xFE},
	{0x64, 0xA6, 0xFE},
	{0x89, 0xCE, 0xFE},
	{0xB0, 0xF6, 0xFE},

	{0x00, 0x10, 0x3A},
	{0x00, 0x31, 0x6E},
	{0x00, 0x55, 0xA2},
	{0x05, 0x79, 0xC8},
	{0x23, 0x9D, 0xEE},
	{0x44, 0xC2, 0xFE},
	{0x68, 0xE9, 0xFE},
	{0x8F, 0xFE, 0xFE},

	{0x00, 0x1F, 0x02},
	{0x00, 0x43, 0x26},
	{0x00, 0x69, 0x57},
	{0x00, 0x8D, 0x7A},
	{0x1B, 0xB1, 0x9E},
	{0x3B, 0xD7, 0xC3},
	{0x5D, 0xFE, 0xE9},
	{0x86, 0xFE, 0xFE},

	{0x00, 0x24, 0x03},
	{0x00, 0x4A, 0x05},
	{0x00, 0x70, 0x0C},
	{0x09, 0x95, 0x2B},
	{0x28, 0xBA, 0x4C},
	{0x49, 0xE0, 0x6E},
	{0x6C, 0xFE, 0x92},
	{0x97, 0xFE, 0xB5},

	{0x00, 0x21, 0x02},
	{0x00, 0x46, 0x04},
	{0x08, 0x6B, 0x00},
	{0x28, 0x90, 0x00},
	{0x49, 0xB5, 0x09},
	{0x6B, 0xDB, 0x28},
	{0x8F, 0xFE, 0x49},
	{0xBB, 0xFE, 0x69},

	{0x00, 0x15, 0x01},
	{0x10, 0x36, 0x00},
	{0x30, 0x59, 0x00},
	{0x53, 0x7E, 0x00},
	{0x76, 0xA3, 0x00},
	{0x9A, 0xC8, 0x00},
	{0xBF, 0xEE, 0x1E},
	{0xE8, 0xFE, 0x3E},

	{0x1A, 0x02, 0x00},
	{0x3B, 0x1F, 0x00},
	{0x5E, 0x41, 0x00},
	{0x83, 0x64, 0x00},
	{0xA8, 0x88, 0x00},
	{0xCE, 0xAD, 0x00},
	{0xF4, 0xD2, 0x18},
	{0xFE, 0xFA, 0x40},

	{0x38, 0x00, 0x00},
	{0x5F, 0x08, 0x00},
	{0x84, 0x27, 0x00},
	{0xAA, 0x49, 0x00},
	{0xD0, 0x6B, 0x00},
	{0xF6, 0x8F, 0x18},
	{0xFE, 0xB4, 0x39},
	{0xFE, 0xDF, 0x70},
}

var palPalette = [128][3]byte{
	{0x00, 0x00, 0x00},
	{0x1A, 0x1A, 0x1A},
	{0x39, 0x39, 0x39},
	{0x5B, 0x5B, 0x5B},
	{0x7E, 0x7E, 0x7E},
	{0xA2, 0xA2, 0xA2},
	{0xC7, 0xC7, 0xC7},
	{0xED, 0xED, 0xED},

	{0x00, 0x00, 0x00},
	{0x1A, 0x1A, 0x1A},
	{0x39, 0x39, 0x39},
	{0x5B, 0x5B, 0x5B},
	{0x7E, 0x7E, 0x7E},
	{0xA2, 0xA2, 0xA2},
	{0xC7, 0xC7, 0xC7},
	{0xED, 0xED, 0xED},

	{0x1E, 0x00, 0x00},
	{0x3F, 0x1C, 0x00},
	{0x63, 0x3D, 0x00},
	{0x88, 0x60, 0x00},
	{0xAD, 0x83, 0x00},
	{0xD2, 0xA8, 0x06},
	{0xF9, 0xCD, 0x26},
	{0xFE, 0xF6, 0x4A},

	{0x00, 0x21, 0x00},
	{0x00, 0x46, 0x00},
	{0x0D, 0x6A, 0x00},
	{0x2D, 0x90, 0x00},
	{0x4F, 0xB5, 0x00},
	{0x71, 0xDA, 0x06},
	{0x95, 0xFE, 0x26},
	{0xC0, 0xFE, 0x4D},

	{0x3A, 0x00, 0x00},
	{0x62, 0x06, 0x00},
	{0x88, 0x25, 0x00},
	{0xAD, 0x45, 0x00},
	{0xD2, 0x67, 0x1B},
	{0xF9, 0x8B, 0x3B},
	{0xFE, 0xB0, 0x5E},
	{0xFE, 0xDB, 0x87},

	{0x00, 0x25, 0x00},
	{0x00, 0x4B, 0x00},
	{0x00, 0x72, 0x00},
	{0x0D, 0x96, 0x00},
	{0x2C, 0xBB, 0x1C},
	{0x4E, 0xE1, 0x3D},
	{0x70, 0xFE, 0x5F},
	{0x9C, 0xFE, 0x8A},

	{0x47, 0x00, 0x00},
	{0x72, 0x00, 0x07},
	{0x97, 0x0F, 0x25},
	{0xBD, 0x2E, 0x45},
	{0xE3, 0x4F, 0x68},
	{0xFE, 0x72, 0x8B},
	{0xFE, 0x98, 0xB2},
	{0xFE, 0xC2, 0xDD},

	{0x00, 0x21, 0x00},
	{0x00, 0x45, 0x05},
	{0x00, 0x6C, 0x26},
	{0x00, 0x90, 0x46},
	{0x1C, 0xB5, 0x69},
	{0x3D, 0xDB, 0x8C},
	{0x5F, 0xFE, 0xB1},
	{0x88, 0xFE, 0xDD},

	{0x41, 0x00, 0x26},
	{0x6C, 0x00, 0x4F},
	{0x92, 0x04, 0x73},
	{0xB8, 0x22, 0x98},
	{0xDE, 0x43, 0xBD},
	{0xFE, 0x65, 0xE3},
	{0xFE, 0x8A, 0xFE},
	{0xFE, 0xB6, 0xFE},

	{0x00, 0x11, 0x2A},
	{0x00, 0x34, 0x4F},
	{0x00, 0x59, 0x75},
	{0x04, 0x7C, 0x9A},
	{0x22, 0xA0, 0xBF},
	{0x43, 0xC5, 0xE5},
	{0x65, 0xEB, 0xFE},
	{0x8C, 0xFE, 0xFE},

	{0x2A, 0x00, 0x65},
	{0x53, 0x00, 0x92},
	{0x78, 0x04, 0xB9},
	{0x9C, 0x22, 0xE0},
	{0xC2, 0x42, 0xFE},
	{0xE8, 0x65, 0xFE},
	{0xFE, 0x8A, 0xFE},
	{0xFE, 0xB6, 0xFE},

	{0x00, 0x00, 0x6B},
	{0x00, 0x1F, 0x94},
	{0x00, 0x40, 0xBC},
	{0x1D, 0x62, 0xE2},
	{0x3D, 0x85, 0xFE},
	{0x5F, 0xA9, 0xFE},
	{0x84, 0xD1, 0xFE},
	{0xAB, 0xF9, 0xFE},

	{0x08, 0x00, 0x8E},
	{0x2D, 0x00, 0xBC},
	{0x4E, 0x10, 0xE4},
	{0x71, 0x2F, 0xFE},
	{0x95, 0x50, 0xFE},
	{0xBB, 0x75, 0xFE},
	{0xE3, 0x9B, 0xFE},
	{0xFE, 0xC2, 0xFE},

	{0x00, 0x00, 0x90},
	{0x06, 0x08, 0xBD},
	{0x24, 0x25, 0xE4},
	{0x44, 0x45, 0xFE},
	{0x66, 0x67, 0xFE},
	{0x8B, 0x8D, 0xFE},
	{0xB2, 0xB3, 0xFE},
	{0xDA, 0xDB, 0xFE},

	{0x00, 0x00, 0x00},
	{0x1A, 0x1A, 0x1A},
	{0x39, 0x39, 0x39},
	{0x5B, 0x5B, 0x5B},
	{0x7E, 0x7E, 0x7E},
	{0xA2, 0xA2, 0xA2},
	{0xC7, 0xC7, 0xC7},
	{0xED, 0xED, 0xED},

	{0x00, 0x00, 0x00},
	{0x1A, 0x1A, 0x1A},
	{0x39, 0x39, 0x39},
	{0x5B, 0x5B, 0x5B},
	{0x7E, 0x7E, 0x7E},
	{0xA2, 0xA2, 0xA2},
	{0xC7, 0xC7, 0xC7},
	{0xED, 0xED, 0xED},
}

var secamPalette = [128][3]byte{
	{0x00, 0x00, 0x00},
	{0x21, 0x21, 0xFF},
	{0xF0, 0x3C, 0x79},
	{0xFF, 0x50, 0xFF},
	{0x7F, 0xFF, 0x00},
	{0x7F, 0xFF, 0xFF},
	{0xFF, 0xFF, 0x3F},
	{0xED, 0xED, 0xED},

	{0x00, 0x00, 0x00},
	{0x21, 0x21, 0xFF},
	{0xF0, 0x3C, 0x79},
	{0xFF, 0x50, 0xFF},
	{0x7F, 0xFF, 0x00},
	{0x7F, 0xFF, 0xFF},
	{0xFF, 0xFF, 0x3F},
	{0xED, 0xED, 0xED},

	{0x00, 0x00, 0x00},
	{0x21, 0x21, 0xFF},
	{0xF0, 0x3C, 0x79},
	{0xFF, 0x50, 0xFF},
	{0x7F, 0xFF, 0x00},
	{0x7F, 0xFF, 0xFF},
	{0xFF, 0xFF, 0x3F},
	{0xED, 0xED, 0xED},

	{0x00, 0x00, 0x00},
	{0x21, 0x21, 0xFF},
	{0xF0, 0x3C, 0x79},
	{0xFF, 0x50, 0xFF},
	{0x7F, 0xFF, 0x00},
	{0x7F, 0xFF, 0xFF},
	{0xFF, 0xFF, 0x3F},
	{0xED, 0xED, 0xED},

	{0x00, 0x00, 0x00},
	{0x21, 0x21, 0xFF},
	{0xF0, 0x3C, 0x79},
	{0xFF, 0x50, 0xFF},
	{0x7F, 0xFF, 0x00},
	{0x7F, 0xFF, 0xFF},
	{0xFF, 0xFF, 0x3F},
	{0xED, 0xED, 0xED},

	{0x00, 0x00, 0x00},
	{0x21, 0x21, 0xFF},
	{0xF0, 0x3C, 0x79},
	{0xFF, 0x50, 0xFF},
	{0x7F, 0xFF, 0x00},
	{0x7F, 0xFF, 0xFF},
	{0xFF, 0xFF, 0x3F},
	{0xED, 0xED, 0xED},

	{0x00, 0x00, 0x00},
	{0x21, 0x21, 0xFF},
	{0xF0, 0x3C, 0x79},
	{0xFF, 0x50, 0xFF},
	{0x7F, 0xFF, 0x00},
	{0x7F, 0xFF, 0xFF},
	{0xFF, 0xFF, 0x3F},
	{0xED, 0xED, 0xED},

	{0x00, 0x00, 0x00},
	{0x21, 0x21, 0xFF},
	{0xF0, 0x3C, 0x79},
	{0xFF, 0x50, 0xFF},
	{0x7F, 0xFF, 0x00},
	{0x7F, 0xFF, 0xFF},
	{0xFF, 0xFF, 0x3F},
	{0xED, 0xED, 0xED},

	{0x00, 0x00, 0x00},
	{0x21, 0x21, 0xFF},
	{0xF0, 0x3C, 0x79},
	{0xFF, 0x50, 0xFF},
	{0x7F, 0xFF, 0x00},
	{0x7F, 0xFF, 0xFF},
	{0xFF, 0xFF, 0x3F},
	{0xED, 0xED, 0xED},

	{0x00, 0x00, 0x00},
	{0x21, 0x21, 0xFF},
	{0xF0, 0x3C, 0x79},
	{0xFF, 0x50, 0xFF},
	{0x7F, 0xFF, 0x00},
	{0x7F, 0xFF, 0xFF},
	{0xFF, 0xFF, 0x3F},
	{0xED, 0xED, 0xED},

	{0x00, 0x00, 0x00},
	{0x21, 0x21, 0xFF},
	{0xF0, 0x3C, 0x79},
	{0xFF, 0x50, 0xFF},
	{0x7F, 0xFF, 0x00},
	{0x7F, 0xFF, 0xFF},
	{0xFF, 0xFF, 0x3F},
	{0xED, 0xED, 0xED},

	{0x00, 0x00, 0x00},
	{0x21, 0x21, 0xFF},
	{0xF0, 0x3C, 0x79},
	{0xFF, 0x50, 0xFF},
	{0x7F, 0xFF, 0x00},
	{0x7F, 0xFF, 0xFF},
	{0xFF, 0xFF, 0x3F},
	{0xED, 0xED, 0xED},

	{0x00, 0x00, 0x00},
	{0x21, 0x21, 0xFF},
	{0xF0, 0x3C, 0x79},
	{0xFF, 0x50, 0xFF},
	{0x7F, 0xFF, 0x00},
	{0x7F, 0xFF, 0xFF},
	{0xFF, 0xFF, 0x3F},
	{0xED, 0xED, 0xED},

	{0x00, 0x00, 0x00},
	{0x21, 0x21, 0xFF},
	{0xF0, 0x3C, 0x79},
	{0xFF, 0x50, 0xFF},
	{0x7F, 0xFF, 0x00},
	{0x7F, 0xFF, 0xFF},
	{0xFF, 0xFF, 0x3F},
	{0xED, 0xED, 0xED},

	{0x00, 0x00, 0x00},
	{0x21, 0x21, 0xFF},
	{0xF0, 0x3C, 0x79},
	{0xFF, 0x50, 0xFF},
	{0x7F, 0xFF, 0x00},
	{0x7F, 0xFF, 0xFF},
	{0xFF, 0xFF, 0x3F},
	{0xED, 0xED, 0xED},

	{0x00, 0x00, 0x00},
	{0x21, 0x21, 0xFF},
	{0xF0, 0x3C, 0x79},
	{0xFF, 0x50, 0xFF},
	{0x7F, 0xFF, 0x00},
	{0x7F, 0xFF, 0xFF},
	{0xFF, 0xFF, 0x3F},
	{0xED, 0xED, 0xED},
}
