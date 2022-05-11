import { IssueTemplate } from "@/plugins";
import {
  Issue,
  IssueCreate,
  Pipeline,
  Project,
  Stage,
  StageCreate,
  Task,
  TaskCreate,
  TaskId,
  TaskPatch,
  TaskStatus,
} from "@/types";
import { Ref } from "vue";

type IssueContext = {
  // base params
  create: Ref<boolean>;
  project: Ref<Project>;
  issue: Ref<Issue | IssueCreate>;
  template: Ref<IssueTemplate>;

  // states
  selectedStage: Ref<Stage | StageCreate>;
  selectedTask: Ref<Task | TaskCreate>;

  // ui state logic
  isTenantMode: Ref<boolean>;
  isGhostMode: Ref<boolean>;
  isValidStage: (stage: Stage | StageCreate, index: number) => boolean;
  activeStageOfPipeline: (pipeline: Pipeline) => Stage;
  activeTaskOfPipeline: (pipeline: Pipeline) => Task;
  activeTaskOfStage: (stage: Stage) => Task;
  taskStatusOfStage: (stage: Stage | StageCreate, index: number) => TaskStatus;

  // edit logic
  patchTask: (
    taskId: TaskId,
    taskPatch: TaskPatch,
    postUpdated?: (updatedTask: Task) => void
  ) => any;
  allowEditStatement: Ref<boolean>;
  selectedStatement: Ref<string>;
  updateStatement: (
    newStatement: string,
    postUpdated?: (updatedTask: Task) => void
  ) => any;
  allowApplyStatementToOtherStages: Ref<boolean>;
  applyStatementToOtherStages: (statement: string) => any;
  doCreate(): any;
  // updateName: (
  //   newName: string,
  //   postUpdated: (updatedIssue: Issue) => void
  // ) => any;
  // updateDescription: (
  //   newDescription: string,
  //   postUpdated: (updatedIssue: Issue) => void
  // ) => any;
  // updateAssigneeId: (newAssigneeId: PrincipalId) => any;
  // updateEarliestAllowedTime: (newEarliestAllowedTsMs: number) => any;
  // addSubscriberId: (subscriberId: PrincipalId) => any;
  // removeSubscriberId: (subscriberId: PrincipalId) => any;

  // events
  emit: {
    (event: "status-changed", eager: boolean): void;
  };
  selectStageOrTask: (
    stageIdOrIndex: number,
    taskSlug?: string | undefined
  ) => void;
};

export default IssueContext;
