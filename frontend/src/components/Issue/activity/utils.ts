import type { Activity, ActivityTaskStatementUpdatePayload } from "@/types";

export const isSimilarActivity = (a: Activity, b: Activity): boolean => {
  // Now, we recognize two "Change SQL from .... to ...." activities are similar
  // when they have the same "from" and "to" values.
  if (
    a.type === "bb.pipeline.task.statement.update" &&
    b.type === "bb.pipeline.task.statement.update"
  ) {
    const payloadA = a.payload as ActivityTaskStatementUpdatePayload;
    const payloadB = b.payload as ActivityTaskStatementUpdatePayload;
    if (
      payloadA.oldStatement === payloadB.oldStatement &&
      payloadB.newStatement === payloadB.newStatement
    ) {
      return true;
    }
  }

  return false;
};
