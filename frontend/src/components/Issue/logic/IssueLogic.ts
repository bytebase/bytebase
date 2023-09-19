import { type useDialog } from "naive-ui";
import type { Ref } from "vue";
import { IssueTemplate } from "@/plugins";
import {
  ComposedDatabase,
  ComposedProject,
  Issue,
  IssueCreate,
  IssuePatch,
  IssueStatus,
  Pipeline,
  SheetId,
  Stage,
  StageCreate,
  Task,
  TaskCreate,
  TaskId,
  TaskPatch,
  TaskStatus,
} from "@/types";

type IssueLogic = {
  // base params
  create: Ref<boolean>;
  project: Ref<ComposedProject>;
  issue: Ref<Issue | IssueCreate>;
  template: Ref<IssueTemplate>;

  // states
  selectedStage: Ref<Stage | StageCreate>;
  selectedTask: Ref<Task | TaskCreate>;
  selectedDatabase: Ref<ComposedDatabase | undefined>;

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

  // state logic
  initialTaskListStatementFromRoute: () => any;

  // edit logic
  allowEditStatement: Ref<boolean>;
  selectedStatement: Ref<string>;
  updateStatement: (newStatement: string) => any;
  updateSheetId: (sheetId: SheetId) => void;
  allowApplyTaskStateToOthers: Ref<boolean>;
  applyTaskStateToOthers: (task: TaskCreate) => any;
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

  // misc
  dialog: ReturnType<typeof useDialog>;
};

export default IssueLogic;
