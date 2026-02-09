package server

import (
	"cmp"
	"fmt"
	"io"
	"regexp"
	"slices"
	"strings"

	grpcruntime "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
)

// suggestingMarshaler wraps a grpc-gateway Marshaler and enhances "unknown field"
// errors with suggestions for similar valid field names.
type suggestingMarshaler struct {
	grpcruntime.Marshaler
}

func newSuggestingMarshaler(inner grpcruntime.Marshaler) *suggestingMarshaler {
	return &suggestingMarshaler{Marshaler: inner}
}

func (m *suggestingMarshaler) NewDecoder(r io.Reader) grpcruntime.Decoder {
	return &suggestingDecoder{
		inner: m.Marshaler.NewDecoder(r),
	}
}

// suggestingDecoder wraps a Decoder and enhances unknown field errors.
type suggestingDecoder struct {
	inner grpcruntime.Decoder
}

// unknownFieldRe matches protojson errors like:
//
//	unknown field "target"
var unknownFieldRe = regexp.MustCompile(`unknown field "([^"]+)"`)

func (d *suggestingDecoder) Decode(v any) error {
	err := d.inner.Decode(v)
	if err == nil {
		return nil
	}

	match := unknownFieldRe.FindStringSubmatch(err.Error())
	if match == nil {
		return err
	}
	unknownField := match[1]

	msg, ok := v.(proto.Message)
	if !ok {
		return err
	}

	fields := msg.ProtoReflect().Descriptor().Fields()
	var candidates []string
	for i := 0; i < fields.Len(); i++ {
		// protojson uses camelCase JSON names.
		candidates = append(candidates, fields.Get(i).JSONName())
	}

	suggestions := findSimilar(unknownField, candidates, 3)
	if len(suggestions) == 0 {
		return errors.Wrapf(err, ". Valid fields: %s", strings.Join(candidates, ", "))
	}
	return errors.Wrapf(err, ". Did you mean: %s?", strings.Join(suggestions, ", "))
}

// findSimilar returns up to maxResults field names sorted by edit distance.
// Only fields within a reasonable distance threshold are included.
func findSimilar(input string, candidates []string, maxResults int) []string {
	type scored struct {
		name string
		dist int
	}
	input = strings.ToLower(input)
	var results []scored
	for _, c := range candidates {
		d := levenshtein(input, strings.ToLower(c))
		// Threshold: at most half the length of the longer string + 1.
		maxLen := len(input)
		if len(c) > maxLen {
			maxLen = len(c)
		}
		if d <= maxLen/2+1 {
			results = append(results, scored{name: c, dist: d})
		}
	}
	slices.SortFunc(results, func(a, b scored) int {
		return cmp.Compare(a.dist, b.dist)
	})
	var out []string
	for i := 0; i < len(results) && i < maxResults; i++ {
		out = append(out, fmt.Sprintf("%q", results[i].name))
	}
	return out
}

// levenshtein computes the edit distance between two strings.
func levenshtein(a, b string) int {
	la, lb := len(a), len(b)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}
	prev := make([]int, lb+1)
	curr := make([]int, lb+1)
	for j := 0; j <= lb; j++ {
		prev[j] = j
	}
	for i := 1; i <= la; i++ {
		curr[0] = i
		for j := 1; j <= lb; j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			curr[j] = min(curr[j-1]+1, min(prev[j]+1, prev[j-1]+cost))
		}
		prev, curr = curr, prev
	}
	return prev[lb]
}
