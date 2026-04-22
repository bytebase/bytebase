import type { Task, TaskRun } from "@/types/proto-es/v1/rollout_service_pb";

export type PendingTaskGroup = {
  environment: string;
  tasks: Task[];
};

export type RollbackItem = {
  task: Task;
  taskRun: TaskRun;
};

export type DeployBranchProps = {
  selectedTask?: Task;
  onCloseTaskPanel: () => void;
};

export type DeployTaskDetailPanelProps = {
  task: Task;
};
