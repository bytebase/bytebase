package mysql

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"github.com/pingcap/tidb/parser/ast"
	"github.com/pingcap/tidb/parser/format"

	"github.com/pkg/errors"
)

type objectType string

const (
	event     objectType = "EVENT"
	function  objectType = "FUNCTION"
	procedure objectType = "PROCEDURE"
	trigger   objectType = "TRIGGER"
)

// extractUnsupportObjNameAndType extract the object name from the CREATE TRIGGER/EVENT/FUNCTION/PROCEDURE statement and returns the object name and type.
func extractUnsupportObjNameAndType(stmt string) (string, objectType, error) {
	fs := []objectType{
		function,
		procedure,
	}
	regexFmt := "(?mUi)^CREATE\\s+(DEFINER=(`(.)+`|(.)+)@(`(.)+`|(.)+)(\\s)+)?%s\\s+(?P<OBJECT_NAME>%s)(\\s)*\\("
	// We should support the naming likes "`abc`" or "abc".
	namingRegex := fmt.Sprintf("(`%s`)|(%s)", "[^\\\\/?%*:|\\\"`<>]+", "[^\\\\/?%*:|\\\"`<>]+")
	for _, obj := range fs {
		regex := fmt.Sprintf(regexFmt, string(obj), namingRegex)
		re := regexp.MustCompile(regex)
		matchList := re.FindStringSubmatch(stmt)
		index := re.SubexpIndex("OBJECT_NAME")
		if index >= 0 && index < len(matchList) {
			objectName := strings.Trim(matchList[index], "`")
			return objectName, obj, nil
		}
	}
	objects := []objectType{
		trigger,
		event,
	}
	regexFmt = "(?mUi)^CREATE\\s+(DEFINER=(`(.)+`|(.)+)@(`(.)+`|(.)+)(\\s)+)?%s\\s+(?P<OBJECT_NAME>%s)(\\s)+"
	for _, obj := range objects {
		regex := fmt.Sprintf(regexFmt, string(obj), namingRegex)
		re := regexp.MustCompile(regex)
		matchList := re.FindStringSubmatch(stmt)
		index := re.SubexpIndex("OBJECT_NAME")
		if index >= 0 && index < len(matchList) {
			objectName := strings.Trim(matchList[index], "`")
			return objectName, obj, nil
		}
	}
	return "", "", errors.Errorf("cannot extract object name and type from %q", stmt)
}

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
