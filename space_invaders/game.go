package main

import (
	"fmt"
	"image"
	"math"

	"github.com/hajimehoshi/ebiten"
)

// SCREENWIDTH - Resolution of the screen (columns)
var SCREENWIDTH = 224

// SCREENHEIGHT - Resolution of the screen (rows)
var SCREENHEIGHT = 256

// SCREENSCALE - How much to scale up the tv screen pixels to current monitor pixels
var SCREENSCALE = 2

// INSTRUCTIONSPERFRAME - How many instructions will be executed per frame of
// of the Ebiten render loop. That loop is "guaranteed" to run at 60FPS
// and the internet tells me that the 4000 instructions per frame is a good target speed
// to run at. This means that we'll call the middle/end of scanline interrupts twice per frame
var INSTRUCTIONSPERFRAME = 24000

// Game - A struct representing the SpaceInvaders game, the i8080, and a display/sound/input object
type Game struct {
	mc      *microcontroller
	sr      ShiftRegister
	display *Display
}

func newGame() *Game {
	game := new(Game)
	game.display = new(Display)
	return game
}

func (g *Game) in() {
	debugPrint(g.mc, "IN", 1)
	device := (*g.mc.memory)[g.mc.programCounter+1]
	switch device {
	case 0: // Hardware inputs that are never actually used in the code
		g.mc.ra = 0xFF
	case 1: // Button presses
		g.mc.ra = 0x8 // Temp value for no P1 buttons pressed, no credit, and a default "always on" pin
	case 2: // Game settings
		g.mc.ra = 0x00 // Temp value for no P2 button pressed, no tilt, extra ship at 1500, display coin info
	case 3: // Give the value in the shift register
		g.mc.ra = g.sr.getResult()
	}

	g.mc.programCounter += 2
}

func (g *Game) out() {
	debugPrint(g.mc, "OUT", 1)
	// Writes data to a connected device with the ID stored
	// in the immediate data
	device := (*g.mc.memory)[g.mc.programCounter+1]
	switch device {
	case 2: // Set shift amount (3 bits representing 8 values)
		g.sr.setOffset(g.mc.ra)
	case 3: // Sound bank 1
		// TODO: Implement me
	case 4: // Shift data
		g.sr.shiftData(g.mc.ra)
	case 5: // Sound bank 2
		// TODO: Implement me
	case 6: // Watchdog
		// Do nothing - this is used to pulse the watchdog
		// so that the i8080 does not reset (?)
	default:
		panic("ERROR: Output device does not exist")
	}
	g.mc.programCounter += 2
}

func (g *Game) tick() {
	instruction := (*g.mc.memory)[g.mc.programCounter]
	switch {
	case instruction == 0xD3:
		g.out()
	case instruction == 0xDB:
		g.in()
	default:
		g.mc.run()
	}

}

// Renders to the ebiten.Image which represents the display
// If the value of 'top' is true, renders the top 112 rows of the screen
// If false, render the bottom 112
func (g *Game) render(display *image.RGBA, top bool) error {

	startMemory := 0x2400
	startPixel := 0
	if !top {
		startMemory = 0x3200
		startPixel = 0xE00 * 8
	}

	for offset := 0; offset < 0xE00; offset++ {
		byte := (*g.mc.memory)[startMemory+offset]
		for shift := 0; shift < 8; shift++ {
			targetColor := uint8(0x0)
			if (byte>>uint32(shift))&0xFF > 0 {
				targetColor = 0xFF
			}

			//pixel := startPixel + offset*8 + int(7-shift)
			pixel := startPixel + offset*8 + int(shift)
			//fmt.Printf("Memory: %X, bit: %d, Drawing to pixel: %d value: %d\n", startMemory+offset, 7-shift, pixel, targetColor)
			display.Pix[4*pixel] = targetColor
			display.Pix[4*pixel+1] = targetColor
			display.Pix[4*pixel+2] = targetColor
			display.Pix[4*pixel+3] = targetColor
		}
	}

	return nil
}

func (g *Game) scanLine(scanline int) {
	if !g.mc.inte { // Interrupts are disabled
		return
	}
	switch scanline {
	case 96:
		g.mc.programCounter = 0x8
	case 224:
		g.mc.programCounter = 0x10
	default:
		panic("Unhandled scanline() call. 96 and 224 are the only valid values")
	}
}

func (g *Game) run() {
	// Starts the main event loop by booting up ebiten
	// displayData is given with width/height swapped since
	// the image will be rotated before being pasted
	//displayData := image.NewRGBA(image.Rect(0, 0, SCREENHEIGHT, SCREENWIDTH))
	displayData, err := ebiten.NewImage(SCREENHEIGHT, SCREENWIDTH, ebiten.FilterNearest)

	fmt.Println("run() start")

	if err != nil {
		fmt.Println("Error creating a displayData canvas")
	}

	f := func(screen *ebiten.Image) error {
		tmpImage := image.NewRGBA(image.Rect(0, 0, SCREENHEIGHT, SCREENWIDTH))
		fmt.Println("Starting to draw frame")
		for i := 0; i < INSTRUCTIONSPERFRAME/2; i++ {
			g.tick()
		}
		fmt.Println("Top render")
		g.render(tmpImage, true)
		fmt.Println("Scanline interrupt 96")
		g.scanLine(96)
		fmt.Println("Bottom render")
		g.render(tmpImage, false)
		fmt.Println("Scanline interrupt 224")
		g.scanLine(224)
		fmt.Println("Flipping buffers")
		displayData.ReplacePixels(tmpImage.Pix)
		opts := &ebiten.DrawImageOptions{}
		opts.GeoM.Rotate(-math.Pi / 2)
		opts.GeoM.Translate(0, float64(SCREENHEIGHT))
		screen.DrawImage(displayData, opts)
		return nil
	}

	fmt.Println("Starting ebiten.run()")

	runErr := ebiten.Run(f, SCREENWIDTH, SCREENHEIGHT, 3, "Space Invaders")
	fmt.Println("Exited run() with error: ", runErr)
}
