import axios from "axios";
import { defineStore } from "pinia";
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
import { getPrincipalFromIncludedList } from "./principal";
import { useProjectStore } from "./project";
import { useVCSStore } from "./vcs";

function convert(
  repository: ResourceObject,
  includedList: ResourceObject[]
): Repository {
  const vcsId = (repository.relationships!.vcs.data as ResourceIdentifier).id;
  let vcs: VCS = unknown("VCS") as VCS;
  vcs.id = parseInt(vcsId);

  const projectId = (
    repository.relationships!.project.data as ResourceIdentifier
  ).id;
  let project: Project = unknown("PROJECT") as Project;
  project.id = parseInt(projectId);

  const vcsStore = useVCSStore();
  const projectStore = useProjectStore();
  for (const item of includedList || []) {
    if (item.type == "vcs" && item.id == vcsId) {
      vcs = vcsStore.convert(item, includedList || []);
    }
    if (item.type == "project" && item.id == projectId) {
      project = projectStore.convert(item, includedList);
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

export const useRepositoryStore = defineStore("repository", {
  state: (): RepositoryState => ({
    repositoryListByVCSId: new Map(),
    repositoryByProjectId: new Map(),
  }),
  actions: {
    getRepositoryListByVCSId(vcsId: VCSId): Repository[] {
      return this.repositoryListByVCSId.get(vcsId) || [];
    },
    getRepositoryByProjectId(projectId: ProjectId): Repository {
      return (
        this.repositoryByProjectId.get(projectId) ||
        (unknown("REPOSITORY") as Repository)
      );
    },
    setRepositoryListByVCSId({
      vcsId,
      repositoryList,
    }: {
      vcsId: VCSId;
      repositoryList: Repository[];
    }) {
      this.repositoryListByVCSId.set(vcsId, repositoryList);
    },
    setRepositoryByProjectId({
      projectId,
      repository,
    }: {
      projectId: ProjectId;
      repository: Repository;
    }) {
      this.repositoryByProjectId.set(projectId, repository);
    },
    removeRepositoryByProjectId(projectId: ProjectId) {
      this.repositoryByProjectId.delete(projectId);
    },
    async createRepository({
      projectId,
      repositoryCreate,
    }: {
      projectId: ProjectId;
      repositoryCreate: RepositoryCreate;
    }): Promise<Repository> {
      const data = (
        await axios.post(`/api/project/${projectId}/repository`, {
          data: {
            type: "RepositoryCreate",
            attributes: repositoryCreate,
          },
        })
      ).data;

      const createdRepository = convert(data.data, data.included);
      this.setRepositoryByProjectId({
        projectId: projectId,
        repository: createdRepository,
      });

      // Refetch the project as the project workflow type has been updated to "VCS"
      useProjectStore().fetchProjectById(projectId);

      return createdRepository;
    },
    async fetchRepositoryListByVCSId(vcsId: VCSId): Promise<Repository[]> {
      const data = (await axios.get(`/api/vcs/${vcsId}/repository`)).data;

      const repositoryList: Repository[] = data.data.map(
        (repository: ResourceObject) => {
          return convert(repository, data.included);
        }
      );

      this.setRepositoryListByVCSId({ vcsId, repositoryList });
      return repositoryList;
    },
    async fetchRepositoryByProjectId(
      projectId: ProjectId
    ): Promise<Repository> {
      const data = (await axios.get(`/api/project/${projectId}/repository`))
        .data;
      const repositoryList: Repository[] = data.data.map(
        (repository: ResourceObject) => {
          return convert(repository, data.included);
        }
      );

      // Expect server to return at most one item, otherwise it will throw error
      if (repositoryList.length > 0) {
        this.setRepositoryByProjectId({
          projectId,
          repository: repositoryList[0],
        });
        return repositoryList[0];
      }

      return unknown("REPOSITORY") as Repository;
    },
    async updateRepositoryByProjectId({
      projectId,
      repositoryPatch,
    }: {
      projectId: ProjectId;
      repositoryPatch: RepositoryPatch;
    }) {
      const data = (
        await axios.patch(`/api/project/${projectId}/repository`, {
          data: {
            type: "repositoryPatch",
            attributes: repositoryPatch,
          },
        })
      ).data;

      const updatedRepository = convert(data.data, data.included);

      this.setRepositoryByProjectId({
        projectId,
        repository: updatedRepository,
      });

      return updatedRepository;
    },
    async deleteRepositoryByProjectId(projectId: ProjectId) {
      await axios.delete(`/api/project/${projectId}/repository`);

      this.removeRepositoryByProjectId(projectId);

      // Refetch the project as the project workflow type has been updated to "UI"
      useProjectStore().fetchProjectById(projectId);
    },
  },
});
