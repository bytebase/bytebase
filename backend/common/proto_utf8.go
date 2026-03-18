//nolint:revive
package common

import (
	"strings"
	"unicode/utf8"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// SanitizeProtoStringFields recursively walks all string fields in a proto
// message and replaces invalid UTF-8 bytes with empty string (removal).
func SanitizeProtoStringFields(msg proto.Message) {
	if msg == nil {
		return
	}
	v := msg.ProtoReflect()
	if !v.IsValid() {
		return
	}
	sanitizeMessage(v)
}

func sanitizeMessage(m protoreflect.Message) {
	m.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		switch {
		case fd.IsList():
			sanitizeList(fd, v.List())
		case fd.IsMap():
			sanitizeMap(fd, v.Map())
		case fd.Kind() == protoreflect.StringKind:
			s := v.String()
			if !utf8.ValidString(s) {
				m.Set(fd, protoreflect.ValueOfString(strings.ToValidUTF8(s, "")))
			}
		case fd.Kind() == protoreflect.MessageKind || fd.Kind() == protoreflect.GroupKind:
			sanitizeMessage(v.Message())
		}
		return true
	})
}

func sanitizeList(fd protoreflect.FieldDescriptor, list protoreflect.List) {
	for i := 0; i < list.Len(); i++ {
		el := list.Get(i)
		switch {
		case fd.Kind() == protoreflect.StringKind:
			s := el.String()
			if !utf8.ValidString(s) {
				list.Set(i, protoreflect.ValueOfString(strings.ToValidUTF8(s, "")))
			}
		case fd.Kind() == protoreflect.MessageKind || fd.Kind() == protoreflect.GroupKind:
			sanitizeMessage(el.Message())
		}
	}
}

func sanitizeMap(fd protoreflect.FieldDescriptor, m protoreflect.Map) {
	valueFD := fd.MapValue()

	// Collect keys that contain invalid UTF-8 so we can fix them after iteration.
	type badKey struct {
		original protoreflect.MapKey
		cleaned  string
		value    protoreflect.Value
	}
	var badKeys []badKey

	m.Range(func(k protoreflect.MapKey, v protoreflect.Value) bool {
		ks := k.String()
		if !utf8.ValidString(ks) {
			badKeys = append(badKeys, badKey{
				original: k,
				cleaned:  strings.ToValidUTF8(ks, ""),
				value:    v,
			})
		}

		switch {
		case valueFD.Kind() == protoreflect.StringKind:
			s := v.String()
			if !utf8.ValidString(s) {
				m.Set(k, protoreflect.ValueOfString(strings.ToValidUTF8(s, "")))
			}
		case valueFD.Kind() == protoreflect.MessageKind || valueFD.Kind() == protoreflect.GroupKind:
			sanitizeMessage(v.Message())
		}
		return true
	})

	// Fix bad keys: remove old key and insert cleaned key.
	for _, bk := range badKeys {
		val := bk.value
		// Re-read the value in case it was also sanitized during the range.
		if m.Has(bk.original) {
			val = m.Get(bk.original)
		}
		m.Clear(bk.original)
		m.Set(protoreflect.ValueOfString(bk.cleaned).MapKey(), val)
	}
}
