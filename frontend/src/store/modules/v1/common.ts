import {
  ResourceId,
  UNKNOWN_ID,
  Instance,
  Database,
  Project,
  Environment,
  SheetId,
} from "@/types";

export const userNamePrefix = "users/";
export const environmentNamePrefix = "environments/";
export const projectNamePrefix = "projects/";
export const instanceNamePrefix = "instances/";
export const databaseNamePrefix = "databases/";
export const idpNamePrefix = "idps/";
export const policyNamePrefix = "policies/";
export const settingNamePrefix = "settings/";
export const sheetNamePrefix = "sheets/";

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

export const getEnvironmentPathByLegacyEnvironment = (
  env: Environment
): string => {
  return `${environmentNamePrefix}${env.resourceId}`;
};

export const getInstancePathByLegacyInstance = (instance: Instance): string => {
  return `${instanceNamePrefix}${instance.resourceId}`;
};

export const getDatabasePathByLegacyDatabase = (database: Database): string => {
  return `${getInstancePathByLegacyInstance(
    database.instance
  )}/${databaseNamePrefix}${database.name}`;
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
