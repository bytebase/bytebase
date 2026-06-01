import { create as createProto } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import { releaseServiceClientConnect } from "@/connect";
import { silentContextKey } from "@/connect/context-key";
import { isValidReleaseName } from "@/types";
import { State } from "@/types/proto-es/v1/common_pb";
import type { Release } from "@/types/proto-es/v1/release_service_pb";
import {
  DeleteReleaseRequestSchema,
  GetReleaseRequestSchema,
  ListReleasesRequestSchema,
  ReleaseSchema,
  UndeleteReleaseRequestSchema,
  UpdateReleaseRequestSchema,
} from "@/types/proto-es/v1/release_service_pb";
import type { AppSliceCreator, ReleaseSlice } from "./types";

export const createReleaseSlice: AppSliceCreator<ReleaseSlice> = (
  set,
  get
) => ({
  releasesByName: {},
  releaseRequests: {},

  listReleasesByProject: async (project, pagination, showDeleted, filter) => {
    const response = await releaseServiceClientConnect.listReleases(
      createProto(ListReleasesRequestSchema, {
        parent: project,
        pageSize: pagination?.pageSize,
        pageToken: pagination?.pageToken ?? "",
        showDeleted: Boolean(showDeleted),
        filter: filter ?? "",
      })
    );
    set((state) => ({
      releasesByName: {
        ...state.releasesByName,
        ...Object.fromEntries(
          response.releases.map((release) => [release.name, release])
        ),
      },
    }));
    return {
      releases: response.releases,
      nextPageToken: response.nextPageToken,
    };
  },

  fetchRelease: async (name, silent = false) => {
    if (!isValidReleaseName(name)) return undefined;
    const existing = get().releasesByName[name];
    if (existing) return existing;
    const pending = get().releaseRequests[name];
    if (pending) return pending;

    const request = releaseServiceClientConnect
      .getRelease(createProto(GetReleaseRequestSchema, { name }), {
        contextValues: createContextValues().set(silentContextKey, silent),
      })
      .then((release) => {
        set((state) => {
          const { [name]: _, ...releaseRequests } = state.releaseRequests;
          return {
            releasesByName: {
              ...state.releasesByName,
              [release.name]: release,
            },
            releaseRequests,
          };
        });
        return release;
      })
      .catch((error) => {
        set((state) => {
          const { [name]: _, ...releaseRequests } = state.releaseRequests;
          return { releaseRequests };
        });
        throw error;
      });
    set((state) => ({
      releaseRequests: { ...state.releaseRequests, [name]: request },
    }));
    return request;
  },

  getReleasesByProject: (project) => {
    return Object.values(get().releasesByName).filter((release) =>
      release.name.startsWith(`${project}/releases/`)
    );
  },

  getReleaseByName: (name) => get().releasesByName[name],

  updateRelease: async (release, updateMask) => {
    const response = await releaseServiceClientConnect.updateRelease(
      createProto(UpdateReleaseRequestSchema, {
        release: createProto(ReleaseSchema, release as Release),
        updateMask: { paths: updateMask },
      })
    );
    set((state) => ({
      releasesByName: { ...state.releasesByName, [response.name]: response },
    }));
    return response;
  },

  deleteRelease: async (name) => {
    await releaseServiceClientConnect.deleteRelease(
      createProto(DeleteReleaseRequestSchema, { name })
    );
    set((state) => {
      const cached = state.releasesByName[name];
      if (!cached) return {};
      return {
        releasesByName: {
          ...state.releasesByName,
          [name]: { ...cached, state: State.DELETED },
        },
      };
    });
  },

  undeleteRelease: async (name) => {
    const response = await releaseServiceClientConnect.undeleteRelease(
      createProto(UndeleteReleaseRequestSchema, { name })
    );
    set((state) => ({
      releasesByName: { ...state.releasesByName, [response.name]: response },
    }));
    return response;
  },
});
