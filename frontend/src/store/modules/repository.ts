import axios from "axios";
import { ProjectId, Repository, RepositoryState } from "../../types";

const state: () => RepositoryState = () => ({
  repositoryByProject: new Map(),
});

const getters = {
  repositoryByProject: (state: RepositoryState) => (projectId: ProjectId) => {
    return state.repositoryByProject.get(projectId);
  },
};

const actions = {
  async fetchRepositoryForProject({ commit }: any, projectId: ProjectId) {
    const repository = (await axios.get(`/api/project/${projectId}/repository`))
      .data.data;
    commit("setRepositoryForProject", {
      projectId,
      repository,
    });
    return repository;
  },
};

const mutations = {
  setRepositoryForProject(
    state: RepositoryState,
    {
      projectId,
      repository,
    }: {
      projectId: ProjectId;
      repository: Repository;
    }
  ) {
    state.repositoryByProject.set(projectId, repository);
  },
};

export default {
  namespaced: true,
  state,
  getters,
  actions,
  mutations,
};
