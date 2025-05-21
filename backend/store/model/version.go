package model

import (
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

type Version struct {
	parts []uint64
}

func NewVersion(v string) (*Version, error) {
	parts := strings.Split(v, ".")
	r := &Version{}
	for _, p := range parts {
		n, err := strconv.ParseUint(p, 10, 64)
		if err != nil {
			return nil, errors.Errorf("invalid version %q", v)
		}
		r.parts = append(r.parts, n)
	}
	return r, nil
}

func (v *Version) LessThan(other *Version) bool {
	l := min(len(other.parts), len(v.parts))
	for i := range l {
		if v.parts[i] > other.parts[i] {
			return false
		}
		if v.parts[i] < other.parts[i] {
			return true
		}
	}
	return len(v.parts) < len(other.parts)
}
