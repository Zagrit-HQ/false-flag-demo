// Package eval evaluates a compiled flag IR against an evaluation
// context and returns a Decision. The behavior must match the JS
// evaluator in js/packages/sdk-js/src/evaluator.ts byte-for-byte; the
// cross-runtime golden corpus at tests/eval-corpus/ asserts that.
package eval

import "hash/fnv"

// rolloutBucket returns the bucket [0,10000) for the given salt and
// attribute value. Both the Go and JS implementations use FNV-1a 64-bit
// over salt + ":" + attrValue. Modulo 10000 gives 4-digit precision,
// which is enough for the percent gate but small enough to keep the
// arithmetic identical between the two runtimes.
//
// The JS implementation lives in
// js/packages/sdk-js/src/rollout.ts — keep these in sync.
func rolloutBucket(salt, attrValue string) int {
	h := fnv.New64a()
	_, _ = h.Write([]byte(salt))
	_, _ = h.Write([]byte{':'})
	_, _ = h.Write([]byte(attrValue))
	return int(h.Sum64() % 10000)
}

// inBucket returns true if value's bucket falls within the given
// percent gate. percent is [0,100]; the gate is half-open: percent=0
// means nobody is in, percent=100 means everybody is in.
func inBucket(salt, attrValue string, percent int) bool {
	if percent <= 0 {
		return false
	}
	if percent >= 100 {
		return true
	}
	return rolloutBucket(salt, attrValue) < percent*100
}
