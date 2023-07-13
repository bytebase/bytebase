import { ResourceId, UNKNOWN_ID, Project, SheetId } from "@/types";

export const userNamePrefix = "users/";
export const environmentNamePrefix = "environments/";
export const projectNamePrefix = "projects/";
export const instanceNamePrefix = "instances/";
export const databaseNamePrefix = "databases/";
export const idpNamePrefix = "idps/";
export const policyNamePrefix = "policies/";
export const settingNamePrefix = "settings/";
export const sheetNamePrefix = "sheets/";
export const databaseGroupNamePrefix = "databaseGroups/";
export const schemaGroupNamePrefix = "schemaGroups/";
export const externalVersionControlPrefix = "externalVersionControls/";
export const logNamePrefix = "logs/";
export const issueNamePrefix = "issues/";
export const secretNamePrefix = "secrets/";
export const schemaDesignNamePrefix = "schemaDesigns/";

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

export const getProjectAndSheetId = (name: string): string[] => {
  const tokens = getNameParentTokens(name, [
    projectNamePrefix,
    sheetNamePrefix,
  ]);

  if (tokens.length != 2) {
    return ["", ""];
  }

  return tokens;
};

export const getProjectAndSchemaDesignSheetId = (name: string): string[] => {
  const tokens = getNameParentTokens(name, [
    projectNamePrefix,
    schemaDesignNamePrefix,
  ]);

  if (tokens.length != 2) {
    return ["", ""];
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

export const getUserEmailFromIdentifier = (identifier: string): string => {
  return identifier.replace(/^(user:|users\/)/, "");
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

export const getProjectNameAndDatabaseGroupNameAndSchemaGroupName = (
  name: string
): string[] => {
  const tokens = getNameParentTokens(name, [
    projectNamePrefix,
    databaseGroupNamePrefix,
    schemaGroupNamePrefix,
  ]);

  if (tokens.length !== 3) {
    return ["", "", ""];
  }

  return tokens;
};

export const getProjectPathByLegacyProject = (project: Project): string => {
  return `${projectNamePrefix}${project.resourceId}`;
};

export const getSheetPathByLegacyProject = (
  project: Project,
  sheetId: SheetId
): string => {
  if (sheetId === UNKNOWN_ID) {
    return "";
  }
  return `${getProjectPathByLegacyProject(
    project
  )}/${sheetNamePrefix}${sheetId}`;
};

export const getProjectPathFromRepoName = (repoName: string): string => {
  return repoName.split("/gitOpsInfo")[0];
};

export const getVCSUid = (name: string): number => {
  const tokens = getNameParentTokens(name, [externalVersionControlPrefix]);
  const vcsUid = Number(tokens[0] || UNKNOWN_ID);
  return vcsUid;
};
