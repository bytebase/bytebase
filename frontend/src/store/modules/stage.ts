import { root } from "postcss";
import {
  ResourceIdentifier,
  ResourceObject,
  Stage,
  StageState,
  Step,
  Task,
  TaskId,
  unknown,
} from "../../types";

const state: () => StageState = () => ({});

function convertPartial(
  stage: ResourceObject,
  includedList: ResourceObject[],
  rootGetters: any
): Omit<Stage, "task"> {
  const creator = rootGetters["principal/principalById"](
    stage.attributes.creatorId
  );
  const updater = rootGetters["principal/principalById"](
    stage.attributes.updaterId
  );

  const database = rootGetters["database/databaseById"](
    (stage.relationships!.database.data as ResourceIdentifier).id
  );

  const stepList: Step[] = [];
  for (const item of includedList || []) {
    if (
      item.type == "step" &&
      (item.relationships!.stage.data as ResourceIdentifier).id == stage.id
    ) {
      const step = rootGetters["step/convertPartial"](item);
      stepList.push(step);
    }
  }

  const result: Omit<Stage, "task"> = {
    ...(stage.attributes as Omit<
      Stage,
      "id" | "creator" | "updater" | "task" | "database" | "stepList"
    >),
    id: stage.id,
    creator,
    updater,
    database,
    stepList,
  };

  return result;
}

const getters = {
  convertPartial: (
    state: StageState,
    getters: any,
    rootState: any,
    rootGetters: any
  ) => (stage: ResourceObject, includedList: ResourceObject[]): Stage => {
    // It's only called when task tries to convert itself, so we don't have a task yet.
    const taskId = stage.attributes.taskId as TaskId;
    let task: Task = unknown("TASK") as Task;
    task.id = taskId;

    const result: Stage = {
      ...convertPartial(stage, includedList, rootGetters),
      task,
    };

    for (const step of result.stepList) {
      step.stage = result;
      step.task = task;
    }

    return result;
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
