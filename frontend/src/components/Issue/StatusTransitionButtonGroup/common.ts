import { type DropdownOption } from "naive-ui";
import {
  Issue,
  IssueCreate,
  IssueStatusTransition,
  Principal,
  Task,
} from "@/types";
import { TaskStatusTransition } from "@/utils";

export type IssueContext = {
  currentUser: Principal;
  create: boolean;
  issue: Issue | IssueCreate;
};

export type ExtraAction<T extends "ISSUE" | "TASK" | "TASK-BATCH"> = {
  type: T;
  transition: T extends "ISSUE" ? IssueStatusTransition : TaskStatusTransition;
  target: T extends "ISSUE"
    ? Issue
    : T extends "TASK"
    ? Task
    : T extends "TASK-BATCH"
    ? Task[]
    : unknown;
};

export type ExtraActionOption = DropdownOption &
  ExtraAction<"ISSUE" | "TASK" | "TASK-BATCH">;
