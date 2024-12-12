import { defineStore } from "pinia";
import PouchDB from "pouchdb";
import PouchDBFind from "pouchdb-find";

PouchDB.plugin(PouchDBFind);

// The simple storage store for saving key-value pairs.
export const useStorageStore = defineStore("storageStore", () => {
  const db = new PouchDB<{
    _id: string;
    value: any;
  }>("bb.storage", {
    // Remove unused and old data automatically.
    auto_compaction: true,
    // Do not save and track old revisions, this is for saving storage usage.
    revs_limit: 1,
  });

  const put = async (key: string, value: any) => {
    await db.put(
      {
        _id: key,
        value,
      },
      { force: true }
    );
  };
  const get = async (key: string) => {
    try {
      const doc = await db.get(key);
      return doc.value;
    } catch {
      return undefined;
    }
  };
  const remove = async (key: string) => {
    const doc = await db.get(key);
    if (doc) {
      await db.remove(doc);
    }
  };

  return {
    get,
    put,
    remove,
  };
});
