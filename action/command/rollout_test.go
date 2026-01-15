package command

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRenderTrain(t *testing.T) {
	// Fixed time for deterministic tests: 2026-01-15 14:30:45 UTC
	testTime := time.Date(2026, 1, 15, 14, 30, 45, 0, time.UTC)

	tests := []struct {
		name      string
		template  string
		timezone  string
		time      time.Time
		want      string
		wantError bool
	}{
		{
			name:     "date only with RC prefix",
			template: "release_{date}-RC{iteration}",
			timezone: "UTC",
			time:     testTime,
			want:     "release_20260115-RC",
		},
		{
			name:     "date and time with RC prefix",
			template: "release_{date}_{time}-RC{iteration}",
			timezone: "UTC",
			time:     testTime,
			want:     "release_20260115_1430-RC",
		},
		{
			name:     "timestamp with RC prefix",
			template: "release_{timestamp}-RC{iteration}",
			timezone: "UTC",
			time:     testTime,
			want:     "release_20260115_1430-RC",
		},
		{
			name:     "custom prefix without RC",
			template: "hello_server_{date}_{time}_{iteration}",
			timezone: "UTC",
			time:     testTime,
			want:     "hello_server_20260115_1430_",
		},
		{
			name:     "version-like format",
			template: "v{date}.{iteration}",
			timezone: "UTC",
			time:     testTime,
			want:     "v20260115.",
		},
		{
			name:     "UTC timezone",
			template: "release_{date}_{time}-RC{iteration}",
			timezone: "UTC",
			time:     testTime,
			want:     "release_20260115_1430-RC",
		},
		{
			name:     "America/Los_Angeles timezone",
			template: "release_{date}_{time}-RC{iteration}",
			timezone: "America/Los_Angeles",
			time:     testTime,
			want:     "release_20260115_0630-RC", // UTC 14:30 = PST 06:30
		},
		{
			name:     "Asia/Tokyo timezone",
			template: "release_{date}_{time}-RC{iteration}",
			timezone: "Asia/Tokyo",
			time:     testTime,
			want:     "release_20260115_2330-RC", // UTC 14:30 = JST 23:30
		},
		{
			name:     "Europe/London timezone",
			template: "release_{date}_{time}-RC{iteration}",
			timezone: "Europe/London",
			time:     testTime,
			want:     "release_20260115_1430-RC", // UTC 14:30 = GMT 14:30 (winter)
		},
		{
			name:      "invalid timezone",
			template:  "release_{date}-RC{iteration}",
			timezone:  "Invalid/Timezone",
			time:      testTime,
			wantError: true,
		},
		{
			name:      "missing iteration placeholder",
			template:  "release_{date}",
			timezone:  "UTC",
			time:      testTime,
			wantError: true,
		},
		{
			name:      "missing time variable",
			template:  "release_{iteration}",
			timezone:  "UTC",
			time:      testTime,
			wantError: true,
		},
		{
			name:     "all variables together",
			template: "{date}_{time}_{timestamp}-{iteration}",
			timezone: "UTC",
			time:     testTime,
			want:     "20260115_1430_20260115_1430-",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := renderTrain(tt.template, tt.timezone, tt.time)
			if tt.wantError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestRenderTrainTimezoneConversion(t *testing.T) {
	// Test edge case: date change across timezones
	// 2026-01-15 23:30:00 UTC
	testTime := time.Date(2026, 1, 15, 23, 30, 0, 0, time.UTC)

	tests := []struct {
		name     string
		timezone string
		wantDate string
		wantTime string
	}{
		{
			name:     "UTC - same date",
			timezone: "UTC",
			wantDate: "20260115",
			wantTime: "2330",
		},
		{
			name:     "America/Los_Angeles - same date (earlier)",
			timezone: "America/Los_Angeles",
			wantDate: "20260115",
			wantTime: "1530",
		},
		{
			name:     "Asia/Tokyo - next date",
			timezone: "Asia/Tokyo",
			wantDate: "20260116",
			wantTime: "0830",
		},
		{
			name:     "Australia/Sydney - next date",
			timezone: "Australia/Sydney",
			wantDate: "20260116",
			wantTime: "1030",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			template := "release_{date}_{time}-RC{iteration}"
			got, err := renderTrain(template, tt.timezone, testTime)
			require.NoError(t, err)
			want := "release_" + tt.wantDate + "_" + tt.wantTime + "-RC"
			require.Equal(t, want, got)
		})
	}
}

func TestValidateTemplate(t *testing.T) {
	tests := []struct {
		name      string
		template  string
		wantError bool
		errorMsg  string
	}{
		{
			name:      "valid with date and iteration",
			template:  "release_{date}-RC{iteration}",
			wantError: false,
		},
		{
			name:      "valid with time and iteration",
			template:  "release_{time}-RC{iteration}",
			wantError: false,
		},
		{
			name:      "valid with timestamp and iteration",
			template:  "release_{timestamp}-RC{iteration}",
			wantError: false,
		},
		{
			name:      "valid with all variables",
			template:  "{date}_{time}_{timestamp}-{iteration}",
			wantError: false,
		},
		{
			name:      "valid date only",
			template:  "v{date}.{iteration}",
			wantError: false,
		},
		{
			name:      "missing iteration",
			template:  "release_{date}",
			wantError: true,
			errorMsg:  "must contain {iteration} placeholder",
		},
		{
			name:      "missing time variable",
			template:  "release_{iteration}",
			wantError: true,
			errorMsg:  "must contain at least one of: {date}, {time}, {timestamp}",
		},
		{
			name:      "only iteration",
			template:  "{iteration}",
			wantError: true,
			errorMsg:  "must contain at least one of: {date}, {time}, {timestamp}",
		},
		{
			name:      "empty template",
			template:  "",
			wantError: true,
			errorMsg:  "must contain {iteration} placeholder",
		},
		{
			name:      "iteration not at end",
			template:  "release_{iteration}_{date}",
			wantError: true,
			errorMsg:  "{iteration} must be at the end of the template",
		},
		{
			name:      "iteration in middle",
			template:  "{date}_{iteration}_suffix",
			wantError: true,
			errorMsg:  "{iteration} must be at the end of the template",
		},
		{
			name:      "iteration with text after",
			template:  "release_{date}-RC{iteration}-final",
			wantError: true,
			errorMsg:  "{iteration} must be at the end of the template",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTemplate(tt.template)
			if tt.wantError {
				require.Error(t, err)
				if tt.errorMsg != "" {
					require.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}
