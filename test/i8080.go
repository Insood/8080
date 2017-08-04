package main

import (
	"fmt"
	"io"
	"os"
)

type microcontroller struct {
	rb, rc, rd, re, rh, rl, ra uint8 // Seven working registers
	rarray                     []*uint8
	programCounter             uint16
	stackPointer               uint16
	memory                     *[]uint8
	zero                       bool
	sign                       bool
	parity                     bool
	carry                      bool
	auxCarry                   bool
	instructionsExecuted       int64 // Not a part of the microcontroller spec
}

func pswByte(mc *microcontroller) uint8 {
	var data uint8 = 0x2 // For some reason bit 1 is always 1
	if mc.sign {
		data |= (0x1 << 7)
	}
	if mc.zero {
		data |= (0x1 << 6)
	}
	if mc.auxCarry {
		data |= (0x1 << 4)
	}
	if mc.parity {
		data |= (0x1 << 2)
	}
	if mc.carry {
		data |= 0x1
	}

	return data
}

func debugPrintHeader(mc *microcontroller) {
	if mc.instructionsExecuted%20 == 0 {
		fmt.Printf("ADDR : instruction\t\t\tB  C  D  E  H  L  A  SZ-X-P-C PW SP\n")
	}
}

func debugPrint(mc *microcontroller, name string, values uint16) {
	debugPrintHeader(mc)
	// Prints out the opcode and the immediate data (if any) based on values
	// passed to this function
	fmt.Printf("%04X : %02X", mc.programCounter, (*mc.memory)[mc.programCounter])
	for i := uint16(1); i < values+1; i++ {
		fmt.Printf(" %02X", (*mc.memory)[mc.programCounter+i])
	}
	fmt.Printf("\t\t %-15s", name)
	//   rb  rc   rd   re   rh   rl   ra   psw
	fmt.Printf("%02X %02X %02X %02X %02X %02X %02X %08b %02X %04X\n",
		mc.rb, mc.rc, mc.rd, mc.re, mc.rh, mc.rl, mc.ra, pswByte(mc), pswByte(mc), mc.stackPointer)
}

func newMicrocontroller() *microcontroller {
	mc := new(microcontroller)
	// the 7th element is nil because some instructions have a memory reference
	// bit pattern which corresponds to 110B
	mc.rarray = []*uint8{&mc.rb, &mc.rc, &mc.rd, &mc.re, &mc.rh, &mc.rl, nil, &mc.ra}
	return mc
}

func (mc *microcontroller) data16bit() uint16 {
	// This functions creates a 16-bit value from the low & high bits
	// of the currently active instruction. This is used in many places
	// <instruction> <low bits> <high bits> -> returns (high << 8) | low
	return uint16((*mc.memory)[mc.programCounter+1]) | uint16((*mc.memory)[mc.programCounter+2])<<8
}

func (mc *microcontroller) memoryReference() uint16 {
	// Lots of instructions refer to a memory reference which is the address
	// stored in the H/L registers. The address is (H << 8) & (L)
	// H for high, L for low!
	return (uint16(mc.rh) << 8) | (uint16(mc.rl))
}

// OP-Codes, arranged alphabetically (in the future)

func (mc *microcontroller) ani() {
	// AND immediate with accumulator
	data := (*mc.memory)[mc.programCounter+1]
	debugPrint(mc, "ANI", 1)
	mc.ra = mc.ra & data
	mc.carry = false // Because of the specification
	mc.zero = (mc.ra == 0)
	mc.sign = (mc.ra & 0x8) > 0
	mc.parity = GetParity(mc.ra)
	//mc.auxCarry is not affected per the 8080 programmer's manual
	mc.programCounter += 2
}

func (mc *microcontroller) nop() {
	debugPrint(mc, "NOP", 0)
	// 0x0: NOP - Do nothing
	// a place to hook in other instructions
	mc.programCounter++
}

func (mc *microcontroller) jc() {
	// Jump if carry
	debugPrint(mc, "JC", 2)
	if mc.carry {
		mc.jmp()
	} else {
		mc.programCounter += 3
	}
}

func (mc *microcontroller) jmp() {
	// 0xC3: JMP <low bits><high bits> - Set the program counter to the new address
	debugPrint(mc, "JMP", 2)
	mc.programCounter = mc.data16bit()
}

// jz : Jump if zero bit is 1
func (mc *microcontroller) jz() {
	debugPrint(mc, "JZ", 2)
	if mc.zero {
		mc.jmp()
	} else {
		mc.programCounter += 3
	}
}

// jnz : Jump if zero bit is 0
func (mc *microcontroller) jnz() {
	debugPrint(mc, "JNZ", 2)
	if mc.zero {
		mc.programCounter += 3
	} else {
		mc.jmp()
	}
}

// jnc : Jump if Carry bit is zero
func (mc *microcontroller) jnc() {
	debugPrint(mc, "JNC", 2)
	if mc.carry {
		mc.programCounter += 3
	} else { // No carry so jump
		mc.jmp()
	}
}

func (mc *microcontroller) lxi() {
	// 0x01, 0x11, 0x21, 0x31 <low data> <high data>
	// Based on the 3rd & 4th most significant bits, set the low/high data
	// To specific registers in memory.
	debugPrint(mc, "LXI", 2)
	target := ((*mc.memory)[mc.programCounter] & 0x30) >> 4
	low := (*mc.memory)[mc.programCounter+1]
	high := (*mc.memory)[mc.programCounter+2]
	switch target {
	case 0x0: // Registers B, C
		mc.rb = high
		mc.rc = low
	case 0x1: // Registers D, E
		mc.rd = high
		mc.re = low
	case 0x2: // Registers H, L
		mc.rh = high
		mc.rl = low
	case 0x3: // Register sp
		mc.stackPointer = mc.data16bit()
	}
	mc.programCounter += 3
}

func (mc *microcontroller) mvi() {
	// (0x06, 0x16, 0x26, 0x36, 0x0E, 0x1E, 0x2E, 0x3E) <data>
	// Sets <data> to the register encoded within the instruction
	debugPrint(mc, "MVI", 1)
	target := ((*mc.memory)[mc.programCounter] & 0x38) >> 3
	data := (*mc.memory)[mc.programCounter+1]
	if target == 6 {
		(*mc.memory)[mc.memoryReference()] = data
	} else {
		*(mc.rarray[target]) = data
	}
	mc.programCounter += 2
}

func (mc *microcontroller) call() {
	debugPrint(mc, "CALL", 2)
	target := mc.data16bit()
	//pcHigh := uint8(mc.stackPointer >> 8)
	//pcLow := uint8(mc.stackPointer & 0xFF)
	next := mc.programCounter + 3                        // The instruction after the CALL
	(*mc.memory)[mc.stackPointer-2] = uint8(next & 0xFF) // LSB
	(*mc.memory)[mc.stackPointer-1] = uint8(next >> 8)   // MSB
	mc.stackPointer -= 2
	mc.programCounter = target
}

func (mc *microcontroller) cc() {
	debugPrint(mc, "CC", 2)
	// Call if Carry bit is 1
	if mc.carry {
		mc.call()
	} else {
		mc.programCounter += 3
	}
}

func (mc *microcontroller) cnc() {
	// Call if No Carry
	debugPrint(mc, "CNC", 2)
	if mc.carry {
		mc.programCounter += 3
	} else {
		mc.call()
	}
}

func (mc *microcontroller) cpe() {
	// Call if Parity is Even
	debugPrint(mc, "CPE", 2)
	if mc.parity { // parity==1 is even
		mc.call()
	} else {
		mc.programCounter += 3
	}
}

func (mc *microcontroller) push() {
	cmd := ((*mc.memory)[mc.programCounter] >> 4) & 0x3
	cmdMap := []string{"BC", "DE", "HL", "PSW"}
	cmdStr := fmt.Sprintf("PUSH %s", cmdMap[cmd])
	debugPrint(mc, cmdStr, 0)
	var first, second uint8
	switch cmd {
	case 0x0: // B & C
		first = mc.rb
		second = mc.rc
	case 0x1: // D & E
		first = mc.rd
		second = mc.re
	case 0x2: // H & L
		first = mc.rh
		second = mc.rl
	case 0x3: // flags & A
		first = mc.ra
		second = pswByte(mc)
	}
	(*mc.memory)[mc.stackPointer-2] = second
	(*mc.memory)[mc.stackPointer-1] = first
	mc.stackPointer -= 2
	mc.programCounter++
}

func (mc *microcontroller) halt() {
	print("halt() is not yet implemented")
	panic("Not yet implemented")
}

func (mc *microcontroller) ldax() {
	// 0x0A, 0x1A : LDAX (no other data)
	// Load the contents of the memory address either in B/C or D/E
	// into the Accumulator
	debugPrint(mc, "LDAX", 0)
	instruction := ((*mc.memory)[mc.programCounter] >> 4) & 1
	var low, high uint8
	switch instruction {
	case 0x0:
		low = mc.rc  // C
		high = mc.rb // B
	case 0x1:
		low = mc.re  // E
		high = mc.rd // D
	}
	address := (uint16(high) << 8) | uint16(low)
	mc.ra = (*mc.memory)[address]
	mc.programCounter++
}

func (mc *microcontroller) mov() {
	letterMap := string("BCDEHLMA")

	dst := ((*mc.memory)[mc.programCounter] >> 3) & 0x7 // Bits 4-6
	src := (*mc.memory)[mc.programCounter] & 0x7        // Lowest 3 bits
	str := fmt.Sprintf("MOV %s%s", string(letterMap[dst]), string(letterMap[src]))
	debugPrint(mc, str, 0)

	var target *uint8
	if dst == 6 { // Memory reference
		target = &(*mc.memory)[mc.memoryReference()] // address of array element
	} else {
		target = mc.rarray[dst]
	}

	var data uint8
	if src == 6 {
		data = (*mc.memory)[mc.memoryReference()]
	} else {
		data = *(mc.rarray[src])
	}
	*target = data

	mc.programCounter++
}

func (mc *microcontroller) cpi() {
	// 0xFE: CPI <data>
	// Compare immediate with accumulator - compares the byte of immediate data
	// with the accumulator using subtraction (A - data) and sets some flags
	debugPrint(mc, "CPI", 1)
	data := (*mc.memory)[mc.programCounter+1]
	Sub(mc.ra, data, mc)
	//fmt.Printf("Comparing %02X to %02X\n", mc.ra, data)
	mc.programCounter += 2
}

func (mc *microcontroller) inx() {
	// Increment Register Pair
	// 00: BC, 01: DE, 10: HL, 11: SP
	debugPrint(mc, "INX", 0)
	target := ((*mc.memory)[mc.programCounter] >> 4) & 0x3
	switch target {
	case 0: // BC
		value := ((uint16(mc.rb) << 8) | uint16(mc.rc)) + 1
		mc.rb = uint8(value >> 8)
		mc.rc = uint8(value & 0xFF)
	case 1: // DE
		value := ((uint16(mc.rd) << 8) | uint16(mc.re)) + 1
		mc.rd = uint8(value >> 8)
		mc.re = uint8(value & 0xFF)
	case 2: // HL
		value := ((uint16(mc.rh) << 8) | uint16(mc.rl)) + 1
		mc.rh = uint8(value >> 8)
		mc.rl = uint8(value & 0xFF)
	case 3: // SP
		mc.programCounter++
	}
	mc.programCounter++
}
func (mc *microcontroller) pop() {
	target := ((*mc.memory)[mc.programCounter] >> 4) & 0x3
	low := (*mc.memory)[mc.stackPointer]
	high := (*mc.memory)[mc.stackPointer+1]
	switch target {
	case 0: // BC
		mc.rb = high
		mc.rc = low
		debugPrint(mc, "POP BC", 0)
	case 1: // DE
		mc.rd = high
		mc.re = low
		debugPrint(mc, "POP DE", 0)
	case 2: // HL
		mc.rh = high
		mc.rl = low
		debugPrint(mc, "POP HL", 0)
	case 3: // flags & A (POP PSW)
		mc.sign = ((low >> 7) & 0x1) == 0x1
		mc.zero = ((low >> 6) & 0x1) == 0x1
		mc.auxCarry = ((low >> 4) & 0x1) == 0x1
		mc.carry = (low & 0x1) == 0x1 // LSB
		mc.parity = ((low >> 2) & 0x1) == 0x1
		mc.ra = high
		debugPrint(mc, "POP PSW", 0)
	}
	mc.programCounter++
	mc.stackPointer += 2
}

func (mc *microcontroller) ret() {
	low := uint16((*mc.memory)[mc.stackPointer])
	high := uint16((*mc.memory)[mc.stackPointer+1])
	target := (high << 8) | low
	debugPrint(mc, fmt.Sprintf("RET %04X", target), 0)
	mc.programCounter = target
	mc.stackPointer += 2
}

func (mc *microcontroller) retC() {
	// Return if Carry. Called ret_c because there is already an mc.rc
	debugPrint(mc, "RC", 0)
	if mc.carry {
		mc.ret()
	} else {
		mc.programCounter++
	}
}

func (mc *microcontroller) rnc() {
	// Return it NOT Carry
	debugPrint(mc, "RNC", 0)
	if mc.carry {
		mc.programCounter++
	} else {
		mc.ret()
	}
}

func (mc *microcontroller) rrc() {
	debugPrint(mc, "RRC", 0)
	lowBit := mc.ra & 0x1
	// Set the carry bit equal to the LSB
	if lowBit == 0x1 {
		mc.carry = true
	} else {
		mc.carry = false
	}
	mc.ra = (mc.ra >> 1) | (lowBit << 7)
	mc.programCounter++
}

func (mc *microcontroller) lda() {
	debugPrint(mc, "LDA", 2)
	// Load Accummulator Direct <low> <high>
	mc.ra = (*mc.memory)[mc.data16bit()]
	mc.programCounter += 3
}

func (mc *microcontroller) run() {
	instruction := (*mc.memory)[mc.programCounter]
	switch {
	case instruction == 0x00:
		mc.nop()
	case instruction == 0xC3:
		mc.jmp()
	case instruction == 0xDA:
		mc.jc()
	case instruction == 0xCA:
		mc.jz()
	case instruction == 0xC2:
		mc.jnz()
	case instruction == 0xD2:
		mc.jnc()
	//  00DD0001 in binary
	//  0x01, 0x11, 0x21, 0x31:
	case instruction&0xCF == 0x1:
		mc.lxi()
	// 00DDD110 in binary
	// 0x06, 0x16, 0x26, 0x36, 0x0E, 0x1E, 0x2E, 0x3E:
	case instruction&0xC7 == 0x6:
		mc.mvi()
	case instruction == 0xCD:
		mc.call()
	case instruction == 0xDC:
		mc.cc()
	case instruction == 0xD4:
		mc.cnc()
	case instruction == 0xEC:
		mc.cpe()
	case (instruction == 0x0A) || (instruction == 0x1A):
		mc.ldax()
	// This need to be above all the other 0x7x instructions
	// since it's the only one that doesn't follow the MOV pattern
	case instruction == 0x76:
		mc.halt()
	case (instruction >> 6) == 0x01:
		mc.mov()
	case instruction == 0xFE:
		mc.cpi()
	case (instruction & 0xCF) == 0x3:
		// 0x03, 0x13, 0x23, 0x33
		mc.inx()
	case instruction&0xCF == 0xC1:
		// 0xC1, 0xD1, 0xE1, 0xF1
		mc.pop()
	case instruction&0xCF == 0xC5:
		// 0xC5, 0xD5, 0xE5, 0xF5
		mc.push()
	case instruction == 0x3A:
		mc.lda()
	case instruction == 0xC9:
		mc.ret()
	case instruction == 0xD8:
		mc.retC()
	case instruction == 0xD0:
		mc.rnc()
	case instruction == 0xE6:
		mc.ani()
	case instruction == 0x0F:
		mc.rrc()
	default:
		err := fmt.Sprintf("[%d] Unknown instruction: %X ", mc.programCounter, instruction)
		fmt.Println(err)
		panic(err)
	}
	mc.instructionsExecuted++
}

func loadTestROM(romName string) []uint8 {
	fi, err := os.Open(romName)
	if err != nil {
		fmt.Println(romName, "is an invalid file. Could not open.")
		panic(err)
	}

	memory := make([]uint8, 0, 65536)
	testOffset := make([]uint8, 0x100) // The i8080-core test roms starts execution at 0x100
	memory = append(memory, testOffset...)
	buf := make([]byte, 1024)
	for {
		bytesRead, error := fi.Read(buf)
		slice := buf[0:bytesRead]
		memory = append(memory, slice...) // The ... mean to expand the second argument

		if error == io.EOF {
			break
		}
	}
	emptyRAM := make([]uint8, cap(memory)-len(memory))
	memory = append(memory, emptyRAM...)
	return memory
}

func memoryDump(mc *microcontroller, size uint16) {
	// Dumps the memory to console - up to size bytes
	address := uint16(0)
	headerStr := string("       ")
	for i := 0; i < 16; i++ {
		headerStr += fmt.Sprintf("%2X ", i)
	}
	fmt.Println(headerStr)
	fmt.Printf("-------------------------------------------------------\n")
	for address < size {
		str := string("")
		for i := 0; i < 16; i++ {
			str += fmt.Sprintf("%02X ", (*mc.memory)[address])
			address++
		}
		fmt.Printf("%04X : %s\n", address-16, str)
	}

}
func main() {
	emulation := newMicrocontroller()
	args := os.Args[1:]
	if len(args) == 0 {
		fmt.Printf("%s <program> - Runs the test program <program>", os.Args[0])
	} else {
		rom := loadTestROM(args[0])
		emulation.programCounter = 0x100 // Hardcoded because the test ROMs start at 0x100
		emulation.memory = &rom
	}

	for {
		startAddress := emulation.programCounter
		emulation.run()
		if emulation.programCounter == 0 {
			fmt.Printf("Jump to 0x0 from %04X\n", startAddress)
			memoryDump(emulation, 0x400)
			break
		}
	}
}
