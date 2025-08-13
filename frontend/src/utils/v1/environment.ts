export const extractEnvironmentResourceName = (name: string) => {
  const pattern = /(?:^|\/)environments\/([^/]+)(?:$|\/)/;
  const matches = name.match(pattern);
  return matches?.[1] ?? "";
};
