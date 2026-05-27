import { isDownloadError } from "./types";

// Accept any translation function that maps a key (plus optional interpolation
// params) to a string; both Vue i18n's ComposerTranslation and react-i18next's
// TFunction satisfy this. Params are passed straight through to whichever
// engine — naming conventions (`{name}` for Vue, `{{name}}` for react-i18next)
// are handled by the engine itself when the matching locale file is loaded.
type AnyTranslate = (
  key: string,
  params?: Record<string, string | number>
) => string;

export function downloadErrorMessage(error: unknown, t: AnyTranslate): string {
  if (isDownloadError(error)) {
    // If the throw site attached an i18n hint, prefer it: this is the only
    // path that produces translated copy for ResultTooLarge / UnsupportedFormat
    // (those error.message strings carry numeric detail that the bare locale
    // fallback can't reproduce, but the hint key/params let us reconstruct
    // the same sentence localized).
    if (error.i18n) {
      return t(error.i18n.key, error.i18n.params);
    }
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
