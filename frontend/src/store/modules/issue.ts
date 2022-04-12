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
  Principal,
  PrincipalId,
  Project,
  ProjectId,
  ResourceIdentifier,
  ResourceObject,
  unknown,
} from "../../types";
import {
  getPrincipalFromIncludedList,
  useActivityStore,
  useDatabaseStore,
  useInstanceStore,
} from "../pinia-modules";

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

  const subscriberList = [] as Principal[];
  if (issue.relationships!.subscriberList.data) {
    for (const subscriberData of issue.relationships!.subscriberList
      .data as ResourceIdentifier[]) {
      subscriberList.push(
        getPrincipalFromIncludedList(subscriberData, includedList)
      );
    }
  }

  return {
    ...(issue.attributes as Omit<
      Issue,
      "id" | "project" | "creator" | "updater" | "assignee" | "subscriberList"
    >),
    id: parseInt(issue.id),
    creator: getPrincipalFromIncludedList(
      issue.relationships!.creator.data,
      includedList
    ),
    updater: getPrincipalFromIncludedList(
      issue.relationships!.updater.data,
      includedList
    ),
    assignee: getPrincipalFromIncludedList(
      issue.relationships!.assignee.data,
      includedList
    ),
    project,
    pipeline,
    subscriberList: subscriberList,
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
  async fetchIssueList(
    { rootGetters }: any,
    {
      issueStatusList,
      userId,
      projectId,
      limit,
    }: {
      issueStatusList?: IssueStatus[];
      userId?: PrincipalId;
      projectId?: ProjectId;
      limit?: number;
    }
  ) {
    const queryList = [];
    if (issueStatusList) {
      queryList.push(`status=${issueStatusList.join(",")}`);
    }
    if (userId) {
      queryList.push(`user=${userId}`);
    }
    if (projectId) {
      queryList.push(`project=${projectId}`);
    }
    if (limit) {
      queryList.push(`limit=${limit}`);
    }
    let url = "/api/issue";
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

  async fetchIssueById({ commit, rootGetters }: any, issueId: IssueId) {
    const data = (await axios.get(`/api/issue/${issueId}`)).data;
    const issue = convert(data.data, data.included, rootGetters);
    commit("setIssueById", {
      issueId,
      issue,
    });

    // It might be the first time the particular instance/database objects are returned,
    // so that we should also update instance/database store, otherwise, we may get
    // unknown instance/database when navigating to other UI from the issue detail page
    // since other UIs are getting instance/database by id from the store.
    const instanceStore = useInstanceStore();
    const databaseStore = useDatabaseStore();
    for (const stage of issue.pipeline.stageList) {
      for (const task of stage.taskList) {
        instanceStore.setInstanceById({
          instanceId: task.instance.id,
          instance: task.instance,
        });

        if (task.database) {
          databaseStore.upsertDatabaseList({
            databaseList: [task.database],
          });
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
            createContext: JSON.stringify(newIssue.createContext),
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

  async validateIssue({ commit, rootGetters }: any, newIssue: IssueCreate) {
    const data = (
      await axios.post(`/api/issue`, {
        data: {
          type: "IssueCreate",
          attributes: {
            ...newIssue,
            // Server expects payload as string, so we stringify first.
            createContext: JSON.stringify(newIssue.createContext),
            payload: JSON.stringify(newIssue.payload),
            validateOnly: true,
          },
        },
      })
    ).data;
    const createdIssue = convert(data.data, data.included, rootGetters);
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

    useActivityStore().fetchActivityListForIssue(issueId);

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

    useActivityStore().fetchActivityListForIssue(issueId);

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
