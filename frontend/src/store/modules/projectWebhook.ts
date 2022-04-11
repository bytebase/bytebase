import axios from "axios";
import {
  ProjectId,
  ProjectWebhook,
  ProjectWebhookCreate,
  ProjectWebhookId,
  ProjectWebhookPatch,
  ProjectWebhookState,
  ProjectWebhookTestResult,
  ResourceObject,
  unknown,
} from "../../types";
import { getPrincipalFromIncludedList } from "../pinia";

function convert(
  projectWebhook: ResourceObject,
  includedList: ResourceObject[],
  rootGetters: any
): ProjectWebhook {
  return {
    ...(projectWebhook.attributes as Omit<
      ProjectWebhook,
      "id" | "creator" | "updater"
    >),
    id: parseInt(projectWebhook.id),
    creator: getPrincipalFromIncludedList(
      projectWebhook.relationships!.creator.data,
      includedList
    ),
    updater: getPrincipalFromIncludedList(
      projectWebhook.relationships!.updater.data,
      includedList
    ),
  };
}

function convertTestResult(
  testResult: ResourceObject
): ProjectWebhookTestResult {
  return {
    ...(testResult.attributes as ProjectWebhookTestResult),
  };
}

const state: () => ProjectWebhookState = () => ({
  projectWebhookListByProjectId: new Map(),
});

const getters = {
  projectWebhookListByProjectId:
    (state: ProjectWebhookState) =>
    (projectId: ProjectId): ProjectWebhook[] => {
      return state.projectWebhookListByProjectId.get(projectId) || [];
    },

  projectWebhookById:
    (state: ProjectWebhookState) =>
    (
      projectId: ProjectId,
      projectWebhookId: ProjectWebhookId
    ): ProjectWebhook => {
      const list = state.projectWebhookListByProjectId.get(projectId);
      if (list) {
        for (const hook of list) {
          if (hook.id == projectWebhookId) {
            return hook;
          }
        }
      }
      return unknown("PROJECT_HOOK") as ProjectWebhook;
    },
};

const actions = {
  async createProjectWebhook(
    { commit, rootGetters }: any,
    {
      projectId,
      projectWebhookCreate,
    }: {
      projectId: ProjectId;
      projectWebhookCreate: ProjectWebhookCreate;
    }
  ): Promise<ProjectWebhook> {
    const data = (
      await axios.post(`/api/project/${projectId}/webhook`, {
        data: {
          type: "projectWebhookCreate",
          attributes: projectWebhookCreate,
        },
      })
    ).data;
    const createdProjectWebhook = convert(
      data.data,
      data.included,
      rootGetters
    );

    commit("upsertProjectWebhookByProjectId", {
      projectId,
      projectWebhook: createdProjectWebhook,
    });

    return createdProjectWebhook;
  },

  async fetchProjectWebhookListByProjectId(
    { commit, rootGetters }: any,
    projectId: ProjectId
  ): Promise<ProjectWebhook[]> {
    const data = (await axios.get(`/api/project/${projectId}/webhook`)).data;
    const projectWebhookList = data.data.map(
      (projectWebhook: ResourceObject) => {
        return convert(projectWebhook, data.included, rootGetters);
      }
    );

    commit("setProjectWebhookListByProjectId", {
      projectId,
      projectWebhookList,
    });

    return projectWebhookList;
  },

  async fetchProjectWebhookById(
    { commit, rootGetters }: any,
    {
      projectId,
      projectWebhookId,
    }: {
      projectId: ProjectId;
      projectWebhookId: ProjectWebhookId;
    }
  ): Promise<ProjectWebhook> {
    const data = (
      await axios.get(`/api/project/${projectId}/webhook/${projectWebhookId}`)
    ).data;
    const projectWebhook = convert(data.data, data.included, rootGetters);

    commit("upsertProjectWebhookByProjectId", {
      projectId,
      projectWebhook,
    });

    return projectWebhook;
  },

  async updateProjectWebhookById(
    { commit, rootGetters }: any,
    {
      projectId,
      projectWebhookId,
      projectWebhookPatch,
    }: {
      projectId: ProjectId;
      projectWebhookId: ProjectWebhookId;
      projectWebhookPatch: ProjectWebhookPatch;
    }
  ) {
    const data = (
      await axios.patch(
        `/api/project/${projectId}/webhook/${projectWebhookId}`,
        {
          data: {
            type: "projectWebhookPatch",
            attributes: projectWebhookPatch,
          },
        }
      )
    ).data;
    const updatedProjectWebhook = convert(
      data.data,
      data.included,
      rootGetters
    );

    commit("upsertProjectWebhookByProjectId", {
      projectId,
      projectWebhook: updatedProjectWebhook,
    });

    return updatedProjectWebhook;
  },

  async deleteProjectWebhookById(
    { dispatch, commit }: any,
    {
      projectId,
      projectWebhookId,
    }: {
      projectId: ProjectId;
      projectWebhookId: ProjectWebhookId;
    }
  ) {
    await axios.delete(`/api/project/${projectId}/webhook/${projectWebhookId}`);

    commit("deleteProjectWebhookById", {
      projectId,
      projectWebhookId,
    });
  },

  async testProjectWebhookById(
    { dispatch, commit }: any,
    {
      projectId,
      projectWebhookId,
    }: {
      projectId: ProjectId;
      projectWebhookId: ProjectWebhookId;
    }
  ) {
    const data = (
      await axios.get(
        `/api/project/${projectId}/webhook/${projectWebhookId}/test`
      )
    ).data;

    return convertTestResult(data.data);
  },
};

const mutations = {
  setProjectWebhookListByProjectId(
    state: ProjectWebhookState,
    {
      projectId,
      projectWebhookList,
    }: {
      projectId: ProjectId;
      projectWebhookList: ProjectWebhook[];
    }
  ) {
    state.projectWebhookListByProjectId.set(projectId, projectWebhookList);
  },

  upsertProjectWebhookByProjectId(
    state: ProjectWebhookState,
    {
      projectId,
      projectWebhook,
    }: {
      projectId: ProjectId;
      projectWebhook: ProjectWebhook;
    }
  ) {
    const list = state.projectWebhookListByProjectId.get(projectId);
    if (list) {
      const i = list.findIndex((item) => item.id == projectWebhook.id);
      if (i >= 0) {
        list[i] = projectWebhook;
      } else {
        list.push(projectWebhook);
      }
    } else {
      state.projectWebhookListByProjectId.set(projectId, [projectWebhook]);
    }
  },

  deleteProjectWebhookById(
    state: ProjectWebhookState,
    {
      projectId,
      projectWebhookId,
    }: {
      projectId: ProjectId;
      projectWebhookId: ProjectWebhookId;
    }
  ) {
    const list = state.projectWebhookListByProjectId.get(projectId);
    if (list) {
      const i = list.findIndex((item) => item.id == projectWebhookId);
      if (i >= 0) {
        list.splice(i, 1);
      }
    }
  },
};

export default {
  namespaced: true,
  state,
  getters,
  actions,
  mutations,
};
