import { defineStore } from "pinia";
import { reactive } from "vue";
import { isEqual, isUndefined } from "lodash-es";
import {
  projectServiceClient,
  externalVersionControlServiceClient,
} from "@/grpcweb";
import { ProjectGitOpsInfo } from "@/types/proto/v1/externalvs_service";
import { ComposedRepository } from "@/types";
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
      gitopsInfo.vcsUid = repo.vcsUid;
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

  const setupSQLReviewCI = async (project: string): Promise<string> => {
    const resp = await projectServiceClient.setupProjectSQLReviewCI({
      name: project + "/gitOpsInfo",
    });
    return resp.pullRequestUrl;
  };

  const fetchRepositoryListByVCS = async (
    vcsName: string
  ): Promise<ProjectGitOpsInfo[]> => {
    const resp =
      await externalVersionControlServiceClient.listProjectGitOpsInfo({
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
    setupSQLReviewCI,
    upsertRepository,
    deleteRepository,
    getRepositoryByProject,
    getOrFetchRepositoryByProject,
    fetchRepositoryListByVCS,
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
  if (
    !isUndefined(update.filePathTemplate) &&
    !isEqual(origin.filePathTemplate, update.filePathTemplate)
  ) {
    updateMask.push("file_path_template");
  }
  if (
    !isUndefined(update.schemaPathTemplate) &&
    !isEqual(origin.schemaPathTemplate, update.schemaPathTemplate)
  ) {
    updateMask.push("schema_path_template");
  }
  if (
    !isUndefined(update.sheetPathTemplate) &&
    !isEqual(origin.sheetPathTemplate, update.sheetPathTemplate)
  ) {
    updateMask.push("sheet_path_template");
  }
  if (
    !isUndefined(update.enableSqlReviewCi) &&
    !isEqual(origin.enableSqlReviewCi, update.enableSqlReviewCi)
  ) {
    updateMask.push("enable_sql_review_ci");
  }
  return updateMask;
};
