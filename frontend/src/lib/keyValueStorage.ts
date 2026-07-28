import PouchDB from "pouchdb";
import PouchDBFind from "pouchdb-find";

PouchDB.plugin(PouchDBFind);

// Simple key-value storage over PouchDB, relocated from the legacy Pinia
// `useStorageStore`. Stateless module-level singleton — no reactive state, so
// it doesn't belong on the app store (and its get/put would shadow Zustand's
// own `get`).
const db = new PouchDB<{ _id: string; value: unknown }>("bb.storage", {
  // Remove unused and old data automatically.
  auto_compaction: true,
  // Do not save and track old revisions, this is for saving storage usage.
  revs_limit: 1,
});

export const keyValueStorage = {
  async put<T = unknown>(key: string, value: T): Promise<void> {
    await db.put({ _id: key, value }, { force: true });
  },
  async get<T = unknown>(key: string): Promise<T | undefined> {
    try {
      const doc = await db.get(key);
      return doc.value as T;
    } catch {
      // Data not found or error occurred.
      return undefined;
    }
  },
  async remove(key: string): Promise<void> {
    const doc = await db.get(key);
    if (doc) {
      await db.remove(doc);
    }
  },
};
