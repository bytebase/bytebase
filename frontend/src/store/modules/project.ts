import axios from "axios";
import {
  PrincipalId,
  ProjectId,
  Project,
  ProjectCreate,
  ProjectPatch,
  ProjectState,
  ResourceObject,
  unknown,
  ProjectMember,
  ProjectMemberCreate,
  ProjectMemberPatch,
  MemberId,
  ResourceIdentifier,
  RowStatus,
  EMPTY_ID,
  empty,
  Principal,
} from "../../types";

function convert(
  project: ResourceObject,
  includedList: ResourceObject[],
  rootGetters: any
): Project {
  const creator = project.attributes.creator as Principal;
  const updater = project.attributes.updater as Principal;

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
    key: attrs.key,
    creator,
    updater,
    createdTs: attrs.createdTs,
    updatedTs: attrs.updatedTs,
    memberList: [],
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
  const creatorId = (
    projectMember.relationships!.creator.data as ResourceIdentifier
  ).id;
  let creator: Principal = unknown("PRINCIPAL") as Principal;
  creator.id = creatorId;

  const updaterId = (
    projectMember.relationships!.updater.data as ResourceIdentifier
  ).id;
  let updater: Principal = unknown("PRINCIPAL") as Principal;
  updater.id = updaterId;

  const principalId = (
    projectMember.relationships!.updater.data as ResourceIdentifier
  ).id;
  let principal: Principal = unknown("PRINCIPAL") as Principal;
  principal.id = principalId;

  for (const item of includedList || []) {
    if (
      item.type == "principal" &&
      (projectMember.relationships!.creator.data as ResourceIdentifier).id ==
        item.id
    ) {
      creator = rootGetters["principal/convert"](item);
    }

    if (
      item.type == "principal" &&
      (projectMember.relationships!.updater.data as ResourceIdentifier).id ==
        item.id
    ) {
      updater = rootGetters["principal/convert"](item);
    }

    if (
      item.type == "principal" &&
      (projectMember.relationships!.principal.data as ResourceIdentifier).id ==
        item.id
    ) {
      principal = rootGetters["principal/convert"](item);
    }
  }

  const attrs = projectMember.attributes as Omit<
    ProjectMember,
    "id" | "project" | "creator" | "updater" | "principal"
  >;

  return {
    id: projectMember.id,
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
    const data = (await axios.get(`/api/project?include=projectMember`)).data;
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
    const data = (await axios.get(`${path}&include=projectMember`)).data;
    const projectList = data.data.map((project: ResourceObject) => {
      return convert(project, data.included, rootGetters);
    });

    commit("upsertProjectList", projectList);
    return projectList;
  },

  async fetchProjectById({ commit, rootGetters }: any, projectId: ProjectId) {
    const data = (
      await axios.get(`/api/project/${projectId}?include=projectMember`)
    ).data;
    const project = convert(data.data, data.included, rootGetters);

    commit("setProjectById", {
      projectId,
      project,
    });
    return project;
  },

  async createProject({ commit, rootGetters }: any, newProject: ProjectCreate) {
    const data = (
      await axios.post(`/api/project?include=projectMember`, {
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
      await axios.patch(`/api/project/${projectId}?include=projectMember`, {
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
