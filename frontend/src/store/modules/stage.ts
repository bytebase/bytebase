import { root } from "postcss";
import {
  ResourceIdentifier,
  ResourceObject,
  Stage,
  StageState,
  Step,
} from "../../types";

const state: () => StageState = () => ({});

function convertPartial(
  stage: ResourceObject,
  includedList: ResourceObject[],
  rootGetters: any
): Omit<Stage, "Task"> {
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

  return {
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
}

const getters = {
  convertPartial: (
    state: StageState,
    getters: any,
    rootState: any,
    rootGetters: any
  ) => (
    stage: ResourceObject,
    includedList: ResourceObject[]
  ): Omit<Stage, "Task"> => {
    // It's only called when task tries to convert itself. So we pass empty includedList here to avoid circular dependency.
    return convertPartial(stage, includedList, rootGetters);
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
