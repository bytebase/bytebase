export const callVar = (css: string) => {
  return getComputedStyle(document.documentElement)
    .getPropertyValue(css)
    .trim();
};
