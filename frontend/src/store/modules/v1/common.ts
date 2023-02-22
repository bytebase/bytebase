export const userNamePrefix = "users/";
export const environmentNamePrefix = "environments/";

export const getNameParentTokens = (
  name: string,
  tokenPrefixes: string[]
): string[] => {
  const parts = name.split("/");
  if (parts.length !== 2 * tokenPrefixes.length) {
    return [];
  }

  const tokens: string[] = [];
  for (let i = 0; i < tokenPrefixes.length; i++) {
    if (parts[2 * i] !== tokenPrefixes[i]) {
      return [];
    }
    if (parts[2 * i + 1] === "") {
      return [];
    }
    tokens.push(parts[2 * i + 1]);
  }
  return tokens;
};

export const getUserId = (name: string): number => {
  const tokens = getNameParentTokens(name, [userNamePrefix]);
  const userId = Number(tokens[0]);
  return userId;
};

export const getEnvironmentId = (name: string): number => {
  const tokens = getNameParentTokens(name, [environmentNamePrefix]);
  const environmentId = Number(tokens[0]);
  return environmentId;
};
