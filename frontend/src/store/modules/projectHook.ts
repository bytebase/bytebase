import axios from "axios";
import {
  ProjectHook,
  ProjectHookCreate,
  ProjectHookId,
  ProjectHookPatch,
  ProjectHookState,
  ProjectId,
  ResourceObject,
  unknown,
} from "../../types";

function convert(
  projectHook: ResourceObject,
  includedList: ResourceObject[],
  rootGetters: any
): ProjectHook {
  return {
    ...(projectHook.attributes as Omit<ProjectHook, "id">),
    id: parseInt(projectHook.id),
  };
}

const state: () => ProjectHookState = () => ({
  projectHookListByProjectId: new Map(),
});

const getters = {
  projectHookListByProjectId:
    (state: ProjectHookState) =>
    (projectId: ProjectId): ProjectHook[] => {
      return state.projectHookListByProjectId.get(projectId) || [];
    },

  projectHookById:
    (state: ProjectHookState) =>
    (projectId: ProjectId, projectHookId: ProjectHookId): ProjectHook => {
      const list = state.projectHookListByProjectId.get(projectId);
      if (list) {
        for (const hook of list) {
          if (hook.id == projectHookId) {
            return hook;
          }
        }
      }
      return unknown("PROJECT_HOOK") as ProjectHook;
    },
};

const actions = {
  async createProjectHook(
    { commit, rootGetters }: any,
    {
      projectId,
      projectHookCreate,
    }: {
      projectId: ProjectId;
      projectHookCreate: ProjectHookCreate;
    }
  ): Promise<ProjectHook> {
    const data = (
      await axios.post(`/api/project/${projectId}/hook`, {
        data: {
          type: "projectHookCreate",
          attributes: projectHookCreate,
        },
      })
    ).data;
    const createdProjectHook = convert(data.data, data.included, rootGetters);

    commit("upsertProjectHookByProjectId", {
      projectId,
      projectHook: createdProjectHook,
    });

    return createdProjectHook;
  },

  async fetchProjectHookListByProjectId(
    { commit, rootGetters }: any,
    projectId: ProjectId
  ): Promise<ProjectHook[]> {
    const data = (await axios.get(`/api/project/${projectId}/hook`)).data;
    const projectHookList = data.data.map((projectHook: ResourceObject) => {
      return convert(projectHook, data.included, rootGetters);
    });

    commit("setProjectHookListByProjectId", { projectId, projectHookList });

    return projectHookList;
  },

  async fetchProjectHookById(
    { commit, rootGetters }: any,
    {
      projectId,
      projectHookId,
    }: {
      projectId: ProjectId;
      projectHookId: ProjectHookId;
    }
  ): Promise<ProjectHook> {
    const data = (
      await axios.get(`/api/project/${projectId}/hook/${projectHookId}`)
    ).data;
    const projectHook = convert(data.data, data.included, rootGetters);

    commit("upsertProjectHookByProjectId", {
      projectId,
      projectHook,
    });

    return projectHook;
  },

  async updateProjectHookById(
    { commit, rootGetters }: any,
    {
      projectId,
      projectHookId,
      projectHookPatch,
    }: {
      projectId: ProjectId;
      projectHookId: ProjectHookId;
      projectHookPatch: ProjectHookPatch;
    }
  ) {
    const data = (
      await axios.patch(`/api/project/${projectId}/hook/${projectHookId}`, {
        data: {
          type: "projectHookPatch",
          attributes: projectHookPatch,
        },
      })
    ).data;
    const updatedProjectHook = convert(data.data, data.included, rootGetters);

    commit("upsertProjectHookByProjectId", {
      projectId,
      projectHook: updatedProjectHook,
    });

    return updatedProjectHook;
  },

  async deleteProjectHookById(
    { dispatch, commit }: any,
    {
      projectId,
      projectHookId,
    }: {
      projectId: ProjectId;
      projectHookId: ProjectHookId;
    }
  ) {
    await axios.delete(`/api/project/${projectId}/hook/${projectHookId}`);

    commit("deleteProjectHookById", {
      projectId,
      projectHookId,
    });
  },
};

const mutations = {
  setProjectHookListByProjectId(
    state: ProjectHookState,
    {
      projectId,
      projectHookList,
    }: {
      projectId: ProjectId;
      projectHookList: ProjectHook[];
    }
  ) {
    state.projectHookListByProjectId.set(projectId, projectHookList);
  },

  upsertProjectHookByProjectId(
    state: ProjectHookState,
    {
      projectId,
      projectHook,
    }: {
      projectId: ProjectId;
      projectHook: ProjectHook;
    }
  ) {
    const list = state.projectHookListByProjectId.get(projectId);
    if (list) {
      const i = list.findIndex((item) => item.id == projectHook.id);
      if (i >= 0) {
        list[i] = projectHook;
      } else {
        list.push(projectHook);
      }
    } else {
      state.projectHookListByProjectId.set(projectId, [projectHook]);
    }
  },

  deleteProjectHookById(
    state: ProjectHookState,
    {
      projectId,
      projectHookId,
    }: {
      projectId: ProjectId;
      projectHookId: ProjectHookId;
    }
  ) {
    const list = state.projectHookListByProjectId.get(projectId);
    if (list) {
      const i = list.findIndex((item) => item.id == projectHookId);
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
