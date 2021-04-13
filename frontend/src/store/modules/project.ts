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
  ProjectRoleMapping,
  ProjectRoleMappingNew,
  ProjectRoleMappingPatch,
  RoleMappingId,
  PrincipalId,
  ResourceIdentifier,
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

function convertRoleMapping(
  projectRoleMapping: ResourceObject,
  rootGetters: any
): ProjectRoleMapping {
  const project = rootGetters["project/projectById"](
    (projectRoleMapping.relationships!.project.data as ResourceIdentifier).id
  );

  const creator = rootGetters["principal/principalById"](
    projectRoleMapping.attributes.creatorId
  );
  const updater = rootGetters["principal/principalById"](
    projectRoleMapping.attributes.updaterId
  );
  const principal = rootGetters["principal/principalById"](
    projectRoleMapping.attributes.principalId
  );

  return {
    id: projectRoleMapping.id,
    project,
    creator,
    updater,
    principal,
    ...(projectRoleMapping.attributes as Omit<
      ProjectRoleMapping,
      "id" | "project" | "creator" | "updater" | "principal"
    >),
  };
}

const state: () => ProjectState = () => ({
  projectById: new Map(),
  projectIdListByUser: new Map(),
  roleMappingListByProject: new Map(),
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

  projectList: (state: ProjectState) => (): Project[] => {
    const result: Project[] = [];
    for (const [_, project] of state.projectById) {
      result.push(project);
    }
    return result;
  },

  projectListByUser: (state: ProjectState) => (userId: UserId): Project[] => {
    const result: Project[] = [];
    const list = state.projectIdListByUser.get(userId) || [];
    for (const projectId of list) {
      const project = state.projectById.get(projectId);
      if (project) {
        result.push(project);
      }
    }

    return result;
  },

  projectById: (state: ProjectState) => (projectId: ProjectId): Project => {
    return state.projectById.get(projectId) || (unknown("PROJECT") as Project);
  },

  roleMappingListById: (state: ProjectState) => (
    projectId: ProjectId
  ): ProjectRoleMapping[] => {
    return state.roleMappingListByProject.get(projectId) || [];
  },

  roleMappingByProjectAndPrincipalId: (state: ProjectState) => (
    projectId: ProjectId,
    id: PrincipalId
  ): ProjectRoleMapping => {
    const list = state.roleMappingListByProject.get(projectId);
    if (list) {
      const item = list.find((item: ProjectRoleMapping) => {
        return item.id == id;
      });
      if (item) {
        return item;
      }
    }

    return unknown("PROJECT_ROLE_MAPPING") as ProjectRoleMapping;
  },
};

const actions = {
  async fetchProjectList({ commit, rootGetters }: any, userId: UserId) {
    const projectList = (await axios.get(`/api/project`)).data.data.map(
      (project: ResourceObject) => {
        return convert(project, rootGetters);
      }
    );

    commit("upsertProjectList", projectList);
    return projectList;
  },

  async fetchProjectListByUser({ commit, rootGetters }: any, userId: UserId) {
    const projectList = (
      await axios.get(`/api/project?userid=${userId}`)
    ).data.data.map((project: ResourceObject) => {
      return convert(project, rootGetters);
    });

    commit("upsertProjectListByUser", { projectList, userId });
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
      projectId: createdProject.id,
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
      projectId,
      project: updatedProject,
    });

    return updatedProject;
  },

  // Project Role Mapping
  async fetchRoleMappingList(
    { commit, rootGetters }: any,
    projectId: ProjectId
  ) {
    const roleMappingList = (
      await axios.get(`/api/project/${projectId}/rolemapping`)
    ).data.data.map((roleMapping: ResourceObject) => {
      return convertRoleMapping(roleMapping, rootGetters);
    });

    commit("upsertProjectRoleMappingByProject", {
      projectRoleMappingList: roleMappingList,
      projectId,
    });
    return roleMappingList;
  },

  // Returns existing roleMapping if the principalId has already been created.
  async createdRoleMapping(
    { commit, rootGetters }: any,
    {
      projectId,
      projectRoleMapping,
    }: {
      projectId: ProjectId;
      projectRoleMapping: ProjectRoleMappingNew;
    }
  ) {
    const createdRoleMapping = convertRoleMapping(
      (
        await axios.post(`/api/project/${projectId}/rolemapping`, {
          data: {
            type: "projectRoleMappingNew",
            attributes: projectRoleMapping,
          },
        })
      ).data.data,
      rootGetters
    );

    commit("upsertProjectRoleMappingByProject", {
      projectRoleMappingList: [createdRoleMapping],
      projectId,
    });

    return createdRoleMapping;
  },

  async patchRoleMapping(
    { commit, rootGetters }: any,
    {
      projectId,
      roleMappingId,
      projectRoleMappingPatch,
    }: {
      projectId: ProjectId;
      roleMappingId: RoleMappingId;
      projectRoleMappingPatch: ProjectRoleMappingPatch;
    }
  ) {
    const updatedRoleMapping = convertRoleMapping(
      (
        await axios.patch(
          `/api/project/${projectId}/rolemapping/${roleMappingId}`,
          {
            data: {
              type: "projectRoleMappingPatch",
              attributes: projectRoleMappingPatch,
            },
          }
        )
      ).data.data,
      rootGetters
    );

    commit("upsertProjectRoleMappingByProject", {
      projectRoleMappingList: [updatedRoleMapping],
      projectId,
    });

    return updatedRoleMapping;
  },

  async deleteRoleMapping({ commit }: any, roleMapping: ProjectRoleMapping) {
    await axios.delete(
      `/api/project/${roleMapping.project.id}/rolemapping/${roleMapping.id}`
    );

    commit("deleteRoleMapping", roleMapping);
  },
};

const mutations = {
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

  upsertProjectList(state: ProjectState, projectList: Project[]) {
    const idList = [];
    for (const project of projectList) {
      state.projectById.set(project.id, project);
      idList.push(project.id);
    }
  },

  upsertProjectListByUser(
    state: ProjectState,
    {
      userId,
      projectList,
    }: {
      userId: UserId;
      projectList: Project[];
    }
  ) {
    const idList = [];
    for (const project of projectList) {
      state.projectById.set(project.id, project);
      idList.push(project.id);
    }
    state.projectIdListByUser.set(userId, idList);
  },

  upsertProjectRoleMappingByProject(
    state: ProjectState,
    {
      projectRoleMappingList,
      projectId,
    }: {
      projectRoleMappingList: ProjectRoleMapping[];
      projectId: ProjectId;
    }
  ) {
    for (const projectRoleMapping of projectRoleMappingList) {
      const list = state.roleMappingListByProject.get(projectId);
      if (list) {
        const i = list.findIndex((item: ProjectRoleMapping) => {
          return item.id == projectRoleMapping.id;
        });
        if (i >= 0) {
          list[i] = projectRoleMapping;
        } else {
          list.push(projectRoleMapping);
        }
      } else {
        state.roleMappingListByProject.set(projectId, [projectRoleMapping]);
      }

      const list2 = state.projectIdListByUser.get(
        projectRoleMapping.principal.id
      );
      if (list2) {
        const i = list2.findIndex((projectId) => {
          return projectId == projectRoleMapping.project.id;
        });
        if (i == -1) {
          list2.push(projectRoleMapping.project.id);
        }
      }
    }
  },

  deleteRoleMapping(state: ProjectState, roleMapping: ProjectRoleMapping) {
    const list = state.roleMappingListByProject.get(roleMapping.project.id);
    if (list) {
      const i = list.findIndex((item: ProjectRoleMapping) => {
        return item.id == roleMapping.id;
      });
      if (i) {
        list.splice(i, 1);
      }
    }

    const list2 = state.projectIdListByUser.get(roleMapping.principal.id);
    if (list2) {
      const i = list2.findIndex((projectId) => {
        return projectId == roleMapping.project.id;
      });
      if (i) {
        list2.splice(i, 1);
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
