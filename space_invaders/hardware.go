package main

// ShiftRegister - Emulates a 16-bit shit register
// that can be written & read from
type ShiftRegister struct {
	offset uint8 // direct write from port 2
	value  uint16
}

// Shifts in data depending on the currently set offset value
// Newly shifted data is inserted into the highest 8-bits
// of the stored value.
// This data is written to port 4
func (sr *ShiftRegister) setOffset(data uint8) {
	sr.value = uint16(data)<<8 | (sr.value >> 8)
}

// Returns the stored data with the offset applied.
// Offset of 0 - get the highest 8 bits
// Offset of 7 - get bits 1-8
// Read from port 3
func (sr *ShiftRegister) getResult() uint8 {
	offset := (8 - sr.offset)
	return uint8(sr.value >> offset)
}
