export const extractReleaseUID = (name: string) => {
  const pattern = /(?:^|\/)releases\/([^/]+)(?:$|\/)/;
  const matches = name.match(pattern);
  return matches?.[1] ?? "";
};
