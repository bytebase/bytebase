import axios from "axios";
import { root } from "postcss";
import {
  ResourceIdentifier,
  ResourceObject,
  Stage,
  StageId,
  StageState,
  StageStatusPatch,
  Step,
  Issue,
  IssueId,
  unknown,
} from "../../types";

const state: () => StageState = () => ({});

function convertPartial(
  stage: ResourceObject,
  includedList: ResourceObject[],
  rootGetters: any
): Omit<Stage, "issue"> {
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

  const result: Omit<Stage, "issue"> = {
    ...(stage.attributes as Omit<
      Stage,
      "id" | "creator" | "updater" | "issue" | "database" | "stepList"
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
    // It's only called when issue tries to convert itself, so we don't have a issue yet.
    const issueId = stage.attributes.issueId as IssueId;
    let issue: Issue = unknown("ISSUE") as Issue;
    issue.id = issueId;

    const result: Stage = {
      ...convertPartial(stage, includedList, rootGetters),
      issue,
    };

    for (const step of result.stepList) {
      step.stage = result;
      step.issue = issue;
    }

    return result;
  },

  async updateStageStatus(
    { dispatch }: any,
    {
      issueId,
      stageId,
      stageStatusPatch,
    }: {
      issueId: IssueId;
      stageId: StageId;
      stageStatusPatch: StageStatusPatch;
    }
  ) {
    const data = (
      await axios.patch(`/api/issue/${issueId}/stage/${stageId}/status`, {
        data: {
          type: "stagestatuspatch",
          attributes: stageStatusPatch,
        },
      })
    ).data;

    dispatch("issue/fetchIssueById", issueId, { root: true });
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
