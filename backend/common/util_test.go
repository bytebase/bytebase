package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHasPrefixes(t *testing.T) {
	type args struct {
		src      string
		prefixes []string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "has prefixes",
			args: args{
				src:      "abc",
				prefixes: []string{"a", "b", "c"},
			},
			want: true,
		},
		{
			name: "has no matching prefix",
			args: args{
				src:      "this is a sentence",
				prefixes: []string{"that", "x", "y"},
			},
			want: false,
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			got := HasPrefixes(tt.args.src, tt.args.prefixes...)
			assert.Equal(t, got, tt.want)
		})
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		name      string
		str       string
		limit     int
		want      string
		truncated bool
	}{
		{
			name:      "simple truncate 0",
			str:       "0123",
			limit:     0,
			want:      "",
			truncated: true,
		},
		{
			name:      "simple truncate 2",
			str:       "0123",
			limit:     2,
			want:      "01",
			truncated: true,
		},
		{
			name:      "simple truncate 3",
			str:       "0123",
			limit:     3,
			want:      "012",
			truncated: true,
		},
		{
			name:      "simple truncate 4",
			str:       "0123",
			limit:     4,
			want:      "0123",
			truncated: false,
		},
		{
			name:      "simple truncate 20",
			str:       "0123",
			limit:     20,
			want:      "0123",
			truncated: false,
		},
		{
			name:      "unicode truncate 5",
			str:       "H㐀〾▓朗퐭텟şüöžåйкл¤",
			limit:     5,
			want:      "H㐀〾▓朗",
			truncated: true,
		},
		{
			name:      "unicode truncate 10",
			str:       "H㐀〾▓朗퐭텟şüöžåйкл¤",
			limit:     10,
			want:      "H㐀〾▓朗퐭텟şüö",
			truncated: true,
		},
		{
			name:      "unicode fit",
			str:       "H㐀〾▓朗퐭텟şüöžåйкл¤",
			limit:     16,
			want:      "H㐀〾▓朗퐭텟şüöžåйкл¤",
			truncated: false,
		},
	}
	a := assert.New(t)
	for i := range tests {
		test := tests[i]
		t.Run(test.name, func(t *testing.T) {
			got, truncated := TruncateString(test.str, test.limit)
			a.Equal(test.want, got)
			a.Equal(test.truncated, truncated)
		})
	}
}

func TestObfuscate(t *testing.T) {
	tests := []struct {
		src  string
		seed string
		dst  string
	}{
		{
			src:  "",
			seed: "01234567890123456789012345678901", // 32 bytes.
			dst:  "",
		},
		{
			src:  "hello",
			seed: "01234567890123456789012345678901", // 32 bytes.
			dst:  "WFReX1s=",
		},
		{
			src:  "你好!",
			seed: "ENuef1JjSvQ6VPfgrB33T2mkshhwRRjp", // 32 bytes.
			dst:  "ofPVgMOMaw==",
		},
		{
			src:  "Bytebase is a database tool for developers. Bytebase 是个数据库 DevOps 工具。",
			seed: "01234567890123456789012345678901", // 32 bytes.
			dst:  "ckhGVlZURVIYUEMRUxNQVEJWWlhDVBJHW1paF15WQhFUVERWWFpGUkpKHhFwSkBQVFZLXBDXqpzQjZzRrYnWvJ7UiKAUcVNBd0lDEdeEkdCzgNu5sg==",
		},
		{
			src:  `{   "type": "service_account",   "project_id": "spanner-test-371702",   "private_key_id": "klsdjfklasjdfas\nklsdjaflkajefjlaksdjf\nlsajdfklsjaldkfjkasldjf\nD PRIVATE KEY-----\n,   "client_email": "test-768@spanner-test-371702.iam.gserviceaccount.com",   "client_id": "102052620181224568340",   "auth_uri": "https://accounts.google.com/o/oauth2/auth",   "token_uri": "https://oauth2.googleapis.com/token",   "auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",   "client_x509_cert_url": "https://www.googleapis.com/robot/v1/metadata/x509/test-768%40spanner-test-371702.iam.gserviceaccount.com" }`,
			seed: `aGgQpKjg7fuwNV6B31sIRQ1qm4Ttqw9s`, // 32 bytes.
			dst:  `GmdHcVI/ExdSRE9XbCVTMEVYECwNMFISAkE6AFNbGVNBZRcjHyEPBEM5HBNsbBZgQEESJzw0Q1wZUScAXEQOQlZ3VXNca0pHFRYHHjg3QidsWhYwDThVU1cUdh8dBF0ZBywLMAMhDgFWFSkZJTpFJllQFSU5MFsUC144FRoEXRkHGwk9AyoAA1ENGQQkN1omWFcZIjMiXRUHUggaNVdpISgRJgU1ayEibktYWmN7aiwfEVNpcDJdGAhaICsUGlgaDWVdcVI/DxRDS0JBdhZFMlJfHSwgfEUUHkB5R0ZGDkNTaQ4wHWUNFFIUAx4tM1chUF4GJyZ/Uh4AFnhUUVcbEA0uAj8EFAMDFVxVVX9mBHIGA0V7YmAJQF8GYEFHTwpHUWVLcVBrSAZCEh0oOyRfYAkRUSEmJUECVxt7FRIUVgYPMxR/FyQFAFsDWxQhOxktHF4SPCY5A14MQSAcU1sZU0FlEz4bLgQ4QhQcVXR2FCpHRQM6aH4eHgxBIBxDWV4cDiALNBE7AxQZBRoaYSJZKVZfUWVycRFTDEEgHC4HSxwXLgM0AhQSUgdfKhQrJEIdRkMfa2hxExkZQCQHS1gWBBYwSTYfJA0LUgcFHj14VS1eHhwoJyVZQ0JCZVsSEksHEmVLcVBrSARbDxAZOglOdwMILCo3I0UuGEY4VktXGxsVMxciSmRFEEARWxAhOVEuVlADICF/Uh4AGyYbExhNXBd2SDwVPwsDVhIUWDZjBnscRRY6JnwGR1URYEQCB1gdDyIVfAQuGRMaVUJGeWYEbFpQHmc1IlQDG103ERAUWhwUKRN/EyQHRRcb`,
		},
	}
	for _, test := range tests {
		obfuscated := Obfuscate(test.src, test.seed)
		require.Equal(t, test.dst, obfuscated)
		ubobfuscated, err := Unobfuscate(obfuscated, test.seed)
		require.NoError(t, err)
		require.Equal(t, test.src, ubobfuscated)
	}
}

func TestNormalizeExternalURL(t *testing.T) {
	tests := []struct {
		url     string
		want    string
		wantErr bool
	}{
		{
			url:     "http://localhost:3000",
			want:    "http://localhost:3000",
			wantErr: false,
		},
		{
			url:     "https://localhost:3000",
			want:    "https://localhost:3000",
			wantErr: false,
		},
		{
			url:     "https://localhost",
			want:    "https://localhost",
			wantErr: false,
		},
		{
			url:     "http://localhost:80",
			want:    "http://localhost",
			wantErr: false,
		},
		{
			url:     "https://localhost:443",
			want:    "https://localhost",
			wantErr: false,
		},
		{
			url:     "  https://localhost:3000/ ",
			want:    "https://localhost:3000",
			wantErr: false,
		},
		// Missing http:// or https://
		{
			url:     "localhost:3000",
			want:    "",
			wantErr: true,
		},
		// Invalid port
		{
			url:     "http://localhost:xxx",
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			g, err := NormalizeExternalURL(tt.url)
			if err != nil {
				if !tt.wantErr {
					t.Errorf("expect no error, got %s", err.Error())
				}
			} else {
				if tt.wantErr {
					t.Errorf("expect error")
				} else if tt.want != g {
					t.Errorf("expect %s, got %s", tt.want, g)
				}
			}
		})
	}
}

func TestValidatePhone(t *testing.T) {
	tests := []struct {
		phone string
		want  bool
	}{
		{
			phone: "1234567890",
			want:  false,
		},
		{
			phone: "+8615655556666",
			want:  true,
		},
	}

	for _, test := range tests {
		got := ValidatePhone(test.phone)
		isValid := got == nil
		if isValid != test.want {
			t.Errorf("validatePhone %s, err %v", test.phone, got)
		}
	}
}
