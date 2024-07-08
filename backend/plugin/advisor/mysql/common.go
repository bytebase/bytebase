package mysql

import "github.com/pkg/errors"

func getColumnIndex(columns []string, name string) (int, error) {
	for i, columnName := range columns {
		if columnName == name {
			return i, nil
		}
	}
	return -1, errors.Errorf("failed to find column %q", name)
}
