// displayPermissionTitle return the formatted permission title.
// e.g., bb.databases.get -> databases.get
export const displayPermissionTitle = (permission: string): string => {
  return permission.slice(3);
};
