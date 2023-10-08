export const escapeMarkdown = (md: string): string => {
  return md.replaceAll(/[*_~\-#`[\]()\\]/g, (ch) => `\\${ch}`);
};

export const escapeFilename = (str: string): string => {
  return str.replace(/[/\\?%*:|"<>]/g, "-");
};
