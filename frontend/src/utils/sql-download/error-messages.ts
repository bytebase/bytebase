import { isDownloadError } from "./types";

// Accept any translation function that maps a key to a string; both
// Vue i18n's ComposerTranslation and react-i18next's TFunction satisfy this.
type AnyTranslate = (key: string) => string;

export function downloadErrorMessage(error: unknown, t: AnyTranslate): string {
  // ResultTooLarge and UnsupportedFormat carry actionable detail in their
  // own messages (cell/byte counts, the offending engine name, the XLSX
  // row/column overflow numbers). Surface that verbatim and only fall back
  // to the generic locale string when there is no message at all —
  // otherwise a "XLSX cannot exceed 1,048,575 data rows" message would be
  // masked by the misleading "SQL download is unavailable" locale string.
  if (isDownloadError(error)) {
    if (error.code === "ResultTooLarge") {
      return error.message || t("sql-editor.download-too-large");
    }
    if (error.code === "UnsupportedFormat") {
      return error.message || t("sql-editor.sql-download-unavailable");
    }
    if (error.code === "EncryptionFailed") {
      return t("sql-editor.download-encryption-failed");
    }
    // SerializationFailed and any future code fall through to the generic
    // message-extraction path below.
  }
  // Truncate multi-line errors (zip.js / exceljs throw deep cause chains).
  // Cap the displayed message at 240 chars INCLUDING the trailing ellipsis.
  const message = (error as Error | null)?.message;
  if (message && message.length > 0) {
    const firstLine = message.split("\n", 1)[0];
    return firstLine.length > 240 ? firstLine.slice(0, 239) + "…" : firstLine;
  }
  return t("sql-editor.download-failed");
}
