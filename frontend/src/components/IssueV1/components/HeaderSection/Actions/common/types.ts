import { DropdownOption } from "naive-ui";
import {
  IssueStatusAction,
  TaskRolloutAction,
} from "@/components/IssueV1/logic";
import { ComposedIssue } from "@/types";
import { Task } from "@/types/proto/v1/rollout_service";

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
