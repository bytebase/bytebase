import {
  ResourceIdentifier,
  ResourceObject,
  StepState,
  Step,
  Task,
  Stage,
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
  ) => (step: ResourceObject): Omit<Step, "task" | "stage"> => {
    // It's only called when task/stage tries to convert themselves. So we pass empty includedList here to avoid circular dependency.
    return convertPartial(step, rootGetters);
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
