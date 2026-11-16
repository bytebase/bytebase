package github

import (
	"github.com/bytebase/bytebase/action/result"
	"github.com/bytebase/bytebase/action/world"
)

// BuildSummaryMarkdown is the step summary of a rollout job. It carries one
// link: the rollout's, or the plan's when the job stopped before creating a
// rollout. The release has no page of its own.
func BuildSummaryMarkdown(w *world.World) string {
	switch {
	case w.Result != nil:
		return result.RenderMarkdown(w.Result)
	case w.OutputMap.Rollout != "":
		return "[View the rollout in Bytebase](" + w.URL + "/" + w.OutputMap.Rollout + ")\n"
	case w.OutputMap.Plan != "":
		return "[View the plan in Bytebase](" + w.URL + "/" + w.OutputMap.Plan + ")\n"
	default:
		return ""
	}
}
