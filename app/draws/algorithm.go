// Package draws picks the winner of a raffle among its paid numbers and
// stores an auditable result (PLAN.md D12). The algorithm is a pure
// function so anyone can recompute a published result; docs/draw-algorithm.md
// describes it byte by byte.
package draws

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"math"
	"slices"
	"strconv"
	"strings"
)

// AlgorithmV1 names the algorithm stored with each draw.
const AlgorithmV1 = "v1"

// SeedSize is the seed length in bytes (256 bits from crypto/rand).
const SeedSize = 32

// ErrNoParticipants is returned by Pick when no number was paid.
var ErrNoParticipants = errors.New("draws: no participants")

// ErrBadSeed is returned by Pick for a seed that is not SeedSize bytes.
var ErrBadSeed = errors.New("draws: seed must be 32 bytes")

// ParticipantsList is the canonical text of the participants: the paid
// numbers in ascending decimal order joined by "," with no spaces
// ("3,7,12").
func ParticipantsList(numbers []int) string {
	sorted := slices.Sorted(slices.Values(numbers))
	parts := make([]string, len(sorted))

	for i, n := range sorted {
		parts[i] = strconv.Itoa(n)
	}

	return strings.Join(parts, ",")
}

// ParticipantsHash is the lowercase hex SHA-256 of ParticipantsList.
func ParticipantsHash(numbers []int) string {
	sum := sha256.Sum256([]byte(ParticipantsList(numbers)))
	return hex.EncodeToString(sum[:])
}

// Pick returns the winning number for seed among numbers (v1):
//
//	n = len(numbers), numbers sorted ascending
//	for counter = 0, 1, 2, ...:
//	    block = HMAC-SHA256(key = seed, message = counter as 8-byte big-endian)
//	    v     = first 8 bytes of block as a big-endian uint64
//	    accept v if v < floor(2^64 / n) * n          (rejection sampling)
//	winner = numbers[v mod n]
//
// Rejecting the top 2^64 mod n values makes every index exactly equally
// likely (no modulo bias).
func Pick(seed []byte, numbers []int) (int, error) {
	if len(seed) != SeedSize {
		return 0, ErrBadSeed
	}

	if len(numbers) == 0 {
		return 0, ErrNoParticipants
	}

	sorted := slices.Sorted(slices.Values(numbers))

	return sorted[index(stream(seed), uint64(len(sorted)))], nil
}

// stream returns the v1 value generator for seed.
func stream(seed []byte) func() uint64 {
	var (
		counter uint64
		msg     [8]byte
	)

	return func() uint64 {
		binary.BigEndian.PutUint64(msg[:], counter)
		counter++

		mac := hmac.New(sha256.New, seed)
		mac.Write(msg[:])

		return binary.BigEndian.Uint64(mac.Sum(nil)[:8])
	}
}

// index draws values from next until one falls below floor(2^64/n)*n and
// returns it mod n.
func index(next func() uint64, n uint64) uint64 {
	// 2^64 mod n, computed without overflowing uint64.
	rem := (math.MaxUint64%n + 1) % n

	for {
		v := next()
		// Accepted values are [0, 2^64 - rem); rem = 0 accepts everything.
		if rem == 0 || v <= math.MaxUint64-rem {
			return v % n
		}
	}
}
