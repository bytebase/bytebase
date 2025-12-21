import { pick } from "lodash-es";
import { defineStore } from "pinia";
import PouchDB from "pouchdb";
import PouchDBFind from "pouchdb-find";
import type { SQLEditorTab } from "@/types";

interface PouchDBError extends Error {
  status?: number;
}

export const EXTENDED_TAB_FIELDS = [
  "statement",
  "batchQueryContext",
  "treeState",
] as const;
type ExtendedTab = Pick<SQLEditorTab, (typeof EXTENDED_TAB_FIELDS)[number]>;
type ExtendedTabEntity = ExtendedTab & {
  _id: string;
  _rev?: string;
  project: string; // for tracing
  user: string; // for tracing
};

PouchDB.plugin(PouchDBFind);

export const useExtendedTabStore = defineStore("sqlEditorExtendedTab", () => {
  // stores
  const extendedTabDB = new PouchDB<ExtendedTabEntity>(
    "bb.sql-editor.extended-tab",
    {
      // Remove unused and old data automatically.
      auto_compaction: true,
      // Do not save and track old revisions, this is for saving storage usage.
      revs_limit: 1,
    }
  );
  const ready = Promise.all([
    extendedTabDB.createIndex({
      index: { name: "idx_tab_user", fields: ["user"] },
    }),
    extendedTabDB.createIndex({
      index: { name: "idx_tab_project", fields: ["project"] },
    }),
  ]);

  const fetchExtendedTab = async (tab: SQLEditorTab, fallback?: () => void) => {
    try {
      await ready;
      const doc = await extendedTabDB.get(tab.id);
      Object.assign(tab, {
        _rev: doc._rev,
        _user: doc.user,
        _project: doc.project,
        ...pick(doc, EXTENDED_TAB_FIELDS),
      });
    } catch (err) {
      if (fallback) {
        fallback();
      }

      if ((err as PouchDBError)?.status === 404) return;
      console.debug("[SQLEditorExtendedTabStore] fetchExtendedTab", err);
    }
  };

  const deleteExtendedTab = async (id: string) => {
    try {
      await ready;
      const doc = await extendedTabDB.get(id);
      await extendedTabDB.remove(doc);
    } catch (err) {
      console.warn("[SQLEditorExtendedTabStore] deleteExtendedTab", err);
    }
  };

  return {
    fetchExtendedTab,
    deleteExtendedTab,
  };
});
