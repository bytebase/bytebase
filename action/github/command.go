package github

import "strings"

// ErrorCommand formats message as a workflow command that annotates the job
// with an error.
// https://docs.github.com/en/actions/reference/workflows-and-actions/workflow-commands#setting-an-error-message
func ErrorCommand(message string) string {
	return "::error::" + escapeCommandData(message)
}

// WarningCommand formats message as a workflow command that annotates the job
// with a warning. A warning does not change the job's conclusion.
// https://docs.github.com/en/actions/reference/workflows-and-actions/workflow-commands#setting-a-warning-message
func WarningCommand(message string) string {
	return "::warning::" + escapeCommandData(message)
}

func escapeCommandData(message string) string {
	// A raw newline would end the command and let the rest of the message be
	// read as another one.
	return strings.NewReplacer("%", "%25", "\r", "%0D", "\n", "%0A").Replace(message)
}
