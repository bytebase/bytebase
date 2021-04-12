import axios from "axios";
import {
  UserId,
  ProjectId,
  Project,
  ProjectNew,
  ProjectPatch,
  ProjectState,
  ResourceObject,
  unknown,
} from "../../types";

function convert(project: ResourceObject, rootGetters: any): Project {
  const creator = rootGetters["principal/principalById"](
    project.attributes.creatorId
  );
  const updater = rootGetters["principal/principalById"](
    project.attributes.updaterId
  );

  return {
    id: project.id,
    creator,
    updater,
    ...(project.attributes as Omit<Project, "id" | "creator" | "updater">),
  };
}

const state: () => ProjectState = () => ({
  projectListByUser: new Map(),
});

const getters = {
  convert: (
    state: ProjectState,
    getters: any,
    rootState: any,
    rootGetters: any
  ) => (instance: ResourceObject): Project => {
    return convert(instance, rootGetters);
  },

  projectListByUser: (state: ProjectState) => (userId: UserId) => {
    return state.projectListByUser.get(userId) || [];
  },

  projectById: (state: ProjectState) => (projectId: ProjectId): Project => {
    for (let [_, projectList] of state.projectListByUser) {
      const project = projectList.find((item: Project) => item.id == projectId);
      if (project) {
        return project;
      }
    }
    return unknown("PROJECT") as Project;
  },
};

const actions = {
  async fetchProjectListByUser({ commit, rootGetters }: any, userId: UserId) {
    const projectList = (
      await axios.get(`/api/project?userid=${userId}`)
    ).data.data.map((project: ResourceObject) => {
      return convert(project, rootGetters);
    });
    commit("upsertProjectList", { projectList, userId });
    return projectList;
  },

  async fetchProjectById({ commit, rootGetters }: any, projectId: ProjectId) {
    const project = convert(
      (await axios.get(`/api/project/${projectId}`)).data.data,
      rootGetters
    );
    commit("upsertProjectList", {
      projectList: [project],
    });
    return project;
  },

  async createProject({ commit, rootGetters }: any, newProject: ProjectNew) {
    const createdProject = convert(
      (
        await axios.post(`/api/project`, {
          data: {
            type: "projectnew",
            attributes: newProject,
          },
        })
      ).data.data,
      rootGetters
    );

    commit("upsertProjectList", {
      projectList: [createdProject],
    });

    return createdProject;
  },

  async patchProject(
    { commit, rootGetters }: any,
    {
      projectId,
      projectPatch,
    }: {
      projectId: ProjectId;
      projectPatch: ProjectPatch;
    }
  ) {
    const updatedProject = convert(
      (
        await axios.patch(`/api/project/${projectId}`, {
          data: {
            type: "projectpatch",
            attributes: projectPatch,
          },
        })
      ).data.data,
      rootGetters
    );

    commit("upsertProjectList", {
      projectList: [updatedProject],
    });

    return updatedProject;
  },
};

const mutations = {
  upsertProjectList(
    state: ProjectState,
    {
      projectList,
      userId,
    }: {
      projectList: Project[];
      userId?: UserId;
    }
  ) {
    if (userId) {
      state.projectListByUser.set(userId, projectList);
    } else {
      for (const project of projectList) {
        for (let [_, projectList] of state.projectListByUser) {
          const i = projectList.findIndex(
            (item: Project) => item.id == project.id
          );
          if (i != -1) {
            projectList[i] = project;
          } else {
            projectList.push(project);
          }
        }
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
