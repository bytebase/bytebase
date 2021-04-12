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
  projectById: new Map(),
});

const getters = {
  projectListByUser: (state: ProjectState) => (userId: UserId) => {
    return state.projectListByUser.get(userId) || [];
  },

  projectById: (state: ProjectState) => (projectId: ProjectId): Project => {
    return state.projectById.get(projectId) || (unknown("PROJECT") as Project);
  },
};

const actions = {
  async fetchProjectListForUser({ commit, rootGetters }: any, userId: UserId) {
    const projectList = (
      await axios.get(`/api/project?userid=${userId}`)
    ).data.data.map((project: ResourceObject) => {
      return convert(project, rootGetters);
    });
    commit("setProjectListForUser", { userId, projectList });
    return projectList;
  },

  async fetchProjectById({ commit, rootGetters }: any, projectId: ProjectId) {
    const project = convert(
      (await axios.get(`/api/project/${projectId}`)).data.data,
      rootGetters
    );
    commit("setProjectById", {
      projectId,
      project,
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

    commit("setProjectById", {
      taskId: createdProject.id,
      project: createdProject,
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

    commit("setProjectById", {
      taskId: projectId,
      task: updatedProject,
    });

    return updatedProject;
  },
};

const mutations = {
  setProjectListForUser(
    state: ProjectState,
    {
      userId,
      projectList,
    }: {
      userId: UserId;
      projectList: Project[];
    }
  ) {
    state.projectListByUser.set(userId, projectList);
  },

  setProjectById(
    state: ProjectState,
    {
      projectId,
      project,
    }: {
      projectId: ProjectId;
      project: Project;
    }
  ) {
    state.projectById.set(projectId, project);
  },
};

export default {
  namespaced: true,
  state,
  getters,
  actions,
  mutations,
};
