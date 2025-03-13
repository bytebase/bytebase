import { pick } from "lodash-es";
import { defineStore, storeToRefs } from "pinia";
import PouchDB from "pouchdb";
import PouchDBFind from "pouchdb-find";
import { computed } from "vue";
import type { SQLEditorTab } from "@/types";
import { useCurrentUserV1 } from "../auth";
import { extractUserId } from "../v1/common";
import { useSQLEditorStore } from "./editor";

export const EXTENDED_TAB_FIELDS = [
  "statement",
  "batchQueryContext",
  "treeState",
] as const;
export type ExtendedTab = Pick<
  SQLEditorTab,
  (typeof EXTENDED_TAB_FIELDS)[number]
>;
export type ExtendedTabEntity = ExtendedTab & {
  _id: string;
  _rev?: string;
  project: string; // for tracing
  user: string; // for tracing
};

PouchDB.plugin(PouchDBFind);

export const useExtendedTabStore = defineStore("sqlEditorExtendedTab", () => {
  // context
  const { project } = storeToRefs(useSQLEditorStore());
  const me = useCurrentUserV1();
  const userUID = computed(() => extractUserId(me.value.name));

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

      if ((err as any)?.status === 404) return;
      console.debug("[SQLEditorExtendedTabStore] fetchExtendedTab", err);
    }
  };

  const saveExtendedTab = async (tab: SQLEditorTab, extended: ExtendedTab) => {
    const rev = (tab as any)._rev as string | undefined;
    const doc: ExtendedTabEntity = {
      _id: tab.id,
      project: (tab as any)._project ?? project.value,
      user: (tab as any)._user ?? userUID.value,
      ...extended,
    };
    if (rev) {
      doc._rev = rev;
    }
    try {
      await ready;
      const saved = await extendedTabDB.put(doc, { force: true });
      (tab as any)._rev = saved.rev;
    } catch (err) {
      // We only keep ONE revision for each tab so we could ignore conflict error
      if ((err as any)?.name === "conflict") return;
      console.warn("[SQLEditorExtendedTabStore] saveExtendedTab", err);
    }
  };

  const deleteExtendedTab = async (tab: SQLEditorTab) => {
    try {
      await ready;
      const doc = await extendedTabDB.get(tab.id);
      await extendedTabDB.remove(doc);
    } catch (err) {
      console.warn("[SQLEditorExtendedTabStore] deleteExtendedTab", err);
    }
  };

  const cleanupExtendedTabs = async (
    user: string,
    validTabIdList: string[]
  ) => {
    try {
      await ready;
      const response = await extendedTabDB.find({
        selector: {
          user: { $eq: user },
          _id: { $nin: validTabIdList },
        },
      });
      const docs = response.docs;
      if (docs.length === 0) {
        return;
      }
      docs.forEach((doc) => {
        (doc as any)._deleted = true;
      });
      console.debug(
        "[SQLEditorExtendedTabStore] cleanupExtendedTabs",
        `delete ${docs.length} tabs`
      );
      await extendedTabDB.bulkDocs(docs);
    } catch (err) {
      console.warn("[SQLEditorExtendedTabStore] cleanupExtendedTabs", err);
    }
  };
  return {
    fetchExtendedTab,
    saveExtendedTab,
    deleteExtendedTab,
    cleanupExtendedTabs,
  };
});
