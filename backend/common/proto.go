//nolint:revive
package common

import (
	"unicode/utf8"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// SanitizeUTF8Message replaces invalid UTF-8 byte sequences in every
// populated string field of m, recursing into nested messages, repeated
// fields, and maps (both keys and values). proto3 requires string fields to
// hold valid UTF-8, so a single raw byte sequence smuggled in by an external
// system (e.g. a database driver passing through unconverted bytes) makes
// proto.Marshal fail for the entire message. Callers that assemble metadata
// from such sources sanitize the finished message once here instead of
// chasing every scan site.
func SanitizeUTF8Message(m proto.Message) {
	if m == nil {
		return
	}
	sanitizeUTF8Value(m.ProtoReflect())
}

func sanitizeUTF8Value(m protoreflect.Message) {
	m.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		switch {
		case fd.IsMap():
			sanitizeUTF8Map(fd, v.Map())
		case fd.IsList():
			list := v.List()
			switch fd.Kind() {
			case protoreflect.StringKind:
				for i := 0; i < list.Len(); i++ {
					s := list.Get(i).String()
					if !utf8.ValidString(s) {
						list.Set(i, protoreflect.ValueOfString(SanitizeUTF8String(s)))
					}
				}
			case protoreflect.MessageKind, protoreflect.GroupKind:
				for i := 0; i < list.Len(); i++ {
					sanitizeUTF8Value(list.Get(i).Message())
				}
			default:
			}
		case fd.Kind() == protoreflect.StringKind:
			if s := v.String(); !utf8.ValidString(s) {
				m.Set(fd, protoreflect.ValueOfString(SanitizeUTF8String(s)))
			}
		case fd.Kind() == protoreflect.MessageKind || fd.Kind() == protoreflect.GroupKind:
			sanitizeUTF8Value(v.Message())
		default:
		}
		return true
	})
}

func sanitizeUTF8Map(fd protoreflect.FieldDescriptor, mp protoreflect.Map) {
	valueKind := fd.MapValue().Kind()
	keyIsString := fd.MapKey().Kind() == protoreflect.StringKind

	type rekey struct {
		oldKey protoreflect.MapKey
		newKey protoreflect.MapKey
		value  protoreflect.Value
	}
	var rekeys []rekey
	mp.Range(func(k protoreflect.MapKey, v protoreflect.Value) bool {
		switch valueKind {
		case protoreflect.StringKind:
			if s := v.String(); !utf8.ValidString(s) {
				mp.Set(k, protoreflect.ValueOfString(SanitizeUTF8String(s)))
			}
		case protoreflect.MessageKind, protoreflect.GroupKind:
			sanitizeUTF8Value(v.Message())
		default:
		}
		if keyIsString {
			if s := k.String(); !utf8.ValidString(s) {
				rekeys = append(rekeys, rekey{
					oldKey: k,
					newKey: protoreflect.ValueOfString(SanitizeUTF8String(s)).MapKey(),
					value:  mp.Get(k),
				})
			}
		}
		return true
	})
	for _, r := range rekeys {
		mp.Clear(r.oldKey)
		mp.Set(r.newKey, r.value)
	}
}
