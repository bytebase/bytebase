import { defineStore } from "pinia";
import { computed, reactive } from "vue";
import { create } from "@bufbuild/protobuf";
import { revisionServiceClientConnect } from "@/grpcweb";
import type { Pagination } from "@/types";
import { Revision } from "@/types/proto/v1/revision_service";
import { convertNewRevisionToOld } from "@/utils/v1/revision-conversions";
import {
  ListRevisionsRequestSchema,
  GetRevisionRequestSchema,
  DeleteRevisionRequestSchema,
} from "@/types/proto-es/v1/revision_service_pb";
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
    const request = create(ListRevisionsRequestSchema, {
      parent: database,
      pageSize: pagination?.pageSize || DEFAULT_PAGE_SIZE,
      pageToken: pagination?.pageToken,
    });
    const resp = await revisionServiceClientConnect.listRevisions(request);
    resp.revisions.forEach((revision) => {
      const oldRevision = convertNewRevisionToOld(revision);
      revisionMapByName.set(oldRevision.name, oldRevision);
    });
    return {
      ...resp,
      revisions: resp.revisions.map(convertNewRevisionToOld),
    };
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
    const oldRevision = convertNewRevisionToOld(revision);
    revisionMapByName.set(oldRevision.name, oldRevision);
    return oldRevision;
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
