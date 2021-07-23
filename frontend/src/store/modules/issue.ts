import axios from "axios";
import {
  empty,
  EMPTY_ID,
  Issue,
  IssueCreate,
  IssueId,
  IssuePatch,
  IssueState,
  IssueStatus,
  IssueStatusPatch,
  Pipeline,
  PrincipalId,
  Project,
  ProjectId,
  ResourceIdentifier,
  ResourceObject,
  unknown,
} from "../../types";

function convert(
  issue: ResourceObject,
  includedList: ResourceObject[],
  rootGetters: any
): Issue {
  const projectId = (issue.relationships!.project.data as ResourceIdentifier)
    .id;
  let project: Project = unknown("PROJECT") as Project;
  project.id = parseInt(projectId);

  const pipelineId = (issue.relationships!.pipeline.data as ResourceIdentifier)
    .id;
  let pipeline = unknown("PIPELINE") as Pipeline;
  pipeline.id = parseInt(pipelineId);

  for (const item of includedList || []) {
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
    ...(issue.attributes as Omit<Issue, "id" | "project">),
    id: parseInt(issue.id),
    project,
    pipeline,
  };
}

const state: () => IssueState = () => ({
  issueById: new Map(),
});

const getters = {
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
    { rootGetters }: any,
    {
      userId,
      issueStatusList,
      limit,
    }: {
      userId: PrincipalId;
      issueStatusList: IssueStatus[];
      limit?: number;
    }
  ) {
    var url = `/api/issue?user=${userId}&status=${issueStatusList.join(",")}`;
    if (limit) {
      url += `&limit=${limit}`;
    }
    const data = (await axios.get(url)).data;
    const issueList = data.data.map((issue: ResourceObject) => {
      return convert(issue, data.included, rootGetters);
    });

    // The caller consumes directly, so we don't store it.
    return issueList;
  },

  async fetchIssueListForProject({ rootGetters }: any, projectId: ProjectId) {
    const data = (await axios.get(`/api/issue?project=${projectId}`)).data;
    const issueList = data.data.map((issue: ResourceObject) => {
      return convert(issue, data.included, rootGetters);
    });

    // The caller consumes directly, so we don't store it.
    return issueList;
  },

  async fetchIssueById({ commit, rootGetters }: any, issueId: IssueId) {
    const data = (await axios.get(`/api/issue/${issueId}`)).data;
    const issue = convert(data.data, data.included, rootGetters);
    commit("setIssueById", {
      issueId,
      issue,
    });
    return issue;
  },

  async createIssue({ commit, rootGetters }: any, newIssue: IssueCreate) {
    const data = (
      await axios.post(`/api/issue`, {
        data: {
          type: "IssueCreate",
          attributes: {
            ...newIssue,
            // Server expects payload as string, so we stringify first.
            payload: JSON.stringify(newIssue.payload),
          },
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
      await axios.patch(`/api/issue/${issueId}`, {
        data: {
          type: "issuePatch",
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
      await axios.patch(`/api/issue/${issueId}/status`, {
        data: {
          type: "issueStatusPatch",
          attributes: issueStatusPatch,
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
};

const mutations = {
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
