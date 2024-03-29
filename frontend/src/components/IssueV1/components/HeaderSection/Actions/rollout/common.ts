import type {
  StageRolloutAction,
  TaskRolloutAction,
} from "@/components/IssueV1/logic";
import type { ContextMenuButtonAction } from "@/components/v2";

export type RolloutAction<T = "TASK" | "STAGE"> = {
  target: T;
  action: T extends "TASK" ? TaskRolloutAction : StageRolloutAction;
};

export type RolloutButtonAction = ContextMenuButtonAction<RolloutAction>;
