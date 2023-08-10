export const extractDatabaseGroupName = (name: string) => {
  const pattern = /(?:^|\/)databaseGroups\/([^/]+)(?:$|\/)/;
  const matches = name.match(pattern);
  return matches?.[1] ?? "";
};
