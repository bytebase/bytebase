package output

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

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
