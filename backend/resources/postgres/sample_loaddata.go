package postgres

// LoadSampleData returns the concatenated sample dataset SQL for seeding a
// sample database. It is exported for reuse by the load-test harness.
func LoadSampleData() (string, error) {
	return loadSampleDataFromFS()
}
