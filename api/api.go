package api

type RowStatus string

const (
	Normal        RowStatus = "NORMAL"
	Archived      RowStatus = "ARCHIVED"
	PendingDelete RowStatus = "PENDING_DELETE"
)

func (e RowStatus) String() string {
	switch e {
	case Normal:
		return "NORMAL"
	case Archived:
		return "ARCHIVED"
	case PendingDelete:
		return "PENDING_DELETE"
	}
	return ""
}
