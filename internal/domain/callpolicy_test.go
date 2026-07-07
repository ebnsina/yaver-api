package domain

import (
	"testing"
	"time"
)

func TestCallPolicyAllows(t *testing.T) {
	dhaka, err := time.LoadLocation("Asia/Dhaka")
	if err != nil {
		t.Skip("tzdata unavailable")
	}
	p := DefaultCallPolicy() // 09:00–21:00 Asia/Dhaka

	cases := []struct {
		name string
		when time.Time
		want bool
	}{
		{"mid-morning allowed", time.Date(2026, 7, 7, 10, 0, 0, 0, dhaka), true},
		{"just before open blocked", time.Date(2026, 7, 7, 8, 59, 0, 0, dhaka), false},
		{"at open allowed", time.Date(2026, 7, 7, 9, 0, 0, 0, dhaka), true},
		{"at close blocked", time.Date(2026, 7, 7, 21, 0, 0, 0, dhaka), false},
		{"late night blocked", time.Date(2026, 7, 7, 2, 0, 0, 0, dhaka), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := p.Allows(tc.when); got != tc.want {
				t.Errorf("Allows(%v) = %v, want %v", tc.when, got, tc.want)
			}
		})
	}
}

func TestCallPolicyAllowsUsesTimezone(t *testing.T) {
	dhaka, err := time.LoadLocation("Asia/Dhaka")
	if err != nil {
		t.Skip("tzdata unavailable")
	}
	// 10:00 UTC is 16:00 in Dhaka (UTC+6) — inside the window.
	utc10 := time.Date(2026, 7, 7, 10, 0, 0, 0, time.UTC)
	if !DefaultCallPolicy().Allows(utc10) {
		t.Errorf("10:00 UTC should be 16:00 Dhaka (allowed)")
	}
	_ = dhaka
}

func TestCallPolicyValid(t *testing.T) {
	cases := []struct {
		name string
		p    CallPolicy
		want bool
	}{
		{"default", DefaultCallPolicy(), true},
		{"wrap-around window", CallPolicy{WindowStart: 21, WindowEnd: 9, Timezone: "Asia/Dhaka"}, false},
		{"equal window", CallPolicy{WindowStart: 9, WindowEnd: 9, Timezone: "Asia/Dhaka"}, false},
		{"hour out of range", CallPolicy{WindowStart: 9, WindowEnd: 25, Timezone: "Asia/Dhaka"}, false},
		{"bad timezone", CallPolicy{WindowStart: 9, WindowEnd: 21, Timezone: "Mars/Olympus"}, false},
		{"too many retries", CallPolicy{WindowStart: 9, WindowEnd: 21, Timezone: "UTC", MaxRetries: 99}, false},
		{"negative retries", CallPolicy{WindowStart: 9, WindowEnd: 21, Timezone: "UTC", MaxRetries: -1}, false},
		{"ok utc", CallPolicy{WindowStart: 0, WindowEnd: 24, Timezone: "UTC", MaxRetries: 3}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.p.Valid(); got != tc.want {
				t.Errorf("Valid() = %v, want %v", got, tc.want)
			}
		})
	}
}
