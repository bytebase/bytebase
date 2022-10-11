// A 'simple' tokenizer splits the source with whitespace and some special chars.
export const simpleTokenize = (source: string): string[] => {
  return source.trim().split(/(\s+|[,=<>"'()])/);
};
