import type { DropdownOption } from "naive-ui";
import type {
  IssueStatusAction,
  TaskRolloutAction,
} from "@/components/IssueV1/logic";
import type { ComposedIssue } from "@/types";
import type { Task } from "@/types/proto-es/v1/rollout_service_pb";

export type ExtraAction<T extends "ISSUE" | "TASK" | "TASK-BATCH"> = {
  type: T;
  action: T extends "ISSUE" ? IssueStatusAction : TaskRolloutAction;
  target: T extends "ISSUE"
    ? ComposedIssue
    : T extends "TASK"
      ? Task
      : T extends "TASK-BATCH"
        ? Task[]
        : unknown;
};

export type ExtraActionOption = DropdownOption &
  ExtraAction<"ISSUE" | "TASK" | "TASK-BATCH">;
