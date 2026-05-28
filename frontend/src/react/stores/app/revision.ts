import { create as createProto } from "@bufbuild/protobuf";
import { revisionServiceClientConnect } from "@/connect";
import type { Revision } from "@/types/proto-es/v1/revision_service_pb";
import {
  DeleteRevisionRequestSchema,
  GetRevisionRequestSchema,
  ListRevisionsRequestSchema,
} from "@/types/proto-es/v1/revision_service_pb";
import type { AppSliceCreator, RevisionSlice } from "./types";

const revisionNamePrefix = "revisions/";

export const createRevisionSlice: AppSliceCreator<RevisionSlice> = (
  set,
  get
) => ({
  revisionsByName: {},

  listRevisionsByDatabase: async (database, pagination) => {
    const response = await revisionServiceClientConnect.listRevisions(
      createProto(ListRevisionsRequestSchema, {
        parent: database,
        pageSize: pagination?.pageSize,
        pageToken: pagination?.pageToken ?? "",
      })
    );
    set((state) => ({
      revisionsByName: {
        ...state.revisionsByName,
        ...Object.fromEntries(
          response.revisions.map((revision) => [revision.name, revision])
        ),
      },
    }));
    return {
      revisions: response.revisions,
      nextPageToken: response.nextPageToken,
    };
  },

  listAllRevisionsByDatabase: async (database, pagination) => {
    const revisions: Revision[] = [];
    let pageToken = "";
    do {
      const response = await get().listRevisionsByDatabase(database, {
        pageSize: pagination?.pageSize,
        pageToken,
      });
      revisions.push(...response.revisions);
      pageToken = response.nextPageToken;
    } while (pageToken);
    return revisions;
  },

  fetchRevision: async (name) => {
    const cached = get().revisionsByName[name];
    if (cached) return cached;
    const revision = await revisionServiceClientConnect.getRevision(
      createProto(GetRevisionRequestSchema, { name })
    );
    set((state) => ({
      revisionsByName: { ...state.revisionsByName, [revision.name]: revision },
    }));
    return revision;
  },

  getRevisionsByDatabase: (database) => {
    return Object.values(get().revisionsByName).filter((revision) =>
      revision.name.startsWith(`${database}/${revisionNamePrefix}`)
    );
  },

  getRevisionByName: (name) => get().revisionsByName[name],

  deleteRevision: async (name) => {
    await revisionServiceClientConnect.deleteRevision(
      createProto(DeleteRevisionRequestSchema, { name })
    );
    set((state) => {
      const { [name]: _, ...revisionsByName } = state.revisionsByName;
      return { revisionsByName };
    });
  },
});
