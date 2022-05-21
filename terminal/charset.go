package terminal

var vtCharSets = map[rune]*map[byte]rune{
	'0': &vt_DEC_Special,
	'U': &vt_ISO_UK,     // should be 'A' 94-charset, using 'U' to avoid map key confliction
	'A': &vt_ISO_8859_1, // should be 'A' 96-charset
	'<': &vt_DEC_Supplement,
	'>': &vt_DEC_Technical,
	'B': nil, // UTF-8
}

/*
These tables perform translation of built-in "hard" character sets
to 16-bit Unicode points. Here we use a map to show the
difference clearly and save the memory. we don't distinguish
96 or 94 characters, even those originally designated by DEC
as 94-character sets.

These tables are referenced by vtCharSets.

Ref: the design of zutty and darktile
*/

// Ref: https://en.wikipedia.org/wiki/DEC_Special_Graphics
var vt_DEC_Special = map[byte]rune{
	0x5f: 0x00A0, // NO-BREAK SPACE
	0x60: 0x25C6, // BLACK DIAMOND
	0x61: 0x2592, // MEDIUM SHADE
	0x62: 0x2409, // SYMBOL FOR HORIZONTAL TABULATION
	0x63: 0x240C, // SYMBOL FOR FORM FEED
	0x64: 0x240D, // SYMBOL FOR CARRIAGE RETURN
	0x65: 0x240A, // SYMBOL FOR LINE FEED
	0x66: 0x00B0, // DEGREE SIGN
	0x67: 0x00B1, // PLUS-MINUS SIGN
	0x68: 0x2424, // SYMBOL FOR NEWLINE
	0x69: 0x240B, // SYMBOL FOR VERTICAL TABULATION
	0x6a: 0x2518, // BOX DRAWINGS LIGHT UP AND LEFT
	0x6b: 0x2510, // BOX DRAWINGS LIGHT DOWN AND LEFT
	0x6c: 0x250C, // BOX DRAWINGS LIGHT DOWN AND RIGHT
	0x6d: 0x2514, // BOX DRAWINGS LIGHT UP AND RIGHT
	0x6e: 0x253C, // BOX DRAWINGS LIGHT VERTICAL AND HORIZONTAL
	0x6f: 0x23BA, // HORIZONTAL SCAN LINE-1
	0x70: 0x23BB, // HORIZONTAL SCAN LINE-3
	0x71: 0x2500, // BOX DRAWINGS LIGHT HORIZONTAL
	0x72: 0x23BC, // HORIZONTAL SCAN LINE-7
	0x73: 0x23BD, // HORIZONTAL SCAN LINE-9
	0x74: 0x251C, // BOX DRAWINGS LIGHT VERTICAL AND RIGHT
	0x75: 0x2524, // BOX DRAWINGS LIGHT VERTICAL AND LEFT
	0x76: 0x2534, // BOX DRAWINGS LIGHT UP AND HORIZONTAL
	0x77: 0x252C, // BOX DRAWINGS LIGHT DOWN AND HORIZONTAL
	0x78: 0x2502, // BOX DRAWINGS LIGHT VERTICAL
	0x79: 0x2264, // LESS-THAN OR EQUAL TO
	0x7a: 0x2265, // GREATER-THAN OR EQUAL TO
	0x7b: 0x03C0, // GREEK SMALL LETTER PI
	0x7c: 0x2260, // NOT EQUAL TO
	0x7d: 0x00A3, // POUND SIGN
	0x7e: 0x00B7, // MIDDLE DOT
}

// Ref: https://en.wikipedia.org/wiki/Multinational_Character_Set
var vt_DEC_Supplement = map[byte]rune{
	0xa0: 0x0020,
	0xa6: 0x0026,
	0xa8: 0x00a4,
	0xac: 0x002c,
	0xad: 0x002d,
	0xae: 0x002e,
	0xaf: 0x002f,
	0xb4: 0x0034,
	0xb8: 0x0038,
	0xbe: 0x003e,
	0xd0: 0x0050,
	0xd7: 0x0152,
	0xdd: 0x0178,
	0xde: 0x005e,
	0xf0: 0x0070,
	0xf7: 0x0153,
	0xfd: 0x00ff,
	0xfe: 0x007e,
	0xff: 0x007f,
}

// Ref: https://en.wikipedia.org/wiki/DEC_Technical_Character_Set
var vt_DEC_Technical = map[byte]rune{
	0x20: 0x0020, 0x21: 0x23b7, 0x22: 0x250c, 0x23: 0x2500, 0x24: 0x2320, 0x25: 0x2321, 0x26: 0x2502, 0x27: 0x23a1,
	0x28: 0x23a3, 0x29: 0x23a4, 0x2a: 0x23a6, 0x2b: 0x239b, 0x2c: 0x239d, 0x2d: 0x239e, 0x2e: 0x23a0, 0x2f: 0x23a8,
	0x30: 0x23ac, 0x31: 0x0020, 0x32: 0x0020, 0x33: 0x0020, 0x34: 0x0020, 0x35: 0x0020, 0x36: 0x0020, 0x37: 0x0020,
	0x38: 0x0020, 0x39: 0x0020, 0x3a: 0x0020, 0x3b: 0x0020, 0x3c: 0x2264, 0x3d: 0x2260, 0x3e: 0x2265, 0x3f: 0x222b,

	0x40: 0x2234, 0x41: 0x221d, 0x42: 0x221e, 0x43: 0x00f7, 0x44: 0x0394, 0x45: 0x2207, 0x46: 0x03a6, 0x47: 0x0393,
	0x48: 0x223c, 0x49: 0x2243, 0x4a: 0x0398, 0x4b: 0x00d7, 0x4c: 0x039b, 0x4d: 0x21d4, 0x4e: 0x21d2, 0x4f: 0x2261,
	0x50: 0x03a0, 0x51: 0x03a8, 0x52: 0x0020, 0x53: 0x03a3, 0x54: 0x0020, 0x55: 0x0020, 0x56: 0x221a, 0x57: 0x03a9,
	0x58: 0x039e, 0x59: 0x03a5, 0x5a: 0x2282, 0x5b: 0x2283, 0x5c: 0x2229, 0x5d: 0x222a, 0x5e: 0x2227, 0x5f: 0x2228,

	0x60: 0x00ac, 0x61: 0x03b1, 0x62: 0x03b2, 0x63: 0x03c7, 0x64: 0x03b4, 0x65: 0x03b5, 0x66: 0x03c6, 0x67: 0x03b3,
	0x68: 0x03b7, 0x69: 0x03b9, 0x6a: 0x03b8, 0x6b: 0x03ba, 0x6c: 0x03bb, 0x6d: 0x0020, 0x6e: 0x03bd, 0x6f: 0x2202,
	0x70: 0x03c0, 0x71: 0x03c8, 0x72: 0x03c1, 0x73: 0x03c3, 0x74: 0x03c4, 0x75: 0x0020, 0x76: 0x0192, 0x77: 0x03c9,
	0x78: 0x03be, 0x79: 0x03c5, 0x7a: 0x03b6, 0x7b: 0x2190, 0x7c: 0x2191, 0x7d: 0x2192, 0x7e: 0x2193, 0x7f: 0x007f,
}

var vt_ISO_8859_1 = map[byte]rune{
	0xa0: 0x00a0,
	0xff: 0x00ff,

	// 0xa0: 0x00a0, 0xa1: 0x00a1, 0xa2: 0x00a2, 0xa3: 0x00a3, 0xa4: 0x00a4, 0xa5: 0x00a5, 0xa6: 0x00a6, 0xa7: 0x00a7,
	// 0xa8: 0x00a8, 0xa9: 0x00a9, 0xaa: 0x00aa, 0xab: 0x00ab, 0xac: 0x00ac, 0xad: 0x00ad, 0xae: 0x00ae, 0xaf: 0x00af,
	// 0xb0: 0x00b0, 0xb1: 0x00b1, 0xb2: 0x00b2, 0xb3: 0x00b3, 0xb4: 0x00b4, 0xb5: 0x00b5, 0xb6: 0x00b6, 0xb7: 0x00b7,
	// 0xb8: 0x00b8, 0xb9: 0x00b9, 0xba: 0x00ba, 0xbb: 0x00bb, 0xbc: 0x00bc, 0xbd: 0x00bd, 0xbe: 0x00be, 0xbf: 0x00bf,
	//
	// 0xc0: 0x00c0, 0xc1: 0x00c1, 0xc2: 0x00c2, 0xc3: 0x00c3, 0xc4: 0x00c4, 0xc5: 0x00c5, 0xc6: 0x00c6, 0xc7: 0x00c7,
	// 0xc8: 0x00c8, 0xc9: 0x00c9, 0xca: 0x00ca, 0xcb: 0x00cb, 0xcc: 0x00cc, 0xcd: 0x00cd, 0xce: 0x00ce, 0xcf: 0x00cf,
	// 0xd0: 0x00d0, 0xd1: 0x00d1, 0xd2: 0x00d2, 0xd3: 0x00d3, 0xd4: 0x00d4, 0xd5: 0x00d5, 0xd6: 0x00d6, 0xd7: 0x00d7,
	// 0xd8: 0x00d8, 0xd9: 0x00d9, 0xda: 0x00da, 0xdb: 0x00db, 0xdc: 0x00dc, 0xdd: 0x00dd, 0xde: 0x00de, 0xdf: 0x00df,
	//
	// 0xe0: 0x00e0, 0xe1: 0x00e1, 0xe2: 0x00e2, 0xe3: 0x00e3, 0xe4: 0x00e4, 0xe5: 0x00e5, 0xe6: 0x00e6, 0xe7: 0x00e7,
	// 0xe8: 0x00e8, 0xe9: 0x00e9, 0xea: 0x00ea, 0xeb: 0x00eb, 0xec: 0x00ec, 0xed: 0x00ed, 0xee: 0x00ee, 0xef: 0x00ef,
	// 0xf0: 0x00f0, 0xf1: 0x00f1, 0xf2: 0x00f2, 0xf3: 0x00f3, 0xf4: 0x00f4, 0xf5: 0x00f5, 0xf6: 0x00f6, 0xf7: 0x00f7,
	// 0xf8: 0x00f8, 0xf9: 0x00f9, 0xfa: 0x00fa, 0xfb: 0x00fb, 0xfc: 0x00fc, 0xfd: 0x00fd, 0xfe: 0x00fe, 0xff: 0x00ff,
}

// Same as ASCII, but with Pound sign (0x00a3 in place of 0x0023)
var vt_ISO_UK = map[byte]rune{
	0x23: 0x00A3,
}

// find a byte in the charset, if not find, return the original value.
func lookupTable(table *map[byte]rune, b byte) rune {
	if table == nil {
		return rune(b)
	}

	chr, ok := (*table)[b]
	if ok {
		return chr
	}
	return rune(b)
}
