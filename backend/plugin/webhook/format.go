package webhook

import (
	"fmt"

	"github.com/bytebase/bytebase/backend/common"
)

// TruncateStringWithDescription tries to truncate the string and append "... (view details in Bytebase)" if truncated.
func TruncateStringWithDescription(str string) string {
	const limit = 450
	if truncatedStr, truncated := common.TruncateString(str, limit); truncated {
		return fmt.Sprintf("%s... (view details in Bytebase)", truncatedStr)
	}
	return str
}
