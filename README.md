# 8080
Intel 8080 Emulator for the Taito Space Invaders game 

There is source code for two executables here:

1) test - A barebones implementation of the KR580VM80A processor that can run all of the "i8080-core" ROMs (https://github.com/begoon/i8080-core/). This emulator can connect to a local server (server.rb) that can compare the output of this emulator against other emulators to detect differences in the register values. The code for the i8080-core will need to be updated to provide this output over port 5679.

2) space_invaders - A superset of 'test,' but with additional functionality to emulate the Taito Space Invaders game as faithfully as possible. 

Dependencies:
1) Space Invaders ROM has to be located in the same directory as the executable. The files must be named as follows:
	invaders_h.rom
	invaders_g.rom
	invaders_f.rom
	invaders_e.rom
	

Built in GO with lots of help from the following resources:
1) http://www.computerarcheology.com/Arcade/SpaceInvaders/Code.html
2) http://www.emulator101.com/reference/8080-by-opcode.html
3) http://www.pastraiser.com/cpu/i8080/i8080_opcodes.html
4) https://github.com/begoon/i8080-core/