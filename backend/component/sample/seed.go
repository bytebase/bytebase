package sample

import (
	"embed"
	"io/fs"
	"slices"
	"strings"

	"github.com/pkg/errors"
)

// Sample data is from https://github.com/bytebase/employee-sample-database/tree/main/postgres/dataset_small.
//
//go:embed seed
var seedFS embed.FS

// LoadSeedData returns the ordered employee sample SQL.
func LoadSeedData() (string, error) {
	names, err := fs.Glob(seedFS, "seed/*.sql")
	if err != nil {
		return "", err
	}
	slices.Sort(names)

	var builder strings.Builder
	for _, name := range names {
		content, err := fs.ReadFile(seedFS, name)
		if err != nil {
			return "", errors.Wrapf(err, "failed to read sample database data: %s", name)
		}
		if _, err := builder.Write(content); err != nil {
			return "", err
		}
	}
	return builder.String(), nil
}
