import axios from "axios";
import {
  Project,
  ProjectId,
  Repository,
  RepositoryCreate,
  RepositoryPatch,
  RepositoryState,
  ResourceIdentifier,
  ResourceObject,
  unknown,
  VCS,
  VCSId,
} from "../../types";
import { getPrincipalFromIncludedList } from "../pinia";
import { useVCSStore, useProjectStore } from "@/store";

function convert(
  repository: ResourceObject,
  includedList: ResourceObject[],
  rootGetters: any
): Repository {
  const vcsId = (repository.relationships!.vcs.data as ResourceIdentifier).id;
  let vcs: VCS = unknown("VCS") as VCS;
  vcs.id = parseInt(vcsId);

  const projectId = (
    repository.relationships!.project.data as ResourceIdentifier
  ).id;
  let project: Project = unknown("PROJECT") as Project;
  project.id = parseInt(projectId);

  for (const item of includedList || []) {
    if (item.type == "vcs" && item.id == vcsId) {
      vcs = useVCSStore().convert(item, includedList || []);
    }
    if (item.type == "project" && item.id == projectId) {
      project = useProjectStore().convert(item, includedList);
    }
  }

  return {
    ...(repository.attributes as Omit<
      Repository,
      "id" | "vcs" | "project" | "creator" | "updater"
    >),
    id: parseInt(repository.id),
    creator: getPrincipalFromIncludedList(
      repository.relationships!.creator.data,
      includedList
    ),
    updater: getPrincipalFromIncludedList(
      repository.relationships!.updater.data,
      includedList
    ),
    vcs,
    project,
  };
}

const state: () => RepositoryState = () => ({
  repositoryListByVCSId: new Map(),
  repositoryByProjectId: new Map(),
});

const getters = {
  repositoryListByVCSId:
    (state: RepositoryState) =>
    (vcsId: VCSId): Repository[] => {
      return state.repositoryListByVCSId.get(vcsId) || [];
    },

  repositoryByProjectId:
    (state: RepositoryState) =>
    (projectId: ProjectId): Repository => {
      return (
        state.repositoryByProjectId.get(projectId) ||
        (unknown("REPOSITORY") as Repository)
      );
    },
};

const actions = {
  async createRepository(
    { dispatch, commit, rootGetters }: any,
    {
      projectId,
      repositoryCreate,
    }: { projectId: ProjectId; repositoryCreate: RepositoryCreate }
  ): Promise<Repository> {
    const data = (
      await axios.post(`/api/project/${projectId}/repository`, {
        data: {
          type: "RepositoryCreate",
          attributes: repositoryCreate,
        },
      })
    ).data;

    const createdRepository = convert(data.data, data.included, rootGetters);
    commit("setRepositoryByProjectId", {
      projectId: projectId,
      repository: createdRepository,
    });

    // Refetch the project as the project workflow type has been updated to "VCS"
    useProjectStore().fetchProjectById(projectId);

    return createdRepository;
  },

  async fetchRepositoryListByVCSId(
    { commit, rootGetters }: any,
    vcsId: VCSId
  ): Promise<Repository[]> {
    const data = (await axios.get(`/api/vcs/${vcsId}/repository`)).data;

    const repositoryList = data.data.map((repository: ResourceObject) => {
      return convert(repository, data.included, rootGetters);
    });

    commit("setRepositoryListByVCSId", { vcsId, repositoryList });
    return repositoryList;
  },

  async fetchRepositoryByProjectId(
    { commit, rootGetters }: any,
    projectId: ProjectId
  ): Promise<Repository> {
    const data = (await axios.get(`/api/project/${projectId}/repository`)).data;
    const repositoryList = data.data.map((repository: ResourceObject) => {
      return convert(repository, data.included, rootGetters);
    });

    // Expect server to return at most one item, otherwise it will throw error
    if (repositoryList.length > 0) {
      commit("setRepositoryByProjectId", {
        projectId,
        repository: repositoryList[0],
      });
      return repositoryList[0];
    }

    return unknown("REPOSITORY") as Repository;
  },

  async updateRepositoryByProjectId(
    { commit, rootGetters }: any,
    {
      projectId,
      repositoryPatch,
    }: {
      projectId: ProjectId;
      repositoryPatch: RepositoryPatch;
    }
  ) {
    const data = (
      await axios.patch(`/api/project/${projectId}/repository`, {
        data: {
          type: "repositoryPatch",
          attributes: repositoryPatch,
        },
      })
    ).data;

    const updatedRepository = convert(data.data, data.included, rootGetters);
    commit("setRepositoryByProjectId", {
      projectId,
      repository: updatedRepository,
    });

    return updatedRepository;
  },

  async deleteRepositoryByProjectId(
    { dispatch, commit }: any,
    projectId: ProjectId
  ) {
    await axios.delete(`/api/project/${projectId}/repository`);
    commit("deleteRepositoryByProjectId", projectId);

    // Refetch the project as the project workflow type has been updated to "UI"
    useProjectStore().fetchProjectById(projectId);
  },
};

const mutations = {
  setRepositoryListByVCSId(
    state: RepositoryState,
    {
      vcsId,
      repositoryList,
    }: {
      vcsId: VCSId;
      repositoryList: Repository[];
    }
  ) {
    state.repositoryListByVCSId.set(vcsId, repositoryList);
  },

  setRepositoryByProjectId(
    state: RepositoryState,
    {
      projectId,
      repository,
    }: {
      projectId: ProjectId;
      repository: Repository;
    }
  ) {
    state.repositoryByProjectId.set(projectId, repository);
  },

  deleteRepositoryByProjectId(state: RepositoryState, projectId: ProjectId) {
    state.repositoryByProjectId.delete(projectId);
  },
};

export default {
  namespaced: true,
  state,
  getters,
  actions,
  mutations,
};
