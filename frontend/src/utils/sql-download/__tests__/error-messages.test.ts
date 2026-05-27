import { describe, expect, it } from "vitest";
import { downloadErrorMessage } from "../error-messages";

const t = (key: string) => key;

// Helper: realistic DownloadError shape — an Error instance with a .code
// field. Matches what `downloadError()` from types.ts produces. Plain
// objects without `instanceof Error` fall through to the generic path
// because `isDownloadError` requires both.
const makeDownloadError = (
  code: string,
  message = ""
): Error & { code: string } => Object.assign(new Error(message), { code });

describe("downloadErrorMessage", () => {
  it("maps ResultTooLarge (no message → locale fallback)", () => {
    expect(downloadErrorMessage(makeDownloadError("ResultTooLarge"), t)).toBe(
      "sql-editor.download-too-large"
    );
  });
  it("maps UnsupportedFormat (no message → locale fallback)", () => {
    expect(
      downloadErrorMessage(makeDownloadError("UnsupportedFormat"), t)
    ).toBe("sql-editor.sql-download-unavailable");
  });
  it("falls back to error message", () => {
    expect(downloadErrorMessage(new Error("oops"), t)).toBe("oops");
  });
  it("truncates multi-line errors to the first line", () => {
    expect(
      downloadErrorMessage(new Error("oops\ncause: something\n  more"), t)
    ).toBe("oops");
  });
  it("truncates excessively long messages", () => {
    const long = "x".repeat(500);
    const out = downloadErrorMessage(new Error(long), t);
    expect(out.length).toBeLessThanOrEqual(240);
    expect(out.endsWith("…")).toBe(true);
  });
  it("ResultTooLarge surfaces the bytes/cells message from the error", () => {
    const e = makeDownloadError(
      "ResultTooLarge",
      "Result is ~520 MB; client-side limit is 200 MB."
    );
    expect(downloadErrorMessage(e, t)).toContain("520 MB");
  });
  it("UnsupportedFormat surfaces the concrete message (e.g. unsupported SQL engine)", () => {
    // serializeSQL throws UnsupportedFormat with the engine name when the
    // engine isn't in SQL_ENGINE_QUOTES; falling back to the generic
    // "sql-download-unavailable" locale would lose that detail.
    const e = makeDownloadError(
      "UnsupportedFormat",
      "SQL download is not supported for engine MONGODB"
    );
    expect(downloadErrorMessage(e, t)).toContain("MONGODB");
  });
  it("UnsupportedFormat falls back to locale string when no message", () => {
    expect(
      downloadErrorMessage(makeDownloadError("UnsupportedFormat"), t)
    ).toBe("sql-editor.sql-download-unavailable");
  });
  it("falls back to a localized generic when no message", () => {
    // Plain object (not an Error instance) — exercises the non-DownloadError
    // branch.
    expect(downloadErrorMessage({}, t)).toBe("sql-editor.download-failed");
  });
  it("falls back to message extraction for non-DownloadError Error instances", () => {
    // A vanilla Error (no .code field) — isDownloadError returns false,
    // so the generic message branch handles it.
    expect(downloadErrorMessage(new Error("plain"), t)).toBe("plain");
  });
});
