package gitlab

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/action/result"
)

// AnnotationsFile is the report a job declares under
// artifacts:reports:annotations.
const AnnotationsFile = "bytebase_annotations.json"

// WriteRolloutAnnotations writes an annotations report that puts a link to
// the rollout on the job page.
// https://docs.gitlab.com/ci/yaml/artifacts_reports/#artifactsreportsannotations
func WriteRolloutAnnotations(rolloutURL string) error {
	type externalLink struct {
		Label string `json:"label"`
		URL   string `json:"url"`
	}
	type annotation struct {
		ExternalLink externalLink `json:"external_link"`
	}
	data, err := json.MarshalIndent(map[string][]annotation{
		"Bytebase": {{ExternalLink: externalLink{Label: "Rollout", URL: rolloutURL}}},
	}, "", "  ")
	if err != nil {
		return errors.Wrapf(err, "failed to marshal annotations")
	}
	return os.WriteFile(AnnotationsFile, data, 0644)
}

// FoldSection wraps a task's lines in a collapsed job log section.
// https://docs.gitlab.com/ci/jobs/job_logs/#custom-collapsible-sections
func FoldSection(now time.Time) result.Folder {
	return func(index int, header string, detail []string) []string {
		name := fmt.Sprintf("bytebase_task_%d", index)
		lines := []string{fmt.Sprintf("\x1b[0Ksection_start:%d:%s[collapsed=true]\r\x1b[0K%s", now.Unix(), name, header)}
		lines = append(lines, detail...)
		return append(lines, fmt.Sprintf("\x1b[0Ksection_end:%d:%s\r\x1b[0K", now.Unix(), name))
	}
}
