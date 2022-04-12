import axios from "axios";
import {
  empty,
  EMPTY_ID,
  Member,
  MemberId,
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
import { getPrincipalFromIncludedList } from "../pinia";

function convert(
  project: ResourceObject,
  includedList: ResourceObject[],
  rootGetters: any
): Project {
  const attrs = project.attributes as Omit<
    Project,
    "id" | "memberList" | "creator" | "updater"
  >;
  // Only able to assign an empty member list, otherwise would cause circular dependency.
  // This should be fine as we shouldn't access member via member.project.memberList
  const projectWithoutMemberList: Project = {
    id: parseInt(project.id),
    rowStatus: attrs.rowStatus,
    name: attrs.name,
    key: attrs.key,
    creator: getPrincipalFromIncludedList(
      project.relationships!.creator.data,
      includedList
    ),
    updater: getPrincipalFromIncludedList(
      project.relationships!.updater.data,
      includedList
    ),
    createdTs: attrs.createdTs,
    updatedTs: attrs.updatedTs,
    memberList: [],
    workflowType: attrs.workflowType,
    visibility: attrs.visibility,
    tenantMode: attrs.tenantMode,
    dbNameTemplate: attrs.dbNameTemplate,
    roleProvider: attrs.roleProvider,
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
  memberList.sort((a: ProjectMember, b: ProjectMember) => {
    if (a.createdTs === b.createdTs) {
      return a.id - b.id;
    }
    return a.createdTs - b.createdTs;
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
  const attrs = projectMember.attributes as Omit<
    ProjectMember,
    "id" | "project"
  >;

  return {
    id: parseInt(projectMember.id),
    project: unknown("PROJECT") as Project,
    creator: getPrincipalFromIncludedList(
      projectMember.relationships!.creator.data,
      includedList
    ),
    updater: getPrincipalFromIncludedList(
      projectMember.relationships!.updater.data,
      includedList
    ),
    createdTs: attrs.createdTs,
    updatedTs: attrs.updatedTs,
    role: attrs.role,
    principal: getPrincipalFromIncludedList(
      projectMember.relationships!.principal.data,
      includedList
    ),
    roleProvider: attrs.roleProvider,
    payload: JSON.parse((attrs.payload as unknown as string) || "{}"),
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
      rowStatusList = [],
    }: {
      userId: PrincipalId;
      rowStatusList?: RowStatus[];
    }
  ) {
    const projectList: Project[] = [];

    const fetchProjectList = async (rowStatus?: RowStatus) => {
      let path = `/api/project?user=${userId}`;
      if (rowStatus) path += `&rowstatus=${rowStatus}`;
      const data = (await axios.get(path)).data;
      const list: Project[] = data.data.map((project: ResourceObject) => {
        return convert(project, data.included, rootGetters);
      });
      // projects are mutual excluded by different rowstatus
      // so we don't need to unique them by id here
      projectList.push(...list);
    };

    if (rowStatusList.length === 0) {
      // if no rowStatus specified, fetch all
      await fetchProjectList();
    } else {
      // otherwise, fetch different rowStatus one-by-one
      for (const rowStatus of rowStatusList) {
        await fetchProjectList(rowStatus);
      }
    }

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

  // sync member role from vcs
  async syncMemberRoleFromVCS(
    { dispatch }: any,
    { projectId }: { projectId: ProjectId }
  ) {
    await axios.post(`/api/project/${projectId}/sync-member`);
    const updatedProject = await dispatch("fetchProjectById", projectId);

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
