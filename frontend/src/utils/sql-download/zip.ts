import type { DownloadOutput } from "./types";
import { downloadError } from "./types";

// zipjs.configure mutates a process-global; hoist to a single lazy init.
let zipjsConfigured = false;
async function loadZipjs(): Promise<typeof import("@zip.js/zip.js")> {
  const zipjs = await import("@zip.js/zip.js");
  if (!zipjsConfigured) {
    zipjs.configure({ useWebWorkers: false });
    zipjsConfigured = true;
  }
  return zipjs;
}

/**
 * Wrap entries in a single ZIP. When `password` is set, every entry is
 * encrypted with WinZip AES-256 (zip.js default), matching the backend's
 * alexmullins/zip output: extra-field 0x9901, vendor "AE", strength=3,
 * actual-method=deflate.
 *
 * The dynamic import is inside try/catch so a chunk-load failure (slow
 * network, blocked CDN, stale service worker) maps to EncryptionFailed
 * rather than an unrecognized rejection.
 */
export async function wrapWithMultiEntryZip(
  entries: Array<{ bytes: Uint8Array; filename: string }>,
  password: string | undefined,
  outerBaseFilename: string
): Promise<DownloadOutput> {
  try {
    const zipjs = await loadZipjs();
    const writer = new zipjs.ZipWriter(
      new zipjs.BlobWriter("application/zip"),
      {
        ...(password ? { password } : {}),
        keepOrder: true,
      }
    );
    for (const entry of entries) {
      await writer.add(entry.filename, new zipjs.Uint8ArrayReader(entry.bytes));
    }
    const blob = await writer.close();
    return {
      blob,
      filename: `${outerBaseFilename}.zip`,
      mimeType: "application/zip",
    };
  } catch (e) {
    // EncryptionFailed only when a password was actually involved. Other
    // failure modes (dynamic-import failure, duplicate filename rejection,
    // OOM) under an unencrypted ZIP shouldn't surface as encryption errors.
    throw downloadError(
      password ? "EncryptionFailed" : "SerializationFailed",
      "Failed to build download ZIP",
      e
    );
  }
}
