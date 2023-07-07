import { computed } from "vue";
// import { maxBy } from "lodash-es";

// import type { Task, TaskCheckRun, TaskCheckType } from "@/types";
// import type { AdviceOption } from "@/components/MonacoEditor";
import { useIssueContext } from "../../../logic";

export const useSQLAdviceMarkers = () => {
  const { isCreating } = useIssueContext();
  const markers = computed(() => {
    if (isCreating.value) return [];

    return []; // TODO
    // const task = selectedTask.value;
    // const types: TaskCheckType[] = [
    //   "bb.task-check.database.statement.advise",
    //   "bb.task-check.database.statement.syntax",
    // ];
    // return types.flatMap((type) =>
    //   getLatestAdviceOptions(
    //     task.taskCheckRunList.filter((check) => check.type === type)
    //   )
    // );
  });
  return { markers };
};

// const getLatestAdviceOptions = (taskCheckRunList: TaskCheckRun[]) => {
//   const latest = maxBy(taskCheckRunList, (advice) => advice.id);
//   if (!latest) {
//     return [];
//   }
//   const { resultList = [] } = latest.result;
//   return resultList
//     .filter((result) => result.status === "ERROR" || result.status === "WARN")
//     .filter((result) => result.line !== undefined)
//     .map<AdviceOption>((result) => {
//       return {
//         severity: result.status === "ERROR" ? "ERROR" : "WARNING",
//         message: result.content,
//         source: `${result.title} (${result.code})`,
//         startLineNumber: result.line!,
//         // We don't know the actual column yet, so we show the marker at then end of the line
//         startColumn: Number.MAX_SAFE_INTEGER,
//         endLineNumber: result.line!,
//         // We don't know the actual column yet, so we show the marker at then end of the line
//         endColumn: Number.MAX_SAFE_INTEGER,
//       };
//     });
// };
