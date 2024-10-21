import { defineStore } from "pinia";
import { computed, reactive } from "vue";
import { databaseServiceClient } from "@/grpcweb";
import type { Pagination } from "@/types";
import { Revision } from "@/types/proto/v1/database_service";
import { DEFAULT_PAGE_SIZE } from "./common";
import { revisionNamePrefix } from "./v1/common";

export const useRevisionStore = defineStore("revision", () => {
  const revisionMapByName = reactive(new Map<string, Revision>());

  const revisionList = computed(() => {
    return Array.from(revisionMapByName.values());
  });

  const fetchRevisionsByDatabase = async (
    database: string,
    pagination?: Pagination
  ) => {
    const resp = await databaseServiceClient.listRevisions({
      parent: database,
      pageSize: pagination?.pageSize || DEFAULT_PAGE_SIZE,
      pageToken: pagination?.pageToken,
    });
    resp.revisions.forEach((revision) => {
      revisionMapByName.set(revision.name, revision);
    });
    return resp;
  };

  const getRevisionsByDatabase = (database: string) => {
    return revisionList.value.filter((revision) =>
      revision.name.startsWith(`${database}/${revisionNamePrefix}`)
    );
  };

  const getOrFetchRevisionByName = async (name: string) => {
    if (revisionMapByName.has(name)) {
      return revisionMapByName.get(name);
    }

    const revision = await databaseServiceClient.getRevision({ name });
    revisionMapByName.set(revision.name, revision);
    return revision;
  };

  const getRevisionByName = (name: string) => {
    return revisionMapByName.get(name);
  };

  const deleteRevision = async (name: string) => {
    await databaseServiceClient.deleteRevision({ name });
    revisionMapByName.delete(name);
  };

  return {
    revisionList,
    fetchRevisionsByDatabase,
    getRevisionsByDatabase,
    getOrFetchRevisionByName,
    getRevisionByName,
    deleteRevision,
  };
});
