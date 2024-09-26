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

  const getRevisionByName = (name: string) => {
    return revisionMapByName.get(name) ?? Revision.fromPartial({});
  };

  return {
    revisionList,
    fetchRevisionsByDatabase,
    getRevisionsByDatabase,
    getRevisionByName,
  };
});
