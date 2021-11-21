import axios from "axios";
import {
  empty,
  EMPTY_ID,
  MemberID,
  Principal,
  PrincipalID,
  Project,
  ProjectCreate,
  ProjectID,
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
  };

  const memberList: ProjectMember[] = [];
  for (const item of includedList || []) {
    if (item.type == "projectMember") {
      const projectMemberIDList = project.relationships!.projectMember
        .data as ResourceIdentifier[];
      for (const idItem of projectMemberIDList) {
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
  projectByID: new Map(),
});

const getters = {
  convert:
    (state: ProjectState, getters: any, rootState: any, rootGetters: any) =>
    (instance: ResourceObject, includedList: ResourceObject[]): Project => {
      return convert(instance, includedList || [], rootGetters);
    },

  projectListByUser:
    (state: ProjectState) =>
    (userID: PrincipalID, rowStatusList?: RowStatus[]): Project[] => {
      const result: Project[] = [];
      for (const [_, project] of state.projectByID) {
        if (
          (!rowStatusList && project.rowStatus == "NORMAL") ||
          (rowStatusList && rowStatusList.includes(project.rowStatus))
        ) {
          for (const member of project.memberList) {
            if (member.principal.id == userID) {
              result.push(project);
              break;
            }
          }
        }
      }

      return result;
    },

  projectByID:
    (state: ProjectState) =>
    (projectID: ProjectID): Project => {
      if (projectID == EMPTY_ID) {
        return empty("PROJECT") as Project;
      }

      return (
        state.projectByID.get(projectID) || (unknown("PROJECT") as Project)
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
      userID,
      rowStatusList,
    }: {
      userID: PrincipalID;
      rowStatusList?: RowStatus[];
    }
  ) {
    const path =
      `/api/project?user=${userID}` +
      (rowStatusList ? "&rowstatus=" + rowStatusList.join(",") : "");
    const data = (await axios.get(`${path}`)).data;
    const projectList = data.data.map((project: ResourceObject) => {
      return convert(project, data.included, rootGetters);
    });

    commit("upsertProjectList", projectList);
    return projectList;
  },

  async fetchProjectByID({ commit, rootGetters }: any, projectID: ProjectID) {
    const data = (await axios.get(`/api/project/${projectID}`)).data;
    const project = convert(data.data, data.included, rootGetters);

    commit("setProjectByID", {
      projectID,
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

    commit("setProjectByID", {
      projectID: createdProject.id,
      project: createdProject,
    });

    return createdProject;
  },

  async patchProject(
    { commit, rootGetters }: any,
    {
      projectID,
      projectPatch,
    }: {
      projectID: ProjectID;
      projectPatch: ProjectPatch;
    }
  ) {
    const data = (
      await axios.patch(`/api/project/${projectID}`, {
        data: {
          type: "projectPatch",
          attributes: projectPatch,
        },
      })
    ).data;
    const updatedProject = convert(data.data, data.included, rootGetters);

    commit("setProjectByID", {
      projectID,
      project: updatedProject,
    });

    return updatedProject;
  },

  // Project Role Mapping
  // Returns existing member if the principalID has already been created.
  async createdMember(
    { dispatch }: any,
    {
      projectID,
      projectMember,
    }: {
      projectID: ProjectID;
      projectMember: ProjectMemberCreate;
    }
  ) {
    await axios.post(`/api/project/${projectID}/member`, {
      data: {
        type: "projectMemberCreate",
        attributes: projectMember,
      },
    });

    const updatedProject = await dispatch("fetchProjectByID", projectID);

    return updatedProject;
  },

  async patchMember(
    { dispatch }: any,
    {
      projectID,
      memberID,
      projectMemberPatch,
    }: {
      projectID: ProjectID;
      memberID: MemberID;
      projectMemberPatch: ProjectMemberPatch;
    }
  ) {
    await axios.patch(`/api/project/${projectID}/member/${memberID}`, {
      data: {
        type: "projectMemberPatch",
        attributes: projectMemberPatch,
      },
    });

    const updatedProject = await dispatch("fetchProjectByID", projectID);

    return updatedProject;
  },

  async deleteMember({ dispatch }: any, member: ProjectMember) {
    await axios.delete(`/api/project/${member.project.id}/member/${member.id}`);

    const updatedProject = await dispatch(
      "fetchProjectByID",
      member.project.id
    );

    return updatedProject;
  },
};

const mutations = {
  setProjectByID(
    state: ProjectState,
    {
      projectID,
      project,
    }: {
      projectID: ProjectID;
      project: Project;
    }
  ) {
    state.projectByID.set(projectID, project);
  },

  upsertProjectList(state: ProjectState, projectList: Project[]) {
    for (const project of projectList) {
      state.projectByID.set(project.id, project);
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
