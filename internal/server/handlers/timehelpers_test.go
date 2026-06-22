package handlers

import (
	"fmt"
	"testing"
	"time"
)

func TestZeroTime(t *testing.T) {
	t.Parallel()
	if !zeroTime().IsZero() {
		t.Errorf("zeroTime is not zero")
	}
}

func TestParseTime(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		in      string
		wantZ   bool
		wantErr bool
	}{
		{"empty-returns-zero", "", true, false},
		{"rfc3339", "2026-05-26T12:34:56Z", false, false},
		{"rfc3339-nano", "2026-05-26T12:34:56.123456789Z", false, false},
		{"with-offset", "2026-05-26T12:34:56+02:00", false, false},
		{"invalid", "not a time", true, true},
		{"date-only", "2026-05-26", true, true},
		{"unix", "1700000000", true, true},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := parseTime(tc.in)
			if (err != nil) != tc.wantErr {
				t.Errorf("err = %v, wantErr = %v", err, tc.wantErr)
			}
			if tc.wantZ && !got.IsZero() {
				t.Errorf("want zero time, got %v", got)
			}
			if !tc.wantZ && got.IsZero() {
				t.Errorf("want non-zero time, got zero")
			}
		})
	}
}

func TestParseTime_RoundTrip(t *testing.T) {
	t.Parallel()
	for i := 0; i < 30; i++ {
		i := i
		t.Run(fmt.Sprintf("offset-min-%d", i), func(t *testing.T) {
			t.Parallel()
			orig := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC).Add(time.Duration(i) * time.Minute)
			s := orig.Format(time.RFC3339Nano)
			got, err := parseTime(s)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if !got.Equal(orig) {
				t.Errorf("round-trip drift: %v -> %s -> %v", orig, s, got)
			}
		})
	}
}

func TestDerefTime(t *testing.T) {
	t.Parallel()
	if !derefTime(nil).IsZero() {
		t.Errorf("derefTime(nil) is not zero")
	}
	now := time.Now().UTC()
	if !derefTime(&now).Equal(now) {
		t.Errorf("derefTime(now) lost time")
	}
}
