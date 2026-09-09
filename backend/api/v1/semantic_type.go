package v1

const (
	defaultSemanticTypeID        = "bb.default"
	defaultPartialSemanticTypeID = "bb.default-partial"
)

func isBuiltinSemanticTypeID(id string) bool {
	return id == defaultSemanticTypeID || id == defaultPartialSemanticTypeID
}
