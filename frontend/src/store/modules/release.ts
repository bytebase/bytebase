import { create } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import { defineStore } from "pinia";
import { computed, reactive, ref, unref, watch } from "vue";
import { releaseServiceClientConnect } from "@/connect";
import { silentContextKey } from "@/connect/context-key";
import type { MaybeRef, Pagination } from "@/types";
import { isValidReleaseName, unknownRelease } from "@/types";
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

export const useReleaseStore = defineStore("release", () => {
  const releaseMapByName = reactive(new Map<string, Release>());

  const releaseList = computed(() => {
    return Array.from(releaseMapByName.values());
  });

  const fetchReleasesByProject = async (
    project: string,
    pagination?: Pagination,
    showDeleted?: boolean
  ) => {
    const request = create(ListReleasesRequestSchema, {
      parent: project,
      pageSize: pagination?.pageSize,
      pageToken: pagination?.pageToken || "",
      showDeleted: Boolean(showDeleted),
    });
    const resp = await releaseServiceClientConnect.listReleases(request);
    const releases = resp.releases;
    releases.forEach((release) => {
      releaseMapByName.set(release.name, release);
    });
    return {
      releases: releases,
      nextPageToken: resp.nextPageToken,
    };
  };

  const fetchReleaseByName = async (name: string, silent = false) => {
    const request = create(GetReleaseRequestSchema, { name });
    const response = await releaseServiceClientConnect.getRelease(request, {
      contextValues: createContextValues().set(silentContextKey, silent),
    });
    releaseMapByName.set(response.name, response);
    return response;
  };

  const getReleasesByProject = (project: string) => {
    return releaseList.value.filter((release) =>
      release.name.startsWith(`${project}/releases/`)
    );
  };

  const getReleaseByName = (name: string) => {
    return releaseMapByName.get(name) ?? unknownRelease();
  };

  const updateRelase = async (
    release: Partial<Release>,
    updateMask: string[]
  ) => {
    const fullRelease = {
      ...create(ReleaseSchema, {}),
      ...release,
    };

    const request = create(UpdateReleaseRequestSchema, {
      release: fullRelease,
      updateMask: { paths: updateMask },
    });
    const resp = await releaseServiceClientConnect.updateRelease(request);
    releaseMapByName.set(resp.name, resp);
    return resp;
  };

  const deleteRelease = async (name: string) => {
    const request = create(DeleteReleaseRequestSchema, { name });
    await releaseServiceClientConnect.deleteRelease(request);
    if (releaseMapByName.get(name)) {
      releaseMapByName.get(name)!.state = State.DELETED;
    }
  };

  const undeleteRelease = async (name: string) => {
    const request = create(UndeleteReleaseRequestSchema, { name });
    const response = await releaseServiceClientConnect.undeleteRelease(request);
    releaseMapByName.set(response.name, response);
  };

  return {
    releaseList,
    fetchReleasesByProject,
    fetchReleaseByName,
    getReleasesByProject,
    getReleaseByName,
    updateRelase,
    deleteRelease,
    undeleteRelease,
  };
});

export const useReleaseByName = (name: MaybeRef<string>) => {
  const store = useReleaseStore();
  const ready = ref(true);
  watch(
    () => unref(name),
    async (name) => {
      if (!isValidReleaseName(name)) {
        return;
      }

      const cachedRelease = store.getReleaseByName(name);
      if (!isValidReleaseName(cachedRelease.name)) {
        ready.value = false;
        await store.fetchReleaseByName(name);
        ready.value = true;
      }
    },
    { immediate: true }
  );
  const release = computed(() => store.getReleaseByName(unref(name)));

  return {
    release,
    ready,
  };
};
