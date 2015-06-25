package main

import (
	"io/ioutil"
	"os"
	"fmt"
    "runtime"
    "log"
	"math/rand"
	"github.com/go-gl/gl/v2.1/gl"
	"github.com/go-gl/glfw/v3.1/glfw"
	"time"
)

var debug[]uint16 

var keyvalues = map[glfw.Key]byte {
	glfw.Key1:           0x1,
	glfw.Key2:           0x2,
	glfw.Key3:           0x3,
	glfw.Key4:           0xc,
	glfw.KeyQ:           0x4,
	glfw.KeyW:           0x5,
	glfw.KeyE:           0x6,
	glfw.KeyR:           0xd,
	glfw.KeyA:           0x7,
	glfw.KeyS:           0x8,
	glfw.KeyD:           0x9,
	glfw.KeyF:           0xe,
	glfw.KeyZ:           0xa,
	glfw.KeyX:           0x0,
	glfw.KeyC:           0xb,
	glfw.KeyV:           0xf,
}

type chip8 struct {

	opcode uint16
	memory [4096]byte
	v [16]byte
	i uint16
	pc uint16
	gfx [64*32]byte
	delay_timer byte
	sound_timer byte
	stack [16]uint16
	stack_p uint16
	key [16]bool
	drawFlag bool
}

func init() {
	runtime.LockOSThread()
}

var myChip8 chip8

func main() {
	if err := glfw.Init(); err != nil {
		log.Fatalln("failed to initialize glfw:", err)
	}
	defer glfw.Terminate()

    glfw.WindowHint(glfw.Resizable, glfw.False)
	glfw.WindowHint(glfw.ContextVersionMajor, 2)
	glfw.WindowHint(glfw.ContextVersionMinor, 1)
	window, err := glfw.CreateWindow(640, 320, "chip8", nil, nil)
	if err != nil {
		panic(err)
	}

	window.MakeContextCurrent()


	if err := gl.Init(); err != nil {
		panic(err)
	}

	gl.ClearColor(0, 0, 0, 0)
	gl.MatrixMode(gl.PROJECTION)

	gl.Ortho(0, 64, 32, 0, 0, 1)

	myChip8.initialize()
	myChip8.loadGame("./roms/Addition Problems [Paul C. Moews].ch8");

	for !window.ShouldClose() {
		myChip8.emulateCycle(window)

		if(myChip8.drawFlag) {
			myChip8.drawGraphics()
      		window.SwapBuffers()
			time.Sleep(32 * time.Millisecond)
		}

        glfw.PollEvents()
		myChip8.setKeys(window);
	} 
}

func (c *chip8) initialize() {
	c.pc 	 = 0x200
	c.opcode = 0
	c.i 	 = 0
	c.stack_p= 0

	for i := 0; i < 80; i++ {
		c.memory[i] = chip8_fontset[i]
	}
}

func (c *chip8) loadGame(filename string) {
	raw, err := ioutil.ReadFile(filename)

	if err != nil {
		// Log error
		os.Exit(-2)
	}

	for i := 0; i < len(raw); i++ {
		c.memory[i+512] = raw[i]
	}

}

func (c *chip8) emulateCycle(window *glfw.Window) {
	// Fetch opcode
	c.opcode = uint16(c.memory[c.pc]) << 8 | uint16(c.memory[c.pc + 1])

	// Debug -- compile list of all seen opcodes so we can compare working and broken games.
		// seen := false
		// for i := 0; i < len(debug); i++ {
		// 	if debug[i] == c.opcode {
		// 		seen = true
		// 		break
		// 	}
		// }
		// if seen == false {
		// 	debug = append(debug, c.opcode)
		// }
		// print("Elements seen: \n")
		// for _,element := range debug {
		// 	fmt.Printf("%X ", element)
		// }
		// print("\n")

	//Debug -- print out complete memory for debugging instructions and such.
		// for i := 0; i < len(c.memory); i++ {
		// 	fmt.Printf("m[%X]:%X ", i, c.memory[i])
		// }

	//Debug	
	fmt.Printf("Op:%X pc:%X sp:%X i:%X delay:%X beep:%X \t", c.opcode, c.pc, c.stack_p, c.i, c.delay_timer, c.sound_timer)
	//fmt.Printf("Opcode: 0x%X i: 0x%X\t", c.opcode, c.i)
	for i := 0; i < len(c.v); i++ {
		fmt.Printf("[%d]:%X ", i, c.v[i])
	}
	print("\n")

	// Decode Opcode
	switch (c.opcode & 0xF000) {
		case 0x0000:
			switch (c.opcode & 0xFFFF) {
				case 0x00E0: //00E0: Clears the screen
					c.gfx = [64*32]byte{0}
					c.drawFlag = true
					c.pc += 2

				case 0x00EE: //00EE: Returns from a subroutine
					if c.stack_p < 0 {
						os.Exit(-1)
					}
					c.pc = c.stack[c.stack_p]
					c.stack_p--
					c.pc += 2

				default:
					//0NNN: Calls RCA 1802 program at address NNN.
					fmt.Printf("RCA 1802 Not Supported.\nExiting...", c.opcode)
					os.Exit(-1)
			}

		//WORKING
		case 0x1000: //1NNN: Jumps to address NNN
			c.pc = c.opcode & 0x0FFF

		//?
		case 0x2000: //2NNN: Calls subroutine at NNN
			if c.stack_p >= 15 {
				os.Exit(-1)
			}
			c.stack_p++
			c.stack[c.stack_p] = c.pc
			c.pc = uint16(c.opcode & 0x0FFF)

		//?
		case 0x3000: //3XNN: Skips the next instruction if VX doesn't equal NN.
			if (c.v[(c.opcode & 0x0F00) >> 8] != byte(c.opcode & 0x00FF)) {
				c.pc += 4
			} else {
				c.pc +=2
			}

		//WORKING
		case 0x4000: //4XNN: Skips the next instruction if VX equals NN
			if (c.v[(c.opcode & 0x0F00) >> 8] == byte(c.opcode & 0x00FF)) {
				c.pc += 4
			} else {
				c.pc +=2
			}

		case 0x5000: //5XY0: Skip the following instruction if the value of register VX is equal to the value of register VY
			if (c.v[(c.opcode & 0x0F00) >> 8] == c.v[(c.opcode & 0x00F0) >> 4]) {
				c.pc += 4
			} else {
				c.pc +=2
			}
		
		//WORKING
		case 0x6000: //6XNN: Sets VX to NN
			c.v[(c.opcode & 0x0F00) >> 8] = byte(c.opcode & 0x00FF)
			c.pc += 2

		//WORKING
		case 0x7000: //7XNN: Adds NN to VX
			c.v[(c.opcode & 0x0F00) >> 8] += byte(c.opcode & 0x00FF)
			c.pc += 2

		case 0x8000:
			switch (c.opcode & 0x000F) {

		//SEEMS RIGHT
				case 0x0000: //8XY0: Sets VX to the value of VY.
					c.v[(c.opcode & 0x0F00) >> 8] = c.v[(c.opcode & 0x00F0) >> 4]
					c.pc += 2

		//SEEMS RIGHT
				case 0x0001: //8XY1: Sets VX to VX or VY.
					c.v[(c.opcode & 0x0F00) >> 8] = (c.v[(c.opcode & 0x00F0) >> 4] | c.v[(c.opcode & 0x0F00) >> 8])
					c.pc += 2

		//SEEMS RIGHT
				case 0x0002: //8XY2: Sets VX to VX and VY.
					c.v[(c.opcode & 0x0F00) >> 8] = (c.v[(c.opcode & 0x00F0) >> 4] & c.v[(c.opcode & 0x0F00) >> 8])
					c.pc += 2

		//SEEMS RIGHT
				case 0x0003: //8XY3: Sets VX to VX xor VY.
					c.v[(c.opcode & 0x0F00) >> 8] = (c.v[(c.opcode & 0x00F0) >> 4] ^ c.v[(c.opcode & 0x0F00) >> 8])
					c.pc += 2

		//WORKING ?
				case 0x0004: //8XY4: Adds VY to VX. VF is set to 1 when there's a carry, and to 0 when there isn't.
					carry_check := uint16(c.v[(c.opcode & 0x00F0) >> 4] + c.v[(c.opcode & 0x00F0) >> 4])
					if ((carry_check & 0xFF00) >> 8) > 0 {
						c.v[0xF] = 1 //cary
					} else {
						c.v[0xf] = 0
					}
					c.v[(c.opcode & 0x0F00) >> 8] += c.v[(c.opcode & 0x00F0) >> 4]
					c.pc += 2

		//?
				case 0x0005: //8XY5: VY is subtracted from VX. VF is set to 0 when there's a borrow, and 1 when there isn't.
					if (c.v[(c.opcode & 0x0F00) >> 8] > c.v[(c.opcode & 0x00F0) >> 4]) {
						c.v[0xF] = 1 
					} else {
						c.v[0xF] = 0 // borrow
					}
					c.v[(c.opcode & 0x0F00) >> 8] = c.v[(c.opcode & 0x0F00) >> 8]-  c.v[(c.opcode & 0x00F0) >> 4]
					c.pc += 2

				case 0x0006: // 8XY6: Shifts VX right by one. VF is set to the value of the least significant bit of VX before the shift
					if (c.v[(c.opcode & 0x0F00) >> 8] & 0x01) == 0x01 {
						c.v[0xF] = 1
					} else {
						c.v[0xF] = 0
					}
					c.v[(c.opcode & 0x0F00) >> 8] = (c.v[(c.opcode & 0x0F00) >> 8] >> 1)
					c.pc += 2

				case 0x0007:
					fmt.Printf("Opcode not implemented:0x%X\n", c.opcode)
					os.Exit(-2)

				case 0x000E: //8XYE: Shifts VX left by one. VF is set to the value of the most significant bit of VX before the shift
				if (c.v[(c.opcode & 0x0F00) >> 8] & 0x80) == 0x80 {
						c.v[0xF] = 1
					} else {
						c.v[0xF] = 0
					}
					c.v[(c.opcode & 0x0F00) >> 8] = (c.v[(c.opcode & 0x0F00) >> 8] << 1)
					c.pc += 2
			
				default:
					fmt.Printf("Unknown opcode:0x%X\n", c.opcode)
					os.Exit(-2)

			}

		case 0x9000: //9XY0: Skips the next instruction if VX doesn't equal VY
			if (c.v[(c.opcode & 0x0F00) >> 8] != c.v[(c.opcode & 0x00F0) >> 4]) {
				c.pc += 4
			} else {
				c.pc += 2
			}

		//WORKING
		case 0xA000: //ANNN: Sets i to the address NNN
			c.i = (c.opcode & 0x0FFF)
			c.pc += 2

		case 0xB000: //BNNN: Jumps to the address NNN plus V0.
			c.pc = (c.opcode & 0x0FFF) + uint16(c.v[0x0])

		//?
		case 0xC000: //CXNN: Sets VX to a random number, masked by NN.
			rand.Seed(time.Now().UnixNano())
			c.v[(c.opcode & 0x0F00) >> 8] = byte(rand.Int()) & byte(c.opcode & 0x00FF)
			c.pc += 2

		//WORKING
		case 0xD000: //DXYN: Draw sprites [search for full description]
			var x = uint16(c.v[(c.opcode & 0x0F00) >> 8])
			var y = uint16(c.v[(c.opcode & 0x00F0) >> 4])
			var height = uint16(c.opcode & 0x000F)
			var pixel uint16

			c.v[0xF] = 0

			for yline := uint16(0); yline < uint16(height); yline++ {

				pixel = uint16(c.memory[c.i + yline])
				for xline := uint16(0); xline < 8; xline++ {

					if ((pixel & (0x80 >> xline)) != 0) {					
						memloc := ((uint16(x) + xline + ((uint16(y) + yline) * 64)) % 2048)
					
						if (c.gfx[memloc] == 1) {
							c.v[0xF] = 1;
							//fmt.Printf("\nxline: %d yline:%d x:%d y:%d memloc:%d\n", xline, yline, x, y, memloc)
						}
						c.gfx[memloc] ^= 1
					}
				}
			}
			c.drawFlag = true
			c.pc += 2

		case 0xE000:
			switch (c.opcode & 0x00FF) {
				case 0x009E: //EX9E: Skips the next instruction if the key stored in VX is pressed
					if(c.key[c.v[(c.opcode & 0x0F00) >> 8]] != false) {
						c.pc += 4;
					} else {
						c.pc += 2;
					}

				case 0x00A1: //EXA1: Skips the next instruction if the key stored in VX isn't pressed
					if(c.key[c.v[(c.opcode & 0x0F00) >> 8]] == false) {
						c.pc += 4;
					} else {
						c.pc += 2;
					}

				default:
					fmt.Printf("Unknown opcode:0x%X\n", c.opcode)
					os.Exit(-2)
			}

		case 0xF000:
			switch (c.opcode & 0x00FF) {
				case 0x000A: //FX0A: A key press is awaited, and then stored in VX.
					glfw.WaitEvents()
					for k, v := range keyvalues {	
						if window.GetKey(k) == glfw.Press {
							c.v[(c.opcode & 0x0F00) >> 8] = v
							c.key[v] = true
							c.pc += 2
							break
						}
					}
				case 0x0007: //FX07: Sets VX to the value of the delay timer
					c.v[(c.opcode & 0x0F00) >> 8] = c.delay_timer
					c.pc += 2

		//WORKING
				case 0x0015: //FX15: Sets the delay timer to VX
					c.delay_timer = c.v[(c.opcode & 0x0F00) >> 8]
					c.pc += 2

		//WORKING
				case 0x0018: //FX18: Sets the sound timer to VX
					c.sound_timer = c.v[(c.opcode & 0x0F00) >> 8]
					c.pc += 2

				case 0x001E: //FX1E: Adds VX to I.
					c.i += uint16(c.v[(c.opcode & 0x0F00) >> 8])
					c.pc += 2
		//WORKING
				case 0x0029: //FX29: Sets I to the location of the sprite for the character in VX. Characters 0-F (in hexadecimal) are represented by a 4x5 font
					c.i = uint16(c.v[(c.opcode & 0x0F00) >> 8] * byte(5))
					c.pc += 2

		//WORKING
				case 0x0033: //FX33: Look it up.  It's a long one.
					c.memory[c.i+2] = byte((c.v[(c.opcode & 0x0F00) >> 8] / 100 )% 10)
					c.memory[c.i+1] = byte((c.v[(c.opcode & 0x0F00) >> 8] / 10 ) % 10)
					c.memory[c.i]   = byte( c.v[(c.opcode & 0x0F00) >> 8] % 10)
					c.pc += 2

				case 0x0055: //FX55: Stores V0 to VX in memory starting at address I.
					for i := uint16(0); i <= ((c.opcode & 0x0F00) >> 8); i++ {
						c.memory[c.i] = c.v[i]
						c.i++
					}
					c.pc += 2

				case 0x0065: //FX65: Fills V0 to VX with values from memory starting at address I
					for i := uint16(0); i <= ((c.opcode & 0x0F00) >> 8); i++ {
						c.v[i] = c.memory[c.i]
						c.i++
					}
					c.pc += 2

				default:
					fmt.Printf("Unknown opcode:0x%X\n", c.opcode)
			}

		default:
			fmt.Printf("Unknown opcode:0x%X\n", c.opcode)
			os.Exit(-2)
	}

	// Update Timers
	if c.delay_timer > 0 {
		c.delay_timer--
	}

	if c.sound_timer > 0 {
		if c.sound_timer == 1{
			fmt.Printf("BEEP!\n")
		}
		c.sound_timer--
	}
}

func (c *chip8) drawGraphics() {
	gl.MatrixMode(gl.POLYGON)

	for y := 0; y < 32; y++ {
		for x := 0; x < 64; x++ {
			if c.gfx[(y * 64) + x] == 1 {
				gl.Color3f(1,1,1)
			} else {
				gl.Color3f(0,0,0)
			}
			gl.Rectf(float32(x), float32(y), float32(x+1), float32(y+1))
		}
	}
	c.drawFlag = false
}

func (c *chip8) setKeys(window *glfw.Window) {
	for k, v := range keyvalues {	
		if window.GetKey(k) == glfw.Release {
			c.key[v] = true
		} else {
			c.key[v] = false
		}
	}
}

var chip8_fontset = [80]byte{ 
  0xF0, 0x90, 0x90, 0x90, 0xF0, // 0
  0x20, 0x60, 0x20, 0x20, 0x70, // 1
  0xF0, 0x10, 0xF0, 0x80, 0xF0, // 2
  0xF0, 0x10, 0xF0, 0x10, 0xF0, // 3
  0x90, 0x90, 0xF0, 0x10, 0x10, // 4
  0xF0, 0x80, 0xF0, 0x10, 0xF0, // 5
  0xF0, 0x80, 0xF0, 0x90, 0xF0, // 6
  0xF0, 0x10, 0x20, 0x40, 0x40, // 7
  0xF0, 0x90, 0xF0, 0x90, 0xF0, // 8
  0xF0, 0x90, 0xF0, 0x10, 0xF0, // 9
  0xF0, 0x90, 0xF0, 0x90, 0x90, // A
  0xE0, 0x90, 0xE0, 0x90, 0xE0, // B
  0xF0, 0x80, 0x80, 0x80, 0xF0, // C
  0xE0, 0x90, 0x90, 0x90, 0xE0, // D
  0xF0, 0x80, 0xF0, 0x80, 0xF0, // E
  0xF0, 0x80, 0xF0, 0x80, 0x80} // F

  // Pong notes:
  // v[1] stores p1 score (and others?)
  // v[2] stores P2 score
  // v[4] v[5] store 
  // v[6] v[7] store ball position
  // v[A] v[B] store paddle-left position
  // v[C] v[D] store paddle-right position
