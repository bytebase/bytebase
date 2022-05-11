import { IssueTemplate } from "@/plugins";
import {
  Issue,
  IssueCreate,
  IssuePatch,
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

type IssueLogic = {
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

  // api endpoint
  patchTask: (
    taskId: TaskId,
    taskPatch: TaskPatch,
    postUpdated?: (updatedTask: Task) => void
  ) => any;
  patchIssue: (
    issuePatch: IssuePatch,
    postUpdated?: (updatedIssue: Issue) => void
  ) => any;

  // edit logic
  allowEditStatement: Ref<boolean>;
  selectedStatement: Ref<string>;
  updateStatement: (
    newStatement: string,
    postUpdated?: (updatedTask: Task) => void
  ) => any;
  allowApplyStatementToOtherStages: Ref<boolean>;
  applyStatementToOtherStages: (statement: string) => any;
  doCreate(): any;

  // events
  selectStageOrTask: (
    stageIdOrIndex: number,
    taskSlug?: string | undefined
  ) => void;
};

export default IssueLogic;
