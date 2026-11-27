// @vitest-environment node
import { afterEach, describe, expect, test, vi } from "vitest";
import { cleanupLegacyPouchDatabases } from "./migration";

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("legacy PouchDB cleanup", () => {
  test("preserves AI conversation and message databases for explicit recovery", async () => {
    const deleteDatabase = vi.fn((_name: string) => {
      const request = {} as IDBOpenDBRequest;
      queueMicrotask(() => request.onsuccess?.(new Event("success")));
      return request;
    });
    vi.stubGlobal("indexedDB", { deleteDatabase });

    await cleanupLegacyPouchDatabases();

    expect(deleteDatabase.mock.calls.map(([name]) => name)).toEqual([
      "_pouch_bb.sql-editor.extended-tab",
      "_pouch_bb.storage",
    ]);
  });

  test("continues when deleting a database throws synchronously", async () => {
    const deleteDatabase = vi.fn((name: string) => {
      if (name === "_pouch_bb.sql-editor.extended-tab") {
        throw new DOMException("IndexedDB unavailable", "InvalidStateError");
      }
      const request = {} as IDBOpenDBRequest;
      queueMicrotask(() => request.onsuccess?.(new Event("success")));
      return request;
    });
    vi.stubGlobal("indexedDB", { deleteDatabase });

    await expect(cleanupLegacyPouchDatabases()).resolves.toBeUndefined();
    expect(deleteDatabase).toHaveBeenCalledTimes(2);
  });
});
