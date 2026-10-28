package migrator

// MigrationFS is the embedded migrations, for the migrator_test tests that
// apply one to a database.
var MigrationFS = migrationFS

// SummarizeMigration3_25_0 exposes the 3.25.0 Go migration's summary to migrator_test.
var SummarizeMigration3_25_0 = summarizeMigration3_25_0

// SQLExportMigrationSummary is the summary type SummarizeMigration3_25_0 returns.
type SQLExportMigrationSummary = sqlExportMigrationSummary
