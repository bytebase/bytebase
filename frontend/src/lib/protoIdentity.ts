import type { DescMessage, MessageShape } from "@bufbuild/protobuf";
import { equals, isMessage } from "@bufbuild/protobuf";
import type {
  Rollout,
  Stage,
  Task,
  TaskRun,
} from "@/types/proto-es/v1/rollout_service_pb";
import {
  RolloutSchema,
  StageSchema,
  TaskRunSchema,
  TaskSchema,
} from "@/types/proto-es/v1/rollout_service_pb";

// Identity-preserving comparison for polled proto data. Every poll tick
// rebuilds objects from the wire; keeping the previous reference when content
// is unchanged is what lets memoized components, effect deps, and prop
// comparisons skip work. Object identity is the contract downstream code keys
// off.

// Reference equality first; structural equality only for actual proto
// messages (test fixtures often cast plain objects, which `equals` rejects).
export const sameMessage = <Desc extends DescMessage>(
  schema: Desc,
  prev: MessageShape<Desc> | undefined,
  next: MessageShape<Desc> | undefined
): boolean =>
  prev === next ||
  (!!prev &&
    !!next &&
    isMessage(prev, schema) &&
    isMessage(next, schema) &&
    equals(schema, prev, next));

export const sameMessageList = <Desc extends DescMessage>(
  schema: Desc,
  prev: MessageShape<Desc>[],
  next: MessageShape<Desc>[]
): boolean =>
  prev === next ||
  (prev.length === next.length &&
    prev.every((item, index) => sameMessage(schema, item, next[index])));

// Rollout gets deep structural sharing: a poll tick that changes one running
// task's fields hands out new references for that task (plus its stage and the
// rollout shell) only — every other task and stage keeps its previous
// reference, so their memoized consumers skip re-rendering. Rebuilt levels are
// plain spreads, which keep $typeName and stay comparable by equals() on the
// next tick.
export const preserveRolloutIdentities = (
  prev: Rollout | undefined,
  next: Rollout | undefined
): Rollout | undefined => {
  if (sameMessage(RolloutSchema, prev, next)) {
    return prev;
  }
  if (!prev || !next) {
    return next;
  }
  const prevStages = new Map(prev.stages.map((stage) => [stage.name, stage]));
  const stages = next.stages.map((nextStage): Stage => {
    const prevStage = prevStages.get(nextStage.name);
    if (!prevStage) {
      return nextStage;
    }
    if (sameMessage(StageSchema, prevStage, nextStage)) {
      return prevStage;
    }
    const prevTasks = new Map(prevStage.tasks.map((task) => [task.name, task]));
    const tasks = nextStage.tasks.map((nextTask): Task => {
      const prevTask = prevTasks.get(nextTask.name);
      return prevTask && sameMessage(TaskSchema, prevTask, nextTask)
        ? prevTask
        : nextTask;
    });
    return { ...nextStage, tasks };
  });
  return { ...next, stages };
};

// Task runs get the same per-element sharing: when one run changes (e.g. its
// updateTime ticks while executing), every other run keeps its previous
// reference so per-task groups downstream stay identity-stable.
export const preserveTaskRunIdentities = (
  prev: TaskRun[],
  next: TaskRun[]
): TaskRun[] => {
  if (sameMessageList(TaskRunSchema, prev, next)) {
    return prev;
  }
  const prevByName = new Map(prev.map((run) => [run.name, run]));
  return next.map((run) => {
    const prevRun = prevByName.get(run.name);
    return prevRun && sameMessage(TaskRunSchema, prevRun, run) ? prevRun : run;
  });
};
