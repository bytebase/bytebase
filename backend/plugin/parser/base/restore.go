package base

type RestoreContext struct {
	InstanceID              string
	GetDatabaseMetadataFunc GetDatabaseMetadataFunc
	ListDatabaseNamesFunc   ListDatabaseNamesFunc
	IgnoreCaseSensitive     bool
}
