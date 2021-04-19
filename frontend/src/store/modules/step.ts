import {
  ResourceIdentifier,
  ResourceObject,
  StepState,
  Step,
  Task,
  Stage,
  unknown,
  TaskId,
  StageId,
} from "../../types";

const state: () => StepState = () => ({});

function convertPartial(
  step: ResourceObject,
  rootGetters: any
): Omit<Step, "task" | "stage"> {
  const creator = rootGetters["principal/principalById"](
    step.attributes.creatorId
  );
  const updater = rootGetters["principal/principalById"](
    step.attributes.updaterId
  );

  return {
    ...(step.attributes as Omit<
      Step,
      "id" | "creator" | "updater" | "task" | "stage"
    >),
    id: step.id,
    creator,
    updater,
  };
}

function convert(step: ResourceObject, rootGetters: any): Step {
  const task: Task = rootGetters["task/taskById"](step.attributes.taskId);
  const stage: Stage = rootGetters["stage/stageById"](step.attributes.stageId);
  return {
    ...convertPartial(step, rootGetters),
    task,
    stage,
  };
}

const getters = {
  convertPartial: (
    state: StepState,
    getters: any,
    rootState: any,
    rootGetters: any
  ) => (step: ResourceObject): Step => {
    // It's only called when task/stage tries to convert themselves, so we don't have a task/stage yet.
    const taskId = step.attributes.taskId as TaskId;
    let task: Task = unknown("TASK") as Task;
    task.id = taskId;

    const stageId = step.attributes.stageId as StageId;
    let stage: Stage = unknown("STAGE") as Stage;
    stage.id = stageId;

    return {
      ...convertPartial(step, rootGetters),
      task,
      stage,
    };
  },
};

const actions = {};

const mutations = {};

export default {
  namespaced: true,
  state,
  getters,
  actions,
  mutations,
};
