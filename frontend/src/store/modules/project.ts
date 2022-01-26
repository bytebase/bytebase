import axios from "axios";
import {
  empty,
  EMPTY_ID,
  MemberId,
  Principal,
  PrincipalId,
  Project,
  ProjectCreate,
  ProjectId,
  ProjectMember,
  ProjectMemberCreate,
  ProjectMemberPatch,
  ProjectPatch,
  ProjectState,
  ResourceIdentifier,
  ResourceObject,
  RowStatus,
  unknown,
} from "../../types";

function convert(
  project: ResourceObject,
  includedList: ResourceObject[],
  rootGetters: any
): Project {
  const attrs = project.attributes as Omit<Project, "id" | "memberList">;
  // Only able to assign an empty member list, otherwise would cause circular dependency.
  // This should be fine as we shouldn't access member via member.project.memberList
  const projectWithoutMemberList: Project = {
    id: parseInt(project.id),
    rowStatus: attrs.rowStatus,
    name: attrs.name,
    key: attrs.key,
    creator: attrs.creator,
    updater: attrs.updater,
    createdTs: attrs.createdTs,
    updatedTs: attrs.updatedTs,
    memberList: [],
    workflowType: attrs.workflowType,
    visibility: attrs.visibility,
    tenantMode: attrs.tenantMode,
  };

  const memberList: ProjectMember[] = [];
  for (const item of includedList || []) {
    if (item.type == "projectMember") {
      const projectMemberIdList = project.relationships!.projectMember
        .data as ResourceIdentifier[];
      for (const idItem of projectMemberIdList) {
        if (idItem.id == item.id) {
          const member = convertMember(item, includedList, rootGetters);
          member.project = projectWithoutMemberList;
          memberList.push(member);
        }
      }
    }
  }

  // sort the member list
  memberList.sort((a, b) => {
    // We use auto incremental id. A smaller id suggest this member is created earlier.
    return a.id - b.id;
  });

  return {
    ...(projectWithoutMemberList as Omit<Project, "memberList">),
    memberList,
  };
}

// For now, this is exclusively used as part of converting the Project.
// Upon calling, the project itself is not constructed yet, so we return
// an unknown project first.
function convertMember(
  projectMember: ResourceObject,
  includedList: ResourceObject[],
  rootGetters: any
): ProjectMember {
  const creator = projectMember.attributes.creator as Principal;
  const updater = projectMember.attributes.updater as Principal;
  const principal = projectMember.attributes.principal as Principal;

  const attrs = projectMember.attributes as Omit<
    ProjectMember,
    "id" | "project" | "creator" | "updater" | "principal"
  >;

  return {
    id: parseInt(projectMember.id),
    project: unknown("PROJECT") as Project,
    creator,
    updater,
    createdTs: attrs.createdTs,
    updatedTs: attrs.updatedTs,
    role: attrs.role,
    principal,
  };
}

const state: () => ProjectState = () => ({
  projectById: new Map(),
});

const getters = {
  convert:
    (state: ProjectState, getters: any, rootState: any, rootGetters: any) =>
    (instance: ResourceObject, includedList: ResourceObject[]): Project => {
      return convert(instance, includedList || [], rootGetters);
    },

  projectListByUser:
    (state: ProjectState) =>
    (userId: PrincipalId, rowStatusList?: RowStatus[]): Project[] => {
      const result: Project[] = [];
      for (const [_, project] of state.projectById) {
        if (
          (!rowStatusList && project.rowStatus == "NORMAL") ||
          (rowStatusList && rowStatusList.includes(project.rowStatus))
        ) {
          for (const member of project.memberList) {
            if (member.principal.id == userId) {
              result.push(project);
              break;
            }
          }
        }
      }

      return result;
    },

  projectById:
    (state: ProjectState) =>
    (projectId: ProjectId): Project => {
      if (projectId == EMPTY_ID) {
        return empty("PROJECT") as Project;
      }

      return (
        state.projectById.get(projectId) || (unknown("PROJECT") as Project)
      );
    },
};

const actions = {
  async fetchProjectList({ commit, rootGetters }: any) {
    const data = (await axios.get(`/api/project`)).data;
    const projectList = data.data.map((project: ResourceObject) => {
      return convert(project, data.included, rootGetters);
    });

    commit("upsertProjectList", projectList);
    return projectList;
  },

  async fetchProjectListByUser(
    { commit, rootGetters }: any,
    {
      userId,
      rowStatusList,
    }: {
      userId: PrincipalId;
      rowStatusList?: RowStatus[];
    }
  ) {
    const path =
      `/api/project?user=${userId}` +
      (rowStatusList ? "&rowstatus=" + rowStatusList.join(",") : "");
    const data = (await axios.get(`${path}`)).data;
    const projectList = data.data.map((project: ResourceObject) => {
      return convert(project, data.included, rootGetters);
    });

    commit("upsertProjectList", projectList);
    return projectList;
  },

  async fetchProjectById({ commit, rootGetters }: any, projectId: ProjectId) {
    const data = (await axios.get(`/api/project/${projectId}`)).data;
    const project = convert(data.data, data.included, rootGetters);

    commit("setProjectById", {
      projectId,
      project,
    });
    return project;
  },

  async createProject({ commit, rootGetters }: any, newProject: ProjectCreate) {
    const data = (
      await axios.post(`/api/project`, {
        data: {
          type: "ProjectCreate",
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
      await axios.patch(`/api/project/${projectId}`, {
        data: {
          type: "projectPatch",
          attributes: projectPatch,
        },
      })
    ).data;
    const updatedProject = convert(data.data, data.included, rootGetters);

    commit("setProjectById", {
      projectId,
      project: updatedProject,
    });

    return updatedProject;
  },

  // Project Role Mapping
  // Returns existing member if the principalId has already been created.
  async createdMember(
    { dispatch }: any,
    {
      projectId,
      projectMember,
    }: {
      projectId: ProjectId;
      projectMember: ProjectMemberCreate;
    }
  ) {
    await axios.post(`/api/project/${projectId}/member`, {
      data: {
        type: "projectMemberCreate",
        attributes: projectMember,
      },
    });

    const updatedProject = await dispatch("fetchProjectById", projectId);

    return updatedProject;
  },

  async patchMember(
    { dispatch }: any,
    {
      projectId,
      memberId,
      projectMemberPatch,
    }: {
      projectId: ProjectId;
      memberId: MemberId;
      projectMemberPatch: ProjectMemberPatch;
    }
  ) {
    await axios.patch(`/api/project/${projectId}/member/${memberId}`, {
      data: {
        type: "projectMemberPatch",
        attributes: projectMemberPatch,
      },
    });

    const updatedProject = await dispatch("fetchProjectById", projectId);

    return updatedProject;
  },

  async deleteMember({ dispatch }: any, member: ProjectMember) {
    await axios.delete(`/api/project/${member.project.id}/member/${member.id}`);

    const updatedProject = await dispatch(
      "fetchProjectById",
      member.project.id
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
    for (const project of projectList) {
      state.projectById.set(project.id, project);
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
