import { IssueTemplate } from "@/plugins";
import {
  Database,
  Issue,
  IssueCreate,
  IssuePatch,
  IssueStatus,
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
  selectedDatabase: Ref<Database | undefined>;

  // ui state logic
  isTenantMode: Ref<boolean>;
  isGhostMode: Ref<boolean>;
  isPITRMode: Ref<boolean>;
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
  createIssue: (issueCreate: IssueCreate) => any;

  // edit logic
  allowEditStatement: Ref<boolean>;
  selectedStatement: Ref<string>;
  updateStatement: (
    newStatement: string,
    postUpdated?: (updatedTask: Task) => void
  ) => any;
  allowApplyStatementToOtherTasks: Ref<boolean>;
  applyStatementToOtherTasks: (statement: string) => any;
  doCreate(): any;

  // events
  onStatusChanged: (eager: boolean) => void;
  selectStageOrTask: (
    stageIdOrIndex: number,
    taskSlug?: string | undefined
  ) => void;
  selectTask: (task: Task) => void;

  // status transition
  allowApplyIssueStatusTransition(issue: Issue, to: IssueStatus): boolean;
  allowApplyTaskStatusTransition(task: Task, to: TaskStatus): boolean;
};

export default IssueLogic;
