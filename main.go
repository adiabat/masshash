package main

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/minio/blake2b-simd"
)

func main() {
	pyramid2()
	return
}

func pyramid2() {
	base := make([]byte, 8)

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

	y := nextLvl(z)
	fmt.Printf("%d bits\n", len(y)*8)
	fmt.Printf("%x\n", y)
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

func nextLvl(base []byte) []byte {
	// try with in-degree 512.
	// reduce width by 511 bits.
	n := uint64(len(base) * 8)

	var i uint64

	for i < n-512 {
		b := i / 8        // b is the byte position
		t := uint8(i % 8) // t is the bit position

		// most bytes are loaded as-is.  The first and last bytes are rotated by t

		input := make([]byte, 64)
		copy(input, base[b:b+62])

		last := base[b+63]

		// shift first byte up by t
		input[0] = input[0] << t

		input[62] = last >> (8 - t)

		sum := blake2b.Sum256(input)

		// clear the bit position we're at
		base[b] &= ^(1 << t)
		// set with MSbit of hash output.
		// actually not using the first bit of the hash output but should be OK.
		base[b] |= sum[0] & (1 << t)

		//		fmt.Printf("%x.", base[:32])
		//		fmt.Printf("b:%d t:%d\n", b, t)

		i++
	}

	return base
}
