package handlers

import "time"

// Small time helpers kept separate from per-resource handler files so
// they don't add noise to the read.

type auditTime = time.Time

func zeroTime() time.Time { return time.Time{} }

func parseTime(s string) (time.Time, error) {
	if s == "" {
		return zeroTime(), nil
	}
	return time.Parse(time.RFC3339Nano, s)
}

func derefTime(p *time.Time) time.Time {
	if p == nil {
		return zeroTime()
	}
	return *p
}
