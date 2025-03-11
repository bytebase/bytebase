import type { ResourceId } from "@/types";
import { UNKNOWN_ID } from "@/types";

export const workspaceNamePrefix = "workspaces/";
export const userNamePrefix = "users/";
export const roleNamePrefix = "roles/";
export const environmentNamePrefix = "environments/";
export const projectNamePrefix = "projects/";
export const instanceNamePrefix = "instances/";
export const databaseNamePrefix = "databases/";
export const idpNamePrefix = "idps/";
export const policyNamePrefix = "policies/";
export const settingNamePrefix = "settings/";
export const sheetNamePrefix = "sheets/";
export const worksheetNamePrefix = "worksheets/";
export const databaseGroupNamePrefix = "databaseGroups/";
export const logNamePrefix = "logs/";
export const issueNamePrefix = "issues/";
export const secretNamePrefix = "secrets/";
export const ssoNamePrefix = "idps/";
export const issueCommentNamePrefix = "issueComments/";
export const groupNamePrefix = "groups/";
export const reviewConfigNamePrefix = "reviewConfigs/";
export const planNamePrefix = "plans/";
export const planCheckRunPrefix = "planCheckRuns/";
export const rolloutNamePrefix = "rollouts/";
export const stageNamePrefix = "stages/";
export const taskNamePrefix = "tasks/";
export const releaseNamePrefix = "releases/";
export const revisionNamePrefix = "revisions/";

export const getNameParentTokens = (
  name: string,
  tokenPrefixes: string[]
): string[] => {
  const parts = name.split("/");
  if (parts.length !== tokenPrefixes.length * 2) {
    return [];
  }

  const tokens: string[] = [];
  for (let i = 0; i < tokenPrefixes.length; i++) {
    if (parts[i * 2] + "/" !== tokenPrefixes[i]) {
      return [];
    }
    if (parts[i * 2 + 1] === "") {
      return [];
    }
    tokens.push(parts[i * 2 + 1]);
  }
  return tokens;
};

export const getUserId = (name: string): number => {
  const tokens = getNameParentTokens(name, [userNamePrefix]);
  const userId = Number(tokens[0] || UNKNOWN_ID);
  return userId;
};

export const getNumberId = (name: string, prefix: string): number => {
  const tokens = getNameParentTokens(name, [prefix]);
  return Number(tokens[0] || UNKNOWN_ID);
};

export const getLogId = (name: string): number => {
  return getNumberId(name, logNamePrefix);
};

export const getIssueId = (name: string): number => {
  return getNumberId(name, issueNamePrefix);
};

export const getProjectName = (name: string): string => {
  const tokens = getNameParentTokens(name, [projectNamePrefix]);
  const projectId = tokens[0];
  return projectId;
};

export const getProjectNamePlanIdPlanCheckRunId = (name: string): string[] => {
  const tokens = getNameParentTokens(name, [
    projectNamePrefix,
    planNamePrefix,
    planCheckRunPrefix,
  ]);
  return [tokens[0], tokens[1], tokens[2]];
};

export const getProjectIdRolloutUidStageUidTaskUid = (
  name: string
): string[] => {
  const tokens = getNameParentTokens(name, [
    projectNamePrefix,
    rolloutNamePrefix,
    stageNamePrefix,
    taskNamePrefix,
  ]);
  return [tokens[0], tokens[1], tokens[2], tokens[3]];
};

export const getWorksheetId = (name: string): string => {
  const tokens = getNameParentTokens(name, [worksheetNamePrefix]);
  return tokens[0];
};

export const getInstanceId = (name: string): string[] => {
  const tokens = getNameParentTokens(name, [instanceNamePrefix]);
  if (tokens.length !== 1) {
    return [""];
  }
  return tokens;
};

export const getInstanceAndDatabaseId = (name: string): string[] => {
  const tokens = getNameParentTokens(name, [
    instanceNamePrefix,
    databaseNamePrefix,
  ]);

  if (tokens.length != 2) {
    return ["", ""];
  }

  return tokens;
};

export const extractUserId = (identifier: string) => {
  const matches = identifier.match(/^(?:user:|users\/)(.+)$/);
  return matches?.[1] ?? identifier;
};

export const getIdentityProviderResourceId = (name: string): ResourceId => {
  const tokens = getNameParentTokens(name, [idpNamePrefix]);
  return tokens[0];
};

export const getProjectNameAndDatabaseGroupName = (name: string): string[] => {
  const tokens = getNameParentTokens(name, [
    projectNamePrefix,
    databaseGroupNamePrefix,
  ]);

  if (tokens.length !== 2) {
    return ["", ""];
  }

  return tokens;
};

export const getSSOId = (name: string) => {
  const tokens = getNameParentTokens(name, [ssoNamePrefix]);
  return tokens[0];
};

export const getProjectIdIssueIdIssueCommentId = (name: string) => {
  const tokens = getNameParentTokens(name, [
    projectNamePrefix,
    issueNamePrefix,
    issueCommentNamePrefix,
  ]);
  if (tokens.length !== 3) {
    return { projectId: "", issueId: "", issueCommentId: "" };
  }
  return {
    projectId: tokens[0],
    issueId: tokens[1],
    issueCommentId: tokens[2],
  };
};

export const getReviewConfigId = (name: string) => {
  const tokens = getNameParentTokens(name, [reviewConfigNamePrefix]);
  return tokens[0];
};

// The name of the policy.
// Format: {resource name}/policies/{policy type}
// Workspace resource name: "".
// Environment resource name: environments/environment-id.
// Instance resource name: instances/instance-id.
// Database resource name: instances/instance-id/databases/database-name.
export const getPolicyResourceNameAndType = (name: string): string[] => {
  const regex = new RegExp(`^(.*)/${policyNamePrefix}(.*)$`);
  const match = name.match(regex);
  if (!match || match.length !== 3) {
    return ["", ""];
  }
  return [match[1], match[2]];
};

export const getInstanceIdPolicyId = (name: string): string[] => {
  const tokens = getNameParentTokens(name, [
    instanceNamePrefix,
    policyNamePrefix,
  ]);
  if (tokens.length !== 2) {
    return ["", ""];
  }
  return tokens;
};

export const getProjectNameReleaseId = (name: string): string[] => {
  const tokens = getNameParentTokens(name, [
    projectNamePrefix,
    releaseNamePrefix,
  ]);
  if (tokens.length !== 2) {
    return ["", ""];
  }
  return tokens;
};

export const getProjectNameRolloutId = (name: string): string[] => {
  const tokens = getNameParentTokens(name, [
    projectNamePrefix,
    rolloutNamePrefix,
  ]);
  if (tokens.length !== 2) {
    return ["", ""];
  }
  return tokens;
};

export const isDatabaseName = (name: string): boolean => {
  const regex = /^instances\/([^/]+)\/databases\/(.+)$/;
  return regex.test(name);
};
