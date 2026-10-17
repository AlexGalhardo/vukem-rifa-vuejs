package draws

import (
	"encoding/hex"
	"errors"
	"math"
	"strings"
	"testing"
)

func seq(from, to int) []int {
	out := make([]int, 0, to-from+1)
	for n := from; n <= to; n++ {
		out = append(out, n)
	}

	return out
}

// Vectors computed with the independent Python reference in
// docs/draw-algorithm.md; a change here breaks every published result.
func TestPick_Vectors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		seed    string
		numbers []int
		winner  int
		hash    string
	}{
		{strings.Repeat("00", 32), []int{1}, 1, "6b86b273ff34fce19d6b804eff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b"},
		{strings.Repeat("00", 32), seq(1, 10), 5, "d10434d8c3b61a402e8c5a9f2e64e628b1e9f8c1dd747e7f480dbe57eb79c425"},
		{strings.Repeat("ff", 32), []int{12, 3, 7}, 7, "6e1edfa51fa67e031108ef80f855db72240de72e7fac84463380b1c9d0e656df"},
		{"000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f", seq(1, 1000), 706, "72827ef050e6908b556c0b28e4a22355ea8099a101da5c5e035ed096e0576db6"},
		{"5f2c9a1e8b7d4c3a2f1e0d9c8b7a6f5e4d3c2b1a0f9e8d7c6b5a493827161504", []int{2, 4, 8, 16, 32, 64, 128, 256, 512}, 2, "8910b9e0523f75a24e8dcfc3a73c0e0a61c9f16e3fd991e102d5f27ff12a0f6d"},
	}

	for _, tt := range tests {
		seed, err := hex.DecodeString(tt.seed)
		if err != nil {
			t.Fatal(err)
		}

		got, err := Pick(seed, tt.numbers)
		if err != nil || got != tt.winner {
			t.Errorf("Pick(%s…, %d numbers) = %d, %v; want %d", tt.seed[:8], len(tt.numbers), got, err, tt.winner)
		}

		if h := ParticipantsHash(tt.numbers); h != tt.hash {
			t.Errorf("ParticipantsHash(%d numbers) = %s, want %s", len(tt.numbers), h, tt.hash)
		}
	}
}

func TestPick_Errors(t *testing.T) {
	t.Parallel()

	if _, err := Pick(make([]byte, SeedSize), nil); !errors.Is(err, ErrNoParticipants) {
		t.Errorf("no numbers: %v", err)
	}

	if _, err := Pick(make([]byte, 16), []int{1}); !errors.Is(err, ErrBadSeed) {
		t.Errorf("short seed: %v", err)
	}

	if got := ParticipantsList([]int{10, 2, 33}); got != "2,10,33" {
		t.Errorf("ParticipantsList = %q", got)
	}

	if got := ParticipantsHash(nil); got != "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855" {
		t.Errorf("empty hash = %s", got)
	}
}

func TestIndex_RejectsBiasedValues(t *testing.T) {
	t.Parallel()

	// For n = 3, 2^64 mod 3 = 1: only MaxUint64 is rejected.
	values := []uint64{math.MaxUint64, math.MaxUint64 - 1}
	next := func() uint64 { v := values[0]; values = values[1:]; return v }

	if got := index(next, 3); got != (math.MaxUint64-1)%3 || len(values) != 0 {
		t.Fatalf("index = %d (left %d values), want the second value mod 3", got, len(values))
	}

	// Powers of two never reject.
	values = []uint64{math.MaxUint64}
	if got := index(next, 4); got != 3 {
		t.Fatalf("index(n=4) = %d, want 3", got)
	}
}

// Each of 10 numbers should win ~10% of 100k draws with random seeds; the
// chi-square bound (df = 9, p ≈ 0.0001) keeps the test stable.
func TestPick_Distribution(t *testing.T) {
	t.Parallel()

	const draws = 100_000

	numbers := seq(1, 10)
	counts := map[int]int{}
	seed := make([]byte, SeedSize)

	for i := range draws {
		seed[0], seed[1], seed[2] = byte(i), byte(i>>8), byte(i>>16)

		w, err := Pick(seed, numbers)
		if err != nil {
			t.Fatal(err)
		}

		counts[w]++
	}

	expected := float64(draws) / float64(len(numbers))
	chi := 0.0

	for _, n := range numbers {
		d := float64(counts[n]) - expected
		chi += d * d / expected
	}

	if chi > 33.72 {
		t.Fatalf("chi-square %.2f too high: %v", chi, counts)
	}
}
