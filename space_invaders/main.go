package main

import (
	"flag"
	"fmt"
	"io"
	"net"
	"os"
)

// DEBUGMODE - Whether or not the program is running in debug mode (ie: pretty print opcodes)
var DEBUGMODE = true

// COMPAREFLAG - When set, the output of debugPrint() will match what is output
// by the modified i8080-core program so that both logs can be diff'd
var COMPAREFLAG = false

// CLIENTMODE - When set, the emulator will connect to a local server and output debug
// data to that server so that it may be compared to the output of another emulator
var CLIENTMODE = false

// TESTMODE - When set, starts execution at 0x100 and also hooks certain debug functionality
// like tracking jumps to 0x0 and CALLs to 0x05
var TESTMODE = false

var connection net.Conn

//var outputBuffer = ""
var readyToWriteFlag = false

// LINESTOWRITE - In client mode, how big of a buffer to create to store the program state
// before sending it to the server for comparion
var LINESTOWRITE = 1024 * 1024

// BYTESPERLINE - In client mode, how many bytes there are in the standard output
var BYTESPERLINE = 49
var outputBuffer = make([]byte, LINESTOWRITE*BYTESPERLINE)
var outputBufferLines = 0

func debugPrintHeader(mc *microcontroller) {
	if mc.instructionsExecuted%20 == 0 {
		fmt.Printf("ADDR : instruction\t\t\tB  C  D  E  H  L  A  SZ-X-P-C PW SP\n")
	}
}

func debugPrint(mc *microcontroller, name string, values uint16) {
	if DEBUGMODE || CLIENTMODE {
		if !COMPAREFLAG && !CLIENTMODE { // Do not print headers in compare output mode or in client mode
			debugPrintHeader(mc)
		}
		// Prints out the opcode and the immediate data (if any) based on values
		// passed to this function
		output := ""
		if COMPAREFLAG || CLIENTMODE { // In compare output mode, print an easily computer parseable string
			// PC, OPCODE, 2BYTES that follow the opcode
			output += fmt.Sprintf("%04X %02X %02X %02X ", mc.programCounter,
				(*mc.memory)[mc.programCounter],
				(*mc.memory)[mc.programCounter+1],
				(*mc.memory)[mc.programCounter+2])
		} else {
			output += fmt.Sprintf("%04X : %02X", mc.programCounter, (*mc.memory)[mc.programCounter])
			for i := uint16(1); i < values+1; i++ {
				output += fmt.Sprintf(" %02X", (*mc.memory)[mc.programCounter+i])
			}
			output += fmt.Sprintf("\t\t %-15s", name)
		}
		//   rb  rc   rd   re   rh   rl   ra   psw
		output += fmt.Sprintf("%02X %02X %02X %02X %02X %02X %02X %08b %04X\n",
			mc.rb, mc.rc, mc.rd, mc.re, mc.rh, mc.rl, mc.ra, pswByte(mc), mc.stackPointer)

		if CLIENTMODE {
			//outputBuffer += output
			for i := 0; i < len(output); i++ {
				//index := BYTES_PER_LINE*outputBufferLines + i
				//fmt.Printf("Output size: %d || String index: %d || Buffer index: %d || String character: %c", len(output), i, index, output[i])
				//fmt.Println(output[i])
				outputBuffer[BYTESPERLINE*outputBufferLines+i] = output[i]
			}
			outputBufferLines++
		} else {
			fmt.Print(output)
		}
	}
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

func loadSpaceInvaders() []uint8 {
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

func memoryDump(mc *microcontroller, size uint16) {
	if COMPAREFLAG { // In compare output mode, do not do a memory dump
		return
	}
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

func conout(mc *microcontroller) {
	if COMPAREFLAG { // Output nothing during CPU state comparison
		return
	}
	if mc.rc == 9 {
		start := (uint16(mc.rd) << 8) | uint16(mc.re)
		message := string("")
		for i := start; (*mc.memory)[i] != '$'; i++ {
			message += string((*mc.memory)[i])
		}
		fmt.Printf("CONOUT (9): %s\n", message)
	} else if mc.rc == 2 {
		if DEBUGMODE {
			fmt.Printf("CONOUT (2): %s\n", string(mc.re))
		} else {
			fmt.Printf(string(mc.re)) // No carriage return
		}
	}
}

func connect(fileName string) {
	conn, err := net.Dial("tcp", "localhost:5679")
	if err != nil {
		fmt.Printf("Could not connect to the local debug server at localhost:5679")
	}
	identifyString := fmt.Sprintf("IDENTIFY 8080-golang %s\n", fileName)
	conn.Write([]byte(identifyString))
	connection = conn
}

func readyToWrite() bool {
	if !CLIENTMODE || readyToWriteFlag {
		return true
	}

	buffer := make([]byte, 1024)
	fmt.Printf("Waiting for a write flag\n")
	n, err := connection.Read(buffer)

	if err != nil {
		fmt.Printf("Error while reading from server: %s", err)
		panic("Error while reading from server")
	}
	//fmt.Printf("Got data from server: %s", string(buffer))
	if n > 0 && buffer[0] == byte('W') { // "W" flag from server means that it's ok to go ahead and write
		readyToWriteFlag = true
	}
	return readyToWriteFlag
}

func writeRemoteOutput() {
	if !CLIENTMODE {
		return
	}

	if outputBufferLines >= LINESTOWRITE {
		n, err := connection.Write(outputBuffer)
		if n != len(outputBuffer) {
			fmt.Println("Could not write the entire buffer")
			panic("Buffer not written")
		}
		if err != nil {
			fmt.Printf("Error writing to buffer: %s", err)
		}
		//outputBuffer = ""
		outputBufferLines = 0
		readyToWriteFlag = false
	}
}

func finalWrite() {
	// Called to just dump whatever is in the buffer at the present
	// at the end of the ROM execution in order to a comparison
	// of the remaining data
	remainingBuffer := outputBuffer[0 : outputBufferLines*BYTESPERLINE]
	n, err := connection.Write(remainingBuffer)
	if n != len(outputBuffer) {
		fmt.Println("Could not write the entire buffer")
		panic("Buffer not written")
	}
	if err != nil {
		fmt.Printf("Error writing to buffer: %s", err)
	}
}

// Runs the emulator in test mode for one instruction
// Checks jumps to 0x0 and also calls to 0x5 (print to scree)
func runTestROM() {
	emulation := newMicrocontroller()
	args := os.Args[1:]
	if len(args) == 0 {
		fmt.Printf("%s <program> - Runs the test program <program>", os.Args[0])
		return
	}

	romName := args[len(args)-1]

	if CLIENTMODE {
		connect(romName)
	}

	rom := []uint8{}

	rom = loadTestROM(romName)
	emulation.programCounter = 0x100 // Hardcoded because the test ROMs start at 0x100
	emulation.memory = &rom
	(*emulation.memory)[5] = 0xC9 // Call RET after handling CALL 5 (call conout)

	for {
		startAddress := emulation.programCounter
		readyToWrite()
		emulation.run()
		writeRemoteOutput()

		if emulation.programCounter == 0 {
			fmt.Printf("OUTPUT: Jump to 0x0 from %04X\n", startAddress)
			if DEBUGMODE {
				memoryDump(emulation, 0x400)
			}
			if CLIENTMODE {
				finalWrite()
			}
			break
		} else if emulation.programCounter == 0x5 { // Error function was called
			conout(emulation)
		}
	}

}

func runSpaceInvaders() {
	spaceInvaders := new(game)
	spaceInvaders.mc = newMicrocontroller()
	rom := loadSpaceInvaders()
	spaceInvaders.mc.memory = &rom
	for {
		spaceInvaders.run()
	}
}

func main() {
	// Parse command line flags
	verboseFlag := flag.Bool("v", true, "Show every instruction being executed (slow)")
	compareFlag := flag.Bool("c", false, "Instructions are output in the format of the i8080-core emulator")
	serverFlag := flag.Bool("s", false, "Connect to a local server and write debug data to it")
	testFlag := flag.Bool("t", false, "Start program in i8080-core test rom mode")
	flag.Parse()

	COMPAREFLAG = *compareFlag
	DEBUGMODE = *verboseFlag
	CLIENTMODE = *serverFlag
	TESTMODE = *testFlag

	if TESTMODE {
		runTestROM()
	} else {
		runSpaceInvaders()
	}
}
