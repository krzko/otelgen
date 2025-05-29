package scenarios

import (
	"crypto/rand"
	"encoding/binary"
	randv2 "math/rand/v2"
)

// NewRand returns a *randv2.Rand seeded with cryptographically secure randomness.
func NewRand() *randv2.Rand {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic("failed to seed PRNG: " + err.Error())
	}
	seed := binary.LittleEndian.Uint64(b[:])
	return randv2.New(randv2.NewPCG(seed, 0))
}
