package api

const UNKNOWN_ID = -1

type RowStatus string

const (
	Normal   RowStatus = "NORMAL"
	Archived RowStatus = "ARCHIVED"
)

func (e RowStatus) String() string {
	switch e {
	case Normal:
		return "NORMAL"
	case Archived:
		return "ARCHIVED"
	}
	return ""
}
