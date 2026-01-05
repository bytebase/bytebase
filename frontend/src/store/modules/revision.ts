import { create } from "@bufbuild/protobuf";
import { defineStore } from "pinia";
import { computed, reactive } from "vue";
import { revisionServiceClientConnect } from "@/connect";
import type { Pagination } from "@/types";
import type { Revision } from "@/types/proto-es/v1/revision_service_pb";
import {
  DeleteRevisionRequestSchema,
  GetRevisionRequestSchema,
  ListRevisionsRequestSchema,
} from "@/types/proto-es/v1/revision_service_pb";
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
    const request = create(ListRevisionsRequestSchema, {
      parent: database,
      pageSize: pagination?.pageSize,
      pageToken: pagination?.pageToken,
    });
    const resp = await revisionServiceClientConnect.listRevisions(request);
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

    const request = create(GetRevisionRequestSchema, { name });
    const revision = await revisionServiceClientConnect.getRevision(request);
    revisionMapByName.set(revision.name, revision);
    return revision;
  };

  const getRevisionByName = (name: string) => {
    return revisionMapByName.get(name);
  };

  const deleteRevision = async (name: string) => {
    const request = create(DeleteRevisionRequestSchema, { name });
    await revisionServiceClientConnect.deleteRevision(request);
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
