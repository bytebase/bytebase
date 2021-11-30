import { Issue, StageId } from "../types";

export function stageName(issue: Issue, stageId: StageId): string {
  for (const stage of issue.pipeline.stageList) {
    if (stage.id == stageId) {
      return stage.name;
    }
  }
  return "<<Unknown stage>>";
}
