import axios from "axios";
import {
  PrincipalId,
  IssueId,
  Issue,
  IssueCreate,
  IssuePatch,
  IssueState,
  ResourceObject,
  Principal,
  unknown,
  Project,
  ResourceIdentifier,
  ProjectId,
  IssueStatusPatch,
  Pipeline,
  empty,
  EMPTY_ID,
} from "../../types";

function convert(
  issue: ResourceObject,
  includedList: ResourceObject[],
  rootGetters: any
): Issue {
  const subscriberList = (issue.attributes.subscriberIdList as Principal[]).map(
    (principalId) => {
      return rootGetters["principal/principalById"](principalId);
    }
  );

  const creatorId = (issue.relationships!.creator.data as ResourceIdentifier)
    .id;
  let creator: Principal = unknown("PRINCIPAL") as Principal;
  creator.id = creatorId;

  const updaterId = (issue.relationships!.updater.data as ResourceIdentifier)
    .id;
  let updater: Principal = unknown("PRINCIPAL") as Principal;
  updater.id = updaterId;

  const assigneeId = (issue.relationships!.assignee.data as ResourceIdentifier)
    .id;
  let assignee: Principal = unknown("PRINCIPAL") as Principal;
  assignee.id = assigneeId;

  const projectId = (issue.relationships!.project.data as ResourceIdentifier)
    .id;
  let project: Project = unknown("PROJECT") as Project;
  project.id = projectId;

  const pipelineId = (issue.relationships!.pipeline.data as ResourceIdentifier)
    .id;
  let pipeline = unknown("PIPELINE") as Pipeline;
  pipeline.id = pipelineId;

  for (const item of includedList || []) {
    if (
      item.type == "principal" &&
      (issue.relationships!.creator.data as ResourceIdentifier).id == item.id
    ) {
      creator = rootGetters["principal/convert"](item);
    }

    if (
      item.type == "principal" &&
      (issue.relationships!.updater.data as ResourceIdentifier).id == item.id
    ) {
      updater = rootGetters["principal/convert"](item);
    }

    if (
      item.type == "principal" &&
      (issue.relationships!.assignee.data as ResourceIdentifier).id == item.id
    ) {
      assignee = rootGetters["principal/convert"](item);
    }

    if (
      item.type == "project" &&
      (issue.relationships!.project.data as ResourceIdentifier).id == item.id
    ) {
      project = rootGetters["project/convert"](item);
    }

    if (
      item.type == "pipeline" &&
      issue.relationships!.pipeline.data &&
      (issue.relationships!.pipeline.data as ResourceIdentifier).id == item.id
    ) {
      pipeline = rootGetters["pipeline/convert"](item, includedList);
    }
  }

  return {
    ...(issue.attributes as Omit<
      Issue,
      "id" | "project" | "creator" | "updater" | "assignee" | "subscriberList"
    >),
    id: issue.id,
    project,
    creator,
    updater,
    assignee,
    subscriberList,
    pipeline,
  };
}

const state: () => IssueState = () => ({
  issueListByUser: new Map(),
  issueById: new Map(),
});

const getters = {
  issueListByUser: (state: IssueState) => (userId: PrincipalId) => {
    return state.issueListByUser.get(userId) || [];
  },

  issueById:
    (state: IssueState) =>
    (issueId: IssueId): Issue => {
      if (issueId == EMPTY_ID) {
        return empty("ISSUE") as Issue;
      }

      return state.issueById.get(issueId) || (unknown("ISSUE") as Issue);
    },
};

const actions = {
  async fetchIssueListForUser(
    { commit, rootGetters }: any,
    userId: PrincipalId
  ) {
    const data = (
      await axios.get(
        `/api/issue?user=${userId}&include=principal,project,pipeline,pipeline.stage,pipeline.task`
      )
    ).data;
    const issueList = data.data.map((issue: ResourceObject) => {
      return convert(issue, data.included, rootGetters);
    });

    commit("setIssueListForUser", { userId, issueList });
    return issueList;
  },

  async fetchIssueListForProject({ rootGetters }: any, projectId: ProjectId) {
    const data = (
      await axios.get(
        `/api/issue?project=${projectId}&include=principal,project,pipeline,pipeline.stage,pipeline.task`
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
      await axios.get(
        `/api/issue/${issueId}?include=principal,project,pipeline,pipeline.stage,pipeline.task`
      )
    ).data;
    const issue = convert(data.data, data.included, rootGetters);
    commit("setIssueById", {
      issueId,
      issue,
    });
    return issue;
  },

  async createIssue({ commit, rootGetters }: any, newIssue: IssueCreate) {
    const data = (
      await axios.post(
        `/api/issue?include=principal,project,pipeline,pipeline.stage,pipeline.task`,
        {
          data: {
            type: "IssueCreate",
            attributes: newIssue,
          },
        }
      )
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
      await axios.patch(
        `/api/issue/${issueId}?include=principal,project,pipeline,pipeline.stage,pipeline.task`,
        {
          data: {
            type: "issuepatch",
            attributes: issuePatch,
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
        `/api/issue/${issueId}/status?include=principal,project,pipeline,pipeline.stage,pipeline.task`,
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
      userId: PrincipalId;
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
