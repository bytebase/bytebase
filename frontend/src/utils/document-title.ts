export const DOCUMENT_TITLE_SEPARATOR = " - ";
export const DOCUMENT_TITLE_SUFFIX = "Bytebase";

export function buildDocumentTitle(...segments: string[]): string {
  const filtered = segments.filter((s) => s.length > 0);
  if (filtered.length === 0) {
    return DOCUMENT_TITLE_SUFFIX;
  }
  return (
    filtered.join(DOCUMENT_TITLE_SEPARATOR) +
    DOCUMENT_TITLE_SEPARATOR +
    DOCUMENT_TITLE_SUFFIX
  );
}

export function setDocumentTitle(...segments: string[]): void {
  document.title = buildDocumentTitle(...segments);
}
