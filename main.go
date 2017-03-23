package main

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/minio/blake2b-simd"
)

func main() {
	pebble()
	return
}

// buildBase creates the lowest base level of the pyramid.
// the lowest level is just 0, 1, 2, 3... in uint64s
// argument log width is base 2 log of how many BITS you want.
// eg 24 will give you 2^24 bits, 16Mbit, ~2Mbyte
func buildBase(logWidth uint8) []byte {

	// wordWidth is how many 64-bit (8 byte) words. >>6 because 64 bits per word.
	wordWidth := uint64(1<<logWidth) / 64

	var buf bytes.Buffer

	for i := uint64(0); i < wordWidth; i++ {
		binary.Write(&buf, binary.BigEndian, i)
	}
	return buf.Bytes()
}

func nextLvl(row []byte) uint64 {
	// for iteration ascending the pyramid, read in 65 bytes.  Some of the rightmost
	// byte will be mixed in to the leftmost byte.

	// loop is per-bit.
	// need to write in-place at bit i; otherwise you're using like 2*row memory
	// and the whole idea is that you max out at n memory

	bitlength := uint64(len(row) * 8)

	var i, hashes uint64

	for i < bitlength-512 {

		b := i / 8        // b is the byte position
		t := uint8(i % 8) // t is the bit position within the byte

		// first and last bytes are special; all middle bytes are copied in as-is
		first := row[b]
		last := row[b+64]

		// zero out t bits from first, and zero out all but t bits from last, then
		// or the two together

		first &= 0xff >> t

		last &= 0xff >> (8 - t)

		input := make([]byte, 64)
		input[0] = first | last

		copy(input[1:], row[b+1:b+63])

		sum := blake2b.Sum256(input)
		hashes++
		// clear the bit position we're at
		row[b] &= ^(1 << t)
		// set current bit position with (any bit of) hash output.

		row[b] |= sum[0] & (1 << t)
		//		fmt.Printf("b:%d t:%d\n", b, t)
		//		fmt.Printf("%x.", row[b:b+32])

		i++
	}
	row = row[:8]
	return hashes
}

func pebble() {
	row := buildBase(20)
	var i, hashes uint64

	for len(row) > 64 {

		fmt.Printf("%x ", row[:32])
		fmt.Printf("row %d is %d bits in width. %d cumulative hash ops\n",
			i, len(row)*8, hashes)

		hashes += nextLvl(row)
		row = row[:len(row)-64]
		i++
		//		fmt.Printf("row is %d bits\n%x\n", len(row)*8, row)
	}
	fmt.Printf("Final output is %x\n", row)
}

/*
func pyramid2() {
	base := make([]byte, )

	// initialize pyramid base
	for i, _ := range base {
		base[i] = uint8(i)
	}

	w := 8
	for h := 0; h < 8; h++ {
		for x := 0; x < (w-h)-1; x++ {
			sum := blake2b.Sum256([]byte{base[x], base[x+1]})
			base[x] = sum[0]
			fmt.Printf("%x %x\t", x, sum[0])
		}
		fmt.Printf("\n")
	}
	z := buildBase(10)
	fmt.Printf("%d bits\n", len(z)*8)
	fmt.Printf("%x\n\n\n", z)

	nextLvl(z)
	fmt.Printf("%d bits\n", len(z)*8)
	fmt.Printf("%x\n", z)
}
*/
