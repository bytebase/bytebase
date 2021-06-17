package api

// Instance migration status
type InstanceMigrationStatus string

const (
	InstanceMigrationUnknown  InstanceMigrationStatus = "UNKNOWN"
	InstanceMigrationOK       InstanceMigrationStatus = "OK"
	InstanceMigrationNotExist InstanceMigrationStatus = "NOT_EXIST"
)

func (e InstanceMigrationStatus) String() string {
	switch e {
	case InstanceMigrationUnknown:
		return "UNKNOWN"
	case InstanceMigrationOK:
		return "OK"
	case InstanceMigrationNotExist:
		return "NOT_EXIST"
	}
	return "UNKNOWN"
}

type InstanceMigration struct {
	Status InstanceMigrationStatus `jsonapi:"attr,status"`
	Error  string                  `jsonapi:"attr,error"`
}
