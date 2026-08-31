import { afterEach, describe, expect, test, vi } from "vitest";
import { cleanupLegacyPouchDatabases } from "./migration";

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("legacy PouchDB cleanup", () => {
  test("continues when deleting a database throws synchronously", async () => {
    const deleteDatabase = vi.fn((name: string) => {
      if (name === "_pouch_bb.plugin.ai.messages") {
        throw new DOMException("IndexedDB unavailable", "InvalidStateError");
      }
      const request = {} as IDBOpenDBRequest;
      queueMicrotask(() => request.onsuccess?.(new Event("success")));
      return request;
    });
    vi.stubGlobal("indexedDB", { deleteDatabase });

    await expect(cleanupLegacyPouchDatabases()).resolves.toBeUndefined();
    expect(deleteDatabase).toHaveBeenCalledTimes(4);
  });
});
