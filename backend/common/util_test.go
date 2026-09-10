//nolint:revive
package common

import (
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
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
		t.Run(test.name, func(_ *testing.T) {
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
		{
			url:     "HTTPS://LOCALHOST:443/mcp/",
			want:    "https://localhost/mcp",
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
		{
			url:     "https://user@localhost",
			want:    "",
			wantErr: true,
		},
		{
			url:     "https://localhost?x=1",
			want:    "",
			wantErr: true,
		},
		{
			url:     "https://localhost#fragment",
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
					t.Error("expect error")
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

// TestSanitizeUTF8String pins the replacement shape: the invalid bytes survive
// as their hex escape so a corrupted value stays diagnosable after sanitizing.
func TestSanitizeUTF8String(t *testing.T) {
	// Vietnamese text encoded in Windows-1258 and stored in an AL32UTF8
	// database by a misconfigured client: 0xe1 is a valid lead byte with no
	// continuation.
	const corrupted = "Tr\xe1ng th\xe1i"
	require.False(t, utf8.ValidString(corrupted), "test precondition")

	sanitized := SanitizeUTF8String(corrupted)

	require.True(t, utf8.ValidString(sanitized))
	require.Contains(t, sanitized, "\\xe1")
	require.Equal(t, "\u6d4b\u8bd5", SanitizeUTF8String("\u6d4b\u8bd5"), "valid UTF-8 must pass through unchanged")
}

// fillAllStringFields recursively sets every string field reachable from m
// (including nested messages, repeated fields, and map keys/values) to the
// given value, instantiating one element for each nested message, list, and
// map so that no string field in the schema shape stays unvisited.
func fillAllStringFields(m protoreflect.Message, value string, depth int) {
	if depth <= 0 {
		return
	}
	fields := m.Descriptor().Fields()
	for i := 0; i < fields.Len(); i++ {
		fd := fields.Get(i)
		switch {
		case fd.IsMap():
			mp := m.Mutable(fd).Map()
			var key protoreflect.MapKey
			if fd.MapKey().Kind() == protoreflect.StringKind {
				key = protoreflect.ValueOfString(value).MapKey()
			} else {
				key = protoreflect.ValueOfInt64(1).MapKey()
			}
			switch fd.MapValue().Kind() {
			case protoreflect.StringKind:
				mp.Set(key, protoreflect.ValueOfString(value))
			case protoreflect.MessageKind:
				nv := mp.NewValue()
				fillAllStringFields(nv.Message(), value, depth-1)
				mp.Set(key, nv)
			default:
				mp.Set(key, mp.NewValue())
			}
		case fd.IsList():
			list := m.Mutable(fd).List()
			switch fd.Kind() {
			case protoreflect.StringKind:
				list.Append(protoreflect.ValueOfString(value))
			case protoreflect.MessageKind:
				nv := list.NewElement()
				fillAllStringFields(nv.Message(), value, depth-1)
				list.Append(nv)
			default:
			}
		case fd.Kind() == protoreflect.StringKind:
			m.Set(fd, protoreflect.ValueOfString(value))
		case fd.Kind() == protoreflect.MessageKind:
			fillAllStringFields(m.Mutable(fd).Message(), value, depth-1)
		default:
		}
	}
}

// countInvalidStringFields walks m and returns how many string values
// (fields, list elements, map keys and values) hold invalid UTF-8.

// countInvalidStringFields walks m and returns how many string values
// (fields, list elements, map keys and values) hold invalid UTF-8.
func countInvalidStringFields(m protoreflect.Message) int {
	count := 0
	m.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		switch {
		case fd.IsMap():
			v.Map().Range(func(k protoreflect.MapKey, mv protoreflect.Value) bool {
				if fd.MapKey().Kind() == protoreflect.StringKind && !utf8.ValidString(k.String()) {
					count++
				}
				switch fd.MapValue().Kind() {
				case protoreflect.StringKind:
					if !utf8.ValidString(mv.String()) {
						count++
					}
				case protoreflect.MessageKind:
					count += countInvalidStringFields(mv.Message())
				default:
				}
				return true
			})
		case fd.IsList():
			list := v.List()
			for i := 0; i < list.Len(); i++ {
				switch fd.Kind() {
				case protoreflect.StringKind:
					if !utf8.ValidString(list.Get(i).String()) {
						count++
					}
				case protoreflect.MessageKind:
					count += countInvalidStringFields(list.Get(i).Message())
				default:
				}
			}
		case fd.Kind() == protoreflect.StringKind:
			if !utf8.ValidString(v.String()) {
				count++
			}
		case fd.Kind() == protoreflect.MessageKind:
			count += countInvalidStringFields(v.Message())
		default:
		}
		return true
	})
	return count
}

// TestSanitizeUTF8MessageCoversEveryStringField proves the BYT-9916 fix
// covers every string field of the database metadata — names, types,
// defaults, definitions — not just comments: fill each one with the raw-GBK
// shape go-ora leaks for values ending in a dangling lead byte, then assert
// sanitization leaves zero invalid strings and proto marshaling succeeds.

// TestSanitizeUTF8MessageCoversEveryStringField proves the BYT-9916 fix
// covers every string field of the database metadata — names, types,
// defaults, definitions — not just comments: fill each one with the raw-GBK
// shape go-ora leaks for values ending in a dangling lead byte, then assert
// sanitization leaves zero invalid strings and proto marshaling succeeds.
func TestSanitizeUTF8MessageCoversEveryStringField(t *testing.T) {
	// 测试 + dangling lead byte 0xb1: the exact wholly-unconverted raw GBK
	// shape go-ora returns (engine-verified against Oracle 11gR2/ZHS16GBK).
	rawGBK := "\xb2\xe2\xca\xd4\xb1"
	require.False(t, utf8.ValidString(rawGBK))

	metadata := &storepb.DatabaseSchemaMetadata{}
	fillAllStringFields(metadata.ProtoReflect(), rawGBK, 8)

	invalidBefore := countInvalidStringFields(metadata.ProtoReflect())
	require.Greater(t, invalidBefore, 100,
		"filler must reach the full schema shape; got only %d string fields", invalidBefore)
	_, err := proto.Marshal(metadata)
	require.Error(t, err, "pre-sanitize marshal must fail — otherwise this test proves nothing")

	SanitizeUTF8Message(metadata)

	require.Equal(t, 0, countInvalidStringFields(metadata.ProtoReflect()),
		"sanitization must leave zero invalid string values anywhere in the message")
	_, err = proto.Marshal(metadata)
	require.NoError(t, err)
	_, err = protojson.Marshal(metadata)
	require.NoError(t, err)
}

// TestSanitizeUTF8MessageNameFields pins the customer-visible BYT-9916 shape:
// object names (schema/table/column) carrying raw GBK bytes must marshal
// after sanitization, and valid names must pass through untouched.

// TestSanitizeUTF8MessageNameFields pins the customer-visible BYT-9916 shape:
// object names (schema/table/column) carrying raw GBK bytes must marshal
// after sanitization, and valid names must pass through untouched.
func TestSanitizeUTF8MessageNameFields(t *testing.T) {
	rawGBK := "AB\xe6"
	metadata := &storepb.DatabaseSchemaMetadata{
		Name: rawGBK,
		Schemas: []*storepb.SchemaMetadata{{
			Name: rawGBK,
			Tables: []*storepb.TableMetadata{{
				Name: rawGBK,
				Columns: []*storepb.ColumnMetadata{
					{Name: rawGBK, Type: rawGBK, Default: rawGBK},
					{Name: "测试列", Type: "VARCHAR2(50)"},
				},
			}},
		}},
	}
	_, err := proto.Marshal(metadata)
	require.Error(t, err)

	SanitizeUTF8Message(metadata)
	_, err = proto.Marshal(metadata)
	require.NoError(t, err)
	require.Equal(t, "测试列", metadata.Schemas[0].Tables[0].Columns[1].Name,
		"valid UTF-8 must pass through unchanged")
	require.True(t, utf8.ValidString(metadata.Schemas[0].Tables[0].Name))
}
