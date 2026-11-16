package output

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/action/gitlab"
	"github.com/bytebase/bytebase/action/result"
	"github.com/bytebase/bytebase/action/world"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

func TestWriteOutputJSON_EmptyOutputPath(t *testing.T) {
	w := world.NewWorld()
	w.Output = ""
	w.OutputMap.Release = "projects/test/releases/123"

	err := writeOutputJSON(w)

	require.NoError(t, err)
}

func TestWriteOutputJSON_StringFieldsOnly(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "output.json")

	w := world.NewWorld()
	w.Output = outputFile
	w.OutputMap.Release = "projects/test/releases/123"
	w.OutputMap.Plan = "projects/test/plans/456"
	w.OutputMap.Rollout = "projects/test/plans/789/rollout"

	err := writeOutputJSON(w)
	require.NoError(t, err)

	// Verify file was created
	data, err := os.ReadFile(outputFile)
	require.NoError(t, err)

	// Verify JSON content
	var result map[string]any
	err = json.Unmarshal(data, &result)
	require.NoError(t, err)

	require.Equal(t, "projects/test/releases/123", result["release"])
	require.Equal(t, "projects/test/plans/456", result["plan"])
	require.Equal(t, "projects/test/plans/789/rollout", result["rollout"])
	require.NotContains(t, result, "checkResults")
}

func TestWriteOutputJSON_CheckResultsWithProtojson(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "output.json")

	w := world.NewWorld()
	w.Output = outputFile
	w.OutputMap.CheckResults = &v1pb.CheckReleaseResponse{
		Results: []*v1pb.CheckReleaseResponse_CheckResult{
			{
				File:   "migration.sql",
				Target: "instances/prod/databases/mydb",
				Advices: []*v1pb.Advice{
					{
						Status:  v1pb.Advice_WARNING,
						Code:    1001,
						Title:   "Schema drift detected",
						Content: "Schema differs from expected",
					},
				},
				AffectedRows: 100,
				RiskLevel:    v1pb.RiskLevel_MODERATE,
			},
		},
		AffectedRows: 100,
		RiskLevel:    v1pb.RiskLevel_MODERATE,
	}

	err := writeOutputJSON(w)
	require.NoError(t, err)

	// Verify file was created
	data, err := os.ReadFile(outputFile)
	require.NoError(t, err)

	// Verify JSON content
	var result map[string]any
	err = json.Unmarshal(data, &result)
	require.NoError(t, err)

	// Verify CheckResults is present
	require.Contains(t, result, "checkResults")
	checkResults, ok := result["checkResults"].(map[string]any)
	require.True(t, ok, "checkResults should be a map")

	// Verify protojson uses camelCase (not snake_case)
	require.Contains(t, checkResults, "results")
	require.Contains(t, checkResults, "affectedRows")
	require.Contains(t, checkResults, "riskLevel")

	// Verify data integrity
	// Note: protojson serializes int64 as strings to avoid JavaScript precision loss
	require.Equal(t, "100", checkResults["affectedRows"])
	require.Equal(t, "MODERATE", checkResults["riskLevel"])

	results, ok := checkResults["results"].([]any)
	require.True(t, ok, "results should be an array")
	require.Len(t, results, 1)

	firstResult, ok := results[0].(map[string]any)
	require.True(t, ok, "first result should be a map")
	require.Equal(t, "migration.sql", firstResult["file"])
	require.Equal(t, "instances/prod/databases/mydb", firstResult["target"])
	require.Equal(t, "100", firstResult["affectedRows"])

	advices, ok := firstResult["advices"].([]any)
	require.True(t, ok, "advices should be an array")
	require.Len(t, advices, 1)
	advice, ok := advices[0].(map[string]any)
	require.True(t, ok, "advice should be a map")
	require.Equal(t, "WARNING", advice["status"])
	require.Equal(t, float64(1001), advice["code"])
	require.Equal(t, "Schema drift detected", advice["title"])
}

func TestWriteOutputJSON_CreatesParentDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "nested", "dir", "output.json")

	w := world.NewWorld()
	w.Output = outputFile
	w.OutputMap.Release = "projects/test/releases/123"

	err := writeOutputJSON(w)
	require.NoError(t, err)

	// Verify file was created
	_, err = os.Stat(outputFile)
	require.NoError(t, err)

	// Verify parent directories were created
	_, err = os.Stat(filepath.Join(tmpDir, "nested", "dir"))
	require.NoError(t, err)
}

func TestWriteOutputJSON_AllFieldsPopulated(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "output.json")

	w := world.NewWorld()
	w.Output = outputFile
	w.OutputMap.Release = "projects/test/releases/123"
	w.OutputMap.Plan = "projects/test/plans/456"
	w.OutputMap.Rollout = "projects/test/plans/789/rollout"
	w.OutputMap.CheckResults = &v1pb.CheckReleaseResponse{
		Results: []*v1pb.CheckReleaseResponse_CheckResult{
			{
				File:         "test.sql",
				Target:       "instances/test/databases/db",
				AffectedRows: 42,
			},
		},
		AffectedRows: 42,
	}

	err := writeOutputJSON(w)
	require.NoError(t, err)

	// Verify file was created
	data, err := os.ReadFile(outputFile)
	require.NoError(t, err)

	// Verify JSON content
	var result map[string]any
	err = json.Unmarshal(data, &result)
	require.NoError(t, err)

	// Verify all fields are present
	require.Equal(t, "projects/test/releases/123", result["release"])
	require.Equal(t, "projects/test/plans/456", result["plan"])
	require.Equal(t, "projects/test/plans/789/rollout", result["rollout"])
	require.Contains(t, result, "checkResults")

	// Verify CheckResults structure
	checkResults, ok := result["checkResults"].(map[string]any)
	require.True(t, ok, "checkResults should be a map")
	require.Equal(t, "42", checkResults["affectedRows"])
	results, ok := checkResults["results"].([]any)
	require.True(t, ok, "results should be an array")
	require.Len(t, results, 1)
}

func failedResult() *result.Result {
	rows := int64(40)
	return &result.Result{
		Outcome:     result.Failed,
		URL:         "https://bytebase.example.com/projects/example/plans/101/rollout",
		TargetStage: "environments/prod",
		Stages: []*result.Stage{{
			Environment: "environments/prod",
			TaskCounts:  map[string]int{"FAILED": 1},
			Tasks: []*result.Task{{
				Target: "instances/prod/databases/crm",
				Status: "FAILED",
				Files: []*result.File{{
					Version: "0004",
					Path:    "migrations/0004_add_risk_score.sql",
					Status:  "FAILED",
					Commands: []*result.Command{
						{Statement: "UPDATE account SET tier = 'basic'", AffectedRows: &rows},
						{Statement: "::set-output name=x::y", Error: "syntax error"},
					},
				}},
			}},
		}},
	}
}

func TestWriteOutputJSON_Result(t *testing.T) {
	outputFile := filepath.Join(t.TempDir(), "output.json")
	w := world.NewWorld()
	w.Output = outputFile
	w.OutputMap.Plan = "projects/example/plans/101"
	w.Result = failedResult()

	require.NoError(t, writeOutputJSON(w))

	data, err := os.ReadFile(outputFile)
	require.NoError(t, err)
	var output struct {
		Plan   string `json:"plan"`
		Result struct {
			Outcome string `json:"outcome"`
			Stages  []struct {
				TaskCounts map[string]int `json:"taskCounts"`
				Tasks      []struct {
					Target string `json:"target"`
					Files  []struct {
						Path     string `json:"path"`
						Commands []struct {
							AffectedRows *int64 `json:"affectedRows"`
							Error        string `json:"error"`
						} `json:"commands"`
					} `json:"files"`
				} `json:"tasks"`
			} `json:"stages"`
		} `json:"result"`
	}
	require.NoError(t, json.Unmarshal(data, &output))
	require.Equal(t, "projects/example/plans/101", output.Plan)
	require.Equal(t, "FAILED", output.Result.Outcome)
	require.Equal(t, map[string]int{"FAILED": 1}, output.Result.Stages[0].TaskCounts)
	commands := output.Result.Stages[0].Tasks[0].Files[0].Commands
	require.Equal(t, int64(40), *commands[0].AffectedRows)
	require.Nil(t, commands[1].AffectedRows, "a failed command has no count, which is not zero")
	require.Equal(t, "syntax error", commands[1].Error)
}

func TestWriteResultBlock(t *testing.T) {
	t.Run("no result prints nothing", func(t *testing.T) {
		var out bytes.Buffer
		writeResultBlock(world.NewWorld(), &out)
		require.Empty(t, out.String())
	})

	t.Run("local", func(t *testing.T) {
		w := world.NewWorld()
		w.Platform = world.LocalPlatform
		w.Result = failedResult()
		var out bytes.Buffer
		writeResultBlock(w, &out)
		require.True(t, strings.HasPrefix(out.String(), "\nRollout to prod: FAILED\n"))
		require.True(t, strings.HasSuffix(out.String(), "\nRollout: https://bytebase.example.com/projects/example/plans/101/rollout\n"))
		require.NotContains(t, out.String(), "::error::")
	})

	t.Run("github stops workflow commands around the block", func(t *testing.T) {
		w := world.NewWorld()
		w.Platform = world.GitHub
		w.Result = failedResult()
		var out bytes.Buffer
		writeResultBlock(w, &out)

		lines := strings.Split(strings.TrimSpace(out.String()), "\n")
		token, ok := strings.CutPrefix(lines[0], "::stop-commands::")
		require.True(t, ok)
		resume := -1
		statement := -1
		for i, line := range lines {
			if line == "::"+token+"::" {
				resume = i
			}
			if strings.Contains(line, "::set-output name=x::y") && !strings.HasPrefix(line, "::error::") {
				statement = i
			}
		}
		require.Positive(t, statement)
		require.Greater(t, resume, statement, "the statement is printed while commands are stopped")
		require.Equal(t, "::error::instances/prod/databases/crm failed at migrations/0004_add_risk_score.sql: syntax error", lines[len(lines)-1])
	})
}

func TestWriteGitLabAnnotations(t *testing.T) {
	t.Chdir(t.TempDir())
	w := world.NewWorld()
	w.Platform = world.GitLab
	w.Result = failedResult()

	require.NoError(t, writeGitLabAnnotations(w))

	data, err := os.ReadFile(gitlab.AnnotationsFile)
	require.NoError(t, err)
	require.JSONEq(t, `{"Bytebase": [{"external_link": {"label": "Rollout", "url": "https://bytebase.example.com/projects/example/plans/101/rollout"}}]}`, string(data))
}

func TestWriteGitHubStepSummary(t *testing.T) {
	summary := filepath.Join(t.TempDir(), "summary.md")
	t.Setenv("GITHUB_STEP_SUMMARY", summary)
	w := world.NewWorld()
	w.Platform = world.GitHub
	w.IsRollout = true
	w.URL = "https://bytebase.example.com"
	w.OutputMap.Release = "projects/example/releases/r1"
	w.OutputMap.Plan = "projects/example/plans/101"
	w.OutputMap.Rollout = "projects/example/plans/101/rollout"
	w.Result = failedResult()

	require.NoError(t, writeGitHubStepSummary(w))

	data, err := os.ReadFile(summary)
	require.NoError(t, err)
	require.Contains(t, string(data), "### Rollout to prod: FAILED")
	require.Contains(t, string(data), "#### crm on prod failed")
	require.NotContains(t, string(data), "View in Bytebase")
	require.NotContains(t, string(data), "https://bytebase.example.com/projects/example/plans/101)", "the plan link is not in the summary")
	require.NotContains(t, string(data), "/releases/")
}

func TestWriteGitHubStepSummary_BeforeTheResult(t *testing.T) {
	tests := []struct {
		name    string
		plan    string
		rollout string
		want    string
	}{
		{
			name:    "a rollout with no result links the rollout",
			plan:    "projects/example/plans/101",
			rollout: "projects/example/plans/101/rollout",
			want:    "[View the rollout in Bytebase](https://bytebase.example.com/projects/example/plans/101/rollout)\n",
		},
		{
			name: "a job that stopped before the rollout links the plan",
			plan: "projects/example/plans/101",
			want: "[View the plan in Bytebase](https://bytebase.example.com/projects/example/plans/101)\n",
		},
		{
			name: "a release alone has no page to link",
			want: "",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			summary := filepath.Join(t.TempDir(), "summary.md")
			t.Setenv("GITHUB_STEP_SUMMARY", summary)
			w := world.NewWorld()
			w.Platform = world.GitHub
			w.IsRollout = true
			w.URL = "https://bytebase.example.com"
			w.OutputMap.Release = "projects/example/releases/r1"
			w.OutputMap.Plan = tc.plan
			w.OutputMap.Rollout = tc.rollout

			require.NoError(t, writeGitHubStepSummary(w))

			data, err := os.ReadFile(summary)
			require.NoError(t, err)
			require.Equal(t, tc.want, string(data))
		})
	}
}
