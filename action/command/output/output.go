package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/action/github"
	"github.com/bytebase/bytebase/action/gitlab"
	"github.com/bytebase/bytebase/action/result"
	"github.com/bytebase/bytebase/action/world"
)

// maxErrorAnnotations is GitHub's limit on error annotations for one step.
const maxErrorAnnotations = 10

// WriteOutput writes what a command reports: the output JSON, and for a
// rollout its result in the job log and on the CI's own surfaces. A surface
// that cannot be written is logged and does not fail the command.
func WriteOutput(w *world.World, out io.Writer) {
	if err := writeOutputJSON(w); err != nil {
		w.Logger.Error("failed to write output JSON", "error", err)
	}
	writeResultBlock(w, out)
	if err := writeGitHubStepSummary(w); err != nil {
		w.Logger.Error("failed to write GitHub step summary", "error", err)
	}
	if err := writeGitLabAnnotations(w); err != nil {
		w.Logger.Error("failed to write GitLab annotations", "error", err)
	}
}

// writeResultBlock prints the rollout result to the job log.
func writeResultBlock(w *world.World, out io.Writer) {
	if w.Result == nil {
		return
	}
	switch w.Platform {
	case world.GitHub:
		// Statements and database errors are printed as they are, so the
		// runner must not read a line of them as a workflow command.
		// https://docs.github.com/en/actions/reference/workflows-and-actions/workflow-commands#stopping-and-starting-workflow-commands
		token := uuid.NewString()
		_, _ = fmt.Fprintf(out, "\n::stop-commands::%s\n%s::%s::\n", token, result.RenderText(w.Result, nil), token)
		for i, failure := range w.Result.Failures() {
			if i == maxErrorAnnotations {
				break
			}
			_, _ = fmt.Fprintln(out, github.ErrorCommand(failure))
		}
	case world.GitLab:
		_, _ = fmt.Fprint(out, "\n"+result.RenderText(w.Result, gitlab.FoldSection(time.Now())))
	default:
		_, _ = fmt.Fprint(out, "\n"+result.RenderText(w.Result, nil))
	}
}

// writeOutputJSON writes the output map to the specified JSON file.
func writeOutputJSON(w *world.World) error {
	if w.Output == "" {
		return nil
	}

	w.Logger.Info("writing output to file", "file", w.Output)

	// Create parent directory if not exists
	if dir := filepath.Dir(w.Output); dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return errors.Wrapf(err, "failed to create output directory: %s", dir)
		}
	}

	f, err := os.Create(w.Output)
	if err != nil {
		return errors.Wrapf(err, "failed to create output file: %s", w.Output)
	}
	defer f.Close()

	// Marshal OutputMap with protojson support for CheckResults
	outputData := make(map[string]any)
	if w.OutputMap.Release != "" {
		outputData["release"] = w.OutputMap.Release
	}
	if w.OutputMap.Plan != "" {
		outputData["plan"] = w.OutputMap.Plan
	}
	if w.OutputMap.Rollout != "" {
		outputData["rollout"] = w.OutputMap.Rollout
	}
	if w.Result != nil {
		outputData["result"] = w.Result
	}
	if w.OutputMap.CheckResults != nil {
		// Use protojson to marshal CheckResults with camelCase keys
		checkResultsJSON, err := protojson.Marshal(w.OutputMap.CheckResults)
		if err != nil {
			return errors.Wrapf(err, "failed to marshal check results")
		}
		var checkResultsMap map[string]any
		if err := json.Unmarshal(checkResultsJSON, &checkResultsMap); err != nil {
			return errors.Wrapf(err, "failed to unmarshal check results")
		}
		outputData["checkResults"] = checkResultsMap
	}

	j, err := json.MarshalIndent(outputData, "", "  ")
	if err != nil {
		return errors.Wrapf(err, "failed to marshal output map")
	}

	if _, err := f.Write(j); err != nil {
		return errors.Wrapf(err, "failed to write output file: %s", w.Output)
	}
	return nil
}

// writeGitHubStepSummary writes the GitHub step summary for rollout operations.
func writeGitHubStepSummary(w *world.World) error {
	if w.Platform != world.GitHub || !w.IsRollout {
		return nil
	}

	filename := os.Getenv("GITHUB_STEP_SUMMARY")
	if filename == "" {
		return nil
	}
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return errors.Wrapf(err, "failed to open GitHub step summary file: %s", filename)
	}
	defer f.Close()

	summary := github.BuildSummaryMarkdown(w)
	if _, err := f.WriteString(summary); err != nil {
		return errors.Wrapf(err, "failed to write GitHub step summary: %s", filename)
	}
	return nil
}

// writeGitLabAnnotations links the GitLab job page to the rollout.
func writeGitLabAnnotations(w *world.World) error {
	if w.Platform != world.GitLab || w.Result == nil {
		return nil
	}
	return gitlab.WriteRolloutAnnotations(w.Result.URL)
}
