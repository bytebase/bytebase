export const extractUserUID = (name: string) => {
  const pattern = /(?:^|\/)users\/([^/]+)(?:$|\/)/;
  const matches = name.match(pattern);
  return matches?.[1] ?? "";
};
