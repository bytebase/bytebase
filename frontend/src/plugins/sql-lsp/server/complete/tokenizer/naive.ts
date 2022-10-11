// A 'naive' tokenizer splits the source with whitespace and some special chars.
export const naiveTokenize = (source: string): string[] => {
  return source.trim().split(/(\s+|[,=<>"'()])/);
};
