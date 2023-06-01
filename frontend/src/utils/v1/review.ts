export const extractReviewId = (name: string) => {
  const pattern = /(?:^|\/)reviews\/(\d+)(?:$|\/)/;
  const matches = name.match(pattern);
  return matches?.[1] ?? "";
};
