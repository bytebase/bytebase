package base

type BackupStatement struct {
	Statement string
	TableName string

	OriginalLine int
}
