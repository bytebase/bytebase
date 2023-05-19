export const extractUserUID = (name: string) => {
  // They are using the same format so we can simply call the method.
  return extractUserResourceName(name);
};

/**
 * @param name Format: users/{email}
 */
export const extractUserResourceName = (name: string) => {
  const pattern = /(?:^|\/)users\/([^/]+)(?:$|\/)/;
  const matches = name.match(pattern);
  return matches?.[1] ?? "";
};
