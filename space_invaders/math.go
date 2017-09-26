package main

/*
Index: (((A & 0x88) >> 1) | ((VAL & 0x88) >> 2) | ((RESULT) & 0x88) >> 3)) & 0x7
Depending on the operation, RESULT may be either A+VAL or A+VAL+C
This is based on the implementation of i8080-core

 A|VAL|RES|INDEX|  Meaning
:--|:--|:--|:--|:--
 0 |  0|   0|   0|   A+Val < 8 (so no carry)
 0 |  0|   1|   0|   Both A&Val are <0x7, carry from 2 to 3 (but no half-carry)
 0 |  1|   0|   1|   There was a carry from 2 to 3 which caused a carry out of 3 (ie: 0111 + 1111 = 1 0000)
 0 |  1|   1|   0|   There was no carry from 2 to 3, so no carry out of 3
 1 |  0|   0|   1|   There was a carry from 2 to 3 which caused a carry out of 3 (ie: 1111 + 0111 = 1 0000)
 1 |  0|   1|   0|   There was no carry from 2 to 3, so n ocarry out of 3
 1 |  1|   0|   1|   There was a carry out of 3 (1000 + 1000 = 1 0000)
 1 |  1|   1|   1|   There was a carry into 3 and out of 3 (1100 + 1100 = 0 1000)

(((A & 0x88) >> 1) | ((VAL & 0x88) >> 2) | ((A+B) & 0x88) >> 3)) & 0x7

 A|  VAL| RES| HALF-CARRY (BORROW) | Meaning
 :--|:--|:--|:--|:--
 0  | 0 |  0 | 1 |  a >= 0; Val <  -8 and result is <-8           (ie:  0 - 9) => 0000 + 0111 =   0111 (8 or -9)
 0  | 0 |  1 | 0 |  a >= 0; Val <  -8 but the result is >= -8     (ie:  1 - 9) => 0001 + 0111 =   1000 (-8)
 0  | 1 |  0 | 0 |  a >= 0; val >= -8 and the result is above 0   (ie:  4 - 2) => 0100 + 1110 = 1 0010 ( 2)
 0  | 1 |  1 | 0 |  a >= 0; Val >= -8 and the result is above -8  (ie:  2 - 9) => 0010 + 0111 =   1001 (-7)
 1  | 0 |  0 | 1 |  a < 0 ; val <  -8 and the result is below -8  (ie: -1 - 9) => 1111 + 0111 = 1 0110 (-10)
 1  | 0 |  1 | 1 |  a < 0 ; val <  -8 and the result is below -17 (ie: -7 -15) => 1001 + 0001 = 0 1010 (-22)
 1  | 1 |  0 | 1 |  a < 0 ; val >= -8 but the result is below -8  (ie: -1 - 8) => 1111 + 1000 = 1 0111 (-9)
 1  | 1 |  1 | 0 |  a < 0 ; val >= -8 but the result is above -8  (ie: -1 - 7) => 1111 + 1001 = 1 0110 (+6)
*/

var addHalfCarryTable = []bool{false, false, true, false, true, false, true, true}
var subHalfCarryTable = []bool{true, false, false, false, true, true, true, false}

// Sub : Subtracts B from A and then sets micro controller flags
// the borrow argument is used by SBB, everything else should call it with a value of 0
func Sub(a uint8, b uint8, mc *microcontroller, borrow uint8) uint8 {
	result16 := uint16(a) - uint16(b) - uint16(borrow)
	result8 := uint8(result16)
	mc.zero = result8 == 0x0
	mc.sign = (result8 >> 7) == 0x1
	mc.parity = GetParity(result8)
	mc.carry = result16&0x100 > 0

	index := (((a & 0x88) >> 1) | ((b & 0x88) >> 2) | ((result8 & 0x88) >> 3)) & 0x7
	mc.auxCarry = subHalfCarryTable[index]
	return result8
}

// GetParity : Returns true if the number of 1-bits is even, false otherwise
func GetParity(value uint8) bool {
	returnValue := true // Because 0x0 has an even number of digits
	for i := uint8(0); i < 8; i++ {
		returnValue = (((value >> i) & 0x1) == 0x1) != returnValue
	}
	return returnValue
}

// Add : Adds two 1-byte values together and sets microcontroller flags
//       the carry flag is used by the ADC instructions
func Add(a uint8, b uint8, mc *microcontroller, carry uint8) uint8 {
	// Do bitwise addition
	result16 := uint16(a) + uint16(b) + uint16(carry)
	result8 := uint8(result16)

	mc.zero = result8 == 0x0
	mc.carry = result16&0x100 != 0x0 // The Carry bit is set when the result is positive (overflow)
	// the i8080-core emulator uses the halfcarry table because apparently the implementation of
	// the KR580VM80A is such that the half carry flag is calculated not based on A+VAL+C
	// but based on the 'magic' that happens in this half carry table
	index := (((a & 0x88) >> 1) | ((b & 0x88) >> 2) | ((result8 & 0x88) >> 3)) & 0x7
	mc.auxCarry = addHalfCarryTable[index]
	mc.sign = (result8 >> 7) == 0x1
	mc.parity = GetParity(result8)
	return result8
}
