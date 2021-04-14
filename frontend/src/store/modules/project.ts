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

function convert(
  project: ResourceObject,
  includedList: ResourceObject[],
  rootGetters: any
): Project {
  const creator = rootGetters["principal/principalById"](
    project.attributes.creatorId
  );
  const updater = rootGetters["principal/principalById"](
    project.attributes.updaterId
  );

  const attrs = project.attributes as Omit<
    Project,
    "id" | "creator" | "updater" | "memberList"
  >;
  // Only able to assign an empty member list, otherwise would cause circular dependency.
  // This should be fine as we shouldn't access member via member.project.memberList
  const projectWithoutMemberList: Project = {
    id: project.id,
    rowStatus: attrs.rowStatus,
    name: attrs.name,
    creator,
    updater,
    createdTs: attrs.createdTs,
    lastUpdatedTs: attrs.lastUpdatedTs,
    memberList: [],
  };

  const memberList: ProjectRoleMapping[] = [];
  for (const item of includedList || []) {
    if (
      item.type == "project-role-mapping" &&
      (item.relationships!.project.data as ResourceIdentifier).id == project.id
    ) {
      const roleMapping = convertRoleMapping(item, rootGetters);
      roleMapping.project = projectWithoutMemberList;
      memberList.push(roleMapping);
    }
  }

  return {
    id: project.id,
    rowStatus: attrs.rowStatus,
    name: attrs.name,
    creator,
    updater,
    createdTs: attrs.createdTs,
    lastUpdatedTs: attrs.lastUpdatedTs,
    memberList,
  };
}

// For now, this is exclusively used as part of converting the Project.
// Upon calling, the project itself is not constructed yet, so we return
// an unknown project first.
function convertRoleMapping(
  projectRoleMapping: ResourceObject,
  rootGetters: any
): ProjectRoleMapping {
  const creator = rootGetters["principal/principalById"](
    projectRoleMapping.attributes.creatorId
  );
  const updater = rootGetters["principal/principalById"](
    projectRoleMapping.attributes.updaterId
  );
  const principal = rootGetters["principal/principalById"](
    projectRoleMapping.attributes.principalId
  );

  const attrs = projectRoleMapping.attributes as Omit<
    ProjectRoleMapping,
    "id" | "project" | "creator" | "updater" | "principal"
  >;

  return {
    id: projectRoleMapping.id,
    project: unknown("PROJECT") as Project,
    creator,
    updater,
    createdTs: attrs.createdTs,
    lastUpdatedTs: attrs.lastUpdatedTs,
    role: attrs.role,
    principal,
  };
}

const state: () => ProjectState = () => ({
  projectById: new Map(),
});

const getters = {
  convert: (
    state: ProjectState,
    getters: any,
    rootState: any,
    rootGetters: any
  ) => (instance: ResourceObject, includedList: ResourceObject[]): Project => {
    return convert(instance, includedList, rootGetters);
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
    for (const [_, project] of state.projectById) {
      for (const member of project.memberList) {
        if (member.principal.id == userId) {
          result.push(project);
          break;
        }
      }
    }

    return result;
  },

  projectById: (state: ProjectState) => (projectId: ProjectId): Project => {
    return state.projectById.get(projectId) || (unknown("PROJECT") as Project);
  },
};

const actions = {
  async fetchProjectList({ commit, rootGetters }: any, userId: UserId) {
    const data = (await axios.get(`/api/project?include=projectRoleMapping`))
      .data;
    const projectList = data.data.map((project: ResourceObject) => {
      return convert(project, data.included, rootGetters);
    });

    commit("upsertProjectList", projectList);
    return projectList;
  },

  async fetchProjectListByUser({ commit, rootGetters }: any, userId: UserId) {
    const data = (
      await axios.get(
        `/api/project?userid=${userId}&include=projectRoleMapping`
      )
    ).data;
    const projectList = data.data.map((project: ResourceObject) => {
      return convert(project, data.included, rootGetters);
    });

    commit("upsertProjectList", projectList);
    return projectList;
  },

  async fetchProjectById({ commit, rootGetters }: any, projectId: ProjectId) {
    const data = (
      await axios.get(`/api/project/${projectId}?include=projectRoleMapping`)
    ).data;
    const project = convert(data.data, data.included, rootGetters);

    commit("setProjectById", {
      projectId,
      project,
    });
    return project;
  },

  async createProject({ commit, rootGetters }: any, newProject: ProjectNew) {
    const data = (
      await axios.post(`/api/project?include=projectRoleMapping`, {
        data: {
          type: "projectnew",
          attributes: newProject,
        },
      })
    ).data;
    const createdProject = convert(data.data, data.included, rootGetters);

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
    const data = (
      await axios.patch(
        `/api/project/${projectId}?include=projectRoleMapping`,
        {
          data: {
            type: "projectpatch",
            attributes: projectPatch,
          },
        }
      )
    ).data;
    const updatedProject = convert(data.data, data.included, rootGetters);

    commit("setProjectById", {
      projectId,
      project: updatedProject,
    });

    return updatedProject;
  },

  // Project Role Mapping
  // Returns existing roleMapping if the principalId has already been created.
  async createdRoleMapping(
    { dispatch }: any,
    {
      projectId,
      projectRoleMapping,
    }: {
      projectId: ProjectId;
      projectRoleMapping: ProjectRoleMappingNew;
    }
  ) {
    await axios.post(`/api/project/${projectId}/rolemapping`, {
      data: {
        type: "projectRoleMappingNew",
        attributes: projectRoleMapping,
      },
    });

    const updatedProject = await dispatch("fetchProjectById", projectId);

    return updatedProject;
  },

  async patchRoleMapping(
    { dispatch }: any,
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
    await axios.patch(
      `/api/project/${projectId}/rolemapping/${roleMappingId}`,
      {
        data: {
          type: "projectRoleMappingPatch",
          attributes: projectRoleMappingPatch,
        },
      }
    );

    const updatedProject = await dispatch("fetchProjectById", projectId);

    return updatedProject;
  },

  async deleteRoleMapping({ dispatch }: any, roleMapping: ProjectRoleMapping) {
    await axios.delete(
      `/api/project/${roleMapping.project.id}/rolemapping/${roleMapping.id}`
    );

    const updatedProject = await dispatch(
      "fetchProjectById",
      roleMapping.project.id
    );

    return updatedProject;
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
};

export default {
  namespaced: true,
  state,
  getters,
  actions,
  mutations,
};
