import { defineStore } from "pinia";
import axios from "axios";
import {
  Project,
  Repository,
  RepositoryState,
  ResourceIdentifier,
  ResourceObject,
  unknown,
  VCS,
  VCSId,
} from "@/types";
import { useLegacyProjectStore } from "./project";
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
  const projectStore = useLegacyProjectStore();
  for (const item of includedList || []) {
    if (item.type == "vcs" && item.id == vcsId) {
      vcs = vcsStore.convert(item, includedList || []);
    }
    if (item.type == "project" && item.id == projectId) {
      project = projectStore.convert(item, includedList);
    }
  }

  return {
    ...(repository.attributes as Omit<Repository, "id" | "vcs" | "project">),
    id: repository.id,
    vcs,
    project,
  };
}

export const useRepositoryStore = defineStore("repository", {
  state: (): RepositoryState => ({
    repositoryListByVCSId: new Map(),
  }),
  actions: {
    getRepositoryListByVCSId(vcsId: VCSId): Repository[] {
      return this.repositoryListByVCSId.get(vcsId) || [];
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
  },
});
