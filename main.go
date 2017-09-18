package main

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/minio/blake2b-simd"
)

const (
	inputSize = 128
)

func main() {
	pebbleVar(16, 0)
	return
}

// buildBase creates the lowest base level of the pyramid.
// the lowest level is just 0, 1, 2, 3... in uint64s
// argument log width is base 2 log of how many bytes you want.
// eg 24 will give you 2^24 bypes, 16Mbyte
func buildBase(logWidth uint8) []byte {

	// wordWidth is how many 64-bit (8 byte) words. >>6 because 64 bits per word.
	wordWidth := uint64(1<<logWidth) / 8

	var buf bytes.Buffer

	for i := uint64(0); i < wordWidth; i++ {
		binary.Write(&buf, binary.BigEndian, i)
	}
	return buf.Bytes()
}

func pebble() {
	row := buildBase(20)
	var totalHashes uint64
	var height int
	levels := len(row) / 64

	for height < levels {
		totalHashes += deg2NextLvl(row)
		height++
	}

	fmt.Printf("Final output is %x\n", row)
	fmt.Printf("width %d bytes, height %d\n", len(row), height)
	fmt.Printf("%d hashes performed\n", totalHashes)

}

// pebbleVar runs the pebbling with variable base size and hash output size.
// both arguments are log numbers of bytes, so (12, 4) will make a base
// 4096 bytes long, and use a hash output size of 16 bytes.  Maximum hash
// output size is 512 bytes. Hash input size is fixed at 1024 bytes.
func pebbleVar(logBase, logWidth uint8) {

	// label size is 2**logWidth
	hSize := uint64(1 << logWidth)
	// can't have hSize more than 64 bytes; blake2b doesn't provide that much
	// arbitrary input size is possible and would speed it up, but breaks
	// memory requirement (random oracle as cache)
	if hSize > 64 {
		panic("hash output size greater than 64")
	}
	if hSize > inputSize {
		panic("hash output size greater than input size (leaves gaps)")
	}
	if hSize == inputSize {
		panic("hash output size greater equal to input size no pyramiding")
	}

	row := buildBase(logBase)

	var totalHashes uint64
	var height uint64

	// total height of the cylinder is row / hashSize
	levels := uint64(len(row)) / (inputSize - hSize)

	for height < levels {
		totalHashes += nextLvl(row, hSize)
		height++
	}

	fmt.Printf("Final output is %x\n", row)
	fmt.Printf("hash output size %d bytes. %d labels per row\n",
		hSize, uint64(len(row))/hSize)
	fmt.Printf("row width %d bytes. fan-in %d bytes per level. height %d\n",
		len(row), inputSize, height)
	fmt.Printf("%d hashes performed\n", totalHashes)

}

// deg2NextLvl computes the next level up with a variable degree (based on hash
// output size)
func nextLvl(row []byte, hSize uint64) uint64 {
	var hashes uint64

	if uint64(len(row)) < hSize+inputSize {
		fmt.Printf("length of input only %d bytes long\n", len(row))
		return 0
	}
	if uint64(len(row))%hSize != 0 {
		fmt.Printf("length of input not multiple of %d\n", hSize)
		return 0
	}

	// rowlength is the length of the actual row before we append to it.
	rowLength := uint64(len(row))

	// modify the row by sticking the first hSize bytes on the end
	row = append(row, row[:inputSize]...)

	// then iterate through till rowlenght (there will be one hash size
	// left at the end of the row

	pos := uint64(0)
	for pos < rowLength {
		input := row[pos : pos+inputSize]
		nHash := blake2b.Sum512(input)
		hashes++
		copy(row[pos:pos+hSize], nHash[:])
		pos += hSize

	}
	return hashes
}

// deg2NextLvl computes the next level up with a degree-2 pyramiding. wraps around
func deg2NextLvl(row []byte) uint64 {
	var hashes uint64

	hSize := uint64(64)
	if uint64(len(row)) < 2*hSize {
		fmt.Printf("length of input only %d bytes long\n", len(row))
		return 0
	}
	if uint64(len(row))%hSize != 0 {
		fmt.Printf("length of input not multiple of %d\n", hSize)
		return 0
	}

	// rowlength is the length of the actual row before we append to it.
	rowLength := uint64(len(row))

	// modify the row by sticking the first hSize bytes on the end
	row = append(row, row[:hSize]...)

	// then iterate through till rowlenght (there will be one hash size
	// left at the end of the row

	pos := uint64(0)
	for pos < rowLength {
		// for clarity, indicate positions from where we draw data
		rStart := pos + hSize
		rEnd := rStart + hSize
		//		fmt.Printf("row len %d, lstart %d rend %d\n", len(row), pos, rEnd)

		input := append(row[pos:rStart], row[rStart:rEnd]...)
		//		input = append(input, 0)
		nHash := blake2b.Sum512(input)
		hashes++
		copy(row[pos:rStart], nHash[:])
		pos += hSize

	}
	return hashes
}

func pebble512() {
	row := buildBase(16)

	var totalHashes uint64
	var height int
	levels := len(row) / 512

	for height < levels {
		totalHashes += deg512NextLvl(row)
		height++
	}

	fmt.Printf("Final output is %x\n", row)
	fmt.Printf("width %d bytes, height %d\n", len(row), height)
	fmt.Printf("%d hashes performed\n", totalHashes)
}

// deg512NextLvl computes the next level of the pyramid / cylinder
// use indegree of 512 by outputting only a single bit per hash evaluation
func deg512NextLvl(row []byte) uint64 {
	// for iteration ascending the pyramid, read in 65 bytes.  We only need 512
	// bits, but it won't be byte-alligned so we need data from the byte directly
	// below us, plus the 64 to the left of us.

	// First, copy the left-most 65 bytes and append it to the right.
	// This allows an easy way to loop around withough having to detect / deal
	// with the edge pebbles.  We can fan out to down and right.

	// This append step does take extra memory equivalent to the degree.
	// This could be eliminated with a left-facing fan-out to save memory, but
	// since our in-degree is so small relative to the row size, (which is
	// a point in the paper) it's not worth the extra code to do so.

	// loop is per-byte

	// rowlength is the length of the actual row before we append to it.
	rowlength := uint64(len(row))

	// modify the row by sticking looping the first 65 bytes on the end

	row = append(row, row[:65]...)

	var cur, hashes uint64

	// iterate per-byte
	for cur < rowlength {
		row[cur] = oneByte(row[cur : cur+65])
		hashes += 8
		cur++
	}

	return hashes
}

// oneByte calculates the byte above the 65 bytes given.  This needs 8 calls to
// the hash function: one per bit.
func oneByte(inRow []byte) uint8 {
	// make sure we have 65 bytes
	if len(inRow) != 65 {
		fmt.Printf("invalid length %d for oneUp\n", len(inRow))
		panic("invalid length")
	}
	var resultByte uint8
	// make 8 calls to oneBit
	for i := 0; i < 8; i++ {
		resultByte |= oneBit(inRow, uint8(i))
	}
	return resultByte
}

// oneBit calculates a single bit with in-degree 512
func oneBit(inRow []byte, position uint8) uint8 {
	// just assume inRow is 65 bytes.  If it's not this will crash
	var hashInput [64]byte
	copy(hashInput[:], inRow[:64])
	// shift the leftmost bits left by the position number, eliminating them
	hashInput[0] <<= position
	// move in the rightmost bits to take the place of the eliminated leftmost bits
	hashInput[0] |= inRow[64] >> (8 - position)
	// calculate the hash (random oracle call)
	hashOutput := blake2b.Sum512(hashInput[:])
	// return one bit from the leftmost byte
	return hashOutput[0] & (1 << position)
}
