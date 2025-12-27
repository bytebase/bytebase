package github

import (
	"strings"

	"github.com/bytebase/bytebase/action/world"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

func BuildSummaryMarkdown(w *world.World) string {
	var sb strings.Builder
	if w.OutputMap.Release != "" {
		_, _ = sb.WriteString("Release: " + w.URL + "/" + w.OutputMap.Release + "\n")
	}
	if w.OutputMap.Plan != "" {
		_, _ = sb.WriteString("Plan: " + w.URL + "/" + w.OutputMap.Plan + "\n")
	}
	if w.OutputMap.Rollout != "" {
		_, _ = sb.WriteString("Rollout: " + w.URL + "/" + w.OutputMap.Rollout + "\n")
	}

	// Add stage status table
	buildOutputStagesMarkdown(w, &sb)

	return sb.String()
}

func buildOutputStagesMarkdown(w *world.World, sb *strings.Builder) {
	if w.IsRollout && w.Rollout != nil {
		rollout := w.Rollout

		if len(rollout.Stages) == 0 {
			sb.WriteString("\n### Rollout Stages\n\n")
			sb.WriteString("_No stages to display. The rollout may not have any stages._\n\n")
			return
		}

		allStages := []string{}
		for _, stage := range rollout.Stages {
			allStages = append(allStages, stage.Environment)
		}

		// Create a map of stage environment to Stage object for quick lookup
		stageMap := make(map[string]*v1pb.Stage)
		for _, stage := range rollout.Stages {
			stageMap[stage.Environment] = stage
		}

		// Build the markdown table
		sb.WriteString("\n### Rollout Stages\n\n")
		sb.WriteString("| Stage | Status |\n")
		sb.WriteString("|-------|--------|\n")

		// Output all stages with their status
		for _, stageEnv := range allStages {
			stageName := extractStageName(stageEnv)

			// Determine status
			var status string
			if stage, exists := stageMap[stageEnv]; exists {
				// Stage exists in rollout, get its aggregated status
				status = getAggregatedStageStatus(stage)
			} else {
				// Stage is only in pendingStages, mark as pending
				status = "⏳ Pending"
			}

			sb.WriteString("| ")
			sb.WriteString(stageName)
			sb.WriteString(" | ")
			sb.WriteString(status)
			sb.WriteString(" |\n")
		}

		sb.WriteString("\n")
	}
}

// getAggregatedStageStatus determines the overall status of a stage based on its tasks
func getAggregatedStageStatus(stage *v1pb.Stage) string {
	if stage == nil || len(stage.Tasks) == 0 {
		return "⏳ Pending"
	}

	hasRunning := false
	hasFailed := false
	hasCanceled := false
	hasSkipped := false
	hasPending := false
	hasNotStarted := false
	allDone := true

	for _, task := range stage.Tasks {
		switch task.Status {
		case v1pb.Task_RUNNING:
			hasRunning = true
			allDone = false
		case v1pb.Task_FAILED:
			hasFailed = true
			allDone = false
		case v1pb.Task_CANCELED:
			hasCanceled = true
			allDone = false
		case v1pb.Task_SKIPPED:
			hasSkipped = true
		case v1pb.Task_PENDING:
			hasPending = true
			allDone = false
		case v1pb.Task_NOT_STARTED, v1pb.Task_STATUS_UNSPECIFIED:
			hasNotStarted = true
			allDone = false
		case v1pb.Task_DONE:
			// Task is done
		default:
			allDone = false
		}
	}

	// Determine overall stage status based on task statuses
	if hasFailed {
		return "❌ Failed"
	}
	if hasCanceled {
		return "⛔ Canceled"
	}
	if hasRunning {
		return "🔄 Running"
	}
	if hasPending {
		return "⏳ Pending"
	}
	if hasNotStarted {
		return "⏸️ Not Started"
	}
	if allDone && !hasSkipped {
		return "✅ Done"
	}
	if allDone && hasSkipped {
		return "⏭️ Skipped"
	}

	return "⏳ Pending"
}

// extractStageName extracts the stage name from the environment path
func extractStageName(environment string) string {
	// Format: environments/{environment}
	parts := strings.Split(environment, "/")
	if len(parts) >= 2 {
		return parts[len(parts)-1]
	}
	return environment
}
