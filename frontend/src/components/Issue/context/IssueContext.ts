import {
  Issue,
  IssueCreate,
  Pipeline,
  Project,
  Stage,
  StageCreate,
  Task,
  TaskCreate,
  TaskStatus,
} from "@/types";
import { Ref } from "vue";

type IssueContext = {
  // base params
  create: Ref<boolean>;
  project: Ref<Project>;
  issue: Ref<Issue | IssueCreate>;

  // states
  selectedStage: Ref<Stage | StageCreate>;
  selectedTask: Ref<Task | TaskCreate>;

  // logic functions
  isValidStage: (stage: Stage | StageCreate, index: number) => boolean;
  activeStageOfPipeline: (pipeline: Pipeline) => Stage;
  activeTaskOfPipeline: (pipeline: Pipeline) => Task;
  activeTaskOfStage: (stage: Stage) => Task;
  taskStatusOfStage: (stage: Stage | StageCreate, index: number) => TaskStatus;

  // event handlers
  selectStageOrTask: (
    stageIdOrIndex: number,
    taskSlug?: string | undefined
  ) => void;
};

export default IssueContext;
