import { Issue, StageID } from "../types";

export function stageName(issue: Issue, stageID: StageID): string {
  for (const stage of issue.pipeline.stageList) {
    if (stage.id == stageID) {
      return stage.name;
    }
  }
  return "<<Unknown stage>>";
}
