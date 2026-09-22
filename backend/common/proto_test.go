//nolint:revive
package common

import (
	"testing"
	"unicode/utf8"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

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
func TestSanitizeUTF8MessageCoversEveryStringField(t *testing.T) {
	// 测试 + dangling lead byte 0xb1: the exact wholly-unconverted raw GBK
	// shape go-ora returns (engine-verified against Oracle 11gR2/ZHS16GBK).
	rawGBK := "\xb2\xe2\xca\xd4\xb1"
	require.False(t, utf8.ValidString(rawGBK))

	metadata := &metadatapb.DatabaseSchemaMetadata{}
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
func TestSanitizeUTF8MessageNameFields(t *testing.T) {
	rawGBK := "AB\xe6"
	metadata := &metadatapb.DatabaseSchemaMetadata{
		Name: rawGBK,
		Schemas: []*metadatapb.SchemaMetadata{{
			Name: rawGBK,
			Tables: []*metadatapb.TableMetadata{{
				Name: rawGBK,
				Columns: []*metadatapb.ColumnMetadata{
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
