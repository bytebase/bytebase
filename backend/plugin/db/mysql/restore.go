package mysql

// This file implements recovery functions for MySQL.
// For example, the original database is `dbfoo`. The suffixTs, derived from the PITR issue's CreateTs, is 1653018005.
// Bytebase will do the following:
// 1. Create a database called `dbfoo_pitr_1653018005`, and do PITR restore to it.
// 2. Create a database called `dbfoo_pitr_1653018005_del`, and move tables
// 	  from `dbfoo` to `dbfoo_pitr_1653018005_del`, and tables from `dbfoo_pitr_1653018005` to `dbfoo`.

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

// ErrParseBinlogName is returned if we failed to parse binlog name.
type ErrParseBinlogName struct {
	err error
}

// IsErrParseBinlogName checks if the underlying error is ErrParseBinlogName.
func IsErrParseBinlogName(err error) bool {
	_, ok := errors.Cause(err).(ErrParseBinlogName)
	return ok
}

func (err ErrParseBinlogName) Error() string {
	return fmt.Sprintf("failed to parse binlog file name: %v", err.err)
}

// BinlogFile is the metadata of the MySQL binlog file.
type BinlogFile struct {
	Name string
	Size int64

	// Seq is parsed from Name and is for the sorting purpose.
	Seq int64
}

// ParseBinlogName parses the numeric extension and the binary log base name by using split the dot.
// Examples:
//   - ("binlog.000001") => ("binlog", 1)
//   - ("binlog000001") => ("", err)
func ParseBinlogName(name string) (string, int64, error) {
	s := strings.Split(name, ".")
	if len(s) != 2 {
		return "", 0, ErrParseBinlogName{err: errors.Errorf("failed to parse binlog extension, expecting two parts in the binlog file name %q but got %d", name, len(s))}
	}
	seq, err := strconv.ParseInt(s[1], 10, 0)
	if err != nil {
		return "", 0, ErrParseBinlogName{err: errors.Wrapf(err, "failed to parse the sequence number %s", s[1])}
	}
	return s[0], seq, nil
}

// GenBinlogFileNames generates the binlog file names between the start end end sequence numbers.
// The generation algorithm refers to the implementation of mysql-server: https://sourcegraph.com/github.com/mysql/mysql-server@a246bad76b9271cb4333634e954040a970222e0a/-/blob/sql/binlog.cc?L3693.
func GenBinlogFileNames(base string, seqStart, seqEnd int64) []string {
	var ret []string
	for i := seqStart; i <= seqEnd; i++ {
		ret = append(ret, fmt.Sprintf("%s.%06d", base, i))
	}
	return ret
}

// CheckBinlogEnabled checks whether binlog is enabled for the current instance.
func (driver *Driver) CheckBinlogEnabled(ctx context.Context) error {
	value, err := driver.getServerVariable(ctx, "log_bin")
	if err != nil {
		return err
	}
	if strings.ToUpper(value) != "ON" {
		return errors.Errorf("binlog is not enabled")
	}
	return nil
}

// CheckBinlogRowFormat checks whether the binlog format is ROW.
func (driver *Driver) CheckBinlogRowFormat(ctx context.Context) error {
	value, err := driver.getServerVariable(ctx, "binlog_format")
	if err != nil {
		return err
	}
	if strings.ToUpper(value) != "ROW" {
		return errors.Errorf("binlog format is not ROW but %s", value)
	}
	return nil
}
