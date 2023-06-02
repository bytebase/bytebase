export const extractBackupResourceName = (name: string) => {
  const pattern = /(?:^|\/)backups\/([^/]+)(?:$|\/)/;
  const matches = name.match(pattern);
  return matches?.[1] ?? "";
};
