import { isEqual, isUndefined } from "lodash-es";
import { defineStore } from "pinia";
import { reactive } from "vue";
import { projectServiceClient, vcsProviderServiceClient } from "@/grpcweb";
import type { ComposedRepository } from "@/types";
import type { ProjectGitOpsInfo } from "@/types/proto/v1/vcs_provider_service";
import { getProjectPathFromRepoName } from "./common";
import { useProjectV1Store } from "./project";

export const useRepositoryV1Store = defineStore("repository_v1", () => {
  const repositoryMapByProject = reactive(new Map<string, ProjectGitOpsInfo>());
  const repositoryMapByVCS = reactive(new Map<string, ComposedRepository[]>());

  const fetchRepositoryByProject = async (
    project: string,
    silent = false
  ): Promise<ProjectGitOpsInfo | undefined> => {
    try {
      const gitopsInfo = await projectServiceClient.getProjectGitOpsInfo(
        {
          name: project + "/gitOpsInfo",
        },
        { silent }
      );

      repositoryMapByProject.set(project, gitopsInfo);
      return gitopsInfo;
    } catch (e) {
      return;
    }
  };

  const getRepositoryByProject = (
    project: string
  ): ProjectGitOpsInfo | undefined => {
    return repositoryMapByProject.get(project);
  };

  const getOrFetchRepositoryByProject = (project: string, silent = false) => {
    if (repositoryMapByProject.has(project)) {
      return Promise.resolve(repositoryMapByProject.get(project));
    }
    return fetchRepositoryByProject(project, silent);
  };

  const upsertRepository = async (
    project: string,
    gitopsInfo: Partial<ProjectGitOpsInfo>
  ): Promise<ProjectGitOpsInfo> => {
    gitopsInfo.name = project + "/gitOpsInfo";
    const repo = await getOrFetchRepositoryByProject(project);
    let gitops: ProjectGitOpsInfo;

    if (!repo) {
      gitops = await projectServiceClient.updateProjectGitOpsInfo({
        projectGitopsInfo: gitopsInfo,
        allowMissing: true,
      });
    } else {
      gitopsInfo.vcs = repo.vcs;
      const updateMask = getUpdateMaskForRepository(repo, gitopsInfo);
      if (updateMask.length === 0) {
        return repo;
      }
      gitops = await projectServiceClient.updateProjectGitOpsInfo({
        projectGitopsInfo: gitopsInfo,
        updateMask: getUpdateMaskForRepository(repo, gitopsInfo),
        allowMissing: false,
      });
    }

    repositoryMapByProject.set(project, gitops);
    return gitops;
  };

  const deleteRepository = async (project: string) => {
    await projectServiceClient.unsetProjectGitOpsInfo({
      name: project + "/gitOpsInfo",
    });
    repositoryMapByProject.delete(project);
  };

  const fetchRepositoryListByVCS = async (
    vcsName: string
  ): Promise<ProjectGitOpsInfo[]> => {
    const resp = await vcsProviderServiceClient.listProjectGitOpsInfo({
      name: vcsName,
    });

    const projectV1Store = useProjectV1Store();
    const repoList: ComposedRepository[] = await Promise.all(
      resp.projectGitopsInfo.map(async (repo) => {
        const project = await projectV1Store.getOrFetchProjectByName(
          getProjectPathFromRepoName(repo.name)
        );
        return {
          ...repo,
          project,
        };
      })
    );

    repositoryMapByVCS.set(vcsName, repoList);
    return repoList;
  };

  const getRepositoryListByVCS = (vcsName: string): ComposedRepository[] => {
    return repositoryMapByVCS.get(vcsName) || [];
  };

  return {
    upsertRepository,
    deleteRepository,
    getRepositoryByProject,
    getOrFetchRepositoryByProject,
    fetchRepositoryListByVCS,
    fetchRepositoryByProject,
    getRepositoryListByVCS,
  };
});

const getUpdateMaskForRepository = (
  origin: ProjectGitOpsInfo,
  update: Partial<ProjectGitOpsInfo>
): string[] => {
  const updateMask: string[] = [];
  if (!isUndefined(update.title) && !isEqual(origin.title, update.title)) {
    updateMask.push("title");
  }
  if (
    !isUndefined(update.branchFilter) &&
    !isEqual(origin.branchFilter, update.branchFilter)
  ) {
    updateMask.push("branch_filter");
  }
  if (
    !isUndefined(update.baseDirectory) &&
    !isEqual(origin.baseDirectory, update.baseDirectory)
  ) {
    updateMask.push("base_directory");
  }
  return updateMask;
};
