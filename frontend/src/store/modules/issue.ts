import axios from "axios";
import {
  empty,
  EMPTY_ID,
  Issue,
  IssueCreate,
  IssueID,
  IssuePatch,
  IssueState,
  IssueStatus,
  IssueStatusPatch,
  Pipeline,
  PrincipalID,
  Project,
  ProjectID,
  ResourceIdentifier,
  ResourceObject,
  unknown,
} from "../../types";

function convert(
  issue: ResourceObject,
  includedList: ResourceObject[],
  rootGetters: any
): Issue {
  const projectID = (issue.relationships!.project.data as ResourceIdentifier)
    .id;
  let project: Project = unknown("PROJECT") as Project;
  project.id = parseInt(projectID);

  const pipelineID = (issue.relationships!.pipeline.data as ResourceIdentifier)
    .id;
  let pipeline = unknown("PIPELINE") as Pipeline;
  pipeline.id = parseInt(pipelineID);

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
  issueByID: new Map(),
});

const getters = {
  issueByID:
    (state: IssueState) =>
    (issueID: IssueID): Issue => {
      if (issueID == EMPTY_ID) {
        return empty("ISSUE") as Issue;
      }

      return state.issueByID.get(issueID) || (unknown("ISSUE") as Issue);
    },
};

const actions = {
  async fetchIssueList(
    { rootGetters }: any,
    {
      issueStatusList,
      userID,
      projectID,
      limit,
    }: {
      issueStatusList?: IssueStatus[];
      userID?: PrincipalID;
      projectID?: ProjectID;
      limit?: number;
    }
  ) {
    var queryList = [];
    if (issueStatusList) {
      queryList.push(`status=${issueStatusList.join(",")}`);
    }
    if (userID) {
      queryList.push(`user=${userID}`);
    }
    if (projectID) {
      queryList.push(`project=${projectID}`);
    }
    if (limit) {
      queryList.push(`limit=${limit}`);
    }
    var url = "/api/issue";
    if (queryList.length > 0) {
      url += `?${queryList.join("&")}`;
    }
    const data = (await axios.get(url)).data;
    const issueList = data.data.map((issue: ResourceObject) => {
      return convert(issue, data.included, rootGetters);
    });

    // The caller consumes directly, so we don't store it.
    return issueList;
  },

  async fetchIssueByID({ commit, rootGetters }: any, issueID: IssueID) {
    const data = (await axios.get(`/api/issue/${issueID}`)).data;
    const issue = convert(data.data, data.included, rootGetters);
    commit("setIssueByID", {
      issueID,
      issue,
    });

    // It might be the first time the particular instance/database objects are returned,
    // so that we should also update instance/database store, otherwise, we may get
    // unknown instance/database when navigating to other UI from the issue detail page
    // since other UIs are getting instance/database by id from the store.
    // An example is if user navigates to an issue and do a rollback, the constructed rollback
    // issue requires the instance/database exist in the store.
    for (const stage of issue.pipeline.stageList) {
      for (const task of stage.taskList) {
        commit(
          "instance/setInstanceByID",
          {
            instanceID: task.instance.id,
            instance: task.instance,
          },
          { root: true }
        );

        if (task.database) {
          commit(
            "database/upsertDatabaseList",
            {
              databaseList: [task.database],
            },
            { root: true }
          );
        }
      }
    }
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

    commit("setIssueByID", {
      issueID: createdIssue.id,
      issue: createdIssue,
    });

    return createdIssue;
  },

  async patchIssue(
    { commit, dispatch, rootGetters }: any,
    {
      issueID,
      issuePatch,
    }: {
      issueID: IssueID;
      issuePatch: IssuePatch;
    }
  ) {
    const data = (
      await axios.patch(`/api/issue/${issueID}`, {
        data: {
          type: "issuePatch",
          attributes: issuePatch,
        },
      })
    ).data;
    const updatedIssue = convert(data.data, data.included, rootGetters);

    commit("setIssueByID", {
      issueID: issueID,
      issue: updatedIssue,
    });

    dispatch("activity/fetchActivityListForIssue", issueID, { root: true });

    return updatedIssue;
  },

  async updateIssueStatus(
    { commit, dispatch, rootGetters }: any,
    {
      issueID,
      issueStatusPatch,
    }: {
      issueID: IssueID;
      issueStatusPatch: IssueStatusPatch;
    }
  ) {
    const data = (
      await axios.patch(`/api/issue/${issueID}/status`, {
        data: {
          type: "issueStatusPatch",
          attributes: issueStatusPatch,
        },
      })
    ).data;
    const updatedIssue = convert(data.data, data.included, rootGetters);

    commit("setIssueByID", {
      issueID: issueID,
      issue: updatedIssue,
    });

    dispatch("activity/fetchActivityListForIssue", issueID, { root: true });

    return updatedIssue;
  },
};

const mutations = {
  setIssueByID(
    state: IssueState,
    {
      issueID,
      issue,
    }: {
      issueID: IssueID;
      issue: Issue;
    }
  ) {
    state.issueByID.set(issueID, issue);
  },
};

export default {
  namespaced: true,
  state,
  getters,
  actions,
  mutations,
};
