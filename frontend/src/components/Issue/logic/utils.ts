import { groupBy, maxBy } from "lodash-es";
import { Task } from "@/types";

export type TaskCheckRunSummary = {
  runningCount: number;
  successCount: number;
  warnCount: number;
  errorCount: number;
};

export function taskCheckRunSummary(task: Task): TaskCheckRunSummary {
  const summary: TaskCheckRunSummary = {
    runningCount: 0,
    successCount: 0,
    warnCount: 0,
    errorCount: 0,
  };

  const taskCheckRunList = task.taskCheckRunList ?? [];

  const listGroupByType = groupBy(
    taskCheckRunList,
    (checkRun) => checkRun.type
  );

  const latestCheckRunOfEachType = Object.keys(listGroupByType).map((type) => {
    const listOfType = listGroupByType[type];
    const latest = maxBy(listOfType, (checkRun) => checkRun.updatedTs)!;
    return latest;
  });

  for (const checkRun of latestCheckRunOfEachType) {
    switch (checkRun.status) {
      case "CANCELED":
        // nothing todo
        break;
      case "FAILED":
        summary.errorCount++;
        break;
      case "RUNNING":
        summary.runningCount++;
        break;
      case "DONE":
        for (const result of checkRun.result.resultList) {
          switch (result.status) {
            case "SUCCESS":
              summary.successCount++;
              break;
            case "WARN":
              summary.warnCount++;
              break;
            case "ERROR":
              summary.errorCount++;
          }
        }
    }
  }

  return summary;
}
