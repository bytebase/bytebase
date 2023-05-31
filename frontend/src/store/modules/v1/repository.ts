import { defineStore } from "pinia";
import { isEqual, isUndefined } from "lodash-es";
import { projectServiceClient } from "@/grpcweb";
import { ProjectGitOpsInfo } from "@/types/proto/v1/externalvs_service";

interface RepositoryState {
  repositoryByProject: Map<string, ProjectGitOpsInfo>;
}

export const useRepositoryV1Store = defineStore("repository_v1", () => {
  const repositoryState: RepositoryState = {
    repositoryByProject: new Map<string, ProjectGitOpsInfo>(),
  };

  const fetchRepositoryByProject = async (
    project: string
  ): Promise<ProjectGitOpsInfo> => {
    const gitopsInfo = await projectServiceClient.getProjectGitOpsInfo({
      project,
    });

    repositoryState.repositoryByProject.set(project, gitopsInfo);
    return gitopsInfo;
  };

  const getRepositoryByProject = (
    project: string
  ): ProjectGitOpsInfo | undefined => {
    return repositoryState.repositoryByProject.get(project);
  };

  const getOrFetchRepositoryByProject = (project: string) => {
    if (repositoryState.repositoryByProject.has(project)) {
      return repositoryState.repositoryByProject.get(project);
    }
    return fetchRepositoryByProject(project);
  };

  const upsertRepository = async (
    project: string,
    gitopsInfo: Partial<ProjectGitOpsInfo>
  ): Promise<ProjectGitOpsInfo> => {
    const repo = await getOrFetchRepositoryByProject(project);
    let gitops: ProjectGitOpsInfo;

    if (!repo) {
      gitops = await projectServiceClient.setProjectGitOpsInfo({
        project,
        projectGitopsInfo: gitopsInfo,
        allowMissing: true,
      });
    } else {
      const updateMask = getUpdateMaskForRepository(repo, gitopsInfo);
      if (updateMask.length === 0) {
        return repo;
      }
      gitops = await projectServiceClient.setProjectGitOpsInfo({
        project,
        projectGitopsInfo: gitopsInfo,
        updateMask: getUpdateMaskForRepository(repo, gitopsInfo),
        allowMissing: false,
      });
    }

    repositoryState.repositoryByProject.set(project, gitops);
    return gitops;
  };

  const deleteRepository = async (project: string) => {
    await projectServiceClient.deleteProjectGitOpsInfo({
      project,
    });
  };

  const setupSQLReviewCI = async (project: string): Promise<string> => {
    const resp = await projectServiceClient.setupProjectSQLReviewCI({
      project,
    });
    return resp.pullRequestUrl;
  };

  return {
    setupSQLReviewCI,
    upsertRepository,
    deleteRepository,
    getRepositoryByProject,
    getOrFetchRepositoryByProject,
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
