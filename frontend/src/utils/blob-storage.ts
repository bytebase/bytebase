import PouchDB from "pouchdb";
import PouchDBFind from "pouchdb-find";

PouchDB.plugin(PouchDBFind);

// Module-level PouchDB instance for the "bb.storage" key/value store used by
// SQL-editor blob-passing (e.g. stashing a SQL statement for the issue/plan
// creation page to pick up). The Pinia `useStorageStore` opens a parallel
// PouchDB instance against the same DB name — both share the underlying
// IndexedDB layer, so reads on one side observe writes from the other.
const db = new PouchDB<{ _id: string; value: unknown }>("bb.storage", {
  auto_compaction: true,
  revs_limit: 1,
});

export const putBlob = async <T = unknown>(
  key: string,
  value: T
): Promise<void> => {
  await db.put({ _id: key, value }, { force: true });
};

export const getBlob = async <T = unknown>(
  key: string
): Promise<T | undefined> => {
  try {
    const doc = await db.get(key);
    return doc.value as T;
  } catch {
    return undefined;
  }
};

export const removeBlob = async (key: string): Promise<void> => {
  try {
    const doc = await db.get(key);
    if (doc) {
      await db.remove(doc);
    }
  } catch {
    // not found: nothing to remove
  }
};
