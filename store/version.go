package store

import "fmt"

type version struct {
	major, minor int
}

func newVersion(major, minor int) version {
	return version{
		major: major,
		minor: minor,
	}
}

func versionFromInt(v int) version {
	return newVersion(v/10000, v%10000)
}

func (v version) biggerThan(other version) bool {
	if v.major > other.major {
		return true
	}
	return v.major == other.major && v.minor > other.minor
}

func (v version) String() string {
	return fmt.Sprintf("%d.%d", v.major, v.minor)
}
