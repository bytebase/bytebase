import axios from "axios";
import {
  UserId,
  IssueId,
  Issue,
  IssueNew,
  IssuePatch,
  IssueState,
  ResourceObject,
  Principal,
  unknown,
  Project,
  ResourceIdentifier,
  ProjectId,
  Stage,
  IssueStatusPatch,
} from "../../types";

function convert(
  issue: ResourceObject,
  includedList: ResourceObject[],
  rootGetters: any
): Issue {
  const creator = rootGetters["principal/principalById"](
    issue.attributes.creatorId
  );
  const updater = rootGetters["principal/principalById"](
    issue.attributes.updaterId
  );

  let assignee = undefined;
  if (issue.attributes.assigneeId) {
    assignee = rootGetters["principal/principalById"](
      issue.attributes.assigneeId
    );
  }

  const subscriberList = (issue.attributes.subscriberIdList as Principal[]).map(
    (principalId) => {
      return rootGetters["principal/principalById"](principalId);
    }
  );

  const projectId = (issue.relationships!.project.data as ResourceIdentifier)
    .id;
  let project: Project = unknown("PROJECT") as Project;
  project.id = projectId;

  const stageList: Stage[] = [];
  for (const item of includedList || []) {
    if (
      item.type == "stage" &&
      (item.relationships!.issue.data as ResourceIdentifier).id == issue.id
    ) {
      const stage: Stage = rootGetters["stage/convertPartial"](
        item,
        includedList
      );
      stageList.push(stage);
    }

    if (
      item.type == "project" &&
      (item.relationships!.issue.data as ResourceIdentifier[]).find(
        (item) => item.id == issue.id
      )
    ) {
      project = rootGetters["project/convert"](item);
    }
  }

  const result: Issue = {
    ...(issue.attributes as Omit<
      Issue,
      | "id"
      | "project"
      | "creator"
      | "updater"
      | "assignee"
      | "subscriberList"
      | "stageList"
    >),
    id: issue.id,
    project,
    creator,
    updater,
    assignee,
    subscriberList,
    stageList,
  };

  // Now we have a complate issue, we assign it back to stage and step
  for (const stage of result.stageList) {
    stage.issue = result;
    for (const step of stage.stepList) {
      step.issue = result;
      step.stage = stage;
    }
  }

  return result;
}

const state: () => IssueState = () => ({
  issueListByUser: new Map(),
  issueById: new Map(),
});

const getters = {
  issueListByUser: (state: IssueState) => (userId: UserId) => {
    return state.issueListByUser.get(userId) || [];
  },

  issueById: (state: IssueState) => (issueId: IssueId): Issue => {
    return state.issueById.get(issueId) || (unknown("ISSUE") as Issue);
  },
};

const actions = {
  async fetchIssueListForUser({ commit, rootGetters }: any, userId: UserId) {
    const data = (
      await axios.get(`/api/issue?user=${userId}&include=project,stage,step`)
    ).data;
    const issueList = data.data.map((issue: ResourceObject) => {
      return convert(issue, data.included, rootGetters);
    });

    commit("setIssueListForUser", { userId, issueList });
    console.log(issueList);
    return issueList;
  },

  async fetchIssueListForProject({ rootGetters }: any, projectId: ProjectId) {
    const data = (
      await axios.get(
        `/api/issue?project=${projectId}&include=project,stage,step`
      )
    ).data;
    const issueList = data.data.map((issue: ResourceObject) => {
      return convert(issue, data.included, rootGetters);
    });

    // The caller consumes directly, so we don't store it.
    return issueList;
  },

  async fetchIssueById({ commit, rootGetters }: any, issueId: IssueId) {
    const data = (
      await axios.get(`/api/issue/${issueId}?include=project,stage,step`)
    ).data;
    const issue = convert(data.data, data.included, rootGetters);
    commit("setIssueById", {
      issueId,
      issue,
    });
    return issue;
  },

  async createIssue({ commit, rootGetters }: any, newIssue: IssueNew) {
    const data = (
      await axios.post(`/api/issue?include=project,stage,step`, {
        data: {
          type: "issuenew",
          attributes: newIssue,
        },
      })
    ).data;
    const createdIssue = convert(data.data, data.included, rootGetters);

    commit("setIssueById", {
      issueId: createdIssue.id,
      issue: createdIssue,
    });

    return createdIssue;
  },

  async patchIssue(
    { commit, dispatch, rootGetters }: any,
    {
      issueId,
      issuePatch,
    }: {
      issueId: IssueId;
      issuePatch: IssuePatch;
    }
  ) {
    const data = (
      await axios.patch(`/api/issue/${issueId}?include=project,stage,step`, {
        data: {
          type: "issuepatch",
          attributes: issuePatch,
        },
      })
    ).data;
    const updatedIssue = convert(data.data, data.included, rootGetters);

    commit("setIssueById", {
      issueId: issueId,
      issue: updatedIssue,
    });

    dispatch("activity/fetchActivityListForIssue", issueId, { root: true });

    return updatedIssue;
  },

  async updateIssueStatus(
    { commit, dispatch, rootGetters }: any,
    {
      issueId,
      issueStatusPatch,
    }: {
      issueId: IssueId;
      issueStatusPatch: IssueStatusPatch;
    }
  ) {
    const data = (
      await axios.patch(
        `/api/issue/${issueId}/status?include=project,stage,step`,
        {
          data: {
            type: "issuestatuspatch",
            attributes: issueStatusPatch,
          },
        }
      )
    ).data;
    const updatedIssue = convert(data.data, data.included, rootGetters);

    commit("setIssueById", {
      issueId: issueId,
      issue: updatedIssue,
    });

    dispatch("activity/fetchActivityListForIssue", issueId, { root: true });

    return updatedIssue;
  },
};

const mutations = {
  setIssueListForUser(
    state: IssueState,
    {
      userId,
      issueList,
    }: {
      userId: UserId;
      issueList: Issue[];
    }
  ) {
    state.issueListByUser.set(userId, issueList);
  },

  setIssueById(
    state: IssueState,
    {
      issueId,
      issue,
    }: {
      issueId: IssueId;
      issue: Issue;
    }
  ) {
    state.issueById.set(issueId, issue);
  },
};

export default {
  namespaced: true,
  state,
  getters,
  actions,
  mutations,
};
