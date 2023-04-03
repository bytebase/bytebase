import type {
  Activity,
  ActivityTaskStatementUpdatePayload,
  ActivityTaskStatusUpdatePayload,
} from "@/types";

export type DistinctActivity = {
  activity: Activity;
  similar: Activity[];
};

export const isSimilarActivity = (a: Activity, b: Activity): boolean => {
  // Now, we recognize two "Change SQL from .... to ...." activities are similar
  // when they have the same "from" and "to" values.
  if (
    a.type === "bb.pipeline.task.statement.update" &&
    b.type === "bb.pipeline.task.statement.update" &&
    a.containerId === b.containerId
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

  if (
    a.type === "bb.pipeline.task.status.update" &&
    b.type === "bb.pipeline.task.status.update" &&
    a.containerId === b.containerId &&
    a.creator.id === b.creator.id
  ) {
    const payloadA = a.payload as ActivityTaskStatusUpdatePayload;
    const payloadB = b.payload as ActivityTaskStatusUpdatePayload;
    if (
      payloadA.oldStatus === payloadB.oldStatus &&
      payloadB.newStatus === payloadB.newStatus
    ) {
      return true;
    }
  }

  return false;
};
