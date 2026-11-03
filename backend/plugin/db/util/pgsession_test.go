package util

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestPgLocation(t *testing.T) {
	// A winter instant, so the zones with DST report their standard offset.
	instant := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name       string
		setting    string
		wantOffset int
	}{
		{name: "UTC", setting: "UTC", wantOffset: 0},
		{name: "unreported setting", setting: "", wantOffset: 0},
		{name: "whole hour", setting: "Asia/Shanghai", wantOffset: 8 * 60 * 60},
		{name: "half hour", setting: "Asia/Kolkata", wantOffset: 5*60*60 + 30*60},
		{name: "three quarter hour", setting: "Pacific/Chatham", wantOffset: 13*60*60 + 45*60},
		{name: "negative", setting: "America/New_York", wantOffset: -5 * 60 * 60},
		{name: "posix name", setting: "EST5EDT", wantOffset: -5 * 60 * 60},
		// PostgreSQL accepts these; time.LoadLocation does not.
		{name: "posix offset falls back to UTC", setting: "+08:00", wantOffset: 0},
		{name: "posix rules fall back to UTC", setting: "EST5EDT,M3.2.0/2,M11.1.0/2", wantOffset: 0},
		{name: "unknown zone falls back to UTC", setting: "Mars/Olympus_Mons", wantOffset: 0},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, offset := instant.In(pgLocation(test.setting)).Zone()
			require.Equal(t, test.wantOffset, offset)
		})
	}
}

func TestPgLocationCachesByName(t *testing.T) {
	first := pgLocation("Australia/Eucla")
	require.Same(t, first, pgLocation("Australia/Eucla"))

	// A fallback is cached too, so a POSIX setting costs one failed load.
	fallback := pgLocation("-04:30")
	require.Same(t, time.UTC, fallback)
	require.Same(t, fallback, pgLocation("-04:30"))
}

func TestPgLocationAppliesDaylightSavingPerInstant(t *testing.T) {
	location := pgLocation("America/New_York")

	_, winter := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC).In(location).Zone()
	_, summer := time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC).In(location).Zone()

	require.Equal(t, -5*60*60, winter)
	require.Equal(t, -4*60*60, summer)
}
