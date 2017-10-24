# 8080
Intel 8080 Emulator for the Taito Space Invaders game implemented in golang

There is source code for two executables here:

1) test - A barebones implementation of the KR580VM80A processor that can run all of the "i8080-core" ROMs (https://github.com/begoon/i8080-core/). This emulator can connect to a local server (server.rb) that can compare the output of this emulator against other emulators to detect differences in the register values. The code for the i8080-core will need to be updated to provide this output over port 5679.

2) space_invaders - A superset of 'test,' but with additional functionality to emulate the Taito Space Invaders game as faithfully as possible. 

Controls for space invaders:

START       - Insert Credit

Key 1       - Start a 1 player game (1 credit required)

Key 2       - Start a 2 player game (2 credits required)

Left Arrow  - Move ship left (P1)

Right Arrow - Move ship right (P1)

Space       - Fire (P1)

Key A       - Move ship left (P2)

Key D       - Move ship right (P2)

Key W       - Fire (P2)

Key T       - Tilt

Key 3       - Toggle DIP Switch 3 (set number of lives; 00=3, 10 = 5)

Key 4       - Toggle DIP Switch 4 (some sort of power on self test)

Key 5       - Toggle DIP Switch 5 (set number of lives; 01 = 4, 11 = 6)

Key 6       - Toggle DIP Switch 6 (0 = extra ship at 1500, 1 = extra ship at 1000)

Key 7       - Toggle DIP Switch 7 (0 = display coin info on demo screen, 1=don't?)

Dependencies:
1) Ebiten 2D library (https://github.com/hajimehoshi/ebiten)

Built in GO with lots of help from the following resources:
1) http://www.computerarcheology.com/Arcade/SpaceInvaders/Code.html
2) http://www.emulator101.com/reference/8080-by-opcode.html
3) http://www.pastraiser.com/cpu/i8080/i8080_opcodes.html
4) https://github.com/begoon/i8080-core/
5) http://typedarray.org/wp-content/projects/Intel8080/index.html (Javascript version)
6) #ebiten on gopher.slack.com

To do:
1) GopherJS-ify the game so that it can be played online

Known Issues:
1) UFO sound does not play correctly (wontfix)
2) Some of the sounds are not implemented - extra ship, cocktail mode (wontfix)