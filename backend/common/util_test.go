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
			src:  `{   "type": "service_account",   "project_id": "spanner-test-371702",   "private_key_id": "9b493f85c24f7a489006f340511709a620801a84",   "private_key": "-----BEGIN PRIVATE KEY-----\nMIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQDDS4j6/84ZD83Q\nJa+8KIoZ9K5F/cRgcdcIr9uFHywXpOkWTCpk4yDtHv/o7F3WfjT0gs8AJHpaqivP\nsn4p30ZFcrXbL/cQ0yws8zYInu5Obr0YJaUpqlN93IDFfQeWD4Fm9Pi3ZlQQztTD\nd/mdfhhF7bzkzeDfdOAw1485okRVTldWFEpil3dmbFy/TxJVlmI6yGvMrA+ndtAy\nFJKkY0z3RU/R9d2mlLTaETDp7tGwwphRQsKOGjB8Na3OpLGCDst81aDrlhksYDiW\nQqO6nRI1ROdjq0aOE2uLAiklp1A26tQt1uUESByteZoafgtbfRecrW13V9olESuz\nD2lXNY/zAgMBAAECggEAUqD0aYJu4HKr2VwAoSMj3zyDudBXVJkF6sKBJi9ct/pJ\ntmnua/YfWcRKcXIEd/q5DBUGYSCN9itW0w9JywyRTlPs9rqyFZfLco9KttYLrPq0\nnfFbkqfioHd5slUmrwqLbRTL4Mj2W5AvPC7YWZbezUN7rvEeqlzoeDHUAwYRVGfN\np/6OSQTZHdMZqfbsLVtsFsc81jEykS2+Df3fFF/zGbh0loFlTl83u/+JYozqg1eZ\nfKh3+KR1pLUuJHsYawkTT4vukZwhjZaJnyo12rKiKfi8zs144X+s0BmNg5Oil24F\n7UYfXwsPXeVCqSSpzL1ZSYMNXJEHSFzbinMOsQ3d7QKBgQDzzmXveNTwz5JRHHmM\nZ99U5TdCY1uiHP6ooWnUWMalpA1cyq7zpss5Gx4Pf3I21fnX6m2OQGveVv3b0cs+\n8a2HE5UKvQHYZw7rpm2G2Y5aIjE3ZNsU5LXN37uL0Ra+F52E5eYC47AbWlYG4hfz\n9su1r5qeGTObXAdyVox4yC7A1wKBgQDNEASs33SOmyIxJYk651gChqEms8vLDC1C\nZGmIzihRD479RakL95yOwgtwwhqViyrwgl07pGRfGyk3TWUkwJY9SrbGHIhXfCn7\nhhwEbZMMSWRTOmHxoEahBu+mLamogW1hMsVJY9MX3bOyVGb4Dz3CtdspvN8VELBt\n7OZ9ubBaRQKBgAh7XcB/C6l1DzoTK4de9b4WW13L5xw0tgdX1j609/Q7SNu5kWyY\nmOlbsCgJ3wdZWl/QoA8a3qXVkO9c1R1Tex3/6Gd/O9kzfKlmGNlgKDuqhNvQfm6z\npj+LURMEKy5h0/ETrnTbRv0sn2GN7Bdotp2ThmWJqun0wa2QpUJudHHxAoGBAIGF\nJo7SLNqN3dDQ9pZ/3LTruAmr8oJzVHrk1UuVex2ICDassxNd+EKrCXLVBtmBp0N1\n89FiCguQKj5F4iaOhdZ8xGjpSKyJPjMiB7w8QW63RGjVLVvicfnvWZrKqKhH54BH\nAxlRtdkTLRbr/IWditLa7my0YOr7OZSU1xh+GodJAoGANZUaFjdXDJaTiPBeAXps\neHIvKsNfoIthew4eQ+R4wApN60/HNLCIxp35vXkArB1Q10ckcfyYedGUy2jemdTq\nZ71yFMAB18NIWBfxtSVKiqKVFUBkToOdSX1SDrz0+v06dNkulkU3G/GbsvJe2m3d\nNndTzwMa3NtYJl9/OpZvr+E=\n-----END PRIVATE KEY-----\n",   "client_email": "test-768@spanner-test-371702.iam.gserviceaccount.com",   "client_id": "102052620181224568340",   "auth_uri": "https://accounts.google.com/o/oauth2/auth",   "token_uri": "https://oauth2.googleapis.com/token",   "auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",   "client_x509_cert_url": "https://www.googleapis.com/robot/v1/metadata/x509/test-768%40spanner-test-371702.iam.gserviceaccount.com" }`,
			seed: `aGgQpKjg7fuwNV6B31sIRQ1qm4Ttqw9s`, // 32 bytes.
			dst:  `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`,
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
