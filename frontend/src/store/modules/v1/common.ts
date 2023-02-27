import { ResourceId, UNKNOWN_ID } from "@/types";

export const userNamePrefix = "users/";
export const environmentNamePrefix = "environments/";
export const instanceNamePrefix = "instances/";
export const idpNamePrefix = "idps/";

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

export const getEnvironmentId = (name: string): number => {
  const tokens = getNameParentTokens(name, [environmentNamePrefix]);
  const environmentId = Number(tokens[0] || UNKNOWN_ID);
  return environmentId;
};

export const getIdentityProviderResourceId = (name: string): ResourceId => {
  const tokens = getNameParentTokens(name, [idpNamePrefix]);
  return tokens[0];
};
