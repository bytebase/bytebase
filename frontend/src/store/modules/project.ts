import axios from "axios";
import { UserId, GroupId, Project, ProjectState } from "../../types";

const state: () => ProjectState = () => ({
  projectListByGroup: new Map(),
  projectListByUser: new Map(),
});

const getters = {
  projectListByGroup: (state: ProjectState) => (groupId: GroupId) => {
    return state.projectListByGroup.get(groupId);
  },

  projectListByUser: (state: ProjectState) => (userId: UserId) => {
    return state.projectListByUser.get(userId);
  },

  projectByNamespaceAndSlug: (state: ProjectState) => (
    namespace: string,
    slug: string
  ) => {
    for (const projectList of state.projectListByGroup.values()) {
      for (const project of projectList) {
        if (
          project.attributes.namespace === namespace &&
          project.attributes.slug === slug
        ) {
          return project;
        }
      }
    }
    return null;
  },
};

const actions = {
  async fetchProjectListForGroup({ commit }: any, groupId: GroupId) {
    const projectList = (await axios.get(`/api/project?groupid=${groupId}`))
      .data.data;
    commit("setProjectListForGroup", {
      groupId,
      projectList,
    });
    return projectList;
  },

  async fetchProjectListForUser({ commit }: any, userId: UserId) {
    const projectList = (await axios.get(`/api/project?userid=${userId}`)).data
      .data;
    commit("setProjectListForUser", {
      userId,
      projectList,
    });
    return projectList;
  },
};

const mutations = {
  setProjectListForGroup(
    state: ProjectState,
    {
      groupId,
      projectList,
    }: {
      groupId: GroupId;
      projectList: Project[];
    }
  ) {
    state.projectListByGroup.set(groupId, projectList);
  },

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
};

export default {
  namespaced: true,
  state,
  getters,
  actions,
  mutations,
};
