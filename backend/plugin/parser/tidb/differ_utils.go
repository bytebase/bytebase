package tidb

import (
	"bytes"

	"github.com/pingcap/tidb/pkg/parser/ast"
	"github.com/pingcap/tidb/pkg/parser/format"

	"github.com/pkg/errors"
)

func toString(node ast.Node) (string, error) {
	var buf bytes.Buffer
	restoreFlag := format.DefaultRestoreFlags | format.RestoreStringWithoutCharset | format.RestorePrettyFormat
	if err := node.Restore(format.NewRestoreCtx(restoreFlag, &buf)); err != nil {
		return "", errors.Wrapf(err, "cannot restore node %v", node)
	}
	return buf.String(), nil
}

func toLowerNameString(node ast.Node) (string, error) {
	var buf bytes.Buffer
	restoreFlag := format.DefaultRestoreFlags | format.RestoreStringWithoutCharset | format.RestorePrettyFormat | format.RestoreNameLowercase
	if err := node.Restore(format.NewRestoreCtx(restoreFlag, &buf)); err != nil {
		return "", errors.Wrapf(err, "cannot restore node %v", node)
	}
	return buf.String(), nil
}
