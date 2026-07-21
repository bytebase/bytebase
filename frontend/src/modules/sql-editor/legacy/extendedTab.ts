import { pick } from "lodash-es";
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

// Pinia-free: a stateless wrapper over a PouchDB instance used only by the
// one-time legacy SQL-editor cache migration (`./migration`). The PouchDB
// instance is created lazily on first use (the former `useExtendedTabStore`
// Pinia store deferred creation to first access the same way) so importing
// this module does not eagerly touch IndexedDB.
let db: PouchDB.Database<ExtendedTabEntity> | null = null;
let ready: Promise<unknown> | null = null;

const getDB = () => {
  if (!db) {
    db = new PouchDB<ExtendedTabEntity>("bb.sql-editor.extended-tab", {
      // Remove unused and old data automatically.
      auto_compaction: true,
      // Do not save and track old revisions, this is for saving storage usage.
      revs_limit: 1,
    });
    ready = Promise.all([
      db.createIndex({ index: { name: "idx_tab_user", fields: ["user"] } }),
      db.createIndex({
        index: { name: "idx_tab_project", fields: ["project"] },
      }),
    ]);
  }
  return { db, ready: ready as Promise<unknown> };
};

export const fetchExtendedTab = async (
  tab: SQLEditorTab,
  fallback?: () => void
) => {
  try {
    const { db, ready } = getDB();
    await ready;
    const doc = await db.get(tab.id);
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
    console.debug("[SQLEditorExtendedTab] fetchExtendedTab", err);
  }
};

export const deleteExtendedTab = async (id: string) => {
  try {
    const { db, ready } = getDB();
    await ready;
    const doc = await db.get(id);
    await db.remove(doc);
  } catch (err) {
    console.warn("[SQLEditorExtendedTab] deleteExtendedTab", err);
  }
};
