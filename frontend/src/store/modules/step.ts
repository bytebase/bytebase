import {
  ResourceIdentifier,
  ResourceObject,
  StepState,
  Step,
  Issue,
  Stage,
  unknown,
  IssueId,
  StageId,
} from "../../types";

const state: () => StepState = () => ({});

function convertPartial(
  step: ResourceObject,
  rootGetters: any
): Omit<Step, "issue" | "stage"> {
  const creator = rootGetters["principal/principalById"](
    step.attributes.creatorId
  );
  const updater = rootGetters["principal/principalById"](
    step.attributes.updaterId
  );

  return {
    ...(step.attributes as Omit<
      Step,
      "id" | "creator" | "updater" | "issue" | "stage"
    >),
    id: step.id,
    creator,
    updater,
  };
}

function convert(step: ResourceObject, rootGetters: any): Step {
  const issue: Issue = rootGetters["issue/issueById"](step.attributes.issueId);
  const stage: Stage = rootGetters["stage/stageById"](step.attributes.stageId);
  return {
    ...convertPartial(step, rootGetters),
    issue,
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
    // It's only called when issue/stage tries to convert themselves, so we don't have a issue/stage yet.
    const issueId = step.attributes.issueId as IssueId;
    let issue: Issue = unknown("ISSUE") as Issue;
    issue.id = issueId;

    const stageId = step.attributes.stageId as StageId;
    let stage: Stage = unknown("STAGE") as Stage;
    stage.id = stageId;

    return {
      ...convertPartial(step, rootGetters),
      issue,
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
