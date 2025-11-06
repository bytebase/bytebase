package output

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/action/github"
	"github.com/bytebase/bytebase/action/world"
)

// WriteOutput writes the output JSON and GitHub step summary if applicable.
func WriteOutput(w *world.World) {
	if err := writeOutputJSON(w); err != nil {
		w.Logger.Error("failed to write output JSON", "error", err)
	}
	if err := writeGitHubStepSummary(w); err != nil {
		w.Logger.Error("failed to write GitHub step summary", "error", err)
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
