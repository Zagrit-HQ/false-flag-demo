package eval

import (
	"fmt"
	"math"
	"testing"
)

func TestRolloutBucket_Range(t *testing.T) {
	t.Parallel()
	for i := 0; i < 1000; i++ {
		i := i
		t.Run(fmt.Sprintf("u-%04d", i), func(t *testing.T) {
			t.Parallel()
			b := rolloutBucket("salt", fmt.Sprintf("user-%d", i))
			if b < 0 || b >= 10000 {
				t.Errorf("bucket out of [0,10000): %d", b)
			}
		})
	}
}

func TestRolloutBucket_DifferentSaltsDifferentBuckets(t *testing.T) {
	t.Parallel()
	// Statistical: across many users, different salts should produce
	// different bucket distributions. We don't aim for cryptographic
	// strength — just sanity that the salt actually influences output.
	mismatches := 0
	for i := 0; i < 200; i++ {
		v := fmt.Sprintf("user-%d", i)
		if rolloutBucket("alpha", v) != rolloutBucket("beta", v) {
			mismatches++
		}
	}
	if mismatches < 150 {
		t.Errorf("salt has near-zero influence: only %d/200 differed", mismatches)
	}
}

func TestRolloutBucket_DeterministicByInput(t *testing.T) {
	t.Parallel()
	for i := 0; i < 200; i++ {
		i := i
		t.Run(fmt.Sprintf("u-%d", i), func(t *testing.T) {
			t.Parallel()
			v := fmt.Sprintf("user-%d", i)
			b1 := rolloutBucket("checkout-v2", v)
			b2 := rolloutBucket("checkout-v2", v)
			if b1 != b2 {
				t.Errorf("non-deterministic: %d vs %d", b1, b2)
			}
		})
	}
}

func TestInBucket_EdgePercents(t *testing.T) {
	t.Parallel()
	cases := []struct {
		percent int
		any     bool
	}{
		{-1, false},
		{0, false},
		{100, true},
		{101, true},
		{1000, true},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(fmt.Sprintf("p=%d", tc.percent), func(t *testing.T) {
			t.Parallel()
			for i := 0; i < 50; i++ {
				v := fmt.Sprintf("user-%d", i)
				got := inBucket("salt", v, tc.percent)
				if got != tc.any {
					t.Errorf("user %q at percent %d: got %v want %v", v, tc.percent, got, tc.any)
				}
			}
		})
	}
}

func TestInBucket_RoughlyMatchesPercent(t *testing.T) {
	t.Parallel()
	// Sampling check: at percent=p, about p% of a large random sample
	// should be in-bucket. Allow a wide tolerance to keep flakiness
	// out of the artificial demo CI.
	percents := []int{10, 25, 50, 75, 90}
	for _, p := range percents {
		p := p
		t.Run(fmt.Sprintf("p=%d", p), func(t *testing.T) {
			t.Parallel()
			n := 2000
			in := 0
			for i := 0; i < n; i++ {
				if inBucket("flag-x", fmt.Sprintf("user-%d", i), p) {
					in++
				}
			}
			got := float64(in) / float64(n) * 100
			if math.Abs(got-float64(p)) > 5 {
				t.Errorf("percent=%d: observed %.1f%% (n=%d)", p, got, n)
			}
		})
	}
}

func TestRolloutBucket_LargeInputs(t *testing.T) {
	t.Parallel()
	// Long attribute values must still produce buckets in range.
	cases := []string{
		"short",
		"medium-length-user-id-1234",
		"very-very-very-very-very-very-long-user-id-1234567890-1234567890",
		"",
	}
	for _, v := range cases {
		v := v
		t.Run(fmt.Sprintf("len=%d", len(v)), func(t *testing.T) {
			t.Parallel()
			b := rolloutBucket("salt", v)
			if b < 0 || b >= 10000 {
				t.Errorf("bucket out of range: %d", b)
			}
		})
	}
}
