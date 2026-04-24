export const workspaceNamePrefix = "workspaces/";
export const userNamePrefix = "users/";
export const projectNamePrefix = "projects/";
export const settingNamePrefix = "settings/";

export function getResourceId(name: string, prefix: string): string {
  if (!name.startsWith(prefix)) {
    return "";
  }
  return name.slice(prefix.length);
}

export function getProjectName(name: string): string {
  return getResourceId(name, projectNamePrefix);
}

export function getUserName(email: string): string {
  return `${userNamePrefix}${email}`;
}

export function isValidProjectName(name: string | undefined): name is string {
  return Boolean(name?.startsWith(projectNamePrefix));
}
