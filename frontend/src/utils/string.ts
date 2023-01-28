export const escapeMarkdown = (md: string): string => {
  return md.replaceAll(/[*_~\-#`[\]()\\]/g, (ch) => `\\${ch}`);
};
