package main

/*
var zero bool     // 1 if the result is 0x0, 0 if otherwise
var carry bool    // 1 if carry out of byte, 0 if no carry
var auxCarry bool // 1 if carry out of the LS nibble
var sign bool     // 1 if the MSB is 1
var parity bool   // the number of 1-bits in the resulting byte; if even then 1, 0 if odd
*/

// Sub : a-b implemented using a + ^b+1 and sets microcontroller flags
func Sub(a uint8, b uint8, mc *microcontroller) uint8 {
	// A-B
	// Do subtraction using two's complement and addition
	return Add(a, ^b+1, mc)
}

// Add : Adds two 1-byte values together and sets microcontrolle flags
func Add(a uint8, b uint8, mc *microcontroller) uint8 {
	// Do bitwise addition
	result16 := uint16(a) + uint16(b)
	result8 := uint8(result16)

	//fmt.Printf("A: %16b\nB: %16b\n   %16b\n", uint16(a), uint16(^b)+1, result16)
	mc.zero = result8 == 0x0
	mc.carry = result16&0x100 == 0x100
	mc.auxCarry = ((a&0xF)+(b&0xF))&0x10 == 0x10
	mc.sign = (result8 >> 7) == 0x1
	mc.parity = true // Because 0x0 has an even number of digits
	for i := uint8(0); i < 8; i++ {
		mc.parity = (((result8 >> i) & 0x1) == 0x1) != mc.parity
	}
	//fmt.Printf("Zero: %t, Carry: %t, auxCarry: %t, sign: %t, parity: %t\n", zero, carry, auxCarry, sign, parity)
	return result8
}
