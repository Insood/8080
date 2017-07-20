package main

import (
	"fmt"
	"io"
	"os"
)

type microcontroller struct {
	rb, rc, rd, re, rh, rl, ra uint8 // Seven working registers
	programCounter             uint16
	stackPointer               uint16
	memory                     *[]uint8
}

func (mc *microcontroller) data16bit() uint16 {
	// This functions creates a 16-bit value from the low & high bits
	// of the currently active instruction. This is used in many places
	// <instruction> <low bits> <high bits> -> returns (high << 8) | low
	return uint16((*mc.memory)[mc.programCounter+1]) | uint16((*mc.memory)[mc.programCounter+2])<<8
}

func (mc *microcontroller) nop() {
	// 0x0: NOP - Do nothing
	// a place to hook in other instructions
	mc.programCounter++
}

func (mc *microcontroller) jmp() {
	// 0xC3: JMP <low bits><high bits> - Set the program counter to the new address
	mc.programCounter = mc.data16bit()
}

func (mc *microcontroller) lxi() {
	// 0x01, 0x11, 0x21, 0x31 <low data> <high data>
	// Based on the 3rd & 4th most significant bits, set the low/high data
	// To specific registers in memory.
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
	target := ((*mc.memory)[mc.programCounter] & 0x38) >> 3
	data := (*mc.memory)[mc.programCounter+1]
	switch target {
	case 0:
		mc.rb = data
	case 1:
		mc.rc = data
	case 2:
		mc.rd = data
	case 3:
		mc.re = data
	case 4:
		mc.rh = data
	case 5:
		mc.rl = data
	case 6:
		// Memory address addressed by the H/L registers
		// H (high), L (low)
		memRef := (uint16(mc.rh) << 8) | (uint16(mc.rl))
		(*mc.memory)[memRef] = data
	case 7:
		mc.ra = data
	}

	mc.programCounter += 2
}

func (mc *microcontroller) call() {

}

func (mc *microcontroller) run() {
	instruction := (*mc.memory)[mc.programCounter]
	fmt.Printf("%X : %X\n", mc.programCounter, instruction)
	switch instruction {
	case 0x00:
		mc.nop()
	case 0xC3:
		mc.jmp()
	case 0x01, 0x11, 0x21, 0x31:
		mc.lxi()
	case 0x06, 0x16, 0x26, 0x36, 0x0E, 0x1E, 0x2E, 0x3E:
		mc.mvi()
	case 0xCD:
		mc.call()
	default:
		err := fmt.Sprintf("[%d] Unknown instruction: %X ", mc.programCounter, instruction)
		fmt.Println(err)
		panic(err)
	}
}

func loadROM() []uint8 {
	files := []string{"invaders_h.rom", "invaders_g.rom", "invaders_f.rom", "invaders_e.rom"}
	memory := make([]uint8, 0, 65536)
	for _, file := range files {
		fi, err := os.Open(file)
		if err != nil {
			panic(err)
		}
		buf := make([]byte, 1024)
		for {
			bytesRead, error := fi.Read(buf)
			slice := buf[0:bytesRead]
			memory = append(memory, slice...) // The ... mean to expand the second argument

			if error == io.EOF {
				break
			}
		}

		fi.Close()
	}
	emptyRAM := make([]uint8, cap(memory)-len(memory))
	memory = append(memory, emptyRAM...)
	return memory
}

func main() {
	spaceInvaders := new(microcontroller)
	rom := loadROM()
	spaceInvaders.memory = &rom
	for {
		spaceInvaders.run()
	}
	/*	outFile, err := os.OpenFile("rom.out", os.O_CREATE|os.O_APPEND|os.O_RDWR, 666)
			if err != nil {
				print(err)
				panic(err)
			}
		    outFile.Write(rom)
	*/
}
