package main

import (
	"errors"
	"fmt"
	"image"
	"math"
	"os"

	"github.com/hajimehoshi/ebiten/audio"

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
var INSTRUCTIONSPERFRAME = 4000

// Game - A struct representing the SpaceInvaders game, the i8080, and a display/sound/input object
type Game struct {
	mc   *microcontroller
	sr   ShiftRegister
	dip4 bool // Some sort of self-test-request
	dip3 bool // number of ships 00 = 3 10 = 5
	dip5 bool // number of ships 01 = 4 11 = 6
	dip6 bool // 0 = extra ship at 1500, 1 = extra ship at 1000
	dip7 bool // 0 = display coin info on demo screen, 1=don't?

	// The below are used to store the states of the last keypress state
	// for keyboard buttons 3-7 to control the dip switches
	lastKeyState map[ebiten.Key]bool

	// stuff for playing sounds
	audioContext *audio.Context
	soundBoard   map[string]*audio.Player
}

func loadSound(context *audio.Context, fileName string) *audio.Player {
	file, err := os.Open(fileName)
	if err != nil {
		panic(fmt.Sprintf("Error while opening '%s'. The error is: %s", fileName, err))
	}
	player, err := audio.NewPlayer(context, file)
	if err != nil {
		panic(fmt.Sprintf("Error while creating a Player for '%s'. The error is: %s", fileName, err))
	}
	return player
}

func newGame() *Game {
	game := new(Game)
	game.dip4 = true
	game.lastKeyState = make(map[ebiten.Key]bool)
	game.lastKeyState[ebiten.Key3] = false
	game.lastKeyState[ebiten.Key4] = true
	game.lastKeyState[ebiten.Key5] = false
	game.lastKeyState[ebiten.Key6] = false
	game.lastKeyState[ebiten.Key7] = false
	context, err := audio.NewContext(44100)
	game.audioContext = context
	if err != nil {
		str := fmt.Sprintf("Error creating audio context: %s", err)
		panic(str)
	}
	game.soundBoard = make(map[string]*audio.Player)
	game.soundBoard["explosion"] = loadSound(context, "sounds/explosion.wav")
	game.soundBoard["fastinvader1"] = loadSound(context, "sounds/fastinvader1.wav")
	game.soundBoard["fastinvader2"] = loadSound(context, "sounds/fastinvader2.wav")
	game.soundBoard["fastinvader3"] = loadSound(context, "sounds/fastinvader3.wav")
	game.soundBoard["fastinvader4"] = loadSound(context, "sounds/fastinvader4.wav")

	game.soundBoard["invaderkilled"] = loadSound(context, "sounds/invaderkilled.wav")
	game.soundBoard["shoot"] = loadSound(context, "sounds/shoot.wav")

	game.soundBoard["ufo_highpitch"] = loadSound(context, "sounds/ufo_highpitch.wav")
	game.soundBoard["ufo_lowpitch"] = loadSound(context, "sounds/ufo_lowpitch.wav")

	return game
}

// There does not appear to be a DB 00 instruction
// anywhere in the space invaders code
// so this code is here just for completeness sake
func (g *Game) inPort0() uint8 {
	data := uint8(0x8E) // 1xxx111x
	if g.dip4 {         // Is DIP switch 4 on?
		data |= 0x1
	}
	// It's not clear if the implementation below is correct
	// The fire/left/right buttons may be an AND of both switches
	// but again, DB 00 is not called anywhere AFAIK
	if ebiten.IsKeyPressed(ebiten.KeySpace) { // 1P Fire button pressed?
		data |= 0x1 << 4
	}
	if ebiten.IsKeyPressed(ebiten.KeyLeft) { // 1P Left button pressed
		data |= 0x1 << 5
	}
	if ebiten.IsKeyPressed(ebiten.KeyRight) { // 1P Right button pressed
		data |= 0x1 << 6
	}
	return data
}

func (g *Game) inPort1() uint8 {
	data := uint8(0x8)                        // Bit 3 is always set. Bit 7 is never set
	if ebiten.IsKeyPressed(ebiten.KeyEnter) { // Deposit a credit
		data |= 0x1
	}
	if ebiten.IsKeyPressed(ebiten.Key2) { // 2P Start Button
		data |= 0x1 << 1
	}
	if ebiten.IsKeyPressed(ebiten.Key1) { // 1P Start Button
		data |= 0x1 << 2
	}
	if ebiten.IsKeyPressed(ebiten.KeySpace) { // 1P fire
		data |= 0x1 << 4
	}
	if ebiten.IsKeyPressed(ebiten.KeyLeft) { // 1P left
		data |= 0x1 << 5
	}
	if ebiten.IsKeyPressed(ebiten.KeyRight) { // 1P right
		data |= 0x1 << 6
	}
	return data
}

func (g *Game) inPort2() uint8 {
	data := uint8(0x0)
	if g.dip3 { // Extra ships, 00=3, 10 = 5
		data |= 0x1
	}
	if g.dip5 { // Extra ships, 01 = 4, 11 = 6
		data |= (0x1 << 1)
	}
	if ebiten.IsKeyPressed(ebiten.KeyT) { // Tilt
		data |= (0x1 << 2)
	}
	if g.dip6 { // Extra ship, 0x0 -> @1500, 0x1 @ 1000 pts
		data |= (0x1 << 3)
	}
	if ebiten.IsKeyPressed(ebiten.KeyW) { // P2 fire
		data |= (0x1 << 4)
	}
	if ebiten.IsKeyPressed(ebiten.KeyA) { // P2 Left
		data |= (0x1 << 5)
	}
	if ebiten.IsKeyPressed(ebiten.KeyD) { // P2 Right
		data |= (0x1 << 6)
	}
	if g.dip7 { // Display coin info; 0x0: on, 0x1: off?
		data |= (0x1 << 7)
	}
	return data
}

func (g *Game) in() {
	debugPrint(g.mc, "IN", 1)
	device := (*g.mc.memory)[g.mc.programCounter+1]
	switch device {
	case 0: // Hardware inputs that are never actually used in the code
		g.mc.ra = g.inPort0()
	case 1: // Button presses
		g.mc.ra = g.inPort1()
	case 2: // Game settings
		g.mc.ra = g.inPort2()
	case 3: // Give the value in the shift register
		g.mc.ra = g.sr.getResult()
	}

	g.mc.programCounter += 2
}

func (g *Game) playSounds(bank uint8) {
	return // Todo: Fix this later
	soundBits := g.mc.ra
	if bank == 1 {
		if soundBits&0x1 > 0 { // bit 0
			g.soundBoard["ufo_lowpitch"].Rewind()
			g.soundBoard["ufo_lowpitch"].Play()
		}
		if (soundBits>>1)&0x1 > 0 { // bit 1 is set
			if !g.soundBoard["shoot"].IsPlaying() {
				g.soundBoard["shoot"].Rewind()
				g.soundBoard["shoot"].Play()
			}
		}
	}
	if bank == 2 {

	}
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
		g.playSounds(1)
	case 4: // Shift data
		g.sr.shiftData(g.mc.ra)
	case 5: // Sound bank 2
		g.playSounds(2)
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
			if (byte>>uint32(shift))&0x1 > 0 {
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

	// First save the current program counter on the stack
	g.mc.stackPointer -= 2
	(*g.mc.memory)[g.mc.stackPointer] = uint8(g.mc.programCounter & 0xFF)
	(*g.mc.memory)[g.mc.stackPointer+1] = uint8(g.mc.programCounter >> 8)

	// Then set the program counter to the RST instruction

	switch scanline {
	case 96:
		g.mc.programCounter = 0x8
	case 224:
		g.mc.programCounter = 0x10
	default:
		panic("Unhandled scanline() call. 96 and 224 are the only valid values")
	}
}

// keyUp() provides functionality for detecting a keyup event
// based on the previous & current state of a specific key
// This function will also update the previous state
func keyUp(g *Game, key ebiten.Key) bool {
	last := g.lastKeyState[key]
	g.lastKeyState[key] = ebiten.IsKeyPressed(key)
	if last == true && ebiten.IsKeyPressed(key) == false {
		return true
	}
	return false
}

// checkKeyboard() - Checks for non game related inputs
// that aren't part of the standard game play (ie: escape key or DIP switches)
func checkKeyboard(g *Game) error {
	// This will interrupt ebiten.Run()
	if ebiten.IsKeyPressed(ebiten.KeyEscape) {
		return errors.New("Exiting normally due to ESCAPE being pushed")
	}
	// Toggle the dip switch state everytime the associated key is pressed
	if keyUp(g, ebiten.Key3) {
		g.dip3 = !g.dip3
	}
	if keyUp(g, ebiten.Key4) {
		g.dip4 = !g.dip4
	}
	if keyUp(g, ebiten.Key5) {
		g.dip5 = !g.dip5
	}
	if keyUp(g, ebiten.Key6) {
		g.dip5 = !g.dip6
	}
	if keyUp(g, ebiten.Key7) {
		g.dip5 = !g.dip7
	}
	return nil
}

func (g *Game) run() {
	// Starts the main event loop by booting up ebiten
	// displayData is given with width/height swapped since
	// the image will be rotated before being pasted
	//displayData := image.NewRGBA(image.Rect(0, 0, SCREENHEIGHT, SCREENWIDTH))
	displayData, err := ebiten.NewImage(SCREENHEIGHT, SCREENWIDTH, ebiten.FilterNearest)

	debugPrintLn("run() start")

	if err != nil {
		debugPrintLn("Error creating a displayData canvas")
	}

	f := func(screen *ebiten.Image) error {
		tmpImage := image.NewRGBA(image.Rect(0, 0, SCREENHEIGHT, SCREENWIDTH))
		debugPrintLn("Starting to draw frame")
		for i := 0; i < INSTRUCTIONSPERFRAME/2; i++ {
			g.tick()
		}
		debugPrintLn("Top render")
		g.render(tmpImage, true)
		debugPrintLn("Scanline interrupt 96")
		g.scanLine(96)

		debugPrintLn("Starting to draw frame")
		for i := 0; i < INSTRUCTIONSPERFRAME/2; i++ {
			g.tick()
		}

		debugPrintLn("Bottom render")
		g.render(tmpImage, false)
		debugPrintLn("Scanline interrupt 224")
		g.scanLine(224)
		debugPrintLn("Flipping buffers")

		displayData.ReplacePixels(tmpImage.Pix)
		opts := &ebiten.DrawImageOptions{}
		opts.GeoM.Rotate(-math.Pi / 2)
		opts.GeoM.Translate(0, float64(SCREENHEIGHT))
		screen.DrawImage(displayData, opts)

		if err := g.audioContext.Update(); err != nil {
			return err
		}

		return checkKeyboard(g)
	}

	debugPrintLn("Starting ebiten.run()")

	ebiten.SetRunnableInBackground(true)
	runErr := ebiten.Run(f, SCREENWIDTH, SCREENHEIGHT, 3, "Space Invaders")
	errStr := fmt.Sprintf("Exited run() with error: %s", runErr)
	debugPrintLn(errStr)
}
