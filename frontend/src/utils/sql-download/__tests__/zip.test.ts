// @vitest-environment node

import * as zipjs from "@zip.js/zip.js";
import { describe, expect, it } from "vitest";
import { wrapWithMultiEntryZip } from "../zip";

describe("wrapWithMultiEntryZip", () => {
  it("produces an encrypted ZIP whose entry decrypts back to original bytes", async () => {
    const inner = new TextEncoder().encode("hello world");
    const out = await wrapWithMultiEntryZip(
      [{ bytes: inner, filename: "hello.txt" }],
      "secret",
      "outer"
    );
    expect(out.filename).toBe("outer.zip");
    expect(out.mimeType).toBe("application/zip");

    const reader = new zipjs.ZipReader(new zipjs.BlobReader(out.blob), {
      password: "secret",
    });
    const entries = await reader.getEntries();
    expect(entries).toHaveLength(1);
    expect(entries[0].filename).toBe("hello.txt");
    const entry = entries[0];
    if (!("getData" in entry) || !entry.getData) {
      throw new Error("expected file entry, got directory");
    }
    const back = await entry.getData(new zipjs.Uint8ArrayWriter());
    await reader.close();
    expect(new TextDecoder().decode(back)).toBe("hello world");
  });
});
