package main

import "testing"
import "fmt"

// MathTest : A structure which has input & output values for a math test
type Operation int

const (
	subtraction Operation = 1
	addition    Operation = 2
)

type MathTest struct {
	op       Operation // Type of math we're doing
	a        uint8     // In
	b        uint8     // In
	result   uint8     // Out
	zero     bool      // Out: 1 if result is 0x0
	carry    bool      // Out: 1 if carry out of the MSB
	parity   bool      // Out: 1 if the number of 1-bits is even
	auxCarry bool      // Out: 1 if carry out of MSB nibble bit
	sign     bool      // Out: equal to the 7th bit (1 if negative)
}

//             operation,   a,   b  result, zero, carry, parity, half, sign
var subTests = []MathTest{
	MathTest{subtraction, 0x4A, 0x40, 0x0A, false, true, true, false, false},
	MathTest{subtraction, 0x1A, 0x0C, 0x0E, false, true, false, false, false},
	MathTest{addition, 0xAE, 0x74, 0x22, false, true, true, true, false},
	MathTest{addition, 0x2E, 0x74, 0xA2, false, false, false, true, true},
	MathTest{addition, 0xA7, 0x59, 0x00, true, true, true, true, false},
}

// TestSub : Run a series of subtraction math tests based on the subTests array above
func TestSub(t *testing.T) {
	for _, test := range subTests {
		mc := newMicrocontroller()
		var result uint8
		if test.op == subtraction {
			result = Sub(test.a, test.b, mc)
			fmt.Printf("%d - %d = %d\n", test.a, test.b, result)
		} else if test.op == addition {
			result = Add(test.a, test.b, mc)
			fmt.Printf("%d + %d = %d\n", test.a, test.b, result)
		}
		if result != test.result {
			t.Errorf("Result is incorrect. Expected: %X, Got %X", test.result, result)
		}
		if test.zero != mc.zero {
			t.Errorf("Zero bit is incorrect. Expected %t, Got %t", test.zero, mc.zero)
		}
		if test.carry != mc.carry {
			t.Errorf("Carry bit is incorrect. Expected %t, Got %t", test.carry, mc.carry)
		}
		if test.parity != mc.parity {
			t.Errorf("Parity bit is incorrect. Expected %t, Got %t", test.parity, mc.parity)
		}
		if test.auxCarry != mc.auxCarry {
			t.Errorf("AuxCarry bit is incorrect. Exepected %t, Got %t", test.auxCarry, mc.auxCarry)
		}
		if test.sign != mc.sign {
			t.Errorf("Sign bit is incorrect. Expected %t, Got %t", test.sign, mc.sign)
		}
	}
}
