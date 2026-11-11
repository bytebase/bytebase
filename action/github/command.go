package github

import "strings"

// ErrorCommand formats message as a workflow command that annotates the job
// with an error.
// https://docs.github.com/en/actions/reference/workflows-and-actions/workflow-commands#setting-an-error-message
func ErrorCommand(message string) string {
	// A raw newline would end the command and let the rest of the message be
	// read as another one.
	escaped := strings.NewReplacer("%", "%25", "\r", "%0D", "\n", "%0A").Replace(message)
	return "::error::" + escaped
}
