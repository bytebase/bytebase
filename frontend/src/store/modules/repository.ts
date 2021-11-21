import axios from "axios";
import {
  Project,
  ProjectID,
  Repository,
  RepositoryCreate,
  RepositoryPatch,
  RepositoryState,
  ResourceIdentifier,
  ResourceObject,
  unknown,
  VCS,
  VCSID,
} from "../../types";

function convert(
  repository: ResourceObject,
  includedList: ResourceObject[],
  rootGetters: any
): Repository {
  const vcsID = (repository.relationships!.vcs.data as ResourceIdentifier).id;
  let vcs: VCS = unknown("VCS") as VCS;
  vcs.id = parseInt(vcsID);

  const projectID = (
    repository.relationships!.project.data as ResourceIdentifier
  ).id;
  let project: Project = unknown("PROJECT") as Project;
  project.id = parseInt(projectID);

  for (const item of includedList || []) {
    if (item.type == "vcs" && item.id == vcsID) {
      vcs = rootGetters["vcs/convert"](item);
    }
    if (item.type == "project" && item.id == projectID) {
      project = rootGetters["project/convert"](item, includedList);
    }
  }

  return {
    ...(repository.attributes as Omit<Repository, "id" | "vcs" | "project">),
    id: parseInt(repository.id),
    vcs,
    project,
  };
}

const state: () => RepositoryState = () => ({
  repositoryListByVCSID: new Map(),
  repositoryByProjectID: new Map(),
});

const getters = {
  repositoryListByVCSID:
    (state: RepositoryState) =>
    (vcsID: VCSID): Repository[] => {
      return state.repositoryListByVCSID.get(vcsID) || [];
    },

  repositoryByProjectID:
    (state: RepositoryState) =>
    (projectID: ProjectID): Repository => {
      return (
        state.repositoryByProjectID.get(projectID) ||
        (unknown("REPOSITORY") as Repository)
      );
    },
};

const actions = {
  async createRepository(
    { dispatch, commit, rootGetters }: any,
    repositoryCreate: RepositoryCreate
  ): Promise<Repository> {
    const data = (
      await axios.post(
        `/api/project/${repositoryCreate.projectID}/repository`,
        {
          data: {
            type: "RepositoryCreate",
            attributes: repositoryCreate,
          },
        }
      )
    ).data;

    const createdRepository = convert(data.data, data.included, rootGetters);
    commit("setRepositoryByProjectID", {
      projectID: repositoryCreate.projectID,
      repository: createdRepository,
    });

    // Refetch the project as the project workflow type has been updated to "VCS"
    dispatch("project/fetchProjectByID", repositoryCreate.projectID, {
      root: true,
    });

    return createdRepository;
  },

  async fetchRepositoryListByVCSID(
    { commit, rootGetters }: any,
    vcsID: VCSID
  ): Promise<Repository[]> {
    const data = (await axios.get(`/api/vcs/${vcsID}/repository`)).data;

    const repositoryList = data.data.map((repository: ResourceObject) => {
      return convert(repository, data.included, rootGetters);
    });

    commit("setRepositoryListByVCSID", { vcsID, repositoryList });
    return repositoryList;
  },

  async fetchRepositoryByProjectID(
    { commit, rootGetters }: any,
    projectID: ProjectID
  ): Promise<Repository> {
    const data = (await axios.get(`/api/project/${projectID}/repository`)).data;
    const repositoryList = data.data.map((repository: ResourceObject) => {
      return convert(repository, data.included, rootGetters);
    });

    // Expect server to return at most one item, otherwise it will throw error
    if (repositoryList.length > 0) {
      commit("setRepositoryByProjectID", {
        projectID,
        repository: repositoryList[0],
      });
      return repositoryList[0];
    }

    return unknown("REPOSITORY") as Repository;
  },

  async updateRepositoryByProjectID(
    { commit, rootGetters }: any,
    {
      projectID,
      repositoryPatch,
    }: {
      projectID: ProjectID;
      repositoryPatch: RepositoryPatch;
    }
  ) {
    const data = (
      await axios.patch(`/api/project/${projectID}/repository`, {
        data: {
          type: "repositoryPatch",
          attributes: repositoryPatch,
        },
      })
    ).data;

    const updatedRepository = convert(data.data, data.included, rootGetters);
    commit("setRepositoryByProjectID", {
      projectID,
      repository: updatedRepository,
    });

    return updatedRepository;
  },

  async deleteRepositoryByProjectID(
    { dispatch, commit }: any,
    projectID: ProjectID
  ) {
    await axios.delete(`/api/project/${projectID}/repository`);
    commit("deleteRepositoryByProjectID", projectID);

    // Refetch the project as the project workflow type has been updated to "UI"
    dispatch("project/fetchProjectByID", projectID, {
      root: true,
    });
  },
};

const mutations = {
  setRepositoryListByVCSID(
    state: RepositoryState,
    {
      vcsID,
      repositoryList,
    }: {
      vcsID: VCSID;
      repositoryList: Repository[];
    }
  ) {
    state.repositoryListByVCSID.set(vcsID, repositoryList);
  },

  setRepositoryByProjectID(
    state: RepositoryState,
    {
      projectID,
      repository,
    }: {
      projectID: ProjectID;
      repository: Repository;
    }
  ) {
    state.repositoryByProjectID.set(projectID, repository);
  },

  deleteRepositoryByProjectID(state: RepositoryState, projectID: ProjectID) {
    state.repositoryByProjectID.delete(projectID);
  },
};

export default {
  namespaced: true,
  state,
  getters,
  actions,
  mutations,
};
