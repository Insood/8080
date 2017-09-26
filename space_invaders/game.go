package main

type game struct {
	mc *microcontroller
	sr ShiftRegister
}

func (g *game) out() {
	// Writes data to a connected device with the ID stored
	// in the immediate data
	device := (*g.mc.memory)[g.mc.programCounter+1]
	switch device {
	case 2: // Set shift amount (3 bits representing 8 values)
		g.sr.setOffset(g.mc.ra)
	case 3: // Sound bank 1
		// TODO: Implement me
	case 5: // Sound bank 2
		// TODO: Implement me
	case 6: // Watchdog
		// Do nothing - this is used to pulse the watchdog
		// so that the i8080 does not reset (?)
	default:
		panic("ERROR: Output device does not exist")
		//case 4: // data
	}
	g.mc.programCounter += 2

}

func (g *game) run() {
	instruction := (*g.mc.memory)[g.mc.programCounter]
	switch {
	case instruction == 0xD3:
		g.out()
	default:
		g.mc.run()
	}
}
