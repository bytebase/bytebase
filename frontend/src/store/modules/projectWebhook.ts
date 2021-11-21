import axios from "axios";
import {
  ProjectID,
  ProjectWebhook,
  ProjectWebhookCreate,
  ProjectWebhookID,
  ProjectWebhookPatch,
  ProjectWebhookState,
  ProjectWebhookTestResult,
  ResourceObject,
  unknown,
} from "../../types";

function convert(
  projectWebhook: ResourceObject,
  includedList: ResourceObject[],
  rootGetters: any
): ProjectWebhook {
  return {
    ...(projectWebhook.attributes as Omit<ProjectWebhook, "id">),
    id: parseInt(projectWebhook.id),
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
  projectWebhookListByProjectID: new Map(),
});

const getters = {
  projectWebhookListByProjectID:
    (state: ProjectWebhookState) =>
    (projectID: ProjectID): ProjectWebhook[] => {
      return state.projectWebhookListByProjectID.get(projectID) || [];
    },

  projectWebhookByID:
    (state: ProjectWebhookState) =>
    (
      projectID: ProjectID,
      projectWebhookID: ProjectWebhookID
    ): ProjectWebhook => {
      const list = state.projectWebhookListByProjectID.get(projectID);
      if (list) {
        for (const hook of list) {
          if (hook.id == projectWebhookID) {
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
      projectID,
      projectWebhookCreate,
    }: {
      projectID: ProjectID;
      projectWebhookCreate: ProjectWebhookCreate;
    }
  ): Promise<ProjectWebhook> {
    const data = (
      await axios.post(`/api/project/${projectID}/webhook`, {
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

    commit("upsertProjectWebhookByProjectID", {
      projectID,
      projectWebhook: createdProjectWebhook,
    });

    return createdProjectWebhook;
  },

  async fetchProjectWebhookListByProjectID(
    { commit, rootGetters }: any,
    projectID: ProjectID
  ): Promise<ProjectWebhook[]> {
    const data = (await axios.get(`/api/project/${projectID}/webhook`)).data;
    const projectWebhookList = data.data.map(
      (projectWebhook: ResourceObject) => {
        return convert(projectWebhook, data.included, rootGetters);
      }
    );

    commit("setProjectWebhookListByProjectID", {
      projectID,
      projectWebhookList,
    });

    return projectWebhookList;
  },

  async fetchProjectWebhookByID(
    { commit, rootGetters }: any,
    {
      projectID,
      projectWebhookID,
    }: {
      projectID: ProjectID;
      projectWebhookID: ProjectWebhookID;
    }
  ): Promise<ProjectWebhook> {
    const data = (
      await axios.get(`/api/project/${projectID}/webhook/${projectWebhookID}`)
    ).data;
    const projectWebhook = convert(data.data, data.included, rootGetters);

    commit("upsertProjectWebhookByProjectID", {
      projectID,
      projectWebhook,
    });

    return projectWebhook;
  },

  async updateProjectWebhookByID(
    { commit, rootGetters }: any,
    {
      projectID,
      projectWebhookID,
      projectWebhookPatch,
    }: {
      projectID: ProjectID;
      projectWebhookID: ProjectWebhookID;
      projectWebhookPatch: ProjectWebhookPatch;
    }
  ) {
    const data = (
      await axios.patch(
        `/api/project/${projectID}/webhook/${projectWebhookID}`,
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

    commit("upsertProjectWebhookByProjectID", {
      projectID,
      projectWebhook: updatedProjectWebhook,
    });

    return updatedProjectWebhook;
  },

  async deleteProjectWebhookByID(
    { dispatch, commit }: any,
    {
      projectID,
      projectWebhookID,
    }: {
      projectID: ProjectID;
      projectWebhookID: ProjectWebhookID;
    }
  ) {
    await axios.delete(`/api/project/${projectID}/webhook/${projectWebhookID}`);

    commit("deleteProjectWebhookByID", {
      projectID,
      projectWebhookID,
    });
  },

  async testProjectWebhookByID(
    { dispatch, commit }: any,
    {
      projectID,
      projectWebhookID,
    }: {
      projectID: ProjectID;
      projectWebhookID: ProjectWebhookID;
    }
  ) {
    const data = (
      await axios.get(
        `/api/project/${projectID}/webhook/${projectWebhookID}/test`
      )
    ).data;

    return convertTestResult(data.data);
  },
};

const mutations = {
  setProjectWebhookListByProjectID(
    state: ProjectWebhookState,
    {
      projectID,
      projectWebhookList,
    }: {
      projectID: ProjectID;
      projectWebhookList: ProjectWebhook[];
    }
  ) {
    state.projectWebhookListByProjectID.set(projectID, projectWebhookList);
  },

  upsertProjectWebhookByProjectID(
    state: ProjectWebhookState,
    {
      projectID,
      projectWebhook,
    }: {
      projectID: ProjectID;
      projectWebhook: ProjectWebhook;
    }
  ) {
    const list = state.projectWebhookListByProjectID.get(projectID);
    if (list) {
      const i = list.findIndex((item) => item.id == projectWebhook.id);
      if (i >= 0) {
        list[i] = projectWebhook;
      } else {
        list.push(projectWebhook);
      }
    } else {
      state.projectWebhookListByProjectID.set(projectID, [projectWebhook]);
    }
  },

  deleteProjectWebhookByID(
    state: ProjectWebhookState,
    {
      projectID,
      projectWebhookID,
    }: {
      projectID: ProjectID;
      projectWebhookID: ProjectWebhookID;
    }
  ) {
    const list = state.projectWebhookListByProjectID.get(projectID);
    if (list) {
      const i = list.findIndex((item) => item.id == projectWebhookID);
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
